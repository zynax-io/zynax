// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/spf13/cobra"
)

// publishUse is the shared Use string for the publish-style aliases (publish,
// workflow publish, agent publish) — they all submit a file or scenario dir.
const publishUse = "publish <file|scenario-dir>"

// publishCmd is a thin, top-level alias over `apply`. It submits a YAML manifest
// (Workflow or AgentDef) to the api-gateway; the manifest kind is auto-detected
// on the apply path exactly as `zynax apply` does, so `publish` adds no behaviour
// of its own — it only gives the quickstart a noun-free "ship it" verb (canvas O20).
var publishCmd = &cobra.Command{
	Use:     publishUse,
	Short:   "Publish a manifest (Workflow or AgentDef) — alias for apply",
	Long:    "Publish a YAML manifest to the api-gateway. The manifest kind is\nauto-detected, exactly like `zynax apply`. This is a thin alias kept for the\nquickstart's noun-first mental model; `apply` remains available unchanged.",
	Args:    cobra.ExactArgs(1),
	GroupID: beginnerGroupID,
	RunE:    applyCmd.RunE,
}

func init() {
	// Re-register apply's flags so `publish --dry-run/--engine` behave identically;
	// the RunE reads the same package-level vars (applyDryRun/applyEngine).
	publishCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "validate manifest without submitting")
	publishCmd.Flags().StringVar(&applyEngine, "engine", "", "engine hint forwarded to SubmitWorkflow (ADR-015)")
	rootCmd.AddCommand(publishCmd)
}
