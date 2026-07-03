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

	"github.com/google/uuid"
	"github.com/zynax-io/zynax/libs/zynaxobs"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
)

// GatewayClients implements domain.CompilerPort, domain.EnginePort, and
// domain.RegistryPort using gRPC connections to downstream services.
type GatewayClients struct {
	compiler    zynaxv1.WorkflowCompilerServiceClient
	engine      zynaxv1.EngineAdapterServiceClient
	registry    zynaxv1.AgentRegistryServiceClient
	eventbus    zynaxv1.EventBusServiceClient
	conns       []*grpc.ClientConn
	callTimeout time.Duration
}

// capabilityEventPattern subscribes to the task-broker task-lifecycle stream,
// whose ".completed" events carry the capability result_payload surfaced by
// `zynax logs`/`zynax result`. It MUST resolve (via StreamSubjectFromPattern) to
// the concrete JetStream stream that owns those events: a bare "**" collapses to
// the subject "x" and lands on a nonexistent stream, so no capability event is
// ever delivered. The event-bus filters further by the workflow_id scope on the
// SubscribeRequest.
const capabilityEventPattern = "zynax.v1.task-broker.task.**"

// ConnectionsReady returns false if any downstream gRPC connection is in
// TRANSIENT_FAILURE or SHUTDOWN state. Used by the /readyz probe handler.
func (c *GatewayClients) ConnectionsReady() bool {
	for _, conn := range c.conns {
		s := conn.GetState()
		if s == connectivity.TransientFailure || s == connectivity.Shutdown {
			return false
		}
	}
	return true
}

// NewGatewayClients dials all three downstream gRPC services. callTimeout is
// applied as a per-call deadline on every unary RPC (streaming Watch excluded).
// tlsCertFile, tlsKeyFile, tlsCAFile are paths to PEM files for mTLS; pass empty
// strings to fall back to insecure credentials (dev/test).
// The returned cleanup function closes all connections and must be deferred by the caller.
func NewGatewayClients(compilerAddr, engineAddr, registryAddr, eventBusAddr string, callTimeout time.Duration, tlsCertFile, tlsKeyFile, tlsCAFile string) (*GatewayClients, func(), error) {
	creds, err := tlsCreds(tlsCertFile, tlsKeyFile, tlsCAFile)
	if err != nil {
		return nil, func() {}, fmt.Errorf("api-gateway: tls credentials: %w", err)
	}
	tracingUnary, tracingStream := zynaxobs.TracingClientInterceptors()
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithStatsHandler(zynaxobs.TracingClientHandler()),
		grpc.WithChainUnaryInterceptor(tracingUnary, requestIDUnaryInterceptor),
		grpc.WithChainStreamInterceptor(tracingStream, requestIDStreamInterceptor),
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
	busConn, err := grpc.NewClient(eventBusAddr, dialOpts...)
	if err != nil {
		_ = compConn.Close()
		_ = engConn.Close()
		_ = regConn.Close()
		return nil, func() {}, fmt.Errorf("api-gateway: event-bus dial: %w", err)
	}
	c := &GatewayClients{
		compiler:    zynaxv1.NewWorkflowCompilerServiceClient(compConn),
		engine:      zynaxv1.NewEngineAdapterServiceClient(engConn),
		registry:    zynaxv1.NewAgentRegistryServiceClient(regConn),
		eventbus:    zynaxv1.NewEventBusServiceClient(busConn),
		conns:       []*grpc.ClientConn{compConn, engConn, regConn, busConn},
		callTimeout: callTimeout,
	}
	cleanup := func() { _ = compConn.Close(); _ = engConn.Close(); _ = regConn.Close(); _ = busConn.Close() }
	return c, cleanup, nil
}

// CompileWorkflow implements domain.CompilerPort.
// When the compiler returns codes.InvalidArgument the gRPC error message is
// surfaced as a CompileError so the handler can return a structured 422.
func (c *GatewayClients) CompileWorkflow(ctx context.Context, manifestYAML []byte, namespace string, dryRun bool) (domain.CompileResult, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
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
// namespace is set explicitly on the request so the engine-adapter can enforce
// namespace-scoped capability routing without re-parsing the IR bytes.
func (c *GatewayClients) SubmitWorkflow(ctx context.Context, irBytes []byte, engineHint, workflowID, namespace string) (string, error) {
	ir := &zynaxv1.WorkflowIR{}
	if err := proto.Unmarshal(irBytes, ir); err != nil {
		return "", fmt.Errorf("api-gateway: unmarshal IR: %w", err)
	}
	ir.WorkflowId = workflowID
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	resp, err := c.engine.SubmitWorkflow(callCtx, &zynaxv1.SubmitWorkflowRequest{
		WorkflowIr: ir,
		EngineHint: engineHint,
		Namespace:  namespace,
	})
	if err != nil {
		return "", mapEngineGRPCError(err)
	}
	return resp.GetRunId(), nil
}

// GetWorkflowStatus implements domain.EnginePort.
func (c *GatewayClients) GetWorkflowStatus(ctx context.Context, runID string) (domain.WorkflowRunSummary, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
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
		Outputs:      run.GetOutputs(),
	}, nil
}

