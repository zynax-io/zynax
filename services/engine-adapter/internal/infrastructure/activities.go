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

	"github.com/zynax-io/zynax/libs/zynaxevents"
	"github.com/zynax-io/zynax/libs/zynaxobs"
)

// EventBusPublisher is the minimal interface this package requires for publishing
// lifecycle events directly to NATS JetStream through the shared events client
// (ADR-046 — the EventBusService gRPC facade is deprecated). It is satisfied by
// *zynaxevents.Client and can be replaced with a test stub without a live broker.
type EventBusPublisher interface {
	Publish(ctx context.Context, event zynaxevents.CloudEvent) (string, error)
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
// by IRInterpreterWorkflow to emit workflow lifecycle events straight to
// JetStream via the shared events client (ADR-046). Publication is best-effort:
// errors are logged but not returned so that broker unavailability never
// interrupts the workflow state machine.
//
// Topic format: zynax.v1.engine-adapter.workflow.<event_type> (see lifecycleTopic).
// The old proto Subject (stateID) and Time attributes were always dropped by the
// facade before the wire envelope — the direct path keeps the wire bytes
// identical (golden-gated); stateID stays in the structured logs only.
func (a *ActivityWorker) PublishLifecycleEventActivity(ctx context.Context, eventType, workflowID, stateID string, payload []byte) error {
	topic := lifecycleTopic(eventType)

	event := zynaxevents.CloudEvent{
		ID:              uuid.New().String(),
		Source:          fmt.Sprintf("/zynax/engine-adapter/%s", workflowID),
		SpecVersion:     "1.0",
		Type:            topic,
		DataContentType: "application/json",
		Time:            time.Now(),
		WorkflowID:      workflowID,
		// Typed terminal payload {"outputs": {...}} on the completed event
		// (nil for transition events) (ADR-042, M7.U).
		Data: payload,
	}

	eventID, err := a.EventBus.Publish(ctx, event)
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
		"event_id", eventID,
	)
	return nil
}
