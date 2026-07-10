// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax/client"
)

func newGW(t *testing.T, handler http.HandlerFunc) *client.Gateway {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return client.New(srv.URL, false, "")
}

// newGWKey is newGW with a configured bearer API key (issue #1517).
func newGWKey(t *testing.T, apiKey string, handler http.HandlerFunc) *client.Gateway {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return client.New(srv.URL, false, apiKey)
}

func TestApply_Workflow_Returns202(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/apply" {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"run_id": "wf-001", "warnings": []string{"w1"}})
	})
	runID, agentID, warnings, err := gw.Apply(context.Background(), []byte("kind: Workflow"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runID != "wf-001" {
		t.Errorf("run_id = %q; want wf-001", runID)
	}
	if agentID != "" {
		t.Errorf("agent_id should be empty for Workflow apply")
	}
	if len(warnings) != 1 || warnings[0] != "w1" {
		t.Errorf("warnings = %v; want [w1]", warnings)
	}
}

// TestApply_AgentDef_Retired410 covers AC1 of #1697: applying a kind: AgentDef
// manifest returns the documented retirement error naming the Agent custom
// resource — the gateway's push forward is deleted (ADR-039) and answers 410.
func TestApply_AgentDef_Retired410(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGone)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "AgentDef push registration retired (ADR-039) — apply a zynax.io/v1alpha1 Agent custom resource with kubectl instead (docs/patterns/agent-crd-migration.md)",
			"code":  "AGENTDEF_RETIRED",
		})
	})
	runID, agentID, _, err := gw.Apply(context.Background(), []byte("kind: AgentDef"), "")
	if err == nil {
		t.Fatal("expected a retirement error for kind: AgentDef, got nil")
	}
	if runID != "" || agentID != "" {
		t.Errorf("no ids expected for a retired AgentDef apply; got run=%q agent=%q", runID, agentID)
	}
	for _, want := range []string{"Agent custom resource", "kubectl", "agent-crd-migration"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("retirement error must mention %q; got %q", want, err.Error())
		}
	}
}

// TestApply_Created201_DecodesAgentID keeps coverage of the 201/agent_id decode
// branch. No production endpoint returns 201 since AgentDef push was retired
// (ADR-039); this asserts the generic decode path only.
func TestApply_Created201_DecodesAgentID(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"agent_id": "agent-xyz"})
	})
	runID, agentID, _, err := gw.Apply(context.Background(), []byte("kind: Workflow"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agentID != "agent-xyz" {
		t.Errorf("agent_id = %q; want agent-xyz", agentID)
	}
	if runID != "" {
		t.Errorf("run_id should be empty for a 201 response")
	}
}

func TestApply_CompileErrors_Returns422(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{{"code": "MISSING_FIELD", "message": "missing name", "line": 3}},
		})
	})
	_, _, _, err := gw.Apply(context.Background(), []byte("bad yaml"), "")
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
}

func TestApplyDryRun_Valid_ReturnsWarnings(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"dry_run": true, "warnings": []string{"unused state"}})
	})
	errs, warnings, err := gw.ApplyDryRun(context.Background(), []byte("kind: Workflow"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	if len(warnings) != 1 {
		t.Errorf("warnings = %v; want 1 entry", warnings)
	}
}

func TestApplyDryRun_CompileErrors_Returned(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{{"message": "bad field", "line": 5}},
		})
	})
	errs, _, err := gw.ApplyDryRun(context.Background(), []byte("bad"))
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 compile error, got %d", len(errs))
	}
}

func TestGetWorkflow_ReturnsStatus(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/workflows/run-123" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"run_id": "run-123", "status": "WORKFLOW_STATUS_RUNNING",
		})
	})
	s, err := gw.GetWorkflow(context.Background(), "run-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status != "WORKFLOW_STATUS_RUNNING" {
		t.Errorf("status = %q", s.Status)
	}
}

func TestGetWorkflow_NotFound(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := gw.GetWorkflow(context.Background(), "missing")
	if !errors.Is(err, client.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetWorkflowOutputs_ReturnsMap(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/workflows/run-42/outputs" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"review": "LGTM", "score": "9"})
	})
	out, err := gw.GetWorkflowOutputs(context.Background(), "run-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["review"] != "LGTM" || out["score"] != "9" {
		t.Errorf("outputs = %v, want review=LGTM score=9", out)
	}
}

