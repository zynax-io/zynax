// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/libs/zynaxevents"
)

// stubEventBusPublisher is a test stub that records calls to Publish.
type stubEventBusPublisher struct {
	publishErr error
	events     []zynaxevents.CloudEvent
}

func (s *stubEventBusPublisher) Publish(_ context.Context, event zynaxevents.CloudEvent) (string, error) {
	s.events = append(s.events, event)
	if s.publishErr != nil {
		return "", s.publishErr
	}
	return "TEST_STREAM:1", nil
}

func newTestActivityWorker(stub *stubEventBusPublisher) *ActivityWorker {
	return &ActivityWorker{EventBus: stub}
}

func TestPublishLifecycleEventActivity_TopicFormat(t *testing.T) {
	stub := &stubEventBusPublisher{}
	w := newTestActivityWorker(stub)

	err := w.PublishLifecycleEventActivity(context.Background(), "submitted", "wf-42", "state-1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.events) != 1 {
		t.Fatalf("expected 1 Publish call, got %d", len(stub.events))
	}
	got := stub.events[0].Type
	want := "zynax.v1.engine-adapter.workflow.submitted"
	if got != want {
		t.Errorf("event type = %q; want %q", got, want)
	}
}

func TestPublishLifecycleEventActivity_WorkflowIDInEvent(t *testing.T) {
	stub := &stubEventBusPublisher{}
	w := newTestActivityWorker(stub)

	err := w.PublishLifecycleEventActivity(context.Background(), "running", "wf-99", "state-2", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.events) != 1 {
		t.Fatalf("expected 1 Publish call, got %d", len(stub.events))
	}
	ev := stub.events[0]
	if ev.WorkflowID != "wf-99" {
		t.Errorf("workflow_id = %q; want %q", ev.WorkflowID, "wf-99")
	}
	// The old proto Subject (stateID) was always dropped before the wire
	// envelope by the facade — the direct path has no Subject attribute.
	if ev.SpecVersion != "1.0" {
		t.Errorf("specversion = %q; want 1.0", ev.SpecVersion)
	}
	if !strings.HasPrefix(ev.Source, "/zynax/engine-adapter/wf-99") {
		t.Errorf("source = %q; want prefix /zynax/engine-adapter/wf-99", ev.Source)
	}
}

func TestPublishLifecycleEventActivity_AllEventTypes(t *testing.T) {
	events := []string{"submitted", "running", "failed", "completed"}
	for _, eventType := range events {
		t.Run(eventType, func(t *testing.T) {
			stub := &stubEventBusPublisher{}
			w := newTestActivityWorker(stub)
			err := w.PublishLifecycleEventActivity(context.Background(), eventType, "wf-1", "", nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(stub.events) != 1 {
				t.Fatalf("expected 1 Publish call, got %d", len(stub.events))
			}
			wantTopic := "zynax.v1.engine-adapter.workflow." + eventType
			if stub.events[0].Type != wantTopic {
				t.Errorf("topic = %q; want %q", stub.events[0].Type, wantTopic)
			}
		})
	}
}

// TestPublishLifecycleEventActivity_InterpreterEventTypes_NoDoublePrefix is
// the regression test for #1149: the interpreter emits canonical lifecycle
// event types ("zynax.workflow.…"), and the activity must map them onto the
// topic taxonomy instead of appending them verbatim (which double-prefixed the
// topic to "zynax.v1.engine-adapter.workflow.zynax.workflow.…" and derived
// overlapping JetStream streams on the event-bus side).
func TestPublishLifecycleEventActivity_InterpreterEventTypes_NoDoublePrefix(t *testing.T) {
	cases := map[string]string{
		"zynax.workflow.state.entered": "zynax.v1.engine-adapter.workflow.state.entered",
		"zynax.workflow.state.exited":  "zynax.v1.engine-adapter.workflow.state.exited",
		"zynax.workflow.completed":     "zynax.v1.engine-adapter.workflow.completed",
		"zynax.workflow.failed":        "zynax.v1.engine-adapter.workflow.failed",
	}
	for eventType, wantTopic := range cases {
		t.Run(eventType, func(t *testing.T) {
			stub := &stubEventBusPublisher{}
			w := newTestActivityWorker(stub)
			if err := w.PublishLifecycleEventActivity(context.Background(), eventType, "wf-1149", "s1", nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(stub.events) != 1 {
				t.Fatalf("expected 1 Publish call, got %d", len(stub.events))
			}
			got := stub.events[0].Type
			if got != wantTopic {
				t.Errorf("topic = %q; want %q", got, wantTopic)
			}
			if strings.Contains(got, "workflow.zynax.workflow") {
				t.Errorf("topic %q is double-prefixed (#1149 regression)", got)
			}
		})
	}
}

func TestPublishLifecycleEventActivity_EventBusError_BestEffort(t *testing.T) {
	// Event bus errors must not be propagated — activity is best-effort.
	stub := &stubEventBusPublisher{publishErr: errors.New("connection refused")}
	w := newTestActivityWorker(stub)

	err := w.PublishLifecycleEventActivity(context.Background(), "failed", "wf-3", "state-3", nil)
	if err != nil {
		t.Errorf("expected nil error on publish failure (best-effort), got: %v", err)
	}
}

func TestPublishLifecycleEventActivity_CloudEventFields(t *testing.T) {
	stub := &stubEventBusPublisher{}
	w := newTestActivityWorker(stub)

	err := w.PublishLifecycleEventActivity(context.Background(), "completed", "wf-5", "end-state", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ev := stub.events[0]
	// id must be a non-empty UUID
	if ev.ID == "" {
		t.Error("CloudEvent id must be non-empty")
	}
	// datacontenttype
	if ev.DataContentType != "application/json" {
		t.Errorf("datacontenttype = %q; want application/json", ev.DataContentType)
	}
	// time must be set (client attribute only — never marshaled to the wire)
	if ev.Time.IsZero() {
		t.Error("CloudEvent time must be set")
	}
}
