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
	nats "github.com/nats-io/nats.go"

	"github.com/zynax-io/zynax/libs/zynaxevents"
	"github.com/zynax-io/zynax/libs/zynaxobs"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// GatewayClients implements domain.CompilerPort and domain.EnginePort using
// gRPC connections to downstream services. It no longer dials the
// AgentRegistryService: push registration is retired (ADR-039) — agent identity
// is the Agent custom resource — so the gateway carries no registry client.
type GatewayClients struct {
	compiler    zynaxv1.WorkflowCompilerServiceClient
	engine      zynaxv1.EngineAdapterServiceClient
	events      *zynaxevents.Client
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

// NewGatewayClients dials the two downstream gRPC services (compiler, engine)
// and connects the shared JetStream events client (ADR-046 — eventing is no
// longer a gRPC peer). The AgentRegistryService is not dialled: push
// registration is retired (ADR-039). callTimeout is applied as a per-call
// deadline on every unary RPC (streaming Watch excluded). tlsCertFile,
// tlsKeyFile, tlsCAFile are paths to PEM files for mTLS; pass empty strings to
// fall back to insecure credentials (dev/test). The returned cleanup closes
// everything and must be deferred.
func NewGatewayClients(compilerAddr, engineAddr, natsURL string, callTimeout time.Duration, tlsCertFile, tlsKeyFile, tlsCAFile, eventsTLSCert, eventsTLSKey, eventsTLSCA string) (*GatewayClients, func(), error) {
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
	// Direct JetStream (ADR-046): RetryOnFailedConnect keeps startup
	// broker-independent — the old gRPC dial was lazy, and a NATS-less
	// profile (ADR-041 lite) must still boot; the /logs event merge is
	// best-effort until the broker is reachable.
	eventsOpts := []nats.Option{nats.RetryOnFailedConnect(true), nats.MaxReconnects(-1)}
	if eventsTLSCert != "" {
		// Dial with the gateway's cert-manager identity (verify_and_map,
		// ADR-046 Decision #4) — decoupled from the gRPC TLS profile.
		eventsOpts = append(eventsOpts, zynaxevents.TLSIdentity(eventsTLSCert, eventsTLSKey, eventsTLSCA)...)
	}
	events, err := zynaxevents.New(natsURL, eventsOpts...)
	if err != nil {
		_ = compConn.Close()
		_ = engConn.Close()
		return nil, func() {}, fmt.Errorf("api-gateway: events client: %w", err)
	}
	c := &GatewayClients{
		compiler:    zynaxv1.NewWorkflowCompilerServiceClient(compConn),
		engine:      zynaxv1.NewEngineAdapterServiceClient(engConn),
		events:      events,
		conns:       []*grpc.ClientConn{compConn, engConn},
		callTimeout: callTimeout,
	}
	cleanup := func() {
		_ = compConn.Close()
		_ = engConn.Close()
		events.Close()
	}
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
// workflow-scoped durable JetStream subscription through the shared events
// client (ADR-046 — the Subscribe→REST bridge is in-process now) and bridges
// each delivered CloudEvent to a domain.WatchEvent, keeping broker types out
// of the domain layer. The subscriber_id is derived from the workflow ID plus
// a monotonic suffix so concurrent followers of the same run do not collide.
// The channel closes on ctx cancel or on the run's terminal lifecycle event
// (the library's workflow-scoped terminal-close) — the same stream-end
// semantics the facade enforced server-side.
func (c *GatewayClients) SubscribeWorkflowEvents(ctx context.Context, workflowID string, send func(domain.WatchEvent) error) error {
	ch, err := c.events.Subscribe(ctx, zynaxevents.SubscribeRequest{
		SubscriberID: fmt.Sprintf("api-gateway-logs-%s-%d", workflowID, time.Now().UnixNano()),
		TypePattern:  capabilityEventPattern,
		WorkflowID:   workflowID,
	})
	if err != nil {
		return fmt.Errorf("api-gateway: events subscribe: %w", err)
	}
	for ce := range ch {
		we := domain.WatchEvent{
			RunID:     ce.WorkflowID,
			EventType: ce.Type,
			Status:    "capability_event",
			Payload:   string(ce.Data),
		}
		// The wire envelope has never carried a time attribute (the facade
		// dropped it before marshalling), so Timestamp stays empty exactly
		// as before.
		if err := send(we); err != nil {
			return err
		}
	}
	return nil
}

// eventSource is the CloudEvent `source` attribute stamped on every event the
// gateway injects on behalf of a CLI caller (CloudEvents spec: a URI-reference
// identifying the producer).
const eventSource = "/zynax/api-gateway/cli"

// cloudEventSpecVersion is the CloudEvents spec version the gateway emits.
const cloudEventSpecVersion = "1.0"

// PublishEvent implements domain.EventBusPort. It wraps the domain event in a
// CloudEvent envelope — filling the required id, source, and specversion
// attributes — and publishes straight to JetStream (ADR-046). The
// STREAM:sequence composite id is returned to the caller, exactly as the
// facade returned it.
func (c *GatewayClients) PublishEvent(ctx context.Context, ev domain.EventPublish) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	eventID, err := c.events.Publish(callCtx, zynaxevents.CloudEvent{
		ID:          uuid.NewString(),
		Source:      eventSource,
		SpecVersion: cloudEventSpecVersion,
		Type:        ev.Type,
		Time:        time.Now(),
		Data:        ev.Data,
		WorkflowID:  ev.RunID,
		RunID:       ev.RunID,
	})
	if err != nil {
		return "", fmt.Errorf("api-gateway: event publish: %w", err)
	}
	return eventID, nil
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
