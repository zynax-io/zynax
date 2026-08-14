// SPDX-License-Identifier: Apache-2.0

package check

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ZECS result vocabulary. A cell is only ever PASS, FAIL or SKIPPED; a leg adds
// INCOMPLETE (it ran, but not everything it was meant to) and NOT_RUN (it never
// ran at all). Only an observed success renders PASS.
//
// The two skips carry different weight. SkipNotInLeg — the manifest says this leg
// does not run this scenario — is expected and gap-tracked, and does NOT make the
// leg incomplete. SkipNotExecuted — it was meant to run here and produced no
// success or failure — does: that is what stops a partial run being published as
// a complete one.
const (
	ResultPass       = "PASS"
	ResultFail       = "FAIL"
	ResultSkipped    = "SKIPPED"
	ResultIncomplete = "INCOMPLETE"
	ResultNotRun     = "NOT_RUN"
	ResultNone       = "NONE"

	SkipNotInLeg    = "not_in_leg"
	SkipNotExecuted = "not_executed"

	// Observed step outcomes: GitHub reports success | failure | cancelled |
	// skipped, and only the first two are evidence.
	outcomeSuccess = "success"
	outcomeFailure = "failure"

	enforcementRequired = "required"
	enforcementAdvisory = "advisory"
	enforcementUnknown  = "unknown"

	matrixSchemaVersion = "1" // bumped on any breaking change to the JSON shape
	outcomeLegKey       = "leg"
)

// ScenarioMatrix is one cell: what one leg did with one ZECS scenario.
type ScenarioMatrix struct {
	ID       string   `json:"id"`
	Result   string   `json:"result"`
	Planned  bool     `json:"planned"`
	SkipKind string   `json:"skip_kind,omitempty"`
	Reason   string   `json:"reason,omitempty"`
	Gap      string   `json:"gap,omitempty"`
	Runner   string   `json:"runner,omitempty"`
	CIStep   string   `json:"ci_step,omitempty"`
	Observed string   `json:"observed,omitempty"`
	Asserts  []string `json:"asserts,omitempty"`
}

// LegMatrix is one engine leg's row. Every engine the engine-adapter can be
// configured with gets one, executed or not — a leg is never absent.
type LegMatrix struct {
	Engine      string           `json:"engine"`
	Enforcement string           `json:"enforcement"`
	Executed    bool             `json:"executed"`
	Complete    bool             `json:"complete"`
	Result      string           `json:"result"`
	Scenarios   []ScenarioMatrix `json:"scenarios"`
}

// MatrixDoc is the machine-readable ZECS result artifact (#1774). Read
// `complete` before `result`: false means a leg, or a scenario a leg was meant to
// run, produced no observation — the document reports what happened and is not a
// conformance result for the suite. `enforced_result` aggregates only the legs
// whose e2e check is required on main (gap G6). See docs/conformance/README.md §10.
type MatrixDoc struct {
	SchemaVersion  string      `json:"schema_version"`
	Suite          string      `json:"suite"`
	ShortName      string      `json:"short_name"`
	Version        string      `json:"version"`
	Definition     string      `json:"definition"`
	GeneratedAt    string      `json:"generated_at"`
	Revision       string      `json:"revision,omitempty"`
	RunURL         string      `json:"run_url,omitempty"`
	Result         string      `json:"result"`
	EnforcedResult string      `json:"enforced_result"`
	Complete       bool        `json:"complete"`
	Legs           []LegMatrix `json:"legs"`
	Notes          []string    `json:"notes"`
}

// MatrixOptions selects the repository and where the observations come from:
// recorded step outcomes (CI, one file per leg) or a live local run of one leg.
// Both feed the same builder, so both emit the same shape.
type MatrixOptions struct {
	Root        string
	OutcomesDir string
	Leg         string
	Run         bool
	Revision    string
	RunURL      string
	Log         io.Writer
}

