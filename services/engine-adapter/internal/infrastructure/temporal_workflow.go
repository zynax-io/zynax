// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
)

const (
	dispatchCapabilityActivityName = "DispatchCapabilityActivity"
	publishEventActivityName       = "PublishLifecycleEventActivity"
	// defaultActivityTimeout is the StartToClose timeout applied to a capability
	// dispatch when the workflow action declares no explicit timeout. It must be
	// agent-scale: real capabilities (LLM inference, CI runs, git ops) routinely
	// take longer than a handful of seconds, and a too-short cap truncates the
	// activity to a nil result — the capability's output never reaches the
	// completion event, so `zynax result` reports "no result payload". 5 minutes
	// covers the slowest shipped capability (codereview, timeout_seconds: 300)
	// with headroom; capabilities needing more should declare an action timeout.
	defaultActivityTimeout = 5 * time.Minute
)

// DefaultActivityMaxAttempts is the maximum number of retry attempts for
// DispatchCapabilityActivity before Temporal marks the activity as permanently
// failed. Overridable from ZYNAX_ENGINE_MAX_ACTIVITY_ATTEMPTS in main before
// the Temporal worker is started.
var DefaultActivityMaxAttempts int32 = 3

// nonRetryableActivityErrors lists Temporal ApplicationError type names that
// must not be retried. These strings must match the Type field set by the
// activity when wrapping permanent domain errors as temporal.ApplicationError.
var nonRetryableActivityErrors = []string{
	"ErrCapabilityNotFound",
	"ErrTaskTerminal",
	"ErrInvalidArgument",
}

// IRInterpreterWorkflow is the Temporal workflow function registered by the worker
// in cmd/engine-adapter/main.go. It bridges Temporal's workflow.Context to
// domain.IRInterpreter.Run via the ActivityExecutor and EventPublisher port interfaces.
func IRInterpreterWorkflow(ctx workflow.Context, ir *zynaxv1.WorkflowIR) (map[string]string, error) {
	exec := &temporalActivityExecutor{ctx: ctx}
	pub := &temporalEventPublisher{ctx: ctx}
	// context.Background() is intentional: workflow.Context cannot be converted to
	// context.Context without breaking Temporal's replay determinism (ADR-015).
	// IRInterpreter.Run performs no I/O itself; all I/O is delegated to exec/pub,
	// which receive workflow.Context via their struct fields.
	outputs, err := (&domain.IRInterpreter{}).Run(context.Background(), ir, exec, pub)
	if err != nil {
		return nil, fmt.Errorf("engine-adapter: %w", err)
	}
	// The resolved workflow outputs are the Temporal workflow result, surfaced on
	// WorkflowRun.outputs by GetStatus (ADR-042, M7.U).
	return outputs, nil
}

// temporalActivityExecutor implements domain.ActivityExecutor by scheduling
// the registered DispatchCapabilityActivity via workflow.ExecuteActivity.
type temporalActivityExecutor struct {
	ctx workflow.Context
}

// DispatchCapability schedules the DispatchCapabilityActivity and waits for its result.
func (e *temporalActivityExecutor) DispatchCapability(_ context.Context, in domain.ActivityInput) (*domain.ActivityResult, error) {
	timeout := time.Duration(in.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultActivityTimeout
	}
	actCtx := workflow.WithActivityOptions(e.ctx, workflow.ActivityOptions{
		TaskQueue:           workflow.GetInfo(e.ctx).TaskQueueName,
		StartToCloseTimeout: timeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second,
			BackoffCoefficient:     2.0,
			MaximumInterval:        30 * time.Second,
			MaximumAttempts:        DefaultActivityMaxAttempts,
			NonRetryableErrorTypes: nonRetryableActivityErrors,
		},
	})
	var result domain.ActivityResult
	if err := workflow.ExecuteActivity(actCtx, dispatchCapabilityActivityName, in).Get(e.ctx, &result); err != nil {
		return nil, fmt.Errorf("engine-adapter: %w", err)
	}
	return &result, nil
}

// temporalEventPublisher implements domain.EventPublisher by scheduling a
// PublishLifecycleEventActivity. Publication is best-effort: activity errors
// are silently discarded so that event bus unavailability does not interrupt
// the state machine.
type temporalEventPublisher struct {
	ctx workflow.Context
}

// Publish schedules the lifecycle event activity; errors are suppressed (best-effort).
// payload carries the typed terminal {"outputs": {...}} JSON on the completed
// event (nil otherwise) and is forwarded to the activity as the CloudEvent data.
func (p *temporalEventPublisher) Publish(_ context.Context, eventType, workflowID, stateID string, payload []byte) error {
	actCtx := workflow.WithActivityOptions(p.ctx, workflow.ActivityOptions{
		TaskQueue:           workflow.GetInfo(p.ctx).TaskQueueName,
		StartToCloseTimeout: 5 * time.Second,
	})
	_ = workflow.ExecuteActivity(actCtx, publishEventActivityName, eventType, workflowID, stateID, payload).Get(p.ctx, nil)
	return nil
}
