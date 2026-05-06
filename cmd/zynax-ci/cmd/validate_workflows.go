// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

var workflowsSchemaDir string
var workflowsFormat string

var validateWorkflowsCmd = &cobra.Command{
	Use:   "workflows <dir>",
	Short: "Validate Workflow YAML manifests against the JSON Schema",
	Long: `Walk <dir> for *.yaml files, filter those with kind: Workflow,
and validate each against spec/schemas/workflow.schema.json (or --schema-dir).

Exits 0 if all manifests pass, 1 on any schema violation.`,
	Args: cobra.ExactArgs(1),
	RunE: runValidateWorkflows,
}

func init() {
	validateWorkflowsCmd.Flags().StringVar(&workflowsSchemaDir, "schema-dir", "spec/schemas", "directory containing JSON Schema files")
	validateWorkflowsCmd.Flags().StringVar(&workflowsFormat, "format", "text", "output format: text or json")
	validateCmd.AddCommand(validateWorkflowsCmd)
}

func runValidateWorkflows(cmd *cobra.Command, args []string) error {
	dir := args[0]
	schemaPath := filepath.Join(workflowsSchemaDir, "workflow.schema.json")

	results, err := validate.BatchManifests(dir, "Workflow", schemaPath)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "(no Workflow manifests found in %s)\n", dir)
		return nil
	}
	return printManifestResults(cmd, results, workflowsFormat)
}

func printManifestResults(cmd *cobra.Command, results []validate.ManifestResult, format string) error {
	failed := false
	for _, r := range results {
		if len(r.Errors) > 0 {
			failed = true
		}
	}

	if format == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
		if failed {
			return fmt.Errorf("manifest validation failed")
		}
		return nil
	}

	for _, r := range results {
		if len(r.Errors) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "FAIL %s:\n", r.File)
			for _, e := range r.Errors {
				if e.Path != "" {
					fmt.Fprintf(cmd.ErrOrStderr(), "  %s: %s\n", e.Path, e.Message)
				} else {
					fmt.Fprintf(cmd.ErrOrStderr(), "  ERROR  %s\n", e.Message)
				}
			}
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  OK   %s\n", r.File)
		}
	}
	if failed {
		return fmt.Errorf("manifest validation failed")
	}
	return nil
}
