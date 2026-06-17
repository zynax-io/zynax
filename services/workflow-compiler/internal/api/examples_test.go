package api_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/api"
)

// examplesDir is the repository-root spec/workflows/examples directory relative
// to this package (services/workflow-compiler/internal/api → four levels up).
const examplesDir = "../../../../spec/workflows/examples"

// binding is a single cross-state data-flow edge declared by an example
// workflow: the consumer state consumes inputKey, whose value is the source
// reference "$.states.<producer>.output.<outputKey>" (ADR-029, EPIC W.4 #1178).
type binding struct {
	consumer  string // state declaring the input binding
	inputKey  string // the input field key on the consuming action
	producer  string // upstream state that publishes the output
	outputKey string // the published output key on the producing action
}

// exampleWorkflow describes the end-to-end expectations for one of the real,
// runnable reference workflows that landed in #1208. EPIC W step 5 (#1179)
// proves these compile green and that their data-flow bindings resolve.
type exampleWorkflow struct {
	file     string
	name     string    // metadata.name after compilation
	bindings []binding // documented data-flow edges that must resolve
}

func realWorkflows() []exampleWorkflow {
	return []exampleWorkflow{
		{
			file: "research-task.yaml",
			name: "research-task-workflow",
		},
		{
			file: "code-review.yaml",
			name: "code-review-workflow",
			bindings: []binding{
				{consumer: "escalate", inputKey: "review_summary", producer: "fix", outputKey: "feedback_summary"},
			},
		},
		{
			file: "ci-pipeline.yaml",
			name: "ci-pipeline-workflow",
			bindings: []binding{
				{consumer: "deploy_staging", inputKey: "image", producer: "build", outputKey: "built_image"},
				{consumer: "integration_test", inputKey: "image", producer: "build", outputKey: "built_image"},
			},
		},
		{
			file: "feature-implementation.yaml",
			name: "feature-implementation-workflow",
			bindings: []binding{
				{consumer: "implement", inputKey: "plan", producer: "plan", outputKey: "implementation_plan"},
				{consumer: "verify", inputKey: "branch", producer: "implement", outputKey: "branch_name"},
				{consumer: "rework", inputKey: "branch", producer: "implement", outputKey: "branch_name"},
				{consumer: "rework", inputKey: "failures", producer: "verify", outputKey: "test_report"},
				{consumer: "open_pr", inputKey: "branch", producer: "implement", outputKey: "branch_name"},
				{consumer: "open_pr", inputKey: "body", producer: "implement", outputKey: "diff_summary"},
			},
		},
	}
}

// compileExample reads one reference workflow from spec/workflows/examples and
// compiles it through the real YAML→IR pipeline (parse → graph build →
// data-flow binding resolution → IR). It fails the test if the file cannot be
// read or the server returns a gRPC error.
func compileExample(t *testing.T, file string) *zynaxv1.CompileWorkflowResponse {
	t.Helper()
	// #nosec G304 -- file is a fixed name from the in-test workflow table, not user input.
	yaml, err := os.ReadFile(filepath.Join(examplesDir, file))
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	resp, err := api.New().CompileWorkflow(context.Background(), &zynaxv1.CompileWorkflowRequest{
		ManifestYaml: yaml,
	})
	if err != nil {
		t.Fatalf("CompileWorkflow(%s): unexpected gRPC error: %v", file, err)
	}
	return resp
}

// stateByID returns the StateIR with the given id, or nil when absent.
func stateByID(ir *zynaxv1.WorkflowIR, id string) *zynaxv1.StateIR {
	for _, s := range ir.GetStates() {
		if s.GetId() == id {
			return s
		}
	}
	return nil
}

