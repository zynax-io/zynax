// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

type capturingBusClient struct {
	requests []*zynaxv1.PublishRequest
	err      error
}

func (c *capturingBusClient) Publish(_ context.Context, in *zynaxv1.PublishRequest, _ ...grpc.CallOption) (*zynaxv1.PublishResponse, error) {
	c.requests = append(c.requests, in)
	return &zynaxv1.PublishResponse{EventId: "ev-1"}, c.err
}

func TestPublishTaskEvent(t *testing.T) {
	task := &domain.Task{TaskID: "task-42", WorkflowID: "wf-orch", CapabilityName: "review",
		Status: domain.TaskStatusDispatched, DispatchedTo: "agent-arch"}

	bus := &capturingBusClient{}
	p := &EventPublisher{client: bus, callTimeout: time.Second}
	p.PublishTaskEvent(context.Background(), task)

	if len(bus.requests) != 1 {
		t.Fatalf("publish calls = %d, want 1", len(bus.requests))
	}
	ev := bus.requests[0].GetEvent()
	if got, want := ev.GetType(), "zynax.v1.task-broker.task.dispatched"; got != want {
		t.Errorf("type = %q, want %q", got, want)
	}
	if ev.GetSubject() != "task-42" || ev.GetWorkflowId() != "wf-orch" || ev.GetCapabilityName() != "review" {
		t.Errorf("subject/workflow/capability = %q/%q/%q", ev.GetSubject(), ev.GetWorkflowId(), ev.GetCapabilityName())
	}
	if ev.GetId() == "" || ev.GetSpecversion() != "1.0" || len(ev.GetData()) == 0 {
		t.Errorf("incomplete CloudEvent envelope: %+v", ev)
	}

	// Dispatched events carry no result, so result_payload must be absent.
	if _, ok := decodeData(t, ev.GetData())["result_payload"]; ok {
		t.Error("dispatched event must not carry result_payload")
	}

	// Best-effort by port contract: a bus error must not panic or propagate.
	failing := &EventPublisher{client: &capturingBusClient{err: fmt.Errorf("bus down")}, callTimeout: time.Second}
	failing.PublishTaskEvent(context.Background(), task)
}

// TestPublishTaskEventSurfacesResultPayload verifies that a completed task's
// result payload (the capability output) is surfaced verbatim in the CloudEvent
// data so it reaches the CLI without a DB query (#1378).
func TestPublishTaskEventSurfacesResultPayload(t *testing.T) {
	task := &domain.Task{
		TaskID: "task-9", WorkflowID: "wf-9", CapabilityName: "review",
		Status: domain.TaskStatusCompleted, DispatchedTo: "agent-ollama",
		ResultPayload: []byte(`{"completion":"LGTM, ship it"}`),
	}
	bus := &capturingBusClient{}
	p := &EventPublisher{client: bus, callTimeout: time.Second}
	p.PublishTaskEvent(context.Background(), task)

	if len(bus.requests) != 1 {
		t.Fatalf("publish calls = %d, want 1", len(bus.requests))
	}
	data := decodeData(t, bus.requests[0].GetEvent().GetData())
	if got, want := data["result_payload"], `{"completion":"LGTM, ship it"}`; got != want {
		t.Errorf("result_payload = %q, want %q", got, want)
	}
}

func decodeData(t *testing.T, raw []byte) map[string]string {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode event data: %v", err)
	}
	return m
}
