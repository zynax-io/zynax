// SPDX-License-Identifier: Apache-2.0
package domain

import (
	"context"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// WorkflowEngine is the port that every execution backend must satisfy.
// The gRPC handler injects a concrete engine at startup; all other domain
// code depends only on this interface — never on engine-specific packages.
// Engine selection is process-wide via ZYNAX_ENGINE_ACTIVE_ENGINE (ADR-015).
type WorkflowEngine interface {
	// Submit starts execution of a compiled workflow and returns the
	// adapter-assigned run identifier. The caller must not supply a run_id.
	Submit(ctx context.Context, ir *zynaxv1.WorkflowIR, labels map[string]string) (*WorkflowRun, error)

	// Signal delivers an external event to a running workflow.
	// Returns ErrExecutionNotFound if the run_id is unknown.
	// Returns ErrTerminalState if the workflow is already terminal.
	Signal(ctx context.Context, runID, eventType string, payload []byte) error

	// Cancel requests cancellation of a running workflow.
	// Returns ErrExecutionNotFound if the run_id is unknown.
	// Returns ErrTerminalState if the workflow is already terminal.
	Cancel(ctx context.Context, runID, reason string) error

	// GetStatus returns current run metadata.
	// Returns ErrExecutionNotFound if the run_id is unknown.
	GetStatus(ctx context.Context, runID string) (*WorkflowRun, error)

	// Watch calls send for each WorkflowEvent until the workflow reaches a
	// terminal state or ctx is cancelled. send must be called at least once
	// with a terminal-status event before Watch returns nil.
	// Returns ErrExecutionNotFound if the run_id is unknown.
	Watch(ctx context.Context, runID string, send func(*WorkflowEvent) error) error
}
