// SPDX-License-Identifier: Apache-2.0

// Package postgres provides a Postgres-backed AgentRepository using pgx/v5.
package postgres

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // register pgx5 migrate driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zynax-io/zynax/services/agent-registry/internal/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// AgentRepository is a pgx/v5 implementation of domain.AgentRepository.
type AgentRepository struct {
	pool *pgxpool.Pool
}

// New opens a connection pool and runs all pending migrations.
func New(ctx context.Context, dsn string) (*AgentRepository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres agent-registry: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres agent-registry: ping: %w", err)
	}
	if err := runMigrations(dsn); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres agent-registry: migrate: %w", err)
	}
	return &AgentRepository{pool: pool}, nil
}

// Close releases the connection pool.
func (r *AgentRepository) Close() {
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

func stripSchema(dsn string) string {
	for _, pfx := range []string{"postgres://", "postgresql://"} {
		if len(dsn) > len(pfx) && dsn[:len(pfx)] == pfx {
			return dsn[len(pfx):]
		}
	}
	return dsn
}

// Save upserts an agent record (insert on conflict update).
func (r *AgentRepository) Save(ctx context.Context, agent domain.Agent) error {
	caps, err := marshalJSON(agent.Capabilities)
	if err != nil {
		return fmt.Errorf("postgres agent-registry: marshal capabilities: %w", err)
	}
	labels, err := marshalJSON(agent.Labels)
	if err != nil {
		return fmt.Errorf("postgres agent-registry: marshal labels: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO agents (id, name, description, endpoint, labels, capabilities, status, registered_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET
			name=EXCLUDED.name,
			description=EXCLUDED.description,
			endpoint=EXCLUDED.endpoint,
			labels=EXCLUDED.labels,
			capabilities=EXCLUDED.capabilities,
			status=EXCLUDED.status,
			registered_at=EXCLUDED.registered_at,
			updated_at=EXCLUDED.updated_at`,
		agent.ID, agent.Name, agent.Description, agent.Endpoint,
		labels, caps, int32(agent.Status),
		agent.RegisteredAt.UTC(), agent.UpdatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("postgres agent-registry: save %q: %w", agent.ID, err)
	}
	return nil
}

// Delete removes an agent by ID.
func (r *AgentRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM agents WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("postgres agent-registry: delete %q: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %q", domain.ErrAgentNotFound, id)
	}
	return nil
}

// FindByID returns the agent with the given ID.
func (r *AgentRepository) FindByID(ctx context.Context, id string) (domain.Agent, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, description, endpoint, labels, capabilities, status, registered_at, updated_at
		FROM agents WHERE id=$1`, id)
	a, err := scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Agent{}, fmt.Errorf("%w: %q", domain.ErrAgentNotFound, id)
	}
	if err != nil {
		return domain.Agent{}, fmt.Errorf("postgres agent-registry: find %q: %w", id, err)
	}
	return a, nil
}

// FindAll returns all agents regardless of status.
func (r *AgentRepository) FindAll(ctx context.Context) ([]domain.Agent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, endpoint, labels, capabilities, status, registered_at, updated_at
		FROM agents ORDER BY registered_at ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("postgres agent-registry: find all: %w", err)
	}
	defer rows.Close()
	return collectAgentRows(rows)
}

// FindByCapability returns REGISTERED agents that declare the named capability.
func (r *AgentRepository) FindByCapability(ctx context.Context, name string) ([]domain.Agent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, endpoint, labels, capabilities, status, registered_at, updated_at
		FROM agents
		WHERE status = $1
		  AND capabilities @> $2::jsonb
		ORDER BY registered_at ASC, id ASC`,
		int32(domain.AgentStatusRegistered),
		fmt.Sprintf(`[{"name":%q}]`, name),
	)
	if err != nil {
		return nil, fmt.Errorf("postgres agent-registry: find by capability %q: %w", name, err)
	}
	defer rows.Close()
	return collectAgentRows(rows)
}

func collectAgentRows(rows pgx.Rows) ([]domain.Agent, error) {
	var agents []domain.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres agent-registry: scan: %w", err)
		}
		agents = append(agents, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres agent-registry: rows: %w", err)
	}
	return agents, nil
}

func scanAgent(row pgx.Row) (domain.Agent, error) {
	var a domain.Agent
	var status int32
	var capsJSON, labelsJSON []byte
	var registeredAt, updatedAt time.Time

	err := row.Scan(
		&a.ID, &a.Name, &a.Description, &a.Endpoint,
		&labelsJSON, &capsJSON,
		&status, &registeredAt, &updatedAt,
	)
	if err != nil {
		return domain.Agent{}, fmt.Errorf("scan agent row: %w", err)
	}
	a.Status = domain.AgentStatus(status)
	a.RegisteredAt = registeredAt
	a.UpdatedAt = updatedAt
	if len(labelsJSON) > 0 {
		_ = json.Unmarshal(labelsJSON, &a.Labels)
	}
	if len(capsJSON) > 0 {
		_ = json.Unmarshal(capsJSON, &a.Capabilities)
	}
	return a, nil
}

func marshalJSON(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}
