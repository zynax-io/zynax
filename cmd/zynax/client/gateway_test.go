// SPDX-License-Identifier: Apache-2.0

package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zynax-io/zynax/cmd/zynax/client"
)

func newGW(t *testing.T, handler http.HandlerFunc) *client.Gateway {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return client.New(srv.URL, false)
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

func TestApply_AgentDef_Returns201(t *testing.T) {
	gw := newGW(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"agent_id": "agent-xyz"})
	})
	runID, agentID, _, err := gw.Apply(context.Background(), []byte("kind: AgentDef"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agentID != "agent-xyz" {
		t.Errorf("agent_id = %q; want agent-xyz", agentID)
	}
	if runID != "" {
		t.Errorf("run_id should be empty for AgentDef apply")
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
	if err != client.ErrNotFound {
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
	if err != client.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
