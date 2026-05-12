// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax/client"
)

var logsFormat string

var logsCmd = &cobra.Command{
	Use:   "logs <run-id>",
	Short: "Stream lifecycle events for a workflow run",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gw := newGateway()
		return gw.WatchWorkflowLogs(cmd.Context(), args[0], func(ev client.LogEvent) error {
			if logsFormat == "json" {
				b, err := json.Marshal(ev)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}
			ts := ev.Timestamp
			if ts == "" {
				ts = "-"
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s] %-30s %s → %s (%s)\n",
				ts, ev.EventType, ev.FromState, ev.ToState, ev.Status)
			return nil
		})
	},
}

func init() {
	logsCmd.Flags().StringVar(&logsFormat, "format", "text", "output format: text|json")
	rootCmd.AddCommand(logsCmd)
}
