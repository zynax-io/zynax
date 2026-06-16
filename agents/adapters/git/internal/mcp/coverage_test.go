// SPDX-License-Identifier: Apache-2.0

package mcp_test

// Coverage tests for the thin MCP shim (#1198, G.2): the gRPC ServerStream sink
// stubs, the JSON-RPC framing/error branches, and the allow-list dedup and
// tools/list schema edge cases. All exercised through the public Serve/NewServer
// surface — no Git logic lives in this layer, so none is asserted here.

import (
	"context"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/mcp"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// streamProbeCaps drives every AgentService_ExecuteCapabilityServer method the
// shim's eventSink implements, proving the stream stubs are safe no-ops before
// the terminal event is collapsed into the MCP tool result.
type streamProbeCaps struct{}

func (streamProbeCaps) ExecuteCapability(req *zynaxv1.ExecuteCapabilityRequest, stream zynaxv1.AgentService_ExecuteCapabilityServer) error {
	_ = stream.Context()
	_ = stream.SetHeader(nil)
	_ = stream.SendHeader(nil)
	stream.SetTrailer(nil)
	_ = stream.SendMsg(nil)
	_ = stream.RecvMsg(nil)
	return stream.Send(&zynaxv1.TaskEvent{ //nolint:wrapcheck // test stub mirrors handler send semantics
		TaskId:    req.TaskId,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED,
		Payload:   []byte(`{"ok":true}`),
	})
}

func (streamProbeCaps) GetCapabilitySchema(_ context.Context, req *zynaxv1.GetCapabilitySchemaRequest) (*zynaxv1.GetCapabilitySchemaResponse, error) {
	return &zynaxv1.GetCapabilitySchemaResponse{
		CapabilityName:  req.CapabilityName,
		InputSchemaJson: `{"type":"object"}`,
	}, nil
}

func TestToolsCall_StreamStubsAreNoOps(t *testing.T) {
	t.Parallel()
	// No "arguments" key ⇒ also covers the empty-args ⇒ "{}" default branch.
	srv := mcp.NewServer(streamProbeCaps{}, []string{"clone"})
	resp := roundtrip(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"clone"}}`)
	result := resp["result"].(map[string]interface{})
	if result["isError"].(bool) {
		t.Fatalf("expected success after exercising stream stubs, got %v", result)
	}
}

func TestDispatch_ParseError(t *testing.T) {
	t.Parallel()
	srv, _ := newFakeServer()
	resp := roundtrip(t, srv, `{not json`)
	errObj, ok := resp["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected JSON-RPC parse error, got %v", resp)
	}
	if errObj["code"].(float64) != -32700 {
		t.Errorf("expected parse error code -32700, got %v", errObj["code"])
	}
}

func TestDispatch_WrongJSONRPCVersion(t *testing.T) {
	t.Parallel()
	srv, _ := newFakeServer()
	resp := roundtrip(t, srv, `{"jsonrpc":"1.0","id":1,"method":"initialize"}`)
	if _, ok := resp["error"]; !ok {
		t.Fatalf("expected invalid-request error for bad jsonrpc version, got %v", resp)
	}
}

func TestDispatch_UnknownNotificationIsSilent(t *testing.T) {
	t.Parallel()
	srv, _ := newFakeServer()
	// No id ⇒ notification; unknown method ⇒ no response and no error.
	resp := roundtrip(t, srv, `{"jsonrpc":"2.0","method":"bogus/method"}`)
	if resp != nil {
		t.Errorf("unknown notification must be silent, got %v", resp)
	}
}

func TestToolsCall_InvalidParams(t *testing.T) {
	t.Parallel()
	srv, _ := newFakeServer()
	resp := roundtrip(t, srv, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":123}`)
	if _, ok := resp["error"]; !ok {
		t.Fatalf("expected invalid-params error, got %v", resp)
	}
}

func TestToolsCall_EmptyName(t *testing.T) {
	t.Parallel()
	srv, _ := newFakeServer()
	resp := roundtrip(t, srv, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":""}}`)
	if _, ok := resp["error"]; !ok {
		t.Fatalf("expected error for empty tool name, got %v", resp)
	}
}

func TestServe_SkipsBlankLines(t *testing.T) {
	t.Parallel()
	srv, _ := newFakeServer()
	var out strings.Builder
	in := "\n" + `{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n"
	if err := srv.Serve(context.Background(), strings.NewReader(in), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if !strings.Contains(out.String(), "protocolVersion") {
		t.Errorf("expected initialize result after a blank line, got %q", out.String())
	}
}

func TestNewServer_DedupesAndDropsEmpty(t *testing.T) {
	t.Parallel()
	caps := streamProbeCaps{}
	srv := mcp.NewServer(caps, []string{"clone", "clone", ""})
	resp := roundtrip(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	tools := resp["result"].(map[string]interface{})["tools"].([]interface{})
	if len(tools) != 1 {
		t.Fatalf("expected dedup + empty-drop to leave 1 tool, got %d", len(tools))
	}
}

// schemaQuirkCaps returns an error for "no_schema" (covering the skip branch) and
// an empty InputSchemaJson for "empty_schema" (covering the default "{}" branch).
type schemaQuirkCaps struct{}

func (schemaQuirkCaps) ExecuteCapability(*zynaxv1.ExecuteCapabilityRequest, zynaxv1.AgentService_ExecuteCapabilityServer) error {
	return nil
}

func (schemaQuirkCaps) GetCapabilitySchema(_ context.Context, req *zynaxv1.GetCapabilitySchemaRequest) (*zynaxv1.GetCapabilitySchemaResponse, error) {
	if req.CapabilityName == "no_schema" {
		return nil, status.Errorf(codes.NotFound, "no schema for %s", req.CapabilityName)
	}
	return &zynaxv1.GetCapabilitySchemaResponse{CapabilityName: req.CapabilityName, InputSchemaJson: ""}, nil
}

func TestToolsList_SkipsUnschemaedAndDefaultsEmpty(t *testing.T) {
	t.Parallel()
	srv := mcp.NewServer(schemaQuirkCaps{}, []string{"no_schema", "empty_schema"})
	resp := roundtrip(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	tools := resp["result"].(map[string]interface{})["tools"].([]interface{})
	if len(tools) != 1 {
		t.Fatalf("expected the unschemaed tool skipped, leaving 1, got %d", len(tools))
	}
}
