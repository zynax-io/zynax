// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

// stubSchedulerClient is a canned zynaxv1.SchedulerServiceClient.
type stubSchedulerClient struct {
	resp  *zynaxv1.SelectAgentResponse
	err   error
	block bool
	got   *zynaxv1.SelectAgentRequest
}

func (s *stubSchedulerClient) SelectAgent(ctx context.Context, req *zynaxv1.SelectAgentRequest, _ ...grpc.CallOption) (*zynaxv1.SelectAgentResponse, error) {
	s.got = req
	if s.block {
		<-ctx.Done()
		return nil, fmt.Errorf("stub: %w", ctx.Err())
	}
	return s.resp, s.err
}

func selResp(agentID, capName, schema string) *zynaxv1.SelectAgentResponse {
	return &zynaxv1.SelectAgentResponse{
		Agent: &zynaxv1.AgentDef{
			AgentId:  agentID,
			Name:     agentID,
			Endpoint: agentID + ".default.svc:50061",
			Capabilities: []*zynaxv1.CapabilityDef{
				{Name: "other", InputSchema: []byte(`{"other":true}`)},
				{Name: capName, InputSchema: []byte(schema)},
			},
		},
		Rationale: &zynaxv1.SelectionRationale{
			CandidatesMatched: 2, CandidatesReady: 1,
			Mode:    zynaxv1.SelectionMode_SELECTION_MODE_METRICS_WEIGHTED,
			Summary: "test",
		},
	}
}

func TestSelect_HappyPathCarriesSchemaAndExpert(t *testing.T) {
	stub := &stubSchedulerClient{resp: selResp("default/reviewer", "review", `{"type":"object"}`)}
	c := &schedulerClient{client: stub, callTimeout: time.Second}

	got, err := c.Select(context.Background(), "review", "security-reviewer")
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if got.AgentID != "default/reviewer" || got.Endpoint == "" {
		t.Errorf("agent = %+v", got)
	}
	// The ADR-028 binding source: the SELECTED capability's schema, not another's.
	if string(got.InputSchema) != `{"type":"object"}` {
		t.Errorf("input schema = %s", got.InputSchema)
	}
	// Expert target rides the request (ADR-039 §4).
	if stub.got.GetExpertTarget() != "security-reviewer" || stub.got.GetCapabilityName() != "review" {
		t.Errorf("request = %+v", stub.got)
	}
}

// TestSelect_ContractErrorsMapToNoEligibleAgent keeps the broker's dispatch
// contract stable: scheduler NOT_FOUND / FAILED_PRECONDITION both mean "no
// eligible agent" to the domain.
func TestSelect_ContractErrorsMapToNoEligibleAgent(t *testing.T) {
	for name, code := range map[string]codes.Code{
		"not found":           codes.NotFound,
		"failed precondition": codes.FailedPrecondition,
	} {
		t.Run(name, func(t *testing.T) {
			stub := &stubSchedulerClient{err: status.Error(code, "nope")}
			c := &schedulerClient{client: stub, callTimeout: time.Second}
			_, err := c.Select(context.Background(), "review", "")
			if !errors.Is(err, domain.ErrNoEligibleAgent) {
				t.Fatalf("err = %v, want ErrNoEligibleAgent", err)
			}
		})
	}
}

func TestSelect_InfraErrorsPassThrough(t *testing.T) {
	stub := &stubSchedulerClient{err: status.Error(codes.Unavailable, "scheduler down")}
	c := &schedulerClient{client: stub, callTimeout: time.Second}
	_, err := c.Select(context.Background(), "review", "")
	if err == nil || errors.Is(err, domain.ErrNoEligibleAgent) {
		t.Fatalf("infra error must not read as no-eligible-agent: %v", err)
	}
}

func TestSelect_DeadlineExceeded(t *testing.T) {
	c := &schedulerClient{client: &stubSchedulerClient{block: true}, callTimeout: 50 * time.Millisecond}
	_, err := c.Select(context.Background(), "review", "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded in chain, got: %v", err)
	}
}

func TestSelect_NilAgentIsContractViolation(t *testing.T) {
	stub := &stubSchedulerClient{resp: &zynaxv1.SelectAgentResponse{}}
	c := &schedulerClient{client: stub, callTimeout: time.Second}
	if _, err := c.Select(context.Background(), "review", ""); err == nil {
		t.Fatal("nil agent in response must error")
	}
}
