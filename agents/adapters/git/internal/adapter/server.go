// SPDX-License-Identifier: Apache-2.0

// Package adapter implements the AgentService gRPC contract for the git-adapter.
package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/credential"
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

// NewAgentServer builds an AgentServer from a validated AdapterConfig and a static
// GitHub token (classic / fine-grained PAT). The token is redacted from every
// caller-visible error and payload (G.3 / #1199). For refreshable App credentials
// use NewAgentServerWithSource (G.7 / #1262).
func NewAgentServer(cfg *config.AdapterConfig, token string) *AgentServer {
	return newAgentServer(cfg, newGitHandler(token))
}

// NewAgentServerWithSource builds an AgentServer backed by a refreshable
// credential.Source — used for GitHub App installation tokens that expire (~1 h)
// and must refresh without a process restart (G.7 / #1262). redactSeed is the
// initial token value (if known) used to seed the egress redactor; pass "" when
// the first token is minted lazily. The PAT path uses NewAgentServer unchanged.
func NewAgentServerWithSource(cfg *config.AdapterConfig, src credential.Source, redactSeed string) *AgentServer {
	return newAgentServer(cfg, newGitHandlerWithSource(src, redactSeed))
}

// NewAgentServerWithURL builds an AgentServer using a custom GitHub API base URL.
// Intended for testing against httptest.Server.
func NewAgentServerWithURL(cfg *config.AdapterConfig, token, baseURL string) *AgentServer {
	return newAgentServer(cfg, newGitHandlerWithURL(token, baseURL))
}

// NewAgentServerWithSourceURL builds a source-backed AgentServer against a custom
// GitHub API base URL. Intended for testing the refreshable-credential path
// against an httptest.Server.
func NewAgentServerWithSourceURL(cfg *config.AdapterConfig, src credential.Source, baseURL string) *AgentServer {
	return newAgentServer(cfg, newGitHandlerFromSourceWithURL(src, baseURL))
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

// newGitHubClient builds a GitHub client authenticated by a static token.
func newGitHubClient(token string) *github.Client {
	return github.NewClient(nil).WithAuthToken(token)
}

// newGitHubClientFromSource builds a GitHub client whose Authorization header is
// resolved per request from a refreshable credential.Source (G.7 / #1262). When
// the source re-mints a token, subsequent requests carry the new value with no
// client rebuild.
func newGitHubClientFromSource(src credential.Source) *github.Client {
	httpClient := &http.Client{Transport: credential.NewTransport(src, nil)}
	return github.NewClient(httpClient)
}

// marshalPayload marshals v to JSON, returning an "UPSTREAM_ERROR" terminal event on failure.
func marshalPayload(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return b, nil
}
