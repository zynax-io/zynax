// SPDX-License-Identifier: Apache-2.0

package domain

import "time"

// TaskStatus mirrors the proto TaskStatus enum; values are stable (ADR-001).
type TaskStatus int32

// TaskStatus values; ordinal values are permanent — never reorder or reassign (ADR-001).
const (
	TaskStatusUnspecified TaskStatus = 0
	TaskStatusPending     TaskStatus = 1
	TaskStatusDispatched  TaskStatus = 2
	TaskStatusRetrying    TaskStatus = 3
	TaskStatusCompleted   TaskStatus = 4
	TaskStatusFailed      TaskStatus = 5
	TaskStatusCancelled   TaskStatus = 6
)

// IsTerminal reports whether s is a terminal state (COMPLETED, FAILED, or CANCELLED).
func (s TaskStatus) IsTerminal() bool {
	return s == TaskStatusCompleted || s == TaskStatusFailed || s == TaskStatusCancelled
}

// String returns the proto-style name of the status.
func (s TaskStatus) String() string {
	switch s {
	case TaskStatusPending:
		return "PENDING"
	case TaskStatusDispatched:
		return "DISPATCHED"
	case TaskStatusRetrying:
		return "RETRYING"
	case TaskStatusCompleted:
		return "COMPLETED"
	case TaskStatusFailed:
		return "FAILED"
	case TaskStatusCancelled:
		return "CANCELLED"
	default:
		return "UNSPECIFIED"
	}
}

// Task is the broker's canonical task record.
type Task struct {
	TaskID         string
	WorkflowID     string
	CapabilityName string
	InputPayload   []byte
	TimeoutSeconds int32
	MaxRetries     int32
	RetryCount     int32
	Status         TaskStatus
	DispatchedTo   string
	ResultPayload  []byte
	Error          *TaskError
	CreatedAt      time.Time
	DispatchedAt   time.Time
	CompletedAt    time.Time
}

// TaskError carries structured failure information from an execution.
type TaskError struct {
	Code    string
	Message string
	Details map[string]string
}

// AgentInfo carries the routing information needed to invoke an agent.
type AgentInfo struct {
	AgentID  string
	Name     string
	Endpoint string
	// InputSchema is the registered JSON Schema (draft-07) of the requested
	// capability. The dispatch-time context-slice injection binding (ADR-028,
	// EPIC #881 O5) reads the declared {files[], max_tokens} defaults from it.
	InputSchema []byte
}

// ListFilter specifies the criteria for a ListTasks call.
// A zero-value Status field means no status filter.
type ListFilter struct {
	WorkflowID string
	Status     TaskStatus
	AgentID    string
	PageToken  string
	PageSize   int32
}

// ListResult carries one page of matching tasks.
type ListResult struct {
	Tasks         []*Task
	NextPageToken string
}
