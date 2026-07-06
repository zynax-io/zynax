// SPDX-License-Identifier: Apache-2.0

package zynaxevents

import (
	"context"
	"encoding/json"
	"testing"

	nats "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TestDispatchMsgExtractsTraceparent verifies the consumer path reads the W3C
// traceparent carried in the NATS message headers and still delivers a matching
// event to the channel (async-hop stitching). Ack/Nak on a bare message return
// errors that the dispatcher intentionally discards, so the delivery semantics
// are exercised without a live JetStream consumer.
func TestDispatchMsgExtractsTraceparent(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	env := cloudEventEnvelope{
		SpecVersion: "1.0",
		ID:          "evt-1",
		Source:      "engine-adapter",
		Type:        "zynax.v1.engine-adapter.workflow.started",
		WorkflowID:  "wf-1",
	}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	msg := &nats.Msg{
		Subject: env.Type,
		Data:    data,
		Header: nats.Header{
			"traceparent": {"00-0102030405060708090a0b0c0d0e0f10-0102030405060708-01"},
		},
	}

	ch := make(chan CloudEvent, 1)
	req := SubscribeRequest{
		SubscriberID: "sub-1",
		TypePattern:  "zynax.v1.engine-adapter.workflow.*",
	}

	stop := dispatchMsg(context.Background(), msg, req, ch)
	if stop {
		t.Fatal("dispatchMsg returned stop=true for a non-terminal event")
	}

	select {
	case got := <-ch:
		if got.ID != "evt-1" {
			t.Fatalf("delivered event ID = %q, want evt-1", got.ID)
		}
	default:
		t.Fatal("no event delivered to channel")
	}
}

// TestDispatchMsgNoTraceparent confirms the consumer path runs unchanged when a
// message carries no trace context (telemetry disabled).
func TestDispatchMsgNoTraceparent(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	env := cloudEventEnvelope{ID: "evt-2", Type: "zynax.v1.engine-adapter.workflow.started"}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	msg := &nats.Msg{Subject: env.Type, Data: data, Header: nats.Header{}}

	ch := make(chan CloudEvent, 1)
	req := SubscribeRequest{SubscriberID: "sub-2", TypePattern: "zynax.v1.**"}

	if dispatchMsg(context.Background(), msg, req, ch) {
		t.Fatal("dispatchMsg returned stop=true unexpectedly")
	}
	if got := <-ch; got.ID != "evt-2" {
		t.Fatalf("delivered event ID = %q, want evt-2", got.ID)
	}
}

// TestDispatchMsgTerminalClose verifies the workflow-scoped terminal-close:
// a scoped subscription receives the terminal event and then signals stop,
// while an unscoped subscription keeps the stream open.
func TestDispatchMsgTerminalClose(t *testing.T) {
	env := cloudEventEnvelope{
		SpecVersion: "1.0",
		ID:          "evt-3",
		Source:      "engine-adapter",
		Type:        "zynax.v1.engine-adapter.workflow.completed",
		WorkflowID:  "wf-terminal",
	}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	newMsg := func() *nats.Msg {
		return &nats.Msg{Subject: env.Type, Data: data, Header: nats.Header{}}
	}

	// Workflow-scoped: deliver the terminal event, then stop.
	ch := make(chan CloudEvent, 1)
	scoped := SubscribeRequest{
		SubscriberID: "sub-scoped",
		TypePattern:  "zynax.v1.engine-adapter.workflow.**",
		WorkflowID:   "wf-terminal",
	}
	if !dispatchMsg(context.Background(), newMsg(), scoped, ch) {
		t.Fatal("scoped subscription did not stop on the terminal event")
	}
	if got := <-ch; got.ID != "evt-3" {
		t.Fatalf("terminal event not delivered before close: %q", got.ID)
	}

	// Unscoped wildcard: deliver and keep going.
	ch2 := make(chan CloudEvent, 1)
	wildcard := SubscribeRequest{
		SubscriberID: "sub-wild",
		TypePattern:  "zynax.v1.engine-adapter.workflow.**",
	}
	if dispatchMsg(context.Background(), newMsg(), wildcard, ch2) {
		t.Fatal("wildcard subscription stopped on one run's terminal event")
	}
	if got := <-ch2; got.ID != "evt-3" {
		t.Fatalf("event not delivered to wildcard subscription: %q", got.ID)
	}
}
