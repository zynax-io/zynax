// SPDX-License-Identifier: Apache-2.0

// Package infrastructure implements the domain ports using real gRPC clients.
// Only this package may import gRPC SDK types or proto-generated stubs.
package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

// grpcCallTimeout is the per-call deadline for all outgoing unary gRPC requests.
// Streaming calls (WatchWorkflow) are excluded — they are bounded by the HTTP
// request context instead.
var grpcCallTimeout = 30 * time.Second

// GatewayClients implements domain.CompilerPort, domain.EnginePort, and
// domain.RegistryPort using gRPC connections to downstream services.
type GatewayClients struct {
	compiler zynaxv1.WorkflowCompilerServiceClient
	engine   zynaxv1.EngineAdapterServiceClient
	registry zynaxv1.AgentRegistryServiceClient
}

// NewGatewayClients dials all three downstream gRPC services. The returned
// cleanup function closes all connections and must be deferred by the caller.
func NewGatewayClients(compilerAddr, engineAddr, registryAddr string) (*GatewayClients, func(), error) {
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(requestIDUnaryInterceptor),
		grpc.WithStreamInterceptor(requestIDStreamInterceptor),
	}
	compConn, err := grpc.NewClient(compilerAddr, dialOpts...)
	if err != nil {
		return nil, func() {}, fmt.Errorf("api-gateway: compiler dial: %w", err)
	}
	engConn, err := grpc.NewClient(engineAddr, dialOpts...)
	if err != nil {
		_ = compConn.Close()
		return nil, func() {}, fmt.Errorf("api-gateway: engine dial: %w", err)
	}
	regConn, err := grpc.NewClient(registryAddr, dialOpts...)
	if err != nil {
		_ = compConn.Close()
		_ = engConn.Close()
		return nil, func() {}, fmt.Errorf("api-gateway: registry dial: %w", err)
	}
	c := &GatewayClients{
		compiler: zynaxv1.NewWorkflowCompilerServiceClient(compConn),
		engine:   zynaxv1.NewEngineAdapterServiceClient(engConn),
		registry: zynaxv1.NewAgentRegistryServiceClient(regConn),
	}
	cleanup := func() { _ = compConn.Close(); _ = engConn.Close(); _ = regConn.Close() }
	return c, cleanup, nil
}

