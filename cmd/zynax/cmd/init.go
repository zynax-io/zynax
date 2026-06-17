// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"
)

// initTemplateDir is the directory holding <kind>/<kind>.template.yaml files.
var (
	initTemplateDir string
	initOutput      string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a new manifest from a reusable template",
	Long: `Scaffold a new, valid, versioned manifest from a reusable template.

Templates ship under <template-dir>/<kind>/<kind>.template.yaml (EPIC T.1).
The scaffolded manifest is written to stdout by default, or to --output.

If [name] is given it replaces the metadata.name field in the emitted manifest;
otherwise the template's baseline name is kept.

Runs entirely locally — no api-gateway connection required.`,
}

var initWorkflowCmd = &cobra.Command{
	Use:   "workflow [name]",
	Short: "Scaffold a new Workflow manifest",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(cmd, "workflow", args)
	},
}

var initExpertCmd = &cobra.Command{
	Use:   "expert [name]",
	Short: "Scaffold a new expert (AgentDef) manifest",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(cmd, "expert", args)
	},
}

func init() {
	initCmd.PersistentFlags().StringVar(&initTemplateDir, "template-dir", "spec/templates",
		"directory containing <kind>/<kind>.template.yaml files")
	initCmd.PersistentFlags().StringVarP(&initOutput, "output", "o", "",
		"write the manifest to this file instead of stdout")
	initCmd.AddCommand(initWorkflowCmd)
	initCmd.AddCommand(initExpertCmd)
	rootCmd.AddCommand(initCmd)
}

// runInit reads the template for kind, applies the optional name override, and
// writes the result to --output or stdout.
func runInit(cmd *cobra.Command, kind string, args []string) error {
	manifest, err := scaffold(initTemplateDir, kind, optionalName(args))
	if err != nil {
		return err
	}
	if initOutput != "" {
		if err := os.WriteFile(initOutput, manifest, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", initOutput, err)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s manifest to %s\n", kind, initOutput)
		return nil
	}
	_, _ = cmd.OutOrStdout().Write(manifest)
	return nil
}

// optionalName returns the first positional arg, or "" when none is given.
func optionalName(args []string) string {
	if len(args) == 1 {
		return args[0]
	}
	return ""
}

// metadataName matches the first metadata-level "  name: <value>" line.
var metadataName = regexp.MustCompile(`(?m)^(\s{2}name:).*$`)

// scaffold reads the template for kind from templateDir and returns the manifest
// bytes with the metadata.name overridden by name when name is non-empty.
func scaffold(templateDir, kind, name string) ([]byte, error) {
	path := filepath.Join(templateDir, kind, kind+".template.yaml")
	body, err := os.ReadFile(path) //nolint:gosec // path built from a fixed flag + kind constant
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", path, err)
	}
	if name == "" {
		return body, nil
	}
	if !metadataName.Match(body) {
		return nil, fmt.Errorf("template %s has no metadata.name field to override", path)
	}
	return metadataName.ReplaceAll(body, []byte("${1} "+name)), nil
}
