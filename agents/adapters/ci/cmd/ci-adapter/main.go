// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the ci-adapter gRPC service.
// Config path is read from ADAPTER_CONFIG env var; auth token from the env var
// named in AdapterConfig.CI.TokenEnv; registry endpoint from config.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/zynax-io/zynax/agents/adapters/ci/internal/adapter"
	"github.com/zynax-io/zynax/agents/adapters/ci/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/ci/internal/registry"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("ci-adapter error", "err", err)
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
	slog.Info("config loaded", "agent_id", cfg.AgentID, "endpoint", cfg.Endpoint) //nolint:gosec

	// Graceful degradation (issue #1375): a missing auth token must not crash the
	// adapter at boot. Without a token the adapter starts, logs a clear warning,
	// and runs degraded — it does NOT register its capabilities and reports
	// NOT_SERVING so readiness reflects the unavailable state. Any other resolution
	// failure (malformed config) is still fatal.
	token, err := config.ResolveToken(cfg)
	degraded := false
	if err != nil {
		if !errors.Is(err, config.ErrTokenMissing) {
			return fmt.Errorf("resolve token: %w", err)
		}
		degraded = true
		//nolint:gosec // G706: token_env is operator config (the env-var NAME), never the secret value or request input
		slog.Warn("ci-adapter starting in degraded mode: auth token not set; capabilities will NOT be registered and readiness is NOT_SERVING",
			"token_env", cfg.CI.TokenEnv)
	}

	regClient, cleanup, err := dialRegistry(cfg.RegistryEndpoint)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	return serve(ctx, cfg, token, degraded, regClient)
}

func dialRegistry(endpoint string) (zynaxv1.AgentRegistryServiceClient, func(), error) {
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("registry dial %s: %w", endpoint, err)
	}
	return zynaxv1.NewAgentRegistryServiceClient(conn), func() { _ = conn.Close() }, nil
}

func serve(ctx context.Context, cfg *config.AdapterConfig, token string, degraded bool, regClient zynaxv1.AgentRegistryServiceClient) error {
	lis, err := net.Listen("tcp", cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Endpoint, err)
	}

	grpcSrv := grpc.NewServer()
	zynaxv1.RegisterAgentServiceServer(grpcSrv, adapter.NewAgentServer(cfg, token))
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)

	// Degraded mode (issue #1375): no token resolved, so the adapter does not
	// register its capabilities and reports NOT_SERVING. The gRPC + health servers
	// still run so the process stays alive and observable instead of crash-looping.
	if degraded {
		healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		slog.Warn("ci-adapter serving DEGRADED (capabilities not registered)", "endpoint", cfg.Endpoint) //nolint:gosec
		return runServer(ctx, grpcSrv, lis)
	}

	def := registry.BuildAgentDef(cfg)
	if err := registry.RegisterAgent(ctx, regClient, def); err != nil {
		return fmt.Errorf("register: %w", err)
	}
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	slog.Info("ci-adapter serving", "endpoint", cfg.Endpoint) //nolint:gosec

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
	if err := registry.DeregisterAgent(deregCtx, regClient, cfg.AgentID); err != nil { //nolint:contextcheck // signal ctx already cancelled; fresh ctx for cleanup is intentional
		slog.Warn("deregister failed", "err", err)
	}
	grpcSrv.GracefulStop()
	slog.Info("ci-adapter stopped")
	return nil
}

// runServer serves gRPC until the context is cancelled or the server errors. It
// is used by the degraded path (issue #1375), which has nothing registered to
// deregister — it just keeps the process alive and drains on shutdown.
func runServer(ctx context.Context, grpcSrv *grpc.Server, lis net.Listener) error {
	serveErr := make(chan error, 1)
	go func() { serveErr <- grpcSrv.Serve(lis) }()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serveErr:
		return fmt.Errorf("grpc serve: %w", err)
	}
	grpcSrv.GracefulStop()
	slog.Info("ci-adapter stopped")
	return nil
}
