// SPDX-License-Identifier: Apache-2.0

package domain_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

// ── fakes ──────────────────────────────────────────────────────────────────

type fakeRepo struct {
	mu    sync.Mutex
	tasks map[string]*domain.Task
}

func newFakeRepo() *fakeRepo { return &fakeRepo{tasks: make(map[string]*domain.Task)} }

func (r *fakeRepo) Save(_ context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := *task
	r.tasks[c.TaskID] = &c
	return nil
}

func (r *fakeRepo) GetByID(_ context.Context, taskID string) (*domain.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("%w: %q", domain.ErrTaskNotFound, taskID)
	}
	c := *t
	return &c, nil
}

func (r *fakeRepo) Update(_ context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tasks[task.TaskID]; !ok {
		return fmt.Errorf("%w: %q", domain.ErrTaskNotFound, task.TaskID)
	}
	c := *task
	r.tasks[task.TaskID] = &c
	return nil
}

func (r *fakeRepo) List(_ context.Context, filter domain.ListFilter) (domain.ListResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.Task
	for _, t := range r.tasks {
		if filter.WorkflowID != "" && t.WorkflowID != filter.WorkflowID {
			continue
		}
		if filter.Status != domain.TaskStatusUnspecified && t.Status != filter.Status {
			continue
		}
		c := *t
		out = append(out, &c)
	}
	return domain.ListResult{Tasks: out}, nil
}

type fakeFinder struct{ agents map[string][]domain.AgentInfo }

// Select implements domain.AgentSelector with the reference semantics of the
// CRD-native scheduler: capability lookup, strict expert filter (no
// fallback), first eligible agent deterministically.
func (f *fakeFinder) Select(_ context.Context, capName, expertTarget string) (domain.AgentInfo, error) {
	agents := f.agents[capName]
	if len(agents) == 0 {
		return domain.AgentInfo{}, fmt.Errorf("%w: no agent declares capability %q", domain.ErrNoEligibleAgent, capName)
	}
	if expertTarget != "" {
		for _, a := range agents {
			if a.Name == expertTarget || a.AgentID == expertTarget {
				return a, nil
			}
		}
		return domain.AgentInfo{}, fmt.Errorf("%w: no agent %q", domain.ErrNoEligibleAgent, expertTarget)
	}
	return agents[0], nil
}

type errorFinder struct{ err error }

func (f *errorFinder) Select(_ context.Context, _, _ string) (domain.AgentInfo, error) {
	return domain.AgentInfo{}, f.err
}

type fakeExecutor struct {
	resultPayload []byte
	taskErr       *domain.TaskError
	execErr       error
}

func (e *fakeExecutor) Execute(_ context.Context, _ domain.AgentInfo, _ *domain.Task) ([]byte, *domain.TaskError, error) {
	return e.resultPayload, e.taskErr, e.execErr
}

type blockingExecutor struct {
	started chan struct{}
	block   chan struct{}
}

func (e *blockingExecutor) Execute(_ context.Context, _ domain.AgentInfo, _ *domain.Task) ([]byte, *domain.TaskError, error) {
	close(e.started)
	<-e.block
	return []byte(`{}`), nil, nil
}

// ── helpers ────────────────────────────────────────────────────────────────

func oneAgent() map[string][]domain.AgentInfo {
	return map[string][]domain.AgentInfo{"summarize": {{AgentID: "a1", Endpoint: "localhost:9000"}}}
}

func validTask() *domain.Task {
	return &domain.Task{WorkflowID: "wf-001", CapabilityName: "summarize", InputPayload: []byte(`{}`)}
}

func storeTask(t *testing.T, repo *fakeRepo, task *domain.Task) {
	t.Helper()
	if err := repo.Save(context.Background(), task); err != nil {
		t.Fatalf("seed task: %v", err)
	}
}

func dispatched(id string) *domain.Task {
	return &domain.Task{TaskID: id, WorkflowID: "wf", CapabilityName: "cap",
		InputPayload: []byte(`{}`), Status: domain.TaskStatusDispatched}
}

// ── DispatchTask ───────────────────────────────────────────────────────────

