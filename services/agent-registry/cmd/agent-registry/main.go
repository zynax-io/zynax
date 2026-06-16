// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the agent-registry service.
// It wires the repository (memory or Postgres), domain service, and gRPC server.
// All business logic lives in internal/.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/zynax-io/zynax/libs/zynaxobs"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/agent-registry/internal/api"
	"github.com/zynax-io/zynax/services/agent-registry/internal/domain"
	"github.com/zynax-io/zynax/services/agent-registry/internal/infrastructure"
	"github.com/zynax-io/zynax/services/agent-registry/internal/infrastructure/postgres"
)

type config struct {
	GRPCPort    int    `envconfig:"GRPC_PORT" default:"50052"`
	MetricsPort int    `envconfig:"METRICS_PORT" default:"9090"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`
	TLSCert     string `envconfig:"TLS_CERT"`
	TLSKey      string `envconfig:"TLS_KEY"`
	TLSCA       string `envconfig:"TLS_CA"`
	DBEnabled   bool   `envconfig:"DB_ENABLED" default:"false"`
	DBDSN       string `envconfig:"DB_DSN"`
}

func main() {
	var cfg config
	if err := envconfig.Process("ZYNAX_REGISTRY", &cfg); err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	})))
	if err := run(cfg); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

func run(cfg config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	tracerShutdown, err := zynaxobs.InitTracer(ctx, "agent-registry")
	if err != nil {
		return fmt.Errorf("agent-registry: tracer init: %w", err)
	}
	defer func() { _ = tracerShutdown(context.Background()) }()

	metricsSrv := zynaxobs.StartMetricsServer(cfg.MetricsPort)
	defer func() { _ = metricsSrv.Shutdown(context.Background()) }()

	creds, err := infrastructure.TLSCreds(cfg.TLSCert, cfg.TLSKey, cfg.TLSCA)
	if err != nil {
		return fmt.Errorf("agent-registry: tls credentials: %w", err)
	}

	var repo domain.AgentRepository
	if cfg.DBEnabled {
		pgRepo, err := postgres.New(ctx, cfg.DBDSN)
		if err != nil {
			return fmt.Errorf("agent-registry: postgres repository: %w", err)
		}
		defer pgRepo.Close()
		repo = pgRepo
		slog.Info("agent-registry: using postgres repository")
	} else {
		repo = infrastructure.NewMemoryRepo()
		slog.Info("agent-registry: using in-memory repository")
	}
	svc := domain.NewAgentRegistryService(repo)

	srv := newGRPCServer(creds, svc)

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSvc)
	setHealth(healthSvc, grpc_health_v1.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("agent-registry: listen: %w", err)
	}

	go func() {
		slog.Info("agent-registry started", "grpc_port", cfg.GRPCPort)
		if err := srv.Serve(lis); err != nil {
			slog.Error("grpc serve error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	// Drain: report NOT_SERVING so load balancers stop routing before the
	// graceful stop completes (canvas O-step 2, #656).
	setHealth(healthSvc, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	srv.GracefulStop()
	return nil
}

// newGRPCServer builds the agent-registry gRPC server with OTEL tracing (server
// stats handler + "<service>.<rpc>" span interceptors, canvas O.3) and the RED
// metrics interceptor, then registers the service handler and reflection.
func newGRPCServer(creds credentials.TransportCredentials, svc *domain.AgentRegistryService) *grpc.Server {
	tracingUnary, tracingStream := zynaxobs.TracingServerInterceptors()
	srv := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StatsHandler(zynaxobs.TracingStatsHandler()),
		grpc.ChainUnaryInterceptor(tracingUnary, zynaxobs.MetricsUnaryInterceptor("agent-registry")),
		grpc.ChainStreamInterceptor(tracingStream),
	)
	reflection.Register(srv)
	zynaxv1.RegisterAgentRegistryServiceServer(srv, api.NewHandler(svc))
	return srv
}

// setHealth sets both the overall "" key and the per-service named key to the
// given serving status (canvas O-step 2, #656).
func setHealth(h *health.Server, st grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.SetServingStatus("", st)
	h.SetServingStatus(zynaxv1.AgentRegistryService_ServiceDesc.ServiceName, st)
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
