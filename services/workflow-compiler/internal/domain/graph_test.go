package domain

import (
	"context"
	"testing"
)

// stateDone is the terminal-state id reused across the terminal-output tests.
const stateDone = "done"

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
	g, errs := Build(context.Background(), buildMinimalManifest())
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
	_, errs := Build(context.Background(), m)
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
	_, errs := Build(context.Background(), m)
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
	_, errs := Build(context.Background(), m)
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
	_, errs := Build(context.Background(), m)
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
	_, errs := Build(context.Background(), m)
	if len(errs) < 2 {
		t.Errorf("expected multiple errors, got %d: %v", len(errs), errs)
	}
}

func TestTransitionsFor_Match(t *testing.T) {
	g, errs := Build(context.Background(), buildMinimalManifest())
	if len(errs) != 0 {
		t.Fatalf("build failed: %v", errs)
	}
	ts := g.TransitionsFor("start", "work.done")
	if len(ts) != 2 {
		t.Errorf("TransitionsFor returned %d transitions, want 2", len(ts))
	}
}

func TestTransitionsFor_NoMatch(t *testing.T) {
	g, _ := Build(context.Background(), buildMinimalManifest())
	ts := g.TransitionsFor("start", "unknown.event")
	if len(ts) != 0 {
		t.Errorf("expected 0 transitions, got %d", len(ts))
	}
}

func TestTransitionsFor_UnknownState(t *testing.T) {
	g, _ := Build(context.Background(), buildMinimalManifest())
	ts := g.TransitionsFor("nonexistent", "any.event")
	if ts != nil {
		t.Errorf("expected nil for unknown state, got %v", ts)
	}
}

func TestTerminalStates(t *testing.T) {
	g, _ := Build(context.Background(), buildMinimalManifest())
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
	g, errs := Build(context.Background(), m)
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
	m, parseErrs := ParseManifest(context.Background(), data)
	if len(parseErrs) != 0 {
		t.Fatalf("parse failed: %v", parseErrs)
	}
	g, buildErrs := Build(context.Background(), m)
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

// dataFlowManifest returns a two-state manifest where "summarize" consumes the
// declared "results" output of "search". The badRef override, when non-empty,
// replaces the consuming input reference to exercise validation failures.
func dataFlowManifest(badRef string) []byte {
	ref := "$.states.search.output.results"
	if badRef != "" {
		ref = badRef
	}
	return []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: data-flow-wf
spec:
  initial_state: search
  states:
    search:
      actions:
        - capability: web_search
          output:
            results: results
      on:
        - event: search.done
          goto: summarize
    summarize:
      type: terminal
      actions:
        - capability: summarize
          input:
            doc: "` + ref + `"
`)
}

func TestBuild_InputBindingResolves(t *testing.T) {
	m, parseErrs := ParseManifest(context.Background(), dataFlowManifest(""))
	if len(parseErrs) != 0 {
		t.Fatalf("parse failed: %v", parseErrs)
	}
	if _, errs := Build(context.Background(), m); len(errs) != 0 {
		t.Fatalf("expected resolvable binding to compile, got: %v", errs)
	}
}

func TestBuild_InputBindingValidationFailures(t *testing.T) {
	tests := []struct {
		name string
		ref  string
	}{
		{"undeclared output key", "$.states.search.output.missing"},
		{"unknown source state", "$.states.nope.output.results"},
		{"malformed reference", "$.states.search.results"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, parseErrs := ParseManifest(context.Background(), dataFlowManifest(tt.ref))
			if len(parseErrs) != 0 {
				t.Fatalf("parse failed: %v", parseErrs)
			}
			_, errs := Build(context.Background(), m)
			if len(errs) == 0 {
				t.Fatalf("expected a COMPILATION_ERROR for %q", tt.ref)
			}
			found := false
			for _, e := range errs {
				if e.Code == ErrorCodeInvalidFieldValue && e.StateName == "summarize" {
					found = true
					if e.Line <= 0 {
						t.Errorf("expected a line number on the error, got %d", e.Line)
					}
				}
			}
			if !found {
				t.Errorf("expected ErrorCodeInvalidFieldValue on 'summarize', got: %v", errs)
			}
		})
	}
}

// terminalOutputsManifest builds a two-state workflow whose initial "greet"
// state produces output "message" and whose terminal "done" state declares the
// given outputs YAML block (indented under `outputs:`). When onGreet is true the
// outputs block is placed on the non-terminal "greet" state instead.
func terminalOutputsManifest(outputsBlock string, onGreet bool) []byte {
	greetOutputs, doneOutputs := "", outputsBlock
	if onGreet {
		greetOutputs, doneOutputs = outputsBlock, ""
	}
	return []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: outputs-wf
spec:
  initial_state: greet
  states:
    greet:
      actions:
        - capability: echo
          output:
            message: message
      on:
        - event: greet.done
          goto: done
` + greetOutputs + `    done:
      type: terminal
` + doneOutputs)
}

