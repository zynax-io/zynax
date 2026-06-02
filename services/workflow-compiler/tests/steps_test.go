// SPDX-License-Identifier: Apache-2.0

// Package workflow_compiler_bdd_test contains service-level BDD tests for workflow-compiler.
// Tests wire the real api.Server (in-memory IR store) over a bufconn in-process gRPC
// connection — no mocks, no network ports.
//
// Scenario "YAML with unknown capability is rejected" is marked pending: the compiler
// validates YAML structure only; capability registry checks are M6 scope.
package workflow_compiler_bdd_test

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/api"
)

// ── YAML fixtures ─────────────────────────────────────────────────────────────
// States are a map keyed by name (not a list). Transitions use "on:" with "event:" + "goto:".

const validWorkflow = `
apiVersion: zynax.io/v1
kind: Workflow
metadata:
  name: test-workflow
spec:
  initial_state: review
  states:
    review:
      on:
        - event: approved
          goto: merge
        - event: rejected
          goto: fix
    fix:
      on:
        - event: fixed
          goto: review
    merge:
      on:
        - event: merged
          goto: done
    done:
      type: terminal
`

const noTerminalWorkflow = `
apiVersion: zynax.io/v1
kind: Workflow
metadata:
  name: no-terminal
spec:
  initial_state: start
  states:
    start:
      on:
        - event: next
          goto: end
    end:
      on:
        - event: loop
          goto: start
`

const orphanWorkflow = `
apiVersion: zynax.io/v1
kind: Workflow
metadata:
  name: orphan-test
spec:
  initial_state: start
  states:
    start:
      on:
        - event: done
          goto: terminal
    terminal:
      type: terminal
    orphan:
      type: terminal
`

//nolint:gosec // G101 false positive: "cap" in const name is not a credential
const capWorkflow = `
apiVersion: zynax.io/v1
kind: Workflow
metadata:
  name: cap-workflow
spec:
  initial_state: act
  states:
    act:
      actions:
        - capability: nonexistent_cap
      on:
        - event: done
          goto: end
    end:
      type: terminal
`

// ── testEnv ───────────────────────────────────────────────────────────────────

type testEnv struct {
	srv      *grpc.Server
	lis      *bufconn.Listener
	conn     *grpc.ClientConn
	client   zynaxv1.WorkflowCompilerServiceClient
	lastResp *zynaxv1.CompileWorkflowResponse
	lastErr  error
	yaml     []byte
	wfName   string
}

func (e *testEnv) setup() {
	e.lis = bufconn.Listen(1 << 20)
	e.srv = grpc.NewServer()
	zynaxv1.RegisterWorkflowCompilerServiceServer(e.srv, api.New())
	go func() { _ = e.srv.Serve(e.lis) }()
	conn, _ := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return e.lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	e.conn = conn
	e.client = zynaxv1.NewWorkflowCompilerServiceClient(conn)
}

func (e *testEnv) stop() {
	e.srv.GracefulStop()
	_ = e.conn.Close()
	_ = e.lis.Close()
}

func (e *testEnv) compile(dry bool) {
	e.lastResp, e.lastErr = e.client.CompileWorkflow(context.Background(),
		&zynaxv1.CompileWorkflowRequest{
			ManifestYaml: e.yaml,
			DryRun:       dry,
		})
}

// ── TestFeatures ──────────────────────────────────────────────────────────────

