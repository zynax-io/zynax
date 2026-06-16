// SPDX-License-Identifier: Apache-2.0

package zynaxobs

import (
	"context"
	"testing"

	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestInitProvidersNoEndpointIsNoop(t *testing.T) {
	t.Setenv(zynaxEndpointEnv, "")

	p, shutdown, err := InitProviders(context.Background(), "svc", "v0.0.0")
	if err != nil {
		t.Fatalf("InitProviders: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown must not be nil")
	}

	if _, ok := p.TracerProvider.(tracenoop.TracerProvider); !ok {
		t.Errorf("TracerProvider = %T, want no-op", p.TracerProvider)
	}
	if _, ok := p.MeterProvider.(metricnoop.MeterProvider); !ok {
		t.Errorf("MeterProvider = %T, want no-op", p.MeterProvider)
	}
	if _, ok := p.LoggerProvider.(lognoop.LoggerProvider); !ok {
		t.Errorf("LoggerProvider = %T, want no-op", p.LoggerProvider)
	}

	if err := shutdown(context.Background()); err != nil {
		t.Errorf("noop shutdown: %v", err)
	}
}

func TestInitProvidersAllProvidersNonNil(t *testing.T) {
	t.Setenv(zynaxEndpointEnv, "")

	p, _, err := InitProviders(context.Background(), "svc", "v1.2.3")
	if err != nil {
		t.Fatalf("InitProviders: %v", err)
	}
	if p.TracerProvider == nil {
		t.Error("TracerProvider is nil")
	}
	if p.MeterProvider == nil {
		t.Error("MeterProvider is nil")
	}
	if p.LoggerProvider == nil {
		t.Error("LoggerProvider is nil")
	}

	// Providers must be usable without panicking even when no-op.
	tr := p.TracerProvider.Tracer("test")
	_, span := tr.Start(context.Background(), "noop-span")
	span.End()

	m := p.MeterProvider.Meter("test")
	if _, err := m.Int64Counter("noop.counter"); err != nil {
		t.Errorf("noop counter: %v", err)
	}

	l := p.LoggerProvider.Logger("test")
	if l == nil {
		t.Error("Logger is nil")
	}
}

func TestNewResourceCarriesServiceAttrs(t *testing.T) {
	res, err := NewResource(context.Background(), "engine-adapter", "v0.6.0")
	if err != nil {
		t.Fatalf("NewResource: %v", err)
	}

	var gotName, gotVersion string
	for _, kv := range res.Attributes() {
		switch string(kv.Key) {
		case "service.name":
			gotName = kv.Value.AsString()
		case "service.version":
			gotVersion = kv.Value.AsString()
		}
	}
	if gotName != "engine-adapter" {
		t.Errorf("service.name = %q, want engine-adapter", gotName)
	}
	if gotVersion != "v0.6.0" {
		t.Errorf("service.version = %q, want v0.6.0", gotVersion)
	}
}
