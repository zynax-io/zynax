// SPDX-License-Identifier: Apache-2.0

// Package postgres provides a Postgres-backed TaskRepository using pgx/v5.
package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // register pgx5 migrate driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// TaskRepository is a pgx/v5 implementation of domain.TaskRepository.
type TaskRepository struct {
	pool *pgxpool.Pool
}

// New opens a connection pool and runs all pending migrations.
// dsn must be a valid Postgres connection string (never hardcoded — read from env).
func New(ctx context.Context, dsn string) (*TaskRepository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres task-broker: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres task-broker: ping: %w", err)
	}
	if err := runMigrations(dsn); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres task-broker: migrate: %w", err)
	}
	return &TaskRepository{pool: pool}, nil
}

// Close releases the connection pool.
func (r *TaskRepository) Close() {
	r.pool.Close()
}

func runMigrations(dsn string) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, "pgx5://"+stripSchema(dsn))
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// stripSchema removes a leading "postgres://" or "postgresql://" prefix so that
// golang-migrate receives the "pgx5://" scheme it expects.
func stripSchema(dsn string) string {
	for _, pfx := range []string{"postgres://", "postgresql://"} {
		if len(dsn) > len(pfx) && dsn[:len(pfx)] == pfx {
			return dsn[len(pfx):]
		}
	}
	return dsn
}

// Save inserts a new task row. Overwrites on conflict (idempotent bootstrap).
func (r *TaskRepository) Save(ctx context.Context, task *domain.Task) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tasks (
			task_id, workflow_id, capability_name, input_payload,
			timeout_seconds, max_retries, retry_count, status,
			dispatched_to, result_payload,
			error_code, error_message, error_details,
			created_at, dispatched_at, completed_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16
		) ON CONFLICT (task_id) DO UPDATE SET
			workflow_id=EXCLUDED.workflow_id,
			capability_name=EXCLUDED.capability_name,
			input_payload=EXCLUDED.input_payload,
			timeout_seconds=EXCLUDED.timeout_seconds,
			max_retries=EXCLUDED.max_retries,
			retry_count=EXCLUDED.retry_count,
			status=EXCLUDED.status,
			dispatched_to=EXCLUDED.dispatched_to,
			result_payload=EXCLUDED.result_payload,
			error_code=EXCLUDED.error_code,
			error_message=EXCLUDED.error_message,
			error_details=EXCLUDED.error_details,
			created_at=EXCLUDED.created_at,
			dispatched_at=EXCLUDED.dispatched_at,
			completed_at=EXCLUDED.completed_at`,
		task.TaskID, task.WorkflowID, task.CapabilityName, task.InputPayload,
		task.TimeoutSeconds, task.MaxRetries, task.RetryCount, int32(task.Status),
		task.DispatchedTo, task.ResultPayload,
		errorCode(task), errorMessage(task), errorDetails(task),
		task.CreatedAt.UTC(), nullTime(task.DispatchedAt), nullTime(task.CompletedAt),
	)
	if err != nil {
		return fmt.Errorf("postgres task-broker: save %q: %w", task.TaskID, err)
	}
	return nil
}

// GetByID retrieves a task by its ID.
func (r *TaskRepository) GetByID(ctx context.Context, taskID string) (*domain.Task, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT task_id, workflow_id, capability_name, input_payload,
		       timeout_seconds, max_retries, retry_count, status,
		       dispatched_to, result_payload,
		       error_code, error_message, error_details,
		       created_at, dispatched_at, completed_at
		FROM tasks WHERE task_id = $1`, taskID)
	t, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %q", domain.ErrTaskNotFound, taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("postgres task-broker: get %q: %w", taskID, err)
	}
	return t, nil
}

