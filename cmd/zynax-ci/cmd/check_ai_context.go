// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

var aiContextRoot string

var checkAIContextCmd = &cobra.Command{
	Use:   "ai-context",
	Short: "Report line counts for AI context files (advisory, always exits 0)",
	Long: `Count lines in CLAUDE.md, AGENTS.md files, and docs/ai-assistant-setup.md.

Thresholds:
  CLAUDE.md                 200 lines
  AGENTS.md (root)          300 lines
  docs/ai-assistant-setup.md 150 lines
  per-directory AGENTS.md   150 lines
  TOTAL                    2000 lines

Exceeding a threshold emits a WARN but never causes a non-zero exit code.
This check is advisory only.`,
	Args: cobra.NoArgs,
	RunE: runCheckAIContext,
}

func init() {
	checkAIContextCmd.Flags().StringVar(&aiContextRoot, "root", ".", "repository root directory")
	checkCmd.AddCommand(checkAIContextCmd)
}

func runCheckAIContext(cmd *cobra.Command, args []string) error {
	root := aiContextRoot
	if root == "." {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("check ai-context: get working directory: %w", err)
		}
	}

	report, err := check.Context(root)
	if err != nil {
		return err
	}
	report.Print(os.Stdout)
	return nil // always exits 0
}
