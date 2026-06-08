// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"context"
	"errors"
	"testing"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
)

// stubArgoClient implements ArgoClient for unit tests. All fields are exported
// to allow table-driven tests to set expected errors without embedding logic.
type stubArgoClient struct {
	submitErr error
	sendErr   error

	// Captured arguments for assertion.
	lastSubmitNamespace string
	lastSubmitWorkflow  *ArgoWorkflow
	lastSendNamespace   string
	lastSendDiscrim     string
	lastSendPayload     []byte
}

func (s *stubArgoClient) SubmitWorkflow(_ context.Context, namespace string, wf *ArgoWorkflow) error {
	s.lastSubmitNamespace = namespace
	s.lastSubmitWorkflow = wf
	return s.submitErr
}

func (s *stubArgoClient) SendEvent(_ context.Context, namespace, discriminator string, payload []byte) error {
	s.lastSendNamespace = namespace
	s.lastSendDiscrim = discriminator
	s.lastSendPayload = payload
	return s.sendErr
}

const (
	testArgoNamespace    = "zynax-test"
	testWorkflowTemplate = "zynax-ir-runner"
	testServiceAccount   = "zynax-sa"
)

func defaultConfig() ArgoConfig {
	return ArgoConfig{
		Namespace:           testArgoNamespace,
		WorkflowTemplateRef: testWorkflowTemplate,
		ServiceAccountName:  testServiceAccount,
	}
}

func newTestArgoEngine(stub *stubArgoClient) *ArgoEngine {
	return NewArgoEngine(stub, defaultConfig())
}

// ─── Submit tests ────────────────────────────────────────────────────────────

