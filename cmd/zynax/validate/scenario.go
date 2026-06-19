// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ScenarioIndexFile is the conventional file name of a scenario index inside a
// scenario directory (the manifest-set convention, A2 / ADR-028).
const ScenarioIndexFile = "scenario.yaml"

// ScenarioMember is one expanded member of a scenario: its declared kind and the
// absolute path to the member manifest, resolved relative to the index file.
type ScenarioMember struct {
	ID   string
	Kind string
	Path string // absolute path to the member manifest
}

// scenarioIndex is the on-disk shape of a scenario index file (kind: Scenario).
// Only the fields the CLI acts on are decoded; the schema owns full validation.
type scenarioIndex struct {
	Kind string `yaml:"kind"`
	Spec struct {
		Members []struct {
			ID   string `yaml:"id"`
			Kind string `yaml:"kind"`
			File string `yaml:"file"`
		} `yaml:"members"`
		ApplyOrder []string `yaml:"apply_order"`
	} `yaml:"spec"`
}

// ResolveScenarioIndex returns the absolute path to a scenario index given either
// the index file itself or the directory that contains it. A non-Scenario or
// missing target yields ("", false, nil): callers treat the path as an ordinary
// single manifest. I/O errors are returned.
func ResolveScenarioIndex(path string) (indexPath string, ok bool, err error) {
	info, statErr := os.Stat(path)
	if statErr != nil {
		return "", false, fmt.Errorf("scenario: stat %q: %w", path, statErr)
	}
	if info.IsDir() {
		candidate := filepath.Join(path, ScenarioIndexFile)
		if _, e := os.Stat(candidate); e != nil {
			return "", false, nil // a plain directory with no index — not a scenario
		}
		return candidate, true, nil
	}
	// A file: it is a scenario index only when its kind is Scenario.
	kind, e := readKind(path)
	if e != nil {
		return "", false, nil // let the manifest validator report parse errors
	}
	return path, kind == "Scenario", nil
}

// ExpandScenario reads the scenario index at indexPath and returns its members
// ordered by spec.apply_order, with each member's file resolved (and confined)
// relative to the index directory. It enforces the cross-field invariants the
// JSON Schema cannot express: apply_order references only declared members, each
// member appears exactly once, and member files do not escape the directory.
func ExpandScenario(indexPath string) ([]ScenarioMember, error) {
	raw, err := os.ReadFile(indexPath) //nolint:gosec // indexPath is caller-supplied scenario path
	if err != nil {
		return nil, fmt.Errorf("scenario: read %q: %w", indexPath, err)
	}
	var idx scenarioIndex
	if err := yaml.Unmarshal(raw, &idx); err != nil {
		return nil, fmt.Errorf("scenario: parse %q: %w", indexPath, err)
	}

	dir := filepath.Dir(indexPath)
	byID := make(map[string]ScenarioMember, len(idx.Spec.Members))
	for _, m := range idx.Spec.Members {
		if m.ID == "" {
			return nil, fmt.Errorf("scenario %q: member with empty id", indexPath)
		}
		if _, dup := byID[m.ID]; dup {
			return nil, fmt.Errorf("scenario %q: duplicate member id %q", indexPath, m.ID)
		}
		abs, err := resolveMemberPath(dir, m.File)
		if err != nil {
			return nil, fmt.Errorf("scenario %q: member %q: %w", indexPath, m.ID, err)
		}
		byID[m.ID] = ScenarioMember{ID: m.ID, Kind: m.Kind, Path: abs}
	}

	if len(idx.Spec.ApplyOrder) == 0 {
		return nil, fmt.Errorf("scenario %q: apply_order is empty", indexPath)
	}
	seen := make(map[string]bool, len(idx.Spec.ApplyOrder))
	ordered := make([]ScenarioMember, 0, len(idx.Spec.ApplyOrder))
	for _, id := range idx.Spec.ApplyOrder {
		m, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("scenario %q: apply_order references unknown member id %q", indexPath, id)
		}
		if seen[id] {
			return nil, fmt.Errorf("scenario %q: apply_order lists member id %q more than once", indexPath, id)
		}
		seen[id] = true
		ordered = append(ordered, m)
	}
	return ordered, nil
}

