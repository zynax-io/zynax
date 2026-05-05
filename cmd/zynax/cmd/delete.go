// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete (cancel) resources",
}

var deleteWorkflowCmd = &cobra.Command{
	Use:   "workflow <run-id>",
	Short: "Cancel a running workflow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gw := newGateway()
		if err := gw.DeleteWorkflow(cmd.Context(), args[0]); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "cancelled: %s\n", args[0])
		return nil
	},
}

func init() {
	deleteCmd.AddCommand(deleteWorkflowCmd)
	rootCmd.AddCommand(deleteCmd)
}
