// SPDX-License-Identifier: Apache-2.0

package domain_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/zynax-io/zynax/services/agent-registry/internal/domain"
)

// ── fakeRepo ──────────────────────────────────────────────────────────────────

type fakeRepo struct {
	mu     sync.Mutex
	agents map[string]domain.Agent
}

func newFakeRepo() *fakeRepo { return &fakeRepo{agents: make(map[string]domain.Agent)} }

func (r *fakeRepo) Save(_ context.Context, agent domain.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[agent.ID] = agent
	return nil
}

func (r *fakeRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.agents[id]; !ok {
		return fmt.Errorf("%w: %q", domain.ErrAgentNotFound, id)
	}
	delete(r.agents, id)
	return nil
}

func (r *fakeRepo) FindByID(_ context.Context, id string) (domain.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.agents[id]
	if !ok {
		return domain.Agent{}, fmt.Errorf("%w: %q", domain.ErrAgentNotFound, id)
	}
	return a, nil
}

func (r *fakeRepo) FindAll(_ context.Context) ([]domain.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.Agent, 0, len(r.agents))
	for _, a := range r.agents {
		out = append(out, a)
	}
	return out, nil
}

func (r *fakeRepo) FindByCapability(_ context.Context, name string) ([]domain.Agent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Agent
	for _, a := range r.agents {
		if a.Status != domain.AgentStatusRegistered {
			continue
		}
		for _, c := range a.Capabilities {
			if c.Name == name {
				out = append(out, a)
				break
			}
		}
	}
	return out, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func validAgent(id string, caps ...string) domain.Agent {
	if len(caps) == 0 {
		caps = []string{"summarize"}
	}
	cs := make([]domain.Capability, len(caps))
	for i, c := range caps {
		cs[i] = domain.Capability{Name: c}
	}
	return domain.Agent{
		ID:           id,
		Name:         "Test Agent",
		Endpoint:     "localhost:9000",
		Capabilities: cs,
	}
}

func seed(t *testing.T, svc *domain.AgentRegistryService, agent domain.Agent) domain.Agent {
	t.Helper()
	a, err := svc.Register(context.Background(), agent)
	if err != nil {
		t.Fatalf("seed Register: %v", err)
	}
	return a
}

// ── Register ─────────────────────────────────────────────────────────────────

func TestRegister_HappyPath(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	a, err := svc.Register(context.Background(), validAgent("agent-01"))
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if a.ID != "agent-01" || a.Status != domain.AgentStatusRegistered {
		t.Errorf("got id=%q status=%s", a.ID, a.Status)
	}
	if a.RegisteredAt.IsZero() || a.UpdatedAt.IsZero() {
		t.Errorf("timestamps not set: registeredAt=%v updatedAt=%v", a.RegisteredAt, a.UpdatedAt)
	}
}

func TestRegister_Validation(t *testing.T) {
	cases := []struct {
		name    string
		agent   domain.Agent
		wantErr error
	}{
		{"empty_id", domain.Agent{Endpoint: "h:1", Capabilities: []domain.Capability{{Name: "x"}}}, domain.ErrInvalidArgument},
		{"empty_endpoint", domain.Agent{ID: "a", Capabilities: []domain.Capability{{Name: "x"}}}, domain.ErrInvalidArgument},
		{"no_capabilities", domain.Agent{ID: "a", Endpoint: "h:1"}, domain.ErrInvalidArgument},
		{"invalid_cap_name_upper", domain.Agent{ID: "a", Endpoint: "h:1", Capabilities: []domain.Capability{{Name: "Invalid"}}}, domain.ErrInvalidArgument},
		{"invalid_cap_name_space", domain.Agent{ID: "a", Endpoint: "h:1", Capabilities: []domain.Capability{{Name: "bad cap"}}}, domain.ErrInvalidArgument},
		{"invalid_cap_name_empty", domain.Agent{ID: "a", Endpoint: "h:1", Capabilities: []domain.Capability{{Name: ""}}}, domain.ErrInvalidArgument},
		{"duplicate_cap_names", domain.Agent{ID: "a", Endpoint: "h:1", Capabilities: []domain.Capability{{Name: "x"}, {Name: "x"}}}, domain.ErrInvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := domain.NewAgentRegistryService(newFakeRepo())
			_, err := svc.Register(context.Background(), tc.agent)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestRegister_TooManyCapabilities(t *testing.T) {
	caps := make([]domain.Capability, 51)
	for i := range caps {
		caps[i] = domain.Capability{Name: fmt.Sprintf("cap%02d", i)}
	}
	a := domain.Agent{ID: "a", Endpoint: "h:1", Capabilities: caps}
	svc := domain.NewAgentRegistryService(newFakeRepo())
	_, err := svc.Register(context.Background(), a)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Errorf("err = %v, want ErrInvalidArgument", err)
	}
}

func TestRegister_AlreadyExists(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	seed(t, svc, validAgent("dup-01"))

	_, err := svc.Register(context.Background(), validAgent("dup-01"))
	if !errors.Is(err, domain.ErrAgentAlreadyExists) {
		t.Errorf("err = %v, want ErrAgentAlreadyExists", err)
	}
}

func TestRegister_ReregistrationAfterDeregister(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	seed(t, svc, validAgent("rereg-01"))
	if _, err := svc.Deregister(context.Background(), "rereg-01"); err != nil {
		t.Fatalf("Deregister: %v", err)
	}

	// Re-registration of a DEREGISTERED agent must succeed.
	a, err := svc.Register(context.Background(), validAgent("rereg-01"))
	if err != nil {
		t.Fatalf("re-Register: %v", err)
	}
	if a.Status != domain.AgentStatusRegistered {
		t.Errorf("status = %s, want REGISTERED", a.Status)
	}
}

// ── Deregister ────────────────────────────────────────────────────────────────

func TestDeregister_HappyPath(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	seed(t, svc, validAgent("dereg-01"))

	deregisteredAt, err := svc.Deregister(context.Background(), "dereg-01")
	if err != nil || deregisteredAt.IsZero() {
		t.Fatalf("Deregister: err=%v deregisteredAt=%v", err, deregisteredAt)
	}

	a, err := svc.GetByID(context.Background(), "dereg-01")
	if err != nil {
		t.Fatalf("GetByID after deregister: %v", err)
	}
	if a.Status != domain.AgentStatusDeregistered {
		t.Errorf("status = %s, want DEREGISTERED", a.Status)
	}
}

func TestDeregister_NotFound(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	_, err := svc.Deregister(context.Background(), "ghost-99")
	if !errors.Is(err, domain.ErrAgentNotFound) {
		t.Errorf("err = %v, want ErrAgentNotFound", err)
	}
}

func TestDeregister_EmptyID(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	_, err := svc.Deregister(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Errorf("err = %v, want ErrInvalidArgument", err)
	}
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestGetByID_HappyPath(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	seed(t, svc, validAgent("get-01", "summarize", "search"))

	a, err := svc.GetByID(context.Background(), "get-01")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if a.ID != "get-01" || len(a.Capabilities) != 2 {
		t.Errorf("got id=%q caps=%d", a.ID, len(a.Capabilities))
	}
}

func TestGetByID_DeregisteredAgentIsRetrievable(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	seed(t, svc, validAgent("audit-01"))
	svc.Deregister(context.Background(), "audit-01") //nolint:errcheck,gosec

	a, err := svc.GetByID(context.Background(), "audit-01")
	if err != nil {
		t.Fatalf("GetByID on deregistered agent: %v", err)
	}
	if a.Status != domain.AgentStatusDeregistered {
		t.Errorf("status = %s, want DEREGISTERED", a.Status)
	}
}

func TestGetByID_Errors(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	cases := []struct {
		name    string
		id      string
		wantErr error
	}{
		{"empty_id", "", domain.ErrInvalidArgument},
		{"unknown_id", "nonexistent", domain.ErrAgentNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.GetByID(context.Background(), tc.id)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// ── FindByCapability ──────────────────────────────────────────────────────────

func TestFindByCapability_ReturnsOnlyRegistered(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	seed(t, svc, validAgent("cap-01", "summarize"))
	seed(t, svc, validAgent("cap-02", "summarize"))
	a3 := seed(t, svc, validAgent("cap-03", "summarize"))
	svc.Deregister(context.Background(), a3.ID) //nolint:errcheck,gosec

	agents, err := svc.FindByCapability(context.Background(), "summarize")
	if err != nil {
		t.Fatalf("FindByCapability: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("want 2 registered agents, got %d", len(agents))
	}
}

func TestFindByCapability_EmptyResult(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	agents, err := svc.FindByCapability(context.Background(), "write")
	if err != nil {
		t.Fatalf("FindByCapability: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("want empty, got %d", len(agents))
	}
}

func TestFindByCapability_EmptyName(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	_, err := svc.FindByCapability(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Errorf("err = %v, want ErrInvalidArgument", err)
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestList_NoFilter(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	seed(t, svc, validAgent("la"))
	seed(t, svc, validAgent("lb"))

	res, err := svc.List(context.Background(), domain.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Agents) != 2 {
		t.Errorf("want 2, got %d", len(res.Agents))
	}
}

func TestList_ExcludesDeregisteredByDefault(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	seed(t, svc, validAgent("l-active"))
	a := seed(t, svc, validAgent("l-dereg"))
	svc.Deregister(context.Background(), a.ID) //nolint:errcheck,gosec

	res, err := svc.List(context.Background(), domain.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Agents) != 1 {
		t.Errorf("want 1 active agent, got %d", len(res.Agents))
	}
}

func TestList_IncludeDeregistered(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	seed(t, svc, validAgent("ld-active"))
	a := seed(t, svc, validAgent("ld-dereg"))
	svc.Deregister(context.Background(), a.ID) //nolint:errcheck,gosec

	res, err := svc.List(context.Background(), domain.ListFilter{IncludeDeregistered: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Agents) != 2 {
		t.Errorf("want 2 agents, got %d", len(res.Agents))
	}
}

func TestList_LabelSelector(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	a1 := validAgent("ls-prod")
	a1.Labels = map[string]string{"env": "production", "tier": "standard"}
	a2 := validAgent("ls-dev")
	a2.Labels = map[string]string{"env": "development"}
	seed(t, svc, a1)
	seed(t, svc, a2)

	res, err := svc.List(context.Background(), domain.ListFilter{LabelSelector: "env=production"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Agents) != 1 || res.Agents[0].ID != "ls-prod" {
		t.Errorf("label filter: got %d agents", len(res.Agents))
	}
}

func TestList_MultiLabelSelector(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	a1 := validAgent("ml-01")
	a1.Labels = map[string]string{"env": "production", "tier": "standard"}
	a2 := validAgent("ml-02")
	a2.Labels = map[string]string{"env": "production", "tier": "premium"}
	seed(t, svc, a1)
	seed(t, svc, a2)

	res, err := svc.List(context.Background(), domain.ListFilter{LabelSelector: "env=production,tier=standard"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Agents) != 1 || res.Agents[0].ID != "ml-01" {
		t.Errorf("multi-label filter: got %d agents", len(res.Agents))
	}
}

// ── Runtime expert (kind: AgentDef) ─────────────────────────────────────────────

// expertAgent builds the domain Agent that a runtime-expert AgentDef manifest
// (spec/workflows/examples/agent-def-expert.yaml) maps to: an ordinary agent whose
// capability is the expert's routing key and whose labels mark it as an expert.
func expertAgent(id string) domain.Agent {
	a := validAgent(id, "go_review")
	a.Labels = map[string]string{
		"team":                  "platform",
		"agent.zynax.io/kind":   "expert",
		"agent.zynax.io/expert": "go-review",
	}
	return a
}

// TestRuntimeExpert_RegistersAndIsDispatchable covers EPIC X step X.3 (#1203):
// a runtime expert registers in the registry, is discoverable by the task broker
// via its capability routing key, and is selectable by the expert kind label.
func TestRuntimeExpert_RegistersAndIsDispatchable(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())

	reg := seed(t, svc, expertAgent("go-review-expert"))
	if reg.Status != domain.AgentStatusRegistered {
		t.Fatalf("expert not registered: status=%s", reg.Status)
	}

	// Dispatchable: the broker resolves the agent endpoint by capability routing key.
	found, err := svc.FindByCapability(context.Background(), "go_review")
	if err != nil {
		t.Fatalf("FindByCapability: %v", err)
	}
	if len(found) != 1 || found[0].ID != "go-review-expert" {
		t.Fatalf("expert not dispatchable by capability: got %d agents", len(found))
	}
	if found[0].Endpoint == "" {
		t.Errorf("dispatch target has empty endpoint")
	}

	// Selectable as an expert via the kind label selector.
	res, err := svc.List(context.Background(), domain.ListFilter{LabelSelector: "agent.zynax.io/kind=expert"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Agents) != 1 || res.Agents[0].ID != "go-review-expert" {
		t.Errorf("expert not selectable by kind label: got %d agents", len(res.Agents))
	}
}

// TestRuntimeExpert_DeregisterStopsDispatch confirms a deregistered expert is no
// longer routed to, so a retired expert cannot receive new workflow dispatches.
func TestRuntimeExpert_DeregisterStopsDispatch(t *testing.T) {
	svc := domain.NewAgentRegistryService(newFakeRepo())
	seed(t, svc, expertAgent("go-review-expert"))

	if _, err := svc.Deregister(context.Background(), "go-review-expert"); err != nil {
		t.Fatalf("Deregister: %v", err)
	}

	found, err := svc.FindByCapability(context.Background(), "go_review")
	if err != nil {
		t.Fatalf("FindByCapability: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("deregistered expert still dispatchable: got %d agents", len(found))
	}
}

// ── AgentStatus ───────────────────────────────────────────────────────────────

func TestAgentStatus_String(t *testing.T) {
	cases := map[domain.AgentStatus]string{
		domain.AgentStatusUnspecified:  "UNSPECIFIED",
		domain.AgentStatusRegistered:   "REGISTERED",
		domain.AgentStatusDeregistered: "DEREGISTERED",
	}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Errorf("AgentStatus(%d).String() = %q, want %q", s, got, want)
		}
	}
}
