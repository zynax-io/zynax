// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// clusterRunner runs a kind bring-up / teardown script with its stdout and
// stderr streamed live to the caller, plus an environment overlay. It is
// injected so `up`/`down` unit tests can record the (script, env) they would
// run without executing kind/helm/kubectl or touching a cluster.
type clusterRunner func(ctx context.Context, stdout, stderr io.Writer, script string, env []string) error

// streamRunner is the production clusterRunner: it execs the script and streams
// its output. Bring-up runs for minutes and prints progressive status, so the
// stdio is wired through directly (as in mcp.go's git-adapter launch) rather
// than buffered with CombinedOutput.
func streamRunner(ctx context.Context, stdout, stderr io.Writer, script string, env []string) error {
	// #nosec G204 — script is the fixed repo-relative cluster-up/cluster-down
	// path resolved under a verified checkout root (isRepoRoot), never derived
	// from prompt/tool input; there are no arguments — all config flows via env.
	cmd := exec.CommandContext(ctx, script)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(script), err)
	}
	return nil
}

// clusterDeps bundles the injectable collaborators shared by `up` and `down`
// so their unit tests are hermetic (no repo-root walk, no real script exec).
type clusterDeps struct {
	run      clusterRunner
	findRoot func(override string) (string, error)
}

// defaultClusterDeps wires the production collaborators.
func defaultClusterDeps() clusterDeps {
	return clusterDeps{run: streamRunner, findRoot: findRepoRoot}
}

var (
	downClusterName string
	downRepoRoot    string
)

var downCmd = &cobra.Command{
	Use:     "down",
	GroupID: beginnerGroupID,
	Short:   "Tear down the local kind cluster created by `zynax up`",
	Long: `Delete the local kind cluster and everything on it — the teardown half of
the write-once-run-on-Temporal-or-Argo local runtime.

This wraps the same script as ` + "`make kind-down`" + ` (scripts/e2e/cluster-down.sh),
but works ` + "`make`" + `-free from any directory inside a checkout. It is idempotent:
it succeeds even when no cluster exists.

Run this from inside a Zynax checkout, or point at one with --repo-root /
$ZYNAX_REPO_ROOT.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runDown(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), defaultClusterDeps())
	},
}

// runDown resolves the repo root and streams scripts/e2e/cluster-down.sh,
// forwarding the cluster name via CLUSTER_NAME when set (otherwise the script's
// own default applies). Teardown takes no other configuration.
func runDown(ctx context.Context, stdout, stderr io.Writer, deps clusterDeps) error {
	root, err := deps.findRoot(downRepoRoot)
	if err != nil {
		return err
	}
	script := filepath.Join(root, clusterDownRel)

	env := os.Environ()
	if downClusterName != "" {
		env = append(env, "CLUSTER_NAME="+downClusterName)
	}
	_, _ = fmt.Fprintln(stdout, "zynax down — tearing down the local kind cluster.")
	return deps.run(ctx, stdout, stderr, script, env)
}

func init() {
	downCmd.Flags().StringVar(&downClusterName, "cluster-name", "",
		"kind cluster name to delete (default: the script's default, zynax-e2e)")
	downCmd.Flags().StringVar(&downRepoRoot, "repo-root", "",
		"path to the Zynax checkout ($ZYNAX_REPO_ROOT); default: walk up from the current directory")
	rootCmd.AddCommand(downCmd)
}
