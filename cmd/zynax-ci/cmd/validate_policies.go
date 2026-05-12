// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

var policiesSchemaDir string
var policiesFormat string

var validatePoliciesCmd = &cobra.Command{
	Use:   "policies <dir>",
	Short: "Validate Policy YAML manifests against the JSON Schema",
	Long: `Walk <dir> for *.yaml files, filter those with kind: Policy,
and validate each against spec/schemas/policy.schema.json (or --schema-dir).

Exits 0 if all manifests pass, 1 on any schema violation.`,
	Args: cobra.ExactArgs(1),
	RunE: runValidatePolicies,
}

func init() {
	validatePoliciesCmd.Flags().StringVar(&policiesSchemaDir, "schema-dir", "spec/schemas", "directory containing JSON Schema files")
	validatePoliciesCmd.Flags().StringVar(&policiesFormat, "format", "text", "output format: text or json")
	validateCmd.AddCommand(validatePoliciesCmd)
}

func runValidatePolicies(cmd *cobra.Command, args []string) error {
	dir := args[0]
	schemaPath := filepath.Join(policiesSchemaDir, "policy.schema.json")

	results, err := validate.BatchManifests(dir, "Policy", schemaPath)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "(no Policy manifests found in %s)\n", dir)
		return nil
	}
	return printManifestResults(cmd, results, policiesFormat)
}
