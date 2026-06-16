// SPDX-License-Identifier: Apache-2.0

package mcp_test

// Unit tests for the thin MCP shim over git-adapter capabilities (#1198, G.2).
// These verify the 1:1 mapping into the adapter handlers, the explicit tool
// allow-list, and the JSON-RPC stdio framing — no Git logic lives in this layer.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/mcp"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeCaps records dispatches and returns canned terminal events. It stands in
// for *adapter.AgentServer so the test asserts the shim translates 1:1 without
// touching real Git logic.
type fakeCaps struct {
	known      map[string]string // capability name → output payload JSON
	calledWith []string          // capability names dispatched, in order
}

func (f *fakeCaps) ExecuteCapability(req *zynaxv1.ExecuteCapabilityRequest, stream zynaxv1.AgentService_ExecuteCapabilityServer) error {
	f.calledWith = append(f.calledWith, req.CapabilityName)
	payload, ok := f.known[req.CapabilityName]
	if !ok {
		return stream.Send(&zynaxv1.TaskEvent{ //nolint:wrapcheck // test stub mirrors handler send semantics
			TaskId:    req.TaskId,
			EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED,
			Error:     &zynaxv1.CapabilityError{Code: "INVALID_INPUT", Message: "unknown capability"},
		})
	}
	// Emit a PROGRESS first to prove the sink drops it.
	_ = stream.Send(&zynaxv1.TaskEvent{TaskId: req.TaskId, EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_PROGRESS})
	return stream.Send(&zynaxv1.TaskEvent{ //nolint:wrapcheck // test stub mirrors handler send semantics
		TaskId:    req.TaskId,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED,
		Payload:   []byte(payload),
	})
}

func (f *fakeCaps) GetCapabilitySchema(_ context.Context, req *zynaxv1.GetCapabilitySchemaRequest) (*zynaxv1.GetCapabilitySchemaResponse, error) {
	if _, ok := f.known[req.CapabilityName]; !ok {
		return nil, status.Errorf(codes.NotFound, "unknown capability: %s", req.CapabilityName)
	}
	return &zynaxv1.GetCapabilitySchemaResponse{
		CapabilityName:  req.CapabilityName,
		InputSchemaJson: `{"type":"object"}`,
		Description:     "desc for " + req.CapabilityName,
	}, nil
}

// roundtrip feeds one JSON-RPC line through Serve and returns the decoded response.
func roundtrip(t *testing.T, srv *mcp.Server, line string) map[string]interface{} {
	t.Helper()
	var out strings.Builder
	if err := srv.Serve(context.Background(), strings.NewReader(line+"\n"), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if out.Len() == 0 {
		return nil
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(out.String()), &resp); err != nil {
		t.Fatalf("decode response %q: %v", out.String(), err)
	}
	return resp
}

func newFakeServer() (*mcp.Server, *fakeCaps) {
	caps := &fakeCaps{known: map[string]string{
		"open_pr":        `{"pr_url":"https://example/pr/1","pr_number":1}`,
		"request_review": `{"requested":true}`,
		"get_diff":       `{"diff":"--- a","truncated":false}`,
	}}
	srv := mcp.NewServer(caps, []string{"open_pr", "request_review", "get_diff"})
	return srv, caps
}

func TestInitialize(t *testing.T) {
	t.Parallel()
	srv, _ := newFakeServer()
	resp := roundtrip(t, srv, `{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("no result: %v", resp)
	}
	if result["protocolVersion"] == "" {
		t.Error("expected protocolVersion")
	}
}

func TestToolsList_AllowListOnly(t *testing.T) {
	t.Parallel()
	srv, _ := newFakeServer()
	resp := roundtrip(t, srv, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	result := resp["result"].(map[string]interface{})
	tools := result["tools"].([]interface{})
	if len(tools) != 3 {
		t.Fatalf("expected 3 allow-listed tools, got %d", len(tools))
	}
	names := map[string]bool{}
	for _, tl := range tools {
		names[tl.(map[string]interface{})["name"].(string)] = true
	}
	for _, want := range []string{"open_pr", "request_review", "get_diff"} {
		if !names[want] {
			t.Errorf("tool %q missing from list", want)
		}
	}
}

func TestToolsCall_DispatchesOneToOne(t *testing.T) {
	t.Parallel()
	srv, caps := newFakeServer()
	resp := roundtrip(t, srv,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"open_pr","arguments":{"title":"x","head":"f","base":"main"}}}`)
	result := resp["result"].(map[string]interface{})
	if result["isError"].(bool) {
		t.Fatalf("expected success, got %v", result)
	}
	// 1:1: exactly the named capability was dispatched into the adapter.
	if len(caps.calledWith) != 1 || caps.calledWith[0] != "open_pr" {
		t.Fatalf("expected single open_pr dispatch, got %v", caps.calledWith)
	}
	content := result["content"].([]interface{})
	text := content[0].(map[string]interface{})["text"].(string)
	if !strings.Contains(text, "pr_url") {
		t.Errorf("expected adapter payload verbatim, got %q", text)
	}
}

func TestToolsCall_RejectsUnknownTool(t *testing.T) {
	t.Parallel()
	srv, caps := newFakeServer()
	resp := roundtrip(t, srv,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"rm_minus_rf"}}`)
	if _, ok := resp["error"]; !ok {
		t.Fatalf("expected JSON-RPC error for unknown tool, got %v", resp)
	}
	// Allow-list enforced before any adapter dispatch.
	if len(caps.calledWith) != 0 {
		t.Errorf("unknown tool must not reach the adapter, got %v", caps.calledWith)
	}
}

func TestToolsCall_FailedCapabilityIsToolError(t *testing.T) {
	t.Parallel()
	// "get_diff" is allow-listed but we drop it from known so the adapter fails.
	caps := &fakeCaps{known: map[string]string{}}
	srv := mcp.NewServer(caps, []string{"get_diff"})
	resp := roundtrip(t, srv,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_diff","arguments":{"pr_number":1}}}`)
	result := resp["result"].(map[string]interface{})
	if !result["isError"].(bool) {
		t.Fatalf("expected isError true for failed capability, got %v", result)
	}
}

func TestNotification_NoResponse(t *testing.T) {
	t.Parallel()
	srv, _ := newFakeServer()
	resp := roundtrip(t, srv, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if resp != nil {
		t.Errorf("notification must not produce a response, got %v", resp)
	}
}

func TestUnknownMethod(t *testing.T) {
	t.Parallel()
	srv, _ := newFakeServer()
	resp := roundtrip(t, srv, `{"jsonrpc":"2.0","id":9,"method":"resources/list"}`)
	if _, ok := resp["error"]; !ok {
		t.Errorf("expected method-not-found error, got %v", resp)
	}
}
