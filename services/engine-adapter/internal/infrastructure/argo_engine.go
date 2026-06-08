// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"context"
	"errors"
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

	// argoWatchPollInterval is the polling interval used by Watch.
	argoWatchPollInterval = 2 * time.Second
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

// GetStatus retrieves the current run metadata for the given runID by querying
// the Argo Workflows REST API. Argo WorkflowStatus.Phase values are mapped to
// domain.WorkflowStatus constants.
//
// Returns domain.ErrExecutionNotFound if the workflow does not exist.
func (e *ArgoEngine) GetStatus(ctx context.Context, runID string) (*domain.WorkflowRun, error) {
	if runID == "" {
		return nil, fmt.Errorf("engine-adapter(argo): GetStatus: runID must not be empty")
	}

	wf, err := e.client.GetWorkflow(ctx, e.cfg.Namespace, runID)
	if err != nil {
		if errors.Is(err, errArgoNotFound) {
			return nil, fmt.Errorf("engine-adapter(argo): GetStatus %q: %w", runID, domain.ErrExecutionNotFound)
		}
		return nil, fmt.Errorf("engine-adapter(argo): GetStatus %q: %w", runID, err)
	}

	run := &domain.WorkflowRun{
		RunID:        runID,
		WorkflowID:   wf.Metadata.Name,
		Namespace:    wf.Metadata.Namespace,
		Engine:       argoEngineName,
		Labels:       wf.Metadata.Labels,
		CurrentState: wf.Status.Phase,
		Status:       mapArgoPhase(wf.Status.Phase),
	}

	if wf.Status.StartedAt != "" {
		if t, err := time.Parse(time.RFC3339, wf.Status.StartedAt); err == nil {
			run.StartedAt = t
		}
	}
	if wf.Status.FinishedAt != "" {
		if t, err := time.Parse(time.RFC3339, wf.Status.FinishedAt); err == nil {
			run.FinishedAt = t
		}
	}

	return run, nil
}

// Cancel requests cancellation of a running workflow by deleting the Argo Workflow
// resource, which causes Argo to terminate all running pods.
//
// Returns domain.ErrExecutionNotFound if the workflow does not exist.
// Returns domain.ErrTerminalState if the workflow has already reached a terminal phase.
func (e *ArgoEngine) Cancel(ctx context.Context, runID, _ string) error {
	if runID == "" {
		return fmt.Errorf("engine-adapter(argo): Cancel: runID must not be empty")
	}

	// Fetch current status first so we can guard against terminal-state cancels.
	wf, err := e.client.GetWorkflow(ctx, e.cfg.Namespace, runID)
	if err != nil {
		if errors.Is(err, errArgoNotFound) {
			return fmt.Errorf("engine-adapter(argo): Cancel %q: %w", runID, domain.ErrExecutionNotFound)
		}
		return fmt.Errorf("engine-adapter(argo): Cancel %q: %w", runID, err)
	}

	if mapArgoPhase(wf.Status.Phase).IsTerminal() {
		return fmt.Errorf("engine-adapter(argo): Cancel %q: %w (phase=%s)",
			runID, domain.ErrTerminalState, wf.Status.Phase)
	}

	if err := e.client.DeleteWorkflow(ctx, e.cfg.Namespace, runID); err != nil {
		if errors.Is(err, errArgoNotFound) {
			// Race: workflow was deleted between GetWorkflow and DeleteWorkflow.
			return fmt.Errorf("engine-adapter(argo): Cancel %q: %w", runID, domain.ErrExecutionNotFound)
		}
		return fmt.Errorf("engine-adapter(argo): Cancel %q: %w", runID, err)
	}
	return nil
}

// Watch polls the Argo Workflows API until the workflow reaches a terminal state
// or ctx is cancelled, calling send for each observed status transition.
// send is called at least once with a terminal-status event before Watch returns nil.
//
// Returns domain.ErrExecutionNotFound if the workflow does not exist on the first poll.
func (e *ArgoEngine) Watch(ctx context.Context, runID string, send func(*domain.WorkflowEvent) error) error {
	if runID == "" {
		return fmt.Errorf("engine-adapter(argo): Watch: runID must not be empty")
	}

	var lastPhase string

	for {
		wf, err := e.client.GetWorkflow(ctx, e.cfg.Namespace, runID)
		if err != nil {
			if errors.Is(err, errArgoNotFound) {
				return fmt.Errorf("engine-adapter(argo): Watch %q: %w", runID, domain.ErrExecutionNotFound)
			}
			return fmt.Errorf("engine-adapter(argo): Watch %q: %w", runID, err)
		}

		currentPhase := wf.Status.Phase
		status := mapArgoPhase(currentPhase)

		if currentPhase != lastPhase {
			event := &domain.WorkflowEvent{
				RunID:     runID,
				EventType: "status.changed",
				FromState: lastPhase,
				ToState:   currentPhase,
				Status:    status,
				Timestamp: time.Now(),
			}
			if err := send(event); err != nil {
				return fmt.Errorf("engine-adapter(argo): Watch %q: send: %w", runID, err)
			}
			lastPhase = currentPhase
		}

		if status.IsTerminal() {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("engine-adapter(argo): Watch %q: %w", runID, ctx.Err())
		case <-time.After(argoWatchPollInterval):
		}
	}
}

// mapArgoPhase converts an Argo WorkflowStatus.Phase string to a domain.WorkflowStatus.
// Unknown or empty phases map to WorkflowStatusPending.
func mapArgoPhase(phase string) domain.WorkflowStatus {
	switch phase {
	case ArgoPhasePending, "":
		return domain.WorkflowStatusPending
	case ArgoPhaseRunning:
		return domain.WorkflowStatusRunning
	case ArgoPhaseSucceeded:
		return domain.WorkflowStatusCompleted
	case ArgoPhaseFailed, ArgoPhaseError:
		return domain.WorkflowStatusFailed
	case ArgoPhaseSkipped:
		return domain.WorkflowStatusCancelled
	default:
		return domain.WorkflowStatusPending
	}
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
