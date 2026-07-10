// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
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

func (s *stubEngine) SubmitWorkflow(_ context.Context, _ []byte, _, _, _ string) (string, error) {
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

// ── helpers ───────────────────────────────────────────────────────────────

func newServer(c domain.CompilerPort, e domain.EnginePort) *httptest.Server {
	svc := domain.NewApplyService(c, e, nil)
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

// ── GET /api/v1/workflows/{id}/outputs (M7.U O.8) ────────────────────────

func TestHandler_WorkflowOutputs_Returns200WithOutputs(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{
		statusRun: domain.WorkflowRunSummary{
			RunID: "r1", Status: statusCompletedTest,
			Outputs: map[string]string{"review": "LGTM", "score": "9"},
		},
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/workflows/r1/outputs")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["review"] != "LGTM" || body["score"] != "9" {
		t.Errorf("outputs body = %v, want review=LGTM score=9", body)
	}
}

func TestHandler_WorkflowOutputs_Empty_ReturnsEmptyObject(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{
		statusRun: domain.WorkflowRunSummary{RunID: "r1", Status: statusCompletedTest},
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/workflows/r1/outputs")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if len(body) != 0 {
		t.Errorf("expected empty object {}, got %v", body)
	}
}

func TestHandler_WorkflowOutputs_NotFound_Returns404(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{statusErr: domain.ErrNotFound})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/workflows/ghost/outputs")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", resp.StatusCode)
	}
}