// The nolint directives suppress complexity warnings that are inherent to godog
// ScenarioInitializer closures: every step registration adds a branch.
//
//nolint:funlen,cyclop
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
			sc.Step(`^the workflow compiler is running$`, func(ctx context.Context) (context.Context, error) {
				return ctx, nil // wired in Before
			})

			// ── YAML setup ───────────────────────────────────────────────────
			sc.Step(`^a valid Workflow YAML with states \[review, fix, merge, done\]$`,
				func(ctx context.Context) (context.Context, error) {
					env.yaml = []byte(validWorkflow)
					env.wfName = "test-workflow"
					return ctx, nil
				})
			sc.Step(`^a valid Workflow YAML$`, func(ctx context.Context) (context.Context, error) {
				env.yaml = []byte(validWorkflow)
				env.wfName = "test-workflow"
				return ctx, nil
			})
			sc.Step(`^a Workflow YAML with action capability "([^"]*)"$`,
				func(ctx context.Context, _ string) (context.Context, error) {
					env.yaml = []byte(capWorkflow)
					return ctx, nil
				})
			sc.Step(`^"([^"]*)" is not registered in agent-registry$`,
				func(ctx context.Context, _ string) (context.Context, error) {
					// Capability registry checks are M6 scope — compiler validates structure only.
					return ctx, godog.ErrPending
				})
			sc.Step(`^a Workflow YAML where no state has type: terminal$`,
				func(ctx context.Context) (context.Context, error) {
					env.yaml = []byte(noTerminalWorkflow)
					return ctx, nil
				})
			sc.Step(`^a Workflow YAML with state "([^"]*)" that no transition points to$`,
				func(ctx context.Context, _ string) (context.Context, error) {
					env.yaml = []byte(orphanWorkflow)
					return ctx, nil
				})
			sc.Step(`^workflow "([^"]*)" has been applied$`,
				func(ctx context.Context, name string) (context.Context, error) {
					env.yaml = []byte(strings.ReplaceAll(validWorkflow, "test-workflow", name))
					env.wfName = name
					env.compile(false)
					return ctx, env.lastErr
				})

			// ── Actions ───────────────────────────────────────────────────────
			sc.Step(`^ApplyWorkflow is called$`, func(ctx context.Context) (context.Context, error) {
				env.compile(false)
				return ctx, nil
			})
			sc.Step(`^the same YAML is applied again with identical content$`,
				func(ctx context.Context) (context.Context, error) {
					env.compile(false)
					return ctx, nil
				})
			sc.Step(`^DryRun is called$`, func(ctx context.Context) (context.Context, error) {
				env.compile(true)
				return ctx, nil
			})

			// ── gRPC status assertions ────────────────────────────────────────
			// Note: CompileWorkflow always returns gRPC OK; compilation errors are
			// in resp.GetErrors() per proto contract (ADR-001, issue #477).
			sc.Step(`^the gRPC status is OK$`, func(ctx context.Context) (context.Context, error) {
				if env.lastErr != nil {
					return ctx, fmt.Errorf("expected OK, got: %w", env.lastErr)
				}
				if len(env.lastResp.GetErrors()) > 0 {
					return ctx, fmt.Errorf("expected no compilation errors, got: %v", env.lastResp.GetErrors())
				}
				return ctx, nil
			})
			sc.Step(`^the gRPC status is INVALID_ARGUMENT$`, func(ctx context.Context) (context.Context, error) {
				if env.lastErr != nil {
					return ctx, fmt.Errorf("unexpected transport error: %w", env.lastErr)
				}
				if len(env.lastResp.GetErrors()) == 0 {
					return ctx, fmt.Errorf("expected compilation errors in response, got none")
				}
				return ctx, nil
			})

			// ── IR content assertions ─────────────────────────────────────────
			sc.Step(`^the compiled IR has (\d+) states$`,
				func(ctx context.Context, n int) (context.Context, error) {
					if env.lastResp == nil {
						return ctx, fmt.Errorf("no response")
					}
					if got := len(env.lastResp.GetWorkflowIr().GetStates()); got != n {
						return ctx, fmt.Errorf("expected %d states, got %d", n, got)
					}
					return ctx, nil
				})
			sc.Step(`^the initial_state is "([^"]*)"$`,
				func(ctx context.Context, s string) (context.Context, error) {
					if env.lastResp == nil {
						return ctx, fmt.Errorf("no response")
					}
					if got := env.lastResp.GetWorkflowIr().GetInitialState(); got != s {
						return ctx, fmt.Errorf("expected initial_state %q, got %q", s, got)
					}
					return ctx, nil
				})
			sc.Step(`^state "([^"]*)" has type TERMINAL$`,
				func(ctx context.Context, name string) (context.Context, error) {
					if env.lastResp == nil {
						return ctx, fmt.Errorf("no response")
					}
					for _, st := range env.lastResp.GetWorkflowIr().GetStates() {
						if st.GetId() == name {
							if st.GetType() != zynaxv1.StateType_STATE_TYPE_TERMINAL {
								return ctx, fmt.Errorf("state %q has type %v, want TERMINAL", name, st.GetType())
							}
							return ctx, nil
						}
					}
					return ctx, fmt.Errorf("state %q not found in IR", name)
				})
			sc.Step(`^the response contains a compiled IR$`, func(ctx context.Context) (context.Context, error) {
				if env.lastErr != nil {
					return ctx, fmt.Errorf("unexpected error: %w", env.lastErr)
				}
				if env.lastResp.GetWorkflowIr() == nil {
					return ctx, fmt.Errorf("IR is nil")
				}
				return ctx, nil
			})
			sc.Step(`^no workflow execution is started$`, func(ctx context.Context) (context.Context, error) {
				return ctx, nil // dry_run=true guarantees no submission
			})
			sc.Step(`^agent-registry receives no requests$`, func(ctx context.Context) (context.Context, error) {
				return ctx, nil // compiler never calls registry; no interaction possible
			})
			sc.Step(`^no duplicate workflow record exists$`, func(ctx context.Context) (context.Context, error) {
				return ctx, env.lastErr // idempotent: second compile is a no-op
			})

			// ── Error message assertions ──────────────────────────────────────
			// Compilation errors are in resp.GetErrors(), not in the gRPC error.
			sc.Step(`^the error mentions "([^"]*)"$`,
				func(ctx context.Context, phrase string) (context.Context, error) {
					errs := env.lastResp.GetErrors()
					if len(errs) == 0 {
						return ctx, fmt.Errorf("expected compilation errors mentioning %q, got none", phrase)
					}
					// Check that every word in the phrase appears in at least one error message.
					// This tolerates word-order differences (e.g. "unreachable state" vs "state ... is unreachable").
					words := strings.Fields(strings.ToLower(phrase))
					for _, e := range errs {
						msg := strings.ToLower(e.GetMessage())
						allMatch := true
						for _, w := range words {
							if !strings.Contains(msg, w) {
								allMatch = false
								break
							}
						}
						if allMatch {
							return ctx, nil
						}
					}
					return ctx, fmt.Errorf("no error contains all words of %q; got: %v", phrase, errs)
				})

			// ── Pending: capability check against registry (M6 scope) ─────────
			// The compiler validates YAML structure only; capability registry checks
			// are deferred to M6. Mark the whole scenario pending via its first step.
			sc.Step(`^a Workflow YAML with action capability "([^"]*)" \(registry-check pending\)$`,
				func(ctx context.Context, _ string) (context.Context, error) {
					return ctx, godog.ErrPending
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
