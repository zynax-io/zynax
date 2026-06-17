// SPDX-License-Identifier: Apache-2.0

// Package bddselect computes the godog BDD package/service matrix to run based
// on changed proto / contract-test files. It is the tested Go replacement for
// tools/ci/bdd-select-packages.sh (ADR-036, M7 EPIC S step S.3).
//
// Output parity with the bash:
//   - "ALL"  → run the full suite
//   - ""     → run nothing
//   - else   → a space-separated, deduplicated, sorted list of packages
package bddselect

import (
	"path"
	"sort"
	"strings"
)

// All is the sentinel selection that triggers the full BDD suite.
const All = "ALL"

// pkgMap maps a changed protos/zynax/v1/*.proto basename to its BDD package
// (parity with the PKG_MAP associative array in bdd-select-packages.sh).
var pkgMap = map[string]string{
	"agent.proto":             "agent_service",
	"agent_registry.proto":    "agent_registry_service",
	"cloudevents.proto":       "cloudevents_envelope",
	"engine_adapter.proto":    "engine_adapter_service",
	"event_bus.proto":         "event_bus_service",
	"memory.proto":            "memory_service",
	"task_broker.proto":       "task_broker_service",
	"workflow_compiler.proto": "workflow_compiler_service",
}

// Select returns the BDD selection for the given changed file paths (the
// `git diff --name-only BASE..HEAD -- protos/` output, one path per element).
// Blank entries are ignored, matching the bash `[ -z "$f" ] && continue`.
func Select(changed []string) string {
	pkgs := map[string]struct{}{}
	for _, f := range changed {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if forcesAll(f) {
			return All
		}
		if pkg := pkgFor(f); pkg != "" {
			pkgs[pkg] = struct{}{}
		}
	}
	return join(pkgs)
}

// forcesAll reports whether a changed path triggers the full suite: shared test
// infrastructure (go.mod/go.sum or testserver/) or any feature file.
func forcesAll(f string) bool {
	if strings.HasPrefix(f, "protos/tests/go.mod") ||
		strings.HasPrefix(f, "protos/tests/go.sum") ||
		strings.HasPrefix(f, "protos/tests/testserver/") {
		return true
	}
	return strings.HasPrefix(f, "protos/tests/features/")
}

// pkgFor maps a single changed path to its BDD package, or "" if none.
func pkgFor(f string) string {
	switch {
	case strings.HasPrefix(f, "protos/zynax/v1/") && strings.HasSuffix(f, ".proto"):
		return pkgMap[path.Base(f)]
	case strings.HasPrefix(f, "protos/tests/"):
		// protos/tests/<pkg>/... — the third path segment is the package,
		// excluding the "features" dir (handled by forcesAll).
		parts := strings.Split(f, "/")
		if len(parts) >= 4 && parts[2] != "features" {
			return parts[2]
		}
	}
	return ""
}

// join renders the package set as a sorted, space-separated string.
func join(set map[string]struct{}) string {
	if len(set) == 0 {
		return ""
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return strings.Join(out, " ")
}
