// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
)

const (
	argoEngineName         = "argo"
	argoWorkflowAPIVersion = "argoproj.io/v1alpha1"
	argoWorkflowKind       = "Workflow"

	// argoIRPayloadParam is the WorkflowTemplate parameter name that receives
	// the serialised WorkflowIR JSON. The WorkflowTemplate in the cluster is
	// expected to expose a parameter with this name.
	argoIRPayloadParam = "workflow-ir"
)

// ArgoConfig holds the runtime configuration for ArgoEngine. All values are
// read from environment variables at startup — never hardcoded (ADR-015).
type ArgoConfig struct {
	// Namespace is the Kubernetes namespace where Argo Workflow resources are created.
	Namespace string

	// WorkflowTemplateRef is the name of the Argo WorkflowTemplate to instantiate.
	WorkflowTemplateRef string

	// ServiceAccountName is the Kubernetes ServiceAccount bound to submitted workflows.
	// May be empty if the cluster default is acceptable.
	ServiceAccountName string
}

// ArgoEngine implements domain.WorkflowEngine backed by the Argo Workflows REST API.
// Selected when ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE=argo (ADR-015).
//
// ArgoEngine only implements Submit and Signal in this step (O2 / issue #796).
// GetStatus, Cancel, and Watch are implemented in O3 / issue #797.
type ArgoEngine struct {
	client ArgoClient
	cfg    ArgoConfig
}

// NewArgoEngine constructs an ArgoEngine with an injected ArgoClient and config.
// The client is always injected — never instantiated inside this constructor —
// so that unit tests can supply a mock without a live Argo server.
func NewArgoEngine(client ArgoClient, cfg ArgoConfig) *ArgoEngine {
	return &ArgoEngine{client: client, cfg: cfg}
}

// Submit translates a WorkflowIR into an Argo Workflow resource and submits it
// to the Argo server via the injected ArgoClient.
//
// The WorkflowIR is serialised to JSON via protojson and passed as a WorkflowTemplate
// parameter (argoIRPayloadParam). The WorkflowID from the IR is used as the Argo
// workflow name so that run IDs are stable and human-readable.
func (e *ArgoEngine) Submit(ctx context.Context, ir *zynaxv1.WorkflowIR, labels map[string]string) (*domain.WorkflowRun, error) {
	if ir == nil {
		return nil, fmt.Errorf("engine-adapter(argo): WorkflowIR must not be nil")
	}
	workflowName := ir.GetWorkflowId()
	if workflowName == "" {
		return nil, fmt.Errorf("engine-adapter(argo): WorkflowIR.workflow_id must not be empty")
	}

	// Serialise the IR to JSON so it can be passed as a WorkflowTemplate parameter.
	irJSON, err := workflowIRToJSON(ir)
	if err != nil {
		return nil, fmt.Errorf("engine-adapter(argo): serialise WorkflowIR: %w", err)
	}

	wf := buildArgoWorkflow(workflowName, irJSON, labels, e.cfg)

	if err := e.client.SubmitWorkflow(ctx, e.cfg.Namespace, wf); err != nil {
		return nil, fmt.Errorf("engine-adapter(argo): submit %q: %w", workflowName, err)
	}

	return &domain.WorkflowRun{
		RunID:       workflowName,
		WorkflowID:  workflowName,
		Namespace:   e.cfg.Namespace,
		Status:      domain.WorkflowStatusPending,
		Engine:      argoEngineName,
		Labels:      labels,
		SubmittedAt: time.Now(),
	}, nil
}

// Signal delivers an external event to a running Argo workflow.
// The runID and eventType are combined into a discriminator so that
// WorkflowEventBindings in the cluster can route events to the correct
// workflow instance and event handler.
func (e *ArgoEngine) Signal(ctx context.Context, runID, eventType string, payload []byte) error {
	if runID == "" {
		return fmt.Errorf("engine-adapter(argo): Signal: runID must not be empty")
	}
	if eventType == "" {
		return fmt.Errorf("engine-adapter(argo): Signal: eventType must not be empty")
	}

	// The discriminator encodes both runID and eventType so the WorkflowEventBinding
	// selector can match on both dimensions.
	discriminator := runID + "." + eventType

	if err := e.client.SendEvent(ctx, e.cfg.Namespace, discriminator, payload); err != nil {
		return fmt.Errorf("engine-adapter(argo): signal run %q event %q: %w", runID, eventType, err)
	}
	return nil
}

// Cancel, GetStatus, and Watch are unimplemented in this step (O2).
// They are implemented in O3 / issue #797 to keep PRs atomic.

// Cancel is not yet implemented — returns an error indicating the method
// will be available after issue #797 is merged.
func (e *ArgoEngine) Cancel(_ context.Context, runID, _ string) error {
	return fmt.Errorf("engine-adapter(argo): Cancel not yet implemented (issue #797): run %q", runID)
}

// GetStatus is not yet implemented — returns ErrExecutionNotFound so callers
// receive a sensible gRPC NOT_FOUND rather than a panic.
func (e *ArgoEngine) GetStatus(_ context.Context, runID string) (*domain.WorkflowRun, error) {
	return nil, fmt.Errorf("engine-adapter(argo): GetStatus not yet implemented (issue #797): %w — run %q",
		domain.ErrExecutionNotFound, runID)
}

// Watch is not yet implemented.
func (e *ArgoEngine) Watch(_ context.Context, runID string, _ func(*domain.WorkflowEvent) error) error {
	return fmt.Errorf("engine-adapter(argo): Watch not yet implemented (issue #797): run %q", runID)
}

// buildArgoWorkflow constructs the Argo Workflow resource from a WorkflowIR
// and the engine configuration.
func buildArgoWorkflow(name, irJSON string, labels map[string]string, cfg ArgoConfig) *ArgoWorkflow {
	wf := &ArgoWorkflow{
		APIVersion: argoWorkflowAPIVersion,
		Kind:       argoWorkflowKind,
		Metadata: ArgoObjectMeta{
			Name:      name,
			Namespace: cfg.Namespace,
			Labels:    labels,
		},
		Spec: ArgoWorkflowSpec{
			Arguments: &ArgoArguments{
				Parameters: []ArgoParameter{
					{Name: argoIRPayloadParam, Value: irJSON},
				},
			},
		},
	}

	if cfg.WorkflowTemplateRef != "" {
		wf.Spec.WorkflowTemplateRef = &ArgoWorkflowTemplateRef{Name: cfg.WorkflowTemplateRef}
	}
	if cfg.ServiceAccountName != "" {
		wf.Spec.ServiceAccountName = cfg.ServiceAccountName
	}

	return wf
}

// workflowIRToJSON serialises a WorkflowIR proto message to a JSON string using
// the canonical protojson encoding. This preserves all proto field names and
// enum values correctly for consumption inside Argo WorkflowTemplates.
func workflowIRToJSON(ir *zynaxv1.WorkflowIR) (string, error) {
	opts := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}
	data, err := opts.Marshal(ir)
	if err != nil {
		return "", fmt.Errorf("protojson marshal: %w", err)
	}
	return string(data), nil
}

// compile-time assertion: ArgoEngine satisfies domain.WorkflowEngine.
var _ domain.WorkflowEngine = (*ArgoEngine)(nil)
