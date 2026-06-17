// SPDX-License-Identifier: Apache-2.0

// Package adapter implements the AgentService gRPC contract for the git-adapter.
package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AgentServer implements AgentServiceServer using go-github for all operations.
// The CapabilityRouter is built once at construction and is immutable after that.
type AgentServer struct {
	zynaxv1.UnimplementedAgentServiceServer
	router  map[string]config.GitCapabilityConfig
	handler *gitHandler
}

// NewAgentServer builds an AgentServer from a validated AdapterConfig and a GitHub token.
// The token is redacted from every caller-visible error and payload (G.3 / #1199).
func NewAgentServer(cfg *config.AdapterConfig, token string) *AgentServer {
	return newAgentServer(cfg, newGitHandler(token))
}

// NewAgentServerWithURL builds an AgentServer using a custom GitHub API base URL.
// Intended for testing against httptest.Server.
func NewAgentServerWithURL(cfg *config.AdapterConfig, token, baseURL string) *AgentServer {
	return newAgentServer(cfg, newGitHandlerWithURL(token, baseURL))
}

func newAgentServer(cfg *config.AdapterConfig, h *gitHandler) *AgentServer {
	router := make(map[string]config.GitCapabilityConfig, len(cfg.Capabilities))
	for _, c := range cfg.Capabilities {
		router[c.Name] = c
	}
	return &AgentServer{router: router, handler: h}
}

// ExecuteCapability validates the request and dispatches to the capability handler.
func (s *AgentServer) ExecuteCapability(req *zynaxv1.ExecuteCapabilityRequest, stream zynaxv1.AgentService_ExecuteCapabilityServer) error {
	if req.TaskId == "" {
		return status.Error(codes.InvalidArgument, "task_id is required")
	}
	if req.CapabilityName == "" {
		return status.Error(codes.InvalidArgument, "capability_name is required")
	}

	gcap, ok := s.router[req.CapabilityName]
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

	return s.handler.execute(ctx, gcap, req.TaskId, req.InputPayload, stream)
}

// GetCapabilitySchema returns the JSON Schema for a named capability.
func (s *AgentServer) GetCapabilitySchema(_ context.Context, req *zynaxv1.GetCapabilitySchemaRequest) (*zynaxv1.GetCapabilitySchemaResponse, error) {
	if req.CapabilityName == "" {
		return nil, status.Error(codes.InvalidArgument, "capability_name is required")
	}
	gcap, ok := s.router[req.CapabilityName]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "unknown capability: %s", req.CapabilityName)
	}
	return &zynaxv1.GetCapabilitySchemaResponse{
		CapabilityName:   gcap.Name,
		InputSchemaJson:  gcap.InputSchemaJSON,
		OutputSchemaJson: gcap.OutputSchemaJSON,
		Description:      gcap.Description,
	}, nil
}

// newGitHubClient builds an authenticated GitHub client.
func newGitHubClient(token string) *github.Client {
	return github.NewClient(nil).WithAuthToken(token)
}

// marshalPayload marshals v to JSON, returning an "UPSTREAM_ERROR" terminal event on failure.
func marshalPayload(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return b, nil
}
