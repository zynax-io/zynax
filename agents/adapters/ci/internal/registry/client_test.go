// SPDX-License-Identifier: Apache-2.0

package registry_test

import (
	"context"
	"testing"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/ci/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/ci/internal/registry"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// stubRegistryClient implements AgentRegistryServiceClient for testing.
type stubRegistryClient struct {
	registerErr   error
	deregisterErr error
	calls         int
}

func (s *stubRegistryClient) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.RegisterAgentResponse, error) {
	s.calls++
	return &zynaxv1.RegisterAgentResponse{}, s.registerErr
}

func (s *stubRegistryClient) DeregisterAgent(_ context.Context, _ *zynaxv1.DeregisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.DeregisterAgentResponse, error) {
	return &zynaxv1.DeregisterAgentResponse{}, s.deregisterErr
}

func (s *stubRegistryClient) GetAgent(_ context.Context, _ *zynaxv1.GetAgentRequest, _ ...grpc.CallOption) (*zynaxv1.AgentDef, error) {
	return nil, nil
}

func (s *stubRegistryClient) ListAgents(_ context.Context, _ *zynaxv1.ListAgentsRequest, _ ...grpc.CallOption) (*zynaxv1.ListAgentsResponse, error) {
	return nil, nil
}

func (s *stubRegistryClient) FindByCapability(_ context.Context, _ *zynaxv1.FindByCapabilityRequest, _ ...grpc.CallOption) (*zynaxv1.FindByCapabilityResponse, error) {
	return nil, nil
}

func TestBuildAgentDef(t *testing.T) {
	t.Parallel()
	cfg := &config.AdapterConfig{
		AgentID:     "ci-adapter",
		Name:        "CI Adapter",
		Description: "triggers GitHub Actions workflows",
		Endpoint:    ":50055",
		Capabilities: []config.CICapabilityConfig{
			{
				Name:            "trigger_workflow",
				Description:     "Trigger a workflow_dispatch event",
				InputSchemaJSON: `{"type":"object"}`,
			},
			{
				Name:             "get_run_status",
				Description:      "Poll a workflow run until terminal state",
				OutputSchemaJSON: `{"type":"object"}`,
			},
		},
	}
	def := registry.BuildAgentDef(cfg)
	if def.AgentId != "ci-adapter" {
		t.Errorf("agent_id mismatch: got %q", def.AgentId)
	}
	if def.Name != "CI Adapter" {
		t.Errorf("name mismatch: got %q", def.Name)
	}
	if def.Endpoint != ":50055" {
		t.Errorf("endpoint mismatch: got %q", def.Endpoint)
	}
	if len(def.Capabilities) != 2 {
		t.Fatalf("expected 2 capabilities, got %d", len(def.Capabilities))
	}
	if def.Capabilities[0].Name != "trigger_workflow" {
		t.Errorf("first capability name: got %q", def.Capabilities[0].Name)
	}
	if string(def.Capabilities[0].InputSchema) != `{"type":"object"}` {
		t.Errorf("input_schema mismatch: got %q", def.Capabilities[0].InputSchema)
	}
}

func TestRegisterAgent_Success(t *testing.T) {
	t.Parallel()
	stub := &stubRegistryClient{}
	def := &zynaxv1.AgentDef{AgentId: "ci-adapter", Endpoint: ":50055"}
	err := registry.RegisterAgent(context.Background(), stub, def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.calls != 1 {
		t.Errorf("expected 1 register call, got %d", stub.calls)
	}
}

func TestRegisterAgent_NonTransientError(t *testing.T) {
	t.Parallel()
	stub := &stubRegistryClient{
		registerErr: status.Error(codes.InvalidArgument, "bad agent_id"),
	}
	def := &zynaxv1.AgentDef{AgentId: "ci-adapter"}
	err := registry.RegisterAgent(context.Background(), stub, def)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Non-transient: should fail on first attempt.
	if stub.calls != 1 {
		t.Errorf("expected 1 call (no retry on InvalidArgument), got %d", stub.calls)
	}
}

func TestRegisterAgent_TransientRetriesUntilCancelled(t *testing.T) {
	t.Parallel()
	// Always-Unavailable forces a retry (isTransient → true); a short ctx deadline
	// trips the backoff select's ctx.Done() branch instead of waiting the base delay.
	stub := &stubRegistryClient{registerErr: status.Error(codes.Unavailable, "down")}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err := registry.RegisterAgent(ctx, stub, &zynaxv1.AgentDef{AgentId: "ci-adapter"})
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
	if stub.calls < 1 {
		t.Errorf("expected at least one register attempt, got %d", stub.calls)
	}
}

func TestDeregisterAgent_Success(t *testing.T) {
	t.Parallel()
	stub := &stubRegistryClient{}
	err := registry.DeregisterAgent(context.Background(), stub, "ci-adapter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeregisterAgent_Error(t *testing.T) {
	t.Parallel()
	stub := &stubRegistryClient{
		deregisterErr: status.Error(codes.Internal, "registry down"),
	}
	err := registry.DeregisterAgent(context.Background(), stub, "ci-adapter")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
