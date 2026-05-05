// SPDX-License-Identifier: Apache-2.0

// Package infrastructure implements the domain ports using real gRPC clients.
// Only this package may import gRPC SDK types or proto-generated stubs.
package infrastructure

import (
	"context"
	"fmt"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// GatewayClients implements domain.CompilerPort and domain.EnginePort using
// gRPC connections to WorkflowCompilerService and EngineAdapterService.
type GatewayClients struct {
	compiler zynaxv1.WorkflowCompilerServiceClient
	engine   zynaxv1.EngineAdapterServiceClient
}

// NewGatewayClients dials both downstream gRPC services. The returned cleanup
// function closes both connections and must be deferred by the caller.
func NewGatewayClients(compilerAddr, engineAddr string) (*GatewayClients, func(), error) {
	compConn, err := grpc.NewClient(compilerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, func() {}, fmt.Errorf("api-gateway: compiler dial: %w", err)
	}
	engConn, err := grpc.NewClient(engineAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = compConn.Close()
		return nil, func() {}, fmt.Errorf("api-gateway: engine dial: %w", err)
	}
	c := &GatewayClients{
		compiler: zynaxv1.NewWorkflowCompilerServiceClient(compConn),
		engine:   zynaxv1.NewEngineAdapterServiceClient(engConn),
	}
	return c, func() { _ = compConn.Close(); _ = engConn.Close() }, nil
}

// CompileWorkflow implements domain.CompilerPort.
// When the compiler returns codes.InvalidArgument the gRPC error message is
// surfaced as a CompileError so the handler can return a structured 422.
func (c *GatewayClients) CompileWorkflow(ctx context.Context, manifestYAML []byte, namespace string, dryRun bool) (domain.CompileResult, error) {
	resp, err := c.compiler.CompileWorkflow(ctx, &zynaxv1.CompileWorkflowRequest{
		ManifestYaml: manifestYAML,
		Namespace:    namespace,
		DryRun:       dryRun,
	})
	if err != nil {
		return mapCompilerGRPCError(err)
	}
	return compileResultFromProto(resp), nil
}

// SubmitWorkflow implements domain.EnginePort.
func (c *GatewayClients) SubmitWorkflow(ctx context.Context, irBytes []byte, engineHint string) (string, error) {
	ir := &zynaxv1.WorkflowIR{}
	if err := proto.Unmarshal(irBytes, ir); err != nil {
		return "", fmt.Errorf("api-gateway: unmarshal IR: %w", err)
	}
	resp, err := c.engine.SubmitWorkflow(ctx, &zynaxv1.SubmitWorkflowRequest{
		WorkflowIr: ir,
		EngineHint: engineHint,
	})
	if err != nil {
		return "", mapEngineGRPCError(err)
	}
	return resp.GetRunId(), nil
}

// GetWorkflowStatus implements domain.EnginePort.
func (c *GatewayClients) GetWorkflowStatus(ctx context.Context, runID string) (domain.WorkflowRunSummary, error) {
	run, err := c.engine.GetWorkflowStatus(ctx, &zynaxv1.GetWorkflowStatusRequest{RunId: runID})
	if err != nil {
		return domain.WorkflowRunSummary{}, mapEngineGRPCError(err)
	}
	return domain.WorkflowRunSummary{
		RunID:        run.GetRunId(),
		WorkflowID:   run.GetWorkflowId(),
		Status:       run.GetStatus().String(),
		CurrentState: run.GetCurrentState(),
	}, nil
}

// ── error mapping ─────────────────────────────────────────────────────────

func mapCompilerGRPCError(err error) (domain.CompileResult, error) {
	st, _ := status.FromError(err)
	if st.Code() == codes.InvalidArgument {
		return domain.CompileResult{
			Errors: []domain.CompileError{{Code: "COMPILER_ERROR", Message: st.Message()}},
		}, nil
	}
	return domain.CompileResult{}, fmt.Errorf("api-gateway: compiler: %w", err)
}

func mapEngineGRPCError(err error) error {
	st, _ := status.FromError(err)
	switch st.Code() {
	case codes.NotFound:
		return fmt.Errorf("api-gateway: %w", domain.ErrNotFound)
	case codes.Unavailable:
		return fmt.Errorf("api-gateway: %w", domain.ErrEngineUnavailable)
	default:
		return fmt.Errorf("api-gateway: engine: %w", err)
	}
}

// ── proto conversion ──────────────────────────────────────────────────────

func compileResultFromProto(resp *zynaxv1.CompileWorkflowResponse) domain.CompileResult {
	result := domain.CompileResult{Warnings: resp.GetWarnings()}
	for _, e := range resp.GetErrors() {
		result.Errors = append(result.Errors, domain.CompileError{
			Code:    e.GetCode().String(),
			Message: e.GetMessage(),
			Line:    e.GetLineNumber(),
		})
	}
	if ir := resp.GetWorkflowIr(); ir != nil {
		result.IRBytes, _ = proto.Marshal(ir)
	}
	return result
}
