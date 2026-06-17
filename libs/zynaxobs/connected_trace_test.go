// SPDX-License-Identifier: Apache-2.0

package zynaxobs

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

// TestConnectedEndToEndTrace is the canvas R.5 observability-validation test.
//
// It asserts that a single logical run produces a *connected* distributed
// trace: the same trace id propagates across every transport the platform uses
// to stitch the api-gateway -> compiler -> engine -> broker -> registry -> agent
// chain (HTTP entry, gRPC metadata between services, and the NATS async hop).
// Without this guard, a regression in the zynaxobs propagation wiring (e.g. a
// dropped propagator registration or a hop that starts a fresh root span) would
// silently fragment the trace and pass every existing per-hop unit test, which
// only check span names in isolation.
//
// The test exercises the real stitching code paths — HTTPMiddleware, the
// gRPC stats handlers + interceptors over an in-memory bufconn connection, and
// InjectMapHeader/ExtractMapHeader for the NATS hop — against an in-memory span
// exporter, so it stays on the normal `go test` path with no live stack or new
// CI gate.
func TestConnectedEndToEndTrace(t *testing.T) {
	// Single in-memory exporter shared by every hop: all spans land here so we
	// can assert they belong to one trace, exactly as a real collector would
	// receive them from all six services.
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(otel.GetTracerProvider())
	})

	healthClient := dialTracedGRPC(t)

	// --- HTTP entry hop (api-gateway): the middleware starts the root server
	// span and is the trace entry point. Inside the handler we make the
	// downstream gRPC call (so its client span hangs off the HTTP span) and then
	// perform the NATS async hop.
	var natsConsumerTraceID trace.TraceID
	handler := HTTPMiddleware("api-gateway", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// gRPC hop — downstream server span is stitched to this request's trace.
		if _, derr := healthClient.Check(ctx, &healthpb.HealthCheckRequest{}); derr != nil {
			t.Errorf("health check: %v", derr)
		}

		// NATS async hop — the engine publishes an event carrying the W3C
		// traceparent in message headers; the consumer extracts it and starts a
		// span stitched to the same trace.
		header := map[string][]string{}
		InjectMapHeader(ctx, header)

		consumerCtx := ExtractMapHeader(context.Background(), header)
		_, consumerSpan := otel.Tracer(tracerName).Start(
			consumerCtx, "agent.HandleTask", trace.WithSpanKind(trace.SpanKindConsumer))
		natsConsumerTraceID = consumerSpan.SpanContext().TraceID()
		consumerSpan.End()

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("handler status = %d, want 200", rec.Code)
	}

	// Flush any batched spans (syncer is synchronous, but be explicit).
	ctxFlush, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if ferr := tp.ForceFlush(ctxFlush); ferr != nil {
		t.Fatalf("force flush: %v", ferr)
	}

	root := assertSingleTrace(t, exp.GetSpans())

	// The NATS consumer span — reached only by extracting the injected
	// traceparent — must be on the same trace, proving the async hop is stitched.
	if natsConsumerTraceID != root {
		t.Errorf("nats consumer trace id %s != root %s: async hop not stitched",
			natsConsumerTraceID, root)
	}
}

// dialTracedGRPC stands up a downstream gRPC service wired exactly as the real
// services are (stats handler extracts the W3C parent, interceptors name the
// span) over an in-memory bufconn connection, and returns a client whose dial
// options inject trace context into outgoing metadata. The health service
// stands in for any zynax RPC; only the trace plumbing matters.
func dialTracedGRPC(t *testing.T) healthpb.HealthClient {
	t.Helper()

	lis := bufconn.Listen(1 << 20)
	srvUnary, srvStream := TracingServerInterceptors()
	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(TracingStatsHandler()),
		grpc.ChainUnaryInterceptor(srvUnary),
		grpc.ChainStreamInterceptor(srvStream),
	)
	healthpb.RegisterHealthServer(grpcSrv, health.NewServer())
	go func() { _ = grpcSrv.Serve(lis) }()
	t.Cleanup(grpcSrv.Stop)

	cliUnary, cliStream := TracingClientInterceptors()
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(TracingClientHandler()),
		grpc.WithChainUnaryInterceptor(cliUnary),
		grpc.WithChainStreamInterceptor(cliStream),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return healthpb.NewHealthClient(conn)
}

// assertSingleTrace asserts the recorded spans form one connected trace and
// returns the shared trace id. "Connected" means two things: every span shares
// the same trace id (no hop dropped the parent or started a fresh root), and
// the spans form an unbroken parent chain rooted at exactly one span (they are
// genuinely linked, not just a set that happens to share an id).
func assertSingleTrace(t *testing.T, spans tracetest.SpanStubs) trace.TraceID {
	t.Helper()

	// At least one span per transport hop: HTTP server, gRPC client, gRPC
	// server, and the NATS consumer. otelgrpc may also emit message-event
	// spans, so assert "at least", not an exact count.
	if len(spans) < 4 {
		t.Fatalf("got %d spans, want >= 4 (http + grpc client + grpc server + nats consumer)", len(spans))
	}

	root := spans[0].SpanContext.TraceID()
	if !root.IsValid() {
		t.Fatal("root trace id is invalid")
	}
	traceIDs := map[trace.TraceID]int{}
	byID := map[trace.SpanID]bool{}
	for _, s := range spans {
		traceIDs[s.SpanContext.TraceID()]++
		byID[s.SpanContext.SpanID()] = true
		if got := s.SpanContext.TraceID(); got != root {
			t.Errorf("span %q has trace id %s, want single connected trace %s", s.Name, got, root)
		}
	}
	if len(traceIDs) != 1 {
		t.Fatalf("trace fragmented: %d distinct trace ids across %d spans, want 1: %v",
			len(traceIDs), len(spans), traceIDs)
	}

	roots := 0
	for _, s := range spans {
		if !s.Parent.IsValid() {
			roots++
			continue
		}
		if !byID[s.Parent.SpanID()] {
			t.Errorf("span %q references parent %s not present in the trace — broken chain",
				s.Name, s.Parent.SpanID())
		}
	}
	if roots != 1 {
		t.Errorf("connected trace must have exactly one root span, found %d", roots)
	}
	return root
}

// TestConnectedTraceLogCorrelation asserts the trace id that correlates log
// lines across services is derivable from the active span context — the
// canvas R.5 "+ log correlation" requirement. Services emit the active span's
// trace id alongside their structured logs, so a single run's log lines can be
// joined to its trace in the observability UI.
func TestConnectedTraceLogCorrelation(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// A request entering at the gateway carries a traceparent; a downstream
	// service extracts it and logs against the same trace id.
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19},
		SpanID:     trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		TraceFlags: trace.FlagsSampled,
	})
	gatewayCtx := trace.ContextWithSpanContext(context.Background(), sc)

	header := map[string][]string{}
	InjectMapHeader(gatewayCtx, header)

	downstreamCtx := ExtractMapHeader(context.Background(), header)
	downstreamTraceID := trace.SpanContextFromContext(downstreamCtx).TraceID()

	// The downstream service derives the same trace id it would attach to its
	// log lines for correlation.
	if downstreamTraceID != sc.TraceID() {
		t.Fatalf("downstream log-correlation trace id = %s, want %s (same trace as gateway)",
			downstreamTraceID, sc.TraceID())
	}
	if downstreamTraceID.String() == (trace.TraceID{}).String() {
		t.Fatal("log-correlation trace id must be non-zero so log lines can be joined to the trace")
	}
}
