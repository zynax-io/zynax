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
	"github.com/zynax-io/zynax/services/agent-registry/internal/infrastructure/promql"
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
	// Prometheus HTTP API base URL for selection-time metrics (ADR-039 §3,
	// canvas O-step 4). Empty => selection runs in the degraded
	// readiness-filtered mode (never fails on metrics).
	PromURL string `envconfig:"PROM_URL"`
	// Namespace holding the leader-election Lease for the single-writer
	// status reconciler (ADR-039, canvas O-step 5). Empty in-cluster
	// (auto-detected); required for out-of-cluster/local runs.
	ElectionNamespace string `envconfig:"ELECTION_NAMESPACE"`
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

	var schedHandler *api.SchedulerHandler
	if cfg.CRDInformerEnabled {
		idx, err := startCRDInformer(ctx, cfg)
		if err != nil {
			return fmt.Errorf("agent-registry: crd informer: %w", err)
		}
		schedHandler = api.NewSchedulerHandler(idx, newScorer(cfg))
	}

	srv := newGRPCServer(creds, svc, schedHandler)

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
func newGRPCServer(creds credentials.TransportCredentials, svc *domain.AgentRegistryService, sched *api.SchedulerHandler) *grpc.Server {
	tracingUnary, tracingStream := zynaxobs.TracingServerInterceptors()
	srv := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StatsHandler(zynaxobs.TracingStatsHandler()),
		grpc.ChainUnaryInterceptor(tracingUnary, zynaxobs.MetricsUnaryInterceptor("agent-registry")),
		grpc.ChainStreamInterceptor(tracingStream),
	)
	reflection.Register(srv)
	zynaxv1.RegisterAgentRegistryServiceServer(srv, api.NewHandler(svc))
	if sched != nil {
		zynaxv1.RegisterSchedulerServiceServer(srv, sched)
		slog.Info("agent-registry: SchedulerService registered (SelectAgent live)")
	}
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

// startCRDInformer builds and starts the controller-runtime manager: the
// informer-fed capability index (every replica, O-step 3), the SelectAgent
// data source (O-step 4), and the Lease-elected readiness reconciler that
// derives Agent status from EndpointSlices (single writer, O-step 5).
// The manager runs until ctx is cancelled; a manager error is fatal for the
// informer only, not for the gRPC surface (logged, not propagated), so the
// legacy registry path keeps serving during partial failures.
func startCRDInformer(ctx context.Context, cfg config) (*scheduler.Index, error) {
	// controller-runtime demands a logger before manager construction;
	// bridge it into the service's structured slog output.
	ctrl.SetLogger(logr.FromSlogHandler(slog.Default().Handler()))
	restCfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	idx := scheduler.NewIndex()
	mgr, err := crd.NewManager(restCfg, idx, cfg.ElectionNamespace)
	if err != nil {
		return nil, fmt.Errorf("build crd manager: %w", err)
	}
	if err := crd.SetupReadiness(mgr); err != nil {
		return nil, fmt.Errorf("setup readiness reconciler: %w", err)
	}
	go func() {
		slog.Info("agent-registry: crd informer started (Agent CRs -> capability index; readiness reconciler Lease-elected)")
		if err := mgr.Start(ctx); err != nil {
			slog.Error("crd informer exited", "err", err)
		}
	}()
	return idx, nil
}

// newScorer builds the selection pipeline with its metrics source: the
// Prometheus client (short-TTL cache) when configured, else the always-
// degraded source — selection never fails on metrics (ADR-039 §3).
func newScorer(cfg config) *scheduler.Scorer {
	var src scheduler.MetricsSource = promql.Unavailable{}
	if cfg.PromURL != "" {
		src = promql.New(cfg.PromURL)
		slog.Info("agent-registry: selection metrics from prometheus", "url", cfg.PromURL) //nolint:gosec // operator-controlled config
	} else {
		slog.Info("agent-registry: no prometheus configured — selection runs readiness-filtered (degraded mode)")
	}
	return &scheduler.Scorer{Metrics: src}
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
