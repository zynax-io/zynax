// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/zynax-io/zynax/libs/zynaxevents"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

type capturingBusClient struct {
	events []zynaxevents.CloudEvent
	err    error
}

func (c *capturingBusClient) Publish(_ context.Context, event zynaxevents.CloudEvent) (string, error) {
	c.events = append(c.events, event)
	return "TEST_STREAM:1", c.err
}

func TestPublishTaskEvent(t *testing.T) {
	task := &domain.Task{TaskID: "task-42", WorkflowID: "wf-orch", CapabilityName: "review",
		Status: domain.TaskStatusDispatched, DispatchedTo: "agent-arch"}

	bus := &capturingBusClient{}
	p := &EventPublisher{client: bus, callTimeout: time.Second}
	p.PublishTaskEvent(context.Background(), task)

	if len(bus.events) != 1 {
		t.Fatalf("publish calls = %d, want 1", len(bus.events))
	}
	ev := bus.events[0]
	if got, want := ev.Type, "zynax.v1.task-broker.task.dispatched"; got != want {
		t.Errorf("type = %q, want %q", got, want)
	}
	// The old proto Subject (task id) was never on the wire; the task id
	// travels in the Data payload instead.
	if ev.WorkflowID != "wf-orch" || ev.CapabilityName != "review" {
		t.Errorf("workflow/capability = %q/%q", ev.WorkflowID, ev.CapabilityName)
	}
	if ev.ID == "" || ev.SpecVersion != "1.0" || len(ev.Data) == 0 {
		t.Errorf("incomplete CloudEvent envelope: %+v", ev)
	}
	if decodeData(t, ev.Data)["task_id"] != "task-42" {
		t.Error("task_id missing from event data")
	}

	// Dispatched events carry no result, so result_payload must be absent.
	if _, ok := decodeData(t, ev.Data)["result_payload"]; ok {
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

	if len(bus.events) != 1 {
		t.Fatalf("publish calls = %d, want 1", len(bus.events))
	}
	data := decodeData(t, bus.events[0].Data)
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
