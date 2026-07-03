// SPDX-License-Identifier: Apache-2.0

// This file implements the SchedulerService gRPC handler (ADR-039):
// SelectAgent over the informer-backed capability index + scoring pipeline.
// Registered only when the CRD informer is enabled — the legacy
// AgentRegistryService surface is untouched.

package api

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/agent-registry/internal/domain/scheduler"
)

// SchedulerHandler implements SchedulerServiceServer over the domain scorer.
type SchedulerHandler struct {
	zynaxv1.UnimplementedSchedulerServiceServer
	index  *scheduler.Index
	scorer *scheduler.Scorer
}

// NewSchedulerHandler constructs the handler over the informer-fed index.
func NewSchedulerHandler(idx *scheduler.Index, sc *scheduler.Scorer) *SchedulerHandler {
	return &SchedulerHandler{index: idx, scorer: sc}
}

// SelectAgent returns exactly one chosen agent + a structured rationale
// (contract invariants 1-7 in zynax/v1/scheduler.proto).
func (h *SchedulerHandler) SelectAgent(ctx context.Context, req *zynaxv1.SelectAgentRequest) (*zynaxv1.SelectAgentResponse, error) {
	if req.GetCapabilityName() == "" {
		return nil, status.Error(codes.InvalidArgument, "capability_name must not be empty")
	}

	res, err := h.scorer.Select(ctx, h.index, scheduler.Request{
		Capability:   req.GetCapabilityName(),
		Constraints:  protoConstraints(req.GetConstraints()),
		RoundRobin:   req.GetPolicy() == zynaxv1.SelectionPolicy_SELECTION_POLICY_ROUND_ROBIN,
		ExpertTarget: req.GetExpertTarget(),
	})
	if err != nil {
		return nil, schedulerErr(err)
	}

	return &zynaxv1.SelectAgentResponse{
		Agent:     candidateToProto(res.Chosen),
		Rationale: rationaleToProto(res.Rationale),
	}, nil
}

// schedulerErr maps domain errors to the contract's gRPC codes.
func schedulerErr(err error) error {
	var fe *scheduler.FilteredError
	switch {
	case errors.Is(err, scheduler.ErrNoCapability):
		return status.Error(codes.NotFound, err.Error())
	case errors.As(err, &fe):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func protoConstraints(c *zynaxv1.SelectionConstraints) scheduler.Constraints {
	if c == nil {
		return scheduler.Constraints{}
	}
	return scheduler.Constraints{
		RequiredTags:      c.GetTags(),
		RequiredLanguage:  c.GetLanguage(),
		RequiredModel:     c.GetModel(),
		RequireGPU:        c.GetRequireGpu(),
		RequiredProtocols: c.GetProtocols(),
	}
}

// candidateToProto builds the returned AgentDef from the scheduler's view of
// the Agent CR: agent_id is the CR key ("namespace/name"), the endpoint is
// directly dialable, and capabilities carry input_schema so the task-broker
// keeps the ADR-028 context-slice binding from the response.
func candidateToProto(c scheduler.Candidate) *zynaxv1.AgentDef {
	caps := make([]*zynaxv1.CapabilityDef, 0, len(c.Capabilities))
	for _, capability := range c.Capabilities {
		caps = append(caps, &zynaxv1.CapabilityDef{
			Name:         capability.ID,
			Description:  capability.Description,
			InputSchema:  []byte(capability.InputSchema),
			OutputSchema: []byte(capability.OutputSchema),
		})
	}
	return &zynaxv1.AgentDef{
		AgentId:      c.Key,
		Name:         c.Name,
		Endpoint:     c.Endpoint,
		Capabilities: caps,
		Status:       zynaxv1.AgentStatus_AGENT_STATUS_REGISTERED,
	}
}

func rationaleToProto(r scheduler.Rationale) *zynaxv1.SelectionRationale {
	return &zynaxv1.SelectionRationale{
		CandidatesMatched:           int32(r.CandidatesMatched),           //nolint:gosec // candidate counts are small
		CandidatesAfterConstraints:  int32(r.CandidatesAfterConstraints),  //nolint:gosec // candidate counts are small
		CandidatesReady:             int32(r.CandidatesReady),             //nolint:gosec // candidate counts are small
		CandidatesAfterExpertFilter: int32(r.CandidatesAfterExpertFilter), //nolint:gosec // candidate counts are small
		Mode:                        modeToProto(r.Mode),
		WinningFactors:              r.WinningFactors,
		Summary:                     r.Summary,
	}
}

func modeToProto(m scheduler.Mode) zynaxv1.SelectionMode {
	switch m {
	case scheduler.ModeMetricsWeighted:
		return zynaxv1.SelectionMode_SELECTION_MODE_METRICS_WEIGHTED
	case scheduler.ModeRoundRobin:
		return zynaxv1.SelectionMode_SELECTION_MODE_ROUND_ROBIN
	case scheduler.ModeDegradedRoundRobin:
		return zynaxv1.SelectionMode_SELECTION_MODE_DEGRADED_ROUND_ROBIN
	default:
		return zynaxv1.SelectionMode_SELECTION_MODE_UNSPECIFIED
	}
}
