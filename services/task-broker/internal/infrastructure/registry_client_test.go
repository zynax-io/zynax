// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"google.golang.org/grpc"
)

// blockingRegistryClient hangs on FindByCapability until the context is cancelled,
// simulating a registry that does not respond within the deadline.
type blockingRegistryClient struct{}

func (b *blockingRegistryClient) FindByCapability(ctx context.Context, _ *zynaxv1.FindByCapabilityRequest, _ ...grpc.CallOption) (*zynaxv1.FindByCapabilityResponse, error) {
	<-ctx.Done()
	return nil, fmt.Errorf("blocking registry: %w", ctx.Err())
}
func (b *blockingRegistryClient) RegisterAgent(_ context.Context, _ *zynaxv1.RegisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.RegisterAgentResponse, error) {
	return nil, nil
}
func (b *blockingRegistryClient) DeregisterAgent(_ context.Context, _ *zynaxv1.DeregisterAgentRequest, _ ...grpc.CallOption) (*zynaxv1.DeregisterAgentResponse, error) {
	return nil, nil
}
func (b *blockingRegistryClient) GetAgent(_ context.Context, _ *zynaxv1.GetAgentRequest, _ ...grpc.CallOption) (*zynaxv1.AgentDef, error) {
	return nil, nil
}
func (b *blockingRegistryClient) ListAgents(_ context.Context, _ *zynaxv1.ListAgentsRequest, _ ...grpc.CallOption) (*zynaxv1.ListAgentsResponse, error) {
	return nil, nil
}

func TestFindByCapability_DeadlineExceeded(t *testing.T) {
	r := &registryClient{client: &blockingRegistryClient{}, callTimeout: 50 * time.Millisecond}
	_, err := r.FindByCapability(context.Background(), "summarize")
	if err == nil {
		t.Fatal("expected error when registry hangs past deadline")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded in chain, got: %v", err)
	}
}
