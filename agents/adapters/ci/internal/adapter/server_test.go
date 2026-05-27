// SPDX-License-Identifier: Apache-2.0

package adapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/ci/internal/adapter"
	"github.com/zynax-io/zynax/agents/adapters/ci/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	codeInvalidInput      = "INVALID_INPUT"
	codeResourceExhausted = "RESOURCE_EXHAUSTED"
)

// stubStream captures events; implements AgentService_ExecuteCapabilityServer.
type stubStream struct {
	events []*zynaxv1.TaskEvent
	ctx    context.Context
}

func newStream() *stubStream                          { return &stubStream{ctx: context.Background()} }
func (s *stubStream) Send(e *zynaxv1.TaskEvent) error { s.events = append(s.events, e); return nil }
func (s *stubStream) Context() context.Context        { return s.ctx }
func (s *stubStream) SetHeader(metadata.MD) error     { return nil }
func (s *stubStream) SendHeader(metadata.MD) error    { return nil }
func (s *stubStream) SetTrailer(metadata.MD)          {}
func (s *stubStream) SendMsg(_ interface{}) error     { return nil }
func (s *stubStream) RecvMsg(_ interface{}) error     { return nil }
func (s *stubStream) last() *zynaxv1.TaskEvent {
	if len(s.events) == 0 {
		return nil
	}
	return s.events[len(s.events)-1]
}

func minimalCfg() *config.AdapterConfig {
	return &config.AdapterConfig{
		AgentID:          "ci-test",
		Endpoint:         ":50099",
		RegistryEndpoint: "localhost:50052",
		CI: config.CIConfig{
			Provider:                  "github-actions",
			TokenEnv:                  "GH_TOKEN",
			PollIntervalSeconds:       1,
			MaxPollIntervalSeconds:    2,
			TriggerPollTimeoutSeconds: 3,
		},
		Capabilities: []config.CICapabilityConfig{
			{Name: "trigger_workflow", Owner: "zynax-io", Repo: "zynax", WorkflowID: "ci.yml"},
			{Name: "get_run_status", Owner: "zynax-io", Repo: "zynax", WorkflowID: "ci.yml"},
		},
	}
}

func jsonBytes(v interface{}) []byte { b, _ := json.Marshal(v); return b }

// ── GetCapabilitySchema ───────────────────────────────────────────────────────

func TestGetCapabilitySchema(t *testing.T) {
	t.Parallel()
	cfg := minimalCfg()
	cfg.Capabilities[0].Description = "runs a workflow"
	cfg.Capabilities[0].InputSchemaJSON = `{"type":"object"}`
	cfg.Capabilities[0].OutputSchemaJSON = `{"type":"object"}`
	srv := adapter.NewAgentServerWithURL(cfg, "tok", "")
	cases := []struct {
		name    string
		cap     string
		wantErr codes.Code
	}{
		{"known", "trigger_workflow", codes.OK},
		{"unknown", "nonexistent", codes.NotFound},
		{"empty", "", codes.InvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := srv.GetCapabilitySchema(context.Background(),
				&zynaxv1.GetCapabilitySchemaRequest{CapabilityName: tc.cap})
			if tc.wantErr == codes.OK {
				if err != nil {
					t.Fatalf("want ok, got %v", err)
				}
				if resp.CapabilityName != tc.cap {
					t.Errorf("cap name: got %q, want %q", resp.CapabilityName, tc.cap)
				}
				return
			}
			if status.Code(err) != tc.wantErr {
				t.Errorf("want code %v, got %v", tc.wantErr, err)
			}
		})
	}
}

// ── ExecuteCapability guards ──────────────────────────────────────────────────

