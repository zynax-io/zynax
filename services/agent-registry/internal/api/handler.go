// SPDX-License-Identifier: Apache-2.0

// Package api implements the agent-registry gRPC surface.
//
// CRD era (ADR-039, canvas 1571 O-step 7): the Agent custom resource is the
// single source of truth for agent identity and SchedulerService.SelectAgent
// is the dispatch surface. The push-era AgentRegistryService RPCs below are
// deprecated — each answers UNIMPLEMENTED with a migration pointer until
// their hard removal in M9 (deprecate-then-remove, ADR-039 Consequences).
package api

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// retiredMsg points every retired RPC at the migration guide.
const retiredMsg = "push registry retired (ADR-039) — agent identity lives in the zynax.io/v1alpha1 Agent custom resource and dispatch uses SchedulerService.SelectAgent; see docs/patterns/agent-crd-migration.md (hard removal: M9)"

// Handler implements the deprecated AgentRegistryServiceServer surface.
type Handler struct {
	zynaxv1.UnimplementedAgentRegistryServiceServer
}

// NewHandler constructs the CRD-era (retired) registry handler.
func NewHandler() *Handler { return &Handler{} }

func retired() error { return status.Error(codes.Unimplemented, retiredMsg) }

// RegisterAgent is retired: apply an Agent custom resource instead.
func (h *Handler) RegisterAgent(context.Context, *zynaxv1.RegisterAgentRequest) (*zynaxv1.RegisterAgentResponse, error) {
	return nil, retired()
}

// DeregisterAgent is retired: delete the Agent custom resource instead.
func (h *Handler) DeregisterAgent(context.Context, *zynaxv1.DeregisterAgentRequest) (*zynaxv1.DeregisterAgentResponse, error) {
	return nil, retired()
}

// GetAgent is retired: kubectl get agent <name> is the CRD-era read path.
func (h *Handler) GetAgent(context.Context, *zynaxv1.GetAgentRequest) (*zynaxv1.AgentDef, error) {
	return nil, retired()
}

// ListAgents is retired: kubectl get agents is the CRD-era read path.
func (h *Handler) ListAgents(context.Context, *zynaxv1.ListAgentsRequest) (*zynaxv1.ListAgentsResponse, error) {
	return nil, retired()
}

// FindByCapability is retired: dispatchers call SchedulerService.SelectAgent.
func (h *Handler) FindByCapability(context.Context, *zynaxv1.FindByCapabilityRequest) (*zynaxv1.FindByCapabilityResponse, error) {
	return nil, retired()
}
