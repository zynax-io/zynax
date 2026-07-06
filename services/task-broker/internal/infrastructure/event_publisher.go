// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	nats "github.com/nats-io/nats.go"

	"github.com/zynax-io/zynax/libs/zynaxevents"
	"github.com/zynax-io/zynax/libs/zynaxobs"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

// taskTopicPrefix is the task-broker entity prefix per the platform topic
// taxonomy "zynax.<version>.<service>.<entity>.<event_type>" (root AGENTS.md).
const taskTopicPrefix = "zynax.v1.task-broker.task."

// EventBusClient is the minimal interface this package requires for publishing
// task lifecycle events directly to NATS JetStream via the shared events
// client (ADR-046 — the EventBusService gRPC facade is deprecated). Satisfied
// by *zynaxevents.Client; replaced with a test stub in unit tests.
type EventBusClient interface {
	Publish(ctx context.Context, event zynaxevents.CloudEvent) (string, error)
}

// EventPublisher implements domain.TaskEventPublisher over direct JetStream.
// Publication is best-effort by port contract: errors are logged and counted
// (Prometheus) but never propagated, so broker unavailability never fails
// a task (mirrors the engine-adapter lifecycle publisher).
type EventPublisher struct {
	client      EventBusClient
	callTimeout time.Duration
}

// NewEventPublisher connects the shared events client and returns a
// TaskEventPublisher. callTimeout is the per-call Publish deadline.
// RetryOnFailedConnect keeps startup broker-independent (the old gRPC dial
// was lazy; a NATS-less profile must still boot) — publishes stay best-effort
// until the broker is reachable. The returned cleanup drains the connection.
func NewEventPublisher(natsURL string, callTimeout time.Duration) (*EventPublisher, func(), error) {
	client, err := zynaxevents.New(natsURL,
		nats.RetryOnFailedConnect(true), nats.MaxReconnects(-1))
	if err != nil {
		return nil, func() {}, fmt.Errorf("task-broker: events client: %w", err)
	}
	return &EventPublisher{
		client:      client,
		callTimeout: callTimeout,
	}, client.Close, nil
}

// PublishTaskEvent emits one CloudEvent for a task lifecycle transition.
// Topic format: zynax.v1.task-broker.task.<status> (e.g. ".dispatched",
// ".completed", ".failed") so a workflow fan-out is observable and
// collectable over the durable bus (ADR-022, EPIC #881 O5).
func (p *EventPublisher) PublishTaskEvent(ctx context.Context, task *domain.Task) {
	eventType := taskTopicPrefix + strings.ToLower(task.Status.String())

	id, err := newRequestID()
	if err != nil {
		p.warn(eventType, task, err)
		return
	}
	fields := map[string]string{
		"task_id":         task.TaskID,
		"workflow_id":     task.WorkflowID,
		"capability_name": task.CapabilityName,
		"status":          task.Status.String(),
		"dispatched_to":   task.DispatchedTo,
	}
	// Surface the capability output on completion so it is observable over the
	// bus and reaches `zynax logs`/`zynax result` without a DB query (#1378).
	// The payload is the executor's raw JSON (e.g. {"completion": "..."}); it is
	// omitted for non-terminal/failed events that carry no result.
	if len(task.ResultPayload) > 0 {
		fields["result_payload"] = string(task.ResultPayload)
	}
	data, err := json.Marshal(fields)
	if err != nil {
		p.warn(eventType, task, err)
		return
	}

	callCtx, cancel := context.WithTimeout(ctx, p.callTimeout)
	defer cancel()
	// The old proto Subject (task id) and Time attributes were always dropped
	// by the facade before the wire envelope — the direct path keeps the wire
	// bytes identical (golden-gated); the task id stays in the Data payload.
	_, err = p.client.Publish(callCtx, zynaxevents.CloudEvent{
		ID:              id,
		Source:          "/zynax/task-broker/" + task.WorkflowID,
		SpecVersion:     "1.0",
		Type:            eventType,
		DataContentType: "application/json",
		Time:            time.Now(),
		Data:            data,
		WorkflowID:      task.WorkflowID,
		CapabilityName:  task.CapabilityName,
	})
	if err != nil {
		p.warn(eventType, task, err)
	}
}

func (p *EventPublisher) warn(eventType string, task *domain.Task, err error) {
	zynaxobs.EventPublishFailed(eventType)
	slog.Warn("task event publish failed",
		"event_type", eventType,
		"task_id", task.TaskID,
		"workflow_id", task.WorkflowID,
		"err", err,
	)
}