// CompileWorkflow implements domain.CompilerPort.
// When the compiler returns codes.InvalidArgument the gRPC error message is
// surfaced as a CompileError so the handler can return a structured 422.
func (c *GatewayClients) CompileWorkflow(ctx context.Context, manifestYAML []byte, namespace string, dryRun bool) (domain.CompileResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
	defer cancel()
	resp, err := c.compiler.CompileWorkflow(callCtx, &zynaxv1.CompileWorkflowRequest{
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
// workflowID overrides the compiler-assigned WorkflowId in the IR so that
// Temporal uses the hash-derived deterministic identifier for deduplication.
func (c *GatewayClients) SubmitWorkflow(ctx context.Context, irBytes []byte, engineHint, workflowID string) (string, error) {
	ir := &zynaxv1.WorkflowIR{}
	if err := proto.Unmarshal(irBytes, ir); err != nil {
		return "", fmt.Errorf("api-gateway: unmarshal IR: %w", err)
	}
	ir.WorkflowId = workflowID
	callCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
	defer cancel()
	resp, err := c.engine.SubmitWorkflow(callCtx, &zynaxv1.SubmitWorkflowRequest{
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
	callCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
	defer cancel()
	run, err := c.engine.GetWorkflowStatus(callCtx, &zynaxv1.GetWorkflowStatusRequest{RunId: runID})
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

// CancelWorkflow implements domain.EnginePort.
func (c *GatewayClients) CancelWorkflow(ctx context.Context, runID string) error {
	callCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
	defer cancel()
	_, err := c.engine.CancelWorkflow(callCtx, &zynaxv1.CancelWorkflowRequest{RunId: runID})
	if err != nil {
		return mapEngineGRPCError(err)
	}
	return nil
}

// WatchWorkflow implements domain.EnginePort. It bridges the gRPC server-stream
// to the callback pattern, keeping gRPC types out of the domain layer.
func (c *GatewayClients) WatchWorkflow(ctx context.Context, runID string, send func(domain.WatchEvent) error) error {
	stream, err := c.engine.WatchWorkflow(ctx, &zynaxv1.WatchWorkflowRequest{RunId: runID})
	if err != nil {
		return mapEngineGRPCError(err)
	}
	for {
		ev, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return mapEngineGRPCError(err)
		}
		we := domain.WatchEvent{
			RunID:     ev.GetRunId(),
			EventType: ev.GetEventType(),
			FromState: ev.GetFromState(),
			ToState:   ev.GetToState(),
			Status:    ev.GetStatus().String(),
			Payload:   string(ev.GetPayload()),
		}
		if ts := ev.GetTimestamp(); ts != nil {
			we.Timestamp = ts.AsTime().Format(time.RFC3339)
		}
		if err := send(we); err != nil {
			return err
		}
	}
}

// RegisterAgent implements domain.RegistryPort.
// The raw YAML is parsed here in the infrastructure layer; the domain never
// sees proto types or YAML-parsed structs (ADR-011, ADR-001).
func (c *GatewayClients) RegisterAgent(ctx context.Context, manifestYAML []byte, _ string) (domain.AgentRegistration, error) {
	var m agentDefManifest
	if err := yaml.Unmarshal(manifestYAML, &m); err != nil {
		return domain.AgentRegistration{}, fmt.Errorf("api-gateway: parse AgentDef: %w", err)
	}
	caps := make([]*zynaxv1.CapabilityDef, len(m.Spec.Capabilities))
	for i, cap := range m.Spec.Capabilities {
		caps[i] = &zynaxv1.CapabilityDef{Name: cap.Name, Description: cap.Description}
	}
	req := &zynaxv1.RegisterAgentRequest{
		Agent: &zynaxv1.AgentDef{
			AgentId:      m.Metadata.Name,
			Name:         m.Metadata.Name,
			Endpoint:     m.Spec.Endpoint,
			Capabilities: caps,
			Labels:       m.Metadata.Labels,
		},
	}
	callCtx, cancel := context.WithTimeout(ctx, grpcCallTimeout)
	defer cancel()
	resp, err := c.registry.RegisterAgent(callCtx, req)
	if err != nil {
		return domain.AgentRegistration{}, mapRegistryGRPCError(err)
	}
	return domain.AgentRegistration{AgentID: resp.GetAgentId()}, nil
}

// ── YAML manifest structs (infrastructure-private) ────────────────────────

type agentDefManifest struct {
	Metadata agentDefMetadata `yaml:"metadata"`
	Spec     agentDefSpec     `yaml:"spec"`
}

type agentDefMetadata struct {
	Name   string            `yaml:"name"`
	Labels map[string]string `yaml:"labels"`
}

type agentDefSpec struct {
	Endpoint     string           `yaml:"endpoint"`
	Capabilities []capabilitySpec `yaml:"capabilities"`
}

type capabilitySpec struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
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

func mapRegistryGRPCError(err error) error {
	st, _ := status.FromError(err)
	switch st.Code() {
	case codes.AlreadyExists:
		return fmt.Errorf("api-gateway: %w", domain.ErrAgentAlreadyExists)
	default:
		return fmt.Errorf("api-gateway: registry: %w", err)
	}
}

// ── gRPC client interceptors ──────────────────────────────────────────────

func requestIDUnaryInterceptor(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	if id := domain.RequestIDFromContext(ctx); id != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "request-id", id)
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

func requestIDStreamInterceptor(
	ctx context.Context,
	desc *grpc.StreamDesc,
	cc *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	if id := domain.RequestIDFromContext(ctx); id != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "request-id", id)
	}
	return streamer(ctx, desc, cc, method, opts...)
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
