// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// capturedRequest records the wire-level request the fake Argo server received.
type capturedRequest struct {
	method      string
	path        string
	contentType string
	authz       string
	body        []byte
}

// newFakeArgoServer starts an httptest server that records the last request and
// replies with the given status code and response body.
func newFakeArgoServer(t *testing.T, statusCode int, respBody string) (*httptest.Server, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.method = r.Method
		captured.path = r.URL.Path
		captured.contentType = r.Header.Get("Content-Type")
		captured.authz = r.Header.Get("Authorization")
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("fake argo server: read body: %v", err)
		}
		captured.body = b
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(srv.Close)
	return srv, captured
}

// ─── SubmitWorkflow wire contract ────────────────────────────────────────────

// TestHTTPArgoClient_SubmitWorkflow_PostsCreateRequestEnvelope is the golden-body
// regression test for #1157: the Argo Workflows server API rejects a bare
// Workflow manifest on POST /api/v1/workflows/{namespace} with HTTP 422 — the
// body must be the WorkflowCreateRequest envelope ({"namespace", "workflow"}).
func TestHTTPArgoClient_SubmitWorkflow_PostsCreateRequestEnvelope(t *testing.T) {
	srv, captured := newFakeArgoServer(t, http.StatusOK, `{}`)
	client := NewHTTPArgoClient(srv.URL, "secret-token", nil)

	wf := buildArgoWorkflow("argo-wf-1157", `{"workflow_id":"argo-wf-1157"}`,
		map[string]string{"env": "e2e"}, defaultConfig())

	if err := client.SubmitWorkflow(context.Background(), testArgoNamespace, wf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.method != http.MethodPost {
		t.Errorf("method = %q; want POST", captured.method)
	}
	wantPath := "/api/v1/workflows/" + testArgoNamespace
	if captured.path != wantPath {
		t.Errorf("path = %q; want %q", captured.path, wantPath)
	}
	if captured.contentType != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", captured.contentType)
	}
	if captured.authz != "Bearer secret-token" {
		t.Errorf("Authorization = %q; want %q", captured.authz, "Bearer secret-token")
	}

	// Golden body: the exact JSON the Argo 3.7 server contract requires
	// (io.argoproj.workflow.v1alpha1.WorkflowCreateRequest). Field order is
	// deterministic — encoding/json follows Go struct field order.
	wantBody := `{"namespace":"zynax-test","workflow":{` +
		`"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow",` +
		`"metadata":{"name":"argo-wf-1157","namespace":"zynax-test","labels":{"env":"e2e"}},` +
		`"spec":{"workflowTemplateRef":{"name":"zynax-ir-runner"},` +
		`"arguments":{"parameters":[{"name":"workflow-ir","value":"{\"workflow_id\":\"argo-wf-1157\"}"}]},` +
		`"serviceAccountName":"zynax-sa"},` +
		`"status":{}}}`
	if string(captured.body) != wantBody {
		t.Errorf("submit body mismatch\n got: %s\nwant: %s", captured.body, wantBody)
	}
}

