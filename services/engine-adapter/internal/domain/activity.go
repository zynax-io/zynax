// SPDX-License-Identifier: Apache-2.0

// Package domain contains the core port definitions and value types for the
// engine-adapter service. It has zero imports from api or infrastructure layers.
package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// ActivityInput carries the parameters for a single capability execution.
type ActivityInput struct {
	CapabilityName string
	InputPayload   []byte
	WorkflowID     string
	TimeoutSeconds int32
}

// ActivityResult carries the outcome of a completed capability execution.
type ActivityResult struct {
	EventType string
	Payload   []byte
}

// grpcCallTimeout is the per-call deadline for outgoing gRPC requests to task-broker.
var grpcCallTimeout = 30 * time.Second

// CapabilityDispatcher dispatches capability activities to the task broker.
// It is a plain Go struct — Temporal registers it as an activity source in
// the infrastructure layer; no Temporal SDK is imported here (ADR-015).
type CapabilityDispatcher struct {
	broker       zynaxv1.TaskBrokerServiceClient
	pollInterval time.Duration
}

// NewCapabilityDispatcher constructs a dispatcher backed by the given broker client.
func NewCapabilityDispatcher(broker zynaxv1.TaskBrokerServiceClient) *CapabilityDispatcher {
	return &CapabilityDispatcher{broker: broker, pollInterval: 500 * time.Millisecond}
}

// DispatchCapabilityActivity submits a task to the broker and polls until terminal.
// It is a plain Go method registered as a Temporal activity function in infrastructure/.
func (d *CapabilityDispatcher) DispatchCapabilityActivity(ctx context.Context, in ActivityInput) (*ActivityResult, error) {
	taskID, err := d.dispatch(ctx, in)
	if err != nil {
		return nil, err
	}
	return d.poll(ctx, taskID, in.CapabilityName)
}

func (d *CapabilityDispatcher) dispatch(ctx context.Context, in ActivityInput) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
	defer cancel()
	resp, err := d.broker.DispatchTask(callCtx, &zynaxv1.DispatchTaskRequest{
		Task: &zynaxv1.WorkflowTask{
			WorkflowId:     in.WorkflowID,
			CapabilityName: in.CapabilityName,
			InputPayload:   in.InputPayload,
			TimeoutSeconds: in.TimeoutSeconds,
		},
	})
	if err != nil {
		return "", fmt.Errorf("engine-adapter: dispatch capability %q: %w", in.CapabilityName, err)
	}
	return resp.GetTaskId(), nil
}

func (d *CapabilityDispatcher) poll(ctx context.Context, taskID, capabilityName string) (*ActivityResult, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("engine-adapter: %w", ctx.Err())
		case <-time.After(d.pollInterval):
		}

		callCtx, callCancel := context.WithTimeout(ctx, grpcCallTimeout)
		task, err := d.broker.GetTask(callCtx, &zynaxv1.GetTaskRequest{TaskId: taskID})
		callCancel()
		if err != nil {
			return nil, fmt.Errorf("engine-adapter: get task %q: %w", taskID, err)
		}

		result, done, err := handleStatus(task, capabilityName)
		if done {
			return result, err
		}
	}
}

// handleStatus inspects the task status and returns (result, done, err).
// done=false means the task is still in progress; caller should poll again.
func handleStatus(task *zynaxv1.WorkflowTask, capabilityName string) (*ActivityResult, bool, error) {
	switch task.GetStatus() {
	case zynaxv1.TaskStatus_TASK_STATUS_COMPLETED:
		result, err := extractResult(task.GetResultPayload(), capabilityName)
		return result, true, err

	case zynaxv1.TaskStatus_TASK_STATUS_FAILED:
		msg := "unknown error"
		if e := task.GetError(); e != nil {
			msg = e.GetMessage()
		}
		return nil, true, fmt.Errorf("engine-adapter: capability %q failed: %s", capabilityName, msg)

	case zynaxv1.TaskStatus_TASK_STATUS_CANCELLED:
		return nil, true, fmt.Errorf("engine-adapter: capability %q task cancelled", capabilityName)

	default:
		return nil, false, nil
	}
}

// extractResult reads the "_event" key from a JSON result payload.
// If absent or the payload is empty, it defaults to "<capability>.completed".
func extractResult(payload []byte, capabilityName string) (*ActivityResult, error) {
	defaultEvent := capabilityName + ".completed"
	if len(payload) == 0 {
		return &ActivityResult{EventType: defaultEvent, Payload: payload}, nil
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("engine-adapter: unmarshal result payload: %w", err)
	}

	eventType := defaultEvent
	if raw, ok := envelope["_event"]; ok {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil && s != "" {
			eventType = s
		}
	}

	return &ActivityResult{EventType: eventType, Payload: payload}, nil
}
