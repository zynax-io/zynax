package ir_test

import (
	"encoding/json"
	"testing"
	"time"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain/ir"
)

// helpers ──────────────────────────────────────────────────────────────────────

const (
	stateStart = "start"
	stateEnd   = "end"
)

func minimalGraph() *domain.WorkflowGraph {
	return &domain.WorkflowGraph{
		Name:         "my-wf",
		Namespace:    "staging",
		InitialState: stateStart,
		States: map[string]*domain.State{
			stateStart: {
				ID:   stateStart,
				Type: domain.StateTypeNormal,
				Transitions: []domain.Transition{
					{EventType: "done", TargetState: stateEnd},
				},
			},
			stateEnd: {ID: stateEnd, Type: domain.StateTypeTerminal},
		},
	}
}

func call(g *domain.WorkflowGraph) (*zynaxv1.WorkflowIR, error) {
	return ir.ToIR(g, "wf-001", "zynax.io/v1alpha1", time.Time{}) //nolint:wrapcheck
}

// ToIR basic shape ─────────────────────────────────────────────────────────────

func TestToIR_EnvelopeFields(t *testing.T) {
	wfIR, err := call(minimalGraph())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wfIR.WorkflowId != "wf-001" {
		t.Errorf("workflow_id: got %q, want %q", wfIR.WorkflowId, "wf-001")
	}
	if wfIR.Name != "my-wf" {
		t.Errorf("name: got %q, want %q", wfIR.Name, "my-wf")
	}
	if wfIR.Namespace != "staging" {
		t.Errorf("namespace: got %q, want %q", wfIR.Namespace, "staging")
	}
	if wfIR.ApiVersion != "zynax.io/v1alpha1" {
		t.Errorf("api_version: got %q, want %q", wfIR.ApiVersion, "zynax.io/v1alpha1")
	}
	if wfIR.IrVersion != "v1" {
		t.Errorf("ir_version: got %q, want %q", wfIR.IrVersion, "v1")
	}
	if wfIR.CompiledAt == nil {
		t.Error("compiled_at must not be nil")
	}
}

func TestToIR_InitialState(t *testing.T) {
	wfIR, err := call(minimalGraph())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wfIR.InitialState != stateStart {
		t.Errorf("initial_state: got %q, want %q", wfIR.InitialState, stateStart)
	}
}

func TestToIR_StateCount(t *testing.T) {
	wfIR, err := call(minimalGraph())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wfIR.States) != 2 {
		t.Errorf("states count: got %d, want 2", len(wfIR.States))
	}
}

// State ordering ───────────────────────────────────────────────────────────────

func TestToIR_StatesAreSortedByID(t *testing.T) {
	wfIR, err := call(minimalGraph())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// sorted: "end" < "start"
	if wfIR.States[0].Id != stateEnd || wfIR.States[1].Id != stateStart {
		t.Errorf("expected sorted order [end, start], got [%s, %s]", wfIR.States[0].Id, wfIR.States[1].Id)
	}
}

// State types ──────────────────────────────────────────────────────────────────

