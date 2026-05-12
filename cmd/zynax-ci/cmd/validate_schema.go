// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

var schemaFormat string

var validateSchemaCmd = &cobra.Command{
	Use:   "schema <path>",
	Short: "Validate JSON Schema files for well-formedness and $schema field",
	Long: `Validate one JSON Schema file or all *.json files in a directory.

Each file is checked for:
  • Valid JSON syntax
  • Presence of the required $schema field

Exits 0 if all files pass, 1 on any failure.`,
	Args: cobra.ExactArgs(1),
	RunE: runValidateSchema,
}

func init() {
	validateSchemaCmd.Flags().StringVar(&schemaFormat, "format", "text", "output format: text or json")
	validateCmd.AddCommand(validateSchemaCmd)
}

func runValidateSchema(cmd *cobra.Command, args []string) error {
	path := args[0]

	results, err := validate.SchemaDir(path)
	if err != nil {
		// path may be a single file
		errs, ferr := validate.Schema(path)
		if ferr != nil {
			return ferr
		}
		results = []validate.SchemaResult{{File: path, Errors: errs}}
	}

	if len(results) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "(no JSON schema files found under %s)\n", path)
		return nil
	}

	failed := false
	for _, r := range results {
		if len(r.Errors) > 0 {
			failed = true
		}
	}

	if schemaFormat == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
		if failed {
			return fmt.Errorf("schema validation failed")
		}
		return nil
	}

	for _, r := range results {
		if len(r.Errors) > 0 {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "FAIL %s:\n", r.File)
			for _, e := range r.Errors {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  ERROR  %s\n", e.Message)
			}
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  OK   %s\n", r.File)
		}
	}
	if failed {
		return fmt.Errorf("schema validation failed")
	}
	return nil
}
