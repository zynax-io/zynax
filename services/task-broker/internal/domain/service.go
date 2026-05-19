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
	bg        sync.WaitGroup
	nextAgent atomic.Uint64
}

// NewTaskService constructs a TaskService with the given port implementations.
func NewTaskService(repo TaskRepository, finder AgentFinder, executor CapabilityExecutor) *TaskService {
	return &TaskService{repo: repo, finder: finder, executor: executor}
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

	task.TaskID = newTaskID()
	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()
	if err := s.repo.Save(ctx, task); err != nil {
		return "", time.Time{}, fmt.Errorf("task-broker: save: %w", err)
	}

	s.bg.Add(1)
	// context.Background() is intentional: the request context expires when the RPC
	// returns, but task execution must outlive it.
	go func() { //nolint:contextcheck,gosec // G118: background ctx required for long-running work
		defer s.bg.Done()
		s.executeAsync(context.Background(), task.TaskID, agents)
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
	return task.CompletedAt, nil
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
	_ = s.repo.Update(ctx, task)
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
