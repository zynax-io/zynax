// SPDX-License-Identifier: Apache-2.0

// Package api wires domain interfaces to the EventBusService gRPC contract.
package api

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/event-bus/internal/domain"
)

// Handler implements zynaxv1.EventBusServiceServer.
// O1–O2: Publish path wired; Subscribe/Unsubscribe remain UNIMPLEMENTED (O3–O4).
type Handler struct {
	zynaxv1.UnimplementedEventBusServiceServer
	bus domain.EventBus
}

// NewHandler constructs a Handler backed by the provided EventBus.
func NewHandler(bus domain.EventBus) *Handler {
	return &Handler{bus: bus}
}

// Publish validates the incoming CloudEvent and delegates to the domain bus.
// Returns INVALID_ARGUMENT when the event envelope is missing or required
// fields (id, source, type) are empty.
func (h *Handler) Publish(ctx context.Context, req *zynaxv1.PublishRequest) (*zynaxv1.PublishResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.FromContextError(err).Err()
	}

	if req.GetEvent() == nil {
		return nil, status.Error(codes.InvalidArgument, "event must not be nil")
	}
	ev := req.GetEvent()
	if ev.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "event.id must not be empty")
	}
	if ev.GetSource() == "" {
		return nil, status.Error(codes.InvalidArgument, "event.source must not be empty")
	}
	if ev.GetType() == "" {
		return nil, status.Error(codes.InvalidArgument, "event.type must not be empty")
	}

	domainEvent := domain.CloudEvent{
		ID:              ev.GetId(),
		Source:          ev.GetSource(),
		SpecVersion:     ev.GetSpecversion(),
		Type:            ev.GetType(),
		DataContentType: ev.GetDatacontenttype(),
		Data:            ev.GetData(),
		WorkflowID:      ev.GetWorkflowId(),
		RunID:           ev.GetRunId(),
		Namespace:       ev.GetNamespace(),
		CapabilityName:  ev.GetCapabilityName(),
	}
	if ev.GetTime() != nil {
		domainEvent.Time = ev.GetTime().AsTime()
	}

	eventID, err := h.bus.Publish(ctx, domainEvent)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "publish failed: %v", err)
	}

	return &zynaxv1.PublishResponse{
		EventId:    eventID,
		AcceptedAt: timestamppb.New(time.Now().UTC()),
	}, nil
}