// ConformanceMatrix builds the ZECS per-engine matrix for the repo at Root. The
// leg set is enumerated from the engines the engine-adapter can be configured
// with (ADR-015) — the same source of truth the drift guard uses — so engine N+1
// appears with no edit here and no engine name in this file. It reports; it
// never gates.
func ConformanceMatrix(opts MatrixOptions) (MatrixDoc, error) {
	man, err := loadZECSManifest(filepath.Join(opts.Root, zecsManifestPath))
	if err != nil {
		return MatrixDoc{}, err
	}
	engines, err := discoverSelectableEngines(filepath.Join(opts.Root, engineMainPath))
	if err != nil {
		return MatrixDoc{}, err
	}
	observed, err := collectObservations(opts, man, engines)
	if err != nil {
		return MatrixDoc{}, err
	}
	return buildMatrixDoc(man, sortedKeysOf(engines), observed, opts), nil
}

// collectObservations resolves the two input modes to leg -> key -> outcome.
func collectObservations(opts MatrixOptions, man zecsManifest, engines map[string]bool) (map[string]map[string]string, error) {
	switch {
	case opts.OutcomesDir != "" && opts.Run:
		return nil, errors.New("conformance matrix: --outcomes and --run are mutually exclusive")
	case opts.OutcomesDir != "":
		return loadOutcomeFiles(opts.OutcomesDir, engines)
	case !opts.Run:
		return nil, errors.New(
			"conformance matrix: no observations — pass --outcomes <dir> (recorded run) or --run --leg <engine>")
	case !engines[opts.Leg]:
		return nil, fmt.Errorf(
			"conformance matrix: --run needs --leg <engine>; engines the engine-adapter can be configured with: %v",
			sortedKeysOf(engines))
	}
	log := opts.Log
	if log == nil {
		log = os.Stderr
	}
	return map[string]map[string]string{opts.Leg: runLegRunners(opts.Root, opts.Leg, man, log)}, nil
}

// loadOutcomeFiles reads recorded step outcomes, one file per executed leg.
//
// An EMPTY directory is not an error — no leg reported, so every leg renders
// NOT_RUN, which is the honest result of a run whose legs all died. A MISSING
// directory is: it means the caller pointed at nothing, and silently reporting
// an all-NOT_RUN matrix for a mistyped path would look exactly like real evidence
// of a dead run.
func loadOutcomeFiles(dir string, engines map[string]bool) (map[string]map[string]string, error) {
	observed := map[string]map[string]string{}
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("conformance matrix: outcomes directory %q: %w", dir, err)
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, rerr := os.ReadFile(path) //nolint:gosec // path from a walk of the caller's directory
		if rerr != nil {
			return rerr
		}
		outcomes := parseOutcomes(string(raw))
		leg := outcomes[outcomeLegKey]
		delete(outcomes, outcomeLegKey)
		switch {
		case leg == "":
			return fmt.Errorf("conformance matrix: %q: no %q line naming the engine leg", path, outcomeLegKey+"=<engine>")
		case !engines[leg]:
			return fmt.Errorf(
				"conformance matrix: %q: leg %q is not an engine the engine-adapter can be configured with", path, leg)
		case observed[leg] != nil:
			return fmt.Errorf("conformance matrix: %q: a second outcome file for leg %q", path, leg)
		}
		observed[leg] = outcomes
		return nil
	})
	return observed, err
}

