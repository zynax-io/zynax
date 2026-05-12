// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the http-adapter gRPC service.
// Config path from ADAPTER_CONFIG env var; registry endpoint from config.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/zynax-io/zynax/agents/adapters/http/internal/adapter"
	"github.com/zynax-io/zynax/agents/adapters/http/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/http/internal/registry"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("http-adapter error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath := os.Getenv("ADAPTER_CONFIG")
	if cfgPath == "" {
		return fmt.Errorf("ADAPTER_CONFIG env var is required")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	slog.Info("config loaded", "agent_id", cfg.AgentID, "endpoint", cfg.Endpoint) //nolint:gosec // values from trusted config file

	regConn, err := grpc.NewClient(cfg.RegistryEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("registry dial %s: %w", cfg.RegistryEndpoint, err)
	}
	defer func() { _ = regConn.Close() }()
	regClient := zynaxv1.NewAgentRegistryServiceClient(regConn)

	lis, err := net.Listen("tcp", cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Endpoint, err)
	}

	grpcSrv := grpc.NewServer()
	zynaxv1.RegisterAgentServiceServer(grpcSrv, adapter.NewAgentServer(cfg))
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	def := registry.BuildAgentDef(cfg)
	if err := registry.RegisterAgent(ctx, regClient, def); err != nil {
		return fmt.Errorf("register: %w", err)
	}
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	slog.Info("http-adapter serving", "endpoint", cfg.Endpoint) //nolint:gosec // value from trusted config file

	serveErr := make(chan error, 1)
	go func() { serveErr <- grpcSrv.Serve(lis) }()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serveErr:
		return fmt.Errorf("grpc serve: %w", err)
	}

	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	deregCtx := context.Background()
	if err := registry.DeregisterAgent(deregCtx, regClient, cfg.AgentID); err != nil {
		slog.Warn("deregister failed", "err", err)
	}
	grpcSrv.GracefulStop()
	slog.Info("http-adapter stopped")
	return nil
}
