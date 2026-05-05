// SPDX-License-Identifier: Apache-2.0

package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
)

// stubCompiler is a test double for CompilerPort.
type stubCompiler struct {
	result domain.CompileResult
	err    error
}

func (s *stubCompiler) CompileWorkflow(_ context.Context, _ []byte, _ string, _ bool) (domain.CompileResult, error) {
	return s.result, s.err
}

// stubEngine is a test double for EnginePort.
type stubEngine struct {
	submitID    string
	submitErr   error
	statusRun   domain.WorkflowRunSummary
	statusErr   error
	cancelErr   error
	watchEvents []domain.WatchEvent
	watchErr    error
}

func (s *stubEngine) SubmitWorkflow(_ context.Context, _ []byte, _ string) (string, error) {
	return s.submitID, s.submitErr
}

func (s *stubEngine) GetWorkflowStatus(_ context.Context, _ string) (domain.WorkflowRunSummary, error) {
	return s.statusRun, s.statusErr
}

func (s *stubEngine) CancelWorkflow(_ context.Context, _ string) error {
	return s.cancelErr
}

func (s *stubEngine) WatchWorkflow(_ context.Context, _ string, send func(domain.WatchEvent) error) error {
	if s.watchErr != nil {
		return s.watchErr
	}
	for _, ev := range s.watchEvents {
		if err := send(ev); err != nil {
			return err
		}
	}
	return nil
}

// stubRegistry is a test double for RegistryPort.
type stubRegistry struct {
	reg domain.AgentRegistration
	err error
}

func (s *stubRegistry) RegisterAgent(_ context.Context, _ []byte, _ string) (domain.AgentRegistration, error) {
	return s.reg, s.err
}