func TestToIR_StateTypes(t *testing.T) {
	cases := []struct {
		domainType domain.StateType
		protoType  zynaxv1.StateType
	}{
		{domain.StateTypeNormal, zynaxv1.StateType_STATE_TYPE_NORMAL},
		{domain.StateTypeTerminal, zynaxv1.StateType_STATE_TYPE_TERMINAL},
		{domain.StateTypeHumanInTheLoop, zynaxv1.StateType_STATE_TYPE_HUMAN_IN_THE_LOOP},
	}
	for _, tc := range cases {
		g := &domain.WorkflowGraph{
			Name:         "t",
			Namespace:    "default",
			InitialState: "s",
			States: map[string]*domain.State{
				"s": {ID: "s", Type: tc.domainType},
			},
		}
		wfIR, err := ir.ToIR(g, "id", "v1", time.Time{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wfIR.States[0].Type != tc.protoType {
			t.Errorf("type %v: got %v, want %v", tc.domainType, wfIR.States[0].Type, tc.protoType)
		}
	}
}

// Transitions ──────────────────────────────────────────────────────────────────

func TestToIR_TransitionFields(t *testing.T) {
	wfIR, err := call(minimalGraph())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var startState *zynaxv1.StateIR
	for _, s := range wfIR.States {
		if s.Id == stateStart {
			startState = s
		}
	}
	if startState == nil {
		t.Fatal("start state not found")
	}
	if len(startState.Transitions) != 1 {
		t.Fatalf("transitions count: got %d, want 1", len(startState.Transitions))
	}
	tr := startState.Transitions[0]
	if tr.EventType != "done" {
		t.Errorf("event_type: got %q, want %q", tr.EventType, "done")
	}
	if tr.TargetState != stateEnd {
		t.Errorf("target_state: got %q, want %q", tr.TargetState, stateEnd)
	}
}

func TestToIR_TransitionConditions(t *testing.T) {
	g := &domain.WorkflowGraph{
		Name:         "t",
		Namespace:    "default",
		InitialState: "a",
		States: map[string]*domain.State{
			"a": {
				ID:   "a",
				Type: domain.StateTypeNormal,
				Transitions: []domain.Transition{
					{
						EventType:   "reviewed",
						TargetState: "b",
						Conditions:  map[string]string{"approved": "event.score >= 90"},
					},
				},
			},
			"b": {ID: "b", Type: domain.StateTypeTerminal},
		},
	}
	wfIR, err := ir.ToIR(g, "id", "v1", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var aState *zynaxv1.StateIR
	for _, s := range wfIR.States {
		if s.Id == "a" {
			aState = s
		}
	}
	cond := aState.Transitions[0].Conditions
	if cond["approved"] != "event.score >= 90" {
		t.Errorf("condition: got %q, want %q", cond["approved"], "event.score >= 90")
	}
}

// Actions ──────────────────────────────────────────────────────────────────────

func TestToIR_ActionFields(t *testing.T) {
	g := &domain.WorkflowGraph{
		Name:         "t",
		Namespace:    "default",
		InitialState: "s",
		States: map[string]*domain.State{
			"s": {
				ID:   "s",
				Type: domain.StateTypeNormal,
				Actions: []domain.Action{
					{
						Capability: "summarize",
						Timeout:    30 * time.Second,
						Input:      map[string]interface{}{"prompt": "hello"},
						Async:      false,
					},
					{
						Capability: "notify_user",
						Async:      true,
					},
				},
			},
		},
	}
	wfIR, err := ir.ToIR(g, "id", "v1", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sState := wfIR.States[0]
	if len(sState.Actions) != 2 {
		t.Fatalf("actions count: got %d, want 2", len(sState.Actions))
	}

	a0 := sState.Actions[0]
	if a0.Capability != "summarize" {
		t.Errorf("capability: got %q, want %q", a0.Capability, "summarize")
	}
	if a0.Timeout == nil || a0.Timeout.AsDuration() != 30*time.Second {
		t.Errorf("timeout: got %v, want 30s", a0.Timeout)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(a0.InputTemplateJson), &parsed); err != nil {
		t.Fatalf("input_template_json not valid JSON: %v", err)
	}
	if parsed["prompt"] != "hello" {
		t.Errorf("input prompt: got %v, want %q", parsed["prompt"], "hello")
	}

	a1 := sState.Actions[1]
	if !a1.Async {
		t.Error("expected async=true for second action")
	}
	if a1.Timeout != nil {
		t.Errorf("zero timeout should be nil, got %v", a1.Timeout)
	}
	if a1.InputTemplateJson != "" {
		t.Errorf("empty input should produce empty JSON, got %q", a1.InputTemplateJson)
	}
}

// Terminal state has no transitions ────────────────────────────────────────────

func TestToIR_TerminalStateHasNoTransitions(t *testing.T) {
	wfIR, err := call(minimalGraph())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range wfIR.States {
		if s.Id == stateEnd && len(s.Transitions) != 0 {
			t.Errorf("terminal state should have 0 transitions, got %d", len(s.Transitions))
		}
	}
}

// compiledAt ───────────────────────────────────────────────────────────────────

func TestToIR_CompiledAtExplicit(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	wfIR, err := ir.ToIR(minimalGraph(), "id", "v1", ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !wfIR.CompiledAt.AsTime().Equal(ts) {
		t.Errorf("compiled_at: got %v, want %v", wfIR.CompiledAt.AsTime(), ts)
	}
}

// Validation errors ────────────────────────────────────────────────────────────

func TestToIR_NilGraphReturnsError(t *testing.T) {
	_, err := ir.ToIR(nil, "id", "v1", time.Time{})
	if err == nil {
		t.Error("expected error for nil graph, got nil")
	}
}

func TestToIR_EmptyWorkflowIDReturnsError(t *testing.T) {
	_, err := ir.ToIR(minimalGraph(), "", "v1", time.Time{})
	if err == nil {
		t.Error("expected error for empty workflowID, got nil")
	}
}

// HumanInTheLoop state ─────────────────────────────────────────────────────────

func TestToIR_HITLGraph(t *testing.T) {
	g := &domain.WorkflowGraph{
		Name:         "review-wf",
		Namespace:    "default",
		InitialState: "submit",
		States: map[string]*domain.State{
			"submit": {
				ID:   "submit",
				Type: domain.StateTypeNormal,
				Transitions: []domain.Transition{
					{EventType: "submitted", TargetState: "review"},
				},
			},
			"review": {
				ID:   "review",
				Type: domain.StateTypeHumanInTheLoop,
				Transitions: []domain.Transition{
					{EventType: "approved", TargetState: "done"},
					{EventType: "rejected", TargetState: "submit"},
				},
			},
			"done": {ID: "done", Type: domain.StateTypeTerminal},
		},
	}
	wfIR, err := ir.ToIR(g, "wf-002", "zynax.io/v1alpha1", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wfIR.States) != 3 {
		t.Fatalf("states count: got %d, want 3", len(wfIR.States))
	}
	var reviewState *zynaxv1.StateIR
	for _, s := range wfIR.States {
		if s.Id == "review" {
			reviewState = s
		}
	}
	if reviewState == nil {
		t.Fatal("review state not found")
	}
	if reviewState.Type != zynaxv1.StateType_STATE_TYPE_HUMAN_IN_THE_LOOP {
		t.Errorf("review type: got %v, want HUMAN_IN_THE_LOOP", reviewState.Type)
	}
	if len(reviewState.Transitions) != 2 {
		t.Errorf("review transitions: got %d, want 2", len(reviewState.Transitions))
	}
}
