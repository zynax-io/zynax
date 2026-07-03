// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// crWorkflowAPIVersion is the CRD apiVersion (ADR-043). The Workflow *manifest*
// pins zynax.io/v1; the CR served by the cluster is v1alpha1, so `--crd` maps
// v1 -> v1alpha1 when it emits the custom resource.
const crWorkflowAPIVersion = "zynax.io/v1alpha1"

// kubectlApply pipes a manifest to `kubectl apply -f -` on the current context.
// Injected so tests run with a fake — no real kubectl or cluster. Per
// cmd/zynax/AGENTS.md the CLI shells out to the user's kubectl (no client-go).
var kubectlApply = execKubectlApply

func execKubectlApply(ctx context.Context, manifest []byte, dryRun bool) (string, error) {
	args := []string{"apply", "-f", "-"}
	if dryRun {
		args = append(args, "--dry-run=client")
	}
	cmd := exec.CommandContext(ctx, "kubectl", args...) //nolint:gosec // G204: fixed binary, static args; manifest arrives on stdin
	cmd.Stdin = bytes.NewReader(manifest)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// runApplyCRD applies a Workflow manifest as a Workflow custom resource on the
// current Kubernetes context (ADR-043, M8.E) — the declarative GitOps front-end.
// It reuses the same manifest body: the controller reconciles the CR through the
// very same compile->submit path the REST apply uses, so the workflow stays
// engine-portable and run state lives in the engine, not the CR.
func runApplyCRD(cmd *cobra.Command, data []byte) error {
	cr, err := workflowManifestToCR(data, applyEngine)
	if err != nil {
		return err
	}
	out, err := kubectlApply(cmd.Context(), cr, applyDryRun)
	if out != "" {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), out)
	}
	if err != nil {
		return fmt.Errorf("kubectl apply: %w", err)
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(),
		"applied as a Workflow custom resource — commit it to git for GitOps sync; "+
			"the controller reconciles it through compile->submit, and run state stays in the engine.")
	return nil
}

// workflowManifestToCR converts a `kind: Workflow` manifest into a Workflow
// custom resource, emitted as JSON. JSON (not YAML) is deliberate: kubectl's
// YAML reader treats a bare `on:` transition key as a boolean, so a YAML CR
// would need every `on` quoted; JSON keys are always strings, sidestepping the
// trap. The manifest apiVersion (zynax.io/v1) maps to the CR's v1alpha1; the
// engine hint and the manifest version move into the CR spec.
func workflowManifestToCR(manifest []byte, engineHint string) ([]byte, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(manifest, &doc); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if kind, _ := doc["kind"].(string); kind != kindWorkflow {
		return nil, fmt.Errorf("--crd applies only kind: Workflow custom resources (got %q)", kind)
	}

	crMeta := map[string]any{}
	var version string
	if meta, ok := doc["metadata"].(map[string]any); ok {
		for k, v := range meta {
			if k == "version" {
				version, _ = v.(string)
				continue
			}
			crMeta[k] = v
		}
	}
	if _, ok := crMeta["name"]; !ok {
		return nil, fmt.Errorf("workflow manifest has no metadata.name")
	}

	crSpec := map[string]any{}
	if spec, ok := doc["spec"].(map[string]any); ok {
		for k, v := range spec {
			crSpec[k] = v
		}
	}
	if engineHint != "" {
		crSpec["engine"] = engineHint
	}
	if version != "" {
		crSpec["version"] = version
	}

	out, err := json.MarshalIndent(map[string]any{
		"apiVersion": crWorkflowAPIVersion,
		"kind":       kindWorkflow,
		"metadata":   crMeta,
		"spec":       crSpec,
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal workflow CR: %w", err)
	}
	return out, nil
}
