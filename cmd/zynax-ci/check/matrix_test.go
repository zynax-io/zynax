// SPDX-License-Identifier: Apache-2.0

package check_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

const (
	outcomesRel = "outcomes"
	scenarioID  = "echo-happy-path"
	legGreen    = "happy-path=success\n"
	legRed      = "happy-path=failure\n"
)

// writeOutcomes lays down one recorded-outcome file for a leg.
func writeOutcomes(t *testing.T, root, leg, body string) {
	t.Helper()
	writeRepoFile(t, root, filepath.Join(outcomesRel, "zecs-outcomes-"+leg+".txt"), "leg="+leg+"\n"+body)
}

// matrixFrom builds the matrix from the recorded outcomes in <root>/outcomes,
// which CI creates whether or not any leg reported into it.
func matrixFrom(t *testing.T, root string) check.MatrixDoc {
	t.Helper()
	dir := filepath.Join(root, outcomesRel)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	doc, err := check.ConformanceMatrix(check.MatrixOptions{Root: root, OutcomesDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return doc
}

// legOf returns the matrix row for one engine — every engine always has one.
func legOf(t *testing.T, doc check.MatrixDoc, engine string) check.LegMatrix {
	t.Helper()
	for _, leg := range doc.Legs {
		if leg.Engine == engine {
			return leg
		}
	}
	t.Fatalf("leg %q is absent from the matrix — a leg must never be omitted, got %+v", engine, doc.Legs)
	return check.LegMatrix{}
}

// cellOf returns one scenario cell of one leg.
func cellOf(t *testing.T, doc check.MatrixDoc, engine, scenario string) check.ScenarioMatrix {
	t.Helper()
	for _, sc := range legOf(t, doc, engine).Scenarios {
		if sc.ID == scenario {
			return sc
		}
	}
	t.Fatalf("scenario %q is absent from leg %q", scenario, engine)
	return check.ScenarioMatrix{}
}

// AC2 — the property this story exists for, asserted rather than intended. A leg
// that was MEANT to run a scenario and produced no outcome renders SKIPPED (never
// PASS), makes its leg INCOMPLETE, and makes the whole run incomplete.
func TestMatrix_SkippedLegNeverRendersPass(t *testing.T) {
	root := buildCleanZECSRepo(t)
	writeOutcomes(t, root, "temporal", legGreen)
	writeOutcomes(t, root, "argo", "") // the leg ran but recorded no scenario outcome

	doc := matrixFrom(t, root)
	cell := cellOf(t, doc, "argo", scenarioID)
	if cell.Result != check.ResultSkipped || cell.SkipKind != check.SkipNotExecuted {
		t.Fatalf("an unobserved scenario rendered %s/%s — it must be SKIPPED/%s",
			cell.Result, cell.SkipKind, check.SkipNotExecuted)
	}
	if leg := legOf(t, doc, "argo"); leg.Result != check.ResultIncomplete || leg.Complete {
		t.Errorf("leg = %s complete=%t, want INCOMPLETE/false", leg.Result, leg.Complete)
	}
	if doc.Result == check.ResultPass || doc.Complete {
		t.Errorf("run = %s complete=%t — a run with an unexecuted scenario is not a pass", doc.Result, doc.Complete)
	}
}

// Only an observed success or failure is evidence. Everything else — skipped,
// cancelled, empty, unrecognised — is SKIPPED/not_executed, never PASS.
func TestMatrix_CellVerdicts(t *testing.T) {
	for _, tc := range []struct {
		outcome, want, skipKind string
	}{
		{"success", check.ResultPass, ""},
		{"failure", check.ResultFail, ""},
		{"skipped", check.ResultSkipped, check.SkipNotExecuted},
		{"cancelled", check.ResultSkipped, check.SkipNotExecuted},
		{"", check.ResultSkipped, check.SkipNotExecuted},
		{"SUCCESS", check.ResultSkipped, check.SkipNotExecuted},
		{"ok", check.ResultSkipped, check.SkipNotExecuted},
	} {
		t.Run("outcome="+tc.outcome, func(t *testing.T) {
			root := buildCleanZECSRepo(t)
			writeOutcomes(t, root, "temporal", "happy-path="+tc.outcome+"\n")
			cell := cellOf(t, matrixFrom(t, root), "temporal", scenarioID)
			if cell.Result != tc.want || cell.SkipKind != tc.skipKind {
				t.Errorf("outcome %q rendered %s/%s, want %s/%s",
					tc.outcome, cell.Result, cell.SkipKind, tc.want, tc.skipKind)
			}
		})
	}
}

// Leg and run aggregation, including the one that matters most: a leg with no
// outcome file at all is present as NOT_RUN — never absent — so a one-leg run
// cannot be consumed as a full matrix, and a run with no evidence at all still
// emits a document rather than nothing.
func TestMatrix_LegAndRunVerdicts(t *testing.T) {
	const absent = "\x00" // no outcome file for this leg at all
	for _, tc := range []struct {
		name, temporal, argo             string
		wantArgo, wantResult, wantForced string
		wantComplete                     bool
	}{
		{"both green", legGreen, legGreen, check.ResultPass, check.ResultPass, check.ResultPass, true},
		{"required leg red", legRed, legGreen, check.ResultPass, check.ResultFail, check.ResultFail, true},
		// A red advisory leg fails the run but blocks no merge (gap G6).
		{"advisory leg red", legGreen, legRed, check.ResultFail, check.ResultFail, check.ResultPass, true},
		{"advisory leg absent", legGreen, absent, check.ResultNotRun, check.ResultIncomplete, check.ResultPass, false},
		{"no leg ran", absent, absent, check.ResultNotRun, check.ResultIncomplete, check.ResultIncomplete, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := buildCleanZECSRepo(t)
			for leg, body := range map[string]string{"temporal": tc.temporal, "argo": tc.argo} {
				if body != absent {
					writeOutcomes(t, root, leg, body)
				}
			}
			doc := matrixFrom(t, root)
			if leg := legOf(t, doc, "argo"); leg.Result != tc.wantArgo {
				t.Errorf("argo leg = %s, want %s", leg.Result, tc.wantArgo)
			}
			if doc.Result != tc.wantResult || doc.EnforcedResult != tc.wantForced || doc.Complete != tc.wantComplete {
				t.Errorf("run = %s enforced = %s complete = %t, want %s/%s/%t",
					doc.Result, doc.EnforcedResult, doc.Complete, tc.wantResult, tc.wantForced, tc.wantComplete)
			}
		})
	}
}

// The by-design skip must stay distinguishable from the dangerous one: a leg the
// manifest excludes carries its reason and gap and does NOT make the leg
// incomplete.
func TestMatrix_NotInLegSkipIsDistinctFromNotExecuted(t *testing.T) {
	root := buildCleanZECSRepo(t)
	writeRepoFile(t, root, zecsManifestRel, strings.Replace(cleanZECS, `      argo:
        status: run
        runner: scripts/e2e/e2e-happy.sh
        ci_step: happy-path
`, "      argo:\n        status: not_run\n        gap: G3\n        reason: temporal-only step guard\n", 1))
	writeOutcomes(t, root, "temporal", legGreen)
	writeOutcomes(t, root, "argo", "")

	doc := matrixFrom(t, root)
	cell := cellOf(t, doc, "argo", scenarioID)
	if cell.Result != check.ResultSkipped || cell.SkipKind != check.SkipNotInLeg || cell.Planned {
		t.Fatalf("cell = %s/%s planned=%t, want SKIPPED/%s/false",
			cell.Result, cell.SkipKind, cell.Planned, check.SkipNotInLeg)
	}
	if cell.Gap != "G3" || cell.Reason == "" {
		t.Errorf("cell gap=%q reason=%q, want the manifest's gap and reason", cell.Gap, cell.Reason)
	}
	if leg := legOf(t, doc, "argo"); leg.Result != check.ResultPass || !leg.Complete {
		t.Errorf("leg = %s complete=%t — a by-design skip must not make a leg incomplete", leg.Result, leg.Complete)
	}
}

// AC4 — engine N+1 appears with no edit to the matrix logic: it is enumerated
// from the engine-adapter, and until it really runs it is NOT_RUN with unknown
// enforcement, never an implicit pass.
func TestMatrix_NewEngineAppearsAsANotRunLeg(t *testing.T) {
	root := buildCleanZECSRepo(t)
	writeRepoFile(t, root, "services/engine-adapter/cmd/engine-adapter/main.go",
		strings.Replace(engineMain, ")\n", "\tengineFlyte = \"flyte\"\n)\n", 1))
	writeOutcomes(t, root, "temporal", legGreen)
	writeOutcomes(t, root, "argo", legGreen)

	doc := matrixFrom(t, root)
	if leg := legOf(t, doc, "flyte"); leg.Result != check.ResultNotRun || leg.Enforcement != "unknown" {
		t.Errorf("new engine leg = %s enforcement=%q, want NOT_RUN/unknown", leg.Result, leg.Enforcement)
	}
	if doc.Complete {
		t.Error("a run missing a selectable engine's leg must not be complete")
	}
}

// AC3 — --run executes the leg's runners, records real exit statuses, and emits
// the same document shape; the legs it did not target stay NOT_RUN.
func TestMatrix_RunModeRecordsRunnerExitStatus(t *testing.T) {
	root := buildCleanZECSRepo(t)
	writeRepoFile(t, root, runnerRel, "#!/usr/bin/env bash\nexit 3\n")

	var log bytes.Buffer
	doc, err := check.ConformanceMatrix(check.MatrixOptions{Root: root, Run: true, Leg: "temporal", Log: &log})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cell := cellOf(t, doc, "temporal", scenarioID); cell.Result != check.ResultFail {
		t.Errorf("cell = %s, want FAIL from the non-zero runner", cell.Result)
	}
	if leg := legOf(t, doc, "argo"); leg.Result != check.ResultNotRun {
		t.Errorf("untargeted leg = %s, want NOT_RUN — a single-engine run is not a suite result", leg.Result)
	}
	if !strings.Contains(log.String(), runnerRel) {
		t.Errorf("the runner's output was not streamed to the log: %q", log.String())
	}
	if _, err := check.ConformanceMatrix(check.MatrixOptions{Root: root, Run: true, Leg: "flyte"}); err == nil {
		t.Error("expected --run against an engine the engine-adapter does not offer to be rejected")
	}
}

// Ambiguous or unattributable input is an error, not a guess: guessing a leg from
// a filename is exactly the shortcut that would mislabel a result.
func TestMatrix_OutcomeFileErrors(t *testing.T) {
	for _, tc := range []struct{ name, file, body string }{
		{"unknown leg", "zecs-outcomes-flyte.txt", "leg=flyte\n" + legGreen},
		{"duplicate leg", "copy.txt", "leg=temporal\n" + legRed},
		{"no leg line", "orphan.txt", legGreen},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := buildCleanZECSRepo(t)
			writeOutcomes(t, root, "temporal", legGreen)
			writeRepoFile(t, root, filepath.Join(outcomesRel, tc.file), tc.body)
			if _, err := check.ConformanceMatrix(check.MatrixOptions{
				Root:        root,
				OutcomesDir: filepath.Join(root, outcomesRel),
			}); err == nil {
				t.Fatalf("expected an error for %s", tc.name)
			}
		})
	}
	// An outcomes directory that does not exist is a mistyped path, not evidence
	// that no leg ran — reporting an all-NOT_RUN matrix for it would look
	// identical to a real dead run.
	if _, err := check.ConformanceMatrix(check.MatrixOptions{
		Root: buildCleanZECSRepo(t), OutcomesDir: filepath.Join(t.TempDir(), "absent"),
	}); err == nil {
		t.Error("expected a missing outcomes directory to be an error")
	}
}

