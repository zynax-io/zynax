package domain

import (
	"context"
	"fmt"
	"strings"
)

// WorkflowGraph is the directed state machine built from a parsed Manifest.
// Nodes are States; edges are the Transitions embedded in each State,
// keyed by EventType.
type WorkflowGraph struct {
	Name         string
	Namespace    string
	InitialState string
	States       map[string]*State
}

// Build constructs a WorkflowGraph from a Manifest, validating graph invariants:
//   - initial_state must reference a known state
//   - every transition target must reference a known state
//   - at least one terminal state must exist (no infinite graph)
//   - no orphan states (unreachable from initial_state via transitions)
//
// Returns all errors found — not just the first.
func Build(_ context.Context, m *Manifest) (*WorkflowGraph, ParseErrors) {
	var errs ParseErrors

	// initial_state must reference a known state.
	if _, ok := m.States[m.InitialState]; !ok {
		errs = append(errs, ParseError{
			Code:      ErrorCodeUnknownStateReference,
			Message:   fmt.Sprintf("initial_state %q is not defined in spec.states", m.InitialState),
			StateName: m.InitialState,
		})
	}

	// All transition targets must reference known states.
	for stateID, state := range m.States {
		for _, t := range state.Transitions {
			if _, ok := m.States[t.TargetState]; !ok {
				errs = append(errs, ParseError{
					Code:      ErrorCodeUnknownStateReference,
					Message:   fmt.Sprintf("state %q transitions to unknown state %q", stateID, t.TargetState),
					Line:      state.Line,
					StateName: t.TargetState,
				})
			}
		}
	}

	// At least one terminal state must exist.
	hasTerminal := false
	for _, state := range m.States {
		if state.Type == StateTypeTerminal {
			hasTerminal = true
			break
		}
	}
	if !hasTerminal {
		errs = append(errs, ParseError{
			Code:    ErrorCodeNoTerminalState,
			Message: "workflow must have at least one terminal state",
		})
	}

	// Detect orphan states: states that no transition points to (other than
	// the initial state, which is reachable by definition).
	reachable := make(map[string]bool, len(m.States))
	reachable[m.InitialState] = true
	for _, state := range m.States {
		for _, t := range state.Transitions {
			reachable[t.TargetState] = true
		}
	}
	for stateID, state := range m.States {
		if !reachable[stateID] {
			errs = append(errs, ParseError{
				Code:      ErrorCodeOrphanState,
				Message:   fmt.Sprintf("state %q is unreachable: no transition targets it", stateID),
				Line:      state.Line,
				StateName: stateID,
			})
		}
	}

	// Every input binding reference must resolve to a declared upstream output.
	errs = append(errs, validateInputBindings(m.States)...)

	// Terminal workflow outputs must be terminal-only and resolve to a declared
	// upstream output (ADR-042, M7.U).
	errs = append(errs, validateWorkflowOutputs(m.States)...)

	if len(errs) > 0 {
		return nil, errs
	}

	return &WorkflowGraph{
		Name:         m.Name,
		Namespace:    m.Namespace,
		InitialState: m.InitialState,
		States:       m.States,
	}, nil
}

// validateInputBindings checks that every action input binding reference of the
// form "$.states.<state>.output.<key>" resolves to a state that exists and that
// declares <key> in its output_bindings. Unresolved references yield a
// COMPILATION_ERROR (ErrorCodeInvalidFieldValue) carrying the manifest line of
// the consuming state. Returns all errors found — not just the first.
func validateInputBindings(states map[string]*State) ParseErrors {
	declared := indexDeclaredOutputs(states)

	var errs ParseErrors
	for id, st := range states {
		for ai, a := range st.Actions {
			for key, ref := range a.InputBindings {
				if perr := resolveBinding(id, ai, key, ref, st.Line, declared); perr != nil {
					errs = append(errs, *perr)
				}
			}
		}
	}
	return errs
}

// indexDeclaredOutputs builds stateID → set of output keys declared by that
// state's action output_bindings. This is the set of "$.states.<id>.output.<key>"
// references resolvable elsewhere in the manifest.
func indexDeclaredOutputs(states map[string]*State) map[string]map[string]struct{} {
	declared := make(map[string]map[string]struct{}, len(states))
	for id, st := range states {
		for _, a := range st.Actions {
			for key := range a.OutputBindings {
				if declared[id] == nil {
					declared[id] = make(map[string]struct{})
				}
				declared[id][key] = struct{}{}
			}
		}
	}
	return declared
}

