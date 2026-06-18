// SPDX-License-Identifier: Apache-2.0

package check_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

// cleanMapping is a well-formed runtime_mapping.yaml: one runtime-backed expert
// and one authoring-only expert.
const cleanMapping = `experts:
  - authoring: go-services
    runtime_mapping: go-review-expert
  - authoring: bdd-contract
    runtime_mapping: authoring-only
`

// cleanADR mirrors cleanMapping in ADR-033 table form.
const cleanADR = "# ADR-033\n" +
	"| Authoring | Runtime | Capability | Status |\n" +
	"|---|---|---|---|\n" +
	"| `go-services` | `go-review-expert` | `code.review.go` | runtime |\n" +
	"| `bdd-contract` | `authoring-only` | — | authoring-only |\n"

func writeExpertFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// buildCleanExpertRepo lays out a minimal repo that reconciles cleanly.
func buildCleanExpertRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeExpertFile(t, root, "automation/experts/runtime_mapping.yaml", cleanMapping)
	writeExpertFile(t, root, "docs/adr/ADR-033-expert-agent-substrate.md", cleanADR)
	writeExpertFile(t, root, ".claude/commands/experts/go-services.md", "# go-services\n")
	writeExpertFile(t, root, ".claude/commands/experts/bdd-contract.md", "# bdd-contract\n")
	writeExpertFile(t, root, "agents/examples/go-review-expert/pyproject.toml", "[project]\nname='go-review-expert'\n")
	return root
}

func TestExpertMapping_CleanReconciles(t *testing.T) {
	problems, count, err := check.ExpertMapping(buildCleanExpertRepo(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Errorf("expected clean repo to reconcile, got: %v", problems)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestExpertMapping_MissingDeclaration(t *testing.T) {
	root := buildCleanExpertRepo(t)
	// Add an authoring expert that is not declared in the mapping.
	writeExpertFile(t, root, ".claude/commands/experts/phantom-expert.md", "# phantom\n")
	problems, _, err := check.ExpertMapping(root)
	if err != nil {
		t.Fatal(err)
	}
	if !anyContains(problems, "phantom-expert") {
		t.Errorf("rule 1 should trip on phantom-expert, got: %v", problems)
	}
}

func TestExpertMapping_DanglingRuntimeRef(t *testing.T) {
	root := buildCleanExpertRepo(t)
	writeExpertFile(t, root, "automation/experts/runtime_mapping.yaml", strings.Replace(
		cleanMapping, "runtime_mapping: go-review-expert", "runtime_mapping: does-not-exist", 1))
	writeExpertFile(t, root, "docs/adr/ADR-033-expert-agent-substrate.md", strings.Replace(
		cleanADR, "`go-review-expert`", "`does-not-exist`", 1))
	problems, _, err := check.ExpertMapping(root)
	if err != nil {
		t.Fatal(err)
	}
	if !anyContains(problems, "does-not-exist") {
		t.Errorf("rule 2 should trip on dangling ref, got: %v", problems)
	}
}

func TestExpertMapping_ADRTableDisagrees(t *testing.T) {
	root := buildCleanExpertRepo(t)
	// Mapping says authoring-only; ADR table says something else for bdd-contract.
	writeExpertFile(t, root, "docs/adr/ADR-033-expert-agent-substrate.md", strings.Replace(
		cleanADR, "| `bdd-contract` | `authoring-only` |", "| `bdd-contract` | `drifted-expert` |", 1))
	problems, _, err := check.ExpertMapping(root)
	if err != nil {
		t.Fatal(err)
	}
	if !anyContains(problems, "ADR-033 table") {
		t.Errorf("rule 3 should trip on table disagreement, got: %v", problems)
	}
}

func TestExpertMapping_EmptyRuntimeMapping(t *testing.T) {
	root := buildCleanExpertRepo(t)
	writeExpertFile(t, root, "automation/experts/runtime_mapping.yaml", strings.Replace(
		cleanMapping, "runtime_mapping: authoring-only", "runtime_mapping: \"\"", 1))
	problems, _, err := check.ExpertMapping(root)
	if err != nil {
		t.Fatal(err)
	}
	if !anyContains(problems, "runtime_mapping") {
		t.Errorf("rule 1 should trip on empty runtime_mapping, got: %v", problems)
	}
}

func TestExpertMapping_MissingMappingFile(t *testing.T) {
	if _, _, err := check.ExpertMapping(t.TempDir()); err == nil {
		t.Error("expected an operational error when runtime_mapping.yaml is absent")
	}
}

func anyContains(items []string, sub string) bool {
	for _, s := range items {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