func TestArgoEngine_Submit_Success(t *testing.T) {
	stub := &stubArgoClient{}
	engine := newTestArgoEngine(stub)

	ir := &zynaxv1.WorkflowIR{WorkflowId: "argo-wf-001", Name: "my-dag"}
	run, err := engine.Submit(context.Background(), ir, map[string]string{"env": "test"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run == nil {
		t.Fatal("expected non-nil WorkflowRun")
	}
	if run.RunID != "argo-wf-001" {
		t.Errorf("RunID = %q; want %q", run.RunID, "argo-wf-001")
	}
	if run.WorkflowID != "argo-wf-001" {
		t.Errorf("WorkflowID = %q; want %q", run.WorkflowID, "argo-wf-001")
	}
	if run.Status != domain.WorkflowStatusPending {
		t.Errorf("Status = %v; want Pending", run.Status)
	}
	if run.Engine != argoEngineName {
		t.Errorf("Engine = %q; want %q", run.Engine, argoEngineName)
	}
	if run.Namespace != testArgoNamespace {
		t.Errorf("Namespace = %q; want %q", run.Namespace, testArgoNamespace)
	}
	if run.Labels["env"] != "test" {
		t.Errorf("Labels not forwarded: %v", run.Labels)
	}
	if run.SubmittedAt.IsZero() {
		t.Error("SubmittedAt must be set")
	}
}

func TestArgoEngine_Submit_ForwardsToClient(t *testing.T) {
	stub := &stubArgoClient{}
	engine := newTestArgoEngine(stub)

	ir := &zynaxv1.WorkflowIR{WorkflowId: "argo-wf-002"}
	_, err := engine.Submit(context.Background(), ir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stub.lastSubmitNamespace != testArgoNamespace {
		t.Errorf("SubmitWorkflow called with namespace=%q; want %q", stub.lastSubmitNamespace, testArgoNamespace)
	}
	if stub.lastSubmitWorkflow == nil {
		t.Fatal("SubmitWorkflow was not called")
	}
	if stub.lastSubmitWorkflow.Metadata.Name != "argo-wf-002" {
		t.Errorf("workflow name = %q; want %q", stub.lastSubmitWorkflow.Metadata.Name, "argo-wf-002")
	}
	if stub.lastSubmitWorkflow.Spec.WorkflowTemplateRef == nil {
		t.Error("WorkflowTemplateRef must be set from config")
	} else if stub.lastSubmitWorkflow.Spec.WorkflowTemplateRef.Name != testWorkflowTemplate {
		t.Errorf("WorkflowTemplateRef.Name = %q; want %q",
			stub.lastSubmitWorkflow.Spec.WorkflowTemplateRef.Name, testWorkflowTemplate)
	}
	if stub.lastSubmitWorkflow.Spec.ServiceAccountName != testServiceAccount {
		t.Errorf("ServiceAccountName = %q; want %q", stub.lastSubmitWorkflow.Spec.ServiceAccountName, testServiceAccount)
	}
}

func TestArgoEngine_Submit_EmbeddsIRJSON(t *testing.T) {
	stub := &stubArgoClient{}
	engine := newTestArgoEngine(stub)

	ir := &zynaxv1.WorkflowIR{WorkflowId: "argo-wf-003", Name: "embed-test"}
	_, err := engine.Submit(context.Background(), ir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wf := stub.lastSubmitWorkflow
	if wf == nil || wf.Spec.Arguments == nil {
		t.Fatal("expected Spec.Arguments to be set")
	}
	params := wf.Spec.Arguments.Parameters
	if len(params) == 0 {
		t.Fatal("expected at least one parameter")
	}
	found := false
	for _, p := range params {
		if p.Name == argoIRPayloadParam {
			found = true
			if p.Value == "" {
				t.Error("IR JSON parameter value must not be empty")
			}
		}
	}
	if !found {
		t.Errorf("parameter %q not found in Spec.Arguments.Parameters", argoIRPayloadParam)
	}
}

func TestArgoEngine_Submit_NilIR(t *testing.T) {
	stub := &stubArgoClient{}
	engine := newTestArgoEngine(stub)

	_, err := engine.Submit(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for nil IR")
	}
}

func TestArgoEngine_Submit_EmptyWorkflowID(t *testing.T) {
	stub := &stubArgoClient{}
	engine := newTestArgoEngine(stub)

	_, err := engine.Submit(context.Background(), &zynaxv1.WorkflowIR{WorkflowId: ""}, nil)
	if err == nil {
		t.Fatal("expected error for empty workflow_id")
	}
}

func TestArgoEngine_Submit_ClientError(t *testing.T) {
	stub := &stubArgoClient{submitErr: errors.New("argo server unreachable")}
	engine := newTestArgoEngine(stub)

	_, err := engine.Submit(context.Background(), &zynaxv1.WorkflowIR{WorkflowId: "wf-err"}, nil)
	if err == nil {
		t.Fatal("expected error from client")
	}
	if !containsStr(err.Error(), "argo server unreachable") {
		t.Errorf("expected wrapped client error, got: %v", err)
	}
}

// ─── Signal tests ─────────────────────────────────────────────────────────────

func TestArgoEngine_Signal_Success(t *testing.T) {
	stub := &stubArgoClient{}
	engine := newTestArgoEngine(stub)

	err := engine.Signal(context.Background(), "argo-run-010", "review.approved", []byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stub.lastSendNamespace != testArgoNamespace {
		t.Errorf("SendEvent namespace = %q; want %q", stub.lastSendNamespace, testArgoNamespace)
	}
	wantDiscrim := "argo-run-010.review.approved"
	if stub.lastSendDiscrim != wantDiscrim {
		t.Errorf("discriminator = %q; want %q", stub.lastSendDiscrim, wantDiscrim)
	}
	if string(stub.lastSendPayload) != `{"ok":true}` {
		t.Errorf("payload = %q; want %q", stub.lastSendPayload, `{"ok":true}`)
	}
}

func TestArgoEngine_Signal_NilPayload(t *testing.T) {
	stub := &stubArgoClient{}
	engine := newTestArgoEngine(stub)

	err := engine.Signal(context.Background(), "argo-run-011", "start", nil)
	if err != nil {
		t.Fatalf("unexpected error with nil payload: %v", err)
	}
}

func TestArgoEngine_Signal_EmptyRunID(t *testing.T) {
	stub := &stubArgoClient{}
	engine := newTestArgoEngine(stub)

	err := engine.Signal(context.Background(), "", "start", nil)
	if err == nil {
		t.Fatal("expected error for empty runID")
	}
}

func TestArgoEngine_Signal_EmptyEventType(t *testing.T) {
	stub := &stubArgoClient{}
	engine := newTestArgoEngine(stub)

	err := engine.Signal(context.Background(), "argo-run-012", "", nil)
	if err == nil {
		t.Fatal("expected error for empty eventType")
	}
}

func TestArgoEngine_Signal_ClientError(t *testing.T) {
	stub := &stubArgoClient{sendErr: errors.New("event endpoint unavailable")}
	engine := newTestArgoEngine(stub)

	err := engine.Signal(context.Background(), "argo-run-013", "timeout", nil)
	if err == nil {
		t.Fatal("expected error from client")
	}
	if !containsStr(err.Error(), "event endpoint unavailable") {
		t.Errorf("expected wrapped client error, got: %v", err)
	}
}

// ─── buildArgoWorkflow helper tests ──────────────────────────────────────────

func TestBuildArgoWorkflow_APIVersionAndKind(t *testing.T) {
	wf := buildArgoWorkflow("test-wf", `{"workflow_id":"test-wf"}`, nil, defaultConfig())

	if wf.APIVersion != argoWorkflowAPIVersion {
		t.Errorf("APIVersion = %q; want %q", wf.APIVersion, argoWorkflowAPIVersion)
	}
	if wf.Kind != argoWorkflowKind {
		t.Errorf("Kind = %q; want %q", wf.Kind, argoWorkflowKind)
	}
}

func TestBuildArgoWorkflow_NoTemplateRef(t *testing.T) {
	cfg := ArgoConfig{Namespace: "ns", WorkflowTemplateRef: ""}
	wf := buildArgoWorkflow("wf", "{}", nil, cfg)

	if wf.Spec.WorkflowTemplateRef != nil {
		t.Errorf("expected nil WorkflowTemplateRef when config is empty, got: %v", wf.Spec.WorkflowTemplateRef)
	}
}

func TestBuildArgoWorkflow_NoServiceAccount(t *testing.T) {
	cfg := ArgoConfig{Namespace: "ns", ServiceAccountName: ""}
	wf := buildArgoWorkflow("wf", "{}", nil, cfg)

	if wf.Spec.ServiceAccountName != "" {
		t.Errorf("expected empty ServiceAccountName, got: %q", wf.Spec.ServiceAccountName)
	}
}

// ─── workflowIRToJSON helper tests ───────────────────────────────────────────

func TestWorkflowIRToJSON_ContainsWorkflowID(t *testing.T) {
	ir := &zynaxv1.WorkflowIR{WorkflowId: "my-workflow", Name: "test"}
	out, err := workflowIRToJSON(ir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !containsStr(out, "my-workflow") {
		t.Errorf("JSON output does not contain workflow_id: %s", out)
	}
}
