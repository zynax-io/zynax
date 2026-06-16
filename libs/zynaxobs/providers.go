// SPDX-License-Identifier: Apache-2.0

package zynaxobs

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otellog "go.opentelemetry.io/otel/log"
	lognoop "go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// zynaxEndpointEnv is the Zynax-prefixed OpenTelemetry OTLP collector endpoint
// (canvas O.2 / Norms: config env prefix `ZYNAX_OTEL_`). Telemetry is off by
// default: when this is unset every provider is a no-op, so a service runs with
// zero exporter overhead and no collector configured.
const zynaxEndpointEnv = "ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT"

// Providers bundles the three OpenTelemetry signal providers a service needs.
// All three are also installed as the OTel globals by InitProviders, so callers
// may use otel.Tracer/otel.Meter and the global logger bridge directly; the
// handles here are returned for callers that prefer explicit wiring.
type Providers struct {
	// TracerProvider produces tracers for span creation.
	TracerProvider trace.TracerProvider
	// MeterProvider produces meters for instrument creation.
	MeterProvider metric.MeterProvider
	// LoggerProvider produces loggers for the OTLP logs pipeline.
	LoggerProvider otellog.LoggerProvider
}

// NewResource builds the shared OpenTelemetry resource carrying the semantic
// convention attributes (service.name, service.version) attached to every span,
// metric and log record. Centralizing this keeps resource attributes identical
// across services (canvas O.2: semconv resource attrs).
func NewResource(ctx context.Context, serviceName, serviceVersion string) (*resource.Resource, error) {
	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
	))
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}
	return res, nil
}

// noopProviders returns the no-op tracer/meter/logger providers used when no
// OTLP endpoint is configured. They satisfy the same interfaces as the real SDK
// providers so callers need no telemetry-on/off branching of their own.
func noopProviders() Providers {
	return Providers{
		TracerProvider: tracenoop.NewTracerProvider(),
		MeterProvider:  metricnoop.NewMeterProvider(),
		LoggerProvider: lognoop.NewLoggerProvider(),
	}
}

// InitProviders configures OTLP/gRPC tracer, meter and logger providers pointed
// at ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT, installs them as the OTel globals along
// with the W3C trace-context propagator, and returns them plus a shutdown that
// flushes all three on exit. When the endpoint is unset it installs no-op
// providers (only the propagator is set) and returns a no-op shutdown, so a
// service runs unchanged without a collector — telemetry is opt-in (canvas
// Norms: off by default, zero overhead when disabled). The returned shutdown
// must be deferred by the caller.
func InitProviders(ctx context.Context, serviceName, serviceVersion string) (Providers, func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	endpoint := os.Getenv(zynaxEndpointEnv)
	if endpoint == "" {
		p := noopProviders()
		otel.SetTracerProvider(p.TracerProvider)
		otel.SetMeterProvider(p.MeterProvider)
		return p, func(context.Context) error { return nil }, nil
	}

	res, err := NewResource(ctx, serviceName, serviceVersion)
	if err != nil {
		return Providers{}, nil, err
	}

	traceExp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpointURL(endpoint))
	if err != nil {
		return Providers{}, nil, fmt.Errorf("otlp trace exporter: %w", err)
	}
	metricExp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpointURL(endpoint))
	if err != nil {
		return Providers{}, nil, fmt.Errorf("otlp metric exporter: %w", err)
	}
	logExp, err := otlploggrpc.New(ctx, otlploggrpc.WithEndpointURL(endpoint))
	if err != nil {
		return Providers{}, nil, fmt.Errorf("otlp log exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
		sdkmetric.WithResource(res),
	)
	lp := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExp)),
		log.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	shutdown := func(ctx context.Context) error {
		tErr := tp.Shutdown(ctx)
		mErr := mp.Shutdown(ctx)
		lErr := lp.Shutdown(ctx)
		switch {
		case tErr != nil:
			return fmt.Errorf("trace provider shutdown: %w", tErr)
		case mErr != nil:
			return fmt.Errorf("meter provider shutdown: %w", mErr)
		case lErr != nil:
			return fmt.Errorf("logger provider shutdown: %w", lErr)
		}
		return nil
	}

	return Providers{TracerProvider: tp, MeterProvider: mp, LoggerProvider: lp}, shutdown, nil
}
