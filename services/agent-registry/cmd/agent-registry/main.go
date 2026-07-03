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

	"github.com/go-logr/logr"
	"github.com/kelseyhightower/envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/zynax-io/zynax/libs/zynaxobs"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/agent-registry/internal/api"
	"github.com/zynax-io/zynax/services/agent-registry/internal/domain"
	"github.com/zynax-io/zynax/services/agent-registry/internal/domain/scheduler"
	"github.com/zynax-io/zynax/services/agent-registry/internal/infrastructure"
	"github.com/zynax-io/zynax/services/agent-registry/internal/infrastructure/crd"
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
	// CRD informer (ADR-039, canvas O-step 3): watch Agent CRs and maintain
	// the scheduler capability index. Off by default until the SelectAgent
	// path consumes it (O-step 4); requires the Agent CRD + scheduler RBAC
	// shipped by the chart (O-step 2).
	CRDInformerEnabled bool `envconfig:"CRD_INFORMER_ENABLED" default:"false"`
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

	repo, closeRepo, err := newRepository(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeRepo()
	svc := domain.NewAgentRegistryService(repo)

	if cfg.CRDInformerEnabled {
		if err := startCRDInformer(ctx); err != nil {
			return fmt.Errorf("agent-registry: crd informer: %w", err)
		}
	}

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

// newRepository selects the AgentRepository adapter (ADR-021): Postgres when
// configured, in-memory otherwise. The returned closer is a no-op for memory.
func newRepository(ctx context.Context, cfg config) (domain.AgentRepository, func(), error) {
	if cfg.DBEnabled {
		pgRepo, err := postgres.New(ctx, cfg.DBDSN)
		if err != nil {
			return nil, nil, fmt.Errorf("agent-registry: postgres repository: %w", err)
		}
		slog.Info("agent-registry: using postgres repository")
		return pgRepo, pgRepo.Close, nil
	}
	slog.Info("agent-registry: using in-memory repository")
	return infrastructure.NewMemoryRepo(), func() {}, nil
}

// startCRDInformer builds and starts the controller-runtime manager whose
// informer cache watches Agent CRs and maintains the scheduler capability
// index (ADR-039, canvas O-step 3). The index becomes the SelectAgent data
// source in O-step 4; until then it is a warmed, observable projection.
// The manager runs until ctx is cancelled; a manager error is fatal for the
// informer only, not for the gRPC surface (logged, not propagated), so the
// legacy registry path keeps serving during partial failures.
func startCRDInformer(ctx context.Context) error {
	// controller-runtime demands a logger before manager construction;
	// bridge it into the service's structured slog output.
	ctrl.SetLogger(logr.FromSlogHandler(slog.Default().Handler()))
	restCfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("load kubeconfig: %w", err)
	}
	idx := scheduler.NewIndex()
	mgr, err := crd.NewManager(restCfg, idx)
	if err != nil {
		return fmt.Errorf("build crd manager: %w", err)
	}
	go func() {
		slog.Info("agent-registry: crd informer started (Agent CRs -> capability index)")
		if err := mgr.Start(ctx); err != nil {
			slog.Error("crd informer exited", "err", err)
		}
	}()
	return nil
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
