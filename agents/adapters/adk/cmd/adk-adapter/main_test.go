// SPDX-License-Identifier: Apache-2.0

// Package main (whitebox test) exercises run(), dialRegistry(), and serve().
package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/adk/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeRegistry implements AgentRegistryServiceClient; only Register/Deregister
// are real (the embedded interface supplies the rest).
type fakeRegistry struct {
	zynaxv1.AgentRegistryServiceClient
	registerErr  error
	deregistered string
}

func (f *fakeRegistry) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.RegisterAgentResponse, error) {
	if f.registerErr != nil {
		return nil, f.registerErr
	}
	return &zynaxv1.RegisterAgentResponse{}, nil
}

func (f *fakeRegistry) DeregisterAgent(_ context.Context, in *zynaxv1.DeregisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.DeregisterAgentResponse, error) {
	f.deregistered = in.AgentId
	return &zynaxv1.DeregisterAgentResponse{}, nil
}

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

func TestDialRegistry(t *testing.T) {
	client, cleanup, err := dialRegistry("127.0.0.1:50052")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	cleanup()
}

func TestServe_ListenError(t *testing.T) {
	cfg := serveConfig()
	cfg.Endpoint = "missing-port" // net.Listen rejects an address with no port
	err := serve(context.Background(), cfg, &fakeRegistry{})
	if err == nil || !strings.Contains(err.Error(), "listen") {
		t.Fatalf("expected listen error, got %v", err)
	}
}

func TestServe_RegisterFailure(t *testing.T) {
	err := serve(context.Background(), serveConfig(), &fakeRegistry{registerErr: status.Error(codes.InvalidArgument, "bad")})
	if err == nil || !strings.Contains(err.Error(), "register") {
		t.Fatalf("expected register error, got %v", err)
	}
}

func TestServe_GracefulShutdown(t *testing.T) {
	// A pre-cancelled context makes serve register, report SERVING, then
	// immediately enter the shutdown path: deregister + GracefulStop.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &fakeRegistry{}
	if err := serve(ctx, serveConfig(), f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.deregistered != "adk-1" {
		t.Errorf("deregistered = %q, want adk-1", f.deregistered)
	}
}
