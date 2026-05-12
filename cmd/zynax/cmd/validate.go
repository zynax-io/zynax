// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax/validate"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate manifests, Canvases, and JSON Schemas",
}

var (
	validateSchemaDir string
	validateFormat    string
)

var validateManifestCmd = &cobra.Command{
	Use:   "manifest <file>",
	Short: "Validate a YAML manifest against its JSON schema",
	Long: `Validate <file> against the JSON Schema for its kind.

The kind is read from the 'kind:' field in the YAML. The matching schema is
loaded from <schema-dir>/<kind>.schema.json. Supported kinds: Workflow, AgentDef, Policy.

Exits 0 on success, 1 on any validation error.`,
	Args: cobra.ExactArgs(1),
	RunE: runValidateManifest,
}

func init() {
	validateManifestCmd.Flags().StringVar(&validateSchemaDir, "schema-dir", "spec/schemas",
		"directory containing <kind>.schema.json files")
	validateManifestCmd.Flags().StringVar(&validateFormat, "format", "text",
		"output format: text or json")
	validateCmd.AddCommand(validateManifestCmd)
	rootCmd.AddCommand(validateCmd)
}

func runValidateManifest(cmd *cobra.Command, args []string) error {
	file := args[0]
	errs, err := validate.Manifest(file, validateSchemaDir)
	if err != nil {
		return err
	}
	if validateFormat == "json" {
		return printValidateJSON(cmd, errs)
	}
	return printValidateText(cmd, file, errs)
}

func printValidateText(cmd *cobra.Command, file string, errs []validate.ValidationError) error {
	if len(errs) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ok: %s\n", file)
		return nil
	}
	for _, e := range errs {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), e.Error())
	}
	return fmt.Errorf("validation failed with %d error(s)", len(errs))
}

func printValidateJSON(cmd *cobra.Command, errs []validate.ValidationError) error {
	if errs == nil {
		errs = []validate.ValidationError{}
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	_ = enc.Encode(errs)
	if len(errs) > 0 {
		return fmt.Errorf("validation failed with %d error(s)", len(errs))
	}
	return nil
}
