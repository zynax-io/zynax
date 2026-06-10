// SPDX-License-Identifier: Apache-2.0

package zynaxobs

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc/stats"
)

// TracingStatsHandler returns a gRPC stats handler that creates one server span per
// incoming request and extracts/injects W3C trace context through gRPC metadata,
// stitching spans across services. Pass it via grpc.StatsHandler(...). When no OTLP
// endpoint is configured the global provider is a no-op, so spans are not exported.
func TracingStatsHandler() stats.Handler {
	return otelgrpc.NewServerHandler()
}

// otelEndpointEnv is the standard OpenTelemetry environment variable that supplies
// the OTLP collector endpoint. The endpoint is never hardcoded (canvas Norms).
const otelEndpointEnv = "OTEL_EXPORTER_OTLP_ENDPOINT"

// InitTracer configures the global OTel tracer provider with an OTLP gRPC exporter
// pointed at OTEL_EXPORTER_OTLP_ENDPOINT and installs the W3C trace-context
// propagator so spans flow across services through gRPC metadata. When the env var
// is unset it returns a no-op shutdown and leaves the default no-op provider in
// place, so a service runs unchanged without a collector configured. The returned
// shutdown function must be deferred by the caller to flush spans on exit.
func InitTracer(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	endpoint := os.Getenv(otelEndpointEnv)
	if endpoint == "" {
		// No collector configured: tracing is opt-in via the env var.
		otel.SetTextMapPropagator(propagation.TraceContext{})
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpointURL(endpoint))
	if err != nil {
		return nil, fmt.Errorf("otlp trace exporter: %w", err)
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(serviceName),
	))
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp.Shutdown, nil
}
