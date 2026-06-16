// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the task-broker service.
// It wires the repository (memory or Postgres), agent registry finder, agent executor,
// and gRPC server. All business logic lives in internal/.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/zynax-io/zynax/libs/zynaxconfig"
	"github.com/zynax-io/zynax/libs/zynaxobs"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/task-broker/internal/api"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
	"github.com/zynax-io/zynax/services/task-broker/internal/infrastructure"
	"github.com/zynax-io/zynax/services/task-broker/internal/infrastructure/postgres"
)

type config struct {
	zynaxconfig.Base
	RegistryAddr     string `envconfig:"REGISTRY_ADDR" default:"localhost:50052"`
	EventBusAddr     string `envconfig:"EVENTBUS_ADDR"`
	GRPCCallTimeoutS int    `envconfig:"GRPC_CALL_TIMEOUT_S" default:"30"`
	TLSCert          string `envconfig:"TLS_CERT"`
	TLSKey           string `envconfig:"TLS_KEY"`
	TLSCA            string `envconfig:"TLS_CA"`
	DBEnabled        bool   `envconfig:"DB_ENABLED" default:"false"`
	DBDSN            string `envconfig:"DB_DSN"`
}

func main() {
	cfg := config{}
	cfg.GRPCPort = 50053 // service default; ZYNAX_BROKER_GRPC_PORT overrides
	if err := zynaxconfig.Load("BROKER", &cfg); err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}
	zynaxconfig.SetDefaultLogger(cfg.LogLevel)
	if err := run(cfg); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

func run(cfg config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	tracerShutdown, err := zynaxobs.InitTracer(ctx, "task-broker")
	if err != nil {
		return fmt.Errorf("task-broker: tracer init: %w", err)
	}
	defer func() { _ = tracerShutdown(context.Background()) }()

	metricsSrv := zynaxobs.StartMetricsServer(cfg.HealthPort)
	defer func() { _ = metricsSrv.Shutdown(context.Background()) }()

	creds, err := infrastructure.TLSCreds(cfg.TLSCert, cfg.TLSKey, cfg.TLSCA)
	if err != nil {
		return fmt.Errorf("task-broker: tls credentials: %w", err)
	}

	callTimeout := time.Duration(cfg.GRPCCallTimeoutS) * time.Second
	finder, finderCleanup, err := infrastructure.NewRegistryClient(cfg.RegistryAddr, callTimeout, creds)
	if err != nil {
		return fmt.Errorf("task-broker: registry client: %w", err)
	}
	defer finderCleanup()

	repo, repoCleanup, err := newRepository(ctx, cfg)
	if err != nil {
		return err
	}
	defer repoCleanup()
	executor := infrastructure.NewAgentExecutor(creds)
	svc := domain.NewTaskService(repo, finder, executor)

	publisherCleanup, err := attachEventPublisher(svc, cfg, callTimeout, creds)
	if err != nil {
		return err
	}
	defer publisherCleanup()

	srv, healthSvc := newGRPCServer(creds, svc)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("task-broker: listen: %w", err)
	}

	go func() {
		slog.Info("task-broker started", "grpc_port", cfg.GRPCPort)
		if err := srv.Serve(lis); err != nil {
			slog.Error("grpc serve error", "err", err)
		}
	}()

	go recoverInFlight(ctx, svc)

	<-ctx.Done()
	gracefulShutdown(srv, healthSvc)
	return nil
}

// newRepository selects the Postgres or in-memory TaskRepository per config.
// The returned cleanup function must be deferred by the caller.
func newRepository(ctx context.Context, cfg config) (domain.TaskRepository, func(), error) {
	if !cfg.DBEnabled {
		slog.Info("task-broker: using in-memory repository")
		return infrastructure.NewMemoryRepo(), func() {}, nil
	}
	pgRepo, err := postgres.New(ctx, cfg.DBDSN)
	if err != nil {
		return nil, nil, fmt.Errorf("task-broker: postgres repository: %w", err)
	}
	slog.Info("task-broker: using postgres repository")
	return pgRepo, func() { pgRepo.Close() }, nil
}

// attachEventPublisher wires the optional EventBus lifecycle publisher so
// capability fan-outs are observable over the bus (EPIC #881 O5). An empty
// address disables publication. The returned cleanup must be deferred.
func attachEventPublisher(svc *domain.TaskService, cfg config, callTimeout time.Duration, creds credentials.TransportCredentials) (func(), error) {
	if cfg.EventBusAddr == "" {
		return func() {}, nil
	}
	publisher, cleanup, err := infrastructure.NewEventPublisher(cfg.EventBusAddr, callTimeout, creds)
	if err != nil {
		return nil, fmt.Errorf("task-broker: event publisher: %w", err)
	}
	svc.WithEventPublisher(publisher)
	slog.Info("task-broker: publishing task events", "eventbus_addr", cfg.EventBusAddr)
	return cleanup, nil
}

// recoverInFlight re-launches every non-terminal task left by a previous run
// (EPIC #881 O5: a broker restart never loses an in-flight fan-out).
func recoverInFlight(ctx context.Context, svc *domain.TaskService) {
	n, err := svc.RecoverInFlight(ctx)
	if err != nil {
		slog.Warn("task recovery incomplete", "recovered", n, "err", err)
		return
	}
	if n > 0 {
		slog.Info("recovered in-flight tasks", "count", n)
	}
}

// gracefulShutdown drains health (NOT_SERVING) before GracefulStop() so load
// balancers stop routing during rolling restarts (canvas O-step 2, #656).
func gracefulShutdown(srv *grpc.Server, healthSvc *health.Server) {
	slog.Info("shutting down")
	setHealth(healthSvc, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	srv.GracefulStop()
}

// newGRPCServer builds the gRPC server with metrics + tracing interceptors,
// reflection, the TaskBroker handler, and the gRPC health service registered.
// It returns the health server so the caller can mark NOT_SERVING on shutdown.
func newGRPCServer(creds credentials.TransportCredentials, svc *domain.TaskService) (*grpc.Server, *health.Server) {
	tracingUnary, tracingStream := zynaxobs.TracingServerInterceptors()
	srv := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StatsHandler(zynaxobs.TracingStatsHandler()),
		grpc.ChainUnaryInterceptor(tracingUnary, zynaxobs.MetricsUnaryInterceptor("task-broker")),
		grpc.ChainStreamInterceptor(tracingStream),
	)
	reflection.Register(srv)
	zynaxv1.RegisterTaskBrokerServiceServer(srv, api.NewHandler(svc))

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSvc)
	setHealth(healthSvc, grpc_health_v1.HealthCheckResponse_SERVING)
	return srv, healthSvc
}

// setHealth sets both the overall "" key and the per-service named key to the
// given serving status (canvas O-step 2, #656).
func setHealth(h *health.Server, st grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.SetServingStatus("", st)
	h.SetServingStatus(zynaxv1.TaskBrokerService_ServiceDesc.ServiceName, st)
}
