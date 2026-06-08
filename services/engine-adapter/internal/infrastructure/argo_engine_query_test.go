// SPDX-License-Identifier: Apache-2.0

// Package infrastructure provides concrete implementations of domain ports backed by
// external services and SDKs. Only this package may import engine-specific dependencies
// such as the Temporal Go SDK (ADR-015).
package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
)

// ─── mapArgoPhase tests ──────────────────────────────────────────────────────

func TestMapArgoPhase(t *testing.T) {
	cases := []struct {
		phase string
		want  domain.WorkflowStatus
	}{
		{"", domain.WorkflowStatusPending},
		{ArgoPhasePending, domain.WorkflowStatusPending},
		{ArgoPhaseRunning, domain.WorkflowStatusRunning},
		{ArgoPhaseSucceeded, domain.WorkflowStatusCompleted},
		{ArgoPhaseFailed, domain.WorkflowStatusFailed},
		{ArgoPhaseError, domain.WorkflowStatusFailed},
		{ArgoPhaseSkipped, domain.WorkflowStatusCancelled},
		{"unknown-future-phase", domain.WorkflowStatusPending},
	}

	for _, tc := range cases {
		got := mapArgoPhase(tc.phase)
		if got != tc.want {
			t.Errorf("mapArgoPhase(%q) = %v; want %v", tc.phase, got, tc.want)
		}
	}
}

// ─── GetStatus tests ─────────────────────────────────────────────────────────

func TestArgoEngine_GetStatus_Success(t *testing.T) {
	stub := &stubArgoClient{
		getWFResult: &ArgoWorkflow{
			Metadata: ArgoObjectMeta{
				Name:      "wf-001",
				Namespace: testArgoNamespace,
				Labels:    map[string]string{"env": "prod"},
			},
			Status: ArgoWorkflowStatus{
				Phase:      ArgoPhaseRunning,
				StartedAt:  "2026-06-09T10:00:00Z",
				FinishedAt: "",
			},
		},
	}
	engine := newTestArgoEngine(stub)

	run, err := engine.GetStatus(context.Background(), "wf-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run == nil {
		t.Fatal("expected non-nil WorkflowRun")
	}
	if run.RunID != "wf-001" {
		t.Errorf("RunID = %q; want %q", run.RunID, "wf-001")
	}
	if run.Status != domain.WorkflowStatusRunning {
		t.Errorf("Status = %v; want Running", run.Status)
	}
	if run.Engine != argoEngineName {
		t.Errorf("Engine = %q; want %q", run.Engine, argoEngineName)
	}
	if run.StartedAt.IsZero() {
		t.Error("StartedAt must be parsed from status")
	}
	if run.Labels["env"] != "prod" {
		t.Errorf("Labels not forwarded: %v", run.Labels)
	}
	if stub.lastGetNamespace != testArgoNamespace {
		t.Errorf("GetWorkflow namespace = %q; want %q", stub.lastGetNamespace, testArgoNamespace)
	}
	if stub.lastGetName != "wf-001" {
		t.Errorf("GetWorkflow name = %q; want %q", stub.lastGetName, "wf-001")
	}
}

func TestArgoEngine_GetStatus_Succeeded(t *testing.T) {
	stub := &stubArgoClient{
		getWFResult: &ArgoWorkflow{
			Metadata: ArgoObjectMeta{Name: "wf-done", Namespace: testArgoNamespace},
			Status: ArgoWorkflowStatus{
				Phase:      ArgoPhaseSucceeded,
				StartedAt:  "2026-06-09T10:00:00Z",
				FinishedAt: "2026-06-09T10:05:00Z",
			},
		},
	}
	engine := newTestArgoEngine(stub)

	run, err := engine.GetStatus(context.Background(), "wf-done")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.Status != domain.WorkflowStatusCompleted {
		t.Errorf("Status = %v; want Completed", run.Status)
	}
	if run.FinishedAt.IsZero() {
		t.Error("FinishedAt must be parsed from status")
	}
	if !run.Status.IsTerminal() {
		t.Error("Completed must be terminal")
	}
}

