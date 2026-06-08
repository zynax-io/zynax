// SPDX-License-Identifier: Apache-2.0

// Package api wires domain interfaces to the EventBusService gRPC contract.
package api

import (
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/event-bus/internal/domain"
)

// Handler implements zynaxv1.EventBusServiceServer.
// O1 scaffold: all RPCs return UNIMPLEMENTED via the embedded base.
// O2–O4 will replace the stub with real domain dispatch.
type Handler struct {
	zynaxv1.UnimplementedEventBusServiceServer
	bus domain.EventBus
}

// NewHandler constructs a Handler. bus may be nil in the O1 scaffold;
// it will be required once O2 wires the Publish path.
func NewHandler(bus domain.EventBus) *Handler {
	return &Handler{bus: bus}
}
