// SPDX-License-Identifier: Apache-2.0

package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/zynax-io/zynax/services/event-bus/internal/domain"
)

func TestCloudEvent_Fields(t *testing.T) {
	now := time.Now().UTC()
	e := domain.CloudEvent{
		ID:              "evt-001",
		Source:          "/zynax/wf-42/engine-adapter",
		SpecVersion:     "1.0",
		Type:            "zynax.v1.engine-adapter.workflow.completed",
		DataContentType: "application/json",
		Time:            now,
		Data:            []byte(`{"status":"ok"}`),
		WorkflowID:      "wf-42",
		RunID:           "run-1",
		Namespace:       "default",
		CapabilityName:  "echo",
	}

	if e.ID != "evt-001" {
		t.Errorf("ID: got %q, want %q", e.ID, "evt-001")
	}
	if e.Source != "/zynax/wf-42/engine-adapter" {
		t.Errorf("Source: got %q", e.Source)
	}
	if e.SpecVersion != "1.0" {
		t.Errorf("SpecVersion: got %q, want 1.0", e.SpecVersion)
	}
	if e.Type != "zynax.v1.engine-adapter.workflow.completed" {
		t.Errorf("Type: got %q", e.Type)
	}
	if e.DataContentType != "application/json" {
		t.Errorf("DataContentType: got %q", e.DataContentType)
	}
	if !e.Time.Equal(now) {
		t.Errorf("Time: got %v, want %v", e.Time, now)
	}
	if string(e.Data) != `{"status":"ok"}` {
		t.Errorf("Data: got %q", e.Data)
	}
	if e.WorkflowID != "wf-42" {
		t.Errorf("WorkflowID: got %q", e.WorkflowID)
	}
	if e.RunID != "run-1" {
		t.Errorf("RunID: got %q", e.RunID)
	}
	if e.Namespace != "default" {
		t.Errorf("Namespace: got %q", e.Namespace)
	}
	if e.CapabilityName != "echo" {
		t.Errorf("CapabilityName: got %q", e.CapabilityName)
	}
}

func TestCloudEvent_ZeroValue(t *testing.T) {
	var e domain.CloudEvent
	if e.ID != "" {
		t.Errorf("zero CloudEvent ID should be empty")
	}
	if e.Data != nil {
		t.Errorf("zero CloudEvent Data should be nil")
	}
}

func TestTopic(t *testing.T) {
	tp := domain.Topic("zynax.v1.engine-adapter.workflow")
	if string(tp) != "zynax.v1.engine-adapter.workflow" {
		t.Errorf("Topic string: got %q", tp)
	}

	var empty domain.Topic
	if empty != "" {
		t.Errorf("zero Topic should be empty string")
	}
}

func TestConsumerGroup(t *testing.T) {
	cg := domain.ConsumerGroup("my-service-consumer")
	if string(cg) != "my-service-consumer" {
		t.Errorf("ConsumerGroup string: got %q", cg)
	}
}

func TestSubscribeRequest_Fields(t *testing.T) {
	req := domain.SubscribeRequest{
		SubscriberID: "sub-1",
		TypePattern:  "zynax.v1.**.completed",
		WorkflowID:   "wf-42",
	}

	if req.SubscriberID != "sub-1" {
		t.Errorf("SubscriberID: got %q", req.SubscriberID)
	}
	if req.TypePattern != "zynax.v1.**.completed" {
		t.Errorf("TypePattern: got %q", req.TypePattern)
	}
	if req.WorkflowID != "wf-42" {
		t.Errorf("WorkflowID: got %q", req.WorkflowID)
	}
}

func TestSubscribeRequest_ZeroValue(t *testing.T) {
	var req domain.SubscribeRequest
	if req.SubscriberID != "" || req.TypePattern != "" || req.WorkflowID != "" {
		t.Errorf("zero SubscribeRequest should have all empty fields")
	}
}

func TestIsTerminalEventType(t *testing.T) {
	cases := []struct {
		name      string
		eventType string
		want      bool
	}{
		{"completed full taxonomy", "zynax.v1.engine-adapter.workflow.completed", true},
		{"failed full taxonomy", "zynax.v1.engine-adapter.workflow.failed", true},
		{"terminated", "zynax.v1.engine-adapter.workflow.terminated", true},
		{"canceled", "zynax.v1.engine-adapter.workflow.canceled", true},
		{"timed_out", "zynax.v1.engine-adapter.workflow.timed_out", true},
		{"bare completed verb", "workflow.completed", true},
		{"engine lifecycle completed", "zynax.workflow.completed", true},
		{"engine lifecycle failed", "zynax.workflow.failed", true},
		{"non-terminal state transition", "zynax.v1.engine-adapter.workflow.state.entered", false},
		{"capability event", "zynax.v1.task-broker.task.completed", false},
		{"empty", "", false},
		{"suffix-only false positive guard", "myworkflow.completed", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := domain.IsTerminalEventType(tc.eventType); got != tc.want {
				t.Errorf("IsTerminalEventType(%q) = %v, want %v", tc.eventType, got, tc.want)
			}
		})
	}
}

func TestErrors_Sentinel(t *testing.T) {
	if domain.ErrTopicNotFound == nil {
		t.Error("ErrTopicNotFound must not be nil")
	}
	if domain.ErrSubscriberNotFound == nil {
		t.Error("ErrSubscriberNotFound must not be nil")
	}
	if domain.ErrDeadLetter == nil {
		t.Error("ErrDeadLetter must not be nil")
	}

	if errors.Is(domain.ErrTopicNotFound, domain.ErrSubscriberNotFound) {
		t.Error("sentinel errors must be distinct")
	}
	if errors.Is(domain.ErrSubscriberNotFound, domain.ErrDeadLetter) {
		t.Error("sentinel errors must be distinct")
	}
}

func TestErrors_Wrapping(t *testing.T) {
	wrapped := errors.Join(domain.ErrTopicNotFound, errors.New("stream ZYNAX_V1_ENGINE_ADAPTER"))
	if !errors.Is(wrapped, domain.ErrTopicNotFound) {
		t.Error("wrapped error should match ErrTopicNotFound via errors.Is")
	}
}
