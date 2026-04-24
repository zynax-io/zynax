// Package validators provides the Validator interface and built-in structural
// and semantic validators for WorkflowGraph.
//
// Usage:
//
//	errs := validators.Run(graph, validators.All()...)
package validators

import "github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"

// Validator checks a WorkflowGraph for a single category of errors.
// Each implementation is stateless and safe for concurrent use.
type Validator interface {
	Validate(g *domain.WorkflowGraph) []domain.ParseError
}

// Run applies each validator to g and accumulates all errors.
// Validators are applied in order; all run even when earlier ones find errors.
func Run(g *domain.WorkflowGraph, vs ...Validator) []domain.ParseError {
	var errs []domain.ParseError
	for _, v := range vs {
		errs = append(errs, v.Validate(g)...)
	}
	return errs
}

// All returns the full set of built-in validators in the recommended order:
// structural checks first, then semantic checks.
func All() []Validator {
	return []Validator{
		TerminalStateValidator{},
		OrphanStateValidator{},
		CircularTransitionDetector{},
		DuplicateTransitionValidator{},
		CapabilityRefValidator{},
		EventNameValidator{},
		NamespaceValidator{},
	}
}
