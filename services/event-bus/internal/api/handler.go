// SPDX-License-Identifier: Apache-2.0

// Package api wires domain interfaces to the EventBusService gRPC contract.
package api

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/event-bus/internal/domain"
)

// Handler implements zynaxv1.EventBusServiceServer.
// O1–O3: Publish and Subscribe paths wired; Unsubscribe remains UNIMPLEMENTED (O4).
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

// Subscribe opens a server-streaming gRPC channel that delivers CloudEvents
// matching the subscriber's type_pattern and optional workflow_id scope.
// Returns INVALID_ARGUMENT when subscriber_id or type_pattern is empty.
// The stream terminates cleanly when the client cancels (context cancellation).
func (h *Handler) Subscribe(req *zynaxv1.SubscribeRequest, stream grpc.ServerStreamingServer[zynaxv1.SubscribeResponse]) error {
	ctx := stream.Context()

	if req.GetSubscriberId() == "" {
		return status.Error(codes.InvalidArgument, "subscriber_id must not be empty")
	}
	if req.GetTypePattern() == "" {
		return status.Error(codes.InvalidArgument, "type_pattern must not be empty")
	}

	domainReq := domain.SubscribeRequest{
		SubscriberID: req.GetSubscriberId(),
		TypePattern:  req.GetTypePattern(),
		WorkflowID:   req.GetWorkflowId(),
	}

	ch, err := h.bus.Subscribe(ctx, domainReq)
	if err != nil {
		return status.Errorf(codes.Internal, "subscribe failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return status.FromContextError(ctx.Err()).Err()
		case event, ok := <-ch:
			if !ok {
				// Channel closed — subscription ended.
				return nil
			}
			resp := &zynaxv1.SubscribeResponse{
				SubscriberId: req.GetSubscriberId(),
				Event:        domainEventToProto(event),
			}
			if sendErr := stream.Send(resp); sendErr != nil {
				return fmt.Errorf("subscribe: send: %w", sendErr)
			}
		}
	}
}

// domainEventToProto converts a domain.CloudEvent to its proto representation.
func domainEventToProto(event domain.CloudEvent) *zynaxv1.CloudEvent {
	pbEvent := &zynaxv1.CloudEvent{
		Id:              event.ID,
		Source:          event.Source,
		Specversion:     event.SpecVersion,
		Type:            event.Type,
		Datacontenttype: event.DataContentType,
		Data:            event.Data,
		WorkflowId:      event.WorkflowID,
		RunId:           event.RunID,
		Namespace:       event.Namespace,
		CapabilityName:  event.CapabilityName,
	}
	if !event.Time.IsZero() {
		pbEvent.Time = timestamppb.New(event.Time)
	}
	return pbEvent
}
