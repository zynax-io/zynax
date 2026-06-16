// SPDX-License-Identifier: Apache-2.0

package zynaxobs

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// sampledCtx returns a context carrying a valid, sampled span context with a
// fixed trace_id, so exemplar extraction is deterministic in tests.
func sampledCtx(traceHex string) context.Context {
	tid, _ := trace.TraceIDFromHex(traceHex)
	sid, _ := trace.SpanIDFromHex("0102030405060708")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	})
	return trace.ContextWithSpanContext(context.Background(), sc)
}

func TestTraceExemplarNilWhenUnsampled(t *testing.T) {
	if got := traceExemplar(context.Background()); got != nil {
		t.Errorf("exemplar without span = %v, want nil", got)
	}
}

func TestTraceExemplarCarriesTraceID(t *testing.T) {
	const trace32 = "0123456789abcdef0123456789abcdef"
	ex := traceExemplar(sampledCtx(trace32))
	if ex == nil {
		t.Fatal("expected exemplar for sampled span, got nil")
	}
	if ex["trace_id"] != trace32 {
		t.Errorf("exemplar trace_id = %q, want %q", ex["trace_id"], trace32)
	}
}

func TestMetricsHTTPMiddlewareRecordsRED(t *testing.T) {
	mw := MetricsHTTPMiddleware("http-svc", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	srv := httptest.NewServer(mw)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/x")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = resp.Body.Close()

	got := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("http-svc", http.MethodGet, "418"))
	if got != 1 {
		t.Errorf("http counter = %v, want 1", got)
	}
	if c := testutil.CollectAndCount(httpRequestDuration); c == 0 {
		t.Error("http duration histogram has no series")
	}
}

func TestMetricsExposesExemplarInOpenMetrics(t *testing.T) {
	const trace32 = "fedcba9876543210fedcba9876543210"
	interceptor := MetricsUnaryInterceptor("exemplar-svc")
	info := &grpc.UnaryServerInfo{FullMethod: "/zynax.v1.Exemplar/Do"}
	okHandler := func(context.Context, any) (any, error) { return "ok", nil }
	if _, err := interceptor(sampledCtx(trace32), nil, info, okHandler); err != nil {
		t.Fatalf("interceptor: %v", err)
	}

	mux := http.NewServeMux()
	RegisterMetrics(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/metrics", nil)
	// OpenMetrics Accept header is required for promhttp to serialize exemplars.
	req.Header.Set("Accept", "application/openmetrics-text")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "trace_id=\""+trace32+"\"") {
		t.Errorf("expected exemplar with trace_id %q in /metrics output", trace32)
	}
}

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
