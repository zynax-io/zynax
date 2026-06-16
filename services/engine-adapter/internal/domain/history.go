// SPDX-License-Identifier: Apache-2.0

package domain

// HistoryEventType mirrors the ordinals of Temporal's history EventType enum so
// the domain layer can classify execution-history events without importing the
// Temporal SDK (the domain package has zero external imports — ADR-001).
//
// Ordinals are permanent and match temporal.api.enums.v1.EventType. Only the
// subset the engine-adapter reasons about is named here; every other ordinal is
// treated as a non-terminal, status-neutral progress event.
type HistoryEventType int32

// History event ordinals mirrored from temporal.api.enums.v1.EventType.
const (
	HistoryEventUnspecified          HistoryEventType = 0
	HistoryEventWorkflowStarted      HistoryEventType = 1
	HistoryEventWorkflowCompleted    HistoryEventType = 2
	HistoryEventWorkflowFailed       HistoryEventType = 3
	HistoryEventWorkflowTimedOut     HistoryEventType = 4
	HistoryEventWorkflowCanceled     HistoryEventType = 21
	HistoryEventWorkflowTerminated   HistoryEventType = 27
	HistoryEventWorkflowContinuedNew HistoryEventType = 28
)

// HistoryEventStatus maps a history event type to the WorkflowStatus that the
// run holds *after* the event is applied. Non-status-changing events (activity
// scheduling, task completion, timers, …) report WorkflowStatusRunning so the
// stream surfaces forward progress while the run is live.
func HistoryEventStatus(t HistoryEventType) WorkflowStatus {
	switch t {
	case HistoryEventWorkflowStarted:
		return WorkflowStatusRunning
	case HistoryEventWorkflowCompleted:
		return WorkflowStatusCompleted
	case HistoryEventWorkflowFailed, HistoryEventWorkflowTimedOut:
		return WorkflowStatusFailed
	case HistoryEventWorkflowCanceled, HistoryEventWorkflowTerminated:
		return WorkflowStatusCancelled
	case HistoryEventWorkflowContinuedNew:
		return WorkflowStatusCompleted
	default:
		return WorkflowStatusRunning
	}
}

// IsTerminalHistoryEvent reports whether the event closes the execution. The
// long-poll loop stops after emitting a terminal event so a finished run does
// not leave a consumer blocked on an iterator that will never advance.
func IsTerminalHistoryEvent(t HistoryEventType) bool {
	switch t {
	case HistoryEventWorkflowCompleted,
		HistoryEventWorkflowFailed,
		HistoryEventWorkflowTimedOut,
		HistoryEventWorkflowCanceled,
		HistoryEventWorkflowTerminated,
		HistoryEventWorkflowContinuedNew:
		return true
	default:
		return false
	}
}
