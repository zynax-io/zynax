// SPDX-License-Identifier: Apache-2.0

// Package api wires domain interfaces to the MemoryService gRPC contract.
package api

import (
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/memory-service/internal/domain"
)

// Handler implements zynaxv1.MemoryServiceServer.
// J.2 scaffold: all RPCs return UNIMPLEMENTED via the embedded base.
// J.6 will replace the embedding stubs with real domain dispatch.
type Handler struct {
	zynaxv1.UnimplementedMemoryServiceServer
	kv  domain.KVStore
	vec domain.VectorStore
}

// NewHandler constructs a Handler. kv and vec may be nil in the J.2 scaffold;
// they will be wired in J.6 once the infrastructure adapters are implemented.
func NewHandler(kv domain.KVStore, vec domain.VectorStore) *Handler {
	return &Handler{kv: kv, vec: vec}
}
