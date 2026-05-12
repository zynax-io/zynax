// SPDX-License-Identifier: Apache-2.0

// Internal tests for the cmd package.  Using package cmd (not cmd_test) lets
// us call unexported RunE functions directly and observe package-level vars,
// while the mere import causes every init() in this package to run, counting
// those statements as covered.
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
)

// repoRoot resolves the repository root from the location of this test file.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	// file = cmd/zynax/cmd/cmd_test.go (absolute)
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
// All init() functions in apply.go, delete.go, get.go, gitops.go, logs.go,
// root.go, status.go, and validate.go run when the test binary starts.
// The test below validates one observable side-effect (flag registration).

func TestInit_FlagsRegistered(t *testing.T) {
	if rootCmd.Flag("api-url") == nil {
		t.Error("--api-url flag not registered (root init() not called)")
	}
	if rootCmd.Flag("insecure") == nil {
		t.Error("--insecure flag not registered")
	}
}

func TestNewGateway_ReturnsNonNil(t *testing.T) {
	if newGateway() == nil {
		t.Fatal("newGateway returned nil")
	}
}

// ── validate subcommand ───────────────────────────────────────────────────────

func TestRunValidateManifest_ValidFile(t *testing.T) {
	root := repoRoot(t)
	validateSchemaDir = filepath.Join(root, "spec/schemas")
	validateFormat = formatText

	cmd := fakeCmd(t)
	fixture := filepath.Join(root, "spec/workflows/examples/code-review.yaml")
	if err := runValidateManifest(cmd, []string{fixture}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunValidateManifest_UnknownKind_TextFormat(t *testing.T) {
	root := repoRoot(t)
	validateSchemaDir = filepath.Join(root, "spec/schemas")
	validateFormat = formatText

	dir := t.TempDir()
	f := filepath.Join(dir, "bad.yaml")
	_ = os.WriteFile(f, []byte("kind: Unknown\n"), 0o600)

	cmd := fakeCmd(t)
	if err := runValidateManifest(cmd, []string{f}); err == nil {
		t.Error("expected error for unknown kind")
	}
}

func TestRunValidateManifest_JSONFormat_Valid(t *testing.T) {
	root := repoRoot(t)
	validateSchemaDir = filepath.Join(root, "spec/schemas")
	validateFormat = formatJSON
	defer func() { validateFormat = formatText }()

	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	fixture := filepath.Join(root, "spec/workflows/examples/code-review.yaml")
	_ = runValidateManifest(cmd, []string{fixture})

	var decoded interface{}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Errorf("JSON output is not valid JSON: %v\noutput: %s", err, out.String())
	}
}

func TestRunValidateManifest_JSONFormat_Errors(t *testing.T) {
	root := repoRoot(t)
	validateSchemaDir = filepath.Join(root, "spec/schemas")
	validateFormat = formatJSON
	defer func() { validateFormat = formatText }()

	dir := t.TempDir()
	f := filepath.Join(dir, "bad.yaml")
	_ = os.WriteFile(f, []byte("kind: Unknown\n"), 0o600)

	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	_ = runValidateManifest(cmd, []string{f})

	var decoded interface{}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Errorf("JSON error output is not valid JSON: %v", err)
	}
}

func TestRunValidateManifest_FileNotFound(t *testing.T) {
	root := repoRoot(t)
	validateSchemaDir = filepath.Join(root, "spec/schemas")
	validateFormat = formatText

	cmd := fakeCmd(t)
	if err := runValidateManifest(cmd, []string{"/nonexistent/manifest.yaml"}); err == nil {
		t.Error("expected error for missing file")
	}
}

// ── apply subcommand ──────────────────────────────────────────────────────────

func TestRunApply_Workflow_202(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"run_id": "wf-001", "warnings": []string{"w1"}})
	}))
	defer srv.Close()
	apiURL = srv.URL
	applyEngine = ""

	cmd := fakeCmd(t)
	if err := runApply(cmd, newGateway(), []byte("kind: Workflow")); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunApply_AgentDef_201(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"agent_id": "agent-xyz"})
	}))
	defer srv.Close()
	apiURL = srv.URL

	cmd := fakeCmd(t)
	if err := runApply(cmd, newGateway(), []byte("kind: AgentDef")); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunApply_GatewayError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	apiURL = srv.URL

	cmd := fakeCmd(t)
	if err := runApply(cmd, newGateway(), []byte("kind: Workflow")); err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestRunDryRun_Valid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"warnings": []string{}})
	}))
	defer srv.Close()
	apiURL = srv.URL

	cmd := fakeCmd(t)
	if err := runDryRun(cmd, newGateway(), []byte("kind: Workflow")); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunDryRun_CompileErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{{"message": "bad field", "line": 5}},
		})
	}))
	defer srv.Close()
	apiURL = srv.URL

	cmd := fakeCmd(t)
	if err := runDryRun(cmd, newGateway(), []byte("bad")); err == nil {
		t.Error("expected error for compile errors")
	}
}

// ── get + delete subcommands ──────────────────────────────────────────────────

func TestGetWorkflow_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_id": "run-1", "status": "WORKFLOW_STATUS_RUNNING",
		})
	}))
	defer srv.Close()
	apiURL = srv.URL

	cmd := fakeCmd(t)
	gw := newGateway()
	run, err := gw.GetWorkflow(cmd.Context(), "run-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.Status != "WORKFLOW_STATUS_RUNNING" {
		t.Errorf("status = %q", run.Status)
	}
}

func TestDeleteWorkflow_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	apiURL = srv.URL

	if err := newGateway().DeleteWorkflow(context.Background(), "run-x"); err == nil {
		t.Error("expected error for 500 response")
	}
}

// ── root Execute() ────────────────────────────────────────────────────────────

func TestRootCmd_Version(t *testing.T) {
	// Call cobra's Execute with --version; cobra handles it without calling os.Exit.
	rootCmd.SetArgs([]string{"--version"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	_ = rootCmd.Execute()
	rootCmd.SetArgs(nil)
	if out.Len() == 0 {
		t.Error("expected version output, got empty")
	}
}
