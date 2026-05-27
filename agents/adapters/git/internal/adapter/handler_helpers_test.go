// SPDX-License-Identifier: Apache-2.0

package adapter_test

// Tests for execute dispatch, sanitise, githubErrCode, and parsePayload helpers.
// Closes #716 — part of the git-adapter coverage epic (#713).

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
)

// codeInvalidInput is the CapabilityError code produced by unknown/bad input paths.
const codeInvalidInput = "INVALID_INPUT"

// newTestServerWithCapability builds a single-capability server whose capability
// name is provided by the caller. Used to exercise execute()'s default branches.
func newTestServerWithCapability(t *testing.T, capName, baseURL string) *adapter.AgentServer {
	t.Helper()
	cfg := &config.AdapterConfig{
		AgentID:          "git-test-helper",
		Name:             "Git Test Helper",
		Endpoint:         ":50061",
		RegistryEndpoint: "localhost:50052",
		Git:              config.GitConfig{Provider: "github", AuthEnv: "TEST_TOKEN"},
		Capabilities: []config.GitCapabilityConfig{
			{Name: capName, Owner: "test-owner", Repo: "test-repo"},
		},
	}
	return adapter.NewAgentServerWithURL(cfg, "fake-token", baseURL)
}

// ── execute() default branches ────────────────────────────────────────────────

// TestExecute_GitlabCapability covers the "gitlab" provider path in execute():
// the capability is registered in the router so execute is called, but the name
// "gitlab" is not yet implemented → INTERNAL "not implemented" error.
func TestExecute_GitlabCapability(t *testing.T) {
	t.Parallel()
	srv := newTestServerWithCapability(t, "gitlab", "")
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]string{"title": "x"})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-gitlab",
		CapabilityName: "gitlab",
		InputPayload:   payload,
	}
	_ = srv.ExecuteCapability(req, stream)
	if len(stream.events) == 0 {
		t.Fatal("expected at least one event")
	}
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Fatalf("expected FAILED, got %v", last.EventType)
	}
	if last.Error.Code != "INTERNAL" {
		t.Errorf("expected INTERNAL for gitlab provider, got %q", last.Error.Code)
	}
}

// TestExecute_UnknownCapabilityName covers the truly-unknown-capability branch
// in execute(): not "open_pr", "request_review", "get_diff", or "gitlab".
func TestExecute_UnknownCapabilityName(t *testing.T) {
	t.Parallel()
	srv := newTestServerWithCapability(t, "unknown-provider", "")
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]string{"foo": "bar"})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-unknown-prov",
		CapabilityName: "unknown-provider",
		InputPayload:   payload,
	}
	_ = srv.ExecuteCapability(req, stream)
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Fatalf("expected FAILED, got %v", last.EventType)
	}
	if last.Error.Code != codeInvalidInput {
		t.Errorf("expected INVALID_INPUT for unknown-provider, got %q", last.Error.Code)
	}
}

// ── parsePayload ──────────────────────────────────────────────────────────────

// TestParsePayload_EmptyPayload covers the `len(payload) == 0` branch.
func TestParsePayload_EmptyPayload(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	stream := &stubStream{ctx: context.Background()}
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-empty-payload",
		CapabilityName: "open_pr",
		InputPayload:   nil, // triggers "input payload is required"
	}
	if err := srv.ExecuteCapability(req, stream); err != nil {
		t.Fatalf("unexpected gRPC error: %v", err)
	}
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Fatalf("expected FAILED, got %v", last.EventType)
	}
	if last.Error.Code != codeInvalidInput {
		t.Errorf("expected INVALID_INPUT, got %q", last.Error.Code)
	}
}

// TestParsePayload_InvalidJSON covers the json.Unmarshal error branch.
func TestParsePayload_InvalidJSON(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	stream := &stubStream{ctx: context.Background()}
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-bad-json",
		CapabilityName: "open_pr",
		InputPayload:   []byte("{not valid json"),
	}
	if err := srv.ExecuteCapability(req, stream); err != nil {
		t.Fatalf("unexpected gRPC error: %v", err)
	}
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Fatalf("expected FAILED, got %v", last.EventType)
	}
	if last.Error.Code != codeInvalidInput {
		t.Errorf("expected INVALID_INPUT, got %q", last.Error.Code)
	}
}

// ── githubErrCode status mappings ────────────────────────────────────────────

// TestGithubErrCode_404 covers `case 404: return "NOT_FOUND"`.
func TestGithubErrCode_404(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
		})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	srv := newTestServer(t, ts.URL)
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]string{
		"title": "t", "head": "feat", "base": "main",
	})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-404",
		CapabilityName: "open_pr",
		InputPayload:   payload,
	}
	_ = srv.ExecuteCapability(req, stream)
	last := stream.last()
	if last.Error.Code != "NOT_FOUND" {
		t.Errorf("expected NOT_FOUND for HTTP 404, got %q", last.Error.Code)
	}
}

// TestGithubErrCode_422_And_Sanitise covers `case 422: return "INVALID_INPUT"` and
// the sanitise() truncation path (error message > 512 chars is truncated).
func TestGithubErrCode_422_And_Sanitise(t *testing.T) {
	t.Parallel()
	longMsg := strings.Repeat("x", 600) // 600 chars — exceeds maxErrMsgLen (512)
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity) // 422
			_ = json.NewEncoder(w).Encode(map[string]string{"message": longMsg})
		})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	srv := newTestServer(t, ts.URL)
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]string{
		"title": "t", "head": "feat", "base": "main",
	})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-422",
		CapabilityName: "open_pr",
		InputPayload:   payload,
	}
	_ = srv.ExecuteCapability(req, stream)
	last := stream.last()

	if last.Error.Code != codeInvalidInput {
		t.Errorf("expected INVALID_INPUT for HTTP 422, got %q", last.Error.Code)
	}
	// sanitise() must truncate the error message to ≤ 512 chars.
	if len(last.Error.Message) > 512 {
		t.Errorf("sanitise() did not truncate: len=%d (expected ≤512)", len(last.Error.Message))
	}
}

// TestGithubErrCode_5xx covers the switch fall-through to "UPSTREAM_ERROR"
// (HTTP 500 creates a *github.ErrorResponse but no matching case).
func TestGithubErrCode_5xx(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Internal Server Error"})
		})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	srv := newTestServer(t, ts.URL)
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]string{
		"title": "t", "head": "feat", "base": "main",
	})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-500",
		CapabilityName: "open_pr",
		InputPayload:   payload,
	}
	_ = srv.ExecuteCapability(req, stream)
	last := stream.last()
	if last.Error.Code != "UPSTREAM_ERROR" {
		t.Errorf("expected UPSTREAM_ERROR for HTTP 500, got %q", last.Error.Code)
	}
}

// TestGithubErrCode_ConnectionRefused covers the `errors.As` failure path
// (non-*github.ErrorResponse error → UPSTREAM_ERROR).
func TestGithubErrCode_ConnectionRefused(t *testing.T) {
	t.Parallel()
	// Start a server and close it immediately so the client gets connection refused.
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	ts.Close() // closed before use

	srv := newTestServer(t, ts.URL)
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]string{
		"title": "t", "head": "feat", "base": "main",
	})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-connrefused",
		CapabilityName: "open_pr",
		InputPayload:   payload,
	}
	_ = srv.ExecuteCapability(req, stream)
	last := stream.last()
	if last.Error.Code != "UPSTREAM_ERROR" {
		t.Errorf("expected UPSTREAM_ERROR for connection refused, got %q", last.Error.Code)
	}
}