func TestExecuteCapability_Guards(t *testing.T) {
	t.Parallel()
	jenkins := minimalCfg()
	jenkins.CI.Provider = "jenkins-stub"
	cases := []struct {
		name     string
		cfg      *config.AdapterConfig
		taskID   string
		cap      string
		payload  []byte
		wantGRPC codes.Code // codes.OK → check event code via wantCode
		wantCode string
	}{
		{
			name: "empty_task_id", cfg: minimalCfg(), cap: "trigger_workflow",
			payload: jsonBytes(map[string]string{"ref": "main"}), wantGRPC: codes.InvalidArgument,
		},
		{
			name: "empty_cap", cfg: minimalCfg(), taskID: "t1", wantGRPC: codes.InvalidArgument,
		},
		{
			name: "unknown_cap",
			cfg:  minimalCfg(), taskID: "t1", cap: "nope",
			wantGRPC: codes.OK, wantCode: codeInvalidInput,
		},
		{
			name: "jenkins_stub",
			cfg:  jenkins, taskID: "t1", cap: "trigger_workflow",
			payload:  jsonBytes(map[string]string{"ref": "main"}),
			wantGRPC: codes.OK, wantCode: "INTERNAL",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := adapter.NewAgentServerWithURL(tc.cfg, "tok", "")
			s := newStream()
			err := srv.ExecuteCapability(&zynaxv1.ExecuteCapabilityRequest{
				TaskId: tc.taskID, CapabilityName: tc.cap, InputPayload: tc.payload,
			}, s)
			if tc.wantGRPC != codes.OK {
				if status.Code(err) != tc.wantGRPC {
					t.Errorf("want grpc %v, got %v", tc.wantGRPC, err)
				}
				return
			}
			last := s.last()
			if last == nil || last.Error == nil || last.Error.Code != tc.wantCode {
				t.Errorf("want event code %q, got %v", tc.wantCode, last)
			}
		})
	}
}

// ── trigger_workflow ──────────────────────────────────────────────────────────

func TestTriggerWorkflow_Success(t *testing.T) {
	t.Parallel()
	var dispatched bool
	var pollCount int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "dispatches"):
			dispatched = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "runs"):
			pollCount++
			if pollCount < 2 {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"workflow_runs": []interface{}{}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []interface{}{
					map[string]interface{}{
						"id":         int64(999),
						"html_url":   "https://github.com/zynax-io/zynax/actions/runs/999",
						"created_at": time.Now().Add(-time.Second).UTC().Format(time.RFC3339),
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	s := newStream()
	err := adapter.NewAgentServerWithURL(minimalCfg(), "tok", ts.URL).ExecuteCapability(
		&zynaxv1.ExecuteCapabilityRequest{
			TaskId: "task-1", CapabilityName: "trigger_workflow",
			InputPayload: jsonBytes(map[string]string{"ref": "main"}), TimeoutSeconds: 10,
		}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dispatched {
		t.Error("dispatch POST not called")
	}
	last := s.last()
	if last == nil || last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Fatalf("expected COMPLETED, got %v", last)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(last.Payload, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out["run_id"] == nil || out["run_url"] == nil {
		t.Error("run_id or run_url missing from output payload")
	}
}

func TestTriggerWorkflow_InputErrors(t *testing.T) {
	t.Parallel()
	srv := adapter.NewAgentServerWithURL(minimalCfg(), "tok", "http://localhost")
	cases := []struct {
		name    string
		payload []byte
	}{
		{"nil_payload", nil},
		{"missing_ref", []byte(`{}`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newStream()
			_ = srv.ExecuteCapability(&zynaxv1.ExecuteCapabilityRequest{
				TaskId: "t1", CapabilityName: "trigger_workflow", InputPayload: tc.payload,
			}, s)
			last := s.last()
			if last == nil || last.Error == nil || last.Error.Code != codeInvalidInput {
				t.Errorf("want INVALID_INPUT, got %v", last)
			}
		})
	}
}

func TestTriggerWorkflow_DispatchHTTPErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		code int
	}{
		{"429", http.StatusTooManyRequests},
		{"403", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.code)
			}))
			defer ts.Close()
			s := newStream()
			_ = adapter.NewAgentServerWithURL(minimalCfg(), "tok", ts.URL).ExecuteCapability(
				&zynaxv1.ExecuteCapabilityRequest{
					TaskId: "t1", CapabilityName: "trigger_workflow",
					InputPayload: jsonBytes(map[string]string{"ref": "main"}),
				}, s)
			last := s.last()
			if last == nil || last.Error == nil || last.Error.Code != codeResourceExhausted {
				t.Errorf("want RESOURCE_EXHAUSTED, got %v", last)
			}
		})
	}
}

