// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/bumprunner"
)

var (
	bumpRunnerRoot   string
	bumpRunnerDryRun bool
)

var bumpRunnerCmd = &cobra.Command{
	Use:   "bump-runner <sha256:digest>",
	Short: "Pin a new ci-runner digest in images.yaml and re-stamp consumers",
	Long: `Update the ci-runner digest in images/images.yaml (the image-reference
source of truth, ADR-024) and re-stamp every consumer file via the images sync
internals so all references stay aligned.

Replaces scripts/bump-ci-runner.sh (ADR-036). The legacy script sed-replaced the
digest in each workflow file directly; this verb updates the SoT once and lets
images sync drive the consumers, keeping a single tested path. It is idempotent:
re-running with the current digest is a no-op.

Pass --dry-run to preview the digest change and the consumer files that would be
re-stamped without writing anything (parity with the script's --check mode).`,
	Args: cobra.ExactArgs(1),
	RunE: runBumpRunner,
}

func init() {
	bumpRunnerCmd.Flags().StringVar(&bumpRunnerRoot, "root", ".", "repository root directory")
	bumpRunnerCmd.Flags().BoolVar(&bumpRunnerDryRun, "dry-run", false, "preview changes; do not write files")
	rootCmd.AddCommand(bumpRunnerCmd)
}

func runBumpRunner(cmd *cobra.Command, args []string) error {
	root, err := resolveRoot(bumpRunnerRoot)
	if err != nil {
		return fmt.Errorf("bump-runner: %w", err)
	}
	res, err := bumprunner.Bump(root, args[0], bumpRunnerDryRun)
	if err != nil {
		return err
	}
	return printBumpResult(cmd, res)
}

// printBumpResult renders the human-readable summary of a bump.
func printBumpResult(cmd *cobra.Command, res bumprunner.Result) error {
	out := cmd.OutOrStdout()
	if !res.YAMLChanged && len(res.ChangedFiles) == 0 {
		_, err := fmt.Fprintf(out, "✅  ci-runner already pinned to %s — nothing to do.\n", res.After)
		return wrapWrite(err)
	}
	verb := "Updated"
	if bumpRunnerDryRun {
		verb = "Would update"
	}
	if _, err := fmt.Fprintf(out, "🔄 %s ci-runner digest %s → %s\n", verb, res.Before, res.After); err != nil {
		return wrapWrite(err)
	}
	for _, f := range res.ChangedFiles {
		if _, err := fmt.Fprintf(out, "   ✓ %s\n", f); err != nil {
			return wrapWrite(err)
		}
	}
	_, err := fmt.Fprintf(out, "✅  %s images.yaml + %d consumer file(s).\n", verb, len(res.ChangedFiles))
	return wrapWrite(err)
}

// wrapWrite annotates a stdout write error for the wrapcheck linter.
func wrapWrite(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("bump-runner: write summary: %w", err)
}
