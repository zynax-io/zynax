package api_test

import (
	"context"
	"testing"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// helpers ──────────────────────────────────────────────────────────────────────

func newServer() *api.Server {
	return api.New()
}

var validYAML = []byte(`apiVersion: zynax.io/v1alpha1
kind: Workflow
metadata:
  name: review-flow
  namespace: team-a
spec:
  initial_state: start
  states:
    start:
      on:
        - event: submitted
          goto: review
    review:
      type: human_in_the_loop
      on:
        - event: approved
          goto: done
    done:
      type: terminal
`)

func compile(t *testing.T, s *api.Server, yaml []byte) *zynaxv1.CompileWorkflowResponse {
	t.Helper()
	resp, err := s.CompileWorkflow(context.Background(), &zynaxv1.CompileWorkflowRequest{
		ManifestYaml: yaml,
	})
	if err != nil {
		t.Fatalf("CompileWorkflow: %v", err)
	}
	return resp
}

func grpcCode(err error) codes.Code {
	if s, ok := status.FromError(err); ok {
		return s.Code()
	}
	return codes.Unknown
}

// CompileWorkflow ──────────────────────────────────────────────────────────────

func TestCompileWorkflow_ValidManifest(t *testing.T) {
	resp := compile(t, newServer(), validYAML)
	if resp.WorkflowIr == nil {
		t.Fatal("expected WorkflowIR, got nil")
	}
	if resp.WorkflowIr.WorkflowId == "" {
		t.Error("workflow_id must not be empty")
	}
	if resp.WorkflowIr.Name != "review-flow" {
		t.Errorf("name: got %q, want %q", resp.WorkflowIr.Name, "review-flow")
	}
	if resp.WorkflowIr.Namespace != "team-a" {
		t.Errorf("namespace: got %q, want %q", resp.WorkflowIr.Namespace, "team-a")
	}
	if resp.CompilationDurationMs <= 0 {
		t.Errorf("compilation_duration_ms: got %d, want > 0", resp.CompilationDurationMs)
	}
}

func TestCompileWorkflow_EmptyManifest(t *testing.T) {
	_, err := newServer().CompileWorkflow(context.Background(), &zynaxv1.CompileWorkflowRequest{})
	if grpcCode(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

func TestCompileWorkflow_InvalidYAML(t *testing.T) {
	resp, err := newServer().CompileWorkflow(context.Background(), &zynaxv1.CompileWorkflowRequest{
		ManifestYaml: []byte("not: yaml: {"),
	})
	if err != nil {
		t.Fatalf("expected OK gRPC status, got: %v", err)
	}
	if len(resp.Errors) == 0 {
		t.Error("expected at least one error in response.Errors")
	}
	if resp.WorkflowIr != nil {
		t.Error("expected nil WorkflowIR when errors are present")
	}
}

func TestCompileWorkflow_NoTerminalState(t *testing.T) {
	yaml := []byte(`apiVersion: zynax.io/v1alpha1
kind: Workflow
metadata:
  name: bad
  namespace: default
spec:
  initial_state: start
  states:
    start:
      on:
        - event: next
          goto: review
    review:
      on: []
`)
	resp, err := newServer().CompileWorkflow(context.Background(), &zynaxv1.CompileWorkflowRequest{
		ManifestYaml: yaml,
	})
	if err != nil {
		t.Fatalf("expected OK gRPC status, got: %v", err)
	}
	if len(resp.Errors) == 0 {
		t.Error("expected at least one error in response.Errors")
	}
	if resp.WorkflowIr != nil {
		t.Error("expected nil WorkflowIR when errors are present")
	}
}

func TestCompileWorkflow_AllErrorsReturned(t *testing.T) {
	// A manifest with multiple distinct validation errors must return all of them,
	// not just the first. Two unreachable states + no terminal state = ≥2 errors.
	yaml := []byte(`apiVersion: zynax.io/v1alpha1
kind: Workflow
metadata:
  name: multi-err
  namespace: default
spec:
  initial_state: start
  states:
    start:
      on:
        - event: go
          goto: step1
    step1:
      on: []
    orphan1:
      on: []
    orphan2:
      on: []
`)
	resp, err := newServer().CompileWorkflow(context.Background(), &zynaxv1.CompileWorkflowRequest{
		ManifestYaml: yaml,
	})
	if err != nil {
		t.Fatalf("expected OK gRPC status, got: %v", err)
	}
	if len(resp.Errors) < 2 {
		t.Errorf("expected ≥2 errors, got %d: %v", len(resp.Errors), resp.Errors)
	}
}

func TestCompileWorkflow_LineNumberBoundsCheck(t *testing.T) {
	// toProtoErrors must clamp line numbers that exceed int32 range.
	// We verify the happy path: a valid parse error has a sensible line number.
	resp, err := newServer().CompileWorkflow(context.Background(), &zynaxv1.CompileWorkflowRequest{
		ManifestYaml: []byte("not: yaml: {"),
	})
	if err != nil {
		t.Fatalf("unexpected gRPC error: %v", err)
	}
	for _, e := range resp.Errors {
		if e.LineNumber < 0 {
			t.Errorf("line_number must be non-negative, got %d", e.LineNumber)
		}
	}
}

func TestCompileWorkflow_DryRunReturnsIR(t *testing.T) {
	resp, err := newServer().CompileWorkflow(context.Background(), &zynaxv1.CompileWorkflowRequest{
		ManifestYaml: validYAML,
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("CompileWorkflow dry_run: %v", err)
	}
	if resp.WorkflowIr == nil {
		t.Fatal("expected WorkflowIR in response even on dry_run")
	}
}

func TestCompileWorkflow_IrVersion(t *testing.T) {
	resp := compile(t, newServer(), validYAML)
	if resp.WorkflowIr.IrVersion != "v1" {
		t.Errorf("ir_version: got %q, want %q", resp.WorkflowIr.IrVersion, "v1")
	}
}

// ValidateManifest ─────────────────────────────────────────────────────────────

func TestValidateManifest_ValidReturnsTrue(t *testing.T) {
	resp, err := newServer().ValidateManifest(context.Background(), &zynaxv1.ValidateManifestRequest{
		ManifestYaml: validYAML,
	})
	if err != nil {
		t.Fatalf("ValidateManifest: %v", err)
	}
	if !resp.Valid {
		t.Errorf("expected valid=true, got false; errors: %v", resp.Errors)
	}
	if len(resp.Errors) != 0 {
		t.Errorf("expected no errors, got %v", resp.Errors)
	}
}

func TestValidateManifest_InvalidReturnsFalse(t *testing.T) {
	yaml := []byte(`apiVersion: zynax.io/v1alpha1
kind: Workflow
metadata:
  name: bad
  namespace: default
spec:
  initial_state: start
  states:
    start:
      on: []
`)
	resp, err := newServer().ValidateManifest(context.Background(), &zynaxv1.ValidateManifestRequest{
		ManifestYaml: yaml,
	})
	if err != nil {
		t.Fatalf("ValidateManifest: %v", err)
	}
	if resp.Valid {
		t.Error("expected valid=false")
	}
	if len(resp.Errors) == 0 {
		t.Error("expected at least one error")
	}
}

func TestValidateManifest_EmptyManifest(t *testing.T) {
	_, err := newServer().ValidateManifest(context.Background(), &zynaxv1.ValidateManifestRequest{})
	if grpcCode(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

func TestValidateManifest_NoWorkflowIR(t *testing.T) {
	// ValidateManifest must never return a WorkflowIR — verified by contract.
	// The response type has no WorkflowIR field; this test documents the invariant.
	resp, err := newServer().ValidateManifest(context.Background(), &zynaxv1.ValidateManifestRequest{
		ManifestYaml: validYAML,
	})
	if err != nil {
		t.Fatalf("ValidateManifest: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	// ValidateManifestResponse has no WorkflowIr field by proto design
}

// GetCompiledWorkflow ──────────────────────────────────────────────────────────

func TestGetCompiledWorkflow_AlwaysNotFound(t *testing.T) {
	// The compiler is stateless: GetCompiledWorkflow always returns NOT_FOUND.
	// Callers must retain the ir_payload from CompileWorkflow response.
	s := newServer()
	_, err := s.GetCompiledWorkflow(context.Background(), &zynaxv1.GetCompiledWorkflowRequest{
		WorkflowId: "any-id",
	})
	if grpcCode(err) != codes.NotFound {
		t.Errorf("expected NotFound (stateless compiler), got %v", err)
	}
}

func TestGetCompiledWorkflow_NotFound(t *testing.T) {
	_, err := newServer().GetCompiledWorkflow(context.Background(), &zynaxv1.GetCompiledWorkflowRequest{
		WorkflowId: "nonexistent-wf",
	})
	if grpcCode(err) != codes.NotFound {
		t.Errorf("expected NotFound, got %v", err)
	}
	if s, ok := status.FromError(err); ok {
		if s.Message() == "" {
			t.Error("expected non-empty error message")
		}
	}
}

func TestGetCompiledWorkflow_EmptyID(t *testing.T) {
	_, err := newServer().GetCompiledWorkflow(context.Background(), &zynaxv1.GetCompiledWorkflowRequest{
		WorkflowId: "",
	})
	if grpcCode(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

func TestGetCompiledWorkflow_IDsAreUnique(t *testing.T) {
	s := newServer()
	r1 := compile(t, s, validYAML)
	r2 := compile(t, s, validYAML)
	if r1.WorkflowIr.WorkflowId == r2.WorkflowIr.WorkflowId {
		t.Error("successive compiles must generate unique workflow IDs")
	}
}

func BenchmarkCompileWorkflow(b *testing.B) {
	s := newServer()
	req := &zynaxv1.CompileWorkflowRequest{ManifestYaml: validYAML}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.CompileWorkflow(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}

// Context cancellation ─────────────────────────────────────────────────────────

func TestCompileWorkflow_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call
	_, err := newServer().CompileWorkflow(ctx, &zynaxv1.CompileWorkflowRequest{
		ManifestYaml: validYAML,
	})
	if grpcCode(err) != codes.Canceled {
		t.Errorf("expected Canceled, got %v", err)
	}
}

func TestValidateManifest_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := newServer().ValidateManifest(ctx, &zynaxv1.ValidateManifestRequest{
		ManifestYaml: validYAML,
	})
	if grpcCode(err) != codes.Canceled {
		t.Errorf("expected Canceled, got %v", err)
	}
}

func TestGetCompiledWorkflow_CancelledContext(t *testing.T) {
	s := newServer()
	compiled := compile(t, s, validYAML)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.GetCompiledWorkflow(ctx, &zynaxv1.GetCompiledWorkflowRequest{
		WorkflowId: compiled.WorkflowIr.WorkflowId,
	})
	if grpcCode(err) != codes.Canceled {
		t.Errorf("expected Canceled, got %v", err)
	}
}
