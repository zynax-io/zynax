// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/provider"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/server"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

const validConfig = `
agent_id: llm-adapter-test
name: LLM Adapter
endpoint: 127.0.0.1:0
registry_endpoint: localhost:50052
capabilities:
  - name: chat_completion
provider:
  name: openai
  model: gpt-4o
  api_key_env: OPENAI_API_KEY
`

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestRun_MissingEnv(t *testing.T) {
	t.Setenv(configEnvVar, "")
	if err := run(); err == nil {
		t.Fatal("expected error when config env var unset")
	}
}

func TestRun_BadConfigPath(t *testing.T) {
	t.Setenv(configEnvVar, "/nonexistent/config.yaml")
	if err := run(); err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestRun_InvalidConfig(t *testing.T) {
	t.Setenv(configEnvVar, writeConfig(t, "{{invalid yaml"))
	if err := run(); err == nil {
		t.Fatal("expected error for invalid config")
	}
}

// TestBuild_UnsetSecretDegrades proves the core fix (issue #1375): with the API
// key env unset, build() does NOT error — it returns a degraded flag, a nil
// server, and the loaded config so the adapter can start without crash-looping.
func TestBuild_UnsetSecretDegrades(t *testing.T) {
	t.Setenv(configEnvVar, writeConfig(t, validConfig))
	t.Setenv("OPENAI_API_KEY", "")
	cfg, srv, degraded, err := build()
	if err != nil {
		t.Fatalf("build must not error when api key unset, got: %v", err)
	}
	if !degraded {
		t.Fatal("expected degraded=true when api key unset")
	}
	if srv != nil {
		t.Fatal("expected nil server in degraded mode (no provider built)")
	}
	if cfg == nil {
		t.Fatal("expected config to be returned in degraded mode")
	}
}

func TestRun_InvalidListenAddr(t *testing.T) {
	// Valid config but an invalid TCP listen address → net.Listen fails after
	// the registry dial (NewClient is lazy and does not fail on a bad target).
	t.Setenv(configEnvVar, writeConfig(t, `
agent_id: llm-adapter-test
name: LLM Adapter
endpoint: 127.0.0.1:-1
registry_endpoint: localhost:50052
capabilities:
  - name: chat_completion
provider:
  name: openai
  model: gpt-4o
  api_key_env: OPENAI_API_KEY
`))
	t.Setenv("OPENAI_API_KEY", "sk-test-value")
	if err := run(); err == nil {
		t.Fatal("expected error for invalid listen address")
	}
}

// mockRegistryServer returns AlreadyExists for RegisterAgent — a non-transient
// gRPC status that causes run() to fail immediately after grpc/health setup,
// exercising the serve path without requiring retry delays or a live signal.
type mockRegistryServer struct {
	zynaxv1.UnimplementedAgentRegistryServiceServer
}

func (m *mockRegistryServer) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest) (*zynaxv1.RegisterAgentResponse, error) {
	return nil, status.Error(codes.AlreadyExists, "already registered")
}

// fakeRegistryClient is an in-process AgentRegistryServiceClient stub that
// records register/deregister calls and always succeeds, so serve() can run its
// full happy path (register → SERVING → SIGTERM → deregister → GracefulStop).
type fakeRegistryClient struct {
	zynaxv1.AgentRegistryServiceClient
	deregistered chan string
}

func (f *fakeRegistryClient) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.RegisterAgentResponse, error) {
	return &zynaxv1.RegisterAgentResponse{}, nil
}

func (f *fakeRegistryClient) DeregisterAgent(_ context.Context, req *zynaxv1.DeregisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.DeregisterAgentResponse, error) {
	f.deregistered <- req.GetAgentId()
	return &zynaxv1.DeregisterAgentResponse{}, nil
}

func TestServe_GracefulShutdown(t *testing.T) {
	cfg := &config.AdapterConfig{
		AgentID:          "llm-adapter-test",
		Endpoint:         "127.0.0.1:0",
		RegistryEndpoint: "localhost:50052",
		Capabilities:     []config.CapabilityConfig{{Name: "chat_completion"}},
		Provider:         config.ProviderConfig{Name: "ollama", Model: "llama3", OllamaBaseURL: "http://localhost:11434"},
	}
	prov, err := provider.New(cfg.Provider, config.Secret{})
	if err != nil {
		t.Fatalf("build provider: %v", err)
	}
	srv, err := server.NewAgentServer(cfg, prov)
	if err != nil {
		t.Fatalf("build server: %v", err)
	}
	fake := &fakeRegistryClient{deregistered: make(chan string, 1)}

	done := make(chan error, 1)
	go func() { done <- serve(cfg, srv, false, fake) }()

	// Allow serve() to register and enter its select before signalling SIGTERM.
	time.Sleep(200 * time.Millisecond)
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}

	select {
	case agentID := <-fake.deregistered:
		if agentID != "llm-adapter-test" {
			t.Errorf("deregistered agent_id = %s, want llm-adapter-test", agentID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for deregister on shutdown")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("serve returned error on clean shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for serve to return")
	}
}

// failRegisterClient errors on RegisterAgent. The degraded path must never call
// it, so serve must return nil even with this client wired in (issue #1375).
type failRegisterClient struct {
	zynaxv1.AgentRegistryServiceClient
}

func (f *failRegisterClient) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.RegisterAgentResponse, error) {
	return nil, status.Error(codes.Internal, "registry must not be called in degraded mode")
}

// TestServe_DegradedNoSecret proves the core fix (issue #1375): with no secret
// (nil server, degraded=true) the adapter serves, reports NOT_SERVING readiness,
// does NOT register, and shuts down cleanly — it does not crash.
func TestServe_DegradedNoSecret(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()

	cfg := &config.AdapterConfig{
		AgentID:          "llm-adapter-test",
		Endpoint:         addr,
		RegistryEndpoint: "localhost:50052",
		Capabilities:     []config.CapabilityConfig{{Name: "chat_completion"}},
		Provider:         config.ProviderConfig{Name: "openai", Model: "gpt-4o", KeyEnvVar: "OPENAI_API_KEY"}, //nolint:gosec // G101: env-var NAME, not a credential value
	}

	done := make(chan error, 1)
	go func() { done <- serve(cfg, nil, true, &failRegisterClient{}) }()

	time.Sleep(200 * time.Millisecond)
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

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("degraded serve must not error (no crash), got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("degraded serve did not return within 5s")
	}
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

	t.Setenv(configEnvVar, writeConfig(t, fmt.Sprintf(`
agent_id: llm-adapter-test
name: LLM Adapter
endpoint: 127.0.0.1:0
registry_endpoint: %q
capabilities:
  - name: chat_completion
provider:
  name: openai
  model: gpt-4o
  api_key_env: OPENAI_API_KEY
`, mockLis.Addr().String())))
	t.Setenv("OPENAI_API_KEY", "sk-test-value")

	if err := run(); err == nil {
		t.Fatal("expected error when registry returns non-transient error")
	}
}
