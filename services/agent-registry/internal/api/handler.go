// SPDX-License-Identifier: Apache-2.0

// Package api implements the AgentRegistryService gRPC server handler.
package api

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/agent-registry/internal/domain"
)

// Handler implements AgentRegistryServiceServer, delegating to domain.AgentRegistryService.
type Handler struct {
	zynaxv1.UnimplementedAgentRegistryServiceServer
	svc *domain.AgentRegistryService
}

// NewHandler constructs a Handler wrapping the given AgentRegistryService.
func NewHandler(svc *domain.AgentRegistryService) *Handler { return &Handler{svc: svc} }

// RegisterAgent records an agent and its capabilities.
func (h *Handler) RegisterAgent(ctx context.Context, req *zynaxv1.RegisterAgentRequest) (*zynaxv1.RegisterAgentResponse, error) {
	registered, err := h.svc.Register(ctx, protoToDomain(req.GetAgent()))
	if err != nil {
		return nil, grpcErr(err)
	}
	return &zynaxv1.RegisterAgentResponse{
		AgentId:      registered.ID,
		RegisteredAt: timestamppb.New(registered.RegisteredAt),
	}, nil
}

// DeregisterAgent marks the agent as DEREGISTERED.
func (h *Handler) DeregisterAgent(ctx context.Context, req *zynaxv1.DeregisterAgentRequest) (*zynaxv1.DeregisterAgentResponse, error) {
	deregisteredAt, err := h.svc.Deregister(ctx, req.GetAgentId())
	if err != nil {
		return nil, grpcErr(err)
	}
	return &zynaxv1.DeregisterAgentResponse{
		DeregisteredAt: timestamppb.New(deregisteredAt),
	}, nil
}

// GetAgent returns the full AgentDef for the given agent_id.
func (h *Handler) GetAgent(ctx context.Context, req *zynaxv1.GetAgentRequest) (*zynaxv1.AgentDef, error) {
	a, err := h.svc.GetByID(ctx, req.GetAgentId())
	if err != nil {
		return nil, grpcErr(err)
	}
	return domainToProto(a), nil
}

// ListAgents returns agents matching the optional label selector and pagination.
func (h *Handler) ListAgents(ctx context.Context, req *zynaxv1.ListAgentsRequest) (*zynaxv1.ListAgentsResponse, error) {
	result, err := h.svc.List(ctx, domain.ListFilter{
		LabelSelector:       req.GetLabelSelector(),
		IncludeDeregistered: req.GetIncludeDeregistered(),
		PageToken:           req.GetPageToken(),
		PageSize:            req.GetPageSize(),
	})
	if err != nil {
		return nil, grpcErr(err)
	}
	agents := make([]*zynaxv1.AgentDef, len(result.Agents))
	for i, a := range result.Agents {
		agents[i] = domainToProto(a)
	}
	return &zynaxv1.ListAgentsResponse{
		Agents:        agents,
		NextPageToken: result.NextPageToken,
	}, nil
}

// FindByCapability returns all active agents that declare the named capability.
func (h *Handler) FindByCapability(ctx context.Context, req *zynaxv1.FindByCapabilityRequest) (*zynaxv1.FindByCapabilityResponse, error) {
	agents, err := h.svc.FindByCapability(ctx, req.GetCapabilityName())
	if err != nil {
		return nil, grpcErr(err)
	}
	defs := make([]*zynaxv1.AgentDef, len(agents))
	for i, a := range agents {
		defs[i] = domainToProto(a)
	}
	return &zynaxv1.FindByCapabilityResponse{Agents: defs}, nil
}

func protoToDomain(p *zynaxv1.AgentDef) domain.Agent {
	if p == nil {
		return domain.Agent{}
	}
	caps := make([]domain.Capability, len(p.GetCapabilities()))
	for i, c := range p.GetCapabilities() {
		caps[i] = domain.Capability{
			Name:         c.GetName(),
			Description:  c.GetDescription(),
			InputSchema:  c.GetInputSchema(),
			OutputSchema: c.GetOutputSchema(),
		}
	}
	return domain.Agent{
		ID:           p.GetAgentId(),
		Name:         p.GetName(),
		Description:  p.GetDescription(),
		Endpoint:     p.GetEndpoint(),
		Capabilities: caps,
		Labels:       p.GetLabels(),
	}
}

func domainToProto(a domain.Agent) *zynaxv1.AgentDef {
	caps := make([]*zynaxv1.CapabilityDef, len(a.Capabilities))
	for i, c := range a.Capabilities {
		caps[i] = &zynaxv1.CapabilityDef{
			Name:         c.Name,
			Description:  c.Description,
			InputSchema:  c.InputSchema,
			OutputSchema: c.OutputSchema,
		}
	}
	return &zynaxv1.AgentDef{
		AgentId:      a.ID,
		Name:         a.Name,
		Description:  a.Description,
		Endpoint:     a.Endpoint,
		Capabilities: caps,
		Labels:       a.Labels,
		Status:       zynaxv1.AgentStatus(a.Status), //nolint:gosec // G115: domain enum mirrors proto enum; values 0-2 fit safely
		RegisteredAt: tsOrNil(a.RegisteredAt),
		UpdatedAt:    tsOrNil(a.UpdatedAt),
	}
}

func tsOrNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func grpcErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrAgentNotFound) {
		return status.Errorf(codes.NotFound, "%v", err)
	}
	if errors.Is(err, domain.ErrInvalidArgument) {
		return status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if errors.Is(err, context.Canceled) {
		return status.Errorf(codes.Canceled, "%v", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Errorf(codes.DeadlineExceeded, "%v", err)
	}
	return status.Errorf(codes.Internal, "%v", err)
}
