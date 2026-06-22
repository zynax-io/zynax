// SPDX-License-Identifier: Apache-2.0

// Package adapter implements the AgentService gRPC contract for the adk-adapter.
//
// This is the S2 skeleton (#1478): GetCapabilitySchema is fully wired, while
// ExecuteCapability validates the request and routes on capability_name but
// emits a single terminal FAILED event with code "UNIMPLEMENTED" — the ADK
// Runner -> TaskEvent bridge lands in S3 (#1479). The AgentService invariant
// (exactly one terminal TaskEvent, echoing task_id) already holds here.
package adapter

import (
	"context"
	"fmt"

	"github.com/zynax-io/zynax/agents/adapters/adk/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// errBridgeNotWired is emitted by the skeleton until the ADK Runner bridge lands
// in S3 (#1479).
const errBridgeNotWired = "adk capability bridge not yet wired (S3, #1479)"

// AgentServer implements AgentServiceServer. The capability router is built once
// at construction from the validated config and is immutable thereafter.
type AgentServer struct {
	zynaxv1.UnimplementedAgentServiceServer
	router map[string]config.CapabilityConfig
}

// NewAgentServer builds an AgentServer from a validated AdapterConfig.
func NewAgentServer(cfg *config.AdapterConfig) *AgentServer {
	router := make(map[string]config.CapabilityConfig, len(cfg.Capabilities))
	for _, c := range cfg.Capabilities {
		router[c.Name] = c
	}
	return &AgentServer{router: router}
}

// ExecuteCapability validates the request, routes on capability_name, and (until
// S3) emits a single terminal FAILED event. An unknown capability yields
// "INVALID_INPUT"; a known one yields "UNIMPLEMENTED" until the bridge is wired.
func (s *AgentServer) ExecuteCapability(req *zynaxv1.ExecuteCapabilityRequest, stream zynaxv1.AgentService_ExecuteCapabilityServer) error {
	if req.TaskId == "" {
		return status.Error(codes.InvalidArgument, "task_id is required")
	}
	if req.CapabilityName == "" {
		return status.Error(codes.InvalidArgument, "capability_name is required")
	}
	if _, ok := s.router[req.CapabilityName]; !ok {
		return sendFailed(stream, req.TaskId, "INVALID_INPUT",
			fmt.Sprintf("unknown capability: %s", req.CapabilityName))
	}
	// S3 (#1479) replaces this with the ADK Runner -> TaskEvent bridge.
	return sendFailed(stream, req.TaskId, "UNIMPLEMENTED", errBridgeNotWired)
}

// GetCapabilitySchema returns the JSON Schema for a named capability.
func (s *AgentServer) GetCapabilitySchema(_ context.Context, req *zynaxv1.GetCapabilitySchemaRequest) (*zynaxv1.GetCapabilitySchemaResponse, error) {
	if req.CapabilityName == "" {
		return nil, status.Error(codes.InvalidArgument, "capability_name is required")
	}
	c, ok := s.router[req.CapabilityName]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "unknown capability: %s", req.CapabilityName)
	}
	return &zynaxv1.GetCapabilitySchemaResponse{
		CapabilityName:   c.Name,
		InputSchemaJson:  c.InputSchemaJSON,
		OutputSchemaJson: c.OutputSchemaJSON,
		Description:      c.Description,
	}, nil
}

// sendFailed emits exactly one terminal FAILED TaskEvent echoing task_id.
func sendFailed(stream zynaxv1.AgentService_ExecuteCapabilityServer, taskID, code, msg string) error {
	return stream.Send(&zynaxv1.TaskEvent{ //nolint:wrapcheck // direct stream send; gRPC error surfaced as-is
		TaskId:    taskID,
		EventType: zynaxv1.TaskEventType_TASK_EVENT_TYPE_FAILED,
		Timestamp: timestamppb.Now(),
		Error:     &zynaxv1.CapabilityError{Code: code, Message: msg},
	})
}
