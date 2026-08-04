// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/provider"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
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
	// Valid config but an invalid TCP listen address → net.Listen fails.
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

func TestServe_GracefulShutdown(t *testing.T) {
	cfg := &config.AdapterConfig{
		AgentID:      "llm-adapter-test",
		Endpoint:     "127.0.0.1:0",
		Capabilities: []config.CapabilityConfig{{Name: "chat_completion"}},
		Provider:     config.ProviderConfig{Name: "ollama", Model: "llama3", OllamaBaseURL: "http://localhost:11434"},
	}
	prov, err := provider.New(cfg.Provider, config.Secret{})
	if err != nil {
		t.Fatalf("build provider: %v", err)
	}
	srv, err := server.NewAgentServer(cfg, prov)
	if err != nil {
		t.Fatalf("build server: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- serve(cfg, srv, false) }()

	// Allow serve() to bind and enter its select before signalling SIGTERM.
	time.Sleep(200 * time.Millisecond)
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM: %v", err)
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

// TestServe_DegradedNoSecret proves the core fix (issue #1375): with no secret
// (nil server, degraded=true) the adapter serves, reports NOT_SERVING readiness,
// serves no AgentService, and shuts down cleanly — it does not crash.
func TestServe_DegradedNoSecret(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()

	cfg := &config.AdapterConfig{
		AgentID:      "llm-adapter-test",
		Endpoint:     addr,
		Capabilities: []config.CapabilityConfig{{Name: "chat_completion"}},
		Provider:     config.ProviderConfig{Name: "openai", Model: "gpt-4o", KeyEnvVar: "OPENAI_API_KEY"}, //nolint:gosec // G101: env-var NAME, not a credential value
	}

	done := make(chan error, 1)
	go func() { done <- serve(cfg, nil, true) }()

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
