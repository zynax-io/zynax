// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

const capabilitiesFormatJSON = "json"

var capabilitiesSchemaDir string
var capabilitiesFormat string

var validateCapabilitiesCmd = &cobra.Command{
	Use:   "capabilities <dir>",
	Short: "Validate capability declarations in AgentDef YAML files",
	Long: `Walk <dir> for *.yaml files, extract capability declarations from
AgentDef manifests (spec.capabilities or top-level capabilities), and validate
each against spec/schemas/capability.schema.json (or --schema-dir).

Exits 0 if all capabilities pass, 1 on any schema violation.`,
	Args: cobra.ExactArgs(1),
	RunE: runValidateCapabilities,
}

func init() {
	validateCapabilitiesCmd.Flags().StringVar(&capabilitiesSchemaDir, "schema-dir", "spec/schemas", "directory containing JSON Schema files")
	validateCapabilitiesCmd.Flags().StringVar(&capabilitiesFormat, "format", "text", "output format: text or json")
	validateCmd.AddCommand(validateCapabilitiesCmd)
}

func runValidateCapabilities(cmd *cobra.Command, args []string) error {
	dir := args[0]
	schemaPath := filepath.Join(capabilitiesSchemaDir, "capability.schema.json")

	results, err := validate.Capabilities(dir, schemaPath)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "(no capability declarations found in %s)\n", dir)
		return nil
	}

	failed := false
	for _, r := range results {
		if len(r.Errors) > 0 {
			failed = true
		}
	}

	if capabilitiesFormat == capabilitiesFormatJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
		if failed {
			return fmt.Errorf("capability validation failed")
		}
		return nil
	}

	for _, r := range results {
		if len(r.Errors) > 0 {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "FAIL %s capability %q:\n", r.File, r.Capability)
			for _, e := range r.Errors {
				if e.Path != "" {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  %s: %s\n", e.Path, e.Message)
				} else {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  ERROR  %s\n", e.Message)
				}
			}
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  OK   %s :: %s\n", r.File, r.Capability)
		}
	}
	if failed {
		return fmt.Errorf("capability validation failed")
	}
	return nil
}
