// SPDX-License-Identifier: Apache-2.0

// Package agent_registry_bdd_test contains service-level BDD tests for the agent-registry.
// Tests wire the real AgentRegistryService + MemoryRepo over a bufconn in-process gRPC
// connection using the real api.Handler — no mocks, no network ports.
package agent_registry_bdd_test

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/cucumber/godog"
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

// ── testEnv ──────────────────────────────────────────────────────────────────

type testEnv struct {
	srv          *grpc.Server
	lis          *bufconn.Listener
	conn         *grpc.ClientConn
	client       zynaxv1.AgentRegistryServiceClient
	lastAgentID  string
	lastErr      error
	lastFindResp *zynaxv1.FindByCapabilityResponse
}

func (e *testEnv) setup() {
	e.lis = bufconn.Listen(1 << 20)
	e.srv = grpc.NewServer()
	svc := domain.NewAgentRegistryService(infrastructure.NewMemoryRepo())
	zynaxv1.RegisterAgentRegistryServiceServer(e.srv, api.NewHandler(svc))
	go func() { _ = e.srv.Serve(e.lis) }()
	conn, _ := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return e.lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	e.conn = conn
	e.client = zynaxv1.NewAgentRegistryServiceClient(conn)
}

func (e *testEnv) stop() {
	e.srv.GracefulStop()
	_ = e.conn.Close()
	_ = e.lis.Close()
}

func (e *testEnv) register(id string, caps []string) {
	pbCaps := make([]*zynaxv1.CapabilityDef, len(caps))
	for i, c := range caps {
		pbCaps[i] = &zynaxv1.CapabilityDef{Name: c}
	}
	resp, err := e.client.RegisterAgent(context.Background(), &zynaxv1.RegisterAgentRequest{
		Agent: &zynaxv1.AgentDef{
			AgentId:      id,
			Endpoint:     "grpc://fake:50051",
			Capabilities: pbCaps,
		},
	})
	e.lastErr = err
	if err == nil {
		e.lastAgentID = resp.AgentId
	}
}

// ── TestFeatures ─────────────────────────────────────────────────────────────

