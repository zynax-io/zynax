// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax/client"
	"github.com/zynax-io/zynax/cmd/zynax/validate"
)

var (
	applyDryRun bool
	applyEngine string
)

var applyCmd = &cobra.Command{
	Use:   "apply <file|scenario-dir>",
	Short: "Apply a YAML manifest (Workflow or AgentDef) or a scenario manifest set",
	Long: `Apply a single YAML manifest, or a scenario manifest set.

A scenario is a directory containing a 'scenario.yaml' index (kind: Scenario)
plus the member Workflow and AgentDef manifests. When the argument is a scenario
directory or index, each member is submitted over the existing /api/v1/apply REST
path in the index's declared apply_order — AgentDefs before the Workflow that
consumes their capabilities. No new api-gateway endpoint is introduced.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		indexPath, isScenario, err := validate.ResolveScenarioIndex(args[0])
		if err != nil {
			return err
		}
		gw := newGateway()
		if isScenario {
			return runApplyScenario(cmd, gw, indexPath)
		}
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("read %s: %w", args[0], err)
		}
		if applyDryRun {
			return runDryRun(cmd, gw, data)
		}
		return runApply(cmd, gw, data)
	},
}

func init() {
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "validate manifest without submitting")
	applyCmd.Flags().StringVar(&applyEngine, "engine", "", "engine hint forwarded to SubmitWorkflow (ADR-015)")
	rootCmd.AddCommand(applyCmd)
}

// runApplyScenario expands a scenario index and applies each member in
// apply_order over the existing /api/v1/apply REST path. On --dry-run each member
// is dry-run-validated instead of submitted.
func runApplyScenario(cmd *cobra.Command, gw *client.Gateway, indexPath string) error {
	members, err := validate.ExpandScenario(indexPath)
	if err != nil {
		return err
	}
	// Resolve the declarative context-injection block (#1387): parse it
	// (rejecting any routing/provider field — data-only, ADR-013), read its file
	// sources, and apply the max_tokens cap. The resolved values bind into the
	// Workflow member's {{ .ctx.* }} references below before it is submitted —
	// keeping injection client-side over the existing REST path (no new proto
	// field, no api-gateway endpoint).
	block, err := validate.ParseContextBlock(indexPath)
	if err != nil {
		return err
	}
	ctxValues, err := validate.ResolveContext(block, filepath.Dir(indexPath))
	if err != nil {
		return err
	}
	for _, m := range members {
		data, err := os.ReadFile(m.Path) //nolint:gosec // member path is confined by ExpandScenario
		if err != nil {
			return fmt.Errorf("read scenario member %s (%s): %w", m.ID, m.Path, err)
		}
		if m.Kind == "Workflow" {
			data, err = validate.BindContextIntoWorkflow(data, ctxValues)
			if err != nil {
				return fmt.Errorf("scenario member %s: %w", m.ID, err)
			}
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "applying %s (%s)...\n", m.ID, m.Kind)
		if applyDryRun {
			if err := runDryRun(cmd, gw, data); err != nil {
				return fmt.Errorf("scenario member %s: %w", m.ID, err)
			}
			continue
		}
		if err := runApply(cmd, gw, data); err != nil {
			return fmt.Errorf("scenario member %s: %w", m.ID, err)
		}
	}
	return nil
}

func runApply(cmd *cobra.Command, gw *client.Gateway, body []byte) error {
	runID, agentID, warnings, err := gw.Apply(cmd.Context(), body, applyEngine)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "warning: %s\n", w)
	}
	if runID != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "run_id: %s\n", runID)
	}
	if agentID != "" {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "agent_id: %s\n", agentID)
	}
	return nil
}

func runDryRun(cmd *cobra.Command, gw *client.Gateway, body []byte) error {
	errs, warnings, err := gw.ApplyDryRun(cmd.Context(), body)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "warning: %s\n", w)
	}
	if len(errs) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "ok (dry-run: no errors)")
		return nil
	}
	for _, e := range errs {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error (line %d): [%s] %s\n", e.Line, e.Code, e.Message)
	}
	return fmt.Errorf("compilation failed with %d error(s)", len(errs))
}
