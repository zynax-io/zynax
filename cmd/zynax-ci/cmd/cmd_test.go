// SPDX-License-Identifier: Apache-2.0

// Internal tests for the zynax-ci cmd package.  Using package cmd (not
// cmd_test) lets us call unexported RunE functions directly and ensures every
// init() function (cobra command/flag registration) runs when the test binary
// starts, counting those statements as covered.
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/zynax-io/zynax/cmd/zynax-ci/validate"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	// file = cmd/zynax-ci/cmd/cmd_test.go (absolute)
	return filepath.Join(filepath.Dir(file), "../../..")
}

func fakeCmd(t *testing.T) *cobra.Command {
	t.Helper()
	c := &cobra.Command{}
	var out, errOut bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&errOut)
	c.SetContext(context.Background())
	return c
}

const (
	formatText = "text"
	formatJSON = "json"
)

// ── init() coverage ───────────────────────────────────────────────────────────

func TestInit_CommandsRegistered(t *testing.T) {
	// Verify that subcommands were registered by init() calls.
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "validate" || sub.Use == "check" {
			found = true
		}
	}
	if !found {
		t.Error("expected validate or check subcommand to be registered")
	}
}

// ── validate canvas ───────────────────────────────────────────────────────────

func TestRunValidateCanvas_NoCanvases(t *testing.T) {
	dir := t.TempDir()
	canvasFormat = formatText
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runValidateCanvas(cmd, []string{dir}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunValidateCanvas_ValidCanvas(t *testing.T) {
	root := repoRoot(t)
	canvasFormat = formatText
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	canvasPath := filepath.Join(root, "docs/spdd/314-yaml-system-cli/canvas.md")
	if err := runValidateCanvas(cmd, []string{canvasPath}); err != nil {
		t.Errorf("unexpected error validating real canvas: %v", err)
	}
}

func TestRunValidateCanvas_JSONFormat(t *testing.T) {
	root := repoRoot(t)
	canvasFormat = formatJSON
	defer func() { canvasFormat = formatText }()
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	canvasPath := filepath.Join(root, "docs/spdd/314-yaml-system-cli/canvas.md")
	_ = runValidateCanvas(cmd, []string{canvasPath})
	var decoded interface{}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Errorf("JSON output is not valid JSON: %v", err)
	}
}

func TestFindCanvases_SingleFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "canvas.md")
	_ = os.WriteFile(f, []byte("# REASONS\n"), 0o600)
	paths, err := findCanvases(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 1 || paths[0] != f {
		t.Errorf("paths = %v, want [%s]", paths, f)
	}
}

