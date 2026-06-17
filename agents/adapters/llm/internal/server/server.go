// SPDX-License-Identifier: Apache-2.0

// Package server implements the AgentService gRPC contract for the llm-adapter.
// It owns the immutable CapabilityRouter built at startup and delegates request
// execution to the domain ChatCompletionHandler (canvas M7.P.4).
package server

import (
	"context"
	"fmt"

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/domain"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/provider"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AgentServer implements zynaxv1.AgentServiceServer. The router is built once
// from AdapterConfig and is immutable after construction (stateless adapter).
type AgentServer struct {
	zynaxv1.UnimplementedAgentServiceServer
	router *domain.CapabilityRouter
}

// NewAgentServer builds an AgentServer from a validated AdapterConfig and a
// pre-built Provider shared by every declared capability handler.
func NewAgentServer(cfg *config.AdapterConfig, p provider.Provider) (*AgentServer, error) {
	router, err := domain.NewRouter(cfg, p)
	if err != nil {
		return nil, fmt.Errorf("server: build router: %w", err)
	}
	return &AgentServer{router: router}, nil
}

// ExecuteCapability validates the request, resolves the capability handler, and
// streams TaskEvents. An empty task_id/capability_name is InvalidArgument; an
// unknown capability is NOT_FOUND with no TaskEvent emitted. Capability-level
// failures (validation, timeout, upstream) are delivered as a terminal FAILED
// event, not as a gRPC error.
func (s *AgentServer) ExecuteCapability(req *zynaxv1.ExecuteCapabilityRequest, stream zynaxv1.AgentService_ExecuteCapabilityServer) error {
	if req.GetTaskId() == "" {
		return status.Error(codes.InvalidArgument, "task_id is required")
	}
	if req.GetCapabilityName() == "" {
		return status.Error(codes.InvalidArgument, "capability_name is required")
	}
	handler, ok := s.router.Dispatch(req.GetCapabilityName())
	if !ok {
		return status.Errorf(codes.NotFound, "unknown capability: %s", req.GetCapabilityName())
	}
	return handler.Execute(req.GetTaskId(), req.GetInputPayload(), req.GetTimeoutSeconds(), stream) //nolint:wrapcheck // handler returns only a gRPC stream Send error, which already carries transport context
}

// GetCapabilitySchema returns the declared JSON Schemas for a named capability.
// An empty name is InvalidArgument; an unknown capability is NOT_FOUND.
func (s *AgentServer) GetCapabilitySchema(_ context.Context, req *zynaxv1.GetCapabilitySchemaRequest) (*zynaxv1.GetCapabilitySchemaResponse, error) {
	if req.GetCapabilityName() == "" {
		return nil, status.Error(codes.InvalidArgument, "capability_name is required")
	}
	description, inputSchema, outputSchema, ok := s.router.Schema(req.GetCapabilityName())
	if !ok {
		return nil, status.Errorf(codes.NotFound, "unknown capability: %s", req.GetCapabilityName())
	}
	return &zynaxv1.GetCapabilitySchemaResponse{
		CapabilityName:   req.GetCapabilityName(),
		InputSchemaJson:  inputSchema,
		OutputSchemaJson: outputSchema,
		Description:      description,
	}, nil
}
