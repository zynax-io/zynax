// SPDX-License-Identifier: Apache-2.0

package check

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ZECS drift-guard inputs. The suite definition NEVER authors the engine-leg set
// (ADR-015): it mirrors the engines the engine-adapter can be configured with, and
// this guard fails when the mirror drifts. No engine name appears in this file.
const (
	zecsManifestPath = "docs/conformance/scenarios.yaml"
	engineMainPath   = "services/engine-adapter/cmd/engine-adapter/main.go"
	e2eWorkflowPath  = ".github/workflows/e2e-smoke.yml"

	// enginePrefix is the identifier prefix of the engine-name constants in
	// engineMainPath ("the ONLY place engine names appear outside of the
	// engine-selection switch").
	enginePrefix = "engine"

	legStatusRun    = "run"
	legStatusNotRun = "not_run"

	sourceKindCorpus = "corpus"
)

type zecsSource struct {
	Kind string `yaml:"kind"`
	Path string `yaml:"path"`
}

type zecsLeg struct {
	Status  string   `yaml:"status"`
	Runner  string   `yaml:"runner"`
	Reason  string   `yaml:"reason"`
	Gap     string   `yaml:"gap"`
	CIStep  string   `yaml:"ci_step"`
	Asserts []string `yaml:"asserts"`
}

type zecsScenario struct {
	ID     string             `yaml:"id"`
	Source zecsSource         `yaml:"source"`
	Legs   map[string]zecsLeg `yaml:"legs"`
}

type zecsNonMember struct {
	Path   string `yaml:"path"`
	Reason string `yaml:"reason"`
}

type zecsManifest struct {
	Suite       string            `yaml:"suite"`
	ShortName   string            `yaml:"short_name"`
	Version     string            `yaml:"version"`
	Definition  string            `yaml:"definition"`
	Corpus      string            `yaml:"corpus"`
	Legs        []string          `yaml:"legs"`
	Enforcement map[string]string `yaml:"enforcement"`
	MatrixNotes []string          `yaml:"matrix_notes"`
	Scenarios   []zecsScenario    `yaml:"scenarios"`
	NonMembers  []zecsNonMember   `yaml:"non_members"`
}

// Conformance runs the ZECS membership drift guard against the repo at root
// (docs/conformance/README.md §8). It reconciles docs/conformance/scenarios.yaml
// with three surfaces — the spec/workflows/examples corpus, the engines the
// engine-adapter can be configured with, and the e2e matrix legs — and returns the
// list of problems (empty == no drift) plus the number of declared scenarios.
//
// It is NOT a conformance run: it executes no workflow and reports no per-engine
// pass/fail. It only proves the suite definition is internally consistent with the
// repository. A returned error is operational (an input is missing or unparseable),
// not a drift finding.
func Conformance(root string) (problems []string, count int, err error) {
	man, err := loadZECSManifest(filepath.Join(root, zecsManifestPath))
	if err != nil {
		return nil, 0, err
	}
	count = len(man.Scenarios)

	engines, err := discoverSelectableEngines(filepath.Join(root, engineMainPath))
	if err != nil {
		return nil, count, err
	}
	matrixLegs, stepIDs, err := discoverE2EJob(filepath.Join(root, e2eWorkflowPath))
	if err != nil {
		return nil, count, err
	}
	corpus, err := discoverCorpusWorkflows(root, man.Corpus)
	if err != nil {
		return nil, count, err
	}

	legs := stringSet(man.Legs)
	problems = append(problems, checkLegMirror(legs, engines, matrixLegs)...)
	problems = append(problems, checkLegEnforcement(man, legs)...)
	problems = append(problems, checkScenarios(root, man, legs, stepIDs)...)
	problems = append(problems, checkCorpusMembership(man, corpus)...)
	return problems, count, nil
}

