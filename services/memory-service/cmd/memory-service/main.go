// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the memory-service.
// It wires the gRPC server and registers MemoryService.
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

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/zynax-io/zynax/libs/zynaxconfig"
	"github.com/zynax-io/zynax/libs/zynaxobs"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/memory-service/internal/api"
	"github.com/zynax-io/zynax/services/memory-service/internal/domain"
	"github.com/zynax-io/zynax/services/memory-service/internal/infrastructure"
)

type config struct {
	zynaxconfig.Base
	TLSCert  string `envconfig:"TLS_CERT"`
	TLSKey   string `envconfig:"TLS_KEY"`
	TLSCA    string `envconfig:"TLS_CA"`
	RedisDSN string `envconfig:"REDIS_DSN"`
	DBDSN    string `envconfig:"DB_DSN"`
}

func main() {
	cfg := config{}
	cfg.GRPCPort = 50057 // ZYNAX_MEMORY_GRPC_PORT overrides
	if err := zynaxconfig.Load("MEMORY", &cfg); err != nil {
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

	tracerShutdown, err := zynaxobs.InitTracer(ctx, "memory-service")
	if err != nil {
		return fmt.Errorf("memory-service: tracer init: %w", err)
	}
	defer func() { _ = tracerShutdown(context.Background()) }()

	metricsSrv := zynaxobs.StartMetricsServer(cfg.HealthPort)
	defer func() { _ = metricsSrv.Shutdown(context.Background()) }()

	creds, err := infrastructure.TLSCreds(cfg.TLSCert, cfg.TLSKey, cfg.TLSCA)
	if err != nil {
		return fmt.Errorf("memory-service: tls credentials: %w", err)
	}

	// J.3: wire the Redis KV adapter when ZYNAX_REDIS_DSN is set.
	// J.4 will wire the pgvector adapter; VectorStore remains nil until then.
	var kv domain.KVStore
	if cfg.RedisDSN != "" {
		rdb, err := infrastructure.NewRedisKVFromDSN(cfg.RedisDSN)
		if err != nil {
			return fmt.Errorf("memory-service: redis kv adapter: %w", err)
		}
		kv = rdb
		slog.Info("redis kv adapter wired")
	} else {
		slog.Warn("ZYNAX_MEMORY_REDIS_DSN not set; KV RPCs will return UNIMPLEMENTED")
	}
	handler := api.NewHandler(kv, nil)

	tracingUnary, tracingStream := zynaxobs.TracingServerInterceptors()
	srv := grpc.NewServer(
		grpc.Creds(creds),
		grpc.StatsHandler(zynaxobs.TracingStatsHandler()),
		grpc.ChainUnaryInterceptor(tracingUnary, zynaxobs.MetricsUnaryInterceptor("memory-service")),
		grpc.ChainStreamInterceptor(tracingStream),
	)
	reflection.Register(srv)
	zynaxv1.RegisterMemoryServiceServer(srv, handler)

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSvc)
	setHealth(healthSvc, grpc_health_v1.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("memory-service: listen: %w", err)
	}

	go func() {
		slog.Info("memory-service started", "grpc_port", cfg.GRPCPort)
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

// setHealth sets both the overall "" key and the per-service named key to the
// given serving status (canvas O-step 2, #656).
func setHealth(h *health.Server, st grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.SetServingStatus("", st)
	h.SetServingStatus(zynaxv1.MemoryService_ServiceDesc.ServiceName, st)
}
