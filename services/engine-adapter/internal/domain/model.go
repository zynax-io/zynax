// SPDX-License-Identifier: Apache-2.0
package domain

import "time"

// WorkflowStatus mirrors the proto enum ordinals so the domain layer can
// compare statuses without importing proto-generated code.
// Ordinals are permanent — never reorder (matches engine_adapter.proto).
type WorkflowStatus int32

const (
	WorkflowStatusUnspecified WorkflowStatus = 0
	WorkflowStatusPending     WorkflowStatus = 1
	WorkflowStatusRunning     WorkflowStatus = 2
	WorkflowStatusCompleted   WorkflowStatus = 3
	WorkflowStatusFailed      WorkflowStatus = 4
	WorkflowStatusCancelled   WorkflowStatus = 5
)

// IsTerminal reports whether s is a terminal (immutable) status.
func (s WorkflowStatus) IsTerminal() bool {
	return s == WorkflowStatusCompleted ||
		s == WorkflowStatusFailed ||
		s == WorkflowStatusCancelled
}

// WorkflowRun is the adapter's record for a submitted workflow execution.
type WorkflowRun struct {
	RunID              string
	WorkflowID         string
	Namespace          string
	Status             WorkflowStatus
	CurrentState       string
	Engine             string
	Labels             map[string]string
	SubmittedAt        time.Time
	StartedAt          time.Time
	FinishedAt         time.Time
	CancellationReason string
}

// WorkflowEvent is emitted when the workflow transitions state or reaches
// a terminal condition. Streamed by Watch and published to the event bus.
type WorkflowEvent struct {
	RunID     string
	EventType string
	FromState string
	ToState   string
	Status    WorkflowStatus
	Payload   []byte
	Timestamp time.Time
}
