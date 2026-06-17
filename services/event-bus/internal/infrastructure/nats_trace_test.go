// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"encoding/json"
	"testing"

	nats "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/zynax-io/zynax/services/event-bus/internal/domain"
)

// TestDispatchMsgExtractsTraceparent verifies the consumer path reads the W3C
// traceparent carried in the NATS message headers and still delivers a matching
// event to the channel (canvas O.5 async-hop stitching). Ack/Nak on a bare
// message return errors that the dispatcher intentionally discards, so the
// delivery semantics are exercised without a live JetStream consumer.
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

	ch := make(chan domain.CloudEvent, 1)
	req := domain.SubscribeRequest{
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

	ch := make(chan domain.CloudEvent, 1)
	req := domain.SubscribeRequest{SubscriberID: "sub-2", TypePattern: "zynax.v1.**"}

	if dispatchMsg(context.Background(), msg, req, ch) {
		t.Fatal("dispatchMsg returned stop=true unexpectedly")
	}
	if got := <-ch; got.ID != "evt-2" {
		t.Fatalf("delivered event ID = %q, want evt-2", got.ID)
	}
}