func TestHandler_WorkflowOutputs_SafeJSONEncoding(t *testing.T) {
	// An attacker-influenced value with control/ANSI bytes must be JSON-escaped,
	// never emitted raw, yet round-trip back to the original string.
	val := "a\x1b[31mred\x00b"
	srv := newServer(&stubCompiler{}, &stubEngine{
		statusRun: domain.WorkflowRunSummary{
			RunID: "r1", Status: statusCompletedTest,
			Outputs: map[string]string{"x": val},
		},
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/workflows/r1/outputs")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if bytes.ContainsAny(raw, "\x00\x1b") {
		t.Errorf("raw control bytes leaked into JSON output: %q", raw)
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("body is not valid JSON: %v (%q)", err, raw)
	}
	if m["x"] != val {
		t.Errorf("round-trip mismatch: got %q, want %q", m["x"], val)
	}
}

func TestHandler_WorkflowLogs_TerminalEventCarriesOutputs(t *testing.T) {
	events := []domain.WatchEvent{
		{RunID: "r1", EventType: "state.entered", ToState: "done", Status: "WORKFLOW_STATUS_RUNNING"},
		{RunID: "r1", EventType: "workflow.completed", Status: statusCompletedTest},
	}
	srv := newServer(&stubCompiler{}, &stubEngine{
		watchEvents: events,
		statusRun:   domain.WorkflowRunSummary{RunID: "r1", Status: statusCompletedTest, Outputs: map[string]string{"review": "LGTM"}},
	})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/workflows/r1/logs")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	var dataLines []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if line := scanner.Text(); strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
		}
	}
	if len(dataLines) == 0 {
		t.Fatal("no SSE data lines")
	}
	var ev struct {
		Status  string `json:"status"`
		Payload string `json:"payload"`
	}
	if err := json.Unmarshal([]byte(dataLines[len(dataLines)-1]), &ev); err != nil {
		t.Fatalf("terminal event not JSON: %v", err)
	}
	if ev.Status != statusCompletedTest {
		t.Errorf("last event status = %q, want COMPLETED", ev.Status)
	}
	var pl struct {
		Outputs map[string]string `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(ev.Payload), &pl); err != nil {
		t.Fatalf("terminal payload not JSON: %v (%q)", err, ev.Payload)
	}
	if pl.Outputs["review"] != "LGTM" {
		t.Errorf("terminal payload outputs = %v, want review=LGTM", pl.Outputs)
	}
}

// ── POST /api/v1/apply (kind: AgentDef) ──────────────────────────────────

const (
	agentDefYAML        = "kind: AgentDef\napiVersion: zynax.io/v1alpha1\nmetadata:\n  name: test-agent\n"
	statusCompletedTest = "WORKFLOW_STATUS_COMPLETED"
)

// CRD era (ADR-039): applying kind: AgentDef answers 410 Gone with the
// migration pointer — the push forward is retired.
func TestHandler_Apply_AgentDef_Returns410Retired(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{})
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", bytes.NewBufferString(agentDefYAML))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusGone {
		t.Errorf("status: got %d, want 410", resp.StatusCode)
	}
	body := decodeBody(t, resp)
	if body["code"] != "AGENTDEF_RETIRED" {
		t.Errorf("code: got %v, want AGENTDEF_RETIRED", body["code"])
	}
	if msg, _ := body["error"].(string); !strings.Contains(msg, "agent-crd-migration") {
		t.Errorf("error message must point at the migration guide, got %q", msg)
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

// ── X-Request-ID middleware ───────────────────────────────────────────────

func newServerWithRequestID(c domain.CompilerPort, e domain.EnginePort) *httptest.Server {
	svc := domain.NewApplyService(c, e, nil)
	h := api.NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return httptest.NewServer(api.RequestIDMiddleware(mux))
}

func TestRequestIDMiddleware_EchoesExistingID(t *testing.T) {
	srv := newServerWithRequestID(&stubCompiler{}, &stubEngine{statusRun: domain.WorkflowRunSummary{RunID: "r1"}})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/workflows/r1", nil)
	req.Header.Set("X-Request-ID", "trace-abc")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if got := resp.Header.Get("X-Request-ID"); got != "trace-abc" {
		t.Errorf("X-Request-ID: got %q, want %q", got, "trace-abc")
	}
}

func TestRequestIDMiddleware_GeneratesID_WhenAbsent(t *testing.T) {
	srv := newServerWithRequestID(&stubCompiler{}, &stubEngine{statusRun: domain.WorkflowRunSummary{RunID: "r1"}})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/workflows/r1")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if got := resp.Header.Get("X-Request-ID"); got == "" {
		t.Error("X-Request-ID header must be set even when not provided by client")
	}
}

// ── Body size enforcement ─────────────────────────────────────────────────

func TestHandler_Apply_OversizedBody_Returns413(t *testing.T) {
	srv := newServer(&stubCompiler{}, &stubEngine{})
	defer srv.Close()

	body := make([]byte, 2<<20) // 2 MB — exceeds the 1 MB limit in readBody
	resp, err := http.Post(srv.URL+"/api/v1/apply", "application/yaml", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("got %d, want 413", resp.StatusCode)
	}
}

// ── POST /api/v1/workflows/{id}/events ────────────────────────────────────

type stubEventBus struct {
	publishID     string
	publishErr    error
	capturedEvent domain.EventPublish
}

func (s *stubEventBus) SubscribeWorkflowEvents(_ context.Context, _ string, _ func(domain.WatchEvent) error) error {
	return nil
}

func (s *stubEventBus) PublishEvent(_ context.Context, ev domain.EventPublish) (string, error) {
	s.capturedEvent = ev
	if s.publishErr != nil {
		return "", s.publishErr
	}
	return s.publishID, nil
}

func newServerWithEventBus(b domain.EventBusPort) *httptest.Server {
	svc := domain.NewApplyService(&stubCompiler{}, &stubEngine{}, b)
	h := api.NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return httptest.NewServer(mux)
}

func TestHandler_PublishEvent_Returns202AndEventID(t *testing.T) {
	bus := &stubEventBus{publishID: "evt-77"}
	srv := newServerWithEventBus(bus)
	defer srv.Close()

	body := `{"event_type":"review.approved","data":{"by":"alice"}}`
	resp, err := http.Post(srv.URL+"/api/v1/workflows/run-7/events", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status: got %d, want 202", resp.StatusCode)
	}
	m := decodeBody(t, resp)
	if m["event_id"] != "evt-77" {
		t.Errorf("event_id: got %v, want evt-77", m["event_id"])
	}
	if bus.capturedEvent.RunID != "run-7" {
		t.Errorf("run id: got %q, want run-7 (from path)", bus.capturedEvent.RunID)
	}
	if bus.capturedEvent.Type != "review.approved" {
		t.Errorf("type: got %q, want review.approved", bus.capturedEvent.Type)
	}
	if !strings.Contains(string(bus.capturedEvent.Data), `"by":"alice"`) {
		t.Errorf("data forwarded verbatim: got %q", bus.capturedEvent.Data)
	}
}

func TestHandler_PublishEvent_MissingType_Returns400(t *testing.T) {
	srv := newServerWithEventBus(&stubEventBus{})
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/workflows/run-7/events", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", resp.StatusCode)
	}
}

func TestHandler_PublishEvent_InvalidJSON_Returns400(t *testing.T) {
	srv := newServerWithEventBus(&stubEventBus{})
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/workflows/run-7/events", "application/json", strings.NewReader(`{not json`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", resp.StatusCode)
	}
}

func TestHandler_PublishEvent_NoEventBus_Returns503(t *testing.T) {
	// nil event bus → service returns ErrEngineUnavailable → 503.
	svc := domain.NewApplyService(&stubCompiler{}, &stubEngine{}, nil)
	h := api.NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/v1/workflows/run-7/events", "application/json", strings.NewReader(`{"event_type":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", resp.StatusCode)
	}
}
