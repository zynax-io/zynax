// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the git-adapter gRPC service.
// Config path from ADAPTER_CONFIG env var; registry endpoint from config.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/adapter"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/mcp"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/registry"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("git-adapter error", "err", err)
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

	token, err := config.ResolveToken(cfg)
	if err != nil {
		return fmt.Errorf("resolve token: %w", err)
	}

	// `git-adapter mcp` runs the thin MCP stdio shim over the same handlers
	// instead of the runtime gRPC server (ADR-032 — one implementation, two
	// surfaces). It needs no registry and binds no port.
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		return serveMCP(cfg, token, os.Stdin, os.Stdout)
	}

	regClient, cleanup, err := dialRegistry(cfg.RegistryEndpoint)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	return serve(ctx, cfg, token, regClient)
}

// serveMCP runs the MCP stdio shim. The exposed tool set is an explicit
// allow-list built from the configured capability names — not "every handler".
func serveMCP(cfg *config.AdapterConfig, token string, in io.Reader, out io.Writer) error {
	srv := adapter.NewAgentServer(cfg, token)
	tools := make([]string, 0, len(cfg.Capabilities))
	for _, c := range cfg.Capabilities {
		tools = append(tools, c.Name)
	}
	slog.Info("git-adapter mcp serving over stdio", "tools", tools) //nolint:gosec
	if err := mcp.NewServer(srv, tools).Serve(context.Background(), in, out); err != nil {
		return fmt.Errorf("mcp serve: %w", err)
	}
	return nil
}

func dialRegistry(endpoint string) (zynaxv1.AgentRegistryServiceClient, func(), error) {
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("registry dial %s: %w", endpoint, err)
	}
	return zynaxv1.NewAgentRegistryServiceClient(conn), func() { _ = conn.Close() }, nil
}

func serve(ctx context.Context, cfg *config.AdapterConfig, token string, regClient zynaxv1.AgentRegistryServiceClient) error {
	lis, err := net.Listen("tcp", cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Endpoint, err)
	}

	grpcSrv := grpc.NewServer()
	zynaxv1.RegisterAgentServiceServer(grpcSrv, adapter.NewAgentServer(cfg, token))
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)

	def := registry.BuildAgentDef(cfg)
	if err := registry.RegisterAgent(ctx, regClient, def); err != nil {
		return fmt.Errorf("register: %w", err)
	}
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	slog.Info("git-adapter serving", "endpoint", cfg.Endpoint) //nolint:gosec

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
	slog.Info("git-adapter stopped")
	return nil
}