// ScenarioFile validates a scenario index plus all of its members. It first
// validates the index file against scenario.schema.json, then expands it and runs
// the full per-member pipeline (File: schema + data-flow) against each member's
// own existing schema. Every error is prefixed with its file so callers can tell
// index errors from member errors. Returns (nil, nil) when everything is valid.
func ScenarioFile(indexPath, schemaDir string) ([]ValidationError, error) {
	idxErrs, err := Manifest(indexPath, schemaDir)
	if err != nil {
		return nil, err
	}
	// If the index itself is invalid, expansion is unreliable — report and stop.
	if len(idxErrs) > 0 {
		return idxErrs, nil
	}

	members, err := ExpandScenario(indexPath)
	if err != nil {
		return single(indexPath, "/spec", err.Error()), nil
	}

	// Resolve the declarative context-injection block (#1387) once for the
	// scenario: parse it (rejecting any routing/provider field — data-only),
	// read its file sources, and apply the max_tokens cap. A block error is an
	// index error.
	block, err := ParseContextBlock(indexPath)
	if err != nil {
		return single(indexPath, "/spec/context", err.Error()), nil
	}
	values, err := ResolveContext(block, filepath.Dir(indexPath))
	if err != nil {
		return single(indexPath, "/spec/context", err.Error()), nil
	}

	var out []ValidationError
	for _, m := range members {
		// For a Workflow member, bind the resolved context into its
		// {{ .ctx.* }} references and validate the BOUND manifest, so an
		// unresolved key (one the block does not supply) fails validation.
		if m.Kind == kindWorkflow {
			memberErrs, err := validateBoundWorkflow(m.Path, schemaDir, values)
			if err != nil {
				return nil, err
			}
			out = append(out, memberErrs...)
			continue
		}
		memberErrs, err := File(m.Path, schemaDir)
		if err != nil {
			return nil, err
		}
		out = append(out, memberErrs...)
	}
	return out, nil
}

// validateBoundWorkflow renders the Workflow member's {{ .ctx.* }} references
// with the resolved context values and validates the result. When the member
// declares no {{ .ctx.* }} references the binding is a no-op. A reference the
// context block does not supply surfaces here as a bind error (the file is
// reported so callers can locate it).
func validateBoundWorkflow(path, schemaDir string, values map[string]string) ([]ValidationError, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is confined by ExpandScenario
	if err != nil {
		return nil, fmt.Errorf("scenario: read member %q: %w", path, err)
	}
	if !strings.Contains(string(raw), "{{") {
		// No template references — validate the file as-is.
		return File(path, schemaDir)
	}
	bound, err := BindContextIntoWorkflow(raw, values)
	if err != nil {
		return single(path, "/spec", err.Error()), nil
	}
	return FileBytes(path, bound, schemaDir)
}

// resolveMemberPath joins a member file to the scenario directory and rejects any
// path that escapes that directory (path traversal / absolute paths).
func resolveMemberPath(dir, file string) (string, error) {
	if file == "" {
		return "", fmt.Errorf("member file is empty")
	}
	if filepath.IsAbs(file) {
		return "", fmt.Errorf("member file %q must be relative to the index directory", file)
	}
	cleanDir := filepath.Clean(dir)
	abs := filepath.Clean(filepath.Join(cleanDir, file))
	rel, err := filepath.Rel(cleanDir, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("member file %q escapes the scenario directory", file)
	}
	return abs, nil
}

// readKind decodes only the top-level kind field of a YAML manifest.
func readKind(path string) (string, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is caller-supplied manifest path
	if err != nil {
		return "", fmt.Errorf("scenario: read %q: %w", path, err)
	}
	var head struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(raw, &head); err != nil {
		return "", fmt.Errorf("scenario: parse %q: %w", path, err)
	}
	return head.Kind, nil
}