func TestArgoEngine_GetStatus_NotFound(t *testing.T) {
	stub := &stubArgoClient{
		getWFErr: fmt.Errorf("argo_client: get workflow %q: %w", "missing", errArgoNotFound),
	}
	engine := newTestArgoEngine(stub)

	_, err := engine.GetStatus(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing workflow")
	}
	if !errors.Is(err, domain.ErrExecutionNotFound) {
		t.Errorf("expected ErrExecutionNotFound, got: %v", err)
	}
}

func TestArgoEngine_GetStatus_EmptyRunID(t *testing.T) {
	stub := &stubArgoClient{}
	engine := newTestArgoEngine(stub)

	_, err := engine.GetStatus(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty runID")
	}
}

func TestArgoEngine_GetStatus_ClientError(t *testing.T) {
	stub := &stubArgoClient{getWFErr: errors.New("argo server unreachable")}
	engine := newTestArgoEngine(stub)

	_, err := engine.GetStatus(context.Background(), "wf-err")
	if err == nil {
		t.Fatal("expected error from client")
	}
	if !containsStr(err.Error(), "argo server unreachable") {
		t.Errorf("expected wrapped client error, got: %v", err)
	}
}

func TestArgoEngine_GetStatus_InvalidTimestamp(t *testing.T) {
	// Invalid timestamps must not cause a panic; StartedAt/FinishedAt stay zero.
	stub := &stubArgoClient{
		getWFResult: &ArgoWorkflow{
			Metadata: ArgoObjectMeta{Name: "wf-ts", Namespace: testArgoNamespace},
			Status: ArgoWorkflowStatus{
				Phase:      ArgoPhaseRunning,
				StartedAt:  "not-a-timestamp",
				FinishedAt: "also-bad",
			},
		},
	}
	engine := newTestArgoEngine(stub)

	run, err := engine.GetStatus(context.Background(), "wf-ts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !run.StartedAt.IsZero() {
		t.Error("invalid StartedAt should remain zero")
	}
	if !run.FinishedAt.IsZero() {
		t.Error("invalid FinishedAt should remain zero")
	}
}

// ─── Cancel tests ─────────────────────────────────────────────────────────────

func TestArgoEngine_Cancel_Success(t *testing.T) {
	stub := &stubArgoClient{
		getWFResult: &ArgoWorkflow{
			Metadata: ArgoObjectMeta{Name: "wf-cancel", Namespace: testArgoNamespace},
			Status:   ArgoWorkflowStatus{Phase: ArgoPhaseRunning},
		},
	}
	engine := newTestArgoEngine(stub)

	err := engine.Cancel(context.Background(), "wf-cancel", "user requested")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastDelName != "wf-cancel" {
		t.Errorf("DeleteWorkflow name = %q; want %q", stub.lastDelName, "wf-cancel")
	}
	if stub.lastDelNamespace != testArgoNamespace {
		t.Errorf("DeleteWorkflow namespace = %q; want %q", stub.lastDelNamespace, testArgoNamespace)
	}
}

func TestArgoEngine_Cancel_NotFound(t *testing.T) {
	stub := &stubArgoClient{
		getWFErr: fmt.Errorf("argo_client: get workflow %q: %w", "missing", errArgoNotFound),
	}
	engine := newTestArgoEngine(stub)

	err := engine.Cancel(context.Background(), "missing", "")
	if err == nil {
		t.Fatal("expected error for missing workflow")
	}
	if !errors.Is(err, domain.ErrExecutionNotFound) {
		t.Errorf("expected ErrExecutionNotFound, got: %v", err)
	}
}

func TestArgoEngine_Cancel_TerminalState(t *testing.T) {
	for _, phase := range []string{ArgoPhaseSucceeded, ArgoPhaseFailed, ArgoPhaseError, ArgoPhaseSkipped} {
		phase := phase
		t.Run(phase, func(t *testing.T) {
			stub := &stubArgoClient{
				getWFResult: &ArgoWorkflow{
					Metadata: ArgoObjectMeta{Name: "wf-done", Namespace: testArgoNamespace},
					Status:   ArgoWorkflowStatus{Phase: phase},
				},
			}
			engine := newTestArgoEngine(stub)

			err := engine.Cancel(context.Background(), "wf-done", "")
			if err == nil {
				t.Fatalf("expected ErrTerminalState for phase %q", phase)
			}
			if !errors.Is(err, domain.ErrTerminalState) {
				t.Errorf("expected ErrTerminalState, got: %v", err)
			}
		})
	}
}

