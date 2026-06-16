// SPDX-License-Identifier: Apache-2.0

// Package zynaxobs provides shared observability primitives reused by every Zynax
// service: Prometheus metrics, a gRPC unary interceptor that records per-request
// counters/histograms, an OTel tracer initialized from OTEL_EXPORTER_OTLP_ENDPOINT,
// and pprof registration. Centralizing these keeps instrumentation identical and
// label cardinality bounded (only service/method/status — never workflow or request
// IDs). See docs/spdd/467-observability-otel-uptrace/canvas.md (issue #491).
package zynaxobs

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// Metric labels are deliberately low-cardinality. service/method/status are bounded
// by the proto surface; workflow IDs, request IDs, and other unbounded values are
// excluded (canvas Safeguards — no high-cardinality labels).
var (
	grpcRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "zynax_grpc_requests_total",
		Help: "Total number of gRPC requests handled, labeled by service, method and status code.",
	}, []string{"service", "method", "status"})

	grpcRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "zynax_grpc_request_duration_seconds",
		Help:    "gRPC handler latency in seconds, labeled by service and method.",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "method"})

	eventbusPublishFailedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "zynax_eventbus_publish_failed_total",
		Help: "Total number of failed event-bus publishes, labeled by event type.",
	}, []string{"event_type"})

	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "zynax_http_requests_total",
		Help: "Total number of HTTP requests handled, labeled by service, method and status code.",
	}, []string{"service", "method", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "zynax_http_request_duration_seconds",
		Help:    "HTTP handler latency in seconds, labeled by service and method.",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "method"})
)

// traceExemplar returns a single-label exemplar carrying the active span's
// trace_id, or nil when no sampled span is in flight (telemetry off by default,
// canvas Norms). trace_id lives only in the exemplar — never as a metric label —
// so series cardinality stays bounded to service/method/status (canvas Safeguards,
// O.4). The returned labels link a metric sample to its trace so a dashboard can
// jump straight to the exemplar's trace in Uptrace.
func traceExemplar(ctx context.Context) prometheus.Labels {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsSampled() {
		return nil
	}
	return prometheus.Labels{"trace_id": sc.TraceID().String()}
}

// addCountWithExemplar increments the counter by one, attaching the trace_id
// exemplar when present. Prometheus stores one exemplar per series, so this never
// grows cardinality.
func addCountWithExemplar(c prometheus.Counter, ex prometheus.Labels) {
	if ex != nil {
		if a, ok := c.(prometheus.ExemplarAdder); ok {
			a.AddWithExemplar(1, ex)
			return
		}
	}
	c.Inc()
}

// observeWithExemplar records a duration sample, attaching the trace_id exemplar
// when present so the histogram bucket links back to a representative trace.
func observeWithExemplar(o prometheus.Observer, v float64, ex prometheus.Labels) {
	if ex != nil {
		if e, ok := o.(prometheus.ExemplarObserver); ok {
			e.ObserveWithExemplar(v, ex)
			return
		}
	}
	o.Observe(v)
}

// RegisterMetrics mounts the Prometheus exposition handler at /metrics on the given
// mux. Every service calls this on its HTTP mux so that
// `curl http://localhost:<port>/metrics` returns the standard text exposition.
// EnableOpenMetrics serializes trace_id exemplars (O.4) when a scraper sends the
// OpenMetrics Accept header; plain Prometheus scrapers still get the text format.
func RegisterMetrics(mux *http.ServeMux) {
	mux.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))
}

// StartMetricsServer starts a dedicated HTTP server exposing /metrics on the given
// port in a background goroutine and returns it so the caller can gracefully shut it
// down. Services that have no other HTTP mux use this to satisfy the per-service
// /metrics requirement without hand-rolling a server each time.
func StartMetricsServer(port int) *http.Server {
	mux := http.NewServeMux()
	RegisterMetrics(mux)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server error", "err", err)
		}
	}()
	slog.Info("metrics server started", "metrics_port", port)
	return srv
}

// RegisterPprof mounts the net/http/pprof endpoints under /debug/pprof on the given
// mux. It is intended for a separate admin port only (engine-adapter) so profiling
// is never reachable on a production API port (canvas Norms).
func RegisterPprof(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

// EventPublishFailed increments the zynax_eventbus_publish_failed_total counter for
// the given event type. It is wired to the M5.D (#483) best-effort publish slog.Warn
// site so a failed publish is observable in Prometheus, not just the logs.
func EventPublishFailed(eventType string) {
	eventbusPublishFailedTotal.WithLabelValues(eventType).Inc()
}

// MetricsUnaryInterceptor returns a gRPC unary server interceptor that increments
// zynax_grpc_requests_total and observes zynax_grpc_request_duration_seconds for
// every incoming call. serviceName labels the metrics so a single Prometheus scrape
// distinguishes services. It composes with other interceptors via ChainUnaryInterceptor.
func MetricsUnaryInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err)
		ex := traceExemplar(ctx)
		addCountWithExemplar(grpcRequestsTotal.WithLabelValues(serviceName, info.FullMethod, code.String()), ex)
		observeWithExemplar(grpcRequestDuration.WithLabelValues(serviceName, info.FullMethod), time.Since(start).Seconds(), ex)
		return resp, err
	}
}

// MetricsHTTPMiddleware returns an HTTP middleware that records the RED metrics
// zynax_http_requests_total and zynax_http_request_duration_seconds for every
// request, attaching the active span's trace_id as an exemplar (O.4). serviceName
// labels the series; method is the URL path template is intentionally NOT used —
// labels stay service/method(HTTP verb)/status to keep cardinality bounded (canvas
// Safeguards). Wire it inside HTTPMiddleware so the otelhttp span is already in the
// request context when the exemplar is read.
func MetricsHTTPMiddleware(serviceName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		ex := traceExemplar(r.Context())
		statusLabel := fmt.Sprintf("%d", sw.status)
		addCountWithExemplar(httpRequestsTotal.WithLabelValues(serviceName, r.Method, statusLabel), ex)
		observeWithExemplar(httpRequestDuration.WithLabelValues(serviceName, r.Method), time.Since(start).Seconds(), ex)
	})
}

// statusRecorder captures the HTTP status code written by downstream handlers so
// the RED metrics can label by status without buffering the response body.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	s.wroteHeader = true
	//nolint:wrapcheck // transparent ResponseWriter passthrough; the error must reach the caller unwrapped
	return s.ResponseWriter.Write(b)
}
