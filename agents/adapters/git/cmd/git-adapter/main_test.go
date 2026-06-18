// SPDX-License-Identifier: Apache-2.0

// Package main (whitebox test) exercises the run(), dialRegistry(), and serve()
// entry-point functions. These tests follow the same pattern as the http-adapter's
// main_test.go and are required to bring total coverage above the 85% CI gate.
// Closes #717 — part of the git-adapter coverage epic (#713).
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/credential"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// ── run() error paths ─────────────────────────────────────────────────────────

func TestRun_MissingEnvVar(t *testing.T) {
	t.Setenv("ADAPTER_CONFIG", "")
	if err := run(); err == nil {
		t.Fatal("expected error when ADAPTER_CONFIG is empty")
	}
}

func TestRun_MissingFile(t *testing.T) {
	t.Setenv("ADAPTER_CONFIG", "/nonexistent/path/git-adapter.yaml")
	if err := run(); err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestRun_InvalidYAML(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "git-adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("{{invalid yaml content")
	_ = f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	if err := run(); err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestRun_MissingRequiredFields(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "git-adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// Valid YAML but missing agent_id → config.validate returns error.
	_, _ = f.WriteString(
		"endpoint: \":50060\"\nregistry_endpoint: \"localhost:50052\"\n" +
			"git:\n  provider: github\n  auth_env: TEST_TOKEN\n" +
			"capabilities:\n  - name: open_pr\n    owner: o\n    repo: r\n",
	)
	_ = f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	if err := run(); err == nil {
		t.Fatal("expected error for missing agent_id")
	}
}

// TestResolveCredentialSource_MissingTokenIsSentinel proves a missing PAT
// surfaces the typed config.ErrCredentialMissing sentinel so the bootstrap layer
// degrades gracefully instead of crash-looping (issue #1375).
func TestResolveCredentialSource_MissingTokenIsSentinel(t *testing.T) {
	os.Unsetenv("GIT_TOKEN_UNSET_XYZ_717")                                                                      //nolint:errcheck // test cleanup
	cfg := &config.AdapterConfig{Git: config.GitConfig{Provider: "github", AuthEnv: "GIT_TOKEN_UNSET_XYZ_717"}} //nolint:gosec // G101: env-var NAME, not a credential value
	_, _, err := resolveCredentialSource(cfg)
	if err == nil {
		t.Fatal("expected error when auth token env var is not set")
	}
	if !errors.Is(err, config.ErrCredentialMissing) {
		t.Fatalf("expected ErrCredentialMissing sentinel, got: %v", err)
	}
}

// ── serve() error paths via run() ────────────────────────────────────────────

// TestRun_InvalidListenAddr verifies that serve() returns an error when the
// configured endpoint cannot be bound (invalid TCP address).
func TestRun_InvalidListenAddr(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "git-adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// Port -1 is an invalid listen address → net.Listen fails.
	_, _ = fmt.Fprintf(f,
		"agent_id: git-test\nname: Git Test\n"+
			"endpoint: \"localhost:-1\"\nregistry_endpoint: \"127.0.0.1:9090\"\n"+
			"git:\n  provider: github\n  auth_env: GIT_TOKEN_TEST_717\n"+
			"capabilities:\n  - name: open_pr\n    owner: o\n    repo: r\n",
	)
	_ = f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	t.Setenv("GIT_TOKEN_TEST_717", "fake-token")
	if err := run(); err == nil {
		t.Fatal("expected error for invalid listen address")
	}
}

// mockRegistryServer returns AlreadyExists — a non-transient gRPC code.
// This makes RegisterAgent return immediately without retrying, so the test
// completes without any delay.
type mockRegistryServer struct {
	zynaxv1.UnimplementedAgentRegistryServiceServer
}

func (m *mockRegistryServer) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest) (*zynaxv1.RegisterAgentResponse, error) {
	return nil, status.Error(codes.AlreadyExists, "already registered")
}

// TestRun_RegistryNonTransientError exercises serve() up through the
// registry.RegisterAgent call. Uses a real mock gRPC server that immediately
// returns AlreadyExists (non-transient) so run() returns without hanging.
func TestRun_RegistryNonTransientError(t *testing.T) {
	// Start a mock registry server on a random free port.
	mockLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	mockGRPC := grpc.NewServer()
	zynaxv1.RegisterAgentRegistryServiceServer(mockGRPC, &mockRegistryServer{})
	go func() { _ = mockGRPC.Serve(mockLis) }()
	defer mockGRPC.Stop()

	f, err := os.CreateTemp(t.TempDir(), "git-adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// endpoint ":0" lets the OS pick a free port → net.Listen succeeds.
	_, _ = fmt.Fprintf(f,
		"agent_id: git-test\nname: Git Test\n"+
			"endpoint: \"127.0.0.1:0\"\nregistry_endpoint: \"%s\"\n"+
			"git:\n  provider: github\n  auth_env: GIT_TOKEN_TEST_717\n"+
			"capabilities:\n  - name: open_pr\n    owner: o\n    repo: r\n",
		mockLis.Addr().String(),
	)
	_ = f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	t.Setenv("GIT_TOKEN_TEST_717", "fake-token")

	if err := run(); err == nil {
		t.Fatal("expected error when registry returns non-transient AlreadyExists")
	}
}

// ── successRegistryClient — direct mock, no network overhead ─────────────────

// successRegistryClient is a minimal zynaxv1.AgentRegistryServiceClient that
// always succeeds, enabling serve() to reach its steady-state select loop.
type successRegistryClient struct{}

func (c *successRegistryClient) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.RegisterAgentResponse, error) {
	return &zynaxv1.RegisterAgentResponse{}, nil
}

func (c *successRegistryClient) DeregisterAgent(_ context.Context, _ *zynaxv1.DeregisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.DeregisterAgentResponse, error) {
	return &zynaxv1.DeregisterAgentResponse{}, nil
}

func (c *successRegistryClient) GetAgent(_ context.Context, _ *zynaxv1.GetAgentRequest, _ ...grpc.CallOption) (*zynaxv1.AgentDef, error) {
	return nil, nil
}

func (c *successRegistryClient) ListAgents(_ context.Context, _ *zynaxv1.ListAgentsRequest, _ ...grpc.CallOption) (*zynaxv1.ListAgentsResponse, error) {
	return nil, nil
}

func (c *successRegistryClient) FindByCapability(_ context.Context, _ *zynaxv1.FindByCapabilityRequest, _ ...grpc.CallOption) (*zynaxv1.FindByCapabilityResponse, error) {
	return nil, nil
}

// TestServe_GracefulShutdown exercises the serve() success path end-to-end:
// registers with registry → sets health SERVING → serves gRPC → receives ctx
// cancellation → sets NOT_SERVING → deregisters → GracefulStop → returns nil.
func TestServe_GracefulShutdown(t *testing.T) {
	cfg := &config.AdapterConfig{
		AgentID: "git-test",
		Name:    "Git Test",
		// Port :0 lets the OS assign a free port — net.Listen succeeds.
		Endpoint:         "127.0.0.1:0",
		RegistryEndpoint: "127.0.0.1:0", // not dialled; client passed directly
		Git: config.GitConfig{
			Provider: "github",
			AuthEnv:  "IGNORED_IN_SERVE",
		},
		Capabilities: []config.GitCapabilityConfig{
			{Name: "open_pr", Owner: "o", Repo: "r"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := &successRegistryClient{}

	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, cfg, credential.NewStaticSource("fake-token"), "fake-token", false, client)
	}()

	// Give serve() time to start up (register + begin listening).
	time.Sleep(50 * time.Millisecond)
	cancel() // trigger graceful shutdown via ctx.Done

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil from serve on graceful shutdown, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve() did not return within 5s after cancel")
	}
}

// failRegisterClient errors on RegisterAgent. The degraded path must never call
// it, so serve must still return nil with this client wired in (issue #1375).
type failRegisterClient struct{ successRegistryClient }

func (c *failRegisterClient) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.RegisterAgentResponse, error) {
	return nil, status.Error(codes.Internal, "registry must not be called in degraded mode")
}

// TestServe_DegradedNoCredential proves the core fix (issue #1375): with no
// credential the adapter serves, reports NOT_SERVING readiness, does NOT register
// its capabilities, and shuts down cleanly on context cancel — it does not crash.
func TestServe_DegradedNoCredential(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()

	cfg := &config.AdapterConfig{
		AgentID:          "git-test",
		Name:             "Git Test",
		Endpoint:         addr,
		RegistryEndpoint: "127.0.0.1:0",
		Git:              config.GitConfig{Provider: "github", AuthEnv: "IGNORED"},
		Capabilities:     []config.GitCapabilityConfig{{Name: "open_pr", Owner: "o", Repo: "r"}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	// degraded=true, nil source, and a registry client that errors on register:
	// if the degraded path tried to register, serve would surface that error.
	go func() { done <- serve(ctx, cfg, nil, "", true, &failRegisterClient{}) }()

	time.Sleep(100 * time.Millisecond)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial degraded adapter: %v", err)
	}
	defer func() { _ = conn.Close() }()
	resp, err := grpc_health_v1.NewHealthClient(conn).Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health check: %v", err)
	}
	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("expected NOT_SERVING in degraded mode, got: %v", resp.GetStatus())
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("degraded serve must not error (no crash), got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("degraded serve() did not return within 5s after cancel")
	}
}
