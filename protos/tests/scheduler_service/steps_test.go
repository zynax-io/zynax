// SPDX-License-Identifier: Apache-2.0
// BDD contract tests for SchedulerService.
//
// The stub below is a REFERENCE implementation of the contract invariants in
// zynax/v1/scheduler.proto (ADR-039). It models the ordered selection
// pipeline — capability match → hard constraints → readiness → expert
// target → scoring mode — over an in-memory view, standing in for the
// production informer-backed scheduler until it lands.
package scheduler_service_test

import (
	"context"
	"fmt"
	"sort"
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
)

// ─── Stub server ─────────────────────────────────────────────────────────────

// schedAgent is the scheduler's view of one agent: the public AgentDef plus
// the scheduling attributes that the production scheduler reads from the
// Agent custom resource (readiness, scoring hints, expert scope).
type schedAgent struct {
	def         *zynaxv1.AgentDef
	ready       bool
	gpu         bool
	language    string
	model       string
	tags        []string
	protocols   []string
	expertScope string
}

type schedulerStub struct {
	zynaxv1.UnimplementedSchedulerServiceServer
	mu        sync.Mutex
	agents    map[string]*schedAgent
	metricsUp bool
}

func newSchedulerStub() *schedulerStub {
	return &schedulerStub{agents: make(map[string]*schedAgent), metricsUp: true}
}

func hasAll(declared, required []string) bool {
	set := make(map[string]bool, len(declared))
	for _, d := range declared {
		set[d] = true
	}
	for _, r := range required {
		if !set[r] {
			return false
		}
	}
	return true
}

// matchesConstraints returns the name of the first eliminating filter, or ""
// when the agent satisfies every populated constraint (contract invariant 4).
func matchesConstraints(a *schedAgent, c *zynaxv1.SelectionConstraints) string {
	if c == nil {
		return ""
	}
	if len(c.Tags) > 0 && !hasAll(a.tags, c.Tags) {
		return "tags"
	}
	if c.Language != "" && a.language != c.Language {
		return "language"
	}
	if c.Model != "" && a.model != c.Model {
		return "model"
	}
	if c.RequireGpu && !a.gpu {
		return "gpu"
	}
	if len(c.Protocols) > 0 && !hasAll(a.protocols, c.Protocols) {
		return "protocols"
	}
	return ""
}

// matchCapability is pipeline stage 1: agents declaring the capability.
func (s *schedulerStub) matchCapability(name string) []*schedAgent {
	var matched []*schedAgent
	for _, a := range s.agents {
		for _, capDef := range a.def.Capabilities {
			if capDef.Name == name {
				matched = append(matched, a)
				break
			}
		}
	}
	return matched
}

// applyConstraints is pipeline stage 2. It returns the surviving candidates
// and the name of the first eliminating filter (contract invariant 4).
func applyConstraints(matched []*schedAgent, c *zynaxv1.SelectionConstraints) ([]*schedAgent, string) {
	var out []*schedAgent
	eliminatedBy := ""
	for _, a := range matched {
		if f := matchesConstraints(a, c); f != "" {
			eliminatedBy = f
			continue
		}
		out = append(out, a)
	}
	return out, eliminatedBy
}

// filterReady is pipeline stage 3 — the stale-liveness fix.
func filterReady(candidates []*schedAgent) []*schedAgent {
	var out []*schedAgent
	for _, a := range candidates {
		if a.ready {
			out = append(out, a)
		}
	}
	return out
}

// filterExpert is pipeline stage 4 — strict, no fallback (ADR-028).
func filterExpert(candidates []*schedAgent, target string) []*schedAgent {
	if target == "" {
		return candidates
	}
	var out []*schedAgent
	for _, a := range candidates {
		if a.expertScope == target {
			out = append(out, a)
		}
	}
	return out
}

// pickMode resolves pipeline stages 5–7 into the reported scoring mode.
func pickMode(policy zynaxv1.SelectionPolicy, metricsUp bool) (zynaxv1.SelectionMode, []string) {
	switch {
	case policy == zynaxv1.SelectionPolicy_SELECTION_POLICY_ROUND_ROBIN:
		return zynaxv1.SelectionMode_SELECTION_MODE_ROUND_ROBIN, []string{"rotation"}
	case !metricsUp:
		return zynaxv1.SelectionMode_SELECTION_MODE_DEGRADED_ROUND_ROBIN, []string{"readiness", "rotation"}
	default:
		return zynaxv1.SelectionMode_SELECTION_MODE_METRICS_WEIGHTED, []string{"load", "latency"}
	}
}

