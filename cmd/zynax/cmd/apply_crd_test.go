// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// crdTestEngine is the engine hint used by the CRD-apply tests (a single literal
// keeps goconst quiet across the CLI test package).
const crdTestEngine = "argo"

const workflowManifest = `apiVersion: zynax.io/v1
kind: Workflow
metadata:
  name: review
  version: 1.2.0
spec:
  initial_state: start
  states:
    start:
      type: normal
      on:
        - event: done
          goto: end
    end:
      type: terminal
`

func TestWorkflowManifestToCR_MapsAndInjects(t *testing.T) {
	cr, err := workflowManifestToCR([]byte(workflowManifest), "temporal")
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(cr, &m); err != nil {
		t.Fatalf("CR is not valid JSON: %v", err)
	}

	// apiVersion maps v1 -> v1alpha1 (ADR-043 §5).
	if m["apiVersion"] != "zynax.io/v1alpha1" {
		t.Fatalf("apiVersion = %v, want zynax.io/v1alpha1", m["apiVersion"])
	}
	if m["kind"] != "Workflow" {
		t.Fatalf("kind = %v, want Workflow", m["kind"])
	}

	meta, _ := m["metadata"].(map[string]any)
	if meta["name"] != "review" {
		t.Fatalf("metadata.name = %v, want review", meta["name"])
	}
	if _, ok := meta["version"]; ok {
		t.Fatalf("metadata.version must move to spec.version, still present: %v", meta)
	}

	spec, _ := m["spec"].(map[string]any)
	if spec["engine"] != "temporal" {
		t.Fatalf("spec.engine = %v, want temporal (from --engine)", spec["engine"])
	}
	if spec["version"] != "1.2.0" {
		t.Fatalf("spec.version = %v, want 1.2.0", spec["version"])
	}

	// The `on` transition key survives as a string (the whole reason JSON is
	// emitted — a YAML CR would need it quoted to dodge the boolean trap).
	states, _ := spec["states"].(map[string]any)
	start, _ := states["start"].(map[string]any)
	if _, ok := start["on"]; !ok {
		t.Fatalf("states.start.on lost in conversion: %v", start)
	}
}

func TestWorkflowManifestToCR_NoEngineHint(t *testing.T) {
	cr, err := workflowManifestToCR([]byte(workflowManifest), "")
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	var m map[string]any
	_ = json.Unmarshal(cr, &m)
	spec, _ := m["spec"].(map[string]any)
	if _, ok := spec["engine"]; ok {
		t.Fatalf("no --engine given: spec.engine must be omitted, got %v", spec["engine"])
	}
}

func TestWorkflowManifestToCR_RejectsNonWorkflow(t *testing.T) {
	if _, err := workflowManifestToCR([]byte("kind: AgentDef\nmetadata:\n  name: x\n"), ""); err == nil {
		t.Fatal("expected an error for a non-Workflow manifest")
	}
}

func TestWorkflowManifestToCR_RequiresName(t *testing.T) {
	if _, err := workflowManifestToCR([]byte("kind: Workflow\nspec:\n  initial_state: s\n"), ""); err == nil {
		t.Fatal("expected an error for a manifest with no metadata.name")
	}
}

func TestRunApplyCRD_InvokesKubectlWithCR(t *testing.T) {
	var gotManifest []byte
	var gotDryRun bool
	orig := kubectlApply
	kubectlApply = func(_ context.Context, m []byte, dryRun bool) (string, error) {
		gotManifest, gotDryRun = m, dryRun
		return "workflow.zynax.io/review created", nil
	}
	t.Cleanup(func() { kubectlApply = orig })

	oe, od, oc := applyEngine, applyDryRun, applyCRD
	applyEngine, applyDryRun, applyCRD = crdTestEngine, false, true
	t.Cleanup(func() { applyEngine, applyDryRun, applyCRD = oe, od, oc })

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())

	if err := runApplyCRD(cmd, []byte(workflowManifest)); err != nil {
		t.Fatalf("runApplyCRD: %v", err)
	}
	if gotDryRun {
		t.Fatal("dryRun should be false")
	}
	var cr map[string]any
	if err := json.Unmarshal(gotManifest, &cr); err != nil {
		t.Fatalf("kubectl received non-JSON: %v", err)
	}
	if cr["apiVersion"] != "zynax.io/v1alpha1" {
		t.Fatalf("kubectl CR apiVersion = %v", cr["apiVersion"])
	}
	if spec, _ := cr["spec"].(map[string]any); spec["engine"] != crdTestEngine {
		t.Fatalf("kubectl CR spec.engine = %v, want %s", spec["engine"], crdTestEngine)
	}
	if !strings.Contains(out.String(), "created") {
		t.Fatalf("kubectl output not forwarded to the user: %q", out.String())
	}
}
