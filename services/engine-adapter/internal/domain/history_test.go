// SPDX-License-Identifier: Apache-2.0

package domain

import "testing"

func TestHistoryEventStatus(t *testing.T) {
	cases := []struct {
		name string
		in   HistoryEventType
		want WorkflowStatus
	}{
		{"started maps to running", HistoryEventWorkflowStarted, WorkflowStatusRunning},
		{"completed", HistoryEventWorkflowCompleted, WorkflowStatusCompleted},
		{"failed", HistoryEventWorkflowFailed, WorkflowStatusFailed},
		{"timed_out maps to failed", HistoryEventWorkflowTimedOut, WorkflowStatusFailed},
		{"canceled maps to cancelled", HistoryEventWorkflowCanceled, WorkflowStatusCancelled},
		{"terminated maps to cancelled", HistoryEventWorkflowTerminated, WorkflowStatusCancelled},
		{"continued-as-new maps to completed", HistoryEventWorkflowContinuedNew, WorkflowStatusCompleted},
		{"unspecified is a running progress event", HistoryEventUnspecified, WorkflowStatusRunning},
		{"unknown progress event maps to running", HistoryEventType(99), WorkflowStatusRunning},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HistoryEventStatus(tc.in); got != tc.want {
				t.Errorf("HistoryEventStatus(%v) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsTerminalHistoryEvent(t *testing.T) {
	cases := []struct {
		name string
		in   HistoryEventType
		want bool
	}{
		{"completed is terminal", HistoryEventWorkflowCompleted, true},
		{"failed is terminal", HistoryEventWorkflowFailed, true},
		{"timed_out is terminal", HistoryEventWorkflowTimedOut, true},
		{"canceled is terminal", HistoryEventWorkflowCanceled, true},
		{"terminated is terminal", HistoryEventWorkflowTerminated, true},
		{"continued-as-new is terminal", HistoryEventWorkflowContinuedNew, true},
		{"started is not terminal", HistoryEventWorkflowStarted, false},
		{"unspecified is not terminal", HistoryEventUnspecified, false},
		{"unknown progress event is not terminal", HistoryEventType(42), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTerminalHistoryEvent(tc.in); got != tc.want {
				t.Errorf("IsTerminalHistoryEvent(%v) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}
