// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// stubEventBusPublisher is a test stub that records calls to Publish.
type stubEventBusPublisher struct {
	publishErr error
	reqs       []*zynaxv1.PublishRequest
	resp       *zynaxv1.PublishResponse
}

func (s *stubEventBusPublisher) Publish(_ context.Context, in *zynaxv1.PublishRequest, _ ...grpc.CallOption) (*zynaxv1.PublishResponse, error) {
	s.reqs = append(s.reqs, in)
	if s.publishErr != nil {
		return nil, s.publishErr
	}
	if s.resp != nil {
		return s.resp, nil
	}
	return &zynaxv1.PublishResponse{EventId: "test-event-id"}, nil
}

func newTestActivityWorker(stub *stubEventBusPublisher) *ActivityWorker {
	return &ActivityWorker{EventBus: stub}
}

func TestPublishLifecycleEventActivity_TopicFormat(t *testing.T) {
	stub := &stubEventBusPublisher{}
	w := newTestActivityWorker(stub)

	err := w.PublishLifecycleEventActivity(context.Background(), "submitted", "wf-42", "state-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.reqs) != 1 {
		t.Fatalf("expected 1 Publish call, got %d", len(stub.reqs))
	}
	got := stub.reqs[0].Event.GetType()
	want := "zynax.v1.engine-adapter.workflow.submitted"
	if got != want {
		t.Errorf("event type = %q; want %q", got, want)
	}
}

func TestPublishLifecycleEventActivity_WorkflowIDInEvent(t *testing.T) {
	stub := &stubEventBusPublisher{}
	w := newTestActivityWorker(stub)

	err := w.PublishLifecycleEventActivity(context.Background(), "running", "wf-99", "state-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.reqs) != 1 {
		t.Fatalf("expected 1 Publish call, got %d", len(stub.reqs))
	}
	ev := stub.reqs[0].Event
	if ev.GetWorkflowId() != "wf-99" {
		t.Errorf("workflow_id = %q; want %q", ev.GetWorkflowId(), "wf-99")
	}
	if ev.GetSubject() != "state-2" {
		t.Errorf("subject = %q; want %q", ev.GetSubject(), "state-2")
	}
	if ev.GetSpecversion() != "1.0" {
		t.Errorf("specversion = %q; want 1.0", ev.GetSpecversion())
	}
	if !strings.HasPrefix(ev.GetSource(), "/zynax/engine-adapter/wf-99") {
		t.Errorf("source = %q; want prefix /zynax/engine-adapter/wf-99", ev.GetSource())
	}
}

func TestPublishLifecycleEventActivity_AllEventTypes(t *testing.T) {
	events := []string{"submitted", "running", "failed", "completed"}
	for _, eventType := range events {
		t.Run(eventType, func(t *testing.T) {
			stub := &stubEventBusPublisher{}
			w := newTestActivityWorker(stub)
			err := w.PublishLifecycleEventActivity(context.Background(), eventType, "wf-1", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(stub.reqs) != 1 {
				t.Fatalf("expected 1 Publish call, got %d", len(stub.reqs))
			}
			wantTopic := "zynax.v1.engine-adapter.workflow." + eventType
			if stub.reqs[0].Event.GetType() != wantTopic {
				t.Errorf("topic = %q; want %q", stub.reqs[0].Event.GetType(), wantTopic)
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
			if err := w.PublishLifecycleEventActivity(context.Background(), eventType, "wf-1149", "s1"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(stub.reqs) != 1 {
				t.Fatalf("expected 1 Publish call, got %d", len(stub.reqs))
			}
			got := stub.reqs[0].Event.GetType()
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

	err := w.PublishLifecycleEventActivity(context.Background(), "failed", "wf-3", "state-3")
	if err != nil {
		t.Errorf("expected nil error on publish failure (best-effort), got: %v", err)
	}
}

func TestPublishLifecycleEventActivity_CloudEventFields(t *testing.T) {
	stub := &stubEventBusPublisher{}
	w := newTestActivityWorker(stub)

	err := w.PublishLifecycleEventActivity(context.Background(), "completed", "wf-5", "end-state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ev := stub.reqs[0].Event
	// id must be a non-empty UUID
	if ev.GetId() == "" {
		t.Error("CloudEvent id must be non-empty")
	}
	// datacontenttype
	if ev.GetDatacontenttype() != "application/json" {
		t.Errorf("datacontenttype = %q; want application/json", ev.GetDatacontenttype())
	}
	// time must be set
	if ev.GetTime() == nil {
		t.Error("CloudEvent time must be set")
	}
}
