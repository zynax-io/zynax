// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// terminalStatuses is the set of workflow statuses that indicate a finished run.
var terminalStatuses = map[string]bool{
	"WORKFLOW_STATUS_COMPLETED": true,
	"WORKFLOW_STATUS_FAILED":    true,
	"WORKFLOW_STATUS_CANCELLED": true,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of resources",
}

var statusWorkflowCmd = &cobra.Command{
	Use:   workflowRunIDUse,
	Short: "Check workflow status (exits 0 if terminal, 2 if still running)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gw := newGateway()
		run, err := gw.GetWorkflow(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", run.Status)
		if run.Version != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "version: %s\n", run.Version)
		}
		if !terminalStatuses[run.Status] {
			os.Exit(2)
		}
		return nil
	},
}

func init() {
	statusCmd.AddCommand(statusWorkflowCmd)
	rootCmd.AddCommand(statusCmd)
}
