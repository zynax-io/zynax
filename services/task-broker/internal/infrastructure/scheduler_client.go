// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

type schedulerClient struct {
	client      zynaxv1.SchedulerServiceClient
	conn        *grpc.ClientConn
	callTimeout time.Duration
}

// NewSchedulerClient dials the CRD-native scheduler (hosted by the
// agent-registry deployment, ADR-039) and returns an AgentSelector: one
// SelectAgent call per dispatch replaces FindByCapability + local
// round-robin. callTimeout is applied as a per-call deadline; creds controls
// transport security. The returned cleanup closes the connection.
func NewSchedulerClient(addr string, callTimeout time.Duration, creds credentials.TransportCredentials) (domain.AgentSelector, func(), error) {
	conn, err := grpc.NewClient(addr, tracingDialOpts(creds)...)
	if err != nil {
		return nil, func() {}, fmt.Errorf("task-broker: scheduler dial: %w", err)
	}
	return &schedulerClient{
		client:      zynaxv1.NewSchedulerServiceClient(conn),
		conn:        conn,
		callTimeout: callTimeout,
	}, func() { _ = conn.Close() }, nil
}

// Select implements domain.AgentSelector. NOT_FOUND (no capability match) and
// FAILED_PRECONDITION (constraints/readiness/expert eliminated everything) map
// onto ErrNoEligibleAgent so the broker's dispatch contract is unchanged; the
// structured selection rationale is logged for the task trail.
func (s *schedulerClient) Select(ctx context.Context, capabilityName, expertTarget string) (domain.AgentInfo, error) {
	callCtx, cancel := context.WithTimeout(ctx, s.callTimeout)
	defer cancel()
	resp, err := s.client.SelectAgent(callCtx, &zynaxv1.SelectAgentRequest{
		CapabilityName: capabilityName,
		ExpertTarget:   expertTarget,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.NotFound, codes.FailedPrecondition:
			return domain.AgentInfo{}, fmt.Errorf("%w: %s", domain.ErrNoEligibleAgent, status.Convert(err).Message())
		default:
			return domain.AgentInfo{}, fmt.Errorf("task-broker: select agent for %q: %w", capabilityName, err)
		}
	}

	a := resp.GetAgent()
	if a == nil {
		return domain.AgentInfo{}, errors.New("task-broker: scheduler returned no agent (contract violation)")
	}
	info := domain.AgentInfo{
		AgentID:  a.GetAgentId(),
		Name:     a.GetName(),
		Endpoint: a.GetEndpoint(),
	}
	// Carry the selected capability's input_schema so the domain layer keeps
	// the ADR-028 context-slice binding from the response (ADR-039 §4).
	for _, c := range a.GetCapabilities() {
		if c.GetName() == capabilityName {
			info.InputSchema = c.GetInputSchema()
			break
		}
	}

	if r := resp.GetRationale(); r != nil {
		slog.Info("task-broker: agent selected",
			"capability", capabilityName,
			"agent", info.AgentID,
			"mode", r.GetMode().String(),
			"candidates_matched", r.GetCandidatesMatched(),
			"candidates_ready", r.GetCandidatesReady(),
			"summary", r.GetSummary())
	}
	return info, nil
}
