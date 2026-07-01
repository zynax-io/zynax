// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"
	"io"
	"sort"

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
	Short:   "Print a workflow run's declared result",
	Long: "See your declared workflow result from one command — the same whether the " +
		"run executed on Temporal or Argo.\n\n" +
		"`zynax result <run>` reads the run's declared outputs from " +
		"GET /api/v1/workflows/{id}/outputs and prints them as name=value lines. If " +
		"the workflow declared no outputs, it falls back to the last capability " +
		"completion text it saw on the event stream (e.g. the model's review text " +
		"from the code-review example).\n\n" +
		"With no run id the command targets your most recent run (recorded by the " +
		"last `zynax apply`/`run`). An explicit run id always overrides.\n\n" +
		"The command tails the run until it reaches a terminal state. A run that " +
		"COMPLETED but declared neither outputs nor completion text exits 0 with a " +
		"note; a FAILED or CANCELLED run exits non-zero. Output is sanitised of " +
		"control/ANSI escapes before printing.",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID, err := resolveRunID(args)
		if err != nil {
			return err
		}
		gw := newGateway()
		// Tail the run to a terminal state, capturing the legacy completion-text
		// fallback and the terminal status for the empty-result decision.
		var completion, terminalStatus string
		err = gw.WatchWorkflowLogs(cmd.Context(), runID, func(ev client.LogEvent) error {
			if text := client.CompletionText(ev.Payload); text != "" {
				completion = text
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

		// Prefer the structured declared outputs (ADR-042, O.8 gateway route). A
		// read error (older gateway, transient) is non-fatal — fall back to the
		// completion text so the command still works against a legacy gateway.
		if outputs, oErr := gw.GetWorkflowOutputs(cmd.Context(), runID); oErr == nil && len(outputs) > 0 {
			printOutputs(cmd.OutOrStdout(), outputs)
			return nil
		}
		// Fallback: the last capability completion text seen on the stream.
		if completion != "" {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), sanitizeForTTY(completion))
			return nil
		}
		// Neither structured outputs nor completion text. A COMPLETED run that
		// simply declared no output is a success — exit 0 with a note pointing at
		// the runbook. FAILED/CANCELLED, or a stream that closed before any
		// terminal state, remain errors so a real failure never reads as success.
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

// printOutputs writes the declared outputs as sanitised name=value lines, sorted
// by name for stable output.
func printOutputs(w io.Writer, outputs map[string]string) {
	keys := make([]string, 0, len(outputs))
	for k := range outputs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, _ = fmt.Fprintf(w, "%s=%s\n", sanitizeForTTY(k), sanitizeForTTY(outputs[k]))
	}
}

func init() {
	rootCmd.AddCommand(resultCmd)
}
