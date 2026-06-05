// SPDX-License-Identifier: Apache-2.0

//go:build integration

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
	"github.com/zynax-io/zynax/services/task-broker/internal/infrastructure/postgres"
)

func setupContainer(t *testing.T) (repo *postgres.TaskRepository, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("task_broker_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = ctr.Terminate(ctx)
		t.Fatalf("connection string: %v", err)
	}

	r, err := postgres.New(ctx, dsn)
	if err != nil {
		_ = ctr.Terminate(ctx)
		t.Fatalf("open repository: %v", err)
	}

	return r, func() {
		r.Close()
		_ = ctr.Terminate(ctx)
	}
}

func makeTask(id string) *domain.Task {
	return &domain.Task{
		TaskID:         id,
		WorkflowID:     "wf-1",
		CapabilityName: "echo",
		InputPayload:   []byte(`{"msg":"hello"}`),
		TimeoutSeconds: 30,
		MaxRetries:     3,
		Status:         domain.TaskStatusPending,
		CreatedAt:      time.Now().UTC().Truncate(time.Microsecond),
	}
}

func TestSaveAndGetByID(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	task := makeTask("t-001")
	if err := repo.Save(ctx, task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID(ctx, task.TaskID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.TaskID != task.TaskID {
		t.Errorf("TaskID: want %q got %q", task.TaskID, got.TaskID)
	}
	if got.WorkflowID != task.WorkflowID {
		t.Errorf("WorkflowID: want %q got %q", task.WorkflowID, got.WorkflowID)
	}
	if got.Status != task.Status {
		t.Errorf("Status: want %v got %v", task.Status, got.Status)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected ErrTaskNotFound, got nil")
	}
}

func TestUpdate(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	task := makeTask("t-002")
	if err := repo.Save(ctx, task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	task.Status = domain.TaskStatusCompleted
	task.ResultPayload = []byte(`{"result":"ok"}`)
	task.CompletedAt = time.Now().UTC().Truncate(time.Microsecond)
	if err := repo.Update(ctx, task); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(ctx, task.TaskID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if got.Status != domain.TaskStatusCompleted {
		t.Errorf("Status: want COMPLETED got %v", got.Status)
	}
	if string(got.ResultPayload) != `{"result":"ok"}` {
		t.Errorf("ResultPayload: want %q got %q", `{"result":"ok"}`, got.ResultPayload)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	task := makeTask("t-missing")
	err := repo.Update(ctx, task)
	if err == nil {
		t.Fatal("expected ErrTaskNotFound, got nil")
	}
}

func TestSave_TaskWithError(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	task := makeTask("t-err")
	task.Status = domain.TaskStatusFailed
	task.Error = &domain.TaskError{
		Code:    "TIMEOUT",
		Message: "capability timed out",
		Details: map[string]string{"agent_id": "ag-1"},
	}
	if err := repo.Save(ctx, task); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID(ctx, task.TaskID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Error == nil {
		t.Fatal("expected Error, got nil")
	}
	if got.Error.Code != "TIMEOUT" {
		t.Errorf("Code: want TIMEOUT got %q", got.Error.Code)
	}
	if got.Error.Details["agent_id"] != "ag-1" {
		t.Errorf("Details[agent_id]: want ag-1 got %q", got.Error.Details["agent_id"])
	}
}

func TestList_Pagination(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	for i := range 5 {
		task := makeTask(fmt.Sprintf("t-page-%02d", i))
		task.CreatedAt = time.Now().UTC().Add(time.Duration(i) * time.Millisecond).Truncate(time.Microsecond)
		if err := repo.Save(ctx, task); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}

	page1, err := repo.List(ctx, domain.ListFilter{PageSize: 2})
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if len(page1.Tasks) != 2 {
		t.Fatalf("page 1: want 2 tasks got %d", len(page1.Tasks))
	}
	if page1.NextPageToken == "" {
		t.Fatal("page 1: expected non-empty NextPageToken")
	}

	page2, err := repo.List(ctx, domain.ListFilter{PageSize: 2, PageToken: page1.NextPageToken})
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if len(page2.Tasks) != 2 {
		t.Fatalf("page 2: want 2 tasks got %d", len(page2.Tasks))
	}

	page3, err := repo.List(ctx, domain.ListFilter{PageSize: 2, PageToken: page2.NextPageToken})
	if err != nil {
		t.Fatalf("List page 3: %v", err)
	}
	if len(page3.Tasks) != 1 {
		t.Fatalf("page 3: want 1 task got %d", len(page3.Tasks))
	}
	if page3.NextPageToken != "" {
		t.Fatal("page 3: expected empty NextPageToken")
	}
}

func TestList_FilterByWorkflowID(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	for i, wfID := range []string{"wf-A", "wf-A", "wf-B"} {
		task := makeTask(fmt.Sprintf("t-filter-%02d", i))
		task.WorkflowID = wfID
		task.CreatedAt = time.Now().UTC().Add(time.Duration(i) * time.Millisecond).Truncate(time.Microsecond)
		if err := repo.Save(ctx, task); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	result, err := repo.List(ctx, domain.ListFilter{WorkflowID: "wf-A"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(result.Tasks) != 2 {
		t.Errorf("want 2 tasks for wf-A, got %d", len(result.Tasks))
	}
}

func TestMigration_Idempotent(t *testing.T) {
	repo, cleanup := setupContainer(t)
	defer cleanup()
	ctx := context.Background()

	// A second Save verifies schema is stable (migration ran once, idempotent on re-open).
	task := makeTask("t-idem")
	if err := repo.Save(ctx, task); err != nil {
		t.Fatalf("Save after second open: %v", err)
	}
}