func TestArgoEngine_Cancel_EmptyRunID(t *testing.T) {
	stub := &stubArgoClient{}
	engine := newTestArgoEngine(stub)

	err := engine.Cancel(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected error for empty runID")
	}
}

func TestArgoEngine_Cancel_DeleteClientError(t *testing.T) {
	stub := &stubArgoClient{
		getWFResult: &ArgoWorkflow{
			Metadata: ArgoObjectMeta{Name: "wf-del-err", Namespace: testArgoNamespace},
			Status:   ArgoWorkflowStatus{Phase: ArgoPhaseRunning},
		},
		delWFErr: errors.New("k8s unavailable"),
	}
	engine := newTestArgoEngine(stub)

	err := engine.Cancel(context.Background(), "wf-del-err", "")
	if err == nil {
		t.Fatal("expected error from delete client")
	}
	if !containsStr(err.Error(), "k8s unavailable") {
		t.Errorf("expected wrapped delete error, got: %v", err)
	}
}

func TestArgoEngine_Cancel_RaceDeleteNotFound(t *testing.T) {
	// Simulate race: GetWorkflow returns running but DeleteWorkflow returns 404.
	stub := &stubArgoClient{
		getWFResult: &ArgoWorkflow{
			Metadata: ArgoObjectMeta{Name: "wf-race", Namespace: testArgoNamespace},
			Status:   ArgoWorkflowStatus{Phase: ArgoPhaseRunning},
		},
		delWFErr: fmt.Errorf("argo_client: delete workflow %q: %w", "wf-race", errArgoNotFound),
	}
	engine := newTestArgoEngine(stub)

	err := engine.Cancel(context.Background(), "wf-race", "")
	if err == nil {
		t.Fatal("expected error for race delete not found")
	}
	if !errors.Is(err, domain.ErrExecutionNotFound) {
		t.Errorf("expected ErrExecutionNotFound for race, got: %v", err)
	}
}

// ─── Watch tests ──────────────────────────────────────────────────────────────

// sequentialStub is a specialised stub that returns a different ArgoWorkflow on
// each successive call to GetWorkflow, cycling through the provided states.
// This lets Watch tests exercise multi-poll transitions.
type sequentialStub struct {
	states  []ArgoWorkflowStatus
	call    int
	delErr  error
	sendErr error
}

func (s *sequentialStub) GetWorkflow(_ context.Context, _, _ string) (*ArgoWorkflow, error) {
	if s.call >= len(s.states) {
		// Return last state when exhausted.
		last := s.states[len(s.states)-1]
		return &ArgoWorkflow{
			Metadata: ArgoObjectMeta{Name: "wf-watch", Namespace: testArgoNamespace},
			Status:   last,
		}, nil
	}
	st := s.states[s.call]
	s.call++
	return &ArgoWorkflow{
		Metadata: ArgoObjectMeta{Name: "wf-watch", Namespace: testArgoNamespace},
		Status:   st,
	}, nil
}

func (s *sequentialStub) SubmitWorkflow(_ context.Context, _ string, _ *ArgoWorkflow) error {
	return nil
}
func (s *sequentialStub) SendEvent(_ context.Context, _, _ string, _ []byte) error {
	return s.sendErr
}
func (s *sequentialStub) DeleteWorkflow(_ context.Context, _, _ string) error {
	return s.delErr
}

