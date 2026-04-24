package validators

import (
	"fmt"
	"strings"

	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
)

// TerminalStateValidator ensures at least one terminal state exists.
// This mirrors the check in Build() but is provided as a standalone Validator
// so callers that construct graphs by other means can still apply it.
type TerminalStateValidator struct{}

// Validate implements Validator.
func (TerminalStateValidator) Validate(g *domain.WorkflowGraph) []domain.ParseError {
	for _, s := range g.States {
		if s.Type == domain.StateTypeTerminal {
			return nil
		}
	}
	return []domain.ParseError{{
		Code:    domain.ErrorCodeNoTerminalState,
		Message: "workflow must have at least one terminal state",
	}}
}

// OrphanStateValidator ensures every state is reachable from the initial state.
// It performs a BFS from InitialState following all outbound transitions.
type OrphanStateValidator struct{}

// Validate implements Validator.
func (OrphanStateValidator) Validate(g *domain.WorkflowGraph) []domain.ParseError {
	reachable := reachableFrom(g, g.InitialState)
	var errs []domain.ParseError
	for id, state := range g.States {
		if !reachable[id] {
			errs = append(errs, domain.ParseError{
				Code:      domain.ErrorCodeOrphanState,
				Message:   fmt.Sprintf("state %q is unreachable from initial state %q", id, g.InitialState),
				Line:      state.Line,
				StateName: id,
			})
		}
	}
	return errs
}

// CircularTransitionDetector finds cycles that have no escape path to a
// terminal state. A cycle is only an error when every state in the cycle
// is unable to reach any terminal state (a pure infinite loop). Cycles that
// include a human_in_the_loop state or that have an outbound edge to a state
// which can reach terminal are accepted.
type CircularTransitionDetector struct{}

// Validate implements Validator.
func (CircularTransitionDetector) Validate(g *domain.WorkflowGraph) []domain.ParseError { //nolint:funlen // DFS with cycle reporting requires tracking stack and visited sets in one pass
	productive := productiveStates(g)

	// DFS with three-colour marking:
	//   0 = white (unvisited)
	//   1 = gray  (on the current DFS stack)
	//   2 = black (fully processed)
	color := make(map[string]int, len(g.States))
	stack := make([]string, 0, len(g.States))
	reported := make(map[string]bool)
	var errs []domain.ParseError

	var dfs func(id string)
	dfs = func(id string) {
		color[id] = 1
		stack = append(stack, id)

		state, ok := g.States[id]
		if !ok {
			stack = stack[:len(stack)-1]
			color[id] = 2
			return
		}

		for _, t := range state.Transitions {
			next := t.TargetState
			switch color[next] {
			case 0:
				dfs(next)
			case 1:
				// Back-edge: found a cycle. Locate the cycle start in the stack.
				start := -1
				for i, s := range stack {
					if s == next {
						start = i
						break
					}
				}
				if start < 0 || reported[next] {
					continue
				}
				cycle := stack[start:]
				trapped := true
				for _, s := range cycle {
					if productive[s] {
						trapped = false
						break
					}
				}
				if trapped {
					reported[next] = true
					path := strings.Join(append(cycle, next), " → ")
					errs = append(errs, domain.ParseError{
						Code:      domain.ErrorCodeCircularTransition,
						Message:   fmt.Sprintf("trapped cycle with no terminal escape: %s", path),
						StateName: next,
					})
				}
			}
		}

		stack = stack[:len(stack)-1]
		color[id] = 2
	}

	// Start from initial state, then mop up any isolated components.
	dfs(g.InitialState)
	for id := range g.States {
		if color[id] == 0 {
			dfs(id)
		}
	}

	return errs
}

// productiveStates returns the set of state IDs from which at least one
// terminal state is reachable. Uses a reverse-BFS from all terminal states.
func productiveStates(g *domain.WorkflowGraph) map[string]bool {
	// Build reverse adjacency list.
	reverse := make(map[string][]string, len(g.States))
	for id := range g.States {
		reverse[id] = nil
	}
	for id, state := range g.States {
		for _, t := range state.Transitions {
			reverse[t.TargetState] = append(reverse[t.TargetState], id)
		}
	}

	productive := make(map[string]bool, len(g.States))
	queue := make([]string, 0, len(g.States))
	for id, state := range g.States {
		if state.Type == domain.StateTypeTerminal {
			productive[id] = true
			queue = append(queue, id)
		}
	}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, pred := range reverse[curr] {
			if !productive[pred] {
				productive[pred] = true
				queue = append(queue, pred)
			}
		}
	}
	return productive
}

// reachableFrom returns the set of state IDs reachable from start via BFS.
func reachableFrom(g *domain.WorkflowGraph, start string) map[string]bool {
	visited := make(map[string]bool, len(g.States))
	queue := []string{start}
	visited[start] = true
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		state, ok := g.States[curr]
		if !ok {
			continue
		}
		for _, t := range state.Transitions {
			if !visited[t.TargetState] {
				visited[t.TargetState] = true
				queue = append(queue, t.TargetState)
			}
		}
	}
	return visited
}
