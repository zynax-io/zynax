// SPDX-License-Identifier: Apache-2.0

// Package domain contains the pure business logic for the task-broker service.
// It has zero imports from the api or infrastructure layers (ADR-001).
package domain

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// TaskService implements core broker logic: dispatch, acknowledge, cancel, and query.
type TaskService struct {
	repo      TaskRepository
	finder    AgentFinder
	executor  CapabilityExecutor
	publisher TaskEventPublisher
	bg        sync.WaitGroup
	nextAgent atomic.Uint64
}

// NewTaskService constructs a TaskService with the given port implementations.
func NewTaskService(repo TaskRepository, finder AgentFinder, executor CapabilityExecutor) *TaskService {
	return &TaskService{repo: repo, finder: finder, executor: executor}
}

// WithEventPublisher attaches a best-effort lifecycle event publisher so the
// capability fan-out is observable over the event bus (EPIC #881 O5). A nil
// publisher disables publication. Returns s for chaining.
func (s *TaskService) WithEventPublisher(p TaskEventPublisher) *TaskService {
	s.publisher = p
	return s
}

// DispatchTask validates the request, records the task as PENDING, and launches
// an async goroutine that routes to an agent and drives the lifecycle to completion.
func (s *TaskService) DispatchTask(ctx context.Context, task *Task) (taskID string, createdAt time.Time, err error) {
	if task.WorkflowID == "" {
		return "", time.Time{}, fmt.Errorf("%w: workflow_id is required", ErrInvalidArgument)
	}
	if task.CapabilityName == "" {
		return "", time.Time{}, fmt.Errorf("%w: capability_name is required", ErrInvalidArgument)
	}
	if !json.Valid(task.InputPayload) {
		return "", time.Time{}, fmt.Errorf("%w: input_payload must be valid JSON", ErrInvalidArgument)
	}

	agents, err := s.finder.FindByCapability(ctx, task.CapabilityName)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("task-broker: find agents: %w", err)
	}
	if len(agents) == 0 {
		return "", time.Time{}, fmt.Errorf("%w: no agent registered for capability %q", ErrNoEligibleAgent, task.CapabilityName)
	}

	// Context-slice injection binding (ADR-028, EPIC #881 O5): an
	// expert-targeted payload is narrowed to exactly that expert and rewritten
	// to carry only its declared slice. Binding happens before Save so the
	// persisted payload — and therefore any restart recovery — carries the
	// bound slice durably.
	agents, boundPayload, err := prepareExpertDispatch(task.InputPayload, agents)
	if err != nil {
		return "", time.Time{}, err
	}
	task.InputPayload = boundPayload

	task.TaskID = newTaskID()
	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()
	if err := s.repo.Save(ctx, task); err != nil {
		return "", time.Time{}, fmt.Errorf("task-broker: save: %w", err)
	}

	s.bg.Add(1)
	go func() {
		defer s.bg.Done()
		// detach preserves context values (request-ID, trace) without inheriting
		// the parent's cancellation deadline — the RPC handler returns before the
		// task finishes, so the goroutine must outlive it.
		s.executeAsync(detach(ctx), task.TaskID, agents)
	}()

	return task.TaskID, task.CreatedAt, nil
}

// AcknowledgeTask records the outcome of a capability execution.
// COMPLETED requires a non-empty result_payload. FAILED triggers retry logic:
// if retries remain the status becomes RETRYING, otherwise FAILED (terminal).
func (s *TaskService) AcknowledgeTask(ctx context.Context, taskID string, newStatus TaskStatus, resultPayload []byte, taskErr *TaskError) (TaskStatus, error) {
	if taskID == "" {
		return 0, fmt.Errorf("%w: task_id is required", ErrInvalidArgument)
	}
	switch newStatus {
	case TaskStatusUnspecified, TaskStatusPending, TaskStatusDispatched, TaskStatusRetrying:
		return 0, fmt.Errorf("%w: status must be COMPLETED, FAILED, or CANCELLED", ErrInvalidArgument)
	}
	if newStatus == TaskStatusCompleted && len(resultPayload) == 0 {
		return 0, fmt.Errorf("%w: result_payload is required for COMPLETED status", ErrInvalidArgument)
	}

	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return 0, fmt.Errorf("task-broker: get task: %w", err)
	}
	if task.Status.IsTerminal() {
		return 0, fmt.Errorf("%w: %s", ErrTaskTerminal, task.Status)
	}

	resulting := s.applyAcknowledgement(task, newStatus, resultPayload, taskErr)
	if err := s.repo.Update(ctx, task); err != nil {
		return 0, fmt.Errorf("task-broker: update: %w", err)
	}
	s.publishEvent(ctx, task)
	return resulting, nil
}

