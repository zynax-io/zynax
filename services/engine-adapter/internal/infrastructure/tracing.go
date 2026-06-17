// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"fmt"

	"go.opentelemetry.io/otel/propagation"
	otelinterceptor "go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
)

// TemporalTracingInterceptor returns a Temporal SDK interceptor that propagates
// W3C trace context across the workflow/activity boundary. On Submit the active
// span is serialized into the workflow's Temporal header; the worker's inbound
// interceptor extracts it before each workflow and activity runs, so a
// capability dispatch or lifecycle-event publish made from an activity is
// stitched to the originating request's trace (canvas O.5).
//
// The propagator is pinned to W3C TraceContext only (no baggage) so traceparent
// is the single piece of context carried — never auth tokens or session data
// (canvas O.5 safeguard). When no OTLP endpoint is configured the global tracer
// provider is a no-op, so the interceptor adds negligible overhead and the
// service runs unchanged with telemetry disabled.
func TemporalTracingInterceptor() (interceptor.Interceptor, error) {
	tracingInterceptor, err := otelinterceptor.NewTracingInterceptor(otelinterceptor.TracerOptions{
		TextMapPropagator: propagation.TraceContext{},
		DisableBaggage:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("temporal tracing interceptor: %w", err)
	}
	return tracingInterceptor, nil
}