// parseOutcomes reads `key=value` lines (blank lines and `#` comments ignored).
func parseOutcomes(raw string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if key, value, found := strings.Cut(line, "="); found {
			out[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return out
}

// runLegRunners executes the runner of every scenario the manifest marks `run`
// for this leg and records success/failure from the exit status. Not a second
// harness: these are the same scripts/e2e/*.sh the e2e workflow runs, invoked
// against a cluster the caller brought up.
func runLegRunners(root, leg string, man zecsManifest, log io.Writer) map[string]string {
	outcomes, byRunner := map[string]string{}, map[string]string{}
	for _, sc := range man.Scenarios {
		entry, ok := sc.Legs[leg]
		if !ok || entry.Status != legStatusRun || entry.Runner == "" {
			continue
		}
		key := observationKey(sc, entry)
		if prior, done := byRunner[entry.Runner]; done {
			outcomes[key] = prior
			continue
		}
		_, _ = fmt.Fprintf(log, "\n=== ZECS %s [%s] -> %s\n", sc.ID, leg, entry.Runner)
		cmd := exec.Command("bash", filepath.Join(root, filepath.FromSlash(entry.Runner))) //nolint:gosec // runner path comes from the ZECS manifest, which the drift guard pins to real repo files
		cmd.Dir, cmd.Stdout, cmd.Stderr = root, log, log
		cmd.Env = append(os.Environ(), "E2E_ENGINE="+leg)
		outcome := outcomeSuccess
		if err := cmd.Run(); err != nil {
			outcome = outcomeFailure
			_, _ = fmt.Fprintf(log, "=== ZECS %s [%s] FAILED: %v\n", sc.ID, leg, err)
		}
		outcomes[key], byRunner[entry.Runner] = outcome, outcome
	}
	return outcomes
}

// observationKey is the shared key between what CI records and what the matrix
// reads: the workflow step id, or the scenario id when the leg declares none.
func observationKey(sc zecsScenario, entry zecsLeg) string {
	if entry.CIStep != "" {
		return entry.CIStep
	}
	return sc.ID
}

func buildMatrixDoc(man zecsManifest, engines []string, observed map[string]map[string]string, opts MatrixOptions) MatrixDoc {
	doc := MatrixDoc{
		SchemaVersion: matrixSchemaVersion,
		Suite:         man.Suite,
		ShortName:     man.ShortName,
		Version:       man.Version,
		Definition:    man.Definition,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Revision:      opts.Revision,
		RunURL:        opts.RunURL,
		Complete:      true,
	}
	var advisory []string
	anyFail, anyNotPass := false, false
	reqFail, reqNotPass, anyRequired := false, false, false
	for _, engine := range engines {
		leg := buildLegMatrix(man, engine, observed)
		doc.Legs = append(doc.Legs, leg)
		if !leg.Executed || !leg.Complete {
			doc.Complete = false
		}
		anyFail = anyFail || leg.Result == ResultFail
		anyNotPass = anyNotPass || leg.Result != ResultPass
		if leg.Enforcement != enforcementRequired {
			advisory = append(advisory, leg.Engine)
			continue
		}
		anyRequired = true
		reqFail = reqFail || leg.Result == ResultFail
		reqNotPass = reqNotPass || leg.Result != ResultPass
	}
	doc.Result = verdict(anyFail, anyNotPass)
	doc.EnforcedResult = ResultNone
	if anyRequired {
		doc.EnforcedResult = verdict(reqFail, reqNotPass)
	}
	doc.Notes = matrixNotes(man, advisory)
	return doc
}

// buildLegMatrix renders one leg. A leg with no observations is NOT_RUN and
// incomplete — never silently absent, and never PASS.
func buildLegMatrix(man zecsManifest, engine string, observed map[string]map[string]string) LegMatrix {
	steps, executed := observed[engine]
	leg := LegMatrix{Engine: engine, Enforcement: enforcementOf(man, engine), Executed: executed}
	failed, notExecuted := false, false
	for _, sc := range man.Scenarios {
		cell := buildScenarioCell(sc, engine, steps, executed)
		failed = failed || cell.Result == ResultFail
		notExecuted = notExecuted || cell.SkipKind == SkipNotExecuted
		leg.Scenarios = append(leg.Scenarios, cell)
	}
	leg.Complete = executed && !notExecuted
	switch {
	case !executed:
		leg.Result = ResultNotRun
	case failed:
		leg.Result = ResultFail
	case notExecuted:
		leg.Result = ResultIncomplete
	default:
		leg.Result = ResultPass
	}
	return leg
}

// buildScenarioCell decides one cell. Every branch that is not an observed
// success yields FAIL or SKIPPED — there is no path to PASS without evidence.
func buildScenarioCell(sc zecsScenario, engine string, steps map[string]string, legExecuted bool) ScenarioMatrix {
	cell := ScenarioMatrix{ID: sc.ID}
	entry, declared := sc.Legs[engine]
	if !declared || (entry.Status != legStatusRun && entry.Status != legStatusNotRun) {
		return skip(cell, SkipNotExecuted, fmt.Sprintf(
			"no 'run'/'not_run' entry for this leg in %s — run `make check-conformance`", zecsManifestPath))
	}
	if entry.Status == legStatusNotRun {
		cell.Gap = entry.Gap
		return skip(cell, SkipNotInLeg, collapse(entry.Reason))
	}

	cell.Planned = true
	cell.Runner, cell.CIStep, cell.Asserts = entry.Runner, entry.CIStep, entry.Asserts
	if !legExecuted {
		return skip(cell, SkipNotExecuted, "the engine leg did not execute in this run")
	}
	key := observationKey(sc, entry)
	outcome, recorded := steps[key]
	cell.Observed = outcome
	switch {
	case outcome == outcomeSuccess:
		cell.Result = ResultPass
	case outcome == outcomeFailure:
		cell.Result = ResultFail
	case !recorded || outcome == "":
		return skip(cell, SkipNotExecuted, fmt.Sprintf("the leg recorded no outcome for %q", key))
	default:
		return skip(cell, SkipNotExecuted, fmt.Sprintf(
			"recorded outcome %q for %q is neither %q nor %q", outcome, key, outcomeSuccess, outcomeFailure))
	}
	return cell
}

func skip(cell ScenarioMatrix, kind, reason string) ScenarioMatrix {
	cell.Result, cell.SkipKind, cell.Reason = ResultSkipped, kind, reason
	return cell
}

func verdict(anyFail, anyNotPass bool) string {
	switch {
	case anyFail:
		return ResultFail
	case anyNotPass:
		return ResultIncomplete
	default:
		return ResultPass
	}
}

// enforcementOf reports whether this leg's e2e check blocks a merge on main. An
// undeclared leg is "unknown" and is never counted as required — a matrix must
// not claim authority branch protection does not give it (gap G6).
func enforcementOf(man zecsManifest, engine string) string {
	if e := strings.TrimSpace(man.Enforcement[engine]); e != "" {
		return e
	}
	return enforcementUnknown
}

// matrixNotes carries the manifest's own caveats into every result document, plus
// the one that depends on the run: which legs block nothing (gap G6).
func matrixNotes(man zecsManifest, advisory []string) []string {
	notes := make([]string, 0, len(man.MatrixNotes)+1)
	for _, note := range man.MatrixNotes {
		notes = append(notes, collapse(note))
	}
	if len(advisory) > 0 {
		sort.Strings(advisory)
		notes = append(notes, fmt.Sprintf(
			"Leg(s) %s are advisory: a non-PASS result there does not block a merge on main (gap G6 in %s).",
			strings.Join(advisory, ", "), man.Definition))
	}
	return notes
}

// EncodeMatrix writes the matrix as indented JSON — the primary artifact.
func EncodeMatrix(w io.Writer, doc MatrixDoc) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("conformance matrix: encode: %w", err)
	}
	return nil
}

// Summary is the one-line verdict for a log. The JSON document is the artifact;
// the published renderings live in matrix_render.go (#1775).
func Summary(doc MatrixDoc) string {
	legs := make([]string, 0, len(doc.Legs))
	for _, leg := range doc.Legs {
		legs = append(legs, fmt.Sprintf("%s(%s)=%s", leg.Engine, leg.Enforcement, leg.Result))
	}
	return fmt.Sprintf("%s %s: result=%s complete=%t required-legs=%s · %s",
		doc.ShortName, doc.Version, doc.Result, doc.Complete, doc.EnforcedResult, strings.Join(legs, " "))
}

// collapse folds a YAML folded scalar into one line.
func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
