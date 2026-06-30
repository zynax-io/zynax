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
	Use:     "result [run-id]",
	GroupID: beginnerGroupID,
	Short:   "Print the capability output (result payload) of a workflow run",
	Long: "Stream a run's events and print the capability result payload — e.g. the " +
		"model's review text from the code-review example — straight from the CLI.\n\n" +
		"With no run id the command targets your most recent run (recorded by the " +
		"last `zynax apply`/`run`). An explicit run id always overrides.\n\n" +
		"The command tails the run until it reaches a terminal state and prints the " +
		"last completion text it saw. A run that COMPLETED but declared no output " +
		"exits 0 with a note (see the see-workflow-result runbook); a FAILED or " +
		"CANCELLED run exits non-zero.",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID, err := resolveRunID(args)
		if err != nil {
			return err
		}
		gw := newGateway()
		var output, terminalStatus string
		err = gw.WatchWorkflowLogs(cmd.Context(), runID, func(ev client.LogEvent) error {
			if text := client.CompletionText(ev.Payload); text != "" {
				output = text
			}
			if terminalStatuses[ev.Status] {
				terminalStatus = ev.Status
				return errResultDone
			}
			return nil
		})
		if err != nil && !errors.Is(err, errResultDone) {
			return err
		}
		if output != "" {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), output)
			return nil
		}
		// No completion text was emitted. A COMPLETED run that simply declared no
		// output is a success — echo/hello-world style runs produce none — so exit
		// 0 with a note pointing at the runbook (the structured-output reader in
		// O.9 supersedes this). FAILED/CANCELLED, or a stream that closed before
		// any terminal state, remain errors so a real failure never reads as success.
		switch terminalStatus {
		case "WORKFLOW_STATUS_COMPLETED":
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(),
				"run completed with no declared output — see docs/runbooks/see-workflow-result.md")
			return nil
		case "":
			return fmt.Errorf("no result payload for run %s", runID)
		default:
			return fmt.Errorf("run %s did not produce a result: status %s", runID, terminalStatus)
		}
	},
}

func init() {
	rootCmd.AddCommand(resultCmd)
}
