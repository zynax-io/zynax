// SPDX-License-Identifier: Apache-2.0

package check_test

import (
	"strings"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

// completeDoc is a full run: one required leg and one advisory leg, with the
// advisory leg legitimately not running one scenario (gap G3).
func completeDoc() check.MatrixDoc {
	return check.MatrixDoc{
		SchemaVersion:  "1",
		Suite:          "Zynax Engine Conformance Suite",
		ShortName:      "ZECS",
		Version:        "v0.8.0",
		Definition:     "docs/conformance/README.md",
		GeneratedAt:    "2026-08-14T09:00:00Z",
		Revision:       "abc1234",
		RunURL:         "https://example.invalid/runs/1",
		Result:         check.ResultPass,
		EnforcedResult: check.ResultPass,
		Complete:       true,
		Notes:          []string{"Results are per leg."},
		Legs: []check.LegMatrix{
			{
				Engine: "temporal", Enforcement: "required", Executed: true, Complete: true,
				Result: check.ResultPass,
				Scenarios: []check.ScenarioMatrix{
					{ID: "echo-happy-path", Result: check.ResultPass, Planned: true, Observed: "success"},
					{ID: "hello-world-outputs", Result: check.ResultPass, Planned: true, Observed: "success"},
				},
			},
			{
				Engine: "argo", Enforcement: "advisory", Executed: true, Complete: true,
				Result: check.ResultPass,
				Scenarios: []check.ScenarioMatrix{
					{ID: "echo-happy-path", Result: check.ResultPass, Planned: true, Observed: "success"},
					{ID: "hello-world-outputs", Result: check.ResultSkipped, SkipKind: check.SkipNotInLeg, Gap: "G3"},
				},
			},
		},
	}
}

func renderString(t *testing.T, doc check.MatrixDoc) string {
	t.Helper()
	var b strings.Builder
	if err := check.RenderMarkdown(&b, doc); err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}
	return b.String()
}

// The headline a release note carries must never state a leg's result without
// that leg's enforcement: an advisory PASS blocks no merge (gap G6, #1778).
func TestHeadline_CarriesEnforcementPerLegAndBothAggregates(t *testing.T) {
	line := check.Headline(completeDoc())
	for _, want := range []string{
		"ZECS v0.8.0", "temporal PASS (required)", "argo PASS (advisory)",
		"required legs: PASS", "all legs: PASS",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("headline %q does not carry %q", line, want)
		}
	}
	if strings.Contains(line, "INCOMPLETE") {
		t.Errorf("a complete run must not be flagged incomplete: %q", line)
	}
}

// A red advisory leg makes `result` FAIL while `enforced_result` stays PASS; the
// published line must show both, or it overstates what branch protection gives.
func TestHeadline_AdvisoryFailureDoesNotHideBehindRequiredPass(t *testing.T) {
	doc := completeDoc()
	doc.Legs[1].Result = check.ResultFail
	doc.Result = check.ResultFail
	line := check.Headline(doc)
	if !strings.Contains(line, "argo FAIL (advisory)") || !strings.Contains(line, "required legs: PASS") {
		t.Errorf("headline %q must report the advisory failure and the required-leg verdict", line)
	}
	if !strings.Contains(line, "all legs: FAIL") {
		t.Errorf("headline %q must report the all-legs verdict as FAIL", line)
	}
}

// The safeguard publication exists to protect: a run that skipped a leg must not
// read as a conformance result — in the FIRST line, not a footnote.
func TestHeadline_IncompleteRunSaysSoUpFront(t *testing.T) {
	doc := completeDoc()
	doc.Complete = false
	doc.Legs[1].Executed = false
	doc.Legs[1].Result = check.ResultNotRun
	doc.Result = check.ResultIncomplete
	line := check.Headline(doc)
	if !strings.Contains(line, "INCOMPLETE run — not a conformance result") {
		t.Errorf("headline %q must state that an incomplete run is not a conformance result", line)
	}
	if !strings.Contains(line, "argo NOT_RUN (advisory)") {
		t.Errorf("headline %q must show the leg that did not run", line)
	}
}

// A suite with no required leg has no merge-blocking verdict to report, and must
// not render the NONE sentinel as if it were a result.
func TestHeadline_NoRequiredLegReportsNoneDeclared(t *testing.T) {
	doc := completeDoc()
	doc.Legs[0].Enforcement = "advisory"
	doc.EnforcedResult = check.ResultNone
	if line := check.Headline(doc); !strings.Contains(line, "required legs: none declared") {
		t.Errorf("headline %q must not present the NONE sentinel as a verdict", line)
	}
}