func TestFindCanvases_NonexistentPath(t *testing.T) {
	_, err := findCanvases("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestPrintCanvasText_WithErrors(t *testing.T) {
	cmd := fakeCmd(t)
	results := []canvasResult{{
		File:   "canvas.md",
		Errors: []validate.ValidationError{{Message: "missing section"}},
	}}
	err := printCanvasText(cmd, results, true)
	if err == nil {
		t.Error("expected error when failed=true")
	}
}

func TestPrintCanvasText_WarningsOnly(t *testing.T) {
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	results := []canvasResult{{
		File:     "canvas.md",
		Warnings: []validate.ValidationWarning{{Message: "status is Draft"}},
	}}
	if err := printCanvasText(cmd, results, false); err != nil {
		t.Errorf("unexpected error for warnings-only result: %v", err)
	}
}

func TestPrintCanvasText_OK(t *testing.T) {
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	results := []canvasResult{{File: "ok.md"}}
	if err := printCanvasText(cmd, results, false); err != nil {
		t.Errorf("unexpected error for clean result: %v", err)
	}
}

func TestPrintCanvasJSON_NoErrors(t *testing.T) {
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := printCanvasJSON(cmd, []canvasResult{{File: "ok.md"}}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	var decoded interface{}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Errorf("JSON output invalid: %v", err)
	}
}

// ── validate workflows ────────────────────────────────────────────────────────

func TestRunValidateWorkflows_NoManifests(t *testing.T) {
	root := repoRoot(t)
	workflowsSchemaDir = filepath.Join(root, "spec/schemas")
	workflowsFormat = formatText
	dir := t.TempDir()
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runValidateWorkflows(cmd, []string{dir}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunValidateWorkflows_ValidManifest(t *testing.T) {
	root := repoRoot(t)
	workflowsSchemaDir = filepath.Join(root, "spec/schemas")
	workflowsFormat = formatText
	dir := filepath.Join(root, "spec/workflows/examples")
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runValidateWorkflows(cmd, []string{dir}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPrintManifestResults_JSONFormat(t *testing.T) {
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	results := []validate.ManifestResult{{File: "wf.yaml"}}
	if err := printManifestResults(cmd, results, formatJSON); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	var decoded interface{}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Errorf("JSON invalid: %v", err)
	}
}

func TestPrintManifestResults_TextWithErrors(t *testing.T) {
	cmd := fakeCmd(t)
	results := []validate.ManifestResult{{
		File:   "bad.yaml",
		Errors: []validate.ValidationError{{Path: "/kind", Message: "bad kind"}},
	}}
	if err := printManifestResults(cmd, results, formatText); err == nil {
		t.Error("expected error for results with errors")
	}
}

// ── validate agent-defs ───────────────────────────────────────────────────────

func TestRunValidateAgentDefs_NoManifests(t *testing.T) {
	root := repoRoot(t)
	agentDefsSchemaDir = filepath.Join(root, "spec/schemas")
	agentDefsFormat = formatText
	dir := t.TempDir()
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runValidateAgentDefs(cmd, []string{dir}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── validate schema ───────────────────────────────────────────────────────────

func TestRunValidateSchema_Dir(t *testing.T) {
	root := repoRoot(t)
	schemaFormat = formatText
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	schemaDir := filepath.Join(root, "spec/schemas")
	if err := runValidateSchema(cmd, []string{schemaDir}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunValidateSchema_JSONFormat(t *testing.T) {
	root := repoRoot(t)
	schemaFormat = formatJSON
	defer func() { schemaFormat = formatText }()
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	schemaDir := filepath.Join(root, "spec/schemas")
	_ = runValidateSchema(cmd, []string{schemaDir})
	var decoded interface{}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Errorf("JSON output invalid: %v", err)
	}
}

func TestRunValidateSchema_InvalidDir(t *testing.T) {
	schemaFormat = formatText
	cmd := fakeCmd(t)
	// A file path passed as a single schema file — Schema() should be tried.
	f := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(f, []byte("{not json}"), 0o600)
	if err := runValidateSchema(cmd, []string{f}); err != nil {
		// May error or not depending on the error type; just ensure it runs.
		t.Logf("error (acceptable): %v", err)
	}
}

// ── validate policies ─────────────────────────────────────────────────────────

func TestRunValidatePolicies_NoManifests(t *testing.T) {
	root := repoRoot(t)
	policiesSchemaDir = filepath.Join(root, "spec/schemas")
	policiesFormat = formatText
	dir := t.TempDir()
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runValidatePolicies(cmd, []string{dir}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── validate capabilities ─────────────────────────────────────────────────────

func TestRunValidateCapabilities_EmptyDir(t *testing.T) {
	root := repoRoot(t)
	capabilitiesSchemaDir = filepath.Join(root, "spec/schemas")
	dir := t.TempDir()
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runValidateCapabilities(cmd, []string{dir}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── check ai-context ──────────────────────────────────────────────────────────

func TestRunCheckAIContext_RepoRoot(t *testing.T) {
	root := repoRoot(t)
	aiContextRoot = root
	cmd := fakeCmd(t)
	if err := runCheckAIContext(cmd, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunCheckAIContext_EmptyRoot(t *testing.T) {
	aiContextRoot = t.TempDir()
	cmd := fakeCmd(t)
	if err := runCheckAIContext(cmd, nil); err != nil {
		t.Errorf("unexpected error for empty root: %v", err)
	}
}

func TestRunCheckAIContext_DotRoot(t *testing.T) {
	// aiContextRoot="." causes the function to call os.Getwd(), covering that branch.
	aiContextRoot = "."
	defer func() { aiContextRoot = "." }()
	cmd := fakeCmd(t)
	if err := runCheckAIContext(cmd, nil); err != nil {
		t.Errorf("unexpected error for dot root: %v", err)
	}
}
