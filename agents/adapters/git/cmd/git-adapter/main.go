// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the git-adapter gRPC service.
// Config path from ADAPTER_CONFIG env var; registry endpoint from config.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/adapter"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/auth"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/mcp"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/redact"
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

	// Least-privilege gate (G.5 / #1260): verify the token cannot reach repos
	// beyond the configured owner/repo before it is ever used. The token value is
	// never passed to the logger — only scope/class metadata.
	probe, err := newScopeProbe(token)
	if err != nil {
		return fmt.Errorf("scope probe init: %w", err)
	}
	if err := validateTokenScope(context.Background(), probe, auth.ParseMode(os.Getenv("GIT_ADAPTER_SCOPE_MODE"))); err != nil {
		return err
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

// scopeValidator is the probe surface auth.Validate consumes at startup (enables
// testing main's wiring with a fake probe without reaching the GitHub API).
type scopeValidator interface {
	Probe(ctx context.Context) (http.Header, error)
}

// newScopeProbe builds the startup scope probe. It is a package var so tests can
// substitute a fake probe that never reaches the network; production resolves to
// the public GitHub API.
var newScopeProbe = func(token string) (scopeValidator, error) {
	return auth.NewGitHubProbe(token, "")
}

// validateTokenScope runs the startup least-privilege check and applies the
// configured mode. In enforce mode an over-broad token aborts startup; in warn
// mode it emits a loud structured warning and continues. The token value is
// never logged — only scope/class metadata from the Result.
func validateTokenScope(ctx context.Context, p scopeValidator, mode auth.Mode) error {
	res, err := auth.Validate(ctx, p, mode)
	if err != nil {
		if errors.Is(err, auth.ErrOverBroadScope) {
			// Metadata only — no token, no secret material in the message.
			return fmt.Errorf("token scope validation failed: %w", err)
		}
		// Probe transport error: surface but do not block startup on a probe that
		// could not run (e.g. offline registry-only bring-up); warn and proceed.
		//nolint:gosec // G706: scope/error metadata only, never the token value
		slog.Warn("token scope probe could not run; skipping least-privilege check",
			"err", err, "mode", mode.String())
		return nil
	}
	if len(res.OverBroad) > 0 {
		//nolint:gosec // G706: scope metadata only, never the token value
		slog.Warn("git token grants scope beyond configured owner/repo (least-privilege)",
			"token_class", res.TokenClass, "over_broad_scopes", res.OverBroad, "mode", mode.String())
		return nil
	}
	//nolint:gosec // G706: scope/class metadata only, never the token value
	slog.Info("git token scope validated", "token_class", res.TokenClass, "mode", mode.String())
	return nil
}

// serveMCP runs the MCP stdio shim. The exposed tool set is an explicit
// allow-list built from the configured capability names — not "every handler".
func serveMCP(cfg *config.AdapterConfig, token string, in io.Reader, out io.Writer) error {
	srv := adapter.NewAgentServer(cfg, token)
	tools := make([]string, 0, len(cfg.Capabilities))
	for _, c := range cfg.Capabilities {
		tools = append(tools, c.Name)
	}
	// The injected token is scrubbed from every tool result at the prompt
	// boundary (G.3 / #1199); the token value itself is never logged here.
	red := redact.New(token)
	slog.Info("git-adapter mcp serving over stdio", "tools", tools) //nolint:gosec
	if err := mcp.NewServerWithRedactor(srv, tools, red).Serve(context.Background(), in, out); err != nil {
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
