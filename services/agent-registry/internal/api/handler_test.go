// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/agent-registry/internal/api"
	"github.com/zynax-io/zynax/services/agent-registry/internal/domain"
	"github.com/zynax-io/zynax/services/agent-registry/internal/infrastructure"
)

const bufSize = 1024 * 1024

func newTestClient(t *testing.T) zynaxv1.AgentRegistryServiceClient {
	t.Helper()
	repo := infrastructure.NewMemoryRepo()
	svc := domain.NewAgentRegistryService(repo)
	h := api.NewHandler(svc)

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	zynaxv1.RegisterAgentRegistryServiceServer(srv, h)
	t.Cleanup(func() { srv.GracefulStop() })
	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("bufconn dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return zynaxv1.NewAgentRegistryServiceClient(conn)
}

func validAgentDef(id string, caps ...string) *zynaxv1.AgentDef {
	capDefs := make([]*zynaxv1.CapabilityDef, len(caps))
	for i, c := range caps {
		capDefs[i] = &zynaxv1.CapabilityDef{Name: c}
	}
	return &zynaxv1.AgentDef{
		AgentId:      id,
		Name:         "Test Agent",
		Endpoint:     "localhost:9000",
		Capabilities: capDefs,
	}
}

// ── RegisterAgent ─────────────────────────────────────────────────────────────

func TestRegisterAgent_HappyPath(t *testing.T) {
	client := newTestClient(t)

	resp, err := client.RegisterAgent(context.Background(), &zynaxv1.RegisterAgentRequest{
		Agent: validAgentDef("reg-01", "summarize"),
	})
	if err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}
	if resp.AgentId != "reg-01" {
		t.Errorf("agent_id = %q, want %q", resp.AgentId, "reg-01")
	}
	if resp.RegisteredAt == nil {
		t.Error("want non-nil registered_at")
	}
}

// TestRegisterAgent_IsIdempotent covers issue #1463: a restarted capability adapter
// pod re-registers the same agent_id; this must return OK (not ALREADY_EXISTS) so the
// pod does not crash-loop. The latest endpoint is upserted.
func TestRegisterAgent_IsIdempotent(t *testing.T) {
	client := newTestClient(t)

	req := &zynaxv1.RegisterAgentRequest{Agent: validAgentDef("dup-01", "summarize")}
	if _, err := client.RegisterAgent(context.Background(), req); err != nil {
		t.Fatalf("first RegisterAgent: %v", err)
	}

	// Simulate the adapter pod restarting and re-registering with a fresh endpoint.
	updated := validAgentDef("dup-01", "summarize")
	updated.Endpoint = "new-host:9100"
	resp, err := client.RegisterAgent(context.Background(), &zynaxv1.RegisterAgentRequest{Agent: updated})
	if err != nil {
		t.Fatalf("re-RegisterAgent: want OK, got %v", err)
	}
	if resp.AgentId != "dup-01" {
		t.Errorf("agent_id = %q, want %q", resp.AgentId, "dup-01")
	}

	def, err := client.GetAgent(context.Background(), &zynaxv1.GetAgentRequest{AgentId: "dup-01"})
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if def.Endpoint != "new-host:9100" {
		t.Errorf("endpoint = %q, want upserted endpoint", def.Endpoint)
	}
}

func TestRegisterAgent_InvalidArgument(t *testing.T) {
	client := newTestClient(t)

	_, err := client.RegisterAgent(context.Background(), &zynaxv1.RegisterAgentRequest{
		Agent: &zynaxv1.AgentDef{AgentId: "", Endpoint: "h:1"},
	})
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", code)
	}
}

// ── DeregisterAgent ───────────────────────────────────────────────────────────