// CancelWorkflow implements domain.EnginePort.
func (c *GatewayClients) CancelWorkflow(ctx context.Context, runID string) error {
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
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

// SubscribeWorkflowEvents implements domain.EventBusPort. It opens a
// workflow-scoped EventBusService.Subscribe stream and bridges each delivered
// CloudEvent to a domain.WatchEvent, keeping gRPC and proto types out of the
// domain layer. The subscriber_id is derived from the workflow ID plus a
// monotonic suffix so concurrent followers of the same run do not collide.
func (c *GatewayClients) SubscribeWorkflowEvents(ctx context.Context, workflowID string, send func(domain.WatchEvent) error) error {
	req := &zynaxv1.SubscribeRequest{
		SubscriberId: fmt.Sprintf("api-gateway-logs-%s-%d", workflowID, time.Now().UnixNano()),
		TypePattern:  capabilityEventPattern,
		WorkflowId:   workflowID,
	}
	stream, err := c.eventbus.Subscribe(ctx, req)
	if err != nil {
		return mapEngineGRPCError(err)
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return mapEngineGRPCError(err)
		}
		ce := resp.GetEvent()
		if ce == nil {
			continue
		}
		we := domain.WatchEvent{
			RunID:     ce.GetWorkflowId(),
			EventType: ce.GetType(),
			Status:    "capability_event",
			Payload:   string(ce.GetData()),
		}
		if ts := ce.GetTime(); ts != nil {
			we.Timestamp = ts.AsTime().Format(time.RFC3339)
		}
		if err := send(we); err != nil {
			return err
		}
	}
}

// eventSource is the CloudEvent `source` attribute stamped on every event the
// gateway injects on behalf of a CLI caller (CloudEvents spec: a URI-reference
// identifying the producer).
const eventSource = "/zynax/api-gateway/cli"

// cloudEventSpecVersion is the CloudEvents spec version the gateway emits.
const cloudEventSpecVersion = "1.0"

// PublishEvent implements domain.EventBusPort. It wraps the domain event in a
// CloudEvent envelope — filling the bus-required id, source, specversion, and
// time attributes — and calls EventBusService.Publish. The bus-assigned
// event_id is returned to the caller.
func (c *GatewayClients) PublishEvent(ctx context.Context, ev domain.EventPublish) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	resp, err := c.eventbus.Publish(callCtx, &zynaxv1.PublishRequest{
		Event: &zynaxv1.CloudEvent{
			Id:          uuid.NewString(),
			Source:      eventSource,
			Specversion: cloudEventSpecVersion,
			Type:        ev.Type,
			Time:        timestamppb.Now(),
			Data:        ev.Data,
			WorkflowId:  ev.RunID,
			RunId:       ev.RunID,
		},
	})
	if err != nil {
		return "", mapEngineGRPCError(err)
	}
	return resp.GetEventId(), nil
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
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	resp, err := c.registry.RegisterAgent( //nolint:staticcheck // SA1019: dead code behind domain.ErrAgentDefRetired; deleted with the M9 hard RPC removal (ADR-039).
		callCtx, req)
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

// ── gRPC client correlation interceptors ──────────────────────────────────

// gRPC metadata keys carrying the correlation context to downstream hops. These
// mirror the X-Request-ID / X-Namespace HTTP headers the gateway accepts and are
// distinct from the W3C traceparent, which the tracing interceptors propagate
// separately (canvas C.2).
const (
	requestIDMetaKey = "request-id"
	namespaceMetaKey = "x-namespace"
)

// withCorrelationMetadata appends the request-id and namespace held on ctx to the
// outgoing gRPC metadata, skipping any identifier that is unset. Only correlation
// ids are attached — never auth tokens or secrets (canvas C safeguard).
func withCorrelationMetadata(ctx context.Context) context.Context {
	if id := domain.RequestIDFromContext(ctx); id != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, requestIDMetaKey, id)
	}
	if ns := domain.NamespaceFromContext(ctx); ns != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, namespaceMetaKey, ns)
	}
	return ctx
}

func requestIDUnaryInterceptor(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	return invoker(withCorrelationMetadata(ctx), method, req, reply, cc, opts...)
}

func requestIDStreamInterceptor(
	ctx context.Context,
	desc *grpc.StreamDesc,
	cc *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	opts ...grpc.CallOption,
) (grpc.ClientStream, error) {
	return streamer(withCorrelationMetadata(ctx), desc, cc, method, opts...)
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
		result.Namespace = ir.GetNamespace()
	}
	return result
}
