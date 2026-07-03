// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/agent-registry/internal/api"
)

// CRD era (ADR-039): every push-era registry RPC answers UNIMPLEMENTED with a
// migration pointer until its M9 hard removal. These tests pin that contract
// so no RPC silently regresses to serving push-era state.
func TestRetiredRegistrySurface(t *testing.T) {
	t.Parallel()
	h := api.NewHandler()
	ctx := context.Background()

	//nolint:wrapcheck // raw handler errors are the assertion subject here
	calls := map[string]func() error{
		"RegisterAgent": func() error {
			_, err := h.RegisterAgent(ctx, &zynaxv1.RegisterAgentRequest{Agent: &zynaxv1.AgentDef{AgentId: "a"}})
			return err
		},
		"DeregisterAgent": func() error {
			_, err := h.DeregisterAgent(ctx, &zynaxv1.DeregisterAgentRequest{AgentId: "a"})
			return err
		},
		"GetAgent": func() error {
			_, err := h.GetAgent(ctx, &zynaxv1.GetAgentRequest{AgentId: "a"})
			return err
		},
		"ListAgents": func() error {
			_, err := h.ListAgents(ctx, &zynaxv1.ListAgentsRequest{})
			return err
		},
		"FindByCapability": func() error {
			_, err := h.FindByCapability(ctx, &zynaxv1.FindByCapabilityRequest{CapabilityName: "echo"})
			return err
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := call()
			if status.Code(err) != codes.Unimplemented {
				t.Fatalf("%s: code = %v, want Unimplemented", name, status.Code(err))
			}
			msg := status.Convert(err).Message()
			if !strings.Contains(msg, "agent-crd-migration") || !strings.Contains(msg, "ADR-039") {
				t.Errorf("%s: message must carry the migration pointer, got %q", name, msg)
			}
		})
	}
}
