// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax/client"
)

// errResultDone is the sentinel returned from the SSE callback to stop the
// stream once the run reaches a terminal state. It is swallowed at the command
// boundary and never surfaces as an error.
var errResultDone = errors.New("result: terminal state reached")

var resultCmd = &cobra.Command{
	Use:   "result <run-id>",
	Short: "Print the capability output (result payload) of a workflow run",
	Long: "Stream a run's events and print the capability result payload — e.g. the " +
		"model's review text from the code-review example — straight from the CLI.\n\n" +
		"The command tails the run until it reaches a terminal state and prints the " +
		"last completion text it saw. Exits with an error if the run finishes with no " +
		"result (e.g. a failed run).",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gw := newGateway()
		var output string
		err := gw.WatchWorkflowLogs(cmd.Context(), args[0], func(ev client.LogEvent) error {
			if text := client.CompletionText(ev.Payload); text != "" {
				output = text
			}
			if terminalStatuses[ev.Status] {
				return errResultDone
			}
			return nil
		})
		if err != nil && !errors.Is(err, errResultDone) {
			return err
		}
		if output == "" {
			return fmt.Errorf("no result payload for run %s", args[0])
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), output)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resultCmd)
}