func TestHTTPArgoClient_SubmitWorkflow_NoAuthHeaderWhenTokenEmpty(t *testing.T) {
	srv, captured := newFakeArgoServer(t, http.StatusOK, `{}`)
	client := NewHTTPArgoClient(srv.URL, "", nil)

	wf := buildArgoWorkflow("wf-noauth", "{}", nil, defaultConfig())
	if err := client.SubmitWorkflow(context.Background(), testArgoNamespace, wf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if captured.authz != "" {
		t.Errorf("Authorization = %q; want empty", captured.authz)
	}
}

func TestHTTPArgoClient_SubmitWorkflow_HTTPError(t *testing.T) {
	srv, _ := newFakeArgoServer(t, http.StatusUnprocessableEntity, `{"message":"unknown field"}`)
	client := NewHTTPArgoClient(srv.URL, "", nil)

	wf := buildArgoWorkflow("wf-422", "{}", nil, defaultConfig())
	err := client.SubmitWorkflow(context.Background(), testArgoNamespace, wf)
	if err == nil {
		t.Fatal("expected error on HTTP 422")
	}
	if !containsStr(err.Error(), "HTTP 422") {
		t.Errorf("error should mention HTTP 422, got: %v", err)
	}
}

// ─── GetWorkflow wire contract ───────────────────────────────────────────────

func TestHTTPArgoClient_GetWorkflow_DecodesBareWorkflow(t *testing.T) {
	// Per the Argo server API, the GetWorkflow response is the Workflow
	// resource itself — no envelope.
	resp := `{"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow",` +
		`"metadata":{"name":"wf-get","namespace":"zynax-test"},` +
		`"status":{"phase":"Succeeded","startedAt":"2026-06-12T10:00:00Z","finishedAt":"2026-06-12T10:01:00Z"}}`
	srv, captured := newFakeArgoServer(t, http.StatusOK, resp)
	client := NewHTTPArgoClient(srv.URL, "", nil)

	wf, err := client.GetWorkflow(context.Background(), testArgoNamespace, "wf-get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantPath := "/api/v1/workflows/" + testArgoNamespace + "/wf-get"
	if captured.path != wantPath {
		t.Errorf("path = %q; want %q", captured.path, wantPath)
	}
	if captured.method != http.MethodGet {
		t.Errorf("method = %q; want GET", captured.method)
	}
	if wf.Status.Phase != ArgoPhaseSucceeded {
		t.Errorf("phase = %q; want %q", wf.Status.Phase, ArgoPhaseSucceeded)
	}
	if wf.Metadata.Name != "wf-get" {
		t.Errorf("name = %q; want wf-get", wf.Metadata.Name)
	}
}

func TestHTTPArgoClient_GetWorkflow_NotFound(t *testing.T) {
	srv, _ := newFakeArgoServer(t, http.StatusNotFound, `{"message":"not found"}`)
	client := NewHTTPArgoClient(srv.URL, "", nil)

	_, err := client.GetWorkflow(context.Background(), testArgoNamespace, "missing")
	if !errors.Is(err, errArgoNotFound) {
		t.Errorf("expected errArgoNotFound, got: %v", err)
	}
}

// ─── DeleteWorkflow wire contract ────────────────────────────────────────────

func TestHTTPArgoClient_DeleteWorkflow_Success(t *testing.T) {
	srv, captured := newFakeArgoServer(t, http.StatusOK, `{}`)
	client := NewHTTPArgoClient(srv.URL, "", nil)

	if err := client.DeleteWorkflow(context.Background(), testArgoNamespace, "wf-del"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantPath := "/api/v1/workflows/" + testArgoNamespace + "/wf-del"
	if captured.path != wantPath {
		t.Errorf("path = %q; want %q", captured.path, wantPath)
	}
	if captured.method != http.MethodDelete {
		t.Errorf("method = %q; want DELETE", captured.method)
	}
}

func TestHTTPArgoClient_DeleteWorkflow_NotFound(t *testing.T) {
	srv, _ := newFakeArgoServer(t, http.StatusNotFound, `{"message":"not found"}`)
	client := NewHTTPArgoClient(srv.URL, "", nil)

	err := client.DeleteWorkflow(context.Background(), testArgoNamespace, "missing")
	if !errors.Is(err, errArgoNotFound) {
		t.Errorf("expected errArgoNotFound, got: %v", err)
	}
}

// ─── SendEvent wire contract ─────────────────────────────────────────────────

// TestHTTPArgoClient_SendEvent_PostsRawPayload asserts the event body is the
// payload itself (Argo EventService_ReceiveEvent takes the Item directly),
// not a {"payload": ...} envelope — same contract class as #1157.
func TestHTTPArgoClient_SendEvent_PostsRawPayload(t *testing.T) {
	srv, captured := newFakeArgoServer(t, http.StatusOK, `{}`)
	client := NewHTTPArgoClient(srv.URL, "", nil)

	payload := []byte(`{"status":"approved"}`)
	if err := client.SendEvent(context.Background(), testArgoNamespace, "run-1.review.approved", payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantPath := "/api/v1/events/" + testArgoNamespace + "/run-1.review.approved"
	if captured.path != wantPath {
		t.Errorf("path = %q; want %q", captured.path, wantPath)
	}
	if string(captured.body) != string(payload) {
		t.Errorf("event body = %s; want raw payload %s", captured.body, payload)
	}
}

func TestHTTPArgoClient_SendEvent_EmptyPayloadSendsEmptyObject(t *testing.T) {
	srv, captured := newFakeArgoServer(t, http.StatusOK, `{}`)
	client := NewHTTPArgoClient(srv.URL, "", nil)

	if err := client.SendEvent(context.Background(), testArgoNamespace, "run-2.start", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(captured.body) != "{}" {
		t.Errorf("event body = %q; want %q", captured.body, "{}")
	}
}

func TestHTTPArgoClient_SendEvent_HTTPError(t *testing.T) {
	srv, _ := newFakeArgoServer(t, http.StatusBadRequest, `{"message":"bad event"}`)
	client := NewHTTPArgoClient(srv.URL, "", nil)

	err := client.SendEvent(context.Background(), testArgoNamespace, "run-3.x", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error on HTTP 400")
	}
	if !containsStr(err.Error(), "HTTP 400") {
		t.Errorf("error should mention HTTP 400, got: %v", err)
	}
}