// checkLegMirror asserts the declared leg set equals BOTH the engines selectable in
// the engine-adapter and the e2e matrix legs — so engine N+1 makes the suite red
// until it is really run, without any engine name being hardcoded here.
func checkLegMirror(legs, engines, matrixLegs map[string]bool) []string {
	var problems []string
	if len(legs) == 0 {
		problems = append(problems, fmt.Sprintf("%s: declares no engine legs", zecsManifestPath))
	}
	if missing := sortedDiff(engines, legs); len(missing) > 0 {
		problems = append(problems, fmt.Sprintf(
			"engines selectable in %s but not declared as ZECS legs: %v — add the leg or record why it is unconformant",
			engineMainPath, missing))
	}
	if extra := sortedDiff(legs, engines); len(extra) > 0 {
		problems = append(problems, fmt.Sprintf(
			"ZECS legs that are not selectable engines in %s: %v", engineMainPath, extra))
	}
	if missing := sortedDiff(legs, matrixLegs); len(missing) > 0 {
		problems = append(problems, fmt.Sprintf(
			"ZECS legs absent from the e2e matrix in %s: %v — the suite claims a leg the runner never runs",
			e2eWorkflowPath, missing))
	}
	if extra := sortedDiff(matrixLegs, legs); len(extra) > 0 {
		problems = append(problems, fmt.Sprintf(
			"e2e matrix legs in %s not declared by ZECS: %v", e2eWorkflowPath, extra))
	}
	return problems
}

// checkLegEnforcement asserts every declared leg states whether its e2e check is
// required or advisory on main (README §3, gap G6). The published matrix carries
// this per leg, so a new leg must not inherit an authority claim by omission.
func checkLegEnforcement(man zecsManifest, legs map[string]bool) []string {
	var problems []string
	for _, leg := range sortedKeysOf(legs) {
		switch e := man.Enforcement[leg]; e {
		case enforcementRequired, enforcementAdvisory:
		case "":
			problems = append(problems, fmt.Sprintf(
				"%s: leg %q has no 'enforcement' entry — a published matrix must state whether the leg blocks a merge",
				zecsManifestPath, leg))
		default:
			problems = append(problems, fmt.Sprintf("%s: enforcement[%s] = %q is not %q or %q",
				zecsManifestPath, leg, e, enforcementRequired, enforcementAdvisory))
		}
	}
	return problems
}

// checkScenarios asserts every scenario resolves to a real file, has a unique id,
// and carries an explicit entry for EVERY declared leg (a leg that does not run must
// say so with a reason — omission would let a skipped leg read as a pass).
func checkScenarios(root string, man zecsManifest, legs, stepIDs map[string]bool) []string {
	var problems []string
	seen := map[string]bool{}
	for i, sc := range man.Scenarios {
		if sc.ID == "" {
			problems = append(problems, fmt.Sprintf("scenario #%d: missing 'id'", i))
			continue
		}
		if seen[sc.ID] {
			problems = append(problems, fmt.Sprintf("%s: declared more than once", sc.ID))
		}
		seen[sc.ID] = true

		if sc.Source.Path == "" {
			problems = append(problems, fmt.Sprintf("%s: missing 'source.path'", sc.ID))
		} else if !fileExists(root, sc.Source.Path) {
			problems = append(problems, fmt.Sprintf(
				"%s: source.path %q does not exist", sc.ID, sc.Source.Path))
		}
		problems = append(problems, checkScenarioLegs(root, sc, legs, stepIDs)...)
	}
	return problems
}

