// SPDX-License-Identifier: Apache-2.0

// Package adapter implements the AgentService gRPC contract for the ci-adapter.
// It wraps the GitHub Actions REST API and surfaces trigger_workflow and
// get_run_status as Zynax capabilities.
package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/ci/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AgentServer implements AgentServiceServer using the GitHub Actions REST API.
// The CapabilityRouter is built once at construction and is immutable afterwards.
type AgentServer struct {
	zynaxv1.UnimplementedAgentServiceServer
	router  map[string]config.CICapabilityConfig
	handler *ciHandler
}

// NewAgentServer builds an AgentServer from a validated AdapterConfig and a resolved auth token.
func NewAgentServer(cfg *config.AdapterConfig, token string) *AgentServer {
	return newAgentServer(cfg, newCIHandler(token, &cfg.CI))
}

// NewAgentServerWithURL builds an AgentServer using a custom GitHub API base URL.
// Intended for testing against httptest.Server.
func NewAgentServerWithURL(cfg *config.AdapterConfig, token, baseURL string) *AgentServer {
	return newAgentServer(cfg, newCIHandlerWithURL(token, &cfg.CI, baseURL))
}

func newAgentServer(cfg *config.AdapterConfig, h *ciHandler) *AgentServer {
	router := make(map[string]config.CICapabilityConfig, len(cfg.Capabilities))
	for _, c := range cfg.Capabilities {
		router[c.Name] = c
	}
	return &AgentServer{router: router, handler: h}
}

// ExecuteCapability validates the request, applies timeout, and dispatches to the handler.
func (s *AgentServer) ExecuteCapability(req *zynaxv1.ExecuteCapabilityRequest, stream zynaxv1.AgentService_ExecuteCapabilityServer) error {
	if req.TaskId == "" {
		return status.Error(codes.InvalidArgument, "task_id is required")
	}
	if req.CapabilityName == "" {
		return status.Error(codes.InvalidArgument, "capability_name is required")
	}

	ccap, ok := s.router[req.CapabilityName]
	if !ok {
		return sendFailed(stream, req.TaskId, "INVALID_INPUT",
			fmt.Sprintf("unknown capability: %s", req.CapabilityName))
	}

	ctx := stream.Context()
	if req.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	return s.handler.execute(ctx, ccap, req.TaskId, req.InputPayload, stream)
}

// GetCapabilitySchema returns the JSON Schema for a named capability.
func (s *AgentServer) GetCapabilitySchema(_ context.Context, req *zynaxv1.GetCapabilitySchemaRequest) (*zynaxv1.GetCapabilitySchemaResponse, error) {
	if req.CapabilityName == "" {
		return nil, status.Error(codes.InvalidArgument, "capability_name is required")
	}
	ccap, ok := s.router[req.CapabilityName]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "unknown capability: %s", req.CapabilityName)
	}
	return &zynaxv1.GetCapabilitySchemaResponse{
		CapabilityName:   ccap.Name,
		InputSchemaJson:  ccap.InputSchemaJSON,
		OutputSchemaJson: ccap.OutputSchemaJSON,
		Description:      ccap.Description,
	}, nil
}
