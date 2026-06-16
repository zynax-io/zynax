// SPDX-License-Identifier: Apache-2.0

// Package domain contains the core port definitions and value types for the
// engine-adapter service. It has zero imports from api or infrastructure layers.
package domain

import "time"

// WorkflowStatus mirrors the proto enum ordinals so the domain layer can
// compare statuses without importing proto-generated code.
// Ordinals are permanent — never reorder (matches engine_adapter.proto).
type WorkflowStatus int32

// Workflow status constants mirror engine_adapter.proto WorkflowStatus ordinals.
// Ordinals are permanent — never reorder or reassign (ADR-001 §backward-compat).
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
//
// EventID carries the engine's monotonically increasing history event ID. It
// gives consumers a total order over the stream even when timestamps collide,
// satisfying the ordered-event-stream contract for history long-poll (#468).
type WorkflowEvent struct {
	RunID     string
	EventID   int64
	EventType string
	FromState string
	ToState   string
	Status    WorkflowStatus
	Payload   []byte
	Timestamp time.Time
}
