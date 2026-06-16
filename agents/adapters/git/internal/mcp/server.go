// SPDX-License-Identifier: Apache-2.0

// Package mcp is a thin Model Context Protocol (MCP) shim over the existing
// git-adapter capability handlers. It exposes the adapter's capabilities as MCP
// tools over a JSON-RPC 2.0 stdio transport so an authoring agent (e.g. Claude
// Code) can drive Git operations interactively.
//
// The shim contains NO Git logic. Every tool call is translated 1:1 into the
// adapter's existing ExecuteCapability / GetCapabilitySchema handlers (ADR-032:
// one Git implementation, two surfaces). The set of exposed tools is an explicit
// allow-list built from the configured capabilities — never "every handler".
//
// Credential injection and redaction are out of scope here; they are delivered
// by G.3 (#1199). This layer never accepts a token as a tool argument and never
// derives owner/repo from input (the adapter already pins those in config —
// SSRF prevention), so no caller-supplied value reaches a privileged Git call as
// a flag or remote.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// protocolVersion is the MCP revision this shim implements.
const protocolVersion = "2024-11-05"

// jsonRPCVersion is the JSON-RPC protocol version every request/response carries.
const jsonRPCVersion = "2.0"

// Capabilities is the minimal surface the shim needs from the git-adapter.
// *adapter.AgentServer satisfies it, so the MCP layer reuses the exact same
// handlers as the runtime gRPC path — no Git logic is duplicated here.
type Capabilities interface {
	ExecuteCapability(req *zynaxv1.ExecuteCapabilityRequest, stream zynaxv1.AgentService_ExecuteCapabilityServer) error
	GetCapabilitySchema(ctx context.Context, req *zynaxv1.GetCapabilitySchemaRequest) (*zynaxv1.GetCapabilitySchemaResponse, error)
}

// Server is a stdio JSON-RPC 2.0 MCP server bound to a fixed allow-list of tools.
type Server struct {
	caps  Capabilities
	tools []string // explicit allow-list; nothing outside this is exposed
}

// NewServer builds a shim that exposes exactly the named capabilities as MCP
// tools. toolNames is the allow-list (typically the configured capability names);
// a tool absent from this slice is never advertised nor dispatchable.
func NewServer(caps Capabilities, toolNames []string) *Server {
	allow := make([]string, 0, len(toolNames))
	seen := make(map[string]struct{}, len(toolNames))
	for _, n := range toolNames {
		if n == "" {
			continue
		}
		if _, dup := seen[n]; dup {
			continue
		}
		seen[n] = struct{}{}
		allow = append(allow, n)
	}
	return &Server{caps: caps, tools: allow}
}

// allowed reports whether name is in the explicit tool allow-list.
func (s *Server) allowed(name string) bool {
	for _, t := range s.tools {
		if t == name {
			return true
		}
	}
	return false
}

// ── JSON-RPC 2.0 envelope ──────────────────────────────────────────────────────

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// JSON-RPC standard error codes used by this shim.
const (
	errParse          = -32700
	errInvalidRequest = -32600
	errMethodNotFound = -32601
	errInvalidParams  = -32602
)

// Serve reads newline-delimited JSON-RPC requests from r and writes responses to
// w until r is exhausted (the transport closes). It is the stdio MCP loop.
func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	enc := json.NewEncoder(w)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		resp := s.dispatch(ctx, line)
		if resp == nil {
			continue // notification — no response
		}
		if err := enc.Encode(resp); err != nil {
			return fmt.Errorf("mcp: encode response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("mcp: read transport: %w", err)
	}
	return nil
}

// dispatch parses one line and routes it. Returns nil for notifications.
func (s *Server) dispatch(ctx context.Context, line []byte) *rpcResponse {
	var req rpcRequest
	if err := json.Unmarshal(line, &req); err != nil {
		return errResponse(nil, errParse, "parse error")
	}
	if req.JSONRPC != jsonRPCVersion {
		return errResponse(req.ID, errInvalidRequest, "jsonrpc must be \"2.0\"")
	}
	// A request without an id is a notification: handle, but never respond.
	notification := len(req.ID) == 0

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req.ID)
	case "tools/list":
		return s.handleToolsList(ctx, req.ID)
	case "tools/call":
		return s.handleToolsCall(ctx, req.ID, req.Params)
	case "notifications/initialized", "initialized":
		return nil // client handshake notification
	default:
		if notification {
			return nil
		}
		return errResponse(req.ID, errMethodNotFound, "unknown method: "+req.Method)
	}
}

func (s *Server) handleInitialize(id json.RawMessage) *rpcResponse {
	return &rpcResponse{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Result: map[string]interface{}{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
			"serverInfo":      map[string]interface{}{"name": "git-adapter-mcp", "version": "0.1.0"},
		},
	}
}

// toolDescriptor is one entry in the tools/list result.
type toolDescriptor struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

func (s *Server) handleToolsList(ctx context.Context, id json.RawMessage) *rpcResponse {
	tools := make([]toolDescriptor, 0, len(s.tools))
	for _, name := range s.tools {
		schema, err := s.caps.GetCapabilitySchema(ctx,
			&zynaxv1.GetCapabilitySchemaRequest{CapabilityName: name})
		if err != nil {
			// A configured tool with no schema is a config defect, not a caller
			// error; skip it rather than expose an un-described tool.
			continue
		}
		in := schema.GetInputSchemaJson()
		if in == "" {
			in = "{}"
		}
		tools = append(tools, toolDescriptor{
			Name:        schema.GetCapabilityName(),
			Description: schema.GetDescription(),
			InputSchema: json.RawMessage(in),
		})
	}
	return &rpcResponse{JSONRPC: jsonRPCVersion, ID: id, Result: map[string]interface{}{"tools": tools}}
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

func (s *Server) handleToolsCall(ctx context.Context, id, params json.RawMessage) *rpcResponse {
	var p toolCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return errResponse(id, errInvalidParams, "invalid params: "+err.Error())
	}
	if p.Name == "" {
		return errResponse(id, errInvalidParams, "tool name is required")
	}
	// Explicit allow-list: a tool outside the configured set is never dispatched,
	// regardless of whether the adapter would recognise it.
	if !s.allowed(p.Name) {
		return errResponse(id, errMethodNotFound, "unknown tool: "+p.Name)
	}

	args := []byte(p.Arguments)
	if len(args) == 0 {
		args = []byte("{}")
	}

	sink := newEventSink(ctx)
	// 1:1 dispatch into the adapter's existing handler. The owner/repo target is
	// fixed in adapter config — never taken from these arguments — so no
	// caller-supplied remote or path reaches the privileged Git call.
	err := s.caps.ExecuteCapability(&zynaxv1.ExecuteCapabilityRequest{
		TaskId:         "mcp-" + p.Name,
		CapabilityName: p.Name,
		InputPayload:   args,
	}, sink)
	if err != nil {
		return errResponse(id, errInvalidRequest, "tool execution failed: "+err.Error())
	}

	text, isError := sink.result()
	return &rpcResponse{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{{"type": "text", "text": text}},
			"isError": isError,
		},
	}
}

func errResponse(id json.RawMessage, code int, msg string) *rpcResponse {
	return &rpcResponse{JSONRPC: jsonRPCVersion, ID: id, Error: &rpcError{Code: code, Message: msg}}
}
