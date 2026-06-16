// SPDX-License-Identifier: Apache-2.0

package zynaxobs

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

// spanName converts a gRPC full method ("/zynax.v1.WorkflowCompilerService/Compile")
// into the canvas O.3 convention "<service>.<rpc>" (e.g. "WorkflowCompilerService.Compile").
// The proto package prefix is dropped so the span name stays short and the leaf
// service name + RPC are what surface in the trace UI. Unparseable input is
// returned unchanged so naming never panics on an unexpected method string.
func spanName(fullMethod string) string {
	trimmed := strings.TrimPrefix(fullMethod, "/")
	svc, rpc, ok := strings.Cut(trimmed, "/")
	if !ok {
		return fullMethod
	}
	// Drop any "pkg.v1." prefix, keeping only the final service identifier.
	if idx := strings.LastIndex(svc, "."); idx != -1 {
		svc = svc[idx+1:]
	}
	return svc + "." + rpc
}

// TracingStatsHandler returns a gRPC stats handler that creates one server span per
// incoming request and extracts/injects W3C trace context through gRPC metadata,
// stitching spans across services. Pass it via grpc.StatsHandler(...). To get the
// canvas O.3 "<service>.<rpc>" span name, also install TracingServerInterceptors.
// When no OTLP endpoint is configured the global provider is a no-op, so spans are
// not exported.
func TracingStatsHandler() stats.Handler {
	return otelgrpc.NewServerHandler()
}

// TracingClientHandler returns a gRPC stats handler that creates one client span per
// outgoing request and injects W3C trace context into gRPC metadata, so a downstream
// service's server span is stitched to the caller's trace. Pass it via
// grpc.WithStatsHandler(...) on the dial options, alongside TracingClientInterceptors
// for the "<service>.<rpc>" span name. When no OTLP endpoint is configured the global
// provider is a no-op.
func TracingClientHandler() stats.Handler {
	return otelgrpc.NewClientHandler()
}

// tracerName is the instrumentation scope for the spans this package creates.
const tracerName = "github.com/zynax-io/zynax/libs/zynaxobs"

// TracingServerInterceptors returns unary and stream server interceptors that open a
// span named per the canvas O.3 "<service>.<rpc>" convention for each incoming RPC.
// They run inside the otelgrpc stats-handler span (TracingStatsHandler), which has
// already extracted the W3C parent context, so the named span is correctly stitched
// into the cross-service trace. When no OTLP endpoint is configured the global tracer
// is a no-op, so these add negligible overhead. Wire both via
// grpc.ChainUnaryInterceptor / grpc.ChainStreamInterceptor.
func TracingServerInterceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
	tracer := otel.Tracer(tracerName)
	unary := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, span := tracer.Start(ctx, spanName(info.FullMethod), trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		return handler(ctx, req)
	}
	stream := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, span := tracer.Start(ss.Context(), spanName(info.FullMethod), trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		return handler(srv, &tracedServerStream{ServerStream: ss, ctx: ctx})
	}
	return unary, stream
}

// tracedServerStream overrides Context so the handler observes the span-carrying
// context started by the stream interceptor.
type tracedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *tracedServerStream) Context() context.Context { return s.ctx }

// TracingClientInterceptors returns unary and stream client interceptors that open a
// span named per the "<service>.<rpc>" convention (canvas O.3) for each outgoing RPC.
// They run alongside the otelgrpc client stats handler (TracingClientHandler), which
// injects the W3C context into outgoing metadata, so the downstream server span hangs
// off this named client span. Wire both via grpc.WithChainUnaryInterceptor /
// grpc.WithChainStreamInterceptor on the dial options.
func TracingClientInterceptors() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
	tracer := otel.Tracer(tracerName)
	unary := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, span := tracer.Start(ctx, spanName(method), trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
	stream := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx, span := tracer.Start(ctx, spanName(method), trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()
		return streamer(ctx, desc, cc, method, opts...)
	}
	return unary, stream
}

// HTTPMiddleware wraps an HTTP handler so every inbound request produces a server
// span and extracts W3C trace context from request headers, making the api-gateway
// the trace entry point that downstream gRPC client spans hang off (canvas O.3).
// serviceName is used as the otelhttp operation prefix. When no OTLP endpoint is
// configured the global provider is a no-op, so requests are unaffected.
func HTTPMiddleware(serviceName string, next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, serviceName,
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return serviceName + "." + r.Method
		}),
	)
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
