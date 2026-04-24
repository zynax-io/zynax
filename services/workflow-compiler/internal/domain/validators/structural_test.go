package validators_test

import (
	"testing"

	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain/validators"
)

// helpers ─────────────────────────────────────────────────────────────────────

func terminalState() *domain.State {
	return &domain.State{ID: "done", Type: domain.StateTypeTerminal}
}

func normalState(id string, transitions ...domain.Transition) *domain.State {
	return &domain.State{ID: id, Type: domain.StateTypeNormal, Transitions: transitions}
}

func hitlState(id string, transitions ...domain.Transition) *domain.State {
	return &domain.State{ID: id, Type: domain.StateTypeHumanInTheLoop, Transitions: transitions}
}

func tr(event, target string) domain.Transition {
	return domain.Transition{EventType: event, TargetState: target}
}

func graphWith(initial string, states map[string]*domain.State) *domain.WorkflowGraph {
	return &domain.WorkflowGraph{
		Name:         "test-wf",
		Namespace:    "default",
		InitialState: initial,
		States:       states,
	}
}

func hasCode(errs []domain.ParseError, code domain.CompilationErrorCode) bool {
	for _, e := range errs {
		if e.Code == code {
			return true
		}
	}
	return false
}

// TerminalStateValidator ──────────────────────────────────────────────────────

func TestTerminalStateValidator(t *testing.T) {
	cases := []struct {
		name    string
		g       *domain.WorkflowGraph
		wantErr bool
	}{
		{
			name: "has terminal",
			g: graphWith("start", map[string]*domain.State{
				"start": normalState("start", tr("done", "end")),
				"end":   terminalState(),
			}),
		},
		{
			name: "no terminal",
			g: graphWith("start", map[string]*domain.State{
				"start": normalState("start", tr("loop", "start")),
			}),
			wantErr: true,
		},
		{
			name: "only terminal",
			g: graphWith("end", map[string]*domain.State{
				"end": terminalState(),
			}),
		},
	}
	v := validators.TerminalStateValidator{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := v.Validate(tc.g)
			if tc.wantErr && len(errs) == 0 {
				t.Error("expected error, got none")
			}
			if !tc.wantErr && len(errs) != 0 {
				t.Errorf("expected no error, got %v", errs)
			}
			if tc.wantErr && !hasCode(errs, domain.ErrorCodeNoTerminalState) {
				t.Errorf("expected ErrorCodeNoTerminalState, got %v", errs)
			}
		})
	}
}

// OrphanStateValidator ────────────────────────────────────────────────────────

func TestOrphanStateValidator(t *testing.T) {
	cases := []struct {
		name       string
		g          *domain.WorkflowGraph
		wantOrphan string // state name expected in error, empty = no error
	}{
		{
			name: "all reachable",
			g: graphWith("a", map[string]*domain.State{
				"a": normalState("a", tr("go", "b")),
				"b": terminalState(),
			}),
		},
		{
			name: "orphan state",
			g: graphWith("a", map[string]*domain.State{
				"a":      normalState("a", tr("go", "b")),
				"b":      terminalState(),
				"island": normalState("island"),
			}),
			wantOrphan: "island",
		},
		{
			name: "chain reachable",
			g: graphWith("a", map[string]*domain.State{
				"a": normalState("a", tr("1", "b")),
				"b": normalState("b", tr("2", "c")),
				"c": terminalState(),
			}),
		},
	}
	v := validators.OrphanStateValidator{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := v.Validate(tc.g)
			if tc.wantOrphan == "" && len(errs) != 0 {
				t.Errorf("expected no error, got %v", errs)
			}
			if tc.wantOrphan != "" {
				if !hasCode(errs, domain.ErrorCodeOrphanState) {
					t.Errorf("expected ErrorCodeOrphanState, got %v", errs)
				}
				found := false
				for _, e := range errs {
					if e.StateName == tc.wantOrphan {
						found = true
					}
				}
				if !found {
					t.Errorf("expected orphan %q in errors, got %v", tc.wantOrphan, errs)
				}
			}
		})
	}
}

// CircularTransitionDetector ──────────────────────────────────────────────────

func TestCircularTransitionDetector(t *testing.T) {
	cases := []struct {
		name    string
		g       *domain.WorkflowGraph
		wantErr bool
	}{
		{
			name: "no cycle",
			g: graphWith("a", map[string]*domain.State{
				"a": normalState("a", tr("go", "b")),
				"b": terminalState(),
			}),
		},
		{
			name: "cycle with terminal escape",
			g: graphWith("a", map[string]*domain.State{
				"a": normalState("a", tr("retry", "a"), tr("done", "b")),
				"b": terminalState(),
			}),
		},
		{
			name: "hitl in cycle — not trapped",
			g: graphWith("a", map[string]*domain.State{
				"a":      normalState("a", tr("submit", "review")),
				"review": hitlState("review", tr("approved", "done"), tr("rejected", "a")),
				"done":   terminalState(),
			}),
		},
		{
			name: "trapped two-state cycle",
			g: graphWith("a", map[string]*domain.State{
				"a": normalState("a", tr("ping", "b")),
				"b": normalState("b", tr("pong", "a")),
			}),
			wantErr: true,
		},
		{
			name: "trapped self-loop",
			g: graphWith("a", map[string]*domain.State{
				"a": normalState("a", tr("loop", "a")),
			}),
			wantErr: true,
		},
		{
			name: "cycle with escape via intermediate",
			g: graphWith("a", map[string]*domain.State{
				"a": normalState("a", tr("go", "b")),
				"b": normalState("b", tr("back", "a"), tr("exit", "c")),
				"c": terminalState(),
			}),
		},
	}
	v := validators.CircularTransitionDetector{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := v.Validate(tc.g)
			if tc.wantErr && len(errs) == 0 {
				t.Error("expected error, got none")
			}
			if !tc.wantErr && len(errs) != 0 {
				t.Errorf("expected no error, got %v", errs)
			}
			if tc.wantErr && !hasCode(errs, domain.ErrorCodeCircularTransition) {
				t.Errorf("expected ErrorCodeCircularTransition, got %v", errs)
			}
		})
	}
}

// Run and All ─────────────────────────────────────────────────────────────────

func TestRun_AccumulatesErrors(t *testing.T) {
	g := graphWith("a", map[string]*domain.State{
		"a": normalState("a", tr("loop", "a")),
		// no terminal + trapped cycle — two validators should each fire
	})
	errs := validators.Run(g, validators.All()...)
	if len(errs) == 0 {
		t.Error("expected errors from multiple validators, got none")
	}
}

func TestRun_NoErrors(t *testing.T) {
	g := graphWith("a", map[string]*domain.State{
		"a": normalState("a", tr("go", "b")),
		"b": terminalState(),
	})
	errs := validators.Run(g, validators.All()...)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}
