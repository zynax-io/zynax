// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
)

const (
	irInterpreterWorkflowName = "IRInterpreterWorkflow"
	temporalEngineName        = "temporal"
)

// temporalClient is a narrow interface over client.Client covering only the
// methods used by TemporalEngine, making unit tests straightforward.
type temporalClient interface {
	ExecuteWorkflow(ctx context.Context, opts client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error)
	SignalWorkflow(ctx context.Context, workflowID, runID, signalName string, arg interface{}) error
	CancelWorkflow(ctx context.Context, workflowID, runID string) error
	DescribeWorkflowExecution(ctx context.Context, workflowID, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error)
	GetWorkflowHistory(ctx context.Context, workflowID, runID string, isLongPoll bool, filterType enumspb.HistoryEventFilterType) client.HistoryEventIterator
	GetWorkflow(ctx context.Context, workflowID, runID string) client.WorkflowRun
}

// TemporalEngine implements domain.WorkflowEngine backed by the Temporal Go SDK.
// Selected when ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE=temporal (ADR-015).
type TemporalEngine struct {
	client    temporalClient
	taskQueue string
	namespace string
}

// NewTemporalEngine constructs a TemporalEngine wrapping the given Temporal client.
func NewTemporalEngine(c client.Client, taskQueue, namespace string) *TemporalEngine {
	return newTemporalEngine(c, taskQueue, namespace)
}

// newTemporalEngine accepts the narrow temporalClient interface so unit tests
// can inject a stub without implementing the full client.Client.
func newTemporalEngine(c temporalClient, taskQueue, namespace string) *TemporalEngine {
	return &TemporalEngine{
		client:    c,
		taskQueue: taskQueue,
		namespace: namespace,
	}
}

// Submit starts the IRInterpreterWorkflow for the given WorkflowIR and returns
// the domain WorkflowRun. The workflow ID from the IR is used as the run ID.
func (e *TemporalEngine) Submit(ctx context.Context, ir *zynaxv1.WorkflowIR, labels map[string]string) (*domain.WorkflowRun, error) {
	opts := client.StartWorkflowOptions{
		ID:        ir.GetWorkflowId(),
		TaskQueue: e.taskQueue,
	}
	_, err := e.client.ExecuteWorkflow(ctx, opts, irInterpreterWorkflowName, ir)
	if err != nil {
		return nil, fmt.Errorf("engine-adapter: start workflow %q: %w", ir.GetWorkflowId(), err)
	}
	return &domain.WorkflowRun{
		RunID:      ir.GetWorkflowId(),
		WorkflowID: ir.GetWorkflowId(),
		Namespace:  e.namespace,
		Status:     domain.WorkflowStatusPending,
		Engine:     temporalEngineName,
		Labels:     labels,
	}, nil
}

// Signal delivers a named event to a running workflow via a Temporal signal.
func (e *TemporalEngine) Signal(ctx context.Context, runID, eventType string, payload []byte) error {
	if err := e.client.SignalWorkflow(ctx, runID, "", eventType, payload); err != nil {
		var nf *serviceerror.NotFound
		if errors.As(err, &nf) {
			return domain.ErrExecutionNotFound
		}
		return fmt.Errorf("engine-adapter: signal workflow %q: %w", runID, err)
	}
	return nil
}

// Cancel requests graceful cancellation of a running workflow.
func (e *TemporalEngine) Cancel(ctx context.Context, runID, _ string) error {
	if err := e.client.CancelWorkflow(ctx, runID, ""); err != nil {
		var nf *serviceerror.NotFound
		if errors.As(err, &nf) {
			return domain.ErrExecutionNotFound
		}
		return fmt.Errorf("engine-adapter: cancel workflow %q: %w", runID, err)
	}
	return nil
}

