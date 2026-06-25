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
	"io"
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

// ── init subcommand ───────────────────────────────────────────────────────────

func TestRunInit_Workflow_EmitsValidManifest(t *testing.T) {
	root := repoRoot(t)
	initTemplateDir = filepath.Join(root, "spec/templates")
	initOutput = ""

	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runInit(cmd, "workflow", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "kind: Workflow") {
		t.Errorf("expected Workflow manifest, got:\n%s", out.String())
	}
	// AC: emits a *versioned* manifest.
	if !strings.Contains(out.String(), "version:") {
		t.Errorf("expected version field in scaffolded manifest, got:\n%s", out.String())
	}

	// AC: the scaffolded manifest must be *valid* — round-trip through validate.
	dir := t.TempDir()
	f := filepath.Join(dir, "wf.yaml")
	if err := os.WriteFile(f, out.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	validateSchemaDir = filepath.Join(root, "spec/schemas")
	validateFormat = formatText
	if err := runValidate(fakeCmd(t), []string{f}); err != nil {
		t.Errorf("scaffolded workflow failed validation: %v", err)
	}
}

func TestRunInit_Expert_EmitsValidManifest(t *testing.T) {
	root := repoRoot(t)
	initTemplateDir = filepath.Join(root, "spec/templates")
	initOutput = ""

	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runInit(cmd, "expert", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "kind: AgentDef") {
		t.Errorf("expected AgentDef manifest, got:\n%s", out.String())
	}

	dir := t.TempDir()
	f := filepath.Join(dir, "expert.yaml")
	if err := os.WriteFile(f, out.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	validateSchemaDir = filepath.Join(root, "spec/schemas")
	validateFormat = formatText
	if err := runValidate(fakeCmd(t), []string{f}); err != nil {
		t.Errorf("scaffolded expert failed validation: %v", err)
	}
}

func TestRunInit_NameOverride(t *testing.T) {
	root := repoRoot(t)
	initTemplateDir = filepath.Join(root, "spec/templates")
	initOutput = ""

	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runInit(cmd, "workflow", []string{"my-pipeline"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "name: my-pipeline") {
		t.Errorf("expected overridden name, got:\n%s", out.String())
	}
}

func TestRunInit_OutputFile(t *testing.T) {
	root := repoRoot(t)
	initTemplateDir = filepath.Join(root, "spec/templates")
	dir := t.TempDir()
	initOutput = filepath.Join(dir, "wf.yaml")
	defer func() { initOutput = "" }()

	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runInit(cmd, "workflow", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), initOutput) {
		t.Errorf("expected confirmation message, got:\n%s", out.String())
	}
	b, err := os.ReadFile(initOutput) //nolint:gosec // test reads the file it just scaffolded into a temp dir
	if err != nil {
		t.Fatalf("output file not written: %v", err)
	}
	if !strings.Contains(string(b), "kind: Workflow") {
		t.Errorf("output file missing manifest, got:\n%s", b)
	}
}

func TestRunInit_TemplateNotFound(t *testing.T) {
	initTemplateDir = t.TempDir()
	initOutput = ""

	cmd := fakeCmd(t)
	if err := runInit(cmd, "workflow", nil); err == nil {
		t.Error("expected error for missing template")
	}
}

func TestScaffold_NoMetadataName(t *testing.T) {
	dir := t.TempDir()
	kindDir := filepath.Join(dir, "workflow")
	if err := os.Mkdir(kindDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kindDir, "workflow.template.yaml"),
		[]byte("kind: Workflow\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := scaffold(dir, "workflow", "x"); err == nil {
		t.Error("expected error when template has no metadata.name")
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

// TestLogsCmd_RendersCompletionOutput verifies that a capability completion
// event's nested result payload is surfaced as an indented output line (#1378).
func TestLogsCmd_RendersCompletionOutput(t *testing.T) {
	srv := sseServer(t, []map[string]any{
		{"run_id": "r4", "event_type": "zynax.v1.task-broker.task.completed", "status": "capability_event",
			"payload": `{"workflow_id":"r4","result_payload":"{\"completion\":\"LGTM, ship it\"}"}`},
		{"run_id": "r4", "event_type": "workflow.completed", "status": "WORKFLOW_STATUS_COMPLETED"},
	})
	apiURL = srv.URL
	logsFormat = formatText
	logsFollow = false

	out, err := runLogs(t, []string{"r4"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "output: LGTM, ship it") {
		t.Errorf("expected completion text in logs output, got:\n%s", out)
	}
}

func runResult(t *testing.T, args []string) (string, error) {
	t.Helper()
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := resultCmd.RunE(cmd, args)
	return out.String(), err
}

func TestResultCmd_PrintsCompletion(t *testing.T) {
	srv := sseServer(t, []map[string]any{
		{"run_id": "r5", "event_type": "task.completed", "status": "capability_event",
			"payload": `{"result_payload":"{\"completion\":\"the model review text\"}"}`},
		{"run_id": "r5", "event_type": "workflow.completed", "status": "WORKFLOW_STATUS_COMPLETED"},
	})
	apiURL = srv.URL

	out, err := runResult(t, []string{"r5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "the model review text" {
		t.Errorf("result output = %q, want %q", strings.TrimSpace(out), "the model review text")
	}
}

func TestResultCmd_NoResult_Errors(t *testing.T) {
	srv := sseServer(t, []map[string]any{
		{"run_id": "r6", "event_type": "workflow.failed", "status": "WORKFLOW_STATUS_FAILED"},
	})
	apiURL = srv.URL

	if _, err := runResult(t, []string{"r6"}); err == nil {
		t.Error("expected error when run finishes with no result payload")
	}
}

// ── last-run state (#1491, canvas O21) ────────────────────────────────────────

// TestRunApply_RecordsLastRunID asserts AC1: a successful apply that returns a
// run id persists it to <config-dir>/last-run so bare logs/result can default
// to it. The config dir is an isolated t.TempDir() via ZYNAX_CONFIG_DIR.
func TestRunApply_RecordsLastRunID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ZYNAX_CONFIG_DIR", dir)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"run_id": "wf-record"})
	}))
	defer srv.Close()
	apiURL = srv.URL
	applyEngine = ""

	cmd := fakeCmd(t)
	if err := runApply(cmd, newGateway(), []byte("kind: Workflow")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, lastRunFile)) //nolint:gosec // test reads the file it just wrote into a temp dir
	if err != nil {
		t.Fatalf("last-run file not written: %v", err)
	}
	if got := strings.TrimSpace(string(b)); got != "wf-record" {
		t.Errorf("last-run file = %q, want %q", got, "wf-record")
	}
}

// TestRunApply_AgentDef_DoesNotRecord asserts an apply that yields only an
// agent id (no run id) records nothing — last-run is for Workflow runs only.
func TestRunApply_AgentDef_DoesNotRecord(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ZYNAX_CONFIG_DIR", dir)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"agent_id": "agent-xyz"})
	}))
	defer srv.Close()
	apiURL = srv.URL

	if err := runApply(fakeCmd(t), newGateway(), []byte("kind: AgentDef")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, lastRunFile)); !os.IsNotExist(err) {
		t.Errorf("expected no last-run file for AgentDef apply, stat err = %v", err)
	}
}

