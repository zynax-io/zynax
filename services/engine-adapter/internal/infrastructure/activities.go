// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/zynax-io/zynax/libs/zynaxobs"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// EventBusPublisher is the minimal interface this package requires for publishing
// lifecycle events. It is satisfied by the generated zynaxv1.EventBusServiceClient
// and can be replaced with a test stub without a live gRPC server.
type EventBusPublisher interface {
	Publish(ctx context.Context, in *zynaxv1.PublishRequest, opts ...grpc.CallOption) (*zynaxv1.PublishResponse, error)
}

// ActivityWorker holds cross-cutting dependencies for Temporal activities and
// exposes them as receiver methods so they can be registered with a Temporal worker.
type ActivityWorker struct {
	EventBus EventBusPublisher
}

// lifecycleTopicPrefix is the engine-adapter entity prefix per the platform
// topic taxonomy "zynax.<version>.<service>.<entity>.<event_type>" (root AGENTS.md).
const lifecycleTopicPrefix = "zynax.v1.engine-adapter.workflow."

// lifecycleTopic maps the interpreter's canonical lifecycle event type onto
// the platform topic taxonomy by replacing the interpreter's "zynax.workflow."
// family prefix with the engine-adapter entity prefix:
//
//	"zynax.workflow.state.entered" → "zynax.v1.engine-adapter.workflow.state.entered"
//	"zynax.workflow.completed"     → "zynax.v1.engine-adapter.workflow.completed"
//
// Before #1149 the interpreter event type was appended verbatim, double-prefixing
// the topic (e.g. "zynax.v1.engine-adapter.workflow.zynax.workflow.completed")
// and deriving overlapping JetStream streams on the event-bus side, which made
// the terminal completed/failed events undeliverable (NATS err 10065).
func lifecycleTopic(eventType string) string {
	return lifecycleTopicPrefix + strings.TrimPrefix(eventType, "zynax.workflow.")
}

// PublishLifecycleEventActivity is registered with the Temporal worker and called
// by IRInterpreterWorkflow to emit workflow lifecycle events to EventBusService.
// Publication is best-effort: errors are logged but not returned so that event-bus
// unavailability never interrupts the workflow state machine.
//
// Topic format: zynax.v1.engine-adapter.workflow.<event_type> (see lifecycleTopic).
func (a *ActivityWorker) PublishLifecycleEventActivity(ctx context.Context, eventType, workflowID, stateID string) error {
	topic := lifecycleTopic(eventType)

	req := &zynaxv1.PublishRequest{
		Event: &zynaxv1.CloudEvent{
			Id:              uuid.New().String(),
			Source:          fmt.Sprintf("/zynax/engine-adapter/%s", workflowID),
			Specversion:     "1.0",
			Type:            topic,
			Datacontenttype: "application/json",
			Subject:         stateID,
			Time:            timestamppb.New(time.Now()),
			WorkflowId:      workflowID,
		},
	}

	resp, err := a.EventBus.Publish(ctx, req)
	if err != nil {
		// Best-effort: log the error but do not fail the activity so that
		// event-bus unavailability never interrupts workflow execution.
		// Surface the failure in Prometheus as well (M5.D #483 counter wiring).
		zynaxobs.EventPublishFailed(eventType)
		slog.Warn("lifecycle event publish failed",
			"event_type", eventType,
			"workflow_id", workflowID,
			"state_id", stateID,
			"topic", topic,
			"err", err,
		)
		return nil
	}

	slog.Debug("lifecycle event published",
		"event_type", eventType,
		"workflow_id", workflowID,
		"state_id", stateID,
		"topic", topic,
		"event_id", resp.GetEventId(),
	)
	return nil
}
