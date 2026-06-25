// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

// workflowCmd is the noun-first parent for workflow operations. Its subcommands
// are thin aliases that DELEGATE to the existing verb commands' RunE — no logic
// is duplicated (canvas O20). `zynax workflow init <name>` is `zynax init
// workflow <name>`; `zynax workflow run|publish <file>` is `zynax apply <file>`
// with kind auto-detection. The underlying `init workflow` and `apply` verbs
// remain available unchanged.
var workflowCmd = &cobra.Command{
	Use:     "workflow",
	Short:   "Create, run, and publish workflows",
	Long:    "Noun-first aliases for workflow operations.\n\n`workflow init` scaffolds a Workflow manifest (alias for `init workflow`);\n`workflow run` and `workflow publish` submit it to the api-gateway (aliases for\n`apply`). The underlying `init workflow` and `apply` verbs remain available.",
	GroupID: beginnerGroupID,
}

var workflowInitCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Scaffold a Workflow manifest — alias for `init workflow`",
	Args:  cobra.MaximumNArgs(1),
	RunE:  initWorkflowCmd.RunE,
}

var workflowRunCmd = &cobra.Command{
	Use:   "run <file|scenario-dir>",
	Short: "Run a Workflow manifest — alias for `apply`",
	Args:  cobra.ExactArgs(1),
	RunE:  applyCmd.RunE,
}

var workflowPublishCmd = &cobra.Command{
	Use:   publishUse,
	Short: "Publish a Workflow manifest — alias for `apply`",
	Args:  cobra.ExactArgs(1),
	RunE:  applyCmd.RunE,
}

func init() {
	// `workflow init` reuses runInit's package vars (initTemplateDir/initOutput);
	// register the same flags with the same defaults so it behaves identically.
	workflowInitCmd.Flags().StringVar(&initTemplateDir, "template-dir", "spec/templates",
		"directory containing <kind>/<kind>.template.yaml files")
	workflowInitCmd.Flags().StringVarP(&initOutput, "output", "o", "",
		"write the manifest to this file instead of stdout")
	// `workflow run|publish` reuse apply's package vars (applyDryRun/applyEngine).
	for _, c := range []*cobra.Command{workflowRunCmd, workflowPublishCmd} {
		c.Flags().BoolVar(&applyDryRun, "dry-run", false, "validate manifest without submitting")
		c.Flags().StringVar(&applyEngine, "engine", "", "engine hint forwarded to SubmitWorkflow (ADR-015)")
	}

	workflowCmd.AddCommand(workflowInitCmd)
	workflowCmd.AddCommand(workflowRunCmd)
	workflowCmd.AddCommand(workflowPublishCmd)
	rootCmd.AddCommand(workflowCmd)
}
