// SPDX-License-Identifier: Apache-2.0

// Package api implements the EngineAdapterService gRPC server handler.
package api

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
)

const runIDRequired = "run_id is required"

// Handler implements EngineAdapterServiceServer, delegating all calls to
// a domain.WorkflowEngine. The concrete engine is injected at startup;
// this handler has no dependency on any engine-specific package (ADR-015).
type Handler struct {
	zynaxv1.UnimplementedEngineAdapterServiceServer
	engine domain.WorkflowEngine
}

// NewHandler constructs a Handler wrapping the given WorkflowEngine.
func NewHandler(engine domain.WorkflowEngine) *Handler {
	return &Handler{engine: engine}
}

// SubmitWorkflow starts a compiled workflow and returns the adapter-assigned run ID.
func (h *Handler) SubmitWorkflow(ctx context.Context, req *zynaxv1.SubmitWorkflowRequest) (*zynaxv1.SubmitWorkflowResponse, error) {
	if req.GetWorkflowIr() == nil {
		return nil, status.Error(codes.InvalidArgument, "workflow_ir is required")
	}
	run, err := h.engine.Submit(ctx, req.GetWorkflowIr(), req.GetLabels())
	if err != nil {
		return nil, grpcErr(err)
	}
	return &zynaxv1.SubmitWorkflowResponse{
		RunId:       run.RunID,
		SubmittedAt: timestamppb.New(time.Now()),
	}, nil
}

// SignalWorkflow delivers a named event to a running workflow.
func (h *Handler) SignalWorkflow(ctx context.Context, req *zynaxv1.SignalWorkflowRequest) (*zynaxv1.SignalWorkflowResponse, error) {
	if req.GetRunId() == "" {
		return nil, status.Error(codes.InvalidArgument, runIDRequired)
	}
	if err := h.engine.Signal(ctx, req.GetRunId(), req.GetEventType(), req.GetPayload()); err != nil {
		return nil, grpcErr(err)
	}
	return &zynaxv1.SignalWorkflowResponse{SignalledAt: timestamppb.New(time.Now())}, nil
}

// CancelWorkflow requests graceful cancellation of a running workflow.
func (h *Handler) CancelWorkflow(ctx context.Context, req *zynaxv1.CancelWorkflowRequest) (*zynaxv1.CancelWorkflowResponse, error) {
	if req.GetRunId() == "" {
		return nil, status.Error(codes.InvalidArgument, runIDRequired)
	}
	if err := h.engine.Cancel(ctx, req.GetRunId(), req.GetReason()); err != nil {
		return nil, grpcErr(err)
	}
	return &zynaxv1.CancelWorkflowResponse{CancelledAt: timestamppb.New(time.Now())}, nil
}

// GetWorkflowStatus returns the current run metadata for a workflow.
func (h *Handler) GetWorkflowStatus(ctx context.Context, req *zynaxv1.GetWorkflowStatusRequest) (*zynaxv1.WorkflowRun, error) {
	if req.GetRunId() == "" {
		return nil, status.Error(codes.InvalidArgument, runIDRequired)
	}
	run, err := h.engine.GetStatus(ctx, req.GetRunId())
	if err != nil {
		return nil, grpcErr(err)
	}
	return runToProto(run), nil
}

// WatchWorkflow streams WorkflowEvents until the workflow reaches a terminal state.
func (h *Handler) WatchWorkflow(req *zynaxv1.WatchWorkflowRequest, stream grpc.ServerStreamingServer[zynaxv1.WorkflowEvent]) error {
	if req.GetRunId() == "" {
		return status.Error(codes.InvalidArgument, runIDRequired)
	}
	err := h.engine.Watch(stream.Context(), req.GetRunId(), func(ev *domain.WorkflowEvent) error {
		return stream.Send(eventToProto(ev))
	})
	return grpcErr(err)
}

func runToProto(r *domain.WorkflowRun) *zynaxv1.WorkflowRun {
	return &zynaxv1.WorkflowRun{
		RunId:              r.RunID,
		WorkflowId:         r.WorkflowID,
		Namespace:          r.Namespace,
		Status:             zynaxv1.WorkflowStatus(r.Status), //nolint:gosec // G115: domain status is a small positive int enum; conversion is safe
		CurrentState:       r.CurrentState,
		Engine:             r.Engine,
		Labels:             r.Labels,
		SubmittedAt:        tsOrNil(r.SubmittedAt),
		StartedAt:          tsOrNil(r.StartedAt),
		FinishedAt:         tsOrNil(r.FinishedAt),
		CancellationReason: r.CancellationReason,
	}
}

func eventToProto(ev *domain.WorkflowEvent) *zynaxv1.WorkflowEvent {
	return &zynaxv1.WorkflowEvent{
		RunId:     ev.RunID,
		EventType: ev.EventType,
		FromState: ev.FromState,
		ToState:   ev.ToState,
		Status:    zynaxv1.WorkflowStatus(ev.Status), //nolint:gosec // G115: domain status is a small positive int enum; conversion is safe
		Payload:   ev.Payload,
		Timestamp: timestamppb.New(ev.Timestamp),
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
	if errors.Is(err, domain.ErrExecutionNotFound) {
		return status.Errorf(codes.NotFound, "%v", err)
	}
	if errors.Is(err, domain.ErrTerminalState) {
		return status.Errorf(codes.FailedPrecondition, "%v", err)
	}
	if errors.Is(err, domain.ErrEngineUnavailable) {
		return status.Errorf(codes.Unavailable, "%v", err)
	}
	if errors.Is(err, context.Canceled) {
		return status.Errorf(codes.Canceled, "%v", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Errorf(codes.DeadlineExceeded, "%v", err)
	}
	return status.Errorf(codes.Internal, "%v", err)
}
