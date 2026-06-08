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

	srv := grpc.NewServer(grpc.Creds(creds))
	reflection.Register(srv)
	zynaxv1.RegisterMemoryServiceServer(srv, handler)

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSvc)
	healthSvc.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

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
	srv.GracefulStop()
	return nil
}
