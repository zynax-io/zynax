// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/services/api-gateway/internal/api"
	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
)

// ── test doubles ─────────────────────────────────────────────────────────

type stubCompiler struct {
	result domain.CompileResult
	err    error
}

func (s *stubCompiler) CompileWorkflow(_ context.Context, _ []byte, _ string, _ bool) (domain.CompileResult, error) {
	return s.result, s.err
}

type stubEngine struct {
	submitID    string
	submitErr   error
	statusRun   domain.WorkflowRunSummary
	statusErr   error
	cancelErr   error
	watchEvents []domain.WatchEvent
	watchErr    error
}

func (s *stubEngine) SubmitWorkflow(_ context.Context, _ []byte, _ string) (string, error) {
	return s.submitID, s.submitErr
}

func (s *stubEngine) GetWorkflowStatus(_ context.Context, _ string) (domain.WorkflowRunSummary, error) {
	return s.statusRun, s.statusErr
}

func (s *stubEngine) CancelWorkflow(_ context.Context, _ string) error {
	return s.cancelErr
}

func (s *stubEngine) WatchWorkflow(_ context.Context, _ string, send func(domain.WatchEvent) error) error {
	if s.watchErr != nil {
		return s.watchErr
	}
	for _, ev := range s.watchEvents {
		if err := send(ev); err != nil {
			return err
		}
	}
	return nil
}

type stubRegistry struct {
	reg domain.AgentRegistration
	err error
}

func (s *stubRegistry) RegisterAgent(_ context.Context, _ []byte, _ string) (domain.AgentRegistration, error) {
	return s.reg, s.err
}

// ── helpers ───────────────────────────────────────────────────────────────

func newServer(c domain.CompilerPort, e domain.EnginePort) *httptest.Server {
	return newServerWithRegistry(c, e, &stubRegistry{})
}

func newServerWithRegistry(c domain.CompilerPort, e domain.EnginePort, r domain.RegistryPort) *httptest.Server {
	return newServerWithAuth(c, e, r, "")
}

func newServerWithAuth(c domain.CompilerPort, e domain.EnginePort, r domain.RegistryPort, apiKey string) *httptest.Server {
	svc := domain.NewApplyService(c, e, r)
	h := api.NewHandler(svc, apiKey)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return httptest.NewServer(mux)
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return m
}

const workflowYAML = "kind: Workflow\napiVersion: zynax.io/v1alpha1\n"

// ── POST /api/v1/apply ────────────────────────────────────────────────────

func TestHandler_Apply_ValidWorkflow_Returns202(t *testing.T) {
	srv := newServer(
		&stubCompiler{result: domain.CompileResult{IRBytes: []byte("ir")}},
		&stubEngine{submitID: "run-abc"},
	)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", bytes.NewBufferString(workflowYAML))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status: got %d, want 202", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["run_id"] != "run-abc" {
		t.Errorf("run_id: got %v, want run-abc", body["run_id"])
	}
}

func TestHandler_Apply_DryRun_Returns200_NoRunID(t *testing.T) {
	srv := newServer(
		&stubCompiler{result: domain.CompileResult{IRBytes: []byte("ir"), Warnings: []string{"deprecated field"}}},
		&stubEngine{submitID: "should-not-appear"},
	)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply?dry_run=true", "application/yaml", bytes.NewBufferString(workflowYAML))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["run_id"] != nil {
		t.Errorf("dry_run must not return run_id, got %v", body["run_id"])
	}
	if body["dry_run"] != true {
		t.Errorf("dry_run field: got %v, want true", body["dry_run"])
	}
}

func TestHandler_Apply_CompilationError_Returns422(t *testing.T) {
	srv := newServer(
		&stubCompiler{result: domain.CompileResult{
			Errors: []domain.CompileError{{Code: "YAML_PARSE_ERROR", Message: "unexpected token", Line: 5}},
		}},
		&stubEngine{},
	)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", bytes.NewBufferString(workflowYAML))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status: got %d, want 422", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	errs, ok := body["errors"].([]any)
	if !ok || len(errs) == 0 {
		t.Errorf("expected non-empty errors array, got %v", body["errors"])
	}
}