func TestArgoEngine_Watch_TerminalImmediate(t *testing.T) {
	// Workflow is already Succeeded on first poll — send must be called once.
	stub := &stubArgoClient{
		getWFResult: &ArgoWorkflow{
			Metadata: ArgoObjectMeta{Name: "wf-done", Namespace: testArgoNamespace},
			Status:   ArgoWorkflowStatus{Phase: ArgoPhaseSucceeded},
		},
	}
	engine := newTestArgoEngine(stub)

	var events []*domain.WorkflowEvent
	err := engine.Watch(context.Background(), "wf-done", func(e *domain.WorkflowEvent) error {
		events = append(events, e)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("Watch must call send at least once")
	}
	last := events[len(events)-1]
	if !last.Status.IsTerminal() {
		t.Errorf("last event must be terminal; got status=%v", last.Status)
	}
}

func TestArgoEngine_Watch_Transitions(t *testing.T) {
	// Workflow transitions: Pending → Running → Succeeded.
	cfg := defaultConfig()
	seq := &sequentialStub{
		states: []ArgoWorkflowStatus{
			{Phase: ArgoPhasePending},
			{Phase: ArgoPhaseRunning},
			{Phase: ArgoPhaseSucceeded},
		},
	}
	engine := NewArgoEngine(seq, cfg)

	var events []*domain.WorkflowEvent
	err := engine.Watch(context.Background(), "wf-watch", func(e *domain.WorkflowEvent) error {
		events = append(events, e)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expect 3 events: Pending, Running, Succeeded.
	if len(events) != 3 {
		t.Fatalf("expected 3 transition events, got %d", len(events))
	}
	if events[0].ToState != ArgoPhasePending {
		t.Errorf("event[0].ToState = %q; want %q", events[0].ToState, ArgoPhasePending)
	}
	if events[1].ToState != ArgoPhaseRunning {
		t.Errorf("event[1].ToState = %q; want %q", events[1].ToState, ArgoPhaseRunning)
	}
	if events[2].ToState != ArgoPhaseSucceeded {
		t.Errorf("event[2].ToState = %q; want %q", events[2].ToState, ArgoPhaseSucceeded)
	}
	if !events[2].Status.IsTerminal() {
		t.Error("last event must be terminal")
	}
}

func TestArgoEngine_Watch_NotFound(t *testing.T) {
	stub := &stubArgoClient{
		getWFErr: fmt.Errorf("argo_client: get workflow %q: %w", "missing", errArgoNotFound),
	}
	engine := newTestArgoEngine(stub)

	err := engine.Watch(context.Background(), "missing", func(_ *domain.WorkflowEvent) error { return nil })
	if err == nil {
		t.Fatal("expected error for missing workflow")
	}
	if !errors.Is(err, domain.ErrExecutionNotFound) {
		t.Errorf("expected ErrExecutionNotFound, got: %v", err)
	}
}

func TestArgoEngine_Watch_ContextCancelled(t *testing.T) {
	// Watch on a running workflow should return ctx.Err() when context is cancelled.
	// Use a sequentialStub that always returns Running so Watch loops forever until cancelled.
	seq := &sequentialStub{
		states: []ArgoWorkflowStatus{
			{Phase: ArgoPhaseRunning},
			{Phase: ArgoPhaseRunning},
			{Phase: ArgoPhaseRunning},
		},
	}
	engine := NewArgoEngine(seq, defaultConfig())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := engine.Watch(ctx, "wf-running", func(_ *domain.WorkflowEvent) error { return nil })
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	// The error wraps context.DeadlineExceeded or context.Canceled.
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected DeadlineExceeded or Canceled, got: %v", err)
	}
}

func TestArgoEngine_Watch_EmptyRunID(t *testing.T) {
	stub := &stubArgoClient{}
	engine := newTestArgoEngine(stub)

	err := engine.Watch(context.Background(), "", func(_ *domain.WorkflowEvent) error { return nil })
	if err == nil {
		t.Fatal("expected error for empty runID")
	}
}

func TestArgoEngine_Watch_SendError(t *testing.T) {
	stub := &stubArgoClient{
		getWFResult: &ArgoWorkflow{
			Metadata: ArgoObjectMeta{Name: "wf-send-err", Namespace: testArgoNamespace},
			Status:   ArgoWorkflowStatus{Phase: ArgoPhaseSucceeded},
		},
	}
	engine := newTestArgoEngine(stub)

	sendErr := errors.New("downstream closed")
	err := engine.Watch(context.Background(), "wf-send-err", func(_ *domain.WorkflowEvent) error {
		return sendErr
	})
	if err == nil {
		t.Fatal("expected error when send returns error")
	}
	if !containsStr(err.Error(), "downstream closed") {
		t.Errorf("expected wrapped send error, got: %v", err)
	}
}
