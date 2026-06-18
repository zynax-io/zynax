// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax/client"
)

var (
	logsFormat string
	logsFollow bool
)

// errFollowDone is the sentinel returned from the SSE callback to stop the
// stream cleanly once a terminal event is seen in --follow mode. It is
// swallowed at the command boundary and never surfaces as an error.
var errFollowDone = errors.New("follow: terminal state reached")

var logsCmd = &cobra.Command{
	Use:   "logs <run-id>",
	Short: "Stream lifecycle events for a workflow run",
	Long: "Stream lifecycle events (state transitions and capability events) for a " +
		"workflow run from the api-gateway SSE endpoint.\n\n" +
		"With --follow the command tails the run live and exits once the workflow " +
		"reaches a terminal state (completed, failed, or cancelled).",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gw := newGateway()
		err := gw.WatchWorkflowLogs(cmd.Context(), args[0], func(ev client.LogEvent) error {
			if logsFormat == "json" {
				b, mErr := json.Marshal(ev)
				if mErr != nil {
					return mErr
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(b))
			} else {
				printLogEvent(cmd, ev)
			}
			if logsFollow && terminalStatuses[ev.Status] {
				return errFollowDone
			}
			return nil
		})
		if errors.Is(err, errFollowDone) {
			return nil
		}
		return err
	},
}

// printLogEvent renders a single event in human-readable text form. State
// transitions show a from→to arrow; capability/lifecycle events without a
// transition show only the event type so progression reads cleanly. When a
// capability event carries a result payload (e.g. the model's review text), the
// completion is printed on an indented follow-up line so it is visible in the
// CLI without a DB query (#1378).
func printLogEvent(cmd *cobra.Command, ev client.LogEvent) {
	ts := ev.Timestamp
	if ts == "" {
		ts = "-"
	}
	out := cmd.OutOrStdout()
	if ev.FromState != "" || ev.ToState != "" {
		_, _ = fmt.Fprintf(out, "[%s] %-30s %s → %s (%s)\n",
			ts, ev.EventType, ev.FromState, ev.ToState, ev.Status)
		return
	}
	_, _ = fmt.Fprintf(out, "[%s] %-30s (%s)\n", ts, ev.EventType, ev.Status)
	if text := client.CompletionText(ev.Payload); text != "" {
		_, _ = fmt.Fprintf(out, "    output: %s\n", text)
	}
}

func init() {
	logsCmd.Flags().StringVar(&logsFormat, "format", "text", "output format: text|json")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "tail the run live until it reaches a terminal state")
	rootCmd.AddCommand(logsCmd)
}