func TestDispatchTask_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	svc := domain.NewTaskService(repo, &fakeFinder{agents: oneAgent()}, &fakeExecutor{resultPayload: []byte(`{"ok":true}`)})

	taskID, createdAt, err := svc.DispatchTask(context.Background(), validTask())
	if err != nil || taskID == "" || createdAt.IsZero() {
		t.Fatalf("DispatchTask: err=%v taskID=%q", err, taskID)
	}

	// Execution runs on a background goroutine, so the in-flight PENDING/DISPATCHED
	// states are not deterministically observable here (an instant executor can reach
	// COMPLETED before this goroutine reads the status back). Mid-flight observability
	// is covered deterministically by TestDispatchTask_NonBlocking (blockingExecutor);
	// the happy path asserts the terminal state once background work drains.
	svc.WaitBackground()

	task, _ := repo.GetByID(context.Background(), taskID)
	if task.Status != domain.TaskStatusCompleted {
		t.Errorf("want COMPLETED, got %s", task.Status)
	}
}

func TestDispatchTask_NonBlocking(t *testing.T) {
	repo := newFakeRepo()
	block := make(chan struct{})
	started := make(chan struct{})
	svc := domain.NewTaskService(repo, &fakeFinder{agents: oneAgent()}, &blockingExecutor{started: started, block: block})

	taskID, _, err := svc.DispatchTask(context.Background(), validTask())
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}

	// Wait until executeAsync has set status=DISPATCHED and called Execute.
	<-started
	task, _ := repo.GetByID(context.Background(), taskID)
	if task.Status != domain.TaskStatusDispatched {
		t.Errorf("want DISPATCHED while executor is blocked, got %s", task.Status)
	}

	close(block)
	svc.WaitBackground()

	task, _ = repo.GetByID(context.Background(), taskID)
	if task.Status != domain.TaskStatusCompleted {
		t.Errorf("want COMPLETED after executor finishes, got %s", task.Status)
	}
}

func TestDispatchTask_ExecutorFailure(t *testing.T) {
	repo := newFakeRepo()
	svc := domain.NewTaskService(repo, &fakeFinder{agents: oneAgent()}, &fakeExecutor{execErr: fmt.Errorf("dial timeout")})

	taskID, _, _ := svc.DispatchTask(context.Background(), validTask())
	svc.WaitBackground()

	task, _ := repo.GetByID(context.Background(), taskID)
	if task.Status != domain.TaskStatusFailed {
		t.Errorf("want FAILED, got %s", task.Status)
	}
}

