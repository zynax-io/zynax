// SPDX-License-Identifier: Apache-2.0

package zynaxobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
)

func TestSpanName(t *testing.T) {
	tests := []struct {
		name string
		full string
		want string
	}{
		{"package-qualified", "/zynax.v1.WorkflowCompilerService/CompileWorkflow", "WorkflowCompilerService.CompileWorkflow"},
		{"no-leading-slash", "zynax.v1.AgentRegistryService/RegisterAgent", "AgentRegistryService.RegisterAgent"},
		{"unqualified-service", "/Svc/Method", "Svc.Method"},
		{"unparseable", "garbage", "garbage"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := spanName(tt.full); got != tt.want {
				t.Errorf("spanName(%q) = %q, want %q", tt.full, got, tt.want)
			}
		})
	}
}

func TestTracingHandlersNonNil(t *testing.T) {
	if TracingStatsHandler() == nil {
		t.Error("TracingStatsHandler returned nil")
	}
	if TracingClientHandler() == nil {
		t.Error("TracingClientHandler returned nil")
	}
}

func TestHTTPMiddlewareNamesSpanByService(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(otel.GetTracerProvider()) })

	called := false
	h := HTTPMiddleware("api-gateway", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("wrapped handler was not invoked")
	}
	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if got := spans[0].Name; got != "api-gateway.POST" {
		t.Errorf("HTTP span name = %q, want %q", got, "api-gateway.POST")
	}
}

func TestServerUnaryInterceptorNamesSpan(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(otel.GetTracerProvider()) })

	unary, _ := TracingServerInterceptors()
	info := &grpc.UnaryServerInfo{FullMethod: "/zynax.v1.TaskBrokerService/Submit"}
	_, err := unary(context.Background(), nil, info, func(context.Context, any) (any, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if got := spans[0].Name; got != "TaskBrokerService.Submit" {
		t.Errorf("server span name = %q, want %q", got, "TaskBrokerService.Submit")
	}
}

func TestClientUnaryInterceptorNamesSpan(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(otel.GetTracerProvider()) })

	unary, _ := TracingClientInterceptors()
	err := unary(context.Background(), "/zynax.v1.EngineAdapterService/SubmitWorkflow", nil, nil, nil,
		func(context.Context, string, any, any, *grpc.ClientConn, ...grpc.CallOption) error {
			return nil
		})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if got := spans[0].Name; got != "EngineAdapterService.SubmitWorkflow" {
		t.Errorf("client span name = %q, want %q", got, "EngineAdapterService.SubmitWorkflow")
	}
}
