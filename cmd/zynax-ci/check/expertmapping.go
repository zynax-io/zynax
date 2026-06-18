// SPDX-License-Identifier: Apache-2.0

package check

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// authoringOnly is the sentinel runtime_mapping value for an authoring expert
// that has no runtime AgentDef counterpart yet (ADR-033).
const authoringOnly = "authoring-only"

// adrTableRowRe matches one ADR-033 mapping-table row: `| `<authoring>` | `<runtime>` | … |`.
var adrTableRowRe = regexp.MustCompile("^\\|\\s*`([^`]+)`\\s*\\|\\s*`([^`]+)`\\s*\\|")

type expertEntry struct {
	Authoring      string `yaml:"authoring"`
	RuntimeMapping string `yaml:"runtime_mapping"`
}

type expertMappingDoc struct {
	Experts []expertEntry `yaml:"experts"`
}

// ExpertMapping runs the three ADR-033 drift-guard rules against the repo at
// root, reconciling automation/experts/runtime_mapping.yaml with the authoring
// experts (.claude/commands/experts), the runtime agents (agents/examples), and
// the ADR-033 table. It returns the list of problems (empty == no drift) and the
// number of declared experts. A returned error is operational (e.g. the mapping
// file is unreadable), not a drift finding.
func ExpertMapping(root string) (problems []string, count int, err error) {
	experts, err := loadExpertMapping(filepath.Join(root, "automation", "experts", "runtime_mapping.yaml"))
	if err != nil {
		return nil, 0, err
	}
	count = len(experts)

	authoring, err := discoverAuthoringExperts(root)
	if err != nil {
		return nil, count, err
	}
	runtimeAgents, err := discoverRuntimeAgents(root)
	if err != nil {
		return nil, count, err
	}
	adrTable, err := parseADRTable(filepath.Join(root, "docs", "adr", "ADR-033-expert-agent-substrate.md"))
	if err != nil {
		return nil, count, err
	}

	declared, p1 := checkDeclared(experts, authoring) // rule 1
	problems = append(problems, p1...)
	problems = append(problems, checkRuntimeRefs(declared, runtimeAgents)...) // rule 2
	problems = append(problems, checkADRTable(declared, adrTable)...)         // rule 3
	return problems, count, nil
}

// checkDeclared applies rule 1: every authoring expert is declared exactly once
// with a non-empty runtime_mapping, and the mapping lists no unknown experts.
// It returns the declared {authoring: runtime_mapping} map and any problems.
func checkDeclared(experts []expertEntry, authoring map[string]bool) (map[string]string, []string) {
	var problems []string
	declared := map[string]string{}
	for i, e := range experts {
		if e.Authoring == "" {
			problems = append(problems, fmt.Sprintf("mapping entry #%d: missing 'authoring' slug", i))
			continue
		}
		if _, dup := declared[e.Authoring]; dup {
			problems = append(problems, fmt.Sprintf("%s: declared more than once in the mapping file", e.Authoring))
		}
		if e.RuntimeMapping == "" {
			problems = append(problems, fmt.Sprintf("%s: empty or missing 'runtime_mapping' (ADR-033)", e.Authoring))
		}
		declared[e.Authoring] = e.RuntimeMapping
	}
	if missing := sortedDiff(authoring, keySet(declared)); len(missing) > 0 {
		problems = append(problems, fmt.Sprintf(
			"authoring experts with no runtime_mapping declaration (ADR-033 rule 1): %v", missing))
	}
	if extra := sortedDiff(keySet(declared), authoring); len(extra) > 0 {
		problems = append(problems, fmt.Sprintf(
			"mapping lists experts with no .claude/commands/experts file: %v", extra))
	}
	return declared, problems
}

// checkRuntimeRefs applies rule 2: a named runtime_mapping must resolve to a
// runtime agent under agents/examples/.
func checkRuntimeRefs(declared map[string]string, runtimeAgents map[string]bool) []string {
	var problems []string
	for _, slug := range sortedKeys(declared) {
		rm := declared[slug]
		if rm != "" && rm != authoringOnly && !runtimeAgents[rm] {
			problems = append(problems, fmt.Sprintf(
				"%s: runtime_mapping '%s' does not resolve to agents/examples/%s (ADR-033 rule 2)", slug, rm, rm))
		}
	}
	return problems
}

// checkADRTable applies rule 3: the mapping must equal ADR-033's table.
func checkADRTable(declared, adrTable map[string]string) []string {
	var problems []string
	for _, slug := range sortedKeys(declared) {
		rm := declared[slug]
		tv, ok := adrTable[slug]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("%s: present in mapping but absent from ADR-033 table", slug))
		case tv != rm:
			problems = append(problems, fmt.Sprintf(
				"%s: ADR-033 table says '%s' but mapping says '%s' (ADR-033 rule 3)", slug, tv, rm))
		}
	}
	for _, slug := range sortedDiff(keySet(adrTable), keySet(declared)) {
		problems = append(problems, fmt.Sprintf("%s: listed in ADR-033 table but absent from mapping", slug))
	}
	return problems
}

// loadExpertMapping reads and parses the runtime_mapping.yaml experts list.
func loadExpertMapping(path string) ([]expertEntry, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // caller-supplied repo path
	if err != nil {
		return nil, fmt.Errorf("check expert-mapping: read %q: %w", path, err)
	}
	var doc expertMappingDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("check expert-mapping: parse %q: %w", path, err)
	}
	if doc.Experts == nil {
		return nil, fmt.Errorf("check expert-mapping: %q: missing or invalid 'experts' list", path)
	}
	return doc.Experts, nil
}

// discoverAuthoringExperts returns the slug set of .claude/commands/experts/*.md
// (read-only; the .claude tree is CODEOWNERS-gated).
func discoverAuthoringExperts(root string) (map[string]bool, error) {
	matches, err := filepath.Glob(filepath.Join(root, ".claude", "commands", "experts", "*.md"))
	if err != nil {
		return nil, fmt.Errorf("check expert-mapping: glob authoring experts: %w", err)
	}
	set := make(map[string]bool, len(matches))
	for _, p := range matches {
		set[strings.TrimSuffix(filepath.Base(p), ".md")] = true
	}
	return set, nil
}

// discoverRuntimeAgents returns the names of registerable runtime agents under
// agents/examples/ (a subdirectory containing a pyproject.toml).
func discoverRuntimeAgents(root string) (map[string]bool, error) {
	entries, err := os.ReadDir(filepath.Join(root, "agents", "examples"))
	if err != nil {
		return nil, fmt.Errorf("check expert-mapping: read agents/examples: %w", err)
	}
	set := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "agents", "examples", e.Name(), "pyproject.toml")); err == nil {
			set[e.Name()] = true
		}
	}
	return set, nil
}

// parseADRTable parses ADR-033's mapping table into {authoring: runtime_mapping}.
func parseADRTable(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // caller-supplied repo path
	if err != nil {
		return nil, fmt.Errorf("check expert-mapping: read %q: %w", path, err)
	}
	rows := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		if m := adrTableRowRe.FindStringSubmatch(line); m != nil {
			rows[m[1]] = m[2]
		}
	}
	return rows, nil
}

func keySet(m map[string]string) map[string]bool {
	s := make(map[string]bool, len(m))
	for k := range m {
		s[k] = true
	}
	return s
}

// sortedDiff returns the sorted elements of a that are not in b.
func sortedDiff(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
