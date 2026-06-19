// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

var scenariosSchemaDir string
var scenariosFormat string

var validateScenariosCmd = &cobra.Command{
	Use:   "scenarios <dir>",
	Short: "Validate Scenario index YAML files against the JSON Schema",
	Long: `Walk <dir> for *.yaml files, filter those with kind: Scenario,
and validate each against spec/schemas/scenario.schema.json (or --schema-dir).

A Scenario index is the client-side manifest-set convention (A2, ADR-028):
it lists the member Workflow/AgentDef manifests, their apply order, and a
reserved context slot. This validates only the INDEX shape; the member
manifests are validated by the workflows/agent-defs subcommands.

Exits 0 if all index files pass, 1 on any schema violation.`,
	Args: cobra.ExactArgs(1),
	RunE: runValidateScenarios,
}

func init() {
	validateScenariosCmd.Flags().StringVar(&scenariosSchemaDir, "schema-dir", "spec/schemas", "directory containing JSON Schema files")
	validateScenariosCmd.Flags().StringVar(&scenariosFormat, "format", "text", "output format: text or json")
	validateCmd.AddCommand(validateScenariosCmd)
}

func runValidateScenarios(cmd *cobra.Command, args []string) error {
	dir := args[0]
	schemaPath := filepath.Join(scenariosSchemaDir, "scenario.schema.json")

	results, err := validate.BatchManifests(dir, "Scenario", schemaPath)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "(no Scenario index files found in %s)\n", dir)
		return nil
	}
	return printManifestResults(cmd, results, scenariosFormat)
}