func TestHandler_Apply_UnknownKind_Returns400(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{})
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", bytes.NewBufferString("kind: SomethingElse\n"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["code"] != "UNSUPPORTED_KIND" {
		t.Errorf("code: got %v, want UNSUPPORTED_KIND", body["code"])
	}
}

func TestHandler_Apply_MissingKind_Returns400(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{})
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", bytes.NewBufferString("apiVersion: zynax.io/v1alpha1\n"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", resp.StatusCode)
	}
}

func TestHandler_Apply_EngineUnavailable_Returns503(t *testing.T) {
	srv := newServer(
		&stubCompiler{result: domain.CompileResult{IRBytes: []byte("ir")}},
		&stubEngine{submitErr: domain.ErrEngineUnavailable},
	)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", bytes.NewBufferString(workflowYAML))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["code"] != "ENGINE_UNAVAILABLE" {
		t.Errorf("code: got %v, want ENGINE_UNAVAILABLE", body["code"])
	}
}

func TestHandler_Apply_BodyTooLarge_Returns413(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{})
	defer srv.Close()

	large := strings.Repeat("a", 1<<20+1) // 1 MB + 1 byte
	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", strings.NewReader(large))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("status: got %d, want 413", resp.StatusCode)
	}
}

// ── GET /api/v1/workflows/{id} ────────────────────────────────────────────

func TestHandler_GetWorkflow_Returns200(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{
		statusRun: domain.WorkflowRunSummary{
			RunID: "r1", WorkflowID: "wf-1", Status: "RUNNING", CurrentState: "review",
		},
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/workflows/r1")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["status"] != "RUNNING" {
		t.Errorf("status field: got %v, want RUNNING", body["status"])
	}
	if body["current_state"] != "review" {
		t.Errorf("current_state: got %v, want review", body["current_state"])
	}
}

func TestHandler_GetWorkflow_NotFound_Returns404(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{statusErr: domain.ErrNotFound})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/workflows/ghost")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", resp.StatusCode)
	}
}

// ── POST /api/v1/apply (kind: AgentDef) ──────────────────────────────────

const agentDefYAML = "kind: AgentDef\napiVersion: zynax.io/v1alpha1\nmetadata:\n  name: test-agent\n"

func TestHandler_Apply_ValidAgentDef_Returns201(t *testing.T) {
	srv := newServerWithRegistry(
		&stubCompiler{},
		&stubEngine{},
		&stubRegistry{reg: domain.AgentRegistration{AgentID: "agent-001"}},
	)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", bytes.NewBufferString(agentDefYAML))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status: got %d, want 201", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["agent_id"] != "agent-001" {
		t.Errorf("agent_id: got %v, want agent-001", body["agent_id"])
	}
}

func TestHandler_Apply_DuplicateAgentDef_Returns409(t *testing.T) {
	srv := newServerWithRegistry(
		&stubCompiler{},
		&stubEngine{},
		&stubRegistry{err: domain.ErrAgentAlreadyExists},
	)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", bytes.NewBufferString(agentDefYAML))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status: got %d, want 409", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["code"] != "ALREADY_EXISTS" {
		t.Errorf("code: got %v, want ALREADY_EXISTS", body["code"])
	}
}

func TestHandler_DeleteWorkflow_Returns204(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/workflows/run-abc", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status: got %d, want 204", resp.StatusCode)
	}
}

func TestHandler_DeleteWorkflow_NotFound_Returns404(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{cancelErr: domain.ErrNotFound})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/workflows/run-missing", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", resp.StatusCode)
	}
}

// ── GET /api/v1/workflows/{id}/logs ──────────────────────────────────────