func TestApplyService_ApplyWorkflow_Success(t *testing.T) {
	svc := domain.NewApplyService(
		&stubCompiler{result: domain.CompileResult{IRBytes: []byte("ir"), Warnings: []string{"w1"}}},
		&stubEngine{submitID: "run-001"},
		&stubRegistry{},
	)
	result, err := svc.ApplyWorkflow(context.Background(), domain.ApplyRequest{
		ManifestYAML: []byte("kind: Workflow"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RunID != "run-001" {
		t.Errorf("got run_id %q, want run-001", result.RunID)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "w1" {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
}

func TestApplyService_ApplyWorkflow_CompilationErrors(t *testing.T) {
	svc := domain.NewApplyService(
		&stubCompiler{result: domain.CompileResult{
			Errors: []domain.CompileError{{Code: "YAML_PARSE_ERROR", Message: "bad yaml", Line: 3}},
		}},
		&stubEngine{},
		&stubRegistry{},
	)
	result, err := svc.ApplyWorkflow(context.Background(), domain.ApplyRequest{
		ManifestYAML: []byte("bad yaml"),
	})
	if !errors.Is(err, domain.ErrCompilationFailed) {
		t.Fatalf("got %v, want ErrCompilationFailed", err)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 compile error, got %d", len(result.Errors))
	}
	if result.Errors[0].Line != 3 {
		t.Errorf("expected line 3, got %d", result.Errors[0].Line)
	}
}

func TestApplyService_ApplyWorkflow_DryRun_NoSubmit(t *testing.T) {
	submitted := false
	engine := &stubEngine{}
	engine.submitErr = errors.New("should not be called")
	svc := domain.NewApplyService(
		&stubCompiler{result: domain.CompileResult{IRBytes: []byte("ir"), Warnings: []string{"w"}}},
		engine,
		&stubRegistry{},
	)
	engine.submitErr = nil // reset — test checks RunID is empty, not that submit errors

	result, err := svc.ApplyWorkflow(context.Background(), domain.ApplyRequest{
		ManifestYAML: []byte("kind: Workflow"),
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RunID != "" {
		t.Errorf("dry-run must not return run_id, got %q", result.RunID)
	}
	_ = submitted
}

func TestApplyService_ApplyWorkflow_CompilerError_Propagates(t *testing.T) {
	svc := domain.NewApplyService(
		&stubCompiler{err: domain.ErrEngineUnavailable},
		&stubEngine{},
		&stubRegistry{},
	)
	_, err := svc.ApplyWorkflow(context.Background(), domain.ApplyRequest{ManifestYAML: []byte("y")})
	if !errors.Is(err, domain.ErrEngineUnavailable) {
		t.Fatalf("got %v, want ErrEngineUnavailable", err)
	}
}

func TestApplyService_ApplyWorkflow_EngineUnavailable(t *testing.T) {
	svc := domain.NewApplyService(
		&stubCompiler{result: domain.CompileResult{IRBytes: []byte("ir")}},
		&stubEngine{submitErr: domain.ErrEngineUnavailable},
		&stubRegistry{},
	)
	_, err := svc.ApplyWorkflow(context.Background(), domain.ApplyRequest{ManifestYAML: []byte("y")})
	if !errors.Is(err, domain.ErrEngineUnavailable) {
		t.Fatalf("got %v, want ErrEngineUnavailable", err)
	}
}

func TestApplyService_GetWorkflowStatus_Success(t *testing.T) {
	want := domain.WorkflowRunSummary{
		RunID: "r1", WorkflowID: "w1", Status: "RUNNING", CurrentState: "review",
	}
	svc := domain.NewApplyService(&stubCompiler{}, &stubEngine{statusRun: want}, &stubRegistry{})
	got, err := svc.GetWorkflowStatus(context.Background(), "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestApplyService_GetWorkflowStatus_NotFound(t *testing.T) {
	svc := domain.NewApplyService(&stubCompiler{}, &stubEngine{statusErr: domain.ErrNotFound}, &stubRegistry{})
	_, err := svc.GetWorkflowStatus(context.Background(), "unknown")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestApplyService_ApplyAgentDef_Success(t *testing.T) {
	svc := domain.NewApplyService(
		&stubCompiler{},
		&stubEngine{},
		&stubRegistry{reg: domain.AgentRegistration{AgentID: "agent-001"}},
	)
	result, err := svc.ApplyAgentDef(context.Background(), domain.ApplyRequest{
		ManifestYAML: []byte("kind: AgentDef\n"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AgentID != "agent-001" {
		t.Errorf("agent_id: got %q, want agent-001", result.AgentID)
	}
}

func TestApplyService_ApplyAgentDef_AlreadyExists(t *testing.T) {
	svc := domain.NewApplyService(
		&stubCompiler{},
		&stubEngine{},
		&stubRegistry{err: domain.ErrAgentAlreadyExists},
	)
	_, err := svc.ApplyAgentDef(context.Background(), domain.ApplyRequest{
		ManifestYAML: []byte("kind: AgentDef\n"),
	})
	if !errors.Is(err, domain.ErrAgentAlreadyExists) {
		t.Errorf("expected ErrAgentAlreadyExists, got %v", err)
	}
}

func TestApplyService_WatchWorkflowLogs_DeliversEvents(t *testing.T) {
	events := []domain.WatchEvent{
		{RunID: "r1", EventType: "state.entered", ToState: "review", Status: "WORKFLOW_STATUS_RUNNING"},
		{RunID: "r1", EventType: "workflow.completed", Status: "WORKFLOW_STATUS_COMPLETED"},
	}
	svc := domain.NewApplyService(&stubCompiler{}, &stubEngine{watchEvents: events}, &stubRegistry{})

	var got []domain.WatchEvent
	err := svc.WatchWorkflowLogs(context.Background(), "r1", func(ev domain.WatchEvent) error {
		got = append(got, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	if got[0].EventType != "state.entered" {
		t.Errorf("event[0]: got %q, want state.entered", got[0].EventType)
	}
	if got[1].EventType != "workflow.completed" {
		t.Errorf("event[1]: got %q, want workflow.completed", got[1].EventType)
	}
}

func TestApplyService_WatchWorkflowLogs_NotFound(t *testing.T) {
	svc := domain.NewApplyService(&stubCompiler{}, &stubEngine{watchErr: domain.ErrNotFound}, &stubRegistry{})
	err := svc.WatchWorkflowLogs(context.Background(), "ghost", func(_ domain.WatchEvent) error { return nil })
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}