func (s *schedulerStub) SelectAgent(_ context.Context, req *zynaxv1.SelectAgentRequest) (*zynaxv1.SelectAgentResponse, error) {
	if req.GetCapabilityName() == "" {
		return nil, status.Error(codes.InvalidArgument, "capability_name must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	matched := s.matchCapability(req.GetCapabilityName())
	if len(matched) == 0 {
		return nil, status.Errorf(codes.NotFound, "no agent declares capability %q", req.GetCapabilityName())
	}

	afterConstraints, eliminatedBy := applyConstraints(matched, req.GetConstraints())
	if len(afterConstraints) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition,
			"no candidate for %q satisfies constraint: %s", req.GetCapabilityName(), eliminatedBy)
	}

	ready := filterReady(afterConstraints)
	if len(ready) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition,
			"no ready candidate for %q", req.GetCapabilityName())
	}

	afterExpert := filterExpert(ready, req.GetExpertTarget())
	if len(afterExpert) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition,
			"no ready candidate declares expert scope %q", req.GetExpertTarget())
	}

	// The reference stub picks deterministically (lowest agent_id) within
	// the mode's semantics.
	mode, factors := pickMode(req.GetPolicy(), s.metricsUp)
	sort.Slice(afterExpert, func(i, j int) bool {
		return afterExpert[i].def.AgentId < afterExpert[j].def.AgentId
	})
	winner := afterExpert[0]

	return &zynaxv1.SelectAgentResponse{
		Agent: winner.def,
		Rationale: &zynaxv1.SelectionRationale{
			CandidatesMatched:           int32(len(matched)),          //nolint:gosec // bounded by test input
			CandidatesAfterConstraints:  int32(len(afterConstraints)), //nolint:gosec // bounded by test input
			CandidatesReady:             int32(len(ready)),            //nolint:gosec // bounded by test input
			CandidatesAfterExpertFilter: int32(len(afterExpert)),      //nolint:gosec // bounded by test input
			Mode:                        mode,
			WinningFactors:              factors,
			Summary:                     fmt.Sprintf("selected %s for %s", winner.def.AgentId, req.GetCapabilityName()),
		},
	}, nil
}

// ─── Test context ────────────────────────────────────────────────────────────

type testCtx struct {
	stub    *schedulerStub
	client  zynaxv1.SchedulerServiceClient
	resp    *zynaxv1.SelectAgentResponse
	grpcErr error
}

func (tc *testCtx) addAgent(agentID, capName string, ready bool) *schedAgent {
	a := &schedAgent{
		def: &zynaxv1.AgentDef{
			AgentId:  agentID,
			Name:     agentID,
			Endpoint: "localhost:50061",
			Capabilities: []*zynaxv1.CapabilityDef{
				{Name: capName, Description: "test capability"},
			},
			Status: zynaxv1.AgentStatus_AGENT_STATUS_REGISTERED,
		},
		ready: ready,
	}
	tc.stub.mu.Lock()
	defer tc.stub.mu.Unlock()
	tc.stub.agents[agentID] = a
	return a
}

func (tc *testCtx) call(req *zynaxv1.SelectAgentRequest) {
	callCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tc.resp, tc.grpcErr = tc.client.SelectAgent(callCtx, req)
}

func codeFromName(name string) (codes.Code, error) {
	switch name {
	case "INVALID_ARGUMENT":
		return codes.InvalidArgument, nil
	case "NOT_FOUND":
		return codes.NotFound, nil
	case "FAILED_PRECONDITION":
		return codes.FailedPrecondition, nil
	default:
		return codes.Unknown, fmt.Errorf("unknown gRPC code name %q", name)
	}
}

func modeFromSuffix(suffix string) (zynaxv1.SelectionMode, error) {
	full := "SELECTION_MODE_" + suffix
	v, ok := zynaxv1.SelectionMode_value[full]
	if !ok {
		return zynaxv1.SelectionMode_SELECTION_MODE_UNSPECIFIED, fmt.Errorf("unknown SelectionMode %q", full)
	}
	return zynaxv1.SelectionMode(v), nil
}

