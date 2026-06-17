// SPDX-License-Identifier: Apache-2.0

package zynaxobs

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// headerCarrier adapts a map[string][]string (e.g. nats.Header, which is
// defined as map[string][]string) to the OpenTelemetry TextMapCarrier
// interface so the W3C trace-context propagator can read and write the
// traceparent across non-gRPC/HTTP transports such as NATS message headers.
//
// Only the first value of each key is read, matching the single-value W3C
// traceparent/tracestate semantics; Set replaces any existing values.
type headerCarrier map[string][]string

// Get returns the first value associated with key, or "" when absent.
func (c headerCarrier) Get(key string) string {
	if v := c[key]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// Set stores key with a single value, replacing any existing values.
func (c headerCarrier) Set(key, value string) {
	c[key] = []string{value}
}

// Keys lists the carrier's keys.
func (c headerCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// compile-time assertion: headerCarrier satisfies the propagator carrier API.
var _ propagation.TextMapCarrier = headerCarrier(nil)

// InjectMapHeader writes the W3C trace context (traceparent) from ctx into the
// given header map using the globally configured TextMapPropagator. The header
// must be non-nil. NATS headers (map[string][]string) can be passed directly.
// When ctx carries no active span, or no propagator/endpoint is configured,
// this is a no-op so a publisher runs unchanged without telemetry enabled.
// Only trace context is propagated — never auth tokens or session data
// (canvas O.5 safeguard).
func InjectMapHeader(ctx context.Context, header map[string][]string) {
	if header == nil {
		return
	}
	otel.GetTextMapPropagator().Inject(ctx, headerCarrier(header))
}

// ExtractMapHeader returns a context derived from ctx with the W3C trace
// context (traceparent) read from the given header map, using the globally
// configured TextMapPropagator. A consumer's span started from the returned
// context is stitched to the publisher's trace across the async NATS hop.
// When the header carries no traceparent the original context is returned
// unchanged, so consumers run normally without telemetry enabled.
func ExtractMapHeader(ctx context.Context, header map[string][]string) context.Context {
	if header == nil {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, headerCarrier(header))
}