// TestResolveRunID covers AC2/AC3: explicit id overrides the stored one, a bare
// invocation falls back to the stored id, and no prior run yields a clear error.
func TestResolveRunID(t *testing.T) {
	tests := []struct {
		name    string
		stored  string // "" means do not write a state file
		args    []string
		want    string
		wantErr bool
	}{
		{name: "explicit id with no stored", args: []string{"explicit-1"}, want: "explicit-1"},
		{name: "explicit id overrides stored", stored: "stored-1", args: []string{"explicit-2"}, want: "explicit-2"},
		{name: "bare falls back to stored", stored: "stored-2", args: nil, want: "stored-2"},
		{name: "bare with no prior run errors", args: nil, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("ZYNAX_CONFIG_DIR", dir)
			if tc.stored != "" {
				if err := saveLastRun(tc.stored); err != nil {
					t.Fatalf("seed saveLastRun: %v", err)
				}
			}
			got, err := resolveRunID(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %s, got id %q", tc.name, got)
				}
				if !strings.Contains(err.Error(), "no prior run") {
					t.Errorf("error not actionable: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("resolveRunID = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestLogsCmd_BareUsesStoredRun asserts AC2: bare `zynax logs` (no positional
// id) streams the most recently recorded run.
func TestLogsCmd_BareUsesStoredRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ZYNAX_CONFIG_DIR", dir)
	if err := saveLastRun("stored-run"); err != nil {
		t.Fatal(err)
	}
	var seenPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "data: %s\n\n",
			`{"run_id":"stored-run","event_type":"workflow.completed","status":"WORKFLOW_STATUS_COMPLETED"}`)
	}))
	defer srv.Close()
	apiURL = srv.URL
	logsFormat = formatText
	logsFollow = false

	if _, err := runLogs(t, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(seenPath, "stored-run") {
		t.Errorf("bare logs targeted %q, want path containing stored-run", seenPath)
	}
}

// TestLogsCmd_BareNoPriorRun asserts AC3: a bare logs with no recorded run
// returns a clear, actionable error rather than calling the gateway.
func TestLogsCmd_BareNoPriorRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ZYNAX_CONFIG_DIR", dir)
	logsFollow = false

	if _, err := runLogs(t, nil); err == nil {
		t.Error("expected error for bare logs with no prior run")
	} else if !strings.Contains(err.Error(), "no prior run") {
		t.Errorf("error not actionable: %v", err)
	}
}

// TestResultCmd_BareUsesStoredRun asserts AC2 for `result`: a bare invocation
// targets the stored run and prints its completion text.
func TestResultCmd_BareUsesStoredRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ZYNAX_CONFIG_DIR", dir)
	if err := saveLastRun("stored-r"); err != nil {
		t.Fatal(err)
	}
	srv := sseServer(t, []map[string]any{
		{"run_id": "stored-r", "event_type": "task.completed", "status": "capability_event",
			"payload": `{"result_payload":"{\"completion\":\"bare result text\"}"}`},
		{"run_id": "stored-r", "event_type": "workflow.completed", "status": "WORKFLOW_STATUS_COMPLETED"},
	})
	apiURL = srv.URL

	out, err := runResult(t, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "bare result text" {
		t.Errorf("result output = %q, want %q", strings.TrimSpace(out), "bare result text")
	}
}

// ── scenario apply + validate ─────────────────────────────────────────────────

// TestRunApplyScenario_AppliesMembersInOrder verifies that applying the shipped
// reference scenario submits each member over the existing /api/v1/apply path in
// apply_order (AgentDef first), recording the order the server observed.
func TestRunApplyScenario_AppliesMembersInOrder(t *testing.T) {
	root := repoRoot(t)
	var kinds []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body := string(b)
		switch {
		case strings.Contains(body, "kind: AgentDef"):
			kinds = append(kinds, "AgentDef")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"agent_id": "agent-xyz"})
		default:
			kinds = append(kinds, "Workflow")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"run_id": "wf-001"})
		}
	}))
	defer srv.Close()
	apiURL = srv.URL
	applyDryRun = false
	applyEngine = ""

	idx := filepath.Join(root, "spec/scenarios/code-review", "scenario.yaml")
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runApplyScenario(cmd, newGateway(), idx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(kinds) != 2 || kinds[0] != "AgentDef" || kinds[1] != "Workflow" {
		t.Errorf("server observed order %v, want [AgentDef Workflow]", kinds)
	}
	if !strings.Contains(out.String(), "run_id: wf-001") {
		t.Errorf("expected workflow run_id in output, got:\n%s", out.String())
	}
}

func TestRunApplyScenario_DryRun(t *testing.T) {
	root := repoRoot(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"warnings": []string{}})
	}))
	defer srv.Close()
	apiURL = srv.URL
	applyDryRun = true
	defer func() { applyDryRun = false }()

	idx := filepath.Join(root, "spec/scenarios/code-review", "scenario.yaml")
	cmd := fakeCmd(t)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runApplyScenario(cmd, newGateway(), idx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunApplyScenario_MemberError(t *testing.T) {
	root := repoRoot(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	apiURL = srv.URL
	applyDryRun = false

	idx := filepath.Join(root, "spec/scenarios/code-review", "scenario.yaml")
	cmd := fakeCmd(t)
	if err := runApplyScenario(cmd, newGateway(), idx); err == nil {
		t.Error("expected error when a scenario member fails to apply")
	}
}

func TestRunValidate_Scenario(t *testing.T) {
	root := repoRoot(t)
	validateSchemaDir = filepath.Join(root, "spec/schemas")
	validateFormat = formatText

	cmd := fakeCmd(t)
	dir := filepath.Join(root, "spec/scenarios/code-review")
	if err := runValidate(cmd, []string{dir}); err != nil {
		t.Errorf("reference scenario should validate, got: %v", err)
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
