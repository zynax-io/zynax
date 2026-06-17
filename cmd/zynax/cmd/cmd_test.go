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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	if err := runValidate(cmd, []string{fixture}); err != nil {
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
	if err := runValidate(cmd, []string{f}); err == nil {
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
	_ = runValidate(cmd, []string{fixture})

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
	_ = runValidate(cmd, []string{f})

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
	if err := runValidate(cmd, []string{"/nonexistent/manifest.yaml"}); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestRunValidate_DataFlowError(t *testing.T) {
	root := repoRoot(t)
	validateSchemaDir = filepath.Join(root, "spec/schemas")
	validateFormat = formatText

	dir := t.TempDir()
	f := filepath.Join(dir, "wf.yaml")
	// Schema-valid Workflow whose initial_state references an undefined state.
	body := "kind: Workflow\napiVersion: zynax.io/v1\nmetadata:\n  name: wf\n" +
		"spec:\n  initial_state: ghost\n  states:\n    start:\n      type: terminal\n"
	_ = os.WriteFile(f, []byte(body), 0o600)

	cmd := fakeCmd(t)
	if err := runValidate(cmd, []string{f}); err == nil {
		t.Error("expected data-flow error for undefined initial_state")
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

func TestGetWorkflowCmd_SurfacesVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_id": "run-1", "status": "WORKFLOW_STATUS_COMPLETED", "version": "1.2.3",
		})
	}))
	defer srv.Close()
	apiURL = srv.URL

	var out bytes.Buffer
	getWorkflowCmd.SetOut(&out)
	getWorkflowCmd.SetContext(context.Background())
	if err := getWorkflowCmd.RunE(getWorkflowCmd, []string{"run-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("version:       1.2.3")) {
		t.Errorf("expected version in output, got:\n%s", out.String())
	}
}

func TestStatusCmd_TerminalSurfacesVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_id": "run-1", "status": "WORKFLOW_STATUS_COMPLETED", "version": "2.0.0",
		})
	}))
	defer srv.Close()
	apiURL = srv.URL

	var out bytes.Buffer
	statusWorkflowCmd.SetOut(&out)
	statusWorkflowCmd.SetContext(context.Background())
	// Terminal status returns without os.Exit(2).
	if err := statusWorkflowCmd.RunE(statusWorkflowCmd, []string{"run-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("version: 2.0.0")) {
		t.Errorf("expected version in status output, got:\n%s", out.String())
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

// ── logs subcommand ───────────────────────────────────────────────────────────

// sseServer returns a test server that emits the given JSON events as SSE
// frames on GET /api/v1/workflows/{id}/logs.
func sseServer(t *testing.T, events []map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/logs") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, ev := range events {
			b, _ := json.Marshal(ev)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func runLogs(t *testing.T, args []string) (string, error) {
	t.Helper()
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := logsCmd.RunE(cmd, args)
	return out.String(), err
}

func TestLogsCmd_Follow_StopsAtTerminal(t *testing.T) {
	srv := sseServer(t, []map[string]any{
		{"run_id": "r1", "event_type": "state.entered", "to_state": "review", "status": "WORKFLOW_STATUS_RUNNING"},
		{"run_id": "r1", "event_type": "capability.invoked", "status": "WORKFLOW_STATUS_RUNNING"},
		{"run_id": "r1", "event_type": "workflow.completed", "from_state": "review", "to_state": "done", "status": "WORKFLOW_STATUS_COMPLETED"},
	})
	apiURL = srv.URL
	logsFormat = formatText
	logsFollow = true
	defer func() { logsFollow = false }()

	out, err := runLogs(t, []string{"r1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Terminal event is the last and exits cleanly (sentinel swallowed).
	if !strings.Contains(out, "workflow.completed") {
		t.Errorf("expected terminal event in output, got:\n%s", out)
	}
	// State transition renders the arrow.
	if !strings.Contains(out, "review → done") {
		t.Errorf("expected state arrow for transition, got:\n%s", out)
	}
	// Capability event without a transition omits the arrow.
	if strings.Contains(out, "capability.invoked  ") && strings.Contains(out, "capability.invoked   →") {
		t.Errorf("capability event should not render a state arrow, got:\n%s", out)
	}
}

func TestLogsCmd_NoFollow_StreamsAll(t *testing.T) {
	srv := sseServer(t, []map[string]any{
		{"run_id": "r2", "event_type": "state.entered", "to_state": "start", "status": "WORKFLOW_STATUS_RUNNING"},
		{"run_id": "r2", "event_type": "state.entered", "to_state": "review", "status": "WORKFLOW_STATUS_RUNNING"},
	})
	apiURL = srv.URL
	logsFormat = formatText
	logsFollow = false

	out, err := runLogs(t, []string{"r2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Count(out, "state.entered") != 2 {
		t.Errorf("expected 2 events in non-follow mode, got:\n%s", out)
	}
}

func TestLogsCmd_JSONFormat(t *testing.T) {
	srv := sseServer(t, []map[string]any{
		{"run_id": "r3", "event_type": "workflow.completed", "status": "WORKFLOW_STATUS_COMPLETED"},
	})
	apiURL = srv.URL
	logsFormat = formatJSON
	logsFollow = true
	defer func() {
		logsFormat = formatText
		logsFollow = false
	}()

	out, err := runLogs(t, []string{"r3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &decoded); err != nil {
		t.Errorf("logs JSON output is not valid JSON: %v\noutput: %s", err, out)
	}
}

func TestLogsCmd_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	apiURL = srv.URL
	logsFollow = false

	if _, err := runLogs(t, []string{"ghost"}); err == nil {
		t.Error("expected error for 404 response")
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
