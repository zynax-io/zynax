// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/zynax-io/zynax/libs/zynaxobs"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

// EventBusClient is the minimal interface this package requires for publishing
// task lifecycle events. It is satisfied by the generated
// zynaxv1.EventBusServiceClient and can be replaced with a test stub.
type EventBusClient interface {
	Publish(ctx context.Context, in *zynaxv1.PublishRequest, opts ...grpc.CallOption) (*zynaxv1.PublishResponse, error)
}

// taskTopicPrefix is the task-broker entity prefix per the platform topic
// taxonomy "zynax.<version>.<service>.<entity>.<event_type>" (root AGENTS.md).
const taskTopicPrefix = "zynax.v1.task-broker.task."

// EventPublisher implements domain.TaskEventPublisher over EventBusService.
// Publication is best-effort by port contract: errors are logged and counted
// (Prometheus) but never propagated, so event-bus unavailability never fails
// a task (mirrors the engine-adapter lifecycle publisher).
type EventPublisher struct {
	client      EventBusClient
	callTimeout time.Duration
}

// NewEventPublisher dials the event bus and returns a TaskEventPublisher.
// callTimeout is the per-call Publish deadline; creds controls transport
// security. The returned cleanup closes the connection (defer it).
func NewEventPublisher(addr string, callTimeout time.Duration, creds credentials.TransportCredentials) (*EventPublisher, func(), error) {
	conn, err := grpc.NewClient(addr, tracingDialOpts(creds)...)
	if err != nil {
		return nil, func() {}, fmt.Errorf("task-broker: event-bus dial: %w", err)
	}
	return &EventPublisher{
		client:      zynaxv1.NewEventBusServiceClient(conn),
		callTimeout: callTimeout,
	}, func() { _ = conn.Close() }, nil
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
	data, err := json.Marshal(map[string]string{
		"task_id":         task.TaskID,
		"workflow_id":     task.WorkflowID,
		"capability_name": task.CapabilityName,
		"status":          task.Status.String(),
		"dispatched_to":   task.DispatchedTo,
	})
	if err != nil {
		p.warn(eventType, task, err)
		return
	}

	callCtx, cancel := context.WithTimeout(ctx, p.callTimeout)
	defer cancel()
	_, err = p.client.Publish(callCtx, &zynaxv1.PublishRequest{
		Event: &zynaxv1.CloudEvent{
			Id:              id,
			Source:          "/zynax/task-broker/" + task.WorkflowID,
			Specversion:     "1.0",
			Type:            eventType,
			Datacontenttype: "application/json",
			Subject:         task.TaskID,
			Time:            timestamppb.New(time.Now()),
			Data:            data,
			WorkflowId:      task.WorkflowID,
			CapabilityName:  task.CapabilityName,
		},
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