// The live inputs — the real manifest, engine-adapter entrypoint and e2e workflow
// — must produce a matrix.
func TestMatrix_RealRepositoryEmits(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, zecsManifestRel)); err != nil {
		t.Skip("not running inside the repository")
	}
	doc, err := check.ConformanceMatrix(check.MatrixOptions{Root: root, OutcomesDir: t.TempDir()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(doc.Legs) == 0 || len(doc.Legs[0].Scenarios) == 0 || doc.Complete {
		t.Fatalf("the real manifest produced %d legs, complete=%t", len(doc.Legs), doc.Complete)
	}
}

// The JSON is the contract: the fields a consumer keys on must be present, and
// the one-line summary must not read greener than the document. Comments and
// stray whitespace in an outcome file are noise, not data.
func TestMatrix_RenderingsCarryTheSameVerdicts(t *testing.T) {
	root := buildCleanZECSRepo(t)
	writeOutcomes(t, root, "temporal", "\n# a comment\n\n  happy-path = success  \n")
	doc := matrixFrom(t, root)
	if cell := cellOf(t, doc, "temporal", scenarioID); cell.Result != check.ResultPass {
		t.Fatalf("cell = %s, want PASS despite the noise in the outcome file", cell.Result)
	}

	var buf bytes.Buffer
	if err := check.EncodeMatrix(&buf, doc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, field := range []string{
		`"schema_version"`, `"complete"`, `"result"`, `"enforced_result"`,
		`"legs"`, `"enforcement"`, `"executed"`, `"skip_kind"`, `"notes"`,
	} {
		if !strings.Contains(buf.String(), field) {
			t.Errorf("the matrix JSON is missing %s", field)
		}
	}

	summary := check.Summary(doc)
	for _, want := range []string{"result=INCOMPLETE", "complete=false", "argo(advisory)=NOT_RUN"} {
		if !strings.Contains(summary, want) {
			t.Errorf("the summary %q is missing %q", summary, want)
		}
	}
}
