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

// blockingCompiler hangs on CompileWorkflow until the context is cancelled,
// simulating a downstream service that does not respond within the deadline.
type blockingCompiler struct{}

func (b *blockingCompiler) CompileWorkflow(ctx context.Context, _ *zynaxv1.CompileWorkflowRequest, _ ...grpc.CallOption) (*zynaxv1.CompileWorkflowResponse, error) {
	<-ctx.Done()
	return nil, fmt.Errorf("blocking compiler: %w", ctx.Err())
}
func (b *blockingCompiler) ValidateManifest(_ context.Context, _ *zynaxv1.ValidateManifestRequest, _ ...grpc.CallOption) (*zynaxv1.ValidateManifestResponse, error) {
	return nil, nil
}
func (b *blockingCompiler) GetCompiledWorkflow(_ context.Context, _ *zynaxv1.GetCompiledWorkflowRequest, _ ...grpc.CallOption) (*zynaxv1.GetCompiledWorkflowResponse, error) {
	return nil, nil
}

func newTestGatewayClients(compiler zynaxv1.WorkflowCompilerServiceClient) *GatewayClients {
	return &GatewayClients{compiler: compiler}
}

func TestCompileWorkflow_DeadlineExceeded(t *testing.T) {
	old := grpcCallTimeout
	grpcCallTimeout = 50 * time.Millisecond
	defer func() { grpcCallTimeout = old }()

	c := newTestGatewayClients(&blockingCompiler{})
	_, err := c.CompileWorkflow(context.Background(), []byte("manifest: {}"), "default", false)
	if err == nil {
		t.Fatal("expected error when compiler hangs past deadline")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded in chain, got: %v", err)
	}
}
