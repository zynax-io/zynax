// SPDX-License-Identifier: Apache-2.0
// BDD contract tests for AgentRegistryService.
package agent_registry_service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cucumber/godog"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/protos/tests/testserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ─── Stub server ─────────────────────────────────────────────────────────────

type registryStub struct {
	zynaxv1.UnimplementedAgentRegistryServiceServer
	mu     sync.Mutex
	agents map[string]*zynaxv1.AgentDef
}

func newRegistryStub() *registryStub {
	return &registryStub{agents: make(map[string]*zynaxv1.AgentDef)}
}

func (s *registryStub) RegisterAgent(_ context.Context, req *zynaxv1.RegisterAgentRequest) (*zynaxv1.RegisterAgentResponse, error) {
	ag := req.GetAgent()
	if ag == nil {
		return nil, status.Error(codes.InvalidArgument, "agent must not be nil")
	}
	if ag.AgentId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id must not be empty")
	}
	if ag.Endpoint == "" {
		return nil, status.Error(codes.InvalidArgument, "endpoint must not be empty")
	}
	if len(ag.Capabilities) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one capability required")
	}
	for _, cap := range ag.Capabilities {
		if cap.Name == "" {
			return nil, status.Error(codes.InvalidArgument, "capability_name must not be empty")
		}
		if len(cap.InputSchema) > 0 && !json.Valid(cap.InputSchema) {
			return nil, status.Error(codes.InvalidArgument, "input_schema must be valid JSON")
		}
		if len(cap.OutputSchema) > 0 && !json.Valid(cap.OutputSchema) {
			return nil, status.Error(codes.InvalidArgument, "output_schema must be valid JSON")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.agents[ag.AgentId]; ok && existing.Status == zynaxv1.AgentStatus_AGENT_STATUS_REGISTERED {
		return nil, status.Errorf(codes.AlreadyExists, "agent %q already registered", ag.AgentId)
	}

	now := timestamppb.Now()
	ag.Status = zynaxv1.AgentStatus_AGENT_STATUS_REGISTERED
	ag.RegisteredAt = now
	ag.UpdatedAt = now
	s.agents[ag.AgentId] = ag

	return &zynaxv1.RegisterAgentResponse{
		AgentId:      ag.AgentId,
		RegisteredAt: now,
	}, nil
}

func (s *registryStub) DeregisterAgent(_ context.Context, req *zynaxv1.DeregisterAgentRequest) (*zynaxv1.DeregisterAgentResponse, error) {
	if req.AgentId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ag, ok := s.agents[req.AgentId]
	if !ok || ag.Status != zynaxv1.AgentStatus_AGENT_STATUS_REGISTERED {
		return nil, status.Errorf(codes.NotFound, "agent %q not found", req.AgentId)
	}
	now := timestamppb.Now()
	ag.Status = zynaxv1.AgentStatus_AGENT_STATUS_DEREGISTERED
	ag.UpdatedAt = now
	return &zynaxv1.DeregisterAgentResponse{DeregisteredAt: now}, nil
}

func (s *registryStub) GetAgent(_ context.Context, req *zynaxv1.GetAgentRequest) (*zynaxv1.AgentDef, error) {
	if req.AgentId == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ag, ok := s.agents[req.AgentId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "agent %q not found", req.AgentId)
	}
	return ag, nil
}

func (s *registryStub) ListAgents(_ context.Context, req *zynaxv1.ListAgentsRequest) (*zynaxv1.ListAgentsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*zynaxv1.AgentDef
	for _, ag := range s.agents {
		if !req.IncludeDeregistered && ag.Status != zynaxv1.AgentStatus_AGENT_STATUS_REGISTERED {
			continue
		}
		if req.LabelSelector != "" {
			if !matchesLabelSelector(ag.Labels, req.LabelSelector) {
				continue
			}
		}
		result = append(result, ag)
	}
	return &zynaxv1.ListAgentsResponse{Agents: result}, nil
}

