// SPDX-License-Identifier: Apache-2.0

// Package main (whitebox test) exercises the run(), dialRegistry(), and serve()
// entry-point functions. These tests follow the same pattern as the git-adapter's
// main_test.go and bring total coverage above the 80% CI gate.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/ci/internal/config"
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
	t.Setenv("ADAPTER_CONFIG", "/nonexistent/path/ci-adapter.yaml")
	if err := run(); err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestRun_InvalidYAML(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "ci-adapter-*.yaml")
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
	f, err := os.CreateTemp(t.TempDir(), "ci-adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// Valid YAML but missing agent_id → validate returns error.
	_, _ = f.WriteString(
		"endpoint: \":50055\"\nregistry_endpoint: \"localhost:50052\"\n" +
			"ci:\n  provider: github-actions\n  token_env: GH_TOKEN\n" +
			"capabilities:\n  - name: trigger_workflow\n    owner: o\n    repo: r\n    workflow_id: ci.yml\n",
	)
	_ = f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	if err := run(); err == nil {
		t.Fatal("expected error for missing agent_id")
	}
}

// TestResolveToken_MissingIsSentinel proves a missing token surfaces the typed
// ErrTokenMissing sentinel so the bootstrap layer can degrade gracefully rather
// than crash-loop (issue #1375).
func TestResolveToken_MissingIsSentinel(t *testing.T) {
	cfg := &config.AdapterConfig{CI: config.CIConfig{TokenEnv: "CI_TOKEN_UNSET_XYZ_407"}} //nolint:gosec // G101: env-var NAME, not a credential value
	os.Unsetenv("CI_TOKEN_UNSET_XYZ_407")                                                 //nolint:errcheck // test cleanup
	_, err := config.ResolveToken(cfg)
	if err == nil {
		t.Fatal("expected error when token env var is not set")
	}
	if !errors.Is(err, config.ErrTokenMissing) {
		t.Fatalf("expected ErrTokenMissing sentinel, got: %v", err)
	}
}

func TestRun_InvalidListenAddr(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "ci-adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// Port -1 is invalid → net.Listen fails.
	_, _ = fmt.Fprintf(f,
		"agent_id: ci-test\nname: CI Test\n"+
			"endpoint: \"localhost:-1\"\nregistry_endpoint: \"127.0.0.1:9090\"\n"+
			"ci:\n  provider: github-actions\n  token_env: CI_TOKEN_TEST_407\n"+
			"capabilities:\n  - name: trigger_workflow\n    owner: o\n    repo: r\n    workflow_id: ci.yml\n",
	)
	_ = f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	t.Setenv("CI_TOKEN_TEST_407", "fake-token")
	if err := run(); err == nil {
		t.Fatal("expected error for invalid listen address")
	}
}

// ── mockRegistryServer — non-transient error ──────────────────────────────────

type mockRegistryServer struct {
	zynaxv1.UnimplementedAgentRegistryServiceServer
}

func (m *mockRegistryServer) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest) (*zynaxv1.RegisterAgentResponse, error) {
	return nil, status.Error(codes.AlreadyExists, "already registered")
}

func TestRun_RegistryNonTransientError(t *testing.T) {
	mockLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	mockGRPC := grpc.NewServer()
	zynaxv1.RegisterAgentRegistryServiceServer(mockGRPC, &mockRegistryServer{})
	go func() { _ = mockGRPC.Serve(mockLis) }()
	defer mockGRPC.Stop()

	f, err := os.CreateTemp(t.TempDir(), "ci-adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fmt.Fprintf(f,
		"agent_id: ci-test\nname: CI Test\n"+
			"endpoint: \"127.0.0.1:0\"\nregistry_endpoint: \"%s\"\n"+
			"ci:\n  provider: github-actions\n  token_env: CI_TOKEN_TEST_407\n"+
			"capabilities:\n  - name: trigger_workflow\n    owner: o\n    repo: r\n    workflow_id: ci.yml\n",
		mockLis.Addr().String(),
	)
	_ = f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	t.Setenv("CI_TOKEN_TEST_407", "fake-token")

	if err := run(); err == nil {
		t.Fatal("expected error when registry returns AlreadyExists (non-transient)")
	}
}

// ── serve() graceful shutdown ─────────────────────────────────────────────────

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

func TestServe_GracefulShutdown(t *testing.T) {
	cfg := &config.AdapterConfig{
		AgentID:          "ci-test",
		Name:             "CI Test",
		Endpoint:         "127.0.0.1:0",
		RegistryEndpoint: "127.0.0.1:0",
		CI: config.CIConfig{
			Provider:                  "github-actions",
			TokenEnv:                  "IGNORED",
			PollIntervalSeconds:       2,
			MaxPollIntervalSeconds:    30,
			TriggerPollTimeoutSeconds: 10,
		},
		Capabilities: []config.CICapabilityConfig{
			{Name: "trigger_workflow", Owner: "o", Repo: "r", WorkflowID: "ci.yml"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := &successRegistryClient{}

	done := make(chan error, 1)
	go func() { done <- serve(ctx, cfg, "fake-token", false, client) }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil from serve on graceful shutdown, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve() did not return within 5s after cancel")
	}
}

// failRegistryClient fails any RegisterAgent call. The degraded path must never
// reach it, so wiring it in proves no registration is attempted (issue #1375).
type failRegistryClient struct{ successRegistryClient }

func (c *failRegistryClient) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.RegisterAgentResponse, error) {
	return nil, status.Error(codes.Internal, "registry must not be called in degraded mode")
}

// TestServe_DegradedNoSecret proves the core fix (issue #1375): with no token the
// adapter serves, reports NOT_SERVING readiness, does NOT register its
// capabilities, and shuts down cleanly on context cancel — it does not crash.
func TestServe_DegradedNoSecret(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()

	cfg := &config.AdapterConfig{
		AgentID:          "ci-test",
		Name:             "CI Test",
		Endpoint:         addr,
		RegistryEndpoint: "127.0.0.1:0",
		CI:               config.CIConfig{Provider: "github-actions", TokenEnv: "IGNORED"},
		Capabilities:     []config.CICapabilityConfig{{Name: "trigger_workflow", Owner: "o", Repo: "r", WorkflowID: "ci.yml"}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	// Degraded=true with a registry client that errors on RegisterAgent: if the
	// degraded path tried to register, serve would return that error.
	go func() { done <- serve(ctx, cfg, "", true, &failRegistryClient{}) }()

	// Give the server a moment to bind, then probe health: must be NOT_SERVING.
	time.Sleep(100 * time.Millisecond)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial degraded adapter: %v", err)
	}
	defer func() { _ = conn.Close() }()
	hc := grpc_health_v1.NewHealthClient(conn)
	resp, err := hc.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
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
