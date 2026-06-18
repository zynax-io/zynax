// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/check"
)

var expertMappingRoot string

var checkExpertMappingCmd = &cobra.Command{
	Use:   "expert-mapping",
	Short: "Drift guard: authoring <-> runtime expert mapping (ADR-033)",
	Long: `Reconcile automation/experts/runtime_mapping.yaml against three surfaces
and fail on any divergence (ADR-033):

  1. Every authoring expert (.claude/commands/experts/<slug>.md) is declared
     exactly once, with a non-empty runtime_mapping.
  2. A named runtime_mapping resolves to a runtime agent under agents/examples/.
  3. The mapping file stays identical to ADR-033's mapping table.

Exits 0 when the mapping reconciles, 1 on any drift.`,
	Args: cobra.NoArgs,
	RunE: runCheckExpertMapping,
}

func init() {
	checkExpertMappingCmd.Flags().StringVar(&expertMappingRoot, "root", ".", "repository root directory")
	checkCmd.AddCommand(checkExpertMappingCmd)
}

func runCheckExpertMapping(cmd *cobra.Command, _ []string) error {
	root, err := resolveRoot(expertMappingRoot)
	if err != nil {
		return fmt.Errorf("check expert-mapping: %w", err)
	}

	problems, count, err := check.ExpertMapping(root)
	if err != nil {
		return err
	}

	if len(problems) > 0 {
		errOut := cmd.ErrOrStderr()
		_, _ = fmt.Fprintln(errOut, "Expert mapping drift guard FAILED (ADR-033):")
		for _, p := range problems {
			_, _ = fmt.Fprintf(errOut, "  - %s\n", p)
		}
		return errors.New("expert mapping drift detected")
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Expert mapping drift guard OK — %d authoring experts reconciled.\n", count)
	return nil
}
