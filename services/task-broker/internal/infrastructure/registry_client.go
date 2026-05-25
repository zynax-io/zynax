// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/task-broker/internal/domain"
)

type registryClient struct {
	client      zynaxv1.AgentRegistryServiceClient
	conn        *grpc.ClientConn
	callTimeout time.Duration
}

// NewRegistryClient dials the agent registry and returns an AgentFinder.
// callTimeout is applied as a per-call deadline on every outgoing gRPC request.
// The returned cleanup function closes the connection and must be deferred by the caller.
func NewRegistryClient(addr string, callTimeout time.Duration) (domain.AgentFinder, func(), error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, func() {}, fmt.Errorf("task-broker: registry dial: %w", err)
	}
	return &registryClient{
		client:      zynaxv1.NewAgentRegistryServiceClient(conn),
		conn:        conn,
		callTimeout: callTimeout,
	}, func() { _ = conn.Close() }, nil
}

func (r *registryClient) FindByCapability(ctx context.Context, capabilityName string) ([]domain.AgentInfo, error) {
	callCtx, cancel := context.WithTimeout(ctx, r.callTimeout)
	defer cancel()
	resp, err := r.client.FindByCapability(callCtx, &zynaxv1.FindByCapabilityRequest{
		CapabilityName: capabilityName,
	})
	if err != nil {
		return nil, fmt.Errorf("task-broker: find by capability %q: %w", capabilityName, err)
	}
	agents := make([]domain.AgentInfo, len(resp.GetAgents()))
	for i, a := range resp.GetAgents() {
		agents[i] = domain.AgentInfo{
			AgentID:  a.GetAgentId(),
			Endpoint: a.GetEndpoint(),
		}
	}
	return agents, nil
}
