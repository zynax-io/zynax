package domain

import (
	"testing"
)

// buildMinimalManifest returns a valid two-state manifest for graph tests.
func buildMinimalManifest() *Manifest {
	return &Manifest{
		Name:         "test-wf",
		Namespace:    "default",
		InitialState: "start",
		States: map[string]*State{
			"start": {
				ID:   "start",
				Type: StateTypeNormal,
				Transitions: []Transition{
					{EventType: "work.done", TargetState: "done"},
					{EventType: "work.done", TargetState: "done", Guard: "{{ .ctx.retry }}"},
				},
			},
			"done": {
				ID:   "done",
				Type: StateTypeTerminal,
			},
		},
	}
}

func TestBuild_Valid(t *testing.T) {
	g, errs := Build(buildMinimalManifest())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if g.InitialState != "start" {
		t.Errorf("InitialState = %q, want start", g.InitialState)
	}
	if len(g.States) != 2 {
		t.Errorf("len(States) = %d, want 2", len(g.States))
	}
}

func TestBuild_UnknownInitialState(t *testing.T) {
	m := buildMinimalManifest()
	m.InitialState = "nonexistent"
	_, errs := Build(m)
	if len(errs) == 0 {
		t.Fatal("expected error for unknown initial_state")
	}
	hasCode := false
	for _, e := range errs {
		if e.Code == ErrorCodeUnknownStateReference {
			hasCode = true
		}
	}
	if !hasCode {
		t.Errorf("expected ErrorCodeUnknownStateReference, got %v", errs)
	}
}

func TestBuild_TransitionToUnknownState(t *testing.T) {
	m := buildMinimalManifest()
	m.States["start"].Transitions = append(m.States["start"].Transitions, Transition{
		EventType:   "oops",
		TargetState: "ghost",
	})
	_, errs := Build(m)
	if len(errs) == 0 {
		t.Fatal("expected error for unknown transition target")
	}
	hasCode := false
	for _, e := range errs {
		if e.Code == ErrorCodeUnknownStateReference && e.StateName == "ghost" {
			hasCode = true
		}
	}
	if !hasCode {
		t.Errorf("expected ErrorCodeUnknownStateReference for 'ghost', got %v", errs)
	}
}

func TestBuild_NoTerminalState(t *testing.T) {
	m := &Manifest{
		Name:         "wf",
		Namespace:    "default",
		InitialState: "a",
		States: map[string]*State{
			"a": {
				ID:   "a",
				Type: StateTypeNormal,
				Transitions: []Transition{
					{EventType: "go", TargetState: "b"},
				},
			},
			"b": {
				ID:   "b",
				Type: StateTypeNormal,
				Transitions: []Transition{
					{EventType: "back", TargetState: "a"},
				},
			},
		},
	}
	_, errs := Build(m)
	if len(errs) == 0 {
		t.Fatal("expected error for no terminal state")
	}
	hasCode := false
	for _, e := range errs {
		if e.Code == ErrorCodeNoTerminalState {
			hasCode = true
		}
	}
	if !hasCode {
		t.Errorf("expected ErrorCodeNoTerminalState, got %v", errs)
	}
}

func TestBuild_OrphanState(t *testing.T) {
	m := &Manifest{
		Name:         "wf",
		Namespace:    "default",
		InitialState: "start",
		States: map[string]*State{
			"start": {
				ID:   "start",
				Type: StateTypeNormal,
				Transitions: []Transition{
					{EventType: "done", TargetState: "end"},
				},
			},
			"end": {
				ID:   "end",
				Type: StateTypeTerminal,
			},
			"orphan": {
				ID:   "orphan",
				Type: StateTypeNormal,
			},
		},
	}
	_, errs := Build(m)
	if len(errs) == 0 {
		t.Fatal("expected error for orphan state")
	}
	hasCode := false
	for _, e := range errs {
		if e.Code == ErrorCodeOrphanState && e.StateName == "orphan" {
			hasCode = true
		}
	}
	if !hasCode {
		t.Errorf("expected ErrorCodeOrphanState for 'orphan', got %v", errs)
	}
}

func TestBuild_MultipleErrors(t *testing.T) {
	m := &Manifest{
		Name:         "wf",
		Namespace:    "default",
		InitialState: "missing",
		States: map[string]*State{
			"start": {
				ID:   "start",
				Type: StateTypeNormal,
				Transitions: []Transition{
					{EventType: "done", TargetState: "also-missing"},
				},
			},
		},
	}
	_, errs := Build(m)
	if len(errs) < 2 {
		t.Errorf("expected multiple errors, got %d: %v", len(errs), errs)
	}
}

func TestTransitionsFor_Match(t *testing.T) {
	g, errs := Build(buildMinimalManifest())
	if len(errs) != 0 {
		t.Fatalf("build failed: %v", errs)
	}
	ts := g.TransitionsFor("start", "work.done")
	if len(ts) != 2 {
		t.Errorf("TransitionsFor returned %d transitions, want 2", len(ts))
	}
}

func TestTransitionsFor_NoMatch(t *testing.T) {
	g, _ := Build(buildMinimalManifest())
	ts := g.TransitionsFor("start", "unknown.event")
	if len(ts) != 0 {
		t.Errorf("expected 0 transitions, got %d", len(ts))
	}
}

func TestTransitionsFor_UnknownState(t *testing.T) {
	g, _ := Build(buildMinimalManifest())
	ts := g.TransitionsFor("nonexistent", "any.event")
	if ts != nil {
		t.Errorf("expected nil for unknown state, got %v", ts)
	}
}

func TestTerminalStates(t *testing.T) {
	g, _ := Build(buildMinimalManifest())
	ts := g.TerminalStates()
	if len(ts) != 1 || ts[0] != "done" {
		t.Errorf("TerminalStates() = %v, want [done]", ts)
	}
}

func TestTerminalStates_Multiple(t *testing.T) {
	m := &Manifest{
		Name:         "wf",
		Namespace:    "default",
		InitialState: "start",
		States: map[string]*State{
			"start": {
				ID:   "start",
				Type: StateTypeNormal,
				Transitions: []Transition{
					{EventType: "ok", TargetState: "success"},
					{EventType: "fail", TargetState: "failure"},
				},
			},
			"success": {ID: "success", Type: StateTypeTerminal},
			"failure": {ID: "failure", Type: StateTypeTerminal},
		},
	}
	g, errs := Build(m)
	if len(errs) != 0 {
		t.Fatalf("build failed: %v", errs)
	}
	ts := g.TerminalStates()
	if len(ts) != 2 {
		t.Errorf("TerminalStates() returned %d states, want 2", len(ts))
	}
}

func TestBuild_ParseAndBuild_Integration(t *testing.T) {
	data := []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: integration-wf
spec:
  initial_state: process
  states:
    process:
      actions:
        - capability: do_work
      on:
        - event: work.done
          goto: finish
    finish:
      type: terminal
`)
	m, parseErrs := ParseManifest(data)
	if len(parseErrs) != 0 {
		t.Fatalf("parse failed: %v", parseErrs)
	}
	g, buildErrs := Build(m)
	if len(buildErrs) != 0 {
		t.Fatalf("build failed: %v", buildErrs)
	}
	if g.InitialState != "process" {
		t.Errorf("InitialState = %q", g.InitialState)
	}
	ts := g.TransitionsFor("process", "work.done")
	if len(ts) != 1 || ts[0].TargetState != "finish" {
		t.Errorf("unexpected transitions: %v", ts)
	}
}
