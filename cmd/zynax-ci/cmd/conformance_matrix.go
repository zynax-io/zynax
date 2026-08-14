// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

var matrixOpts struct {
	root     string
	outcomes string
	leg      string
	run      bool
	revision string
	runURL   string
	out      string
}

var conformanceCmd = &cobra.Command{
	Use:   "conformance",
	Short: "Zynax Engine Conformance Suite (ZECS) reporting (#1692)",
}

var conformanceMatrixCmd = &cobra.Command{
	Use:   "matrix",
	Short: "Emit the ZECS per-engine pass/fail matrix as JSON (#1774)",
	Long: `Emit the machine-readable per-engine ZECS matrix as JSON — the artifact
adopters and adapter authors read (docs/conformance/README.md §10).

Every engine the engine-adapter can be configured with (ADR-015) gets a row,
executed or not, and every scenario in docs/conformance/scenarios.yaml gets a
cell on every leg: PASS, FAIL, or SKIPPED with skip_kind 'not_in_leg' (the
manifest says this leg does not run it) or 'not_executed' (it was meant to and
produced nothing, which makes the leg INCOMPLETE). There is no path to PASS
without an observed success, so a partial run cannot be published as a complete
one: read 'complete' before 'result'.

Two input modes, one output shape:

  --outcomes <dir>     recorded run: one '<ci_step>=<outcome>' file per leg, each
                       naming its leg with a 'leg=<engine>' line (CI writes these
                       from the e2e job's step outcomes)
  --run --leg <engine> live run: execute this leg's runners from the manifest
                       against a cluster you brought up (scripts/e2e/cluster-up.sh)

This reports; it never gates. It exits 0 once it has emitted a matrix — consumers
assert on the JSON, for example: jq -e '.result == "PASS" and .complete'.`,
	Args: cobra.NoArgs,
	RunE: runConformanceMatrix,
}

func init() {
	f := conformanceMatrixCmd.Flags()
	f.StringVar(&matrixOpts.root, "root", ".", "repository root directory")
	f.StringVar(&matrixOpts.outcomes, "outcomes", "", "directory of recorded per-leg outcome files")
	f.StringVar(&matrixOpts.leg, "leg", "", "engine leg to execute (with --run)")
	f.BoolVar(&matrixOpts.run, "run", false, "execute the leg's runners now instead of reading recorded outcomes")
	f.StringVar(&matrixOpts.revision, "revision", "", "commit SHA the run was produced from")
	f.StringVar(&matrixOpts.runURL, "run-url", "", "URL of the run that produced the outcomes")
	f.StringVar(&matrixOpts.out, "out", "-", "write the JSON matrix here ('-' for stdout)")

	conformanceCmd.AddCommand(conformanceMatrixCmd)
	rootCmd.AddCommand(conformanceCmd)
}

func runConformanceMatrix(cmd *cobra.Command, _ []string) error {
	root, err := resolveRoot(matrixOpts.root)
	if err != nil {
		return fmt.Errorf("conformance matrix: %w", err)
	}
	doc, err := check.ConformanceMatrix(check.MatrixOptions{
		Root:        root,
		OutcomesDir: matrixOpts.outcomes,
		Leg:         matrixOpts.leg,
		Run:         matrixOpts.run,
		Revision:    matrixOpts.revision,
		RunURL:      matrixOpts.runURL,
		Log:         cmd.ErrOrStderr(),
	})
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if matrixOpts.out != "-" {
		f, ferr := os.Create(matrixOpts.out) //nolint:gosec // caller-supplied output path
		if ferr != nil {
			return fmt.Errorf("conformance matrix: create %q: %w", matrixOpts.out, ferr)
		}
		defer func() { _ = f.Close() }()
		out = f
	}
	if err := check.EncodeMatrix(out, doc); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), check.Summary(doc))
	return nil
}
