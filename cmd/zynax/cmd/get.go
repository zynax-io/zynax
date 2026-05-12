// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get resources",
}

var getWorkflowCmd = &cobra.Command{
	Use:   "workflow <run-id>",
	Short: "Get the current status of a workflow run",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gw := newGateway()
		run, err := gw.GetWorkflow(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "run_id:        %s\n", run.RunID)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "workflow_id:   %s\n", run.WorkflowID)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "status:        %s\n", run.Status)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "current_state: %s\n", run.CurrentState)
		return nil
	},
}

func init() {
	getCmd.AddCommand(getWorkflowCmd)
	rootCmd.AddCommand(getCmd)
}
