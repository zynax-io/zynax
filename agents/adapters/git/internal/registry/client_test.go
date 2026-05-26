// SPDX-License-Identifier: Apache-2.0

package registry_test

import (
	"context"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/registry"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
)

func TestBuildAgentDef(t *testing.T) {
	t.Parallel()
	cfg := &config.AdapterConfig{
		AgentID:     "git-adapter",
		Name:        "Git Adapter",
		Description: "wraps GitHub ops",
		Endpoint:    ":50060",
		Capabilities: []config.GitCapabilityConfig{
			{
				Name:            "open_pr",
				Description:     "Open a PR",
				InputSchemaJSON: `{"type":"object"}`,
			},
		},
	}
	def := registry.BuildAgentDef(cfg)
	if def.AgentId != "git-adapter" {
		t.Errorf("agent_id mismatch: got %q", def.AgentId)
	}
	if def.Endpoint != ":50060" {
		t.Errorf("endpoint mismatch: got %q", def.Endpoint)
	}
	if len(def.Capabilities) != 1 {
		t.Fatalf("expected 1 capability, got %d", len(def.Capabilities))
	}
	if def.Capabilities[0].Name != "open_pr" {
		t.Errorf("capability name mismatch: got %q", def.Capabilities[0].Name)
	}
	if string(def.Capabilities[0].InputSchema) != `{"type":"object"}` {
		t.Errorf("input_schema mismatch: got %q", def.Capabilities[0].InputSchema)
	}
}

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

func TestRegisterAgent_Success(t *testing.T) {
	t.Parallel()
	stub := &stubRegistryClient{}
	def := &zynaxv1.AgentDef{AgentId: "git-adapter", Endpoint: ":50060"}
	err := registry.RegisterAgent(context.Background(), stub, def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.calls != 1 {
		t.Errorf("expected 1 register call, got %d", stub.calls)
	}
}

func TestDeregisterAgent_Success(t *testing.T) {
	t.Parallel()
	stub := &stubRegistryClient{}
	err := registry.DeregisterAgent(context.Background(), stub, "git-adapter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
