// Package main is the entry point for the workflow-compiler service.
// Bootstrap only — WorkflowCompilerService RPC handlers are implemented in
// subsequent issues. This file wires server lifecycle only; no business logic.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/zynax-io/zynax/libs/zynaxobs"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/api"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/config"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/infrastructure"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	})))

	serverCreds, err := infrastructure.TLSCreds(cfg.TLSCert, cfg.TLSKey, cfg.TLSCA)
	if err != nil {
		slog.Error("tls credentials failed", "err", err)
		os.Exit(1)
	}

	tracerShutdown, err := zynaxobs.InitTracer(context.Background(), "workflow-compiler")
	if err != nil {
		slog.Error("tracer init failed", "err", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.Creds(serverCreds),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			zynaxobs.MetricsUnaryInterceptor("workflow-compiler"),
			requestIDServerInterceptor,
		),
	)
	reflection.Register(grpcServer)

	zynaxv1.RegisterWorkflowCompilerServiceServer(grpcServer, api.NewWithPolicy(buildPolicyGate(cfg)))

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSvc)
	setHealth(healthSvc, grpc_health_v1.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		slog.Error("listen failed", "port", cfg.GRPCPort, "err", err)
		os.Exit(1)
	}

	metricsSrv := startMetricsServer(cfg.MetricsPort)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		slog.Info("workflow-compiler started", "grpc_port", cfg.GRPCPort, "metrics_port", cfg.MetricsPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc serve error", "err", err)
		}
	}()

	<-ctx.Done()
	drainAndStop(grpcServer, healthSvc)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("metrics server shutdown error", "err", err)
	}
	if err := tracerShutdown(shutdownCtx); err != nil {
		slog.Error("tracer shutdown error", "err", err)
	}
	slog.Info("shutdown complete")
}

// startMetricsServer starts the HTTP server exposing /metrics (Prometheus) and a
// /healthz probe on the metrics port in a background goroutine.
func startMetricsServer(port int) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
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
	return srv
}

// setHealth sets both the overall "" key and the per-service named key to the
// given serving status (canvas O-step 2, #656).
func setHealth(h *health.Server, st grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.SetServingStatus("", st)
	h.SetServingStatus(zynaxv1.WorkflowCompilerService_ServiceDesc.ServiceName, st)
}

// drainAndStop drains health (NOT_SERVING) before GracefulStop() so load
// balancers stop routing during rolling restarts (canvas O-step 2, #656).
func drainAndStop(srv *grpc.Server, h *health.Server) {
	slog.Info("shutting down")
	setHealth(h, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	srv.GracefulStop()
}

// buildPolicyGate constructs a domain.PolicyGate from the environment-backed
// config. Returns nil when policy enforcement is disabled (no namespace set).
func buildPolicyGate(cfg *config.Config) *domain.PolicyGate {
	routing, quotas := cfg.PolicyGates()
	if len(routing) == 0 && len(quotas) == 0 {
		return nil
	}
	slog.Info("policy gate enabled",
		"routing_policies", len(routing),
		"quota_configs", len(quotas),
	)
	return domain.NewPolicyGate(routing, quotas, nil)
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func requestIDServerInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("request-id"); len(vals) > 0 {
			slog.Info("grpc request", "method", info.FullMethod, "request_id", vals[0])
		}
	}
	return handler(ctx, req)
}
