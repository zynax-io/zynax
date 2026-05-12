// SPDX-License-Identifier: Apache-2.0

// Package adapter implements the AgentService gRPC contract for the http-adapter.
package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/zynax-io/zynax/agents/adapters/http/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AgentServer implements AgentServiceServer.
// The CapabilityRouter is built once from AdapterConfig and is immutable after construction.
type AgentServer struct {
	zynaxv1.UnimplementedAgentServiceServer
	router  map[string]config.RouteConfig
	handler *httpHandler
}

// NewAgentServer builds an AgentServer from a validated AdapterConfig.
func NewAgentServer(cfg *config.AdapterConfig) *AgentServer {
	router := make(map[string]config.RouteConfig, len(cfg.Capabilities))
	for _, c := range cfg.Capabilities {
		router[c.Name] = c
	}
	return &AgentServer{router: router, handler: newHTTPHandler()}
}

// ExecuteCapability validates the request, resolves the route, and proxies to the upstream.
func (s *AgentServer) ExecuteCapability(req *zynaxv1.ExecuteCapabilityRequest, stream zynaxv1.AgentService_ExecuteCapabilityServer) error {
	if req.TaskId == "" {
		return status.Error(codes.InvalidArgument, "task_id is required")
	}
	if req.CapabilityName == "" {
		return status.Error(codes.InvalidArgument, "capability_name is required")
	}

	route, ok := s.router[req.CapabilityName]
	if !ok {
		return sendFailed(stream, req.TaskId, "INVALID_INPUT",
			fmt.Sprintf("unknown capability: %s", req.CapabilityName))
	}

	if route.InputSchemaJSON != "" && len(req.InputPayload) > 0 {
		if err := validatePayload(route.InputSchemaJSON, req.InputPayload); err != nil {
			return sendFailed(stream, req.TaskId, "INVALID_INPUT", sanitise(err.Error()))
		}
	}

	ctx := stream.Context()
	if req.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	return s.handler.execute(ctx, route, req.TaskId, req.InputPayload, stream)
}

// GetCapabilitySchema returns the JSON Schema for a named capability from the router.
func (s *AgentServer) GetCapabilitySchema(_ context.Context, req *zynaxv1.GetCapabilitySchemaRequest) (*zynaxv1.GetCapabilitySchemaResponse, error) {
	if req.CapabilityName == "" {
		return nil, status.Error(codes.InvalidArgument, "capability_name is required")
	}
	route, ok := s.router[req.CapabilityName]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "unknown capability: %s", req.CapabilityName)
	}
	return &zynaxv1.GetCapabilitySchemaResponse{
		CapabilityName:   route.Name,
		InputSchemaJson:  route.InputSchemaJSON,
		OutputSchemaJson: route.OutputSchemaJSON,
		Description:      route.Description,
	}, nil
}

func validatePayload(schemaJSON string, payload []byte) error {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", strings.NewReader(schemaJSON)); err != nil {
		return fmt.Errorf("invalid schema configuration: %w", err)
	}
	sch, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("invalid schema configuration: %w", err)
	}
	var v interface{}
	if err := json.Unmarshal(payload, &v); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	return sch.Validate(v)
}
