// SPDX-License-Identifier: Apache-2.0

// Package main (whitebox test) exercises run() and serve().
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/adk/internal/config"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "adk-adapter.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func serveConfig() *config.AdapterConfig {
	return &config.AdapterConfig{
		AgentID:          "adk-1",
		Endpoint:         "127.0.0.1:0", // port 0 → a free port
		RegistryEndpoint: "127.0.0.1:50052",
		Capabilities:     []config.CapabilityConfig{{Name: "triage"}},
	}
}

func TestRun_MissingEnvVar(t *testing.T) {
	t.Setenv("ADAPTER_CONFIG", "")
	if err := run(); err == nil {
		t.Fatal("expected error when ADAPTER_CONFIG is empty")
	}
}

func TestRun_MissingFile(t *testing.T) {
	t.Setenv("ADAPTER_CONFIG", filepath.Join(t.TempDir(), "nope.yaml"))
	if err := run(); err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestRun_InvalidConfig(t *testing.T) {
	// Valid YAML, but missing agent_id → config.validate fails before serve().
	path := writeConfig(t, "registry_endpoint: r:1\ncapabilities:\n  - {name: c}\n")
	t.Setenv("ADAPTER_CONFIG", path)
	if err := run(); err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestServe_ListenError(t *testing.T) {
	cfg := serveConfig()
	cfg.Endpoint = "missing-port" // net.Listen rejects an address with no port
	err := serve(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "listen") {
		t.Fatalf("expected listen error, got %v", err)
	}
}

func TestServe_GracefulShutdown(t *testing.T) {
	// A pre-cancelled context makes serve report SERVING, then immediately
	// enter the shutdown path: NOT_SERVING + GracefulStop.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := serve(ctx, serveConfig()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