func TestHandler_WorkflowLogs_StreamsSSEEvents(t *testing.T) {
	events := []domain.WatchEvent{
		{RunID: "r1", EventType: "state.entered", ToState: "review", Status: "WORKFLOW_STATUS_RUNNING"},
		{RunID: "r1", EventType: "workflow.completed", Status: "WORKFLOW_STATUS_COMPLETED"},
	}
	srv := newServer(&stubCompiler{}, &stubEngine{watchEvents: events})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/workflows/r1/logs")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type: got %q, want text/event-stream", ct)
	}

	var dataLines []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
		}
	}
	if len(dataLines) != 2 {
		t.Fatalf("got %d data lines, want 2", len(dataLines))
	}
	var ev map[string]any
	if err := json.Unmarshal([]byte(dataLines[1]), &ev); err != nil {
		t.Fatalf("unmarshal last event: %v", err)
	}
	if ev["event_type"] != "workflow.completed" {
		t.Errorf("last event_type: got %v, want workflow.completed", ev["event_type"])
	}
}

func TestHandler_WorkflowLogs_NotFound_Returns404(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{watchErr: domain.ErrNotFound})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/workflows/ghost/logs")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", resp.StatusCode)
	}
}

// ── Bearer-token auth middleware ──────────────────────────────────────────

func TestHandler_Auth_CorrectKey_Passes(t *testing.T) {
	srv := newServerWithAuth(
		&stubCompiler{result: domain.CompileResult{IRBytes: []byte("ir")}},
		&stubEngine{submitID: "run-auth"},
		&stubRegistry{},
		"secret-key",
	)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/apply", bytes.NewBufferString(workflowYAML))
	req.Header.Set("Content-Type", "application/yaml")
	req.Header.Set("Authorization", "Bearer secret-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status: got %d, want 202", resp.StatusCode)
	}
}

func TestHandler_Auth_MissingKey_Returns401(t *testing.T) {
	srv := newServerWithAuth(
		&stubCompiler{},
		&stubEngine{},
		&stubRegistry{},
		"secret-key",
	)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", bytes.NewBufferString(workflowYAML))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["code"] != "UNAUTHORIZED" {
		t.Errorf("code: got %v, want UNAUTHORIZED", body["code"])
	}
}

func TestHandler_Auth_WrongKey_Returns401(t *testing.T) {
	srv := newServerWithAuth(
		&stubCompiler{},
		&stubEngine{},
		&stubRegistry{},
		"secret-key",
	)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/v1/apply", bytes.NewBufferString(workflowYAML))
	req.Header.Set("Content-Type", "application/yaml")
	req.Header.Set("Authorization", "Bearer wrong-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", resp.StatusCode)
	}
}

func TestHandler_Auth_EmptyAPIKey_DisablesGate(t *testing.T) {
	// When ZYNAX_API_KEY is empty, auth is disabled — unauthenticated requests must pass.
	srv := newServerWithAuth(
		&stubCompiler{result: domain.CompileResult{IRBytes: []byte("ir")}},
		&stubEngine{submitID: "run-noauth"},
		&stubRegistry{},
		"",
	)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", bytes.NewBufferString(workflowYAML))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status: got %d, want 202 (auth disabled)", resp.StatusCode)
	}
}

func TestHandler_Auth_DeleteWithKey_Returns401(t *testing.T) {
	srv := newServerWithAuth(
		&stubCompiler{},
		&stubEngine{},
		&stubRegistry{},
		"secret-key",
	)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/v1/workflows/run-abc", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", resp.StatusCode)
	}
}

func TestHandler_Auth_GetNotProtected(t *testing.T) {
	// GET endpoints must remain open even when ZYNAX_API_KEY is set.
	srv := newServerWithAuth(
		&stubCompiler{},
		&stubEngine{statusRun: domain.WorkflowRunSummary{RunID: "r1", Status: "RUNNING"}},
		&stubRegistry{},
		"secret-key",
	)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/workflows/r1")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200 (GET must not require auth)", resp.StatusCode)
	}
}
