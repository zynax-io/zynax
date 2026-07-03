// SPDX-License-Identifier: Apache-2.0

// UNIMPLEMENTED tolerance (ADR-039): a CRD-era registry retires push
// registration; the adapter must keep serving, not crash-loop.
package registry

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

var errUnimpl = status.Error(codes.Unimplemented, "push registration retired (ADR-039)")

func TestRegisterAgent_UnimplementedTolerated(t *testing.T) {
	t.Parallel()
	f := &fakeRegistry{registerErr: errUnimpl}
	if err := RegisterAgent(context.Background(), f, &zynaxv1.AgentDef{AgentId: "a"}); err != nil {
		t.Fatalf("UNIMPLEMENTED must be tolerated: %v", err)
	}
}

func TestDeregisterAgent_UnimplementedTolerated(t *testing.T) {
	t.Parallel()
	f := &fakeRegistry{deregErr: errUnimpl}
	if err := DeregisterAgent(context.Background(), f, "a"); err != nil {
		t.Fatalf("UNIMPLEMENTED must be tolerated: %v", err)
	}
}