func TestHeadline_NoLegsIsStatedNotImplied(t *testing.T) {
	doc := completeDoc()
	doc.Legs = nil
	if line := check.Headline(doc); !strings.Contains(line, "no engine legs recorded") {
		t.Errorf("headline %q must state that no leg was recorded", line)
	}
}

// The table is one column per leg, headed by enforcement, one row per scenario —
// and the two skips stay visibly different.
func TestRenderMarkdown_TableCarriesLegsScenariosAndSkipKinds(t *testing.T) {
	out := renderString(t, completeDoc())
	for _, want := range []string{
		"### Engine conformance — ZECS v0.8.0",
		"| Scenario | temporal (required) | argo (advisory) |",
		"| `echo-happy-path` | PASS | PASS |",
		"| `hello-world-outputs` | PASS | SKIPPED — not in leg (G3) |",
		"Revision `abc1234`",
		"[run](https://example.invalid/runs/1)",
		"- Results are per leg.",
		"`docs/conformance/README.md`",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendering does not contain %q\n---\n%s", want, out)
		}
	}
}

// `not_executed` is the dangerous skip — it is what made the leg INCOMPLETE — so
// it must not render identically to a scenario the leg was never meant to run.
func TestRenderMarkdown_IncompleteRunIsFlaggedAndNotExecutedIsDistinct(t *testing.T) {
	doc := completeDoc()
	doc.Complete = false
	doc.Legs[0].Complete = false
	doc.Legs[0].Result = check.ResultIncomplete
	doc.Legs[0].Scenarios[1] = check.ScenarioMatrix{
		ID: "hello-world-outputs", Result: check.ResultSkipped,
		SkipKind: check.SkipNotExecuted, Planned: true,
	}
	out := renderString(t, doc)
	if !strings.Contains(out, "Incomplete run — not a conformance result") {
		t.Errorf("an incomplete run must be flagged in the rendering:\n%s", out)
	}
	if !strings.Contains(out, "| `hello-world-outputs` | SKIPPED — not executed | SKIPPED — not in leg (G3) |") {
		t.Errorf("the two skip kinds must render differently:\n%s", out)
	}
}

// A leg missing a scenario the other legs carry must render as absent, never as
// a blank that could read like a pass.
func TestRenderMarkdown_ScenarioAbsentFromALeg(t *testing.T) {
	doc := completeDoc()
	doc.Legs[1].Scenarios = doc.Legs[1].Scenarios[:1]
	if out := renderString(t, doc); !strings.Contains(out, "| `hello-world-outputs` | PASS | — |") {
		t.Errorf("a scenario absent from a leg must render as absent:\n%s", out)
	}
}

func TestRenderMarkdown_NoLegs(t *testing.T) {
	doc := completeDoc()
	doc.Legs = nil
	if out := renderString(t, doc); !strings.Contains(out, "no engine legs recorded.") {
		t.Errorf("expected the no-leg statement, got:\n%s", out)
	}
}

// Round-trip: what the emitter writes is what the renderer reads. This is the
// property the publication path depends on — one document, two views.
func TestDecodeMatrix_RoundTripsAnEmittedDocument(t *testing.T) {
	var encoded strings.Builder
	if err := check.EncodeMatrix(&encoded, completeDoc()); err != nil {
		t.Fatalf("EncodeMatrix: %v", err)
	}
	doc, err := check.DecodeMatrix(strings.NewReader(encoded.String()))
	if err != nil {
		t.Fatalf("DecodeMatrix: %v", err)
	}
	if check.Headline(doc) != check.Headline(completeDoc()) {
		t.Errorf("headline changed across a round trip:\n got %q\nwant %q",
			check.Headline(doc), check.Headline(completeDoc()))
	}
}

// A document this build cannot render must fail loudly rather than be published
// as a table that silently drops meaning.
func TestDecodeMatrix_RejectsUnknownSchemaAndGarbage(t *testing.T) {
	for _, tc := range []struct{ name, raw, want string }{
		{"future schema", `{"schema_version":"2"}`, "schema_version"},
		{"no schema", `{"result":"PASS"}`, "schema_version"},
		{"not json", `nope`, "decode matrix"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := check.DecodeMatrix(strings.NewReader(tc.raw))
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}
