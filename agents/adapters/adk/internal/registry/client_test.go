// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/adk/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const testAgentID = "adk-1"

// fakeRegistry implements AgentRegistryServiceClient. Embedding the interface
// supplies the methods serve() never calls; only Register/Deregister are real.
type fakeRegistry struct {
	zynaxv1.AgentRegistryServiceClient
	registerErr  error
	registered   *zynaxv1.AgentDef
	deregistered string
	deregErr     error
}

func (f *fakeRegistry) RegisterAgent(_ context.Context, in *zynaxv1.RegisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.RegisterAgentResponse, error) {
	if f.registerErr != nil {
		return nil, f.registerErr
	}
	f.registered = in.Agent
	return &zynaxv1.RegisterAgentResponse{}, nil
}

func (f *fakeRegistry) DeregisterAgent(_ context.Context, in *zynaxv1.DeregisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.DeregisterAgentResponse, error) {
	if f.deregErr != nil {
		return nil, f.deregErr
	}
	f.deregistered = in.AgentId
	return &zynaxv1.DeregisterAgentResponse{}, nil
}

func testConfig() *config.AdapterConfig {
	return &config.AdapterConfig{
		AgentID:     testAgentID,
		Name:        "adk-adapter",
		Description: "desc",
		Endpoint:    "adk-adapter:50080",
		Capabilities: []config.CapabilityConfig{
			{Name: "triage", Description: "d", InputSchemaJSON: "{}", OutputSchemaJSON: "{}"},
		},
	}
}

func TestBuildAgentDef(t *testing.T) {
	def := BuildAgentDef(testConfig())
	if def.AgentId != testAgentID {
		t.Errorf("agent_id = %q", def.AgentId)
	}
	if def.Endpoint != "adk-adapter:50080" {
		t.Errorf("endpoint = %q", def.Endpoint)
	}
	if len(def.Capabilities) != 1 || def.Capabilities[0].Name != "triage" {
		t.Fatalf("capabilities = %+v", def.Capabilities)
	}
	if string(def.Capabilities[0].InputSchema) != "{}" {
		t.Errorf("input schema = %q", def.Capabilities[0].InputSchema)
	}
}

func TestRegisterAgent_Success(t *testing.T) {
	f := &fakeRegistry{}
	if err := RegisterAgent(context.Background(), f, BuildAgentDef(testConfig())); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.registered == nil || f.registered.AgentId != testAgentID {
		t.Errorf("registered = %+v", f.registered)
	}
}

func TestRegisterAgent_NonTransientReturnsImmediately(t *testing.T) {
	f := &fakeRegistry{registerErr: status.Error(codes.InvalidArgument, "bad")}
	err := RegisterAgent(context.Background(), f, BuildAgentDef(testConfig()))
	if err == nil {
		t.Fatal("expected non-transient error")
	}
	if !strings.Contains(err.Error(), "non-transient") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestRegisterAgent_TransientRetriesUntilCancelled(t *testing.T) {
	// Always-Unavailable forces a retry; a short ctx deadline trips the backoff
	// select's ctx.Done() branch instead of waiting the full base delay.
	f := &fakeRegistry{registerErr: status.Error(codes.Unavailable, "down")}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := RegisterAgent(ctx, f, BuildAgentDef(testConfig())); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("err = %v, want cancellation", err)
	}
}

func TestDeregisterAgent(t *testing.T) {
	f := &fakeRegistry{}
	if err := DeregisterAgent(context.Background(), f, testAgentID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.deregistered != testAgentID {
		t.Errorf("deregistered = %q", f.deregistered)
	}

	f2 := &fakeRegistry{deregErr: errors.New("boom")}
	if err := DeregisterAgent(context.Background(), f2, testAgentID); err == nil {
		t.Error("expected deregister error")
	}
}

func TestIsTransient(t *testing.T) {
	cases := []struct {
		code codes.Code
		want bool
	}{
		{codes.Unavailable, true},
		{codes.Internal, true},
		{codes.DeadlineExceeded, true},
		{codes.InvalidArgument, false},
		{codes.NotFound, false},
	}
	for _, tc := range cases {
		if got := isTransient(status.Error(tc.code, "x")); got != tc.want {
			t.Errorf("isTransient(%v) = %v, want %v", tc.code, got, tc.want)
		}
	}
	if isTransient(errors.New("not a status error")) {
		t.Error("non-status error should not be transient")
	}
}