// validateWorkflowOutputs enforces the terminal-output contract (ADR-042, M7.U):
//   - outputs: may be declared only on a TERMINAL state;
//   - each value is a literal, or a well-formed "$.states.<state>.output.<key>"
//     reference whose target state declares <key> in its output_bindings.
//
// Violations yield a COMPILATION_ERROR (ErrorCodeInvalidFieldValue) carrying the
// offending state's manifest line. Returns all errors found — not just the first.
func validateWorkflowOutputs(states map[string]*State) ParseErrors {
	declared := indexDeclaredOutputs(states)

	var errs ParseErrors
	for id, st := range states {
		if len(st.Outputs) == 0 {
			continue
		}
		if st.Type != StateTypeTerminal {
			errs = append(errs, ParseError{
				Code:      ErrorCodeInvalidFieldValue,
				Message:   fmt.Sprintf("state %q: outputs may be declared only on a terminal state", id),
				Line:      st.Line,
				StateName: id,
			})
			continue
		}
		for name, ref := range st.Outputs {
			// Literals (anything not rooted at the data-reference prefix) pass
			// through verbatim — there is no transform language in M7 (ADR-029).
			if !strings.HasPrefix(ref, inputBindingPrefix) {
				continue
			}
			if perr := resolveWorkflowOutput(id, name, ref, st.Line, declared); perr != nil {
				errs = append(errs, *perr)
			}
		}
	}
	return errs
}

// resolveWorkflowOutput parses a terminal output's "$.states.<state>.output.<key>"
// reference and verifies the target state and output key are declared upstream.
// Returns nil when the reference resolves, or a ParseError describing why not.
func resolveWorkflowOutput(
	state, outputName, ref string,
	line int,
	declared map[string]map[string]struct{},
) *ParseError {
	srcState, srcKey, ok := parseBindingRef(ref)
	if !ok {
		return &ParseError{
			Code:      ErrorCodeInvalidFieldValue,
			Message:   fmt.Sprintf("state %q: outputs[%q] reference %q is malformed; expected \"$.states.<state>.output.<key>\"", state, outputName, ref),
			Line:      line,
			StateName: state,
		}
	}
	outputs, stateOK := declared[srcState]
	if !stateOK {
		return &ParseError{
			Code:      ErrorCodeInvalidFieldValue,
			Message:   fmt.Sprintf("state %q: outputs[%q] references output of unknown or output-less state %q", state, outputName, srcState),
			Line:      line,
			StateName: state,
		}
	}
	if _, keyOK := outputs[srcKey]; !keyOK {
		return &ParseError{
			Code:      ErrorCodeInvalidFieldValue,
			Message:   fmt.Sprintf("state %q: outputs[%q] references undeclared output %q of state %q", state, outputName, srcKey, srcState),
			Line:      line,
			StateName: state,
		}
	}
	return nil
}

// resolveBinding parses a single "$.states.<state>.output.<key>" reference and
// verifies the target state and output key are declared. Returns nil when the
// reference resolves, or a ParseError describing why it does not.
func resolveBinding(
	consumer string,
	actionIdx int,
	inputKey, ref string,
	line int,
	declared map[string]map[string]struct{},
) *ParseError {
	srcState, srcKey, ok := parseBindingRef(ref)
	if !ok {
		return &ParseError{
			Code:      ErrorCodeInvalidFieldValue,
			Message:   fmt.Sprintf("state %q: actions[%d].input[%q] reference %q is malformed; expected \"$.states.<state>.output.<key>\"", consumer, actionIdx, inputKey, ref),
			Line:      line,
			StateName: consumer,
		}
	}
	outputs, stateOK := declared[srcState]
	if !stateOK {
		return &ParseError{
			Code:      ErrorCodeInvalidFieldValue,
			Message:   fmt.Sprintf("state %q: actions[%d].input[%q] references output of unknown or output-less state %q", consumer, actionIdx, inputKey, srcState),
			Line:      line,
			StateName: consumer,
		}
	}
	if _, keyOK := outputs[srcKey]; !keyOK {
		return &ParseError{
			Code:      ErrorCodeInvalidFieldValue,
			Message:   fmt.Sprintf("state %q: actions[%d].input[%q] references undeclared output %q of state %q", consumer, actionIdx, inputKey, srcKey, srcState),
			Line:      line,
			StateName: consumer,
		}
	}
	return nil
}

// parseBindingRef splits "$.states.<state>.output.<key>" into its state and
// output key. <key> may itself contain dots (a nested path). Returns ok=false
// for any other shape.
func parseBindingRef(ref string) (state, key string, ok bool) {
	const root = "$.states."
	if !strings.HasPrefix(ref, root) {
		return "", "", false
	}
	rest := ref[len(root):]
	const mid = ".output."
	idx := strings.Index(rest, mid)
	if idx <= 0 {
		return "", "", false
	}
	state = rest[:idx]
	key = rest[idx+len(mid):]
	if state == "" || key == "" {
		return "", "", false
	}
	return state, key, true
}

// TransitionsFor returns all transitions from a state that match the given
// event type. Multiple transitions may match the same event when guards
// disambiguate them. Guards are not evaluated here — the engine handles that.
func (g *WorkflowGraph) TransitionsFor(stateID, eventType string) []Transition {
	state, ok := g.States[stateID]
	if !ok {
		return nil
	}
	var out []Transition
	for _, t := range state.Transitions {
		if t.EventType == eventType {
			out = append(out, t)
		}
	}
	return out
}

// TerminalStates returns the IDs of all terminal states in the graph.
func (g *WorkflowGraph) TerminalStates() []string {
	var out []string
	for id, state := range g.States {
		if state.Type == StateTypeTerminal {
			out = append(out, id)
		}
	}
	return out
}
