// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the adk-adapter gRPC service. The config
// path comes from ADAPTER_CONFIG; the registry endpoint from the config file.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/zynax-io/zynax/agents/adapters/adk/internal/adapter"
	"github.com/zynax-io/zynax/agents/adapters/adk/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/adk/internal/registry"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("adk-adapter error", "err", err)
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
	slog.Info("config loaded", "agent_id", cfg.AgentID, "endpoint", cfg.Endpoint) //nolint:gosec // G706: config values are operator-controlled, not untrusted input

	regClient, cleanup, err := dialRegistry(cfg.RegistryEndpoint)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	return serve(ctx, cfg, regClient)
}

// dialRegistry opens a lazy gRPC client to AgentRegistryService. The connection
// is established on first use; a bad endpoint surfaces at RegisterAgent time.
func dialRegistry(endpoint string) (zynaxv1.AgentRegistryServiceClient, func(), error) {
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("registry dial %s: %w", endpoint, err)
	}
	return zynaxv1.NewAgentRegistryServiceClient(conn), func() { _ = conn.Close() }, nil
}

// serve binds the gRPC server, registers AgentService + health, registers the
// adapter's capabilities, reports SERVING, and blocks until the context is
// cancelled — then deregisters, reports NOT_SERVING, and drains gracefully.
func serve(ctx context.Context, cfg *config.AdapterConfig, regClient zynaxv1.AgentRegistryServiceClient) error {
	lis, err := net.Listen("tcp", cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Endpoint, err)
	}

	grpcSrv := grpc.NewServer()
	agentSrv, err := adapter.NewAgentServer(cfg)
	if err != nil {
		return fmt.Errorf("build agent server: %w", err)
	}
	zynaxv1.RegisterAgentServiceServer(grpcSrv, agentSrv)
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)

	def := registry.BuildAgentDef(cfg)
	if err := registry.RegisterAgent(ctx, regClient, def); err != nil {
		return fmt.Errorf("register: %w", err)
	}
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	slog.Info("adk-adapter serving", "endpoint", cfg.Endpoint) //nolint:gosec // G706: config values are operator-controlled, not untrusted input

	serveErr := make(chan error, 1)
	go func() { serveErr <- grpcSrv.Serve(lis) }()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serveErr:
		return fmt.Errorf("grpc serve: %w", err)
	}

	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	if err := registry.DeregisterAgent(context.Background(), regClient, cfg.AgentID); err != nil { //nolint:contextcheck // signal ctx already cancelled; fresh ctx for cleanup is intentional
		slog.Warn("deregister failed", "err", err)
	}
	grpcSrv.GracefulStop()
	slog.Info("adk-adapter stopped")
	return nil
}