// checkScenarioLegs applies the per-leg rules for one scenario.
func checkScenarioLegs(root string, sc zecsScenario, legs, stepIDs map[string]bool) []string {
	var problems []string
	for _, leg := range sortedKeysOf(legs) {
		entry, ok := sc.Legs[leg]
		if !ok {
			problems = append(problems, fmt.Sprintf(
				"%s: no entry for leg %q — every leg must be explicit (a skipped leg is SKIPPED, never PASS)",
				sc.ID, leg))
			continue
		}
		switch entry.Status {
		case legStatusRun:
			if entry.Runner == "" {
				problems = append(problems, fmt.Sprintf("%s[%s]: status 'run' with no 'runner'", sc.ID, leg))
			} else if !fileExists(root, entry.Runner) {
				problems = append(problems, fmt.Sprintf(
					"%s[%s]: runner %q does not exist", sc.ID, leg, entry.Runner))
			}
			// The matrix reads this leg's outcome under its ci_step key (#1774).
			// Resolving that key against the workflow's real step ids is what stops
			// a renamed step from surfacing only later, as a `not_executed` cell in
			// a PUBLISHED matrix (#1775): publication is exactly where that failure
			// becomes externally visible and hard to retract.
			switch {
			case entry.CIStep == "":
				problems = append(problems, fmt.Sprintf(
					"%s[%s]: status 'run' with no 'ci_step' — the matrix would have no outcome to read", sc.ID, leg))
			case !stepIDs[entry.CIStep]:
				problems = append(problems, fmt.Sprintf(
					"%s[%s]: ci_step %q is not the id of any step in %s — the matrix would render this scenario 'not_executed' and the leg INCOMPLETE",
					sc.ID, leg, entry.CIStep, e2eWorkflowPath))
			}
		case legStatusNotRun:
			if strings.TrimSpace(entry.Reason) == "" {
				problems = append(problems, fmt.Sprintf("%s[%s]: status 'not_run' with no 'reason'", sc.ID, leg))
			}
		default:
			problems = append(problems, fmt.Sprintf(
				"%s[%s]: status %q is not %q or %q", sc.ID, leg, entry.Status, legStatusRun, legStatusNotRun))
		}
	}
	for leg := range sc.Legs {
		if !legs[leg] {
			problems = append(problems, fmt.Sprintf("%s: entry for unknown leg %q", sc.ID, leg))
		}
	}
	return problems
}

// checkCorpusMembership asserts every kind:Workflow manifest in the corpus is either
// a scenario source or an explicitly reasoned non-member — one corpus, annotated, so
// nothing drops out of the suite silently.
func checkCorpusMembership(man zecsManifest, corpus map[string]bool) []string {
	var problems []string
	members := map[string]bool{}
	for _, sc := range man.Scenarios {
		if sc.Source.Kind == sourceKindCorpus && sc.Source.Path != "" {
			members[sc.Source.Path] = true
		}
	}
	declared := map[string]bool{}
	for _, nm := range man.NonMembers {
		if nm.Path == "" {
			problems = append(problems, "non_members: entry with no 'path'")
			continue
		}
		if strings.TrimSpace(nm.Reason) == "" {
			problems = append(problems, fmt.Sprintf("non_members[%s]: missing 'reason'", nm.Path))
		}
		if !corpus[nm.Path] {
			problems = append(problems, fmt.Sprintf(
				"non_members[%s]: not a kind:Workflow manifest in %s", nm.Path, man.Corpus))
		}
		if members[nm.Path] {
			problems = append(problems, fmt.Sprintf(
				"non_members[%s]: also declared as a scenario source", nm.Path))
		}
		declared[nm.Path] = true
	}
	for _, path := range sortedKeysOf(corpus) {
		if !members[path] && !declared[path] {
			problems = append(problems, fmt.Sprintf(
				"%s: kind:Workflow manifest is neither a ZECS scenario nor a declared non_member", path))
		}
	}
	for _, sc := range man.Scenarios {
		if sc.Source.Kind == sourceKindCorpus && sc.Source.Path != "" && !corpus[sc.Source.Path] {
			problems = append(problems, fmt.Sprintf(
				"%s: source.kind 'corpus' but %q is not a kind:Workflow manifest in %s",
				sc.ID, sc.Source.Path, man.Corpus))
		}
	}
	return problems
}

