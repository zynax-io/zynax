// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var eventData []string

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Inject events into running workflows",
}

var eventsPublishCmd = &cobra.Command{
	Use:   "publish <run-id> <event-type>",
	Short: "Publish a lifecycle/business event to a run to drive event-driven workflows",
	Long: "Publish an event (e.g. review.approved, merge.success) to a running workflow\n" +
		"via the api-gateway. Event-driven workflows transition on the event type, so\n" +
		"this drives loops like code-review review → fix → merge → done from the CLI.\n\n" +
		"Attach payload fields with repeated --data key=value flags.",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := parseDataFlags(eventData)
		if err != nil {
			return err
		}
		gw := newGateway()
		eventID, err := gw.PublishEvent(cmd.Context(), args[0], args[1], data)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "published %s to %s\n", args[1], args[0])
		if eventID != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "event_id: %s\n", eventID)
		}
		return nil
	},
}

// parseDataFlags converts repeated "key=value" strings into a map. An entry
// without "=" or with an empty key is rejected. Returns nil when no flags are
// given so the payload is omitted entirely.
func parseDataFlags(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	data := make(map[string]string, len(pairs))
	for _, p := range pairs {
		key, val, found := strings.Cut(p, "=")
		if !found || key == "" {
			return nil, fmt.Errorf("invalid --data %q: expected key=value", p)
		}
		data[key] = val
	}
	return data, nil
}

func init() {
	eventsPublishCmd.Flags().StringArrayVar(&eventData, "data", nil, "event payload field as key=value (repeatable)")
	eventsCmd.AddCommand(eventsPublishCmd)
	rootCmd.AddCommand(eventsCmd)
}
