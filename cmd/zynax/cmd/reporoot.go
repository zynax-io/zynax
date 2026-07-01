// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// repoRootEnv overrides repo-root discovery for the `up`/`down` cluster wrappers.
const repoRootEnv = "ZYNAX_REPO_ROOT"

// clusterUpRel and clusterDownRel are the kind bring-up / teardown scripts,
// relative to the repo root. cluster-up.sh doubles as the discovery sentinel:
// the one file `zynax up`/`down` must have is the script they exec, so probing
// for it is both the checkout marker and a pre-flight existence check.
const (
	clusterUpRel   = "scripts/e2e/cluster-up.sh"
	clusterDownRel = "scripts/e2e/cluster-down.sh"
)

// errNoRepoRoot is returned when no Zynax checkout can be located. `zynax up`/
// `down` drive the in-repo kind scripts (ADR-041), so a checkout is required;
// the message points at the three ways to supply one.
var errNoRepoRoot = errors.New(
	"zynax up/down needs a Zynax checkout — could not find " + clusterUpRel + " above the current directory.\n" +
		"  • run this from inside a `git clone https://github.com/zynax-io/zynax` checkout, or\n" +
		"  • set " + repoRootEnv + "=/path/to/zynax, or\n" +
		"  • pass --repo-root /path/to/zynax")

// findRepoRoot resolves the Zynax checkout root that holds the kind bring-up
// scripts, trying in order: an explicit --repo-root value, $ZYNAX_REPO_ROOT,
// then a walk up from the current directory for the sentinel cluster-up.sh.
// It returns errNoRepoRoot (clone-pointing) when no checkout is found. The
// returned path is always absolute so callers can join the script path safely.
func findRepoRoot(override string) (string, error) {
	if override != "" {
		if isRepoRoot(override) {
			return filepath.Abs(override)
		}
		return "", fmt.Errorf("--repo-root %q does not contain %s", override, clusterUpRel)
	}
	if env := os.Getenv(repoRootEnv); env != "" {
		if isRepoRoot(env) {
			return filepath.Abs(env)
		}
		return "", fmt.Errorf("%s=%q does not contain %s", repoRootEnv, env, clusterUpRel)
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if isRepoRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir { // reached the filesystem root without a match
			return "", errNoRepoRoot
		}
		dir = parent
	}
}

// isRepoRoot reports whether dir looks like a Zynax checkout — the bring-up
// script exists there. Using the script itself as the sentinel means discovery
// fails loudly if pointed at a tree without the kind harness.
func isRepoRoot(dir string) bool {
	// #nosec G304 G703 — existence probe only (os.Stat reads no file content); the
	// joined path is a fixed repo-relative sentinel, and the discovery root is
	// operator-supplied (flag/env/cwd), never prompt/tool input.
	_, err := os.Stat(filepath.Join(dir, clusterUpRel))
	return err == nil
}
