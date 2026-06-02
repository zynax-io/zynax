// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the agent-registry service.
// It wires the in-memory repository, domain service, and gRPC server.
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
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/agent-registry/internal/api"
	"github.com/zynax-io/zynax/services/agent-registry/internal/domain"
	"github.com/zynax-io/zynax/services/agent-registry/internal/infrastructure"
)

type config struct {
	GRPCPort int    `envconfig:"GRPC_PORT" default:"50052"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
	TLSCert  string `envconfig:"TLS_CERT"`
	TLSKey   string `envconfig:"TLS_KEY"`
	TLSCA    string `envconfig:"TLS_CA"`
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
	creds, err := infrastructure.TLSCreds(cfg.TLSCert, cfg.TLSKey, cfg.TLSCA)
	if err != nil {
		return fmt.Errorf("agent-registry: tls credentials: %w", err)
	}

	repo := infrastructure.NewMemoryRepo()
	svc := domain.NewAgentRegistryService(repo)

	srv := grpc.NewServer(grpc.Creds(creds))
	reflection.Register(srv)
	zynaxv1.RegisterAgentRegistryServiceServer(srv, api.NewHandler(svc))

	healthSvc := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSvc)
	healthSvc.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("agent-registry: listen: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		slog.Info("agent-registry started", "grpc_port", cfg.GRPCPort)
		if err := srv.Serve(lis); err != nil {
			slog.Error("grpc serve error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	srv.GracefulStop()
	return nil
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