// ─── Step registration (split by Gherkin keyword to keep functions small) ───

func registerGivenSteps(t *testing.T, sc *godog.ScenarioContext, tc *testCtx) {
	t.Helper()

	sc.Step(`^a SchedulerService is running on a test gRPC server$`, func(ctx context.Context) (context.Context, error) {
		tc.stub = newSchedulerStub()
		srv, dialer := testserver.NewBufconnServer(t)
		zynaxv1.RegisterSchedulerServiceServer(srv, tc.stub)
		conn, err := grpc.NewClient("passthrough:///bufnet",
			grpc.WithContextDialer(dialer),
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return ctx, fmt.Errorf("dial bufconn: %w", err)
		}
		tc.client = zynaxv1.NewSchedulerServiceClient(conn)
		return ctx, nil
	})

	sc.Step(`^the scheduler view is empty$`, func(ctx context.Context) (context.Context, error) {
		tc.stub.mu.Lock()
		defer tc.stub.mu.Unlock()
		tc.stub.agents = make(map[string]*schedAgent)
		tc.stub.metricsUp = true
		tc.resp = nil
		tc.grpcErr = nil
		return ctx, nil
	})

	sc.Step(`^a ready agent "([^"]*)" declaring capability "([^"]*)"$`, func(ctx context.Context, id, capName string) (context.Context, error) {
		tc.addAgent(id, capName, true)
		return ctx, nil
	})

	sc.Step(`^a not-ready agent "([^"]*)" declaring capability "([^"]*)"$`, func(ctx context.Context, id, capName string) (context.Context, error) {
		tc.addAgent(id, capName, false)
		return ctx, nil
	})

	sc.Step(`^a ready agent "([^"]*)" declaring capability "([^"]*)" with expert scope "([^"]*)"$`, func(ctx context.Context, id, capName, scope string) (context.Context, error) {
		a := tc.addAgent(id, capName, true)
		tc.stub.mu.Lock()
		defer tc.stub.mu.Unlock()
		a.expertScope = scope
		return ctx, nil
	})

	sc.Step(`^agent "([^"]*)" declares no gpu$`, func(ctx context.Context, id string) (context.Context, error) {
		tc.stub.mu.Lock()
		defer tc.stub.mu.Unlock()
		a, ok := tc.stub.agents[id]
		if !ok {
			return ctx, fmt.Errorf("unknown agent %q", id)
		}
		a.gpu = false
		return ctx, nil
	})

	sc.Step(`^the metrics backend is unavailable$`, func(ctx context.Context) (context.Context, error) {
		tc.stub.mu.Lock()
		defer tc.stub.mu.Unlock()
		tc.stub.metricsUp = false
		return ctx, nil
	})
}

func registerWhenSteps(sc *godog.ScenarioContext, tc *testCtx) {
	sc.Step(`^SelectAgent is called with capability_name "([^"]*)"$`, func(ctx context.Context, capName string) (context.Context, error) {
		tc.call(&zynaxv1.SelectAgentRequest{CapabilityName: capName})
		return ctx, nil
	})

	sc.Step(`^SelectAgent is called with capability_name "([^"]*)" requiring gpu$`, func(ctx context.Context, capName string) (context.Context, error) {
		tc.call(&zynaxv1.SelectAgentRequest{
			CapabilityName: capName,
			Constraints:    &zynaxv1.SelectionConstraints{RequireGpu: true},
		})
		return ctx, nil
	})

	sc.Step(`^SelectAgent is called with capability_name "([^"]*)" and expert_target "([^"]*)"$`, func(ctx context.Context, capName, target string) (context.Context, error) {
		tc.call(&zynaxv1.SelectAgentRequest{CapabilityName: capName, ExpertTarget: target})
		return ctx, nil
	})

	sc.Step(`^SelectAgent is called with capability_name "([^"]*)" and policy SELECTION_POLICY_ROUND_ROBIN$`, func(ctx context.Context, capName string) (context.Context, error) {
		tc.call(&zynaxv1.SelectAgentRequest{
			CapabilityName: capName,
			Policy:         zynaxv1.SelectionPolicy_SELECTION_POLICY_ROUND_ROBIN,
		})
		return ctx, nil
	})
}

