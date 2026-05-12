// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	f.Close()
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
	f.Close()
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
	// Valid config but an invalid TCP listen address → net.Listen fails before registry dial.
	_, _ = f.WriteString("agent_id: test\nname: test\nendpoint: \"localhost:-1\"\nregistry_endpoint: \"127.0.0.1:9090\"\ncapabilities:\n  - name: api\n    method: POST\n    url: http://example.com\n")
	f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	if err := run(); err == nil {
		t.Fatal("expected error for invalid listen address")
	}
}

// mockRegistryServer returns AlreadyExists for RegisterAgent — a non-transient
// gRPC status that causes run() to fail immediately after grpc/health setup,
// exercising those code paths without requiring retry delays.
type mockRegistryServer struct {
	zynaxv1.UnimplementedAgentRegistryServiceServer
}

func (m *mockRegistryServer) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest) (*zynaxv1.RegisterAgentResponse, error) {
	return nil, status.Error(codes.AlreadyExists, "already registered")
}

func TestRun_RegistryNonTransientError(t *testing.T) {
	// Start a mock registry that immediately returns AlreadyExists (non-transient).
	mockLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	mockGRPC := grpc.NewServer()
	zynaxv1.RegisterAgentRegistryServiceServer(mockGRPC, &mockRegistryServer{})
	go func() { _ = mockGRPC.Serve(mockLis) }()
	defer mockGRPC.Stop()

	f, err := os.CreateTemp(t.TempDir(), "adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// Use port 0 so the OS picks a free port — net.Listen succeeds.
	_, _ = f.WriteString(fmt.Sprintf(
		"agent_id: test\nname: test\nendpoint: \"127.0.0.1:0\"\nregistry_endpoint: \"%s\"\ncapabilities:\n  - name: api\n    method: POST\n    url: http://example.com\n",
		mockLis.Addr().String(),
	))
	f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())

	if err := run(); err == nil {
		t.Fatal("expected error when registry returns non-transient error")
	}
}
