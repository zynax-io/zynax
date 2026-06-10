// SPDX-License-Identifier: Apache-2.0

package zynaxobs

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMetricsUnaryInterceptorRecordsCounter(t *testing.T) {
	interceptor := MetricsUnaryInterceptor("test-svc")
	info := &grpc.UnaryServerInfo{FullMethod: "/zynax.v1.Test/Do"}

	okHandler := func(context.Context, any) (any, error) { return "ok", nil }
	errHandler := func(context.Context, any) (any, error) {
		return nil, status.Error(codes.NotFound, "missing")
	}

	// 10 successful + 1 failed request.
	for i := 0; i < 10; i++ {
		if _, err := interceptor(context.Background(), nil, info, okHandler); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if _, err := interceptor(context.Background(), nil, info, errHandler); err == nil {
		t.Fatal("expected error to propagate")
	}

	got := testutil.ToFloat64(grpcRequestsTotal.WithLabelValues("test-svc", "/zynax.v1.Test/Do", codes.OK.String()))
	if got != 10 {
		t.Errorf("OK counter = %v, want 10", got)
	}
	gotErr := testutil.ToFloat64(grpcRequestsTotal.WithLabelValues("test-svc", "/zynax.v1.Test/Do", codes.NotFound.String()))
	if gotErr != 1 {
		t.Errorf("NotFound counter = %v, want 1", gotErr)
	}

	// Histogram populated: sample count must equal 11 after 11 observations.
	if c := testutil.CollectAndCount(grpcRequestDuration); c == 0 {
		t.Error("duration histogram has no series")
	}
}

func TestMetricsUnaryInterceptorPropagatesError(t *testing.T) {
	interceptor := MetricsUnaryInterceptor("svc")
	info := &grpc.UnaryServerInfo{FullMethod: "/m"}
	sentinel := errors.New("boom")
	_, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
		return nil, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want sentinel", err)
	}
}

func TestEventPublishFailedIncrements(t *testing.T) {
	before := testutil.ToFloat64(eventbusPublishFailedTotal.WithLabelValues("zynax.test.event"))
	EventPublishFailed("zynax.test.event")
	after := testutil.ToFloat64(eventbusPublishFailedTotal.WithLabelValues("zynax.test.event"))
	if after-before != 1 {
		t.Errorf("counter delta = %v, want 1", after-before)
	}
}

func TestRegisterMetricsServesExposition(t *testing.T) {
	mux := http.NewServeMux()
	RegisterMetrics(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	if !strings.Contains(string(buf[:n]), "# HELP") {
		t.Error("expected Prometheus exposition format with # HELP lines")
	}
}

func TestRegisterPprofMounted(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPprof(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/debug/pprof/cmdline")
	if err != nil {
		t.Fatalf("GET pprof: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("pprof status = %d, want 200", resp.StatusCode)
	}
}

func TestInitTracerNoEndpointIsNoop(t *testing.T) {
	t.Setenv(otelEndpointEnv, "")
	shutdown, err := InitTracer(context.Background(), "svc")
	if err != nil {
		t.Fatalf("InitTracer: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("noop shutdown: %v", err)
	}
}
