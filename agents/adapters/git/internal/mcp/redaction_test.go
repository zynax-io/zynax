// SPDX-License-Identifier: Apache-2.0

package mcp_test

// Prompt-boundary redaction test (G.3 / #1199, type: security): the tool-result
// text the shim returns becomes model context, so the injected token must never
// appear in it — even if an upstream payload echoes the token back. The token is
// a PAT-shaped placeholder, never a real credential.

import (
	"context"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/mcp"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/redact"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

const fakeTokenMCP = "ghp_FAKE0000000000000000000000000000fake" //nolint:gosec // test fixture

// leakyCaps returns a COMPLETED payload that embeds the token, simulating an
// adapter path that did not redact, so the shim's prompt-boundary scrub is the
// last line of defense.
type leakyCaps struct{}

func (leakyCaps) ExecuteCapability(req *zynaxv1.ExecuteCapabilityRequest, stream zynaxv1.AgentService_ExecuteCapabilityServer) error {
	return stream.Send(&zynaxv1.TaskEvent{ //nolint:wrapcheck // test stub mirrors handler send semantics
		TaskId:    req.TaskId,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED,
		Payload:   []byte(`{"diff":"token=` + fakeTokenMCP + `"}`),
	})
}

func (leakyCaps) GetCapabilitySchema(_ context.Context, req *zynaxv1.GetCapabilitySchemaRequest) (*zynaxv1.GetCapabilitySchemaResponse, error) {
	return &zynaxv1.GetCapabilitySchemaResponse{CapabilityName: req.CapabilityName, InputSchemaJson: `{"type":"object"}`}, nil
}

func TestToolsCall_RedactsTokenInResult(t *testing.T) {
	t.Parallel()
	srv := mcp.NewServerWithRedactor(leakyCaps{}, []string{"get_diff"}, redact.New(fakeTokenMCP))
	resp := roundtrip(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_diff"}}`)
	result := resp["result"].(map[string]interface{})
	content := result["content"].([]interface{})
	text := content[0].(map[string]interface{})["text"].(string)
	if strings.Contains(text, fakeTokenMCP) {
		t.Fatalf("token leaked into MCP tool result (prompt content): %q", text)
	}
	if !strings.Contains(text, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in tool result, got %q", text)
	}
}