func registerThenSteps(sc *godog.ScenarioContext, tc *testCtx) {
	sc.Step(`^the response contains exactly one agent$`, func(ctx context.Context) (context.Context, error) {
		if tc.grpcErr != nil {
			return ctx, fmt.Errorf("unexpected error: %w", tc.grpcErr)
		}
		if tc.resp.GetAgent() == nil {
			return ctx, fmt.Errorf("expected an agent in the response, got none")
		}
		return ctx, nil
	})

	sc.Step(`^the selected agent is "([^"]*)"$`, func(ctx context.Context, id string) (context.Context, error) {
		if tc.grpcErr != nil {
			return ctx, fmt.Errorf("unexpected error: %w", tc.grpcErr)
		}
		if got := tc.resp.GetAgent().GetAgentId(); got != id {
			return ctx, fmt.Errorf("expected agent %q, got %q", id, got)
		}
		return ctx, nil
	})

	sc.Step(`^the selected agent endpoint is not empty$`, func(ctx context.Context) (context.Context, error) {
		if tc.resp.GetAgent().GetEndpoint() == "" {
			return ctx, fmt.Errorf("expected non-empty endpoint")
		}
		return ctx, nil
	})

	sc.Step(`^the rationale reports (\d+) candidates matched$`, func(ctx context.Context, n int) (context.Context, error) {
		if got := int(tc.resp.GetRationale().GetCandidatesMatched()); got != n {
			return ctx, fmt.Errorf("expected %d candidates matched, got %d", n, got)
		}
		return ctx, nil
	})

	sc.Step(`^the rationale reports (\d+) candidates ready$`, func(ctx context.Context, n int) (context.Context, error) {
		if got := int(tc.resp.GetRationale().GetCandidatesReady()); got != n {
			return ctx, fmt.Errorf("expected %d candidates ready, got %d", n, got)
		}
		return ctx, nil
	})

	sc.Step(`^the rationale reports (\d+) candidates after expert filter$`, func(ctx context.Context, n int) (context.Context, error) {
		if got := int(tc.resp.GetRationale().GetCandidatesAfterExpertFilter()); got != n {
			return ctx, fmt.Errorf("expected %d candidates after expert filter, got %d", n, got)
		}
		return ctx, nil
	})

	sc.Step(`^the rationale mode is SELECTION_MODE_([A-Z_]+)$`, func(ctx context.Context, suffix string) (context.Context, error) {
		want, err := modeFromSuffix(suffix)
		if err != nil {
			return ctx, err
		}
		if got := tc.resp.GetRationale().GetMode(); got != want {
			return ctx, fmt.Errorf("expected mode %v, got %v", want, got)
		}
		return ctx, nil
	})

	sc.Step(`^the rationale winning_factors are not empty$`, func(ctx context.Context) (context.Context, error) {
		if len(tc.resp.GetRationale().GetWinningFactors()) == 0 {
			return ctx, fmt.Errorf("expected non-empty winning_factors")
		}
		return ctx, nil
	})

}

func registerErrorSteps(sc *godog.ScenarioContext, tc *testCtx) {
	sc.Step(`^the call fails with ([A-Z_]+)$`, func(ctx context.Context, codeName string) (context.Context, error) {
		want, err := codeFromName(codeName)
		if err != nil {
			return ctx, err
		}
		if tc.grpcErr == nil {
			return ctx, fmt.Errorf("expected error with code %v, got success", want)
		}
		if got := status.Code(tc.grpcErr); got != want {
			return ctx, fmt.Errorf("expected code %v, got %v: %w", want, got, tc.grpcErr)
		}
		return ctx, nil
	})

	sc.Step(`^the error message mentions "([^"]*)"$`, func(ctx context.Context, needle string) (context.Context, error) {
		if tc.grpcErr == nil {
			return ctx, fmt.Errorf("expected an error mentioning %q, got success", needle)
		}
		if !strings.Contains(tc.grpcErr.Error(), needle) {
			return ctx, fmt.Errorf("error %q does not mention %q", tc.grpcErr.Error(), needle)
		}
		return ctx, nil
	})
}

// ─── Suite ───────────────────────────────────────────────────────────────────

func TestFeatures(t *testing.T) {
	tc := &testCtx{}

	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			registerGivenSteps(t, sc, tc)
			registerWhenSteps(sc, tc)
			registerThenSteps(sc, tc)
			registerErrorSteps(sc, tc)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features/scheduler_service.feature"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
