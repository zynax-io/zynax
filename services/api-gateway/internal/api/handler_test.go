// SPDX-License-Identifier: Apache-2.0

package api_test

import (
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
	submitID  string
	submitErr error
	statusRun domain.WorkflowRunSummary
	statusErr error
}

func (s *stubEngine) SubmitWorkflow(_ context.Context, _ []byte, _ string) (string, error) {
	return s.submitID, s.submitErr
}

func (s *stubEngine) GetWorkflowStatus(_ context.Context, _ string) (domain.WorkflowRunSummary, error) {
	return s.statusRun, s.statusErr
}

// ── helpers ───────────────────────────────────────────────────────────────

func newServer(c domain.CompilerPort, e domain.EnginePort) *httptest.Server {
	svc := domain.NewApplyService(c, e)
	h := api.NewHandler(svc)
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
