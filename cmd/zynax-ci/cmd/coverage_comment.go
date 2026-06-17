// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/coveragecomment"
)

var (
	covResultsPath string
	covOutPath     string
)

var coverageCommentCmd = &cobra.Command{
	Use:   "coverage-comment",
	Short: "Render the PR coverage markdown comment from a coverage-results file",
	Long: `Read a coverage-results file (pipe-delimited type|name|pkg|pct rows) and
render the PR coverage markdown comment.

Replaces .github/scripts/build-coverage-comment.sh (ADR-036). Output is written
to --out (default $OUT or /tmp/coverage-comment.md) and echoed to stdout.

Inputs (flags override env, env overrides defaults):
  --results  $RESULTS  (default /tmp/coverage-results.txt)
  --out      $OUT      (default /tmp/coverage-comment.md)

Footer env:  RUN_NUMBER, RUN_URL, SHA
Gate env:    COVERAGE_DOMAIN_GATE, COVERAGE_ADAPTER_GATE,
             COVERAGE_CLI_ZYNAX_GATE, COVERAGE_CLI_ZYNAX_CI_GATE,
             COVERAGE_PYTHON_GATE`,
	Args: cobra.NoArgs,
	RunE: runCoverageComment,
}

func init() {
	coverageCommentCmd.Flags().StringVar(&covResultsPath, "results", "", "coverage-results file ($RESULTS or /tmp/coverage-results.txt)")
	coverageCommentCmd.Flags().StringVar(&covOutPath, "out", "", "output markdown file ($OUT or /tmp/coverage-comment.md)")
	rootCmd.AddCommand(coverageCommentCmd)
}

func runCoverageComment(cmd *cobra.Command, _ []string) error {
	results := pick(covResultsPath, os.Getenv("RESULTS"), "/tmp/coverage-results.txt")
	out := pick(covOutPath, os.Getenv("OUT"), "/tmp/coverage-comment.md")

	f, err := os.Open(results) //nolint:gosec // results path is CI-controlled (RESULTS env / flag)
	if err != nil {
		return fmt.Errorf("coverage-comment: open results: %w", err)
	}
	defer func() { _ = f.Close() }()

	md, err := coveragecomment.Render(f, gatesFromEnv(), metaFromEnv())
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, []byte(md), 0o600); err != nil { //nolint:gosec // out path is CI-controlled (OUT env / flag)
		return fmt.Errorf("coverage-comment: write %s: %w", out, err)
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), md); err != nil {
		return fmt.Errorf("coverage-comment: write stdout: %w", err)
	}
	return nil
}

func gatesFromEnv() coveragecomment.Gates {
	return coveragecomment.Gates{
		Domain:     os.Getenv("COVERAGE_DOMAIN_GATE"),
		Adapter:    os.Getenv("COVERAGE_ADAPTER_GATE"),
		CLIZynax:   os.Getenv("COVERAGE_CLI_ZYNAX_GATE"),
		CLIZynaxCI: os.Getenv("COVERAGE_CLI_ZYNAX_CI_GATE"),
		Python:     os.Getenv("COVERAGE_PYTHON_GATE"),
	}
}

func metaFromEnv() coveragecomment.Meta {
	return coveragecomment.Meta{
		RunNumber: os.Getenv("RUN_NUMBER"),
		RunURL:    os.Getenv("RUN_URL"),
		SHA:       os.Getenv("SHA"),
	}
}

// pick returns the first non-empty value among the candidates.
func pick(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
