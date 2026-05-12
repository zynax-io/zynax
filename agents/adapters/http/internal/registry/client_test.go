// SPDX-License-Identifier: Apache-2.0

package registry_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/http/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/http/internal/registry"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockClient is a test double for AgentRegistryServiceClient.
// registerResponses is consumed in order; the last entry is repeated.
type mockClient struct {
	registerResponses []error
	registerCalls     int
	deregisterErr     error
	deregisterCalls   int
}

func (m *mockClient) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.RegisterAgentResponse, error) {
	idx := m.registerCalls
	m.registerCalls++
	if idx < len(m.registerResponses) {
		return nil, m.registerResponses[idx]
	}
	return nil, m.registerResponses[len(m.registerResponses)-1]
}

func (m *mockClient) DeregisterAgent(_ context.Context, _ *zynaxv1.DeregisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.DeregisterAgentResponse, error) {
	m.deregisterCalls++
	return nil, m.deregisterErr
}

func (m *mockClient) GetAgent(_ context.Context, _ *zynaxv1.GetAgentRequest, _ ...grpc.CallOption) (*zynaxv1.AgentDef, error) {
	return nil, status.Error(codes.Unimplemented, "not used in tests")
}

func (m *mockClient) ListAgents(_ context.Context, _ *zynaxv1.ListAgentsRequest, _ ...grpc.CallOption) (*zynaxv1.ListAgentsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not used in tests")
}

func (m *mockClient) FindByCapability(_ context.Context, _ *zynaxv1.FindByCapabilityRequest, _ ...grpc.CallOption) (*zynaxv1.FindByCapabilityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not used in tests")
}

var testDef = &zynaxv1.AgentDef{
	AgentId:  "test-agent",
	Endpoint: "0.0.0.0:8080",
	Capabilities: []*zynaxv1.CapabilityDef{
		{Name: "call_api"},
	},
}

func TestRegisterAgent_SuccessFirstAttempt(t *testing.T) {
	mock := &mockClient{registerResponses: []error{nil}}
	if err := registry.RegisterAgent(context.Background(), mock, testDef); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.registerCalls != 1 {
		t.Errorf("registerCalls = %d, want 1", mock.registerCalls)
	}
}

func TestRegisterAgent_RetryAfterTransientFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping retry backoff test")
	}
	unavail := status.Error(codes.Unavailable, "service down")
	mock := &mockClient{
		registerResponses: []error{unavail, unavail, nil},
	}
	if err := registry.RegisterAgent(context.Background(), mock, testDef); err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if mock.registerCalls != 3 {
		t.Errorf("registerCalls = %d, want 3", mock.registerCalls)
	}
}

func TestRegisterAgent_ExhaustsAllAttempts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping retry exhaustion test")
	}
	unavail := status.Error(codes.Unavailable, "always down")
	mock := &mockClient{registerResponses: []error{unavail}}
	err := registry.RegisterAgent(context.Background(), mock, testDef)
	if err == nil {
		t.Fatal("expected error after all attempts exhausted")
	}
	if mock.registerCalls != 5 {
		t.Errorf("registerCalls = %d, want 5", mock.registerCalls)
	}
}

func TestRegisterAgent_NonTransientFailure_NoRetry(t *testing.T) {
	invalid := status.Error(codes.InvalidArgument, "bad request")
	mock := &mockClient{registerResponses: []error{invalid}}
	err := registry.RegisterAgent(context.Background(), mock, testDef)
	if err == nil {
		t.Fatal("expected error for non-transient failure")
	}
	if mock.registerCalls != 1 {
		t.Errorf("registerCalls = %d, want 1 (no retry on non-transient)", mock.registerCalls)
	}
}

func TestRegisterAgent_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	unavail := status.Error(codes.Unavailable, "down")
	mock := &mockClient{registerResponses: []error{unavail}}
	err := registry.RegisterAgent(ctx, mock, testDef)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled in error chain, got: %v", err)
	}
}

func TestDeregisterAgent_Success(t *testing.T) {
	mock := &mockClient{deregisterErr: nil}
	if err := registry.DeregisterAgent(context.Background(), mock, "test-agent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.deregisterCalls != 1 {
		t.Errorf("deregisterCalls = %d, want 1", mock.deregisterCalls)
	}
}

func TestDeregisterAgent_Error(t *testing.T) {
	mock := &mockClient{deregisterErr: status.Error(codes.NotFound, "not found")}
	err := registry.DeregisterAgent(context.Background(), mock, "unknown")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegisterAgent_NonGRPCError_NonTransient(t *testing.T) {
	// A plain Go error (not a gRPC status) is not transient; RegisterAgent must
	// return immediately without retrying, covering the !ok branch of isTransient.
	plainErr := errors.New("plain network error")
	mock := &mockClient{registerResponses: []error{plainErr}}
	err := registry.RegisterAgent(context.Background(), mock, testDef)
	if err == nil {
		t.Fatal("expected error for non-gRPC error")
	}
	if mock.registerCalls != 1 {
		t.Errorf("registerCalls = %d, want 1 (no retry for non-gRPC errors)", mock.registerCalls)
	}
}

func TestBuildAgentDef(t *testing.T) {
	cfg := &config.AdapterConfig{
		AgentID:     "http-adapter",
		Name:        "HTTP Adapter",
		Description: "REST proxy",
		Endpoint:    "0.0.0.0:8080",
		Capabilities: []config.RouteConfig{
			{Name: "call_api", Description: "calls api", InputSchemaJSON: `{"type":"object"}`},
		},
	}
	def := registry.BuildAgentDef(cfg)
	if def.AgentId != "http-adapter" {
		t.Errorf("agent_id = %s", def.AgentId)
	}
	if len(def.Capabilities) != 1 {
		t.Fatalf("capabilities len = %d, want 1", len(def.Capabilities))
	}
	if def.Capabilities[0].Name != "call_api" {
		t.Errorf("capability name = %s", def.Capabilities[0].Name)
	}
	if string(def.Capabilities[0].InputSchema) != `{"type":"object"}` {
		t.Errorf("input_schema = %s", def.Capabilities[0].InputSchema)
	}
}