// GetTask returns the current state of a task by its broker-assigned id.
func (s *TaskService) GetTask(ctx context.Context, taskID string) (*Task, error) {
	if taskID == "" {
		return nil, fmt.Errorf("%w: task_id is required", ErrInvalidArgument)
	}
	t, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task-broker: get task: %w", err)
	}
	return t, nil
}

// ListTasks returns a filtered, paginated list of tasks.
func (s *TaskService) ListTasks(ctx context.Context, filter ListFilter) (ListResult, error) {
	result, err := s.repo.List(ctx, filter)
	if err != nil {
		return ListResult{}, fmt.Errorf("task-broker: list tasks: %w", err)
	}
	return result, nil
}

// CancelTask transitions a non-terminal task to CANCELLED.
func (s *TaskService) CancelTask(ctx context.Context, taskID string) (time.Time, error) {
	if taskID == "" {
		return time.Time{}, fmt.Errorf("%w: task_id is required", ErrInvalidArgument)
	}
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return time.Time{}, fmt.Errorf("task-broker: get task: %w", err)
	}
	if task.Status.IsTerminal() {
		return time.Time{}, fmt.Errorf("%w: %s", ErrTaskTerminal, task.Status)
	}
	task.Status = TaskStatusCancelled
	task.CompletedAt = time.Now()
	if err := s.repo.Update(ctx, task); err != nil {
		return time.Time{}, fmt.Errorf("task-broker: update: %w", err)
	}
	s.publishEvent(ctx, task)
	return task.CompletedAt, nil
}

// recoveryPageSize is the repository page size used by startup recovery.
const recoveryPageSize = 100

// RecoverInFlight re-launches execution for every non-terminal task in the
// repository, called once at startup so a broker restart never loses an
// in-flight fan-out (EPIC #881 O5): task state — including the context slice
// bound into input_payload at original dispatch time — is durable in the
// repository (#626 Postgres-backed). Returns the number of tasks re-launched.
func (s *TaskService) RecoverInFlight(ctx context.Context) (int, error) {
	recovered := 0
	for _, status := range []TaskStatus{TaskStatusPending, TaskStatusDispatched, TaskStatusRetrying} {
		n, err := s.recoverByStatus(ctx, status)
		recovered += n
		if err != nil {
			return recovered, err
		}
	}
	return recovered, nil
}

func (s *TaskService) recoverByStatus(ctx context.Context, status TaskStatus) (int, error) {
	recovered := 0
	token := ""
	for {
		page, err := s.repo.List(ctx, ListFilter{Status: status, PageToken: token, PageSize: recoveryPageSize})
		if err != nil {
			return recovered, fmt.Errorf("task-broker: recover list %s tasks: %w", status, err)
		}
		for _, task := range page.Tasks {
			if s.relaunch(ctx, task) {
				recovered++
			}
		}
		if page.NextPageToken == "" {
			return recovered, nil
		}
		token = page.NextPageToken
	}
}

// relaunch resolves the eligible agents for a recovered task and restarts its
// async execution. The persisted payload already carries the slice bound at
// original dispatch time — no re-binding; expert targeting is re-applied for
// routing, and a task targeted at a now-missing expert is left untouched
// (never re-routed) for a later recovery pass.
func (s *TaskService) relaunch(ctx context.Context, task *Task) bool {
	agents, err := s.finder.FindByCapability(ctx, task.CapabilityName)
	if err != nil || len(agents) == 0 {
		return false
	}
	if expert, ok := expertTarget(task.InputPayload); ok {
		agent, found := selectExpert(agents, expert)
		if !found {
			return false
		}
		agents = []AgentInfo{agent}
	}
	s.bg.Add(1)
	go func() {
		defer s.bg.Done()
		s.executeAsync(detach(ctx), task.TaskID, agents)
	}()
	return true
}

