// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

var canvasFormat string

var validateCanvasCmd = &cobra.Command{
	Use:   "canvas <path>",
	Short: "Validate REASONS Canvas files for structural completeness",
	Long: `Walk <path> recursively and validate every canvas.md file found.

Checks per canvas:
  • Seven REASONS sections (R, E, A, S-structure, O, N, S-safeguards)
  • Header fields: **Issue:**, **Author:**, **Date:**, **Status:**
  • Status value one of: Draft, Aligned, Implemented, Synced
  • Context Security checklist marker present

Status: Draft emits a warning but does not fail.
Exits 0 if all canvas files pass, 1 on any structural error.`,
	Args: cobra.ExactArgs(1),
	RunE: runValidateCanvas,
}

func init() {
	validateCanvasCmd.Flags().StringVar(&canvasFormat, "format", "text", "output format: text or json")
	validateCmd.AddCommand(validateCanvasCmd)
}

type canvasResult struct {
	File     string                       `json:"file"`
	Errors   []validate.ValidationError   `json:"errors,omitempty"`
	Warnings []validate.ValidationWarning `json:"warnings,omitempty"`
}

func runValidateCanvas(cmd *cobra.Command, args []string) error {
	root := args[0]

	canvases, err := findCanvases(root)
	if err != nil {
		return err
	}
	if len(canvases) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "(no canvas.md files found under %s)\n", root)
		return nil
	}

	var results []canvasResult
	failed := false

	for _, path := range canvases {
		errs, warns, valErr := validate.Canvas(path)
		if valErr != nil {
			return valErr
		}
		if len(errs) > 0 {
			failed = true
		}
		results = append(results, canvasResult{File: path, Errors: errs, Warnings: warns})
	}

	if canvasFormat == "json" {
		return printCanvasJSON(cmd, results, failed)
	}
	return printCanvasText(cmd, results, failed)
}

func printCanvasText(cmd *cobra.Command, results []canvasResult, failed bool) error {
	for _, r := range results {
		if len(r.Errors) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "FAIL %s:\n", r.File)
			for _, e := range r.Errors {
				fmt.Fprintf(cmd.ErrOrStderr(), "  ERROR  %s\n", e.Message)
			}
			for _, w := range r.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "  WARN   %s\n", w.Message)
			}
		} else if len(r.Warnings) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  WARN %s:\n", r.File)
			for _, w := range r.Warnings {
				fmt.Fprintf(cmd.OutOrStdout(), "         %s\n", w.Message)
			}
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  OK   %s\n", r.File)
		}
	}
	if failed {
		return fmt.Errorf("canvas validation failed")
	}
	return nil
}

func printCanvasJSON(cmd *cobra.Command, results []canvasResult, failed bool) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	_ = enc.Encode(results)
	if failed {
		return fmt.Errorf("canvas validation failed")
	}
	return nil
}

func findCanvases(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("validate canvas: %w", err)
	}
	if !info.IsDir() {
		return []string{root}, nil
	}
	var paths []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() && d.Name() == "canvas.md" {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}