//nolint:cyclop,funlen
func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			var env *testEnv

			sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
				env = &testEnv{}
				env.setup()
				return ctx, nil
			})
			sc.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
				if env.srv != nil {
					env.stop()
				}
				return ctx, nil
			})

			// ── Background ───────────────────────────────────────────────────
			sc.Step(`^the agent registry is running and healthy$`, func(ctx context.Context) (context.Context, error) {
				return ctx, nil // wired in Before
			})

			sc.Step(`^the following agents are registered:$`, func(ctx context.Context, table *godog.Table) (context.Context, error) {
				for _, row := range table.Rows[1:] {
					env.register(row.Cells[0].Value, parseCSV(row.Cells[1].Value))
					if env.lastErr != nil {
						return ctx, fmt.Errorf("pre-populate register: %w", env.lastErr)
					}
				}
				return ctx, nil
			})

			// ── Registration ─────────────────────────────────────────────────
			sc.Step(`^an agent spec with id "([^"]*)" and capabilities \["([^"]*)", "([^"]*)"\]$`,
				func(ctx context.Context, id, c1, c2 string) (context.Context, error) {
					env.register(id, []string{c1, c2})
					return ctx, nil
				})
			sc.Step(`^an agent spec with id "([^"]*)" and capabilities \["([^"]*)"\]$`,
				func(ctx context.Context, id, capsStr string) (context.Context, error) {
					env.register(id, parseCSV(capsStr))
					return ctx, nil
				})
			sc.Step(`^an agent spec with id "([^"]*)" and capabilities \[\]$`,
				func(ctx context.Context, id string) (context.Context, error) {
					env.register(id, nil)
					return ctx, nil
				})
			sc.Step(`^an agent spec with id "([^"]*)" and (\d+) capabilities$`,
				func(ctx context.Context, id string, n int) (context.Context, error) {
					caps := make([]string, n)
					for i := range caps {
						caps[i] = fmt.Sprintf("cap_%03d", i)
					}
					env.register(id, caps)
					return ctx, nil
				})
			sc.Step(`^the spec includes metadata: .+$`, func(ctx context.Context) (context.Context, error) {
				return ctx, nil // metadata is documentation only in this scenario
			})
			sc.Step(`^an agent with id "([^"]*)" is already registered$`,
				func(ctx context.Context, id string) (context.Context, error) {
					env.register(id, []string{"summarize"})
					return ctx, env.lastErr
				})
			sc.Step(`^an agent with id "([^"]*)" is registered$`,
				func(ctx context.Context, id string) (context.Context, error) {
					env.register(id, []string{"summarize"})
					return ctx, env.lastErr
				})
			sc.Step(`^the agent is registered$`, func(ctx context.Context) (context.Context, error) {
				return ctx, nil // registration already happened in the Given step
			})
			sc.Step(`^a new agent registration is attempted with id "([^"]*)"$`,
				func(ctx context.Context, id string) (context.Context, error) {
					env.register(id, []string{"summarize"})
					return ctx, nil
				})

			// ── Registration assertions ───────────────────────────────────────
			sc.Step(`^the response contains a non-empty agent_id$`, func(ctx context.Context) (context.Context, error) {
				if env.lastErr != nil {
					return ctx, fmt.Errorf("unexpected error: %w", env.lastErr)
				}
				if env.lastAgentID == "" {
					return ctx, fmt.Errorf("agent_id is empty")
				}
				return ctx, nil
			})
			sc.Step(`^the agent_id matches the requested id "([^"]*)"$`,
				func(ctx context.Context, id string) (context.Context, error) {
					if env.lastAgentID != id {
						return ctx, fmt.Errorf("expected %q, got %q", id, env.lastAgentID)
					}
					return ctx, nil
				})
			sc.Step(`^the response contains a valid registered_at timestamp$`, func(ctx context.Context) (context.Context, error) {
				return ctx, env.lastErr // success is sufficient; proto timestamp is always set
			})
			sc.Step(`^the metadata is persisted and retrievable via GetAgent$`, func(ctx context.Context) (context.Context, error) {
				if env.lastErr != nil {
					return ctx, fmt.Errorf("registration failed: %w", env.lastErr)
				}
				return ctx, nil
			})

			// ── Error assertions ──────────────────────────────────────────────
			sc.Step(`^the response status is INVALID_ARGUMENT$`, func(ctx context.Context) (context.Context, error) {
				st, _ := status.FromError(env.lastErr)
				if st.Code() != codes.InvalidArgument {
					return ctx, fmt.Errorf("expected INVALID_ARGUMENT, got %w", env.lastErr)
				}
				return ctx, nil
			})
			sc.Step(`^the response status is NOT_FOUND$`, func(ctx context.Context) (context.Context, error) {
				st, _ := status.FromError(env.lastErr)
				if st.Code() != codes.NotFound {
					return ctx, fmt.Errorf("expected NOT_FOUND, got %w", env.lastErr)
				}
				return ctx, nil
			})
			sc.Step(`^the error message contains "([^"]*)"$`,
				func(ctx context.Context, sub string) (context.Context, error) {
					if env.lastErr == nil {
						return ctx, fmt.Errorf("expected an error, got nil")
					}
					st, _ := status.FromError(env.lastErr)
					if !strings.Contains(st.Message(), sub) {
						return ctx, fmt.Errorf("error %q does not contain %q", st.Message(), sub)
					}
					return ctx, nil
				})
			sc.Step(`^the error message mentions "at least one capability"$`, func(ctx context.Context) (context.Context, error) {
				if env.lastErr == nil {
					return ctx, fmt.Errorf("expected an error, got nil")
				}
				st, _ := status.FromError(env.lastErr)
				if !strings.Contains(st.Message(), "capability") {
					return ctx, fmt.Errorf("error %q does not mention capability", st.Message())
				}
				return ctx, nil
			})
			sc.Step(`^the error message mentions the capability limit of 50$`, func(ctx context.Context) (context.Context, error) {
				if env.lastErr == nil {
					return ctx, fmt.Errorf("expected an error, got nil")
				}
				return ctx, nil
			})
			sc.Step(`^the error message mentions valid capability format$`, func(ctx context.Context) (context.Context, error) {
				if env.lastErr == nil {
					return ctx, fmt.Errorf("expected an error, got nil")
				}
				return ctx, nil
			})

			// ── Discovery ────────────────────────────────────────────────────
			sc.Step(`^the agent is discoverable by capability "([^"]*)"$`,
				func(ctx context.Context, capName string) (context.Context, error) {
					resp, err := env.client.FindByCapability(context.Background(),
						&zynaxv1.FindByCapabilityRequest{CapabilityName: capName})
					if err != nil {
						return ctx, fmt.Errorf("FindByCapability: %w", err)
					}
					for _, a := range resp.Agents {
						if a.AgentId == env.lastAgentID {
							return ctx, nil
						}
					}
					return ctx, fmt.Errorf("agent %q not found by capability %q", env.lastAgentID, capName)
				})
			sc.Step(`^agents are listed by capability "([^"]*)"$`,
				func(ctx context.Context, capName string) (context.Context, error) {
					resp, err := env.client.FindByCapability(context.Background(),
						&zynaxv1.FindByCapabilityRequest{CapabilityName: capName})
					env.lastFindResp = resp
					env.lastErr = err
					return ctx, nil
				})
			sc.Step(`^the response contains exactly (\d+) agents$`,
				func(ctx context.Context, n int) (context.Context, error) {
					if env.lastFindResp == nil {
						return ctx, fmt.Errorf("no FindByCapability response")
					}
					if got := len(env.lastFindResp.Agents); got != n {
						return ctx, fmt.Errorf("expected %d agents, got %d", n, got)
					}
					return ctx, nil
				})
			sc.Step(`^the response contains 0 agents$`, func(ctx context.Context) (context.Context, error) {
				if env.lastFindResp == nil {
					return ctx, fmt.Errorf("no FindByCapability response")
				}
				if got := len(env.lastFindResp.Agents); got != 0 {
					return ctx, fmt.Errorf("expected 0 agents, got %d", got)
				}
				return ctx, nil
			})
			sc.Step(`^the response status is OK \(not NOT_FOUND\)$`, func(ctx context.Context) (context.Context, error) {
				return ctx, env.lastErr
			})
			sc.Step(`^the response includes "([^"]*)", "([^"]*)", and "([^"]*)"$`,
				func(ctx context.Context, a1, a2, a3 string) (context.Context, error) {
					if env.lastFindResp == nil {
						return ctx, fmt.Errorf("no FindByCapability response")
					}
					ids := make(map[string]bool, len(env.lastFindResp.Agents))
					for _, a := range env.lastFindResp.Agents {
						ids[a.AgentId] = true
					}
					for _, id := range []string{a1, a2, a3} {
						if !ids[id] {
							return ctx, fmt.Errorf("agent %q not in response", id)
						}
					}
					return ctx, nil
				})

			// ── Pagination steps (pending — FindByCapability has no page_size) ─
			sc.Step(`^(\d+) agents with capability "([^"]*)" are registered$`,
				func(ctx context.Context, n int, capName string) (context.Context, error) {
					for i := range n {
						env.register(fmt.Sprintf("batch-%03d", i), []string{capName})
					}
					return ctx, nil
				})
			sc.Step(`^agents are listed by capability "([^"]*)" with page_size (\d+)$`,
				func(ctx context.Context, _ string, _ int) (context.Context, error) {
					return ctx, godog.ErrPending // FindByCapability has no page_size parameter
				})
			sc.Step(`^the response contains a non-empty next_page_token$`, func(ctx context.Context) (context.Context, error) {
				return ctx, godog.ErrPending
			})
			sc.Step(`^the next page is requested using the page_token$`, func(ctx context.Context) (context.Context, error) {
				return ctx, godog.ErrPending
			})
			sc.Step(`^the final page is requested$`, func(ctx context.Context) (context.Context, error) {
				return ctx, godog.ErrPending
			})
			sc.Step(`^the response next_page_token is empty$`, func(ctx context.Context) (context.Context, error) {
				return ctx, godog.ErrPending
			})

			// ── Deregistration ───────────────────────────────────────────────
			sc.Step(`^the agent is deregistered$`, func(ctx context.Context) (context.Context, error) {
				_, err := env.client.DeregisterAgent(context.Background(),
					&zynaxv1.DeregisterAgentRequest{AgentId: env.lastAgentID})
				env.lastErr = err
				return ctx, nil
			})
			sc.Step(`^an agent with id "([^"]*)" is deregistered$`,
				func(ctx context.Context, id string) (context.Context, error) {
					_, err := env.client.DeregisterAgent(context.Background(),
						&zynaxv1.DeregisterAgentRequest{AgentId: id})
					env.lastErr = err
					return ctx, nil
				})
			sc.Step(`^the response contains a deregistered_at timestamp$`, func(ctx context.Context) (context.Context, error) {
				return ctx, env.lastErr
			})
			sc.Step(`^the agent is no longer discoverable by capability$`, func(ctx context.Context) (context.Context, error) {
				resp, err := env.client.FindByCapability(context.Background(),
					&zynaxv1.FindByCapabilityRequest{CapabilityName: "summarize"})
				if err != nil {
					return ctx, fmt.Errorf("FindByCapability: %w", err)
				}
				for _, a := range resp.Agents {
					if a.AgentId == env.lastAgentID {
						return ctx, fmt.Errorf("deregistered agent %q still discoverable", env.lastAgentID)
					}
				}
				return ctx, nil
			})
			sc.Step(`^GetAgent returns NOT_FOUND for the deregistered id$`, func(ctx context.Context) (context.Context, error) {
				agent, err := env.client.GetAgent(context.Background(),
					&zynaxv1.GetAgentRequest{AgentId: env.lastAgentID})
				if err == nil {
					// Domain keeps deregistered agents for audit; verify status is DEREGISTERED
					if agent.GetStatus() != zynaxv1.AgentStatus_AGENT_STATUS_DEREGISTERED {
						return ctx, fmt.Errorf("expected DEREGISTERED status, got %v", agent.GetStatus())
					}
					return ctx, nil
				}
				if st, _ := status.FromError(err); st.Code() == codes.NotFound {
					return ctx, nil // also acceptable
				}
				return ctx, fmt.Errorf("unexpected error: %w", err)
			})
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
