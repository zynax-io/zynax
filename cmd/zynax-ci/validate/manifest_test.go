// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

// minimalSchema returns a minimal Draft 2020-12 JSON Schema that requires
// kind, apiVersion, and metadata fields — good enough to exercise the validator.
func writeMinimalWorkflowSchema(t *testing.T, dir string) string {
	t.Helper()
	schema := map[string]interface{}{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id":     "https://test.zynax.io/workflow",
		"type":    "object",
		"required": []string{"kind", "apiVersion", "metadata"},
		"properties": map[string]interface{}{
			"kind":       map[string]interface{}{"type": "string", "const": "Workflow"},
			"apiVersion": map[string]interface{}{"type": "string"},
			"metadata":   map[string]interface{}{"type": "object"},
		},
	}
	b, _ := json.Marshal(schema)
	path := filepath.Join(dir, "workflow.schema.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeYAML(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBatchManifests_ValidWorkflow(t *testing.T) {
	dir := t.TempDir()
	schema := writeMinimalWorkflowSchema(t, dir)
	writeYAML(t, dir, "wf.yaml", `
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: test-wf
`)
	results, err := validate.BatchManifests(dir, "Workflow", schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Errors) != 0 {
		t.Errorf("expected no errors, got %v", results[0].Errors)
	}
}

func TestBatchManifests_InvalidWorkflow(t *testing.T) {
	dir := t.TempDir()
	schema := writeMinimalWorkflowSchema(t, dir)
	writeYAML(t, dir, "bad.yaml", `
kind: Workflow
apiVersion: zynax.io/v1
`)
	// metadata is required but missing
	results, err := validate.BatchManifests(dir, "Workflow", schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Errors) == 0 {
		t.Error("expected schema errors for missing metadata field")
	}
}

func TestBatchManifests_FiltersByKind(t *testing.T) {
	dir := t.TempDir()
	schema := writeMinimalWorkflowSchema(t, dir)
	// This file is AgentDef — should be skipped when validating Workflow kind
	writeYAML(t, dir, "agent.yaml", `
kind: AgentDef
apiVersion: zynax.io/v1
metadata:
  name: my-agent
`)
	results, err := validate.BatchManifests(dir, "Workflow", schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results (AgentDef should be filtered), got %d", len(results))
	}
}

func TestBatchManifests_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	schema := writeMinimalWorkflowSchema(t, dir)
	results, err := validate.BatchManifests(dir, "Workflow", schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results in empty dir, got %d", len(results))
	}
}

func TestBatchManifests_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	schema := writeMinimalWorkflowSchema(t, dir)
	writeYAML(t, dir, "bad.yaml", "kind: Workflow\n  bad_indent: {")
	results, err := validate.BatchManifests(dir, "Workflow", schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Invalid YAML with kind Workflow cannot be parsed — returns error entry
	if len(results) != 1 || len(results[0].Errors) == 0 {
		t.Errorf("expected error result for invalid YAML, got %v", results)
	}
}
