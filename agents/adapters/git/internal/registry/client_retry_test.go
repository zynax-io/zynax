// SPDX-License-Identifier: Apache-2.0

package registry_test

// Tests for RegisterAgent retry behaviour (transient errors, permanent errors,
// context cancellation) — blackbox tests.
// Closes #717 — part of the git-adapter coverage epic (#713).

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/registry"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// multiResponseStub allows configuring a list of errors to return on successive
// RegisterAgent calls. If calls exceed the list length it returns nil (success).
type multiResponseStub struct {
	mu        sync.Mutex
	responses []error
	calls     int
}

func (s *multiResponseStub) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.RegisterAgentResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.calls
	s.calls++
	if idx < len(s.responses) {
		if s.responses[idx] != nil {
			return nil, s.responses[idx]
		}
	}
	return &zynaxv1.RegisterAgentResponse{}, nil
}

func (s *multiResponseStub) DeregisterAgent(_ context.Context, _ *zynaxv1.DeregisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.DeregisterAgentResponse, error) {
	return &zynaxv1.DeregisterAgentResponse{}, nil
}

func (s *multiResponseStub) GetAgent(_ context.Context, _ *zynaxv1.GetAgentRequest, _ ...grpc.CallOption) (*zynaxv1.AgentDef, error) {
	return nil, nil
}

func (s *multiResponseStub) ListAgents(_ context.Context, _ *zynaxv1.ListAgentsRequest, _ ...grpc.CallOption) (*zynaxv1.ListAgentsResponse, error) {
	return nil, nil
}

func (s *multiResponseStub) FindByCapability(_ context.Context, _ *zynaxv1.FindByCapabilityRequest, _ ...grpc.CallOption) (*zynaxv1.FindByCapabilityResponse, error) {
	return nil, nil
}

// ── RegisterAgent retry tests ─────────────────────────────────────────────────

// TestRegisterAgent_TransientTriggersRetry verifies that a transient error causes
// the retry loop to be entered and that context cancellation short-circuits it.
// Timeline: attempt 1 → Unavailable (transient) → delay starts (2s) → ctx cancelled
// after 20ms → return "registration cancelled".
func TestRegisterAgent_TransientTriggersRetry(t *testing.T) {
	t.Parallel()
	stub := &multiResponseStub{
		responses: []error{status.Error(codes.Unavailable, "unavailable")},
	}
	def := &zynaxv1.AgentDef{AgentId: "git-adapter"}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel context after 20ms — the retry delay (2s) fires much later, so
	// ctx.Done() wins in the select, proving the retry path is entered.
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := registry.RegisterAgent(ctx, stub, def)
	if err == nil {
		t.Fatal("expected error after transient failure + cancellation")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("expected cancellation message, got: %v", err)
	}
	// One attempt was made before the retry delay was entered.
	stub.mu.Lock()
	calls := stub.calls
	stub.mu.Unlock()
	if calls != 1 {
		t.Errorf("expected exactly 1 call before ctx cancelled, got %d", calls)
	}
}

// TestRegisterAgent_NonTransient_ReturnsImmediately verifies that a non-transient
// gRPC error causes RegisterAgent to return without retrying.
func TestRegisterAgent_NonTransient_ReturnsImmediately(t *testing.T) {
	t.Parallel()
	stub := &multiResponseStub{
		responses: []error{status.Error(codes.NotFound, "agent-def not found")},
	}
	def := &zynaxv1.AgentDef{AgentId: "git-adapter"}

	err := registry.RegisterAgent(context.Background(), stub, def)
	if err == nil {
		t.Fatal("expected error for non-transient failure")
	}
	if !strings.Contains(err.Error(), "non-transient") {
		t.Errorf("expected 'non-transient' in error, got: %v", err)
	}
	stub.mu.Lock()
	calls := stub.calls
	stub.mu.Unlock()
	if calls != 1 {
		t.Errorf("non-transient error must return immediately (1 call); got %d", calls)
	}
}

// TestRegisterAgent_DeregisterError verifies that DeregisterAgent wraps errors properly.
func TestRegisterAgent_DeregisterError(t *testing.T) {
	t.Parallel()
	stub := &stubRegistryClient{
		deregisterErr: status.Error(codes.Unavailable, "registry down"),
	}
	err := registry.DeregisterAgent(context.Background(), stub, "git-adapter")
	if err == nil {
		t.Fatal("expected error from DeregisterAgent")
	}
	if !strings.Contains(err.Error(), "deregister failed") {
		t.Errorf("expected 'deregister failed' in error, got: %v", err)
	}
}
