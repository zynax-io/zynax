package domain

import "fmt"

// WorkflowGraph is the directed state machine built from a parsed Manifest.
// Nodes are States; edges are the Transitions embedded in each State,
// keyed by EventType.
type WorkflowGraph struct {
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
func Build(m *Manifest) (*WorkflowGraph, ParseErrors) {
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

	if len(errs) > 0 {
		return nil, errs
	}

	return &WorkflowGraph{
		InitialState: m.InitialState,
		States:       m.States,
	}, nil
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
