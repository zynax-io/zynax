// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

// agentCmd is the noun-first parent for agent (expert) operations. Its
// subcommands are thin aliases that DELEGATE to the existing verb commands'
// RunE — no logic is duplicated (canvas O20). `zynax agent init <name>` is
// `zynax init expert <name>`; `zynax agent publish <file>` is `zynax apply
// <file>` with kind auto-detection.
var agentCmd = &cobra.Command{
	Use:     "agent",
	Short:   "Create and publish agents (experts)",
	Long:    "Noun-first aliases for agent (expert) operations.\n\n`agent init` scaffolds an AgentDef manifest (alias for `init expert`); `agent\npublish` submits it to the api-gateway (alias for `apply`). The underlying\n`init expert` and `apply` verbs remain available unchanged.",
	GroupID: beginnerGroupID,
}

var agentInitCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Scaffold an agent (expert) manifest — alias for `init expert`",
	Args:  cobra.MaximumNArgs(1),
	RunE:  initExpertCmd.RunE,
}

var agentPublishCmd = &cobra.Command{
	Use:   publishUse,
	Short: "Publish an agent manifest — alias for `apply`",
	Args:  cobra.ExactArgs(1),
	RunE:  applyCmd.RunE,
}

func init() {
	// `agent init` reuses runInit's package vars (initTemplateDir/initOutput);
	// register the same flags with the same defaults so it behaves identically.
	agentInitCmd.Flags().StringVar(&initTemplateDir, "template-dir", "spec/templates",
		"directory containing <kind>/<kind>.template.yaml files")
	agentInitCmd.Flags().StringVarP(&initOutput, "output", "o", "",
		"write the manifest to this file instead of stdout")
	// `agent publish` reuses apply's package vars (applyDryRun/applyEngine).
	agentPublishCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "validate manifest without submitting")
	agentPublishCmd.Flags().StringVar(&applyEngine, "engine", "", "engine hint forwarded to SubmitWorkflow (ADR-015)")

	agentCmd.AddCommand(agentInitCmd)
	agentCmd.AddCommand(agentPublishCmd)
	rootCmd.AddCommand(agentCmd)
}