func (s *registryStub) FindByCapability(_ context.Context, req *zynaxv1.FindByCapabilityRequest) (*zynaxv1.FindByCapabilityResponse, error) {
	if req.CapabilityName == "" {
		return nil, status.Error(codes.InvalidArgument, "capability_name must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*zynaxv1.AgentDef
	for _, ag := range s.agents {
		if ag.Status != zynaxv1.AgentStatus_AGENT_STATUS_REGISTERED {
			continue
		}
		for _, cap := range ag.Capabilities {
			if cap.Name == req.CapabilityName {
				result = append(result, ag)
				break
			}
		}
	}
	return &zynaxv1.FindByCapabilityResponse{Agents: result}, nil
}

func matchesLabelSelector(labels map[string]string, selector string) bool {
	parts := strings.Split(selector, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			return false
		}
		k, v := kv[0], kv[1]
		if labels[k] != v {
			return false
		}
	}
	return true
}

// ─── Test context ─────────────────────────────────────────────────────────────

type testCtx struct {
	client          zynaxv1.AgentRegistryServiceClient
	stub            *registryStub
	pendingAgent    *zynaxv1.AgentDef
	lastRegResp     *zynaxv1.RegisterAgentResponse
	lastAgent       *zynaxv1.AgentDef
	listResp        *zynaxv1.ListAgentsResponse
	findResp        *zynaxv1.FindByCapabilityResponse
	grpcErr         error
	lastLabels      map[string]string
}

func newTestCtx() *testCtx {
	return &testCtx{
		lastLabels: make(map[string]string),
	}
}

func (tc *testCtx) setupServer(t *testing.T) error {
	tc.stub = newRegistryStub()
	srv, dialer := testserver.NewBufconnServer(t)
	zynaxv1.RegisterAgentRegistryServiceServer(srv, tc.stub)
	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	t.Cleanup(func() { conn.Close() })
	tc.client = zynaxv1.NewAgentRegistryServiceClient(conn)
	return nil
}

func (tc *testCtx) registerAgent(agentID, cap string, labels map[string]string) error {
	caps := []*zynaxv1.CapabilityDef{{Name: cap}}
	ag := &zynaxv1.AgentDef{
		AgentId:      agentID,
		Name:         agentID,
		Endpoint:     "localhost:50051",
		Capabilities: caps,
		Labels:       labels,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := tc.client.RegisterAgent(ctx, &zynaxv1.RegisterAgentRequest{Agent: ag})
	return err
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			var tc *testCtx

			sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
				tc = newTestCtx()
				return ctx, nil
			})

			sc.Step(`^an AgentRegistryService is running on a test gRPC server$`, func(ctx context.Context) (context.Context, error) {
				return ctx, tc.setupServer(t)
			})

			sc.Step(`^the registry is empty$`, func(ctx context.Context) (context.Context, error) {
				return ctx, nil
			})

			sc.Step(`^a valid AgentDef with agent_id "([^"]*)"$`, func(ctx context.Context, agentID string) (context.Context, error) {
				tc.pendingAgent = &zynaxv1.AgentDef{
					AgentId:  agentID,
					Name:     agentID,
					Endpoint: "localhost:50051",
					Capabilities: []*zynaxv1.CapabilityDef{
						{Name: "default_cap"},
					},
					Labels: make(map[string]string),
				}
				return ctx, nil
			})

			sc.Step(`^the AgentDef declares capabilities \["([^"]+)"(?:, "([^"]+)")?\]$`, func(ctx context.Context, cap1, cap2 string) (context.Context, error) {
				tc.pendingAgent.Capabilities = []*zynaxv1.CapabilityDef{{Name: cap1}}
				if cap2 != "" {
					tc.pendingAgent.Capabilities = append(tc.pendingAgent.Capabilities, &zynaxv1.CapabilityDef{Name: cap2})
				}
				return ctx, nil
			})

			sc.Step(`^the AgentDef endpoint is "([^"]*)"$`, func(ctx context.Context, ep string) (context.Context, error) {
				if tc.pendingAgent != nil {
					tc.pendingAgent.Endpoint = ep
				}
				return ctx, nil
			})

			sc.Step(`^the AgentDef has labels \{"([^"]+)": "([^"]+)"(?:, "([^"]+)": "([^"]+)")?\}$`, func(ctx context.Context, k1, v1, k2, v2 string) (context.Context, error) {
				if tc.pendingAgent == nil {
					tc.pendingAgent = &zynaxv1.AgentDef{
						AgentId:  "agent-temp",
						Name:     "agent-temp",
						Endpoint: "localhost:50051",
						Capabilities: []*zynaxv1.CapabilityDef{{Name: "cap"}},
						Labels:   make(map[string]string),
					}
				}
				tc.pendingAgent.Labels[k1] = v1
				if k2 != "" {
					tc.pendingAgent.Labels[k2] = v2
				}
				// Also update the already-registered agent in the stub if it exists.
				if tc.stub != nil {
					tc.stub.mu.Lock()
					if ag, ok := tc.stub.agents[tc.pendingAgent.AgentId]; ok {
						if ag.Labels == nil {
							ag.Labels = make(map[string]string)
						}
						ag.Labels[k1] = v1
						if k2 != "" {
							ag.Labels[k2] = v2
						}
					}
					tc.stub.mu.Unlock()
				}
				return ctx, nil
			})

			sc.Step(`^RegisterAgent is called with the AgentDef$`, func(ctx context.Context) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				resp, err := tc.client.RegisterAgent(callCtx, &zynaxv1.RegisterAgentRequest{Agent: tc.pendingAgent})
				tc.lastRegResp = resp
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^RegisterAgent is called$`, func(ctx context.Context) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				var req *zynaxv1.RegisterAgentRequest
				if tc.pendingAgent != nil {
					req = &zynaxv1.RegisterAgentRequest{Agent: tc.pendingAgent}
				} else {
					req = &zynaxv1.RegisterAgentRequest{}
				}
				resp, err := tc.client.RegisterAgent(callCtx, req)
				tc.lastRegResp = resp
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^the response contains agent_id "([^"]*)"$`, func(ctx context.Context, agentID string) (context.Context, error) {
				if tc.lastRegResp == nil {
					return ctx, fmt.Errorf("no registration response")
				}
				if tc.lastRegResp.AgentId != agentID {
					return ctx, fmt.Errorf("expected agent_id %q, got %q", agentID, tc.lastRegResp.AgentId)
				}
				return ctx, nil
			})

			sc.Step(`^GetAgent for "([^"]*)" returns status REGISTERED$`, func(ctx context.Context, agentID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				ag, err := tc.client.GetAgent(callCtx, &zynaxv1.GetAgentRequest{AgentId: agentID})
				if err != nil {
					return ctx, err
				}
				if ag.Status != zynaxv1.AgentStatus_AGENT_STATUS_REGISTERED {
					return ctx, fmt.Errorf("expected REGISTERED, got %v", ag.Status)
				}
				return ctx, nil
			})

			sc.Step(`^GetAgent for "([^"]*)" returns both declared capabilities$`, func(ctx context.Context, agentID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				ag, err := tc.client.GetAgent(callCtx, &zynaxv1.GetAgentRequest{AgentId: agentID})
				if err != nil {
					return ctx, err
				}
				if len(ag.Capabilities) < 2 {
					return ctx, fmt.Errorf("expected 2 capabilities, got %d", len(ag.Capabilities))
				}
				return ctx, nil
			})

			sc.Step(`^GetAgent for "([^"]*)" returns a non-zero registered_at timestamp$`, func(ctx context.Context, agentID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				ag, err := tc.client.GetAgent(callCtx, &zynaxv1.GetAgentRequest{AgentId: agentID})
				if err != nil {
					return ctx, err
				}
				if ag.RegisteredAt == nil || ag.RegisteredAt.AsTime().IsZero() {
					return ctx, fmt.Errorf("expected non-zero registered_at")
				}
				return ctx, nil
			})

			sc.Step(`^agent "([^"]*)" is registered with capability "([^"]*)"$`, func(ctx context.Context, agentID, cap string) (context.Context, error) {
				err := tc.registerAgent(agentID, cap, nil)
				if err == nil {
					// Keep pendingAgent populated so subsequent label steps know which agent to update.
					tc.pendingAgent = &zynaxv1.AgentDef{
						AgentId: agentID,
						Labels:  make(map[string]string),
					}
				}
				return ctx, err
			})

			sc.Step(`^agent "([^"]*)" is registered with capabilities \["([^"]+)"(?:, "([^"]+)")?\]$`, func(ctx context.Context, agentID, cap1, cap2 string) (context.Context, error) {
				caps := []*zynaxv1.CapabilityDef{{Name: cap1}}
				if cap2 != "" {
					caps = append(caps, &zynaxv1.CapabilityDef{Name: cap2})
				}
				ag := &zynaxv1.AgentDef{
					AgentId:      agentID,
					Name:         agentID,
					Endpoint:     "localhost:50051",
					Capabilities: caps,
					Labels:       make(map[string]string),
				}
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_, err := tc.client.RegisterAgent(callCtx, &zynaxv1.RegisterAgentRequest{Agent: ag})
				return ctx, err
			})

			sc.Step(`^agent "([^"]*)" is registered with labels \{"([^"]+)": "([^"]+)"\}$`, func(ctx context.Context, agentID, k, v string) (context.Context, error) {
				return ctx, tc.registerAgent(agentID, "cap", map[string]string{k: v})
			})

			sc.Step(`^FindByCapability is called with capability_name "([^"]*)"$`, func(ctx context.Context, cap string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				resp, err := tc.client.FindByCapability(callCtx, &zynaxv1.FindByCapabilityRequest{CapabilityName: cap})
				tc.findResp = resp
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^the response contains agent "([^"]*)"$`, func(ctx context.Context, agentID string) (context.Context, error) {
				var agents []*zynaxv1.AgentDef
				if tc.findResp != nil {
					agents = tc.findResp.Agents
				} else if tc.listResp != nil {
					agents = tc.listResp.Agents
				}
				for _, ag := range agents {
					if ag.AgentId == agentID {
						return ctx, nil
					}
				}
				return ctx, fmt.Errorf("agent %q not in response", agentID)
			})

			sc.Step(`^the response does not contain agent "([^"]*)"$`, func(ctx context.Context, agentID string) (context.Context, error) {
				var agents []*zynaxv1.AgentDef
				if tc.findResp != nil {
					agents = tc.findResp.Agents
				} else if tc.listResp != nil {
					agents = tc.listResp.Agents
				}
				for _, ag := range agents {
					if ag.AgentId == agentID {
						return ctx, fmt.Errorf("agent %q should not be in response", agentID)
					}
				}
				return ctx, nil
			})

			sc.Step(`^the response contains no agents$`, func(ctx context.Context) (context.Context, error) {
				var count int
				if tc.findResp != nil {
					count = len(tc.findResp.Agents)
				} else if tc.listResp != nil {
					count = len(tc.listResp.Agents)
				}
				if count != 0 {
					return ctx, fmt.Errorf("expected no agents, got %d", count)
				}
				return ctx, nil
			})

			sc.Step(`^the gRPC status is OK$`, func(ctx context.Context) (context.Context, error) {
				if tc.grpcErr != nil {
					return ctx, fmt.Errorf("expected OK but got: %v", tc.grpcErr)
				}
				return ctx, nil
			})

			sc.Step(`^ListAgents is called with label selector "([^"]*)"$`, func(ctx context.Context, sel string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				resp, err := tc.client.ListAgents(callCtx, &zynaxv1.ListAgentsRequest{LabelSelector: sel})
				tc.listResp = resp
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^ListAgents is called with no label selector$`, func(ctx context.Context) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				resp, err := tc.client.ListAgents(callCtx, &zynaxv1.ListAgentsRequest{})
				tc.listResp = resp
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^GetAgent is called with agent_id "([^"]*)"$`, func(ctx context.Context, agentID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				ag, err := tc.client.GetAgent(callCtx, &zynaxv1.GetAgentRequest{AgentId: agentID})
				tc.lastAgent = ag
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^the response agent_id is "([^"]*)"$`, func(ctx context.Context, agentID string) (context.Context, error) {
				if tc.lastAgent == nil {
					return ctx, fmt.Errorf("no agent response")
				}
				if tc.lastAgent.AgentId != agentID {
					return ctx, fmt.Errorf("expected agent_id %q, got %q", agentID, tc.lastAgent.AgentId)
				}
				return ctx, nil
			})

			sc.Step(`^the response includes capability "([^"]*)"$`, func(ctx context.Context, cap string) (context.Context, error) {
				if tc.lastAgent == nil {
					return ctx, fmt.Errorf("no agent response")
				}
				for _, c := range tc.lastAgent.Capabilities {
					if c.Name == cap {
						return ctx, nil
					}
				}
				return ctx, fmt.Errorf("capability %q not found", cap)
			})

			sc.Step(`^the response includes label "([^"]*)" with value "([^"]*)"$`, func(ctx context.Context, k, v string) (context.Context, error) {
				if tc.lastAgent == nil {
					return ctx, fmt.Errorf("no agent response")
				}
				if tc.lastAgent.Labels[k] != v {
					return ctx, fmt.Errorf("label %q=%q not found", k, v)
				}
				return ctx, nil
			})

			sc.Step(`^the response status is REGISTERED$`, func(ctx context.Context) (context.Context, error) {
				if tc.lastAgent == nil {
					return ctx, fmt.Errorf("no agent response")
				}
				if tc.lastAgent.Status != zynaxv1.AgentStatus_AGENT_STATUS_REGISTERED {
					return ctx, fmt.Errorf("expected REGISTERED, got %v", tc.lastAgent.Status)
				}
				return ctx, nil
			})

			sc.Step(`^DeregisterAgent is called with agent_id "([^"]*)"$`, func(ctx context.Context, agentID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_, err := tc.client.DeregisterAgent(callCtx, &zynaxv1.DeregisterAgentRequest{AgentId: agentID})
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^GetAgent for "([^"]*)" returns status DEREGISTERED$`, func(ctx context.Context, agentID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				ag, err := tc.client.GetAgent(callCtx, &zynaxv1.GetAgentRequest{AgentId: agentID})
				if err != nil {
					return ctx, err
				}
				if ag.Status != zynaxv1.AgentStatus_AGENT_STATUS_DEREGISTERED {
					return ctx, fmt.Errorf("expected DEREGISTERED, got %v", ag.Status)
				}
				return ctx, nil
			})

			sc.Step(`^DeregisterAgent has been called for "([^"]*)"$`, func(ctx context.Context, agentID string) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_, err := tc.client.DeregisterAgent(callCtx, &zynaxv1.DeregisterAgentRequest{AgentId: agentID})
				return ctx, err
			})

			sc.Step(`^RegisterAgent is called again with agent_id "([^"]*)"$`, func(ctx context.Context, agentID string) (context.Context, error) {
				ag := &zynaxv1.AgentDef{
					AgentId:      agentID,
					Name:         agentID,
					Endpoint:     "localhost:50051",
					Capabilities: []*zynaxv1.CapabilityDef{{Name: "cap"}},
				}
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_, err := tc.client.RegisterAgent(callCtx, &zynaxv1.RegisterAgentRequest{Agent: ag})
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^the gRPC status is ALREADY_EXISTS$`, func(ctx context.Context) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				st, _ := status.FromError(tc.grpcErr)
				if st.Code() != codes.AlreadyExists {
					return ctx, fmt.Errorf("expected ALREADY_EXISTS, got %v", st.Code())
				}
				return ctx, nil
			})

			sc.Step(`^the error message contains "([^"]*)"$`, func(ctx context.Context, substr string) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				if !strings.Contains(tc.grpcErr.Error(), substr) {
					return ctx, fmt.Errorf("error %q doesn't contain %q", tc.grpcErr.Error(), substr)
				}
				return ctx, nil
			})

			sc.Step(`^the gRPC status is NOT_FOUND$`, func(ctx context.Context) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				st, _ := status.FromError(tc.grpcErr)
				if st.Code() != codes.NotFound {
					return ctx, fmt.Errorf("expected NOT_FOUND, got %v", st.Code())
				}
				return ctx, nil
			})

			sc.Step(`^a RegisterAgentRequest with agent_id set to ""$`, func(ctx context.Context) (context.Context, error) {
				tc.pendingAgent = &zynaxv1.AgentDef{
					AgentId:      "",
					Name:         "test",
					Endpoint:     "localhost:50051",
					Capabilities: []*zynaxv1.CapabilityDef{{Name: "cap"}},
				}
				return ctx, nil
			})

			sc.Step(`^the gRPC status is INVALID_ARGUMENT$`, func(ctx context.Context) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				st, _ := status.FromError(tc.grpcErr)
				if st.Code() != codes.InvalidArgument {
					return ctx, fmt.Errorf("expected INVALID_ARGUMENT, got %v", st.Code())
				}
				return ctx, nil
			})

			sc.Step(`^the error message mentions "([^"]*)"$`, func(ctx context.Context, field string) (context.Context, error) {
				if tc.grpcErr == nil {
					return ctx, fmt.Errorf("expected error")
				}
				if !strings.Contains(tc.grpcErr.Error(), field) {
					return ctx, fmt.Errorf("error %q doesn't mention %q", tc.grpcErr.Error(), field)
				}
				return ctx, nil
			})

			sc.Step(`^a RegisterAgentRequest with endpoint set to ""$`, func(ctx context.Context) (context.Context, error) {
				tc.pendingAgent = &zynaxv1.AgentDef{
					AgentId:      "agent-test",
					Name:         "test",
					Endpoint:     "",
					Capabilities: []*zynaxv1.CapabilityDef{{Name: "cap"}},
				}
				return ctx, nil
			})

			sc.Step(`^a RegisterAgentRequest where one CapabilityDef has name set to ""$`, func(ctx context.Context) (context.Context, error) {
				tc.pendingAgent = &zynaxv1.AgentDef{
					AgentId:      "agent-test",
					Name:         "test",
					Endpoint:     "localhost:50051",
					Capabilities: []*zynaxv1.CapabilityDef{{Name: ""}},
				}
				return ctx, nil
			})

			sc.Step(`^a FindByCapabilityRequest with capability_name set to ""$`, func(ctx context.Context) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				resp, err := tc.client.FindByCapability(callCtx, &zynaxv1.FindByCapabilityRequest{CapabilityName: ""})
				tc.findResp = resp
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^FindByCapability is called$`, func(ctx context.Context) (context.Context, error) {
				return ctx, nil // already called
			})

			sc.Step(`^a GetAgentRequest with agent_id set to ""$`, func(ctx context.Context) (context.Context, error) {
				callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				ag, err := tc.client.GetAgent(callCtx, &zynaxv1.GetAgentRequest{AgentId: ""})
				tc.lastAgent = ag
				tc.grpcErr = err
				return ctx, nil
			})

			sc.Step(`^GetAgent is called$`, func(ctx context.Context) (context.Context, error) {
				return ctx, nil // already called
			})

			sc.Step(`^a RegisterAgentRequest where one CapabilityDef has input_schema "([^"]*)"$`, func(ctx context.Context, schema string) (context.Context, error) {
				tc.pendingAgent = &zynaxv1.AgentDef{
					AgentId:  "agent-test",
					Name:     "test",
					Endpoint: "localhost:50051",
					Capabilities: []*zynaxv1.CapabilityDef{{
						Name:        "cap",
						InputSchema: []byte(schema),
					}},
				}
				return ctx, nil
			})

			sc.Step(`^a RegisterAgentRequest where one CapabilityDef has output_schema "([^"]*)"$`, func(ctx context.Context, schema string) (context.Context, error) {
				tc.pendingAgent = &zynaxv1.AgentDef{
					AgentId:  "agent-test",
					Name:     "test",
					Endpoint: "localhost:50051",
					Capabilities: []*zynaxv1.CapabilityDef{{
						Name:         "cap",
						OutputSchema: []byte(schema),
					}},
				}
				return ctx, nil
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/agent_registry_service.feature"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
