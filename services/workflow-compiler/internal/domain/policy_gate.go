// Package domain contains the pure business logic for the workflow compiler.
// It has zero imports from the api or infrastructure layers.
package domain

import (
	"fmt"
)

// AnnotationEngineHint is the manifest annotation key used to declare a
// preferred workflow engine (e.g. "temporal", "argo"). The routing policy
// gate rejects the compilation if the named engine is not in the namespace's
// allowed-engine list.
const AnnotationEngineHint = "zynax.io/engine-hint"

// PolicyGateError is a domain-level policy violation that the API layer maps
// to a gRPC error status. The Kind field identifies which kind of violation
// occurred so the API layer can choose the correct gRPC code.
type PolicyGateError struct {
	Kind    PolicyViolationKind
	Message string
}

func (e *PolicyGateError) Error() string { return e.Message }

// PolicyViolationKind classifies the type of policy gate rejection.
type PolicyViolationKind int

const (
	// PolicyViolationRouting indicates the engine hint is not in the
	// namespace's allowed-engine list. The API layer returns PERMISSION_DENIED.
	PolicyViolationRouting PolicyViolationKind = iota + 1
)

// RoutingPolicyConfig describes which engines a namespace is allowed to use.
// An empty AllowedEngines list means "no restriction".
// This is the domain representation — the API layer maps the proto
// RoutingPolicy message to this struct.
type RoutingPolicyConfig struct {
	// Namespace this policy applies to. Empty disables enforcement.
	Namespace string
	// AllowedEngines is the engine allow-list. Empty means "any engine permitted".
	AllowedEngines []string
}

// PolicyGate enforces the namespace engine allow-list at compile time for the
// REST submission path. It is the interim dual-guard of ADR-045 §3: the
// Kubernetes CR path is guarded by a ValidatingAdmissionPolicy on the Workflow
// CR, while this gate covers `zynax apply` → gateway → compiler, which never
// touches the API server. The concurrent-invocation quota check that used to
// live here was removed by ADR-045 §2 — it was never enforced in production
// (the gate was always constructed with a nil counter) and quota remains
// unenforced until the engine-adapter QuotaChecker is wired live.
//
// The gate is stateless once constructed and safe for concurrent use.
// Use NewPolicyGate to create an instance.
type PolicyGate struct {
	routingPolicies map[string]RoutingPolicyConfig // keyed by namespace
}

// NewPolicyGate constructs a PolicyGate from the given per-namespace routing
// policies. Policies with an empty Namespace are silently ignored.
func NewPolicyGate(policies []RoutingPolicyConfig) *PolicyGate {
	pg := &PolicyGate{
		routingPolicies: make(map[string]RoutingPolicyConfig, len(policies)),
	}
	for _, p := range policies {
		if p.Namespace != "" {
			pg.routingPolicies[p.Namespace] = p
		}
	}
	return pg
}

// Check runs the policy gate against the given WorkflowGraph and returns a
// *PolicyGateError if the routing policy rejects the compilation, or nil if
// the graph is allowed through.
func (pg *PolicyGate) Check(g *WorkflowGraph, annotations map[string]string) *PolicyGateError {
	return pg.checkRoutingPolicy(g.Namespace, annotations)
}

// checkRoutingPolicy returns a PolicyGateError with Kind==PolicyViolationRouting
// if the engine hint in the manifest annotations is not in the namespace's
// allowed-engine list. Returns nil when:
//   - no routing policy is configured for the namespace
//   - the namespace's allowed-engine list is empty (no restriction)
//   - no engine hint annotation is present in the manifest
//   - the engine hint is in the allowed list
func (pg *PolicyGate) checkRoutingPolicy(namespace string, annotations map[string]string) *PolicyGateError {
	policy, ok := pg.routingPolicies[namespace]
	if !ok {
		return nil // no policy for this namespace
	}
	if len(policy.AllowedEngines) == 0 {
		return nil // empty allow-list → no restriction
	}

	hint, ok := annotations[AnnotationEngineHint]
	if !ok || hint == "" {
		return nil // no engine hint → no restriction to enforce
	}

	for _, allowed := range policy.AllowedEngines {
		if allowed == hint {
			return nil // engine is permitted
		}
	}

	return &PolicyGateError{
		Kind: PolicyViolationRouting,
		Message: fmt.Sprintf(
			"engine %q is not allowed in namespace %q; allowed engines: %v",
			hint, namespace, policy.AllowedEngines,
		),
	}
}