// Update writes the full task state. Returns ErrTaskNotFound if the row is absent.
func (r *TaskRepository) Update(ctx context.Context, task *domain.Task) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE tasks SET
			workflow_id=$2, capability_name=$3, input_payload=$4,
			timeout_seconds=$5, max_retries=$6, retry_count=$7, status=$8,
			dispatched_to=$9, result_payload=$10,
			error_code=$11, error_message=$12, error_details=$13,
			created_at=$14, dispatched_at=$15, completed_at=$16
		WHERE task_id=$1`,
		task.TaskID, task.WorkflowID, task.CapabilityName, task.InputPayload,
		task.TimeoutSeconds, task.MaxRetries, task.RetryCount, int32(task.Status),
		task.DispatchedTo, task.ResultPayload,
		errorCode(task), errorMessage(task), errorDetails(task),
		task.CreatedAt.UTC(), nullTime(task.DispatchedAt), nullTime(task.CompletedAt),
	)
	if err != nil {
		return fmt.Errorf("postgres task-broker: update %q: %w", task.TaskID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %q", domain.ErrTaskNotFound, task.TaskID)
	}
	return nil
}

const defaultPageSize = 50
const maxPageSize = 500

// buildListQuery returns the WHERE + ORDER BY + LIMIT clause and args for List.
func buildListQuery(filter domain.ListFilter, pageSize int, cursorTime time.Time, cursorID string) (string, []any) {
	args := []any{}
	clause := "WHERE 1=1"
	idx := 1

	if filter.WorkflowID != "" {
		clause += fmt.Sprintf(" AND workflow_id=$%d", idx)
		args = append(args, filter.WorkflowID)
		idx++
	}
	if filter.Status != domain.TaskStatusUnspecified {
		clause += fmt.Sprintf(" AND status=$%d", idx)
		args = append(args, int32(filter.Status))
		idx++
	}
	if filter.AgentID != "" {
		clause += fmt.Sprintf(" AND dispatched_to=$%d", idx)
		args = append(args, filter.AgentID)
		idx++
	}
	if !cursorTime.IsZero() {
		clause += fmt.Sprintf(" AND (created_at, task_id) > ($%d, $%d)", idx, idx+1)
		args = append(args, cursorTime.UTC(), cursorID)
		idx += 2
	}
	clause += fmt.Sprintf(" ORDER BY created_at ASC, task_id ASC LIMIT $%d", idx)
	args = append(args, pageSize+1)
	return clause, args
}

// List returns a paginated page of tasks matching filter. Cursor is (created_at, task_id).
func (r *TaskRepository) List(ctx context.Context, filter domain.ListFilter) (domain.ListResult, error) {
	pageSize := int(filter.PageSize)
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	var cursorTime time.Time
	var cursorID string
	if filter.PageToken != "" {
		var err error
		cursorTime, cursorID, err = decodeCursor(filter.PageToken)
		if err != nil {
			return domain.ListResult{}, fmt.Errorf("task-broker: invalid page_token: %w", err)
		}
	}

	clause, args := buildListQuery(filter, pageSize, cursorTime, cursorID)
	rows, err := r.pool.Query(ctx, `
		SELECT task_id, workflow_id, capability_name, input_payload,
		       timeout_seconds, max_retries, retry_count, status,
		       dispatched_to, result_payload,
		       error_code, error_message, error_details,
		       created_at, dispatched_at, completed_at
		FROM tasks `+clause, args...)
	if err != nil {
		return domain.ListResult{}, fmt.Errorf("postgres task-broker: list: %w", err)
	}
	defer rows.Close()

	tasks, err := collectTaskRows(rows)
	if err != nil {
		return domain.ListResult{}, err
	}

	var nextToken string
	if len(tasks) > pageSize {
		last := tasks[pageSize-1]
		nextToken = encodeCursor(last.CreatedAt, last.TaskID)
		tasks = tasks[:pageSize]
	}
	return domain.ListResult{Tasks: tasks, NextPageToken: nextToken}, nil
}

// collectTaskRows drains a pgx.Rows result set into a task slice.
func collectTaskRows(rows pgx.Rows) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres task-broker: list scan: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres task-broker: list rows: %w", err)
	}
	return tasks, nil
}

// scanTask reads one task row from a pgx.Row or pgx.Rows.
func scanTask(row pgx.Row) (*domain.Task, error) {
	var t domain.Task
	var status int32
	var errCode, errMsg string
	var errDetails []byte
	var dispatchedAt, completedAt *time.Time

	err := row.Scan(
		&t.TaskID, &t.WorkflowID, &t.CapabilityName, &t.InputPayload,
		&t.TimeoutSeconds, &t.MaxRetries, &t.RetryCount, &status,
		&t.DispatchedTo, &t.ResultPayload,
		&errCode, &errMsg, &errDetails,
		&t.CreatedAt, &dispatchedAt, &completedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan task row: %w", err)
	}
	t.Status = domain.TaskStatus(status)
	if errCode != "" || errMsg != "" {
		t.Error = &domain.TaskError{Code: errCode, Message: errMsg}
		if len(errDetails) > 0 {
			t.Error.Details = parseJSONBDetails(errDetails)
		}
	}
	if dispatchedAt != nil {
		t.DispatchedAt = *dispatchedAt
	}
	if completedAt != nil {
		t.CompletedAt = *completedAt
	}
	return &t, nil
}

func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	u := t.UTC()
	return &u
}

func errorCode(t *domain.Task) string {
	if t.Error != nil {
		return t.Error.Code
	}
	return ""
}

func errorMessage(t *domain.Task) string {
	if t.Error != nil {
		return t.Error.Message
	}
	return ""
}

func errorDetails(t *domain.Task) []byte {
	if t.Error == nil || len(t.Error.Details) == 0 {
		return nil
	}
	return marshalJSONBDetails(t.Error.Details)
}
