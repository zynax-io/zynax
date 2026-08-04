// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/http/internal/config"
)

func TestRun_MissingEnvVar(t *testing.T) {
	t.Setenv("ADAPTER_CONFIG", "")
	if err := run(); err == nil {
		t.Fatal("expected error when ADAPTER_CONFIG is unset")
	}
}

func TestRun_MissingFile(t *testing.T) {
	t.Setenv("ADAPTER_CONFIG", "/nonexistent/path/adapter.yaml")
	if err := run(); err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestRun_InvalidConfig(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("{{invalid yaml")
	_ = f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	if err := run(); err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestRun_MissingRequiredFields(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// valid YAML but missing agent_id
	_, _ = f.WriteString("endpoint: \"0.0.0.0:8080\"\nregistry_endpoint: \"localhost:9090\"\ncapabilities:\n  - name: x\n    method: POST\n    url: http://x\n")
	_ = f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	if err := run(); err == nil {
		t.Fatal("expected error for config missing agent_id")
	}
}

func TestRun_InvalidListenAddr(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// Valid config but an invalid TCP listen address → net.Listen fails.
	_, _ = f.WriteString("agent_id: test\nname: test\nendpoint: \"localhost:-1\"\nregistry_endpoint: \"127.0.0.1:9090\"\ncapabilities:\n  - name: api\n    method: POST\n    url: http://example.com\n")
	_ = f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	if err := run(); err == nil {
		t.Fatal("expected error for invalid listen address")
	}
}

func serveConfig() *config.AdapterConfig {
	return &config.AdapterConfig{
		AgentID:      "http-test",
		Name:         "HTTP Test",
		Endpoint:     "127.0.0.1:0", // port 0 → a free port
		Capabilities: []config.RouteConfig{{Name: "api", Method: "POST", URL: "http://example.com"}},
	}
}

// TestServe_GracefulShutdown drives the serving path with a pre-cancelled
// context: bind → SERVING → shutdown signal → NOT_SERVING → GracefulStop.
func TestServe_GracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := serve(ctx, serveConfig()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
