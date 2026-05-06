// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

var agentDefsSchemaDir string
var agentDefsFormat string

var validateAgentDefsCmd = &cobra.Command{
	Use:   "agent-defs <dir>",
	Short: "Validate AgentDef YAML manifests against the JSON Schema",
	Long: `Walk <dir> for *.yaml files, filter those with kind: AgentDef,
and validate each against spec/schemas/agent-def.schema.json (or --schema-dir).

Exits 0 if all manifests pass, 1 on any schema violation.`,
	Args: cobra.ExactArgs(1),
	RunE: runValidateAgentDefs,
}

func init() {
	validateAgentDefsCmd.Flags().StringVar(&agentDefsSchemaDir, "schema-dir", "spec/schemas", "directory containing JSON Schema files")
	validateAgentDefsCmd.Flags().StringVar(&agentDefsFormat, "format", "text", "output format: text or json")
	validateCmd.AddCommand(validateAgentDefsCmd)
}

func runValidateAgentDefs(cmd *cobra.Command, args []string) error {
	dir := args[0]
	schemaPath := filepath.Join(agentDefsSchemaDir, "agent-def.schema.json")

	results, err := validate.BatchManifests(dir, "AgentDef", schemaPath)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "(no AgentDef manifests found in %s)\n", dir)
		return nil
	}
	return printManifestResults(cmd, results, agentDefsFormat)
}