// GetStatus returns current run metadata by describing the Temporal execution.
func (e *TemporalEngine) GetStatus(ctx context.Context, runID string) (*domain.WorkflowRun, error) {
	resp, err := e.client.DescribeWorkflowExecution(ctx, runID, "")
	if err != nil {
		var nf *serviceerror.NotFound
		if errors.As(err, &nf) {
			return nil, domain.ErrExecutionNotFound
		}
		return nil, fmt.Errorf("engine-adapter: describe workflow %q: %w", runID, err)
	}
	run := describeToWorkflowRun(resp, runID, e.namespace)
	// For a COMPLETED run, read the workflow result (the resolved outputs) and
	// surface it on WorkflowRun.outputs (ADR-042, M7.U). Get() returns the stored
	// result immediately for a finished workflow; gating on the terminal-completed
	// status ensures it never blocks on a still-running execution. A read failure
	// (e.g. history expired past Temporal retention) is non-fatal — status is
	// still returned, just without outputs.
	if run.Status == domain.WorkflowStatusCompleted {
		var outputs map[string]string
		if gerr := e.client.GetWorkflow(ctx, runID, "").Get(ctx, &outputs); gerr != nil {
			slog.Warn("read workflow result failed", "run_id", runID, "err", gerr)
		} else {
			run.Outputs = outputs
		}
	}
	return run, nil
}

// Watch long-polls the Temporal execution history and calls send for each
// history event in order until the workflow reaches a terminal state or ctx is
// cancelled. Long-poll replaces the previous DescribeWorkflowExecution polling
// loop: the iterator blocks server-side until new events are available, so
// transitions and capability events stream with low latency rather than at a
// fixed poll cadence (#468, EPIC L step L.1).
func (e *TemporalEngine) Watch(ctx context.Context, runID string, send func(*domain.WorkflowEvent) error) error {
	iter := e.client.GetWorkflowHistory(
		ctx, runID, "", true, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	for iter.HasNext() {
		hev, err := iter.Next()
		if err != nil {
			var nf *serviceerror.NotFound
			if errors.As(err, &nf) {
				return domain.ErrExecutionNotFound
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return fmt.Errorf("engine-adapter: %w", ctxErr)
			}
			return fmt.Errorf("engine-adapter: workflow history %q: %w", runID, err)
		}
		ev := historyEventToWorkflowEvent(hev, runID)
		if err := send(ev); err != nil {
			return fmt.Errorf("engine-adapter: send event: %w", err)
		}
		if domain.IsTerminalHistoryEvent(domain.HistoryEventType(hev.GetEventType())) {
			return nil
		}
	}
	return nil
}

// historyEventToWorkflowEvent maps a Temporal history event onto the ordered
// domain WorkflowEvent. EventID preserves the engine's total order and the
// event type drives the status the run holds after the event is applied.
func historyEventToWorkflowEvent(hev *historypb.HistoryEvent, runID string) *domain.WorkflowEvent {
	hetype := domain.HistoryEventType(hev.GetEventType())
	ts := time.Time{}
	if t := hev.GetEventTime(); t != nil {
		ts = t.AsTime()
	}
	return &domain.WorkflowEvent{
		RunID:     runID,
		EventID:   hev.GetEventId(),
		EventType: hev.GetEventType().String(),
		Status:    domain.HistoryEventStatus(hetype),
		Timestamp: ts,
	}
}

func describeToWorkflowRun(resp *workflowservice.DescribeWorkflowExecutionResponse, runID, namespace string) *domain.WorkflowRun {
	status := domain.WorkflowStatusPending
	if info := resp.GetWorkflowExecutionInfo(); info != nil {
		status = temporalStatusToDomain(info.GetStatus())
	}
	return &domain.WorkflowRun{
		RunID:      runID,
		WorkflowID: runID,
		Namespace:  namespace,
		Status:     status,
		Engine:     temporalEngineName,
	}
}

// temporalStatusToDomain maps Temporal execution status values to domain WorkflowStatus.
func temporalStatusToDomain(s enumspb.WorkflowExecutionStatus) domain.WorkflowStatus {
	switch s {
	case enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:
		return domain.WorkflowStatusRunning
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return domain.WorkflowStatusCompleted
	case enumspb.WORKFLOW_EXECUTION_STATUS_FAILED, enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return domain.WorkflowStatusFailed
	case enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED, enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		return domain.WorkflowStatusCancelled
	default:
		return domain.WorkflowStatusPending
	}
}

// compile-time assertion: TemporalEngine satisfies domain.WorkflowEngine.
var _ domain.WorkflowEngine = (*TemporalEngine)(nil)