func TestGetWorkflowOutputs_NotFound(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := gw.GetWorkflowOutputs(context.Background(), "missing")
	if !errors.Is(err, client.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteWorkflow_Returns204(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	if err := gw.DeleteWorkflow(context.Background(), "run-abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteWorkflow_NotFound(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	err := gw.DeleteWorkflow(context.Background(), "missing")
	if !errors.Is(err, client.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestWatchWorkflowLogs_StreamsEvents(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/workflows/run-sse/logs" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		events := []map[string]any{
			{"run_id": "run-sse", "event_type": "state.entered", "to_state": "review", "status": "WORKFLOW_STATUS_RUNNING"},
			{"run_id": "run-sse", "event_type": "workflow.completed", "status": "WORKFLOW_STATUS_COMPLETED"},
		}
		for _, ev := range events {
			b, _ := json.Marshal(ev)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		}
	})

	var got []client.LogEvent
	err := gw.WatchWorkflowLogs(context.Background(), "run-sse", func(ev client.LogEvent) error {
		got = append(got, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	if got[0].EventType != "state.entered" {
		t.Errorf("event[0].EventType = %q; want state.entered", got[0].EventType)
	}
	if got[1].EventType != "workflow.completed" {
		t.Errorf("event[1].EventType = %q; want workflow.completed", got[1].EventType)
	}
}

func TestWatchWorkflowLogs_NotFound(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	err := gw.WatchWorkflowLogs(context.Background(), "ghost", func(_ client.LogEvent) error { return nil })
	if !errors.Is(err, client.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPublishEvent_Returns202(t *testing.T) {
	var gotPath, gotMethod, gotType string
	var gotBody struct {
		EventType string            `json:"event_type"`
		Data      map[string]string `json:"data"`
	}
	gw := newGW(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		gotType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"event_id": "evt-42"})
	})
	eventID, err := gw.PublishEvent(context.Background(), "run-7", "review.approved", map[string]string{"by": "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eventID != "evt-42" {
		t.Errorf("event_id = %q; want evt-42", eventID)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q; want POST", gotMethod)
	}
	if gotPath != "/api/v1/workflows/run-7/events" {
		t.Errorf("path = %q; want /api/v1/workflows/run-7/events", gotPath)
	}
	if gotType != "application/json" {
		t.Errorf("content-type = %q; want application/json", gotType)
	}
	if gotBody.EventType != "review.approved" {
		t.Errorf("event_type = %q; want review.approved", gotBody.EventType)
	}
	if gotBody.Data["by"] != "alice" {
		t.Errorf("data[by] = %q; want alice", gotBody.Data["by"])
	}
}

func TestPublishEvent_NotFound(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := gw.PublishEvent(context.Background(), "ghost", "review.approved", nil)
	if !errors.Is(err, client.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestPublishEvent_ServerError(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	_, err := gw.PublishEvent(context.Background(), "run-7", "review.approved", nil)
	if err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}

// TestAuthorizationHeader_WithKey asserts every verb sends the bearer token when
// an API key is configured (issue #1517 — AC2: key on all gateway calls). The
// handler writes a permissive response so each verb completes without error; the
// assertion is only on the captured Authorization header.
func TestAuthorizationHeader_WithKey(t *testing.T) {
	const key = "s3cr3t-token"
	want := "Bearer " + key

	cases := []struct {
		name   string
		status int
		call   func(*client.Gateway) error
	}{
		{"apply POST", http.StatusAccepted, func(g *client.Gateway) error {
			_, _, _, err := g.Apply(context.Background(), []byte("kind: Workflow"), "")
			return err
		}},
		{"get workflow GET", http.StatusOK, func(g *client.Gateway) error {
			_, err := g.GetWorkflow(context.Background(), "run-1")
			return err
		}},
		{"delete workflow DELETE", http.StatusNoContent, func(g *client.Gateway) error {
			return g.DeleteWorkflow(context.Background(), "run-1")
		}},
		{"publish event POST", http.StatusAccepted, func(g *client.Gateway) error {
			_, err := g.PublishEvent(context.Background(), "run-1", "approved", nil)
			return err
		}},
		{"watch logs SSE GET", http.StatusOK, func(g *client.Gateway) error {
			return g.WatchWorkflowLogs(context.Background(), "run-1", func(client.LogEvent) error { return nil })
		}},
		{"health GET", http.StatusOK, func(g *client.Gateway) error {
			_, err := g.Health(context.Background())
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			gw := newGWKey(t, key, func(w http.ResponseWriter, r *http.Request) {
				got = r.Header.Get("Authorization")
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"run_id": "run-1", "status": "ok", "event_id": "e1",
				})
			})
			if err := tc.call(gw); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != want {
				t.Errorf("Authorization = %q; want %q", got, want)
			}
		})
	}
}

// TestAuthorizationHeader_NoKey asserts no Authorization header is sent when no
// API key is configured (issue #1517 — AC3: backward compatible, auth-disabled
// gateways keep working).
func TestAuthorizationHeader_NoKey(t *testing.T) {
	var got string
	var seen bool
	gw := newGW(t, func(w http.ResponseWriter, r *http.Request) {
		_, seen = r.Header["Authorization"]
		got = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{"run_id": "wf-1"})
	})
	if _, _, _, err := gw.Apply(context.Background(), []byte("kind: Workflow"), ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen || got != "" {
		t.Errorf("Authorization header present (%q) without an API key; want none", got)
	}
}

func TestCompletionText(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{"empty", "", ""},
		{"not json", "not json", ""},
		{"wrapped result_payload", `{"workflow_id":"w","result_payload":"{\"completion\":\"hello world\"}"}`, "hello world"},
		{"bare completion", `{"completion":"bare text"}`, "bare text"},
		{"no completion field", `{"task_id":"t","status":"COMPLETED"}`, ""},
		{"result_payload not json", `{"result_payload":"plain"}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := client.CompletionText(tt.payload); got != tt.want {
				t.Errorf("CompletionText(%q) = %q, want %q", tt.payload, got, tt.want)
			}
		})
	}
}
