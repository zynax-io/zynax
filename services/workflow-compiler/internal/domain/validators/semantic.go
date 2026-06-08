package validators

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
)

// capabilityNameRe matches snake_case capability names with an optional
// namespace qualifier prefix. Two forms are accepted:
//
//   - Unqualified: snake_case only (e.g. "summarize", "send_email")
//   - Qualified:   <namespace>/<capability> where the namespace is a DNS-label
//     (e.g. "team-a/send_email", "ns-b/summarize")
//
// The cross-namespace validator checks whether the namespace prefix, if present,
// matches the workflow's own namespace.
var capabilityNameRe = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?/)?[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// eventNameRe matches dot-separated event names, e.g. "review.approved", "push".
// Each segment is a lowercase identifier.
var eventNameRe = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*$`)

// namespaceRe matches DNS-label-safe namespace values: lowercase alphanumeric,
// hyphens allowed in the middle, ≤63 characters, no leading/trailing hyphen.
var namespaceRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// CapabilityRefValidator checks that every action capability name follows
// snake_case naming and carries no reserved prefixes.
type CapabilityRefValidator struct{}

var reservedPrefixes = []string{"zynax_", "system_", "internal_"}

// Validate implements Validator.
func (CapabilityRefValidator) Validate(_ context.Context, g *domain.WorkflowGraph) []domain.ParseError {
	var errs []domain.ParseError
	for stateID, state := range g.States {
		for i, action := range state.Actions {
			capName := action.Capability
			if !capabilityNameRe.MatchString(capName) {
				errs = append(errs, domain.ParseError{
					Code:      domain.ErrorCodeInvalidFieldValue,
					Message:   fmt.Sprintf("state %q action[%d]: capability %q must be snake_case or <namespace>/snake_case (e.g. summarize, send_email, team-a/send_email)", stateID, i, capName),
					Line:      state.Line,
					StateName: stateID,
				})
				continue
			}
			// Extract the bare capability name (strip optional namespace prefix).
			bareCap := capName
			if slashIdx := strings.IndexByte(capName, '/'); slashIdx >= 0 {
				bareCap = capName[slashIdx+1:]
			}
			for _, prefix := range reservedPrefixes {
				if len(bareCap) >= len(prefix) && bareCap[:len(prefix)] == prefix {
					errs = append(errs, domain.ParseError{
						Code:      domain.ErrorCodeInvalidFieldValue,
						Message:   fmt.Sprintf("state %q action[%d]: capability %q uses reserved prefix %q", stateID, i, capName, prefix),
						Line:      state.Line,
						StateName: stateID,
					})
					break
				}
			}
		}
	}
	return errs
}

// EventNameValidator checks that every transition event_type follows the
// dot-separated naming convention (e.g. "review.approved", "push").
type EventNameValidator struct{}

// Validate implements Validator.
func (EventNameValidator) Validate(_ context.Context, g *domain.WorkflowGraph) []domain.ParseError {
	var errs []domain.ParseError
	for stateID, state := range g.States {
		for _, t := range state.Transitions {
			if !eventNameRe.MatchString(t.EventType) {
				errs = append(errs, domain.ParseError{
					Code:      domain.ErrorCodeInvalidFieldValue,
					Message:   fmt.Sprintf("state %q transition: event_type %q must be dot-separated lowercase (e.g. review.approved, push)", stateID, t.EventType),
					Line:      state.Line,
					StateName: stateID,
				})
			}
		}
	}
	return errs
}

// NamespaceValidator checks that the workflow namespace is DNS-label safe:
// lowercase alphanumeric and hyphens, no leading/trailing hyphen, ≤63 chars.
type NamespaceValidator struct{}

// Validate implements Validator.
func (NamespaceValidator) Validate(_ context.Context, g *domain.WorkflowGraph) []domain.ParseError {
	ns := g.Namespace
	if ns == "" {
		return nil // empty namespace is caught by the manifest parser
	}
	if len(ns) > 63 {
		return []domain.ParseError{{
			Code:    domain.ErrorCodeInvalidFieldValue,
			Message: fmt.Sprintf("namespace %q exceeds 63 characters (%d)", ns, len(ns)),
		}}
	}
	if !namespaceRe.MatchString(ns) {
		return []domain.ParseError{{
			Code:    domain.ErrorCodeInvalidFieldValue,
			Message: fmt.Sprintf("namespace %q must be DNS-label safe: lowercase alphanumeric and hyphens, no leading/trailing hyphen", ns),
		}}
	}
	return nil
}

// TransitionSetValidator rejects transitions where a .set{} map contains a
// non-string value. The engine serialises Set entries into context variables at
// runtime; non-string values cannot be round-tripped through the proto wire
// format and would silently produce empty strings.
type TransitionSetValidator struct{}

// Validate implements Validator.
func (TransitionSetValidator) Validate(_ context.Context, g *domain.WorkflowGraph) []domain.ParseError {
	var errs []domain.ParseError
	for stateID, state := range g.States {
		for _, t := range state.Transitions {
			for k, v := range t.Set {
				if _, ok := v.(string); !ok {
					errs = append(errs, domain.ParseError{
						Code:      domain.ErrorCodeInvalidFieldValue,
						Message:   fmt.Sprintf("state %q transition %q: set key %q must be a string, got %T", stateID, t.EventType, k, v),
						Line:      state.Line,
						StateName: stateID,
					})
				}
			}
		}
	}
	return errs
}

// CrossNamespaceCapabilityValidator rejects capability references that explicitly
// target a namespace different from the workflow's own namespace.
//
// Capability names may optionally carry a namespace qualifier in the form
// "<namespace>/<capability_name>". When a qualifier is present, it must match
// WorkflowGraph.Namespace so that workflows cannot dispatch across namespace
// boundaries at compile time. Unqualified capability names (no "/") are always
// allowed — they resolve to agents in the workflow's own namespace at runtime.
//
// Example — rejected: workflow namespace "ns-a", capability "ns-b/send_email"
// Example — allowed:  workflow namespace "ns-a", capability "summarize"
// Example — allowed:  workflow namespace "ns-a", capability "ns-a/summarize"
type CrossNamespaceCapabilityValidator struct{}

// Validate implements Validator.
func (CrossNamespaceCapabilityValidator) Validate(_ context.Context, g *domain.WorkflowGraph) []domain.ParseError {
	var errs []domain.ParseError
	for stateID, state := range g.States {
		for i, action := range state.Actions {
			capRef := action.Capability
			slashIdx := strings.IndexByte(capRef, '/')
			if slashIdx < 0 {
				// Unqualified capability — resolves within the workflow's namespace.
				continue
			}
			capNS := capRef[:slashIdx]
			if capNS != g.Namespace {
				errs = append(errs, domain.ParseError{
					Code: domain.ErrorCodeInvalidFieldValue,
					Message: fmt.Sprintf(
						"state %q action[%d]: capability %q targets namespace %q but workflow namespace is %q; cross-namespace capability dispatch is not allowed",
						stateID, i, capRef, capNS, g.Namespace,
					),
					Line:      state.Line,
					StateName: stateID,
				})
			}
		}
	}
	return errs
}

// DuplicateTransitionValidator checks that no two transitions from the same
// state share the same event_type. Duplicate event types create ambiguous
// routing when the engine receives an event (guards are evaluated at runtime,
// but the compiler must reject structurally ambiguous definitions).
type DuplicateTransitionValidator struct{}

// Validate implements Validator.
func (DuplicateTransitionValidator) Validate(_ context.Context, g *domain.WorkflowGraph) []domain.ParseError {
	var errs []domain.ParseError
	for stateID, state := range g.States {
		seen := make(map[string]struct{}, len(state.Transitions))
		for _, t := range state.Transitions {
			if t.Guard != "" {
				// Guarded transitions may share an event_type; the guard
				// disambiguates them at runtime.
				continue
			}
			if _, ok := seen[t.EventType]; ok {
				errs = append(errs, domain.ParseError{
					Code:      domain.ErrorCodeInvalidFieldValue,
					Message:   fmt.Sprintf("state %q has duplicate unguarded transition for event_type %q", stateID, t.EventType),
					Line:      state.Line,
					StateName: stateID,
				})
			} else {
				seen[t.EventType] = struct{}{}
			}
		}
	}
	return errs
}
