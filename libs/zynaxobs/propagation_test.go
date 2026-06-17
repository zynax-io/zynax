// SPDX-License-Identifier: Apache-2.0

package zynaxobs

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestHeaderCarrierGetSetKeys(t *testing.T) {
	c := headerCarrier{}

	if got := c.Get("absent"); got != "" {
		t.Fatalf("Get on empty carrier = %q, want empty", got)
	}

	c.Set("traceparent", "value-1")
	if got := c.Get("traceparent"); got != "value-1" {
		t.Fatalf("Get after Set = %q, want value-1", got)
	}

	// Set replaces any existing values.
	c.Set("traceparent", "value-2")
	if got := c.Get("traceparent"); got != "value-2" {
		t.Fatalf("Get after second Set = %q, want value-2", got)
	}
	if vals := c["traceparent"]; len(vals) != 1 {
		t.Fatalf("Set must replace, got %d values", len(vals))
	}

	c.Set("tracestate", "ts")
	keys := c.Keys()
	if len(keys) != 2 {
		t.Fatalf("Keys() = %d entries, want 2", len(keys))
	}
}

func TestHeaderCarrierGetFirstValue(t *testing.T) {
	c := headerCarrier{"k": {"first", "second"}}
	if got := c.Get("k"); got != "first" {
		t.Fatalf("Get = %q, want first", got)
	}
}

// TestInjectExtractRoundTrip verifies a span injected into a header map is
// recovered as a valid remote span context on extract — the core async-hop
// stitching guarantee for NATS (canvas O.5).
func TestInjectExtractRoundTrip(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		SpanID:     trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		TraceFlags: trace.FlagsSampled,
		Remote:     false,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	header := map[string][]string{}
	InjectMapHeader(ctx, header)

	if _, ok := header["traceparent"]; !ok {
		t.Fatalf("InjectMapHeader did not write a traceparent header, got keys %v", header)
	}

	got := trace.SpanContextFromContext(ExtractMapHeader(context.Background(), header))
	if got.TraceID() != sc.TraceID() {
		t.Fatalf("extracted TraceID = %s, want %s", got.TraceID(), sc.TraceID())
	}
	if got.SpanID() != sc.SpanID() {
		t.Fatalf("extracted SpanID = %s, want %s", got.SpanID(), sc.SpanID())
	}
	if !got.IsRemote() {
		t.Fatal("extracted span context must be remote")
	}
}

func TestInjectMapHeaderNilSafe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("InjectMapHeader panicked on nil header: %v", r)
		}
	}()
	InjectMapHeader(context.Background(), nil)
}

func TestExtractMapHeaderNilReturnsCtx(t *testing.T) {
	ctx := context.Background()
	if got := ExtractMapHeader(ctx, nil); got != ctx {
		t.Fatal("ExtractMapHeader(nil) must return the original context")
	}
}

func TestExtractMapHeaderNoTraceparent(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	ctx := ExtractMapHeader(context.Background(), map[string][]string{"ce-id": {"x"}})
	if trace.SpanContextFromContext(ctx).IsValid() {
		t.Fatal("extract from header without traceparent must yield no valid span context")
	}
}
