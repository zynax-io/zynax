// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the llm-adapter gRPC service. It loads
// config, resolves the API-key secret, builds the provider + AgentService
// server, registers with AgentRegistryService (backoff), serves gRPC with the
// health service, and drains gracefully on SIGTERM (canvas M7.P.5).
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

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/provider"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/registry"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/server"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// configEnvVar names the env var holding the YAML config path (prefix ZYNAX_LLM_).
const configEnvVar = "ZYNAX_LLM_CONFIG"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("llm-adapter error", "err", err)
		os.Exit(1)
	}
}

// run loads config, builds the server + registry client, and runs the gRPC
// serve loop until SIGTERM. Splitting the wiring out of main keeps the process
// lifecycle test-friendly: a non-transient registry error returns before the
// blocking serve loop, exercising the bootstrap paths without a live signal.
func run() error {
	cfg, srv, degraded, err := build()
	if err != nil {
		return err
	}

	regConn, err := grpc.NewClient(cfg.RegistryEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("registry dial %s: %w", cfg.RegistryEndpoint, err)
	}
	defer func() { _ = regConn.Close() }()
	regClient := zynaxv1.NewAgentRegistryServiceClient(regConn)

	return serve(cfg, srv, degraded, regClient)
}

// build loads config, resolves the credential, and constructs the provider and
// AgentService server. The API-key Secret is bound into the provider and never
// logged.
func build() (*config.AdapterConfig, *server.AgentServer, bool, error) {
	cfgPath := os.Getenv(configEnvVar)
	if cfgPath == "" {
		return nil, nil, false, fmt.Errorf("%s env var is required", configEnvVar)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, nil, false, fmt.Errorf("load config: %w", err)
	}
	secret, err := cfg.ResolveSecret()
	if err != nil {
		// Graceful degradation (issue #1375): a missing API key must not crash the
		// adapter at boot. Without it the adapter starts, logs a clear warning, and
		// runs degraded — it does NOT build a provider/server, does NOT register its
		// capabilities, and reports NOT_SERVING so readiness reflects unavailability.
		// Any other resolution failure (malformed config) is still fatal.
		if !errors.Is(err, config.ErrSecretMissing) {
			return nil, nil, false, fmt.Errorf("resolve secret: %w", err)
		}
		//nolint:gosec // G706: api_key_env/provider are operator config (names), never the secret value or request input
		slog.Warn("llm-adapter starting in degraded mode: API key not set; capabilities will NOT be registered and readiness is NOT_SERVING",
			"api_key_env", cfg.Provider.APIKeyEnv, "provider", cfg.Provider.Name)
		return cfg, nil, true, nil
	}
	prov, err := provider.New(cfg.Provider, secret)
	if err != nil {
		return nil, nil, false, fmt.Errorf("build provider: %w", err)
	}
	srv, err := server.NewAgentServer(cfg, prov)
	if err != nil {
		return nil, nil, false, fmt.Errorf("build server: %w", err)
	}
	// Fields are operator-controlled config (not request input); the API-key
	// Secret is never logged. //nolint:gosec — matches sibling adapters.
	slog.Info("llm-adapter config loaded", //nolint:gosec
		"agent_id", cfg.AgentID,
		"provider", cfg.Provider.Name,
		"endpoint", cfg.Endpoint,
		"capabilities", len(cfg.Capabilities),
	)
	return cfg, srv, false, nil
}

// serve binds the listener, registers the agent (backoff), serves gRPC with the
// health service, and drains on SIGTERM: NOT_SERVING → deregister → GracefulStop.
func serve(cfg *config.AdapterConfig, srv *server.AgentServer, degraded bool, regClient zynaxv1.AgentRegistryServiceClient) error {
	lis, err := net.Listen("tcp", cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Endpoint, err)
	}

	grpcSrv := grpc.NewServer()
	if !degraded {
		zynaxv1.RegisterAgentServiceServer(grpcSrv, srv)
	}
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthSrv)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Degraded mode (issue #1375): no secret resolved, so the adapter does not
	// register its capabilities and reports NOT_SERVING. The gRPC + health servers
	// still run so the process stays alive and observable instead of crash-looping.
	if degraded {
		healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		slog.Warn("llm-adapter serving DEGRADED (capabilities not registered)", "endpoint", cfg.Endpoint) //nolint:gosec
		return runDegraded(ctx, grpcSrv, lis)
	}

	def := registry.BuildAgentDef(cfg)
	if err := registry.RegisterAgent(ctx, regClient, def); err != nil {
		return fmt.Errorf("register: %w", err)
	}
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	slog.Info("llm-adapter serving", "endpoint", cfg.Endpoint) //nolint:gosec // value from trusted config file

	serveErr := make(chan error, 1)
	go func() { serveErr <- grpcSrv.Serve(lis) }()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serveErr:
		return fmt.Errorf("grpc serve: %w", err)
	}

	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	if err := registry.DeregisterAgent(context.Background(), regClient, cfg.AgentID); err != nil {
		slog.Warn("deregister failed", "err", err)
	}
	grpcSrv.GracefulStop()
	slog.Info("llm-adapter stopped")
	return nil
}

// runDegraded serves gRPC until the context is cancelled or the server errors.
// The degraded path (issue #1375) registers no AgentService and no registry
// entry, so it just keeps the process alive and drains on shutdown.
func runDegraded(ctx context.Context, grpcSrv *grpc.Server, lis net.Listener) error {
	serveErr := make(chan error, 1)
	go func() { serveErr <- grpcSrv.Serve(lis) }()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serveErr:
		return fmt.Errorf("grpc serve: %w", err)
	}
	grpcSrv.GracefulStop()
	slog.Info("llm-adapter stopped")
	return nil
}
