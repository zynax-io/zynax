// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot resolves the repository root from this test file's location.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	// file = cmd/zynax/validate/scenario_test.go
	return filepath.Join(filepath.Dir(file), "../../..")
}

const validIndex = `kind: Scenario
apiVersion: zynax.io/v1alpha1
metadata:
  name: demo
spec:
  members:
    - id: agent
      kind: AgentDef
      file: agent.yaml
    - id: workflow
      kind: Workflow
      file: workflow.yaml
  apply_order:
    - agent
    - workflow
`

func writeIndex(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	idx := filepath.Join(dir, ScenarioIndexFile)
	if err := os.WriteFile(idx, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestExpandScenario_OrdersMembers(t *testing.T) {
	idx := writeIndex(t, validIndex)
	members, err := ExpandScenario(idx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("got %d members, want 2", len(members))
	}
	if members[0].ID != "agent" || members[1].ID != "workflow" {
		t.Errorf("apply order = [%s, %s], want [agent, workflow]", members[0].ID, members[1].ID)
	}
	if !strings.HasSuffix(members[0].Path, "agent.yaml") {
		t.Errorf("member path = %q, want suffix agent.yaml", members[0].Path)
	}
}

func TestExpandScenario_UnknownApplyOrderID(t *testing.T) {
	body := strings.Replace(validIndex, "    - agent\n", "    - ghost\n", 1)
	idx := writeIndex(t, body)
	if _, err := ExpandScenario(idx); err == nil {
		t.Error("expected error for apply_order referencing unknown member id")
	}
}

func TestExpandScenario_DuplicateApplyOrder(t *testing.T) {
	body := strings.Replace(validIndex, "    - workflow\n", "    - agent\n", 1)
	idx := writeIndex(t, body)
	if _, err := ExpandScenario(idx); err == nil {
		t.Error("expected error for duplicate id in apply_order")
	}
}

func TestExpandScenario_EmptyApplyOrder(t *testing.T) {
	body := `kind: Scenario
apiVersion: zynax.io/v1alpha1
metadata:
  name: demo
spec:
  members:
    - id: agent
      kind: AgentDef
      file: agent.yaml
  apply_order: []
`
	idx := writeIndex(t, body)
	if _, err := ExpandScenario(idx); err == nil {
		t.Error("expected error for empty apply_order")
	}
}

func TestExpandScenario_DuplicateMemberID(t *testing.T) {
	body := `kind: Scenario
apiVersion: zynax.io/v1alpha1
metadata:
  name: demo
spec:
  members:
    - id: agent
      kind: AgentDef
      file: agent.yaml
    - id: agent
      kind: Workflow
      file: workflow.yaml
  apply_order:
    - agent
`
	idx := writeIndex(t, body)
	if _, err := ExpandScenario(idx); err == nil {
		t.Error("expected error for duplicate member id")
	}
}

func TestExpandScenario_PathTraversalRejected(t *testing.T) {
	body := strings.Replace(validIndex, "file: agent.yaml", "file: ../../etc/passwd", 1)
	idx := writeIndex(t, body)
	if _, err := ExpandScenario(idx); err == nil {
		t.Error("expected error for member file escaping the scenario directory")
	}
}

func TestExpandScenario_AbsolutePathRejected(t *testing.T) {
	body := strings.Replace(validIndex, "file: agent.yaml", "file: /etc/passwd", 1)
	idx := writeIndex(t, body)
	if _, err := ExpandScenario(idx); err == nil {
		t.Error("expected error for absolute member file path")
	}
}

func TestResolveScenarioIndex_Directory(t *testing.T) {
	idx := writeIndex(t, validIndex)
	dir := filepath.Dir(idx)
	got, ok, err := ResolveScenarioIndex(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected directory with scenario.yaml to resolve as a scenario")
	}
	if got != idx {
		t.Errorf("index path = %q, want %q", got, idx)
	}
}

func TestResolveScenarioIndex_PlainDirNotScenario(t *testing.T) {
	dir := t.TempDir()
	_, ok, err := ResolveScenarioIndex(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("a directory with no scenario.yaml must not resolve as a scenario")
	}
}

func TestResolveScenarioIndex_NonScenarioFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "wf.yaml")
	if err := os.WriteFile(f, []byte("kind: Workflow\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, ok, err := ResolveScenarioIndex(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("a kind: Workflow file must not resolve as a scenario")
	}
}

func TestResolveScenarioIndex_ScenarioFile(t *testing.T) {
	idx := writeIndex(t, validIndex)
	got, ok, err := ResolveScenarioIndex(idx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok || got != idx {
		t.Errorf("scenario index file did not resolve: ok=%v path=%q", ok, got)
	}
}

func TestResolveScenarioIndex_Missing(t *testing.T) {
	if _, _, err := ResolveScenarioIndex("/nonexistent/path"); err == nil {
		t.Error("expected error for missing path")
	}
}

// ScenarioFile end-to-end against the shipped reference scenario: the index and
// every member must validate against their existing schemas.
func TestScenarioFile_ReferenceScenarioValid(t *testing.T) {
	root := repoRoot(t)
	schemaDir := filepath.Join(root, "spec/schemas")
	idx := filepath.Join(root, "spec/scenarios/code-review", ScenarioIndexFile)
	errs, err := ScenarioFile(idx, schemaDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("reference scenario should be valid, got errors: %v", errs)
	}
}

func TestScenarioFile_InvalidIndexStops(t *testing.T) {
	root := repoRoot(t)
	schemaDir := filepath.Join(root, "spec/schemas")
	// Index missing apply_order — schema-invalid.
	body := `kind: Scenario
apiVersion: zynax.io/v1alpha1
metadata:
  name: demo
spec:
  members:
    - id: agent
      kind: AgentDef
      file: agent.yaml
`
	idx := writeIndex(t, body)
	errs, err := ScenarioFile(idx, schemaDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("expected schema errors for index missing apply_order")
	}
}
