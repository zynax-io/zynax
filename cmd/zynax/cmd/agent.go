// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// agentCmd is the noun-first parent for agent (expert) operations. Its
// subcommands are thin aliases that DELEGATE to the existing verb commands'
// RunE — no logic is duplicated (canvas O20). `zynax agent init <name>` is
// `zynax init expert <name>`.
//
// `agent publish` was removed in M9.A (ADR-039): it was an alias for `apply`
// whose only reason to exist was pushing an AgentDef through the api-gateway,
// and that route is gone — agent identity is declared by the zynax.io/v1alpha1
// Agent custom resource (`kubectl apply`, docs/patterns/agent-crd-migration.md).
var agentCmd = &cobra.Command{
	Use:     "agent",
	Short:   "Scaffold agent (expert) manifests",
	Long:    "Noun-first alias for agent (expert) authoring.\n\n`agent init` scaffolds an AgentDef manifest (alias for `init expert`). The\nunderlying `init expert` verb remains available unchanged.\n\nAgents are NOT published through the api-gateway: declare identity with a\nzynax.io/v1alpha1 Agent custom resource (`kubectl apply -f agent.yaml`) — see\ndocs/patterns/agent-crd-migration.md.",
	GroupID: beginnerGroupID,
}

var agentInitCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Scaffold an agent (expert) manifest — alias for `init expert`",
	Args:  cobra.MaximumNArgs(1),
	RunE:  initExpertCmd.RunE,
}

// agentPublishCmd is a hidden retirement stub. The command is gone from help,
// but an existing script that still calls it must fail loudly with the
// replacement path rather than print help and exit 0 — a cobra parent with no
// Run answers an unknown subcommand with help + exit 0 (execute() returns
// flag.ErrHelp before ValidateArgs ever runs), which would silently no-op a
// deployment step.
var agentPublishCmd = &cobra.Command{
	Use:    publishUse,
	Short:  "Retired (ADR-039) — declare an Agent custom resource instead",
	Hidden: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		return errors.New(
			"`zynax agent publish` is retired (ADR-039): agent identity is no longer " +
				"pushed through the api-gateway. Declare a zynax.io/v1alpha1 Agent custom " +
				"resource and apply it with kubectl — see docs/patterns/agent-crd-migration.md")
	},
}

func init() {
	// `agent init` reuses runInit's package vars (initTemplateDir/initOutput);
	// register the same flags with the same defaults so it behaves identically.
	agentInitCmd.Flags().StringVar(&initTemplateDir, "template-dir", "spec/templates",
		"directory containing <kind>/<kind>.template.yaml files")
	agentInitCmd.Flags().StringVarP(&initOutput, "output", "o", "",
		"write the manifest to this file instead of stdout")

	agentCmd.AddCommand(agentInitCmd)
	agentCmd.AddCommand(agentPublishCmd)
	rootCmd.AddCommand(agentCmd)
}
