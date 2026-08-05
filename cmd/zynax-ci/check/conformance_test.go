// SPDX-License-Identifier: Apache-2.0

package check_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

const (
	zecsManifestRel = "docs/conformance/scenarios.yaml"
	corpusRel       = "spec/workflows/examples"
	runnerRel       = "scripts/e2e/e2e-happy.sh"
	memberRel       = "spec/workflows/examples/e2e-demo.yaml"
	nonMemberRel    = "spec/workflows/examples/code-review.yaml"
	workflowKind    = "kind: Workflow\nmetadata:\n  name: x\n"
)

// cleanZECS is a well-formed membership manifest: one scenario that runs on both
// legs, and one corpus manifest declared as a reasoned non-member.
const cleanZECS = `suite: Zynax Engine Conformance Suite
version: v0.0.0
corpus: spec/workflows/examples
legs:
  - temporal
  - argo
scenarios:
  - id: echo-happy-path
    source:
      kind: corpus
      path: spec/workflows/examples/e2e-demo.yaml
    legs:
      temporal:
        status: run
        runner: scripts/e2e/e2e-happy.sh
      argo:
        status: run
        runner: scripts/e2e/e2e-happy.sh
non_members:
  - path: spec/workflows/examples/code-review.yaml
    reason: needs agents the e2e cluster does not deploy
`

// engineMain mirrors the engine-adapter entrypoint's engine-name constants — the
// leg source of truth the guard parses (ADR-015).
const engineMain = `package main

const (
	engineTemporal = "temporal"
	engineArgo     = "argo"
)
`

// e2eWorkflow mirrors the e2e matrix leg list.
const e2eWorkflow = `name: e2e smoke
jobs:
  e2e:
    strategy:
      matrix:
        engine: [temporal, argo]
`

func writeRepoFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// buildCleanZECSRepo lays out a minimal repo whose ZECS definition reconciles.
func buildCleanZECSRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeRepoFile(t, root, zecsManifestRel, cleanZECS)
	writeRepoFile(t, root, "services/engine-adapter/cmd/engine-adapter/main.go", engineMain)
	writeRepoFile(t, root, ".github/workflows/e2e-smoke.yml", e2eWorkflow)
	writeRepoFile(t, root, memberRel, workflowKind)
	writeRepoFile(t, root, nonMemberRel, workflowKind)
	writeRepoFile(t, root, corpusRel+"/agent-def-example.yaml", "kind: AgentDef\n")
	writeRepoFile(t, root, runnerRel, "#!/usr/bin/env bash\n")
	return root
}

// assertProblem fails unless exactly one problem is reported and it mentions want.
func assertProblem(t *testing.T, root, want string) {
	t.Helper()
	problems, _, err := check.Conformance(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("expected exactly 1 problem, got %d: %v", len(problems), problems)
	}
	if !strings.Contains(problems[0], want) {
		t.Errorf("problem %q does not mention %q", problems[0], want)
	}
}

func TestConformance_CleanReconciles(t *testing.T) {
	problems, count, err := check.Conformance(buildCleanZECSRepo(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) != 0 {
		t.Errorf("expected a clean definition to reconcile, got: %v", problems)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

// A scenario naming a manifest that does not exist must be caught (AC3 forward).
func TestConformance_MissingScenarioSource(t *testing.T) {
	root := buildCleanZECSRepo(t)
	if err := os.Remove(filepath.Join(root, memberRel)); err != nil {
		t.Fatal(err)
	}
	problems, _, err := check.Conformance(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) == 0 {
		t.Fatal("expected the deleted scenario manifest to be reported")
	}
}

// A corpus workflow that is neither a scenario nor a declared non-member must be
// caught (AC3 reverse) — a manifest can never drop out of the suite silently.
func TestConformance_UnaccountedCorpusManifest(t *testing.T) {
	root := buildCleanZECSRepo(t)
	writeRepoFile(t, root, corpusRel+"/new-thing.yaml", workflowKind)
	assertProblem(t, root, "neither a ZECS scenario nor a declared non_member")
}

// An engine the engine-adapter can be configured with, but which the suite does not
// declare as a leg, must be caught (AC4) — with no engine name in the check.
func TestConformance_NewEngineNotDeclaredAsLeg(t *testing.T) {
	root := buildCleanZECSRepo(t)
	writeRepoFile(t, root, "services/engine-adapter/cmd/engine-adapter/main.go",
		strings.Replace(engineMain, ")\n", "\tengineFlyte = \"flyte\"\n)\n", 1))
	problems, _, err := check.Conformance(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(problems) == 0 || !strings.Contains(problems[0], "flyte") {
		t.Fatalf("expected the new engine to be reported as an undeclared leg, got: %v", problems)
	}
}

// A leg the suite claims but the runner never runs must be caught (AC4).
func TestConformance_LegMissingFromE2EMatrix(t *testing.T) {
	root := buildCleanZECSRepo(t)
	writeRepoFile(t, root, ".github/workflows/e2e-smoke.yml",
		strings.Replace(e2eWorkflow, "[temporal, argo]", "[temporal]", 1))
	assertProblem(t, root, "absent from the e2e matrix")
}

// Omitting a leg from a scenario must be caught: silence would let a leg that never
// ran read as a pass.
func TestConformance_ScenarioOmitsLeg(t *testing.T) {
	root := buildCleanZECSRepo(t)
	manifest := strings.Replace(cleanZECS, `      argo:
        status: run
        runner: scripts/e2e/e2e-happy.sh
`, "", 1)
	writeRepoFile(t, root, zecsManifestRel, manifest)
	assertProblem(t, root, "no entry for leg")
}

// A not_run leg must carry a reason.
func TestConformance_NotRunLegWithoutReason(t *testing.T) {
	root := buildCleanZECSRepo(t)
	manifest := strings.Replace(cleanZECS, `      argo:
        status: run
        runner: scripts/e2e/e2e-happy.sh
`, "      argo:\n        status: not_run\n", 1)
	writeRepoFile(t, root, zecsManifestRel, manifest)
	assertProblem(t, root, "no 'reason'")
}

// A non-member without a reason must be caught.
func TestConformance_NonMemberWithoutReason(t *testing.T) {
	root := buildCleanZECSRepo(t)
	manifest := strings.Replace(cleanZECS,
		"    reason: needs agents the e2e cluster does not deploy\n", "", 1)
	writeRepoFile(t, root, zecsManifestRel, manifest)
	assertProblem(t, root, "missing 'reason'")
}

// The guard must fail loudly — not pass vacuously — when the engine-name constants
// move out of the entrypoint it parses.
func TestConformance_EngineConstantsMoved(t *testing.T) {
	root := buildCleanZECSRepo(t)
	writeRepoFile(t, root, "services/engine-adapter/cmd/engine-adapter/main.go", "package main\n")
	if _, _, err := check.Conformance(root); err == nil {
		t.Fatal("expected an operational error when no engine constants are found")
	}
}

// A missing membership manifest is an operational error, not a silent pass.
func TestConformance_MissingManifest(t *testing.T) {
	if _, _, err := check.Conformance(t.TempDir()); err == nil {
		t.Fatal("expected an error when the membership manifest is absent")
	}
}