func TestDispatchTask_Validation(t *testing.T) {
	svc := domain.NewTaskService(newFakeRepo(), &fakeFinder{}, &fakeExecutor{})
	cases := []struct {
		name    string
		mutate  func(*domain.Task)
		wantErr error
	}{
		{"empty workflow_id", func(tk *domain.Task) { tk.WorkflowID = "" }, domain.ErrInvalidArgument},
		{"empty capability_name", func(tk *domain.Task) { tk.CapabilityName = "" }, domain.ErrInvalidArgument},
		{"invalid JSON", func(tk *domain.Task) { tk.InputPayload = []byte("not json") }, domain.ErrInvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tk := validTask()
			tc.mutate(tk)
			_, _, err := svc.DispatchTask(context.Background(), tk)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestDispatchTask_NoAgent(t *testing.T) {
	svc := domain.NewTaskService(newFakeRepo(), &fakeFinder{}, &fakeExecutor{})
	_, _, err := svc.DispatchTask(context.Background(), validTask())
	if !errors.Is(err, domain.ErrNoEligibleAgent) {
		t.Errorf("err = %v, want ErrNoEligibleAgent", err)
	}
}

func TestDispatchTask_FinderError(t *testing.T) {
	svc := domain.NewTaskService(newFakeRepo(), &errorFinder{err: fmt.Errorf("registry down")}, &fakeExecutor{})
	_, _, err := svc.DispatchTask(context.Background(), validTask())
	if err == nil || errors.Is(err, domain.ErrInvalidArgument) {
		t.Errorf("expected wrapped finder error, got %v", err)
	}
}

// ── AcknowledgeTask ────────────────────────────────────────────────────────

func TestAcknowledgeTask_Completed(t *testing.T) {
	repo := newFakeRepo()
	svc := domain.NewTaskService(repo, &fakeFinder{}, &fakeExecutor{})
	storeTask(t, repo, dispatched("t1"))

	res, err := svc.AcknowledgeTask(context.Background(), "t1", domain.TaskStatusCompleted, []byte(`{"r":1}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != domain.TaskStatusCompleted {
		t.Errorf("resulting = %s", res)
	}
	task, _ := repo.GetByID(context.Background(), "t1")
	if task.Status != domain.TaskStatusCompleted || string(task.ResultPayload) != `{"r":1}` {
		t.Errorf("status=%s payload=%s", task.Status, task.ResultPayload)
	}
}

func TestAcknowledgeTask_Cancelled(t *testing.T) {
	repo := newFakeRepo()
	svc := domain.NewTaskService(repo, &fakeFinder{}, &fakeExecutor{})
	storeTask(t, repo, dispatched("t-cancel"))

	res, err := svc.AcknowledgeTask(context.Background(), "t-cancel", domain.TaskStatusCancelled, nil, nil)
	if err != nil || res != domain.TaskStatusCancelled {
		t.Errorf("err=%v resulting=%s", err, res)
	}
}

func TestAcknowledgeTask_FailedTransitions(t *testing.T) {
	cases := []struct {
		name       string
		maxRetries int32
		retryCount int32
		wantStatus domain.TaskStatus
		wantCount  int32
	}{
		{"no_retries", 0, 0, domain.TaskStatusFailed, 0},
		{"retries_remain", 2, 1, domain.TaskStatusRetrying, 2},
		{"retries_exhausted", 2, 2, domain.TaskStatusFailed, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			svc := domain.NewTaskService(repo, &fakeFinder{}, &fakeExecutor{})
			storeTask(t, repo, &domain.Task{
				TaskID: "tf", WorkflowID: "wf", CapabilityName: "cap",
				InputPayload: []byte(`{}`), Status: domain.TaskStatusDispatched,
				MaxRetries: tc.maxRetries, RetryCount: tc.retryCount,
			})

			res, err := svc.AcknowledgeTask(context.Background(), "tf", domain.TaskStatusFailed, nil,
				&domain.TaskError{Code: "TIMEOUT", Message: "timed out"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res != tc.wantStatus {
				t.Errorf("resulting = %s, want %s", res, tc.wantStatus)
			}
			task, _ := repo.GetByID(context.Background(), "tf")
			if task.Status != tc.wantStatus {
				t.Errorf("stored status = %s", task.Status)
			}
			if tc.wantStatus == domain.TaskStatusRetrying && task.RetryCount != tc.wantCount {
				t.Errorf("retry_count = %d, want %d", task.RetryCount, tc.wantCount)
			}
		})
	}
}

func TestAcknowledgeTask_Errors(t *testing.T) {
	repo := newFakeRepo()
	svc := domain.NewTaskService(repo, &fakeFinder{}, &fakeExecutor{})
	storeTask(t, repo, &domain.Task{
		TaskID: "terminal", WorkflowID: "wf", CapabilityName: "cap",
		InputPayload: []byte(`{}`), Status: domain.TaskStatusCompleted,
	})

	cases := []struct {
		name    string
		taskID  string
		status  domain.TaskStatus
		payload []byte
		wantErr error
	}{
		{"empty_task_id", "", domain.TaskStatusCompleted, []byte(`{}`), domain.ErrInvalidArgument},
		{"unspecified_status", "any", domain.TaskStatusUnspecified, nil, domain.ErrInvalidArgument},
		{"completed_no_payload", "any", domain.TaskStatusCompleted, nil, domain.ErrInvalidArgument},
		{"unknown_task", "ghost-task", domain.TaskStatusCompleted, []byte(`{}`), domain.ErrTaskNotFound},
		{"terminal_task", "terminal", domain.TaskStatusCompleted, []byte(`{}`), domain.ErrTaskTerminal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.AcknowledgeTask(context.Background(), tc.taskID, tc.status, tc.payload, nil)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// ── CancelTask ─────────────────────────────────────────────────────────────

func TestCancelTask(t *testing.T) {
	repo := newFakeRepo()
	svc := domain.NewTaskService(repo, &fakeFinder{}, &fakeExecutor{})
	storeTask(t, repo, &domain.Task{
		TaskID: "c1", WorkflowID: "wf", CapabilityName: "cap",
		InputPayload: []byte(`{}`), Status: domain.TaskStatusPending,
	})

	cancelledAt, err := svc.CancelTask(context.Background(), "c1")
	if err != nil || cancelledAt.IsZero() {
		t.Fatalf("CancelTask: err=%v", err)
	}
	task, _ := repo.GetByID(context.Background(), "c1")
	if task.Status != domain.TaskStatusCancelled {
		t.Errorf("status = %s", task.Status)
	}
}

func TestCancelTask_Errors(t *testing.T) {
	repo := newFakeRepo()
	svc := domain.NewTaskService(repo, &fakeFinder{}, &fakeExecutor{})
	storeTask(t, repo, &domain.Task{
		TaskID: "completed", WorkflowID: "wf", CapabilityName: "cap",
		InputPayload: []byte(`{}`), Status: domain.TaskStatusCompleted,
	})

	cases := []struct {
		name    string
		taskID  string
		wantErr error
	}{
		{"empty_id", "", domain.ErrInvalidArgument},
		{"unknown_task", "nonexistent", domain.ErrTaskNotFound},
		{"terminal_task", "completed", domain.ErrTaskTerminal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CancelTask(context.Background(), tc.taskID)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// ── GetTask ────────────────────────────────────────────────────────────────

func TestGetTask(t *testing.T) {
	repo := newFakeRepo()
	svc := domain.NewTaskService(repo, &fakeFinder{}, &fakeExecutor{})
	storeTask(t, repo, &domain.Task{TaskID: "g1", WorkflowID: "wf-get", CapabilityName: "cap", InputPayload: []byte(`{}`), Status: domain.TaskStatusPending})

	task, err := svc.GetTask(context.Background(), "g1")
	if err != nil || task.WorkflowID != "wf-get" {
		t.Errorf("err=%v wfID=%s", err, task.WorkflowID)
	}

	_, err = svc.GetTask(context.Background(), "nonexistent-task")
	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Errorf("err = %v, want ErrTaskNotFound", err)
	}

	_, err = svc.GetTask(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Errorf("err = %v, want ErrInvalidArgument", err)
	}
}

// ── ListTasks ──────────────────────────────────────────────────────────────

func TestListTasks(t *testing.T) {
	repo := newFakeRepo()
	svc := domain.NewTaskService(repo, &fakeFinder{}, &fakeExecutor{})

	storeTask(t, repo, &domain.Task{TaskID: "la", WorkflowID: "wf-A", CapabilityName: "cap", InputPayload: []byte(`{}`), Status: domain.TaskStatusPending})
	storeTask(t, repo, &domain.Task{TaskID: "lb", WorkflowID: "wf-B", CapabilityName: "cap", InputPayload: []byte(`{}`), Status: domain.TaskStatusCompleted})

	// Filter by workflow_id.
	res, err := svc.ListTasks(context.Background(), domain.ListFilter{WorkflowID: "wf-A"})
	if err != nil || len(res.Tasks) != 1 || res.Tasks[0].TaskID != "la" {
		t.Errorf("workflow filter: err=%v tasks=%v", err, res.Tasks)
	}

	// Filter by status.
	res, err = svc.ListTasks(context.Background(), domain.ListFilter{Status: domain.TaskStatusCompleted})
	if err != nil || len(res.Tasks) != 1 || res.Tasks[0].TaskID != "lb" {
		t.Errorf("status filter: err=%v tasks=%v", err, res.Tasks)
	}
}

// ── TaskStatus ─────────────────────────────────────────────────────────────

func TestTaskStatus_IsTerminal(t *testing.T) {
	for _, s := range []domain.TaskStatus{domain.TaskStatusCompleted, domain.TaskStatusFailed, domain.TaskStatusCancelled} {
		if !s.IsTerminal() {
			t.Errorf("%s should be terminal", s)
		}
	}
	for _, s := range []domain.TaskStatus{domain.TaskStatusUnspecified, domain.TaskStatusPending, domain.TaskStatusDispatched, domain.TaskStatusRetrying} {
		if s.IsTerminal() {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestTaskStatus_String(t *testing.T) {
	cases := map[domain.TaskStatus]string{
		domain.TaskStatusUnspecified: "UNSPECIFIED",
		domain.TaskStatusPending:     "PENDING",
		domain.TaskStatusDispatched:  "DISPATCHED",
		domain.TaskStatusRetrying:    "RETRYING",
		domain.TaskStatusCompleted:   "COMPLETED",
		domain.TaskStatusFailed:      "FAILED",
		domain.TaskStatusCancelled:   "CANCELLED",
	}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Errorf("TaskStatus(%d).String() = %q, want %q", s, got, want)
		}
	}
}
