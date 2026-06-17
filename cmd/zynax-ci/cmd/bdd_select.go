// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/bddselect"
)

var (
	bddBase string
	bddHead string
)

var bddSelectCmd = &cobra.Command{
	Use:   "bdd-select",
	Short: "Emit the godog BDD package selection from changed proto files",
	Long: `Print the BDD packages to run based on the proto / contract-test files changed
between BASE and HEAD:

  ALL  → run the full suite          (empty) → run nothing
  else → a sorted, deduplicated, space-separated package list

Replaces tools/ci/bdd-select-packages.sh (ADR-036). Fail-open: if the git diff
fails, "ALL" is printed and the verb exits 0 (parity with the bash).

Inputs (flags override env):
  --base  $BASE    base commit SHA / ref
  --head  $HEAD    head commit SHA / ref`,
	Args: cobra.NoArgs,
	RunE: runBDDSelect,
}

func init() {
	bddSelectCmd.Flags().StringVar(&bddBase, "base", "", "base commit SHA/ref ($BASE)")
	bddSelectCmd.Flags().StringVar(&bddHead, "head", "", "head commit SHA/ref ($HEAD)")
	rootCmd.AddCommand(bddSelectCmd)
}

func runBDDSelect(cmd *cobra.Command, _ []string) error {
	base := pick(bddBase, os.Getenv("BASE"))
	head := pick(bddHead, os.Getenv("HEAD"))

	changed, err := gitDiffProtos(base, head)
	if err != nil {
		// Fail-open: a diff failure runs the full suite (parity with the bash).
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), err)
		return writeLine(cmd, bddselect.All)
	}
	return writeLine(cmd, bddselect.Select(changed))
}

// writeLine writes s and a newline to the command's stdout.
func writeLine(cmd *cobra.Command, s string) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), s); err != nil {
		return fmt.Errorf("bdd-select: write stdout: %w", err)
	}
	return nil
}

// gitDiffProtos returns the protos/ files changed between base and head.
func gitDiffProtos(base, head string) ([]string, error) {
	out, err := exec.Command("git", "diff", "--name-only", base+".."+head, "--", "protos/").Output() //nolint:gosec // base/head are CI refs (BASE/HEAD env)
	if err != nil {
		return nil, fmt.Errorf("git diff %s..%s: %w", base, head, err)
	}
	return strings.Split(strings.TrimRight(string(out), "\n"), "\n"), nil
}
