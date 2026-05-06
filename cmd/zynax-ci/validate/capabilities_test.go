// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

// writeCapabilitySchema writes a minimal capability schema to dir and returns its path.
func writeCapabilitySchema(t *testing.T, dir string) string {
	t.Helper()
	schema := map[string]interface{}{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id":     "https://test.zynax.io/capability",
		"type":    "object",
		"required": []string{"name"},
		"properties": map[string]interface{}{
			"name":        map[string]interface{}{"type": "string"},
			"description": map[string]interface{}{"type": "string"},
		},
	}
	b, _ := json.Marshal(schema)
	path := filepath.Join(dir, "capability.schema.json")
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCapabilities_ValidSpecCapabilities(t *testing.T) {
	dir := t.TempDir()
	schema := writeCapabilitySchema(t, dir)
	writeYAML(t, dir, "agent.yaml", `
kind: AgentDef
spec:
  capabilities:
    - name: summarize
      description: Summarise text
`)
	results, err := validate.Capabilities(dir, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 capability result, got %d", len(results))
	}
	if len(results[0].Errors) != 0 {
		t.Errorf("expected no errors, got %v", results[0].Errors)
	}
	if results[0].Capability != "summarize" {
		t.Errorf("expected capability name 'summarize', got %q", results[0].Capability)
	}
}

func TestCapabilities_TopLevelCapabilities(t *testing.T) {
	dir := t.TempDir()
	schema := writeCapabilitySchema(t, dir)
	writeYAML(t, dir, "agent.yaml", `
capabilities:
  - name: legacy-cap
`)
	results, err := validate.Capabilities(dir, schema)
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

func TestCapabilities_MissingRequiredField(t *testing.T) {
	dir := t.TempDir()
	schema := writeCapabilitySchema(t, dir)
	writeYAML(t, dir, "agent.yaml", `
spec:
  capabilities:
    - description: no name field here
`)
	results, err := validate.Capabilities(dir, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(results[0].Errors) == 0 {
		t.Error("expected schema error for missing 'name' field")
	}
}

func TestCapabilities_NoCapabilities(t *testing.T) {
	dir := t.TempDir()
	schema := writeCapabilitySchema(t, dir)
	// A YAML file with no capability fields at all
	writeYAML(t, dir, "workflow.yaml", `
kind: Workflow
spec:
  states: []
`)
	results, err := validate.Capabilities(dir, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for file with no capabilities, got %d", len(results))
	}
}

func TestCapabilities_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	schema := writeCapabilitySchema(t, dir)
	results, err := validate.Capabilities(dir, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results in empty dir, got %d", len(results))
	}
}
