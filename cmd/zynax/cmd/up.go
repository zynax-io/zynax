// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	upProfile     string
	upEngine      string
	upNoLoad      bool
	upClusterName string
	upNamespace   string
	upRepoRoot    string
)

var upCmd = &cobra.Command{
	Use:     "up",
	GroupID: beginnerGroupID,
	Short:   "Bring up the full Zynax platform on a local kind cluster",
	Long: `Create (or reuse) a local kind cluster and deploy the full Zynax umbrella —
every core service plus the always-on echo capability — so a single command
takes you from a fresh checkout to a live gateway on http://localhost:8080.

Write your workflow ONCE — it runs on Temporal OR Argo on the same kind cluster
that mirrors production. Pick the engine with a flag:

  zynax up --engine temporal    # (default)
  zynax up --engine argo        # same platform, same workflow, Argo engine

This wraps the same proven, idempotent script as ` + "`make kind-up`" + `
(scripts/e2e/cluster-up.sh), but works ` + "`make`" + `-free from any directory inside a
checkout. Re-running against an existing cluster reuses it.

Prerequisites: Docker, kind, kubectl, helm on PATH (~4 CPU / 8 GB RAM).
Run this from inside a Zynax checkout, or point at one with --repo-root /
$ZYNAX_REPO_ROOT. Tear down with: zynax down`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runUp(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), defaultClusterDeps())
	},
}

// runUp resolves the repo root and streams scripts/e2e/cluster-up.sh, mapping
// the command flags 1:1 onto the script's env-var contract. The child env is the
// process environment plus our overrides appended last (last-wins, matching the
// script's ${VAR:-default} semantics) so power-user vars (WAIT_TIMEOUT,
// KIND_NODE_IMAGE, …) still pass through untouched. Profile/engine validation is
// owned by the script (full|lite, temporal|argo) to avoid drift.
func runUp(ctx context.Context, stdout, stderr io.Writer, deps clusterDeps) error {
	root, err := deps.findRoot(upRepoRoot)
	if err != nil {
		return err
	}
	script := filepath.Join(root, clusterUpRel)

	env := os.Environ()
	env = append(env, "PROFILE="+upProfile, "E2E_ENGINE="+upEngine)
	if upClusterName != "" {
		env = append(env, "CLUSTER_NAME="+upClusterName)
	}
	if upNamespace != "" {
		env = append(env, "NAMESPACE="+upNamespace)
	}
	// Default: side-load local images (mirrors `make kind-up`, which forces
	// KIND_LOAD_IMAGES=1). --no-load-images omits it, so the script pulls from
	// the registry instead.
	if !upNoLoad {
		env = append(env, "KIND_LOAD_IMAGES=1")
	}

	_, _ = fmt.Fprintf(stdout,
		"zynax up — full platform on kind (profile: %s, engine: %s). Write once, run on Temporal or Argo.\n",
		upProfile, upEngine)
	return deps.run(ctx, stdout, stderr, script, env)
}

func init() {
	upCmd.Flags().StringVar(&upProfile, "profile", "full",
		"cluster profile: full (3-node, prod-mirroring) or lite (1-node, lean)")
	upCmd.Flags().StringVar(&upEngine, "engine", "temporal",
		"workflow engine to run on: temporal or argo")
	upCmd.Flags().BoolVar(&upNoLoad, "no-load-images", false,
		"do not side-load local images into the cluster (pull from the registry instead)")
	upCmd.Flags().StringVar(&upClusterName, "cluster-name", "",
		"kind cluster name (default: the script's default, zynax-e2e)")
	upCmd.Flags().StringVar(&upNamespace, "namespace", "",
		"Kubernetes namespace for the Zynax release (default: the script's default, zynax)")
	upCmd.Flags().StringVar(&upRepoRoot, "repo-root", "",
		"path to the Zynax checkout ($ZYNAX_REPO_ROOT); default: walk up from the current directory")
	rootCmd.AddCommand(upCmd)
}