// loadZECSManifest reads and parses the membership manifest.
func loadZECSManifest(path string) (zecsManifest, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // caller-supplied repo path
	if err != nil {
		return zecsManifest{}, fmt.Errorf("check conformance: read %q: %w", path, err)
	}
	var man zecsManifest
	if err := yaml.Unmarshal(raw, &man); err != nil {
		return zecsManifest{}, fmt.Errorf("check conformance: parse %q: %w", path, err)
	}
	if man.Corpus == "" {
		return zecsManifest{}, fmt.Errorf("check conformance: %q: missing 'corpus'", path)
	}
	if len(man.Scenarios) == 0 {
		return zecsManifest{}, fmt.Errorf("check conformance: %q: missing or empty 'scenarios'", path)
	}
	return man, nil
}

// discoverSelectableEngines parses the engine-adapter entrypoint and returns the
// engine names it can be configured with. Zero discovered engines is an error, not
// a vacuous pass: it means the constants moved and the guard has gone blind.
func discoverSelectableEngines(path string) (map[string]bool, error) {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("check conformance: parse %q: %w", path, err)
	}
	engines := map[string]bool{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if !strings.HasPrefix(name.Name, enginePrefix) || i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				if v, uerr := strconv.Unquote(lit.Value); uerr == nil && v != "" {
					engines[v] = true
				}
			}
		}
	}
	if len(engines) == 0 {
		return nil, fmt.Errorf(
			"check conformance: no %q* engine-name constants found in %q — the leg source of truth moved",
			enginePrefix, path)
	}
	return engines, nil
}

// discoverE2EJob returns the two things the suite mirrors from the e2e workflow:
// jobs.e2e.strategy.matrix.engine (the leg set) and the ids of the job's steps
// (the keys the result matrix reads per-scenario outcomes under).
func discoverE2EJob(path string) (legs, stepIDs map[string]bool, err error) {
	raw, err := os.ReadFile(path) //nolint:gosec // caller-supplied repo path
	if err != nil {
		return nil, nil, fmt.Errorf("check conformance: read %q: %w", path, err)
	}
	var wf struct {
		Jobs struct {
			E2E struct {
				Strategy struct {
					Matrix struct {
						Engine []string `yaml:"engine"`
					} `yaml:"matrix"`
				} `yaml:"strategy"`
				Steps []struct {
					ID string `yaml:"id"`
				} `yaml:"steps"`
			} `yaml:"e2e"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		return nil, nil, fmt.Errorf("check conformance: parse %q: %w", path, err)
	}
	engines := wf.Jobs.E2E.Strategy.Matrix.Engine
	if len(engines) == 0 {
		return nil, nil, fmt.Errorf(
			"check conformance: %q: jobs.e2e.strategy.matrix.engine is empty — the runner's leg list moved", path)
	}
	stepIDs = map[string]bool{}
	for _, step := range wf.Jobs.E2E.Steps {
		if step.ID != "" {
			stepIDs[step.ID] = true
		}
	}
	return stringSet(engines), stepIDs, nil
}

// discoverCorpusWorkflows returns the repo-relative paths of the kind:Workflow
// manifests in the corpus directory (AgentDef and Policy manifests are not
// workflows and are out of scope by construction).
func discoverCorpusWorkflows(root, corpus string) (map[string]bool, error) {
	matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(corpus), "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("check conformance: glob %q: %w", corpus, err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("check conformance: corpus %q holds no manifests", corpus)
	}
	set := map[string]bool{}
	for _, m := range matches {
		raw, rerr := os.ReadFile(m) //nolint:gosec // path from a glob of the corpus dir
		if rerr != nil {
			return nil, fmt.Errorf("check conformance: read %q: %w", m, rerr)
		}
		var doc struct {
			Kind string `yaml:"kind"`
		}
		if uerr := yaml.Unmarshal(raw, &doc); uerr != nil {
			return nil, fmt.Errorf("check conformance: parse %q: %w", m, uerr)
		}
		if doc.Kind == "Workflow" {
			set[corpus+"/"+filepath.Base(m)] = true
		}
	}
	return set, nil
}

func fileExists(root, relPath string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(relPath)))
	return err == nil
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}

// sortedKeysOf returns the sorted keys of a set.
func sortedKeysOf(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
