package validators

import (
	"fmt"
	"regexp"

	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
)

// capabilityNameRe matches snake_case capability names: lowercase letters and
// digits, words separated by single underscores, no leading/trailing underscore.
var capabilityNameRe = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

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
func (CapabilityRefValidator) Validate(g *domain.WorkflowGraph) []domain.ParseError {
	var errs []domain.ParseError
	for stateID, state := range g.States {
		for i, action := range state.Actions {
			capName := action.Capability
			if !capabilityNameRe.MatchString(capName) {
				errs = append(errs, domain.ParseError{
					Code:      domain.ErrorCodeInvalidFieldValue,
					Message:   fmt.Sprintf("state %q action[%d]: capability %q must be snake_case (e.g. summarize, send_email)", stateID, i, capName),
					Line:      state.Line,
					StateName: stateID,
				})
				continue
			}
			for _, prefix := range reservedPrefixes {
				if len(capName) >= len(prefix) && capName[:len(prefix)] == prefix {
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
func (EventNameValidator) Validate(g *domain.WorkflowGraph) []domain.ParseError {
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
func (NamespaceValidator) Validate(g *domain.WorkflowGraph) []domain.ParseError {
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

// DuplicateTransitionValidator checks that no two transitions from the same
// state share the same event_type. Duplicate event types create ambiguous
// routing when the engine receives an event (guards are evaluated at runtime,
// but the compiler must reject structurally ambiguous definitions).
type DuplicateTransitionValidator struct{}

// Validate implements Validator.
func (DuplicateTransitionValidator) Validate(g *domain.WorkflowGraph) []domain.ParseError {
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