// publishEvent forwards a task snapshot to the lifecycle event publisher.
// No-op when unset; best-effort by port contract, never affects task state.
func (s *TaskService) publishEvent(ctx context.Context, task *Task) {
	if s.publisher == nil {
		return
	}
	snapshot := *task
	s.publisher.PublishTaskEvent(ctx, &snapshot)
}

// WaitBackground blocks until all goroutines launched by DispatchTask have finished.
// Intended for tests that need deterministic task-lifecycle assertions.
func (s *TaskService) WaitBackground() { s.bg.Wait() }

func (s *TaskService) executeAsync(ctx context.Context, taskID string, agents []AgentInfo) {
	idx := s.nextAgent.Add(1) - 1
	agent := agents[idx%uint64(len(agents))] //nolint:gosec // G115: idx bounded by len

	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil || task.Status.IsTerminal() {
		return
	}
	task.Status = TaskStatusDispatched
	task.DispatchedTo = agent.AgentID
	task.DispatchedAt = time.Now()
	if err := s.repo.Update(ctx, task); err != nil {
		return
	}
	s.publishEvent(ctx, task)

	resultPayload, taskErr, execErr := s.executor.Execute(ctx, agent, task)

	task, err = s.repo.GetByID(ctx, taskID)
	if err != nil || task.Status.IsTerminal() {
		return
	}

	if execErr != nil {
		taskErr = &TaskError{Code: "INTERNAL", Message: execErr.Error()}
	}
	if taskErr != nil {
		s.applyAcknowledgement(task, TaskStatusFailed, nil, taskErr)
	} else {
		s.applyAcknowledgement(task, TaskStatusCompleted, resultPayload, nil)
	}
	if err := s.repo.Update(ctx, task); err == nil {
		s.publishEvent(ctx, task)
	}
}

func (s *TaskService) applyAcknowledgement(task *Task, newStatus TaskStatus, resultPayload []byte, taskErr *TaskError) TaskStatus {
	switch newStatus {
	case TaskStatusCompleted:
		task.Status = TaskStatusCompleted
		task.ResultPayload = resultPayload
		task.CompletedAt = time.Now()
		return TaskStatusCompleted

	case TaskStatusFailed:
		newCount := task.RetryCount + 1
		if newCount <= task.MaxRetries {
			task.Status = TaskStatusRetrying
			task.RetryCount = newCount
			return TaskStatusRetrying
		}
		task.Status = TaskStatusFailed
		task.Error = taskErr
		task.CompletedAt = time.Now()
		return TaskStatusFailed

	default: // TaskStatusCancelled
		task.Status = TaskStatusCancelled
		task.CompletedAt = time.Now()
		return TaskStatusCancelled
	}
}

func newTaskID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is unrecoverable; the broker cannot safely assign IDs.
		panic(fmt.Sprintf("task-broker: newTaskID: %v", err))
	}
	return "task-" + hex.EncodeToString(b)
}

// detachedCtx carries context values from parent but is never cancelled.
// This allows async goroutines to outlive the RPC handler that spawned them
// while still propagating request-ID and trace metadata via Value.
type detachedCtx struct{ parent context.Context }

func (d detachedCtx) Deadline() (time.Time, bool) { return time.Time{}, false }
func (d detachedCtx) Done() <-chan struct{}       { return nil }
func (d detachedCtx) Err() error                  { return nil }
func (d detachedCtx) Value(key any) any           { return d.parent.Value(key) }

// detach wraps ctx in a detachedCtx. The returned context preserves all
// values but ignores any cancellation signal from ctx.
func detach(ctx context.Context) context.Context { return detachedCtx{parent: ctx} }
