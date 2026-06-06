// SPDX-License-Identifier: Apache-2.0

package check

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// mustMatchModules lists shared deps that must pin identical versions across
// every go.mod that declares them. "go" is the toolchain directive and must
// match in ALL go.mod files.
var mustMatchModules = []string{
	"go", // Go toolchain directive — must match in every module
	"github.com/kelseyhightower/envconfig",
	"google.golang.org/grpc",
	"google.golang.org/protobuf",
	"gopkg.in/yaml.v3",
}

// DepsViolation records a module whose version differs across go.mod files.
type DepsViolation struct {
	Module   string            // dependency path ("go" for toolchain directive)
	Versions map[string]string // go.mod file path → version string
}

// DepsReport is the result of a Deps check.
type DepsReport struct {
	GoModFiles []string        // all go.mod files found
	Violations []DepsViolation // non-empty when versions diverge
}

// Deps scans repoRoot for all go.mod files and verifies that each module in
// mustMatchModules pins the same version across all files that declare it.
// The "go" directive is checked across ALL go.mod files.
// Other modules are only compared where they appear (absent = skip).
func Deps(repoRoot string) (DepsReport, error) {
	modFiles, err := findGoModFiles(repoRoot)
	if err != nil {
		return DepsReport{}, err
	}

	versions, err := collectVersions(repoRoot, modFiles)
	if err != nil {
		return DepsReport{}, err
	}

	report := DepsReport{GoModFiles: make([]string, len(modFiles))}
	for i, p := range modFiles {
		report.GoModFiles[i] = relPath(repoRoot, p)
	}
	for _, mod := range mustMatchModules {
		vmap := versions[mod]
		if len(vmap) < 2 {
			continue
		}
		if len(uniqueVersions(vmap)) > 1 {
			report.Violations = append(report.Violations, DepsViolation{
				Module:   mod,
				Versions: copyMap(vmap),
			})
		}
	}
	return report, nil
}

// findGoModFiles walks repoRoot and returns all go.mod paths, skipping vendor/.git/hidden dirs.
func findGoModFiles(repoRoot string) ([]string, error) {
	var modFiles []string
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == "vendor" || base == ".git" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "go.mod" {
			modFiles = append(modFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("check deps: walk %s: %w", repoRoot, err)
	}
	sort.Strings(modFiles)
	return modFiles, nil
}

// collectVersions parses each go.mod file and builds a versions[module][path]=version map.
func collectVersions(repoRoot string, modFiles []string) (map[string]map[string]string, error) {
	versions := make(map[string]map[string]string, len(mustMatchModules))
	for _, mod := range mustMatchModules {
		versions[mod] = make(map[string]string)
	}
	for _, modPath := range modFiles {
		parsed, err := parseGoMod(modPath)
		if err != nil {
			return nil, fmt.Errorf("check deps: parse %s: %w", modPath, err)
		}
		rel := relPath(repoRoot, modPath)
		for mod, vmap := range versions {
			if v, ok := parsed[mod]; ok {
				vmap[rel] = v
			}
		}
	}
	return versions, nil
}

// PrintDepsReport writes the report to w. Returns true if all versions agree.
func PrintDepsReport(w *os.File, r DepsReport) bool {
	if len(r.Violations) == 0 {
		_, _ = fmt.Fprintf(w, "✅  All %d go.mod files agree on shared dependency versions.\n", len(r.GoModFiles))
		return true
	}
	_, _ = fmt.Fprintf(w, "❌  Shared dependency version mismatch found in %d module(s):\n\n", len(r.Violations))
	for _, v := range r.Violations {
		_, _ = fmt.Fprintf(w, "  %s\n", v.Module)
		files := sortedKeys(v.Versions)
		for _, f := range files {
			_, _ = fmt.Fprintf(w, "    %-60s %s\n", f, v.Versions[f])
		}
		_, _ = fmt.Fprintln(w)
	}
	_, _ = fmt.Fprintln(w, "Fix: align all go.mod files to the same version before merging.")
	return false
}

// parseGoMod reads a go.mod file and returns a map of module path → version
// for the "go" directive and all require entries.
func parseGoMod(path string) (map[string]string, error) {
	f, err := os.Open(path) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	inRequire := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		// toolchain directive: "go 1.26.4"
		if strings.HasPrefix(line, "go ") {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				result["go"] = parts[1]
			}
			continue
		}
		// require block start
		if line == "require (" {
			inRequire = true
			continue
		}
		// require block end
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		// single-line require: "require foo v1.2.3"
		if strings.HasPrefix(line, "require ") {
			parts := strings.Fields(strings.TrimPrefix(line, "require "))
			if len(parts) >= 2 {
				result[parts[0]] = parts[1]
			}
			continue
		}
		// require block entry: "foo v1.2.3" or "foo v1.2.3 // indirect"
		if inRequire {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				result[parts[0]] = parts[1]
			}
		}
	}
	return result, scanner.Err()
}

func uniqueVersions(m map[string]string) map[string]struct{} {
	u := make(map[string]struct{})
	for _, v := range m {
		u[v] = struct{}{}
	}
	return u
}

func copyMap(m map[string]string) map[string]string {
	c := make(map[string]string, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
