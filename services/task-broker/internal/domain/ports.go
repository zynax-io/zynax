// SPDX-License-Identifier: Apache-2.0

package domain

import "context"

// TaskRepository persists and retrieves task records.
type TaskRepository interface {
	Save(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, taskID string) (*Task, error)
	Update(ctx context.Context, task *Task) error
	List(ctx context.Context, filter ListFilter) (ListResult, error)
}

// AgentFinder resolves which agents declare a given capability.
type AgentFinder interface {
	FindByCapability(ctx context.Context, capabilityName string) ([]AgentInfo, error)
}

// CapabilityExecutor invokes a capability on a specific agent and returns the outcome.
// resultPayload is non-nil on success; taskErr is non-nil on failure; err signals infra errors.
type CapabilityExecutor interface {
	Execute(ctx context.Context, agent AgentInfo, task *Task) (resultPayload []byte, taskErr *TaskError, err error)
}

// TaskEventPublisher publishes task lifecycle events to the event bus so a
// parallel capability fan-out is observable and collectable over the bus
// (ADR-022, EPIC #881 O5). Publication is best-effort: implementations must
// not block task progress and must swallow (log) delivery errors — event-bus
// unavailability never fails a task.
type TaskEventPublisher interface {
	PublishTaskEvent(ctx context.Context, task *Task)
}
