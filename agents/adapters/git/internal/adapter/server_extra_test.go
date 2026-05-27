// SPDX-License-Identifier: Apache-2.0

package adapter_test

// Additional server coverage — NewAgentServer, ExecuteCapability with a
// non-zero TimeoutSeconds, and GetCapabilitySchema empty-name guard.
// Closes #717 — part of the git-adapter coverage epic (#713).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/adapter"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestNewAgentServer_Direct calls the production constructor (not the test-URL
// variant) to verify it returns a non-nil server. This is the only test that
// exercises NewAgentServer's body.
func TestNewAgentServer_Direct(t *testing.T) {
	t.Parallel()
	cfg := &config.AdapterConfig{
		AgentID: "git-test",
		Name:    "Git Test",
		Capabilities: []config.GitCapabilityConfig{
			{Name: "open_pr", Owner: "o", Repo: "r"},
		},
	}
	srv := adapter.NewAgentServer(cfg, "fake-token")
	if srv == nil {
		t.Fatal("expected non-nil *AgentServer from NewAgentServer")
	}
}

// TestExecuteCapability_WithTimeout verifies that a non-zero TimeoutSeconds in
// the request wraps the stream context with a deadline and the capability still
// completes successfully before the timeout expires.
func TestExecuteCapability_WithTimeout(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"number":   99,
			"html_url": "https://github.com/test-owner/test-repo/pull/99",
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	srv := newTestServer(t, ts.URL)
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]string{
		"title": "Timeout test PR",
		"head":  "feat-branch",
		"base":  "main",
	})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-with-timeout",
		CapabilityName: "open_pr",
		InputPayload:   payload,
		TimeoutSeconds: 10, // exercises the req.TimeoutSeconds > 0 branch
	}
	err := srv.ExecuteCapability(req, stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Errorf("expected COMPLETED, got %v", last.EventType)
	}
}

// TestGetCapabilitySchema_EmptyName verifies that an empty capability_name in
// GetCapabilitySchema returns InvalidArgument (the early-return guard on line 74).
func TestGetCapabilitySchema_EmptyName(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	_, err := srv.GetCapabilitySchema(context.Background(),
		&zynaxv1.GetCapabilitySchemaRequest{CapabilityName: ""})
	if err == nil {
		t.Fatal("expected error for empty capability_name")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}
}

// TestExecuteCapability_MissingCapabilityName verifies the second guard in
// ExecuteCapability: an empty capability_name returns InvalidArgument.
func TestExecuteCapability_MissingCapabilityName(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	stream := &stubStream{ctx: context.Background()}
	req := &zynaxv1.ExecuteCapabilityRequest{TaskId: "t-no-cap"}
	err := srv.ExecuteCapability(req, stream)
	if err == nil {
		t.Fatal("expected gRPC error for missing capability_name")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}
}

// TestExecuteCapability_OpenPR_MissingFields verifies that open_pr returns
// INVALID_INPUT when the payload is valid JSON but missing required fields
// (title, head, or base).
func TestExecuteCapability_OpenPR_MissingFields(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	stream := &stubStream{ctx: context.Background()}
	// Valid JSON but empty title → triggers "title, head, and base are required".
	payload, _ := json.Marshal(map[string]string{"head": "feat", "base": "main"})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-missing-title",
		CapabilityName: "open_pr",
		InputPayload:   payload,
	}
	err := srv.ExecuteCapability(req, stream)
	if err != nil {
		t.Fatalf("unexpected gRPC error: %v", err)
	}
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Errorf("expected FAILED, got %v", last.EventType)
	}
	if last.Error.Code != "INVALID_INPUT" {
		t.Errorf("expected INVALID_INPUT, got %q", last.Error.Code)
	}
}

// TestExecuteCapability_GetDiff_InvalidPRNumber verifies that get_diff returns
// INVALID_INPUT when pr_number is zero (the boundary guard in getDiff).
func TestExecuteCapability_GetDiff_InvalidPRNumber(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, "")
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]int{"pr_number": 0})
	req := &zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "t-diff-zero",
		CapabilityName: "get_diff",
		InputPayload:   payload,
	}
	err := srv.ExecuteCapability(req, stream)
	if err != nil {
		t.Fatalf("unexpected gRPC error: %v", err)
	}
	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED {
		t.Errorf("expected FAILED, got %v", last.EventType)
	}
	if last.Error.Code != "INVALID_INPUT" {
		t.Errorf("expected INVALID_INPUT, got %q", last.Error.Code)
	}
}
