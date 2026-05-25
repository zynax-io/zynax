// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
)

const (
	irInterpreterWorkflowName = "IRInterpreterWorkflow"
	temporalEngineName        = "temporal"
	defaultWatchPollInterval  = 2 * time.Second
)

// temporalClient is a narrow interface over client.Client covering only the
// methods used by TemporalEngine, making unit tests straightforward.
type temporalClient interface {
	ExecuteWorkflow(ctx context.Context, opts client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error)
	SignalWorkflow(ctx context.Context, workflowID, runID, signalName string, arg interface{}) error
	CancelWorkflow(ctx context.Context, workflowID, runID string) error
	DescribeWorkflowExecution(ctx context.Context, workflowID, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error)
}

// TemporalEngine implements domain.WorkflowEngine backed by the Temporal Go SDK.
// Selected when ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE=temporal (ADR-015).
type TemporalEngine struct {
	client       temporalClient
	taskQueue    string
	namespace    string
	pollInterval time.Duration
}

// NewTemporalEngine constructs a TemporalEngine wrapping the given Temporal client.
func NewTemporalEngine(c client.Client, taskQueue, namespace string) *TemporalEngine {
	return newTemporalEngine(c, taskQueue, namespace)
}

// newTemporalEngine accepts the narrow temporalClient interface so unit tests
// can inject a stub without implementing the full client.Client.
func newTemporalEngine(c temporalClient, taskQueue, namespace string) *TemporalEngine {
	return &TemporalEngine{
		client:       c,
		taskQueue:    taskQueue,
		namespace:    namespace,
		pollInterval: defaultWatchPollInterval,
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
	return describeToWorkflowRun(resp, runID, e.namespace), nil
}

// Watch polls DescribeWorkflowExecution and calls send for each status update
// until the workflow reaches a terminal state or ctx is cancelled.
func (e *TemporalEngine) Watch(ctx context.Context, runID string, send func(*domain.WorkflowEvent) error) error {
	for {
		run, err := e.GetStatus(ctx, runID)
		if err != nil {
			return err
		}
		ev := &domain.WorkflowEvent{RunID: runID, Status: run.Status, Timestamp: time.Now()}
		if err := send(ev); err != nil {
			return fmt.Errorf("engine-adapter: send event: %w", err)
		}
		if run.Status.IsTerminal() {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("engine-adapter: %w", ctx.Err())
		case <-time.After(e.pollInterval):
		}
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
