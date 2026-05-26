// SPDX-License-Identifier: Apache-2.0

package adapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/adapter"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// stubStream implements AgentService_ExecuteCapabilityServer for testing.
type stubStream struct {
	events []*zynaxv1.TaskEvent
	ctx    context.Context
}

func (s *stubStream) Send(e *zynaxv1.TaskEvent) error {
	s.events = append(s.events, e)
	return nil
}
func (s *stubStream) Context() context.Context     { return s.ctx }
func (s *stubStream) SetHeader(metadata.MD) error  { return nil }
func (s *stubStream) SendHeader(metadata.MD) error { return nil }
func (s *stubStream) SetTrailer(metadata.MD)       {}
func (s *stubStream) SendMsg(interface{}) error    { return nil }
func (s *stubStream) RecvMsg(interface{}) error    { return nil }

func (s *stubStream) last() *zynaxv1.TaskEvent {
	return s.events[len(s.events)-1]
}

func newTestServer(t *testing.T, apiURL string) *adapter.AgentServer {
	t.Helper()
	cfg := &config.AdapterConfig{
		AgentID:          "git-test",
		Name:             "Git Test",
		Endpoint:         ":50060",
		RegistryEndpoint: "localhost:50052",
		Git: config.GitConfig{
			Provider: "github",
			AuthEnv:  "TEST_GITHUB_TOKEN",
		},
		Capabilities: []config.GitCapabilityConfig{
			{Name: "open_pr", Owner: "test-owner", Repo: "test-repo", TimeoutSeconds: 5},
			{Name: "request_review", Owner: "test-owner", Repo: "test-repo", TimeoutSeconds: 5},
			{Name: "get_diff", Owner: "test-owner", Repo: "test-repo", TimeoutSeconds: 5},
		},
	}
	return adapter.NewAgentServerWithURL(cfg, "fake-token", apiURL)
}

func TestExecuteCapability_UnknownCapability(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	stream := &stubStream{ctx: context.Background()}
	req := &zynaxv1.ExecuteCapabilityRequest{TaskId: "t1", CapabilityName: "nonexistent"}
	err := srv.ExecuteCapability(req, stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stream.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(stream.events))
	}
	if stream.events[0].EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Errorf("expected FAILED, got %v", stream.events[0].EventType)
	}
}

func TestExecuteCapability_MissingTaskID(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	stream := &stubStream{ctx: context.Background()}
	req := &zynaxv1.ExecuteCapabilityRequest{CapabilityName: "open_pr"}
	err := srv.ExecuteCapability(req, stream)
	if err == nil {
		t.Fatal("expected gRPC error for missing task_id")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestExecuteCapability_OpenPR_Success(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"number":   42,
			"html_url": "https://github.com/test-owner/test-repo/pull/42",
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	srv := newTestServer(t, ts.URL)
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]string{
		"title": "Test PR",
		"head":  "feature-branch",
		"base":  "main",
	})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-open-pr",
		CapabilityName: "open_pr",
		InputPayload:   payload,
	}
	err := srv.ExecuteCapability(req, stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Fatalf("expected COMPLETED, got %v", last.EventType)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(last.Payload, &out); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if out["pr_url"] == "" {
		t.Errorf("expected non-empty pr_url")
	}
}

func TestExecuteCapability_OpenPR_GitHubError429(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Limit", "60")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1893456000")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "API rate limit exceeded"})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	srv := newTestServer(t, ts.URL)
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]string{"title": "PR", "head": "feat", "base": "main"})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-429",
		CapabilityName: "open_pr",
		InputPayload:   payload,
	}
	_ = srv.ExecuteCapability(req, stream)
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Fatalf("expected FAILED, got %v", last.EventType)
	}
	if last.Error.Code != "RESOURCE_EXHAUSTED" {
		t.Errorf("expected RESOURCE_EXHAUSTED, got %q", last.Error.Code)
	}
	if len(last.Error.Message) > 512 {
		t.Errorf("error message exceeds 512 chars (sanitise failed): len=%d", len(last.Error.Message))
	}
}

func TestGetCapabilitySchema_Found(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	resp, err := srv.GetCapabilitySchema(context.Background(),
		&zynaxv1.GetCapabilitySchemaRequest{CapabilityName: "open_pr"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CapabilityName != "open_pr" {
		t.Errorf("got %q", resp.CapabilityName)
	}
}

func TestGetCapabilitySchema_NotFound(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	_, err := srv.GetCapabilitySchema(context.Background(),
		&zynaxv1.GetCapabilitySchemaRequest{CapabilityName: "nope"})
	if err == nil {
		t.Fatal("expected error")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}

func TestExecuteCapability_GetDiff_Success(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/7", func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "diff") {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("diff --git a/foo.go b/foo.go\n+foo\n"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"number": 7})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	srv := newTestServer(t, ts.URL)
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]int{"pr_number": 7})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-diff",
		CapabilityName: "get_diff",
		InputPayload:   payload,
	}
	err := srv.ExecuteCapability(req, stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Fatalf("expected COMPLETED, got %v", last.EventType)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(last.Payload, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	diff, _ := out["diff"].(string)
	if !strings.Contains(diff, "diff --git") {
		t.Errorf("expected diff content, got %q", diff)
	}
	if out["truncated"] != false {
		t.Errorf("expected truncated=false")
	}
}
