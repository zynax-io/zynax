// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax/client"
)

var (
	applyDryRun bool
	applyEngine string
)

var applyCmd = &cobra.Command{
	Use:   "apply <file>",
	Short: "Apply a YAML manifest (Workflow or AgentDef)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("read %s: %w", args[0], err)
		}
		gw := newGateway()
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

func runApply(cmd *cobra.Command, gw *client.Gateway, body []byte) error {
	runID, agentID, warnings, err := gw.Apply(cmd.Context(), body, applyEngine)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Fprintf(cmd.OutOrStdout(), "warning: %s\n", w)
	}
	if runID != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "run_id: %s\n", runID)
	}
	if agentID != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "agent_id: %s\n", agentID)
	}
	return nil
}

func runDryRun(cmd *cobra.Command, gw *client.Gateway, body []byte) error {
	errs, warnings, err := gw.ApplyDryRun(cmd.Context(), body)
	if err != nil {
		return err
	}
	for _, w := range warnings {
		fmt.Fprintf(cmd.OutOrStdout(), "warning: %s\n", w)
	}
	if len(errs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "ok (dry-run: no errors)")
		return nil
	}
	for _, e := range errs {
		fmt.Fprintf(cmd.ErrOrStderr(), "error (line %d): [%s] %s\n", e.Line, e.Code, e.Message)
	}
	return fmt.Errorf("compilation failed with %d error(s)", len(errs))
}
