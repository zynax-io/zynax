// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

// Canonical, repo-relative display names — kept stable so output matches the
// former scripts/validate_milestone_state.py regardless of --root.
const (
	milestoneStateName  = "state/milestone.yaml"
	milestoneSchemaName = "state/milestone.schema.json"
)

var milestoneRoot string

var validateMilestoneCmd = &cobra.Command{
	Use:   "milestone",
	Short: "Validate state/milestone.yaml against its JSON Schema",
	Long: `Validate state/milestone.yaml against state/milestone.schema.json
(Draft 2020-12). state/milestone.yaml is updated only by /milestone-close and
/milestone-new, never by hand; this is the CI gate for that output.

Exits 0 when the file conforms, 1 on any schema violation.`,
	Args: cobra.NoArgs,
	RunE: runValidateMilestone,
}

func init() {
	validateMilestoneCmd.Flags().StringVar(&milestoneRoot, "root", ".", "repository root directory")
	validateCmd.AddCommand(validateMilestoneCmd)
}

func runValidateMilestone(cmd *cobra.Command, _ []string) error {
	root, err := resolveRoot(milestoneRoot)
	if err != nil {
		return fmt.Errorf("validate milestone: %w", err)
	}
	stateFile := filepath.Join(root, milestoneStateName)
	schemaFile := filepath.Join(root, milestoneSchemaName)

	errs, err := validate.Milestone(stateFile, schemaFile)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if len(errs) > 0 {
		for _, e := range errs {
			path := strings.TrimPrefix(e.Path, "/")
			if path == "" {
				path = "<root>"
			}
			_, _ = fmt.Fprintf(out, "FAIL %s: %s: %s\n", milestoneStateName, path, e.Message)
		}
		return errors.New("milestone state does not conform to schema")
	}
	_, _ = fmt.Fprintf(out, "OK %s conforms to %s\n", milestoneStateName, milestoneSchemaName)
	return nil
}