// TestExampleWorkflows_CompileGreen proves every real reference workflow from
// #1208 compiles end-to-end with no errors and yields a WorkflowIR — the
// keystone assertion for EPIC W step 5 (#1179). A non-empty resp.Errors here
// means an unresolved data-flow binding, an orphan state, or a missing terminal,
// because graph.Build reports all of those as compile errors.
func TestExampleWorkflows_CompileGreen(t *testing.T) {
	for _, wf := range realWorkflows() {
		t.Run(wf.file, func(t *testing.T) {
			resp := compileExample(t, wf.file)
			if len(resp.GetErrors()) != 0 {
				t.Fatalf("%s did not compile green: %v", wf.file, resp.GetErrors())
			}
			if resp.GetWorkflowIr() == nil {
				t.Fatalf("%s: expected WorkflowIR, got nil", wf.file)
			}
			if got := resp.GetWorkflowIr().GetName(); got != wf.name {
				t.Errorf("%s: name = %q, want %q", wf.file, got, wf.name)
			}
		})
	}
}

// TestExampleWorkflows_ReachTerminal asserts each workflow has at least one
// terminal state in its compiled IR, so an execution can in principle reach a
// terminal state rather than loop forever (AC: workflows run to terminal).
func TestExampleWorkflows_ReachTerminal(t *testing.T) {
	for _, wf := range realWorkflows() {
		t.Run(wf.file, func(t *testing.T) {
			ir := compileExample(t, wf.file).GetWorkflowIr()
			if ir == nil {
				t.Fatalf("%s: nil WorkflowIR", wf.file)
			}
			hasTerminal := false
			for _, s := range ir.GetStates() {
				if s.GetType() == zynaxv1.StateType_STATE_TYPE_TERMINAL {
					hasTerminal = true
					break
				}
			}
			if !hasTerminal {
				t.Errorf("%s: compiled IR has no terminal state", wf.file)
			}
		})
	}
}

// TestExampleWorkflows_DataFlowResolves asserts that every documented
// cross-state data-flow binding survives compilation: the consumer action
// carries the "$.states.<producer>.output.<key>" reference as an input binding,
// and the producer action declares the matching output binding. This is the
// concrete proof that data-flow is wired end-to-end (AC: summarize consumes
// search output, etc.).
func TestExampleWorkflows_DataFlowResolves(t *testing.T) {
	for _, wf := range realWorkflows() {
		if len(wf.bindings) == 0 {
			continue
		}
		t.Run(wf.file, func(t *testing.T) {
			ir := compileExample(t, wf.file).GetWorkflowIr()
			if ir == nil {
				t.Fatalf("%s: nil WorkflowIR", wf.file)
			}
			for _, b := range wf.bindings {
				want := "$.states." + b.producer + ".output." + b.outputKey
				assertInputBinding(t, ir, b.consumer, b.inputKey, want)
				assertOutputBinding(t, ir, b.producer, b.outputKey)
			}
		})
	}
}

// assertInputBinding fails the test unless some action in the consumer state
// declares an input binding inputKey == want.
func assertInputBinding(t *testing.T, ir *zynaxv1.WorkflowIR, consumer, inputKey, want string) {
	t.Helper()
	st := stateByID(ir, consumer)
	if st == nil {
		t.Fatalf("consumer state %q not found in IR", consumer)
	}
	for _, a := range st.GetActions() {
		if got, ok := a.GetInputBindings()[inputKey]; ok {
			if got != want {
				t.Errorf("state %q input[%q] = %q, want %q", consumer, inputKey, got, want)
			}
			return
		}
	}
	t.Errorf("state %q has no input binding for key %q", consumer, inputKey)
}

// assertOutputBinding fails the test unless some action in the producer state
// publishes outputKey via an output binding.
func assertOutputBinding(t *testing.T, ir *zynaxv1.WorkflowIR, producer, outputKey string) {
	t.Helper()
	st := stateByID(ir, producer)
	if st == nil {
		t.Fatalf("producer state %q not found in IR", producer)
	}
	for _, a := range st.GetActions() {
		if _, ok := a.GetOutputBindings()[outputKey]; ok {
			return
		}
	}
	t.Errorf("state %q publishes no output binding for key %q", producer, outputKey)
}