func TestDeregisterAgent_HappyPath(t *testing.T) {
	client := newTestClient(t)

	if _, err := client.RegisterAgent(context.Background(), &zynaxv1.RegisterAgentRequest{
		Agent: validAgentDef("dereg-01", "search"),
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	resp, err := client.DeregisterAgent(context.Background(), &zynaxv1.DeregisterAgentRequest{
		AgentId: "dereg-01",
	})
	if err != nil {
		t.Fatalf("DeregisterAgent: %v", err)
	}
	if resp.DeregisteredAt == nil {
		t.Error("want non-nil deregistered_at")
	}
}

func TestDeregisterAgent_NotFound(t *testing.T) {
	client := newTestClient(t)

	_, err := client.DeregisterAgent(context.Background(), &zynaxv1.DeregisterAgentRequest{
		AgentId: "ghost-99",
	})
	if code := status.Code(err); code != codes.NotFound {
		t.Errorf("want NotFound, got %v", code)
	}
}

// ── GetAgent ─────────────────────────────────────────────────────────────────

func TestGetAgent_HappyPath(t *testing.T) {
	client := newTestClient(t)

	if _, err := client.RegisterAgent(context.Background(), &zynaxv1.RegisterAgentRequest{
		Agent: validAgentDef("get-01", "write", "search"),
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	def, err := client.GetAgent(context.Background(), &zynaxv1.GetAgentRequest{AgentId: "get-01"})
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if def.AgentId != "get-01" || len(def.Capabilities) != 2 {
		t.Errorf("got agent_id=%q caps=%d", def.AgentId, len(def.Capabilities))
	}
	if def.Status != zynaxv1.AgentStatus_AGENT_STATUS_REGISTERED {
		t.Errorf("status = %v, want REGISTERED", def.Status)
	}
}

func TestGetAgent_NotFound(t *testing.T) {
	client := newTestClient(t)

	_, err := client.GetAgent(context.Background(), &zynaxv1.GetAgentRequest{AgentId: "no-such-agent"})
	if code := status.Code(err); code != codes.NotFound {
		t.Errorf("want NotFound, got %v", code)
	}
}

// ── ListAgents ────────────────────────────────────────────────────────────────

func TestListAgents_Empty(t *testing.T) {
	client := newTestClient(t)

	resp, err := client.ListAgents(context.Background(), &zynaxv1.ListAgentsRequest{})
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(resp.Agents) != 0 {
		t.Errorf("want 0 agents, got %d", len(resp.Agents))
	}
}

func TestListAgents_ExcludesDeregistered(t *testing.T) {
	client := newTestClient(t)

	for _, id := range []string{"la-01", "la-02"} {
		if _, err := client.RegisterAgent(context.Background(), &zynaxv1.RegisterAgentRequest{
			Agent: validAgentDef(id, "summarize"),
		}); err != nil {
			t.Fatalf("RegisterAgent %s: %v", id, err)
		}
	}
	if _, err := client.DeregisterAgent(context.Background(), &zynaxv1.DeregisterAgentRequest{
		AgentId: "la-02",
	}); err != nil {
		t.Fatalf("DeregisterAgent: %v", err)
	}

	resp, err := client.ListAgents(context.Background(), &zynaxv1.ListAgentsRequest{})
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(resp.Agents) != 1 {
		t.Errorf("want 1 active agent, got %d", len(resp.Agents))
	}
}

// ── FindByCapability ──────────────────────────────────────────────────────────

func TestFindByCapability_HappyPath(t *testing.T) {
	client := newTestClient(t)

	for _, id := range []string{"fc-01", "fc-02"} {
		if _, err := client.RegisterAgent(context.Background(), &zynaxv1.RegisterAgentRequest{
			Agent: validAgentDef(id, "summarize"),
		}); err != nil {
			t.Fatalf("RegisterAgent %s: %v", id, err)
		}
	}

	resp, err := client.FindByCapability(context.Background(), &zynaxv1.FindByCapabilityRequest{
		CapabilityName: "summarize",
	})
	if err != nil {
		t.Fatalf("FindByCapability: %v", err)
	}
	if len(resp.Agents) != 2 {
		t.Errorf("want 2 agents, got %d", len(resp.Agents))
	}
}

func TestFindByCapability_Empty(t *testing.T) {
	client := newTestClient(t)

	resp, err := client.FindByCapability(context.Background(), &zynaxv1.FindByCapabilityRequest{
		CapabilityName: "no-such-cap",
	})
	if err != nil {
		t.Fatalf("FindByCapability: %v", err)
	}
	if len(resp.Agents) != 0 {
		t.Errorf("want 0 agents, got %d", len(resp.Agents))
	}
}

func TestFindByCapability_InvalidArgument(t *testing.T) {
	client := newTestClient(t)

	_, err := client.FindByCapability(context.Background(), &zynaxv1.FindByCapabilityRequest{
		CapabilityName: "",
	})
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", code)
	}
}

func TestFindByCapability_ExcludesDeregistered(t *testing.T) {
	client := newTestClient(t)

	if _, err := client.RegisterAgent(context.Background(), &zynaxv1.RegisterAgentRequest{
		Agent: validAgentDef("fc-active", "search"),
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}
	if _, err := client.RegisterAgent(context.Background(), &zynaxv1.RegisterAgentRequest{
		Agent: validAgentDef("fc-dereg", "search"),
	}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}
	if _, err := client.DeregisterAgent(context.Background(), &zynaxv1.DeregisterAgentRequest{
		AgentId: "fc-dereg",
	}); err != nil {
		t.Fatalf("DeregisterAgent: %v", err)
	}

	resp, err := client.FindByCapability(context.Background(), &zynaxv1.FindByCapabilityRequest{
		CapabilityName: "search",
	})
	if err != nil {
		t.Fatalf("FindByCapability: %v", err)
	}
	if len(resp.Agents) != 1 {
		t.Errorf("want 1 registered agent, got %d", len(resp.Agents))
	}
}
