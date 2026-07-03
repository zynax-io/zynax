// SPDX-License-Identifier: Apache-2.0

// UNIMPLEMENTED tolerance (ADR-039): a CRD-era registry retires push
// registration; the adapter must keep serving, not crash-loop.
package registry_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zynax-io/zynax/agents/adapters/ci/internal/registry"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

var errUnimpl = status.Error(codes.Unimplemented, "push registration retired (ADR-039)")

func TestRegisterAgent_UnimplementedTolerated(t *testing.T) {
	t.Parallel()
	stub := &stubRegistryClient{registerErr: errUnimpl}
	if err := registry.RegisterAgent(context.Background(), stub, &zynaxv1.AgentDef{AgentId: "a"}); err != nil {
		t.Fatalf("UNIMPLEMENTED must be tolerated: %v", err)
	}
	if stub.calls != 1 {
		t.Errorf("calls = %d, want 1 (no retries against a retired RPC)", stub.calls)
	}
}

func TestDeregisterAgent_UnimplementedTolerated(t *testing.T) {
	t.Parallel()
	stub := &stubRegistryClient{deregisterErr: errUnimpl}
	if err := registry.DeregisterAgent(context.Background(), stub, "a"); err != nil {
		t.Fatalf("UNIMPLEMENTED must be tolerated: %v", err)
	}
}