func TestBuild_TerminalOutputs_LiteralAndValidRef(t *testing.T) {
	block := "      outputs:\n        review: \"$.states.greet.output.message\"\n        note: \"shipped\"\n"
	m, parseErrs := ParseManifest(context.Background(), terminalOutputsManifest(block, false))
	if len(parseErrs) != 0 {
		t.Fatalf("parse failed: %v", parseErrs)
	}
	// The terminal state carries both outputs after parsing.
	if got := m.States[stateDone].Outputs; len(got) != 2 || got["review"] != "$.states.greet.output.message" || got["note"] != "shipped" {
		t.Fatalf("parsed terminal outputs = %v, want review+note", got)
	}
	if _, errs := Build(context.Background(), m); len(errs) != 0 {
		t.Fatalf("expected literal + resolvable ref to compile, got: %v", errs)
	}
}

func TestBuild_TerminalOutputs_ValidationFailures(t *testing.T) {
	tests := []struct {
		name string
		ref  string
	}{
		{"undeclared output key", "$.states.greet.output.missing"},
		{"unknown source state", "$.states.nope.output.message"},
		{"malformed reference", "$.states.greet.message"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block := "      outputs:\n        review: \"" + tt.ref + "\"\n"
			m, parseErrs := ParseManifest(context.Background(), terminalOutputsManifest(block, false))
			if len(parseErrs) != 0 {
				t.Fatalf("parse failed: %v", parseErrs)
			}
			_, errs := Build(context.Background(), m)
			found := false
			for _, e := range errs {
				if e.Code == ErrorCodeInvalidFieldValue && e.StateName == stateDone {
					found = true
					if e.Line <= 0 {
						t.Errorf("expected a line number on the error, got %d", e.Line)
					}
				}
			}
			if !found {
				t.Errorf("expected ErrorCodeInvalidFieldValue on 'done' for %q, got: %v", tt.ref, errs)
			}
		})
	}
}

func TestBuild_OutputsOnNonTerminalState(t *testing.T) {
	block := "      outputs:\n        review: \"done\"\n"
	m, parseErrs := ParseManifest(context.Background(), terminalOutputsManifest(block, true))
	if len(parseErrs) != 0 {
		t.Fatalf("parse failed: %v", parseErrs)
	}
	_, errs := Build(context.Background(), m)
	found := false
	for _, e := range errs {
		if e.Code == ErrorCodeInvalidFieldValue && e.StateName == "greet" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a COMPILATION_ERROR for outputs on a non-terminal state, got: %v", errs)
	}
}

func TestConvertWorkflowOutputs_NonStringValue(t *testing.T) {
	block := "      outputs:\n        review:\n          nested: true\n"
	_, parseErrs := ParseManifest(context.Background(), terminalOutputsManifest(block, false))
	found := false
	for _, e := range parseErrs {
		if e.Code == ErrorCodeInvalidFieldValue && e.StateName == stateDone {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a parse error for a non-string output value, got: %v", parseErrs)
	}
}