func TestTriggerWorkflow_TriggerTimeout(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"workflow_runs": []interface{}{}})
	}))
	defer ts.Close()
	cfg := minimalCfg()
	cfg.CI.TriggerPollTimeoutSeconds = 2
	s := newStream()
	_ = adapter.NewAgentServerWithURL(cfg, "tok", ts.URL).ExecuteCapability(
		&zynaxv1.ExecuteCapabilityRequest{
			TaskId: "t1", CapabilityName: "trigger_workflow",
			InputPayload: jsonBytes(map[string]string{"ref": "main"}), TimeoutSeconds: 10,
		}, s)
	last := s.last()
	if last == nil || last.Error == nil || last.Error.Code != "TIMEOUT" {
		t.Fatalf("want TIMEOUT, got %v", last)
	}
}

// ── get_run_status ────────────────────────────────────────────────────────────

func TestGetRunStatus_SuccessWithProgress(t *testing.T) {
	t.Parallel()
	var calls int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		st, c := "in_progress", ""
		if calls >= 3 {
			st, c = "completed", "success"
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": st, "conclusion": c})
	}))
	defer ts.Close()

	s := newStream()
	err := adapter.NewAgentServerWithURL(minimalCfg(), "tok", ts.URL).ExecuteCapability(
		&zynaxv1.ExecuteCapabilityRequest{
			TaskId: "task-2", CapabilityName: "get_run_status",
			InputPayload: jsonBytes(map[string]int64{"run_id": 42}), TimeoutSeconds: 15,
		}, s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.events) < 2 {
		t.Fatalf("expected ≥2 events (progress+completed), got %d", len(s.events))
	}
	var progressSeen bool
	for _, e := range s.events {
		if e.EventType == zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS {
			progressSeen = true
		}
	}
	if !progressSeen {
		t.Error("no PROGRESS event emitted")
	}
	last := s.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Errorf("expected COMPLETED, got %v", last.EventType)
	}
	var out map[string]string
	_ = json.Unmarshal(last.Payload, &out)
	if out["conclusion"] != "success" {
		t.Errorf("expected conclusion=success, got %q", out["conclusion"])
	}
}

func TestGetRunStatus_InputErrors(t *testing.T) {
	t.Parallel()
	srv := adapter.NewAgentServerWithURL(minimalCfg(), "tok", "http://localhost")
	cases := []struct {
		name    string
		payload []byte
	}{
		{"nil_payload", nil},
		{"zero_run_id", []byte(`{"run_id": 0}`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newStream()
			_ = srv.ExecuteCapability(&zynaxv1.ExecuteCapabilityRequest{
				TaskId: "t1", CapabilityName: "get_run_status", InputPayload: tc.payload,
			}, s)
			last := s.last()
			if last == nil || last.Error == nil || last.Error.Code != codeInvalidInput {
				t.Errorf("want INVALID_INPUT, got %v", last)
			}
		})
	}
}

func TestGetRunStatus_ContextTimeout(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "in_progress", "conclusion": ""})
	}))
	defer ts.Close()
	s := newStream()
	_ = adapter.NewAgentServerWithURL(minimalCfg(), "tok", ts.URL).ExecuteCapability(
		&zynaxv1.ExecuteCapabilityRequest{
			TaskId: "t1", CapabilityName: "get_run_status",
			InputPayload: jsonBytes(map[string]int64{"run_id": 99}), TimeoutSeconds: 3,
		}, s)
	last := s.last()
	if last == nil || last.Error == nil || last.Error.Code != "TIMEOUT" {
		t.Fatalf("want TIMEOUT, got %v", last)
	}
}

func TestGetRunStatus_APIError429(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()
	s := newStream()
	_ = adapter.NewAgentServerWithURL(minimalCfg(), "tok", ts.URL).ExecuteCapability(
		&zynaxv1.ExecuteCapabilityRequest{
			TaskId: "t1", CapabilityName: "get_run_status",
			InputPayload: jsonBytes(map[string]int64{"run_id": 99}), TimeoutSeconds: 5,
		}, s)
	last := s.last()
	if last == nil || last.Error == nil || last.Error.Code != codeResourceExhausted {
		t.Fatalf("want RESOURCE_EXHAUSTED, got %v", last)
	}
}
