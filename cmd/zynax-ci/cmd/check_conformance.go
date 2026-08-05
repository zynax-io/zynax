// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

var conformanceRoot string

var checkConformanceCmd = &cobra.Command{
	Use:   "conformance",
	Short: "Drift guard: ZECS membership <-> corpus <-> engine legs (#1692)",
	Long: `Reconcile docs/conformance/scenarios.yaml — the Zynax Engine Conformance
Suite (ZECS) membership manifest — against the repository and fail on any drift:

  1. Every scenario source (and every runner of a leg that runs) resolves to a
     real file; scenario ids are unique.
  2. Every kind:Workflow manifest in spec/workflows/examples is either a scenario
     source or an explicitly reasoned non-member — nothing drops out silently.
  3. The declared engine legs equal the engines the engine-adapter can be
     configured with (ADR-015) and the e2e matrix legs in e2e-smoke.yml.
  4. Every scenario carries an explicit entry for every leg; a leg that does not
     run says so with a reason (a skipped leg is SKIPPED, never PASS).

This is NOT a conformance run: it executes no workflow and reports no per-engine
result. It only proves the suite definition is consistent with the repo, in about
a second and with no cluster. See docs/conformance/README.md.

Exits 0 when the definition reconciles, 1 on any drift.`,
	Args: cobra.NoArgs,
	RunE: runCheckConformance,
}

func init() {
	checkConformanceCmd.Flags().StringVar(&conformanceRoot, "root", ".", "repository root directory")
	checkCmd.AddCommand(checkConformanceCmd)
}

func runCheckConformance(cmd *cobra.Command, _ []string) error {
	root, err := resolveRoot(conformanceRoot)
	if err != nil {
		return fmt.Errorf("check conformance: %w", err)
	}

	problems, count, err := check.Conformance(root)
	if err != nil {
		return err
	}

	if len(problems) > 0 {
		errOut := cmd.ErrOrStderr()
		_, _ = fmt.Fprintln(errOut, "ZECS definition drift guard FAILED (docs/conformance/README.md):")
		for _, p := range problems {
			_, _ = fmt.Fprintf(errOut, "  - %s\n", p)
		}
		return errors.New("conformance definition drift detected")
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(),
		"ZECS definition drift guard OK — %d scenarios reconciled against the corpus and the engine legs.\n", count)
	return nil
}
