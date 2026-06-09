// Package domain contains the pure business logic for the workflow compiler.
// It has zero imports from the api or infrastructure layers.
package domain

import (
	"context"
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
	// PolicyViolationQuota indicates the namespace has exceeded its
	// concurrent capability invocation quota. The API layer returns
	// RESOURCE_EXHAUSTED.
	PolicyViolationQuota
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

// CapabilityQuotaConfig describes the concurrent invocation limit for a
// namespace. MaxConcurrent == 0 means "no quota configured" (unbounded).
// This is the domain representation — the API layer maps the proto
// CapabilityQuota message to this struct.
type CapabilityQuotaConfig struct {
	// Namespace this quota applies to. Empty disables enforcement.
	Namespace string
	// MaxConcurrent is the hard ceiling for concurrent capability invocations.
	// Zero means "no quota configured".
	MaxConcurrent int32
}

// ActiveInvocationCounter is the port (interface) the policy gate uses to
// query the current number of active capability invocations for a namespace.
// The infrastructure layer provides the concrete implementation; unit tests
// supply a simple stub.
type ActiveInvocationCounter interface {
	// ActiveCount returns the number of in-flight capability invocations for
	// the given namespace.
	ActiveCount(ctx context.Context, namespace string) (int32, error)
}

// PolicyGate enforces routing policies and capability quotas at compile time.
// It is stateless except for the injected counter and the policy configs it
// holds — it is safe for concurrent use once constructed.
//
// Use NewPolicyGate to create an instance.
type PolicyGate struct {
	routingPolicies map[string]RoutingPolicyConfig   // keyed by namespace
	quotaConfigs    map[string]CapabilityQuotaConfig // keyed by namespace
	counter         ActiveInvocationCounter
}

// NewPolicyGate constructs a PolicyGate from the given per-namespace
// routing policies and capability quota configs. Policies or quotas with an
// empty Namespace are silently ignored. counter may be nil — when nil the gate
// skips quota checks (treats every namespace as unconstrained).
func NewPolicyGate(
	policies []RoutingPolicyConfig,
	quotas []CapabilityQuotaConfig,
	counter ActiveInvocationCounter,
) *PolicyGate {
	pg := &PolicyGate{
		routingPolicies: make(map[string]RoutingPolicyConfig, len(policies)),
		quotaConfigs:    make(map[string]CapabilityQuotaConfig, len(quotas)),
		counter:         counter,
	}
	for _, p := range policies {
		if p.Namespace != "" {
			pg.routingPolicies[p.Namespace] = p
		}
	}
	for _, q := range quotas {
		if q.Namespace != "" {
			pg.quotaConfigs[q.Namespace] = q
		}
	}
	return pg
}

// Check runs all policy gates against the given WorkflowGraph and returns a
// *PolicyGateError if any gate rejects the compilation, or nil if the graph
// is allowed through.
//
// Order of checks:
//  1. Routing policy — engine hint vs. namespace allow-list
//  2. Capability quota — active invocation count vs. namespace ceiling
//
// The first violation encountered is returned; subsequent checks are skipped.
func (pg *PolicyGate) Check(ctx context.Context, g *WorkflowGraph, annotations map[string]string) *PolicyGateError {
	if err := pg.checkRoutingPolicy(g.Namespace, annotations); err != nil {
		return err
	}
	if err := pg.checkCapabilityQuota(ctx, g.Namespace); err != nil {
		return err
	}
	return nil
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

// checkCapabilityQuota returns a PolicyGateError with Kind==PolicyViolationQuota
// if the namespace has reached its concurrent invocation ceiling. Returns nil
// when:
//   - no quota is configured for the namespace
//   - max_concurrent is zero (unbounded)
//   - the counter is nil (quota enforcement is disabled)
//   - the active count is below the ceiling
func (pg *PolicyGate) checkCapabilityQuota(ctx context.Context, namespace string) *PolicyGateError {
	if pg.counter == nil {
		return nil
	}
	quota, ok := pg.quotaConfigs[namespace]
	if !ok {
		return nil // no quota for this namespace
	}
	if quota.MaxConcurrent == 0 {
		return nil // unbounded
	}

	active, err := pg.counter.ActiveCount(ctx, namespace)
	if err != nil {
		// Counter errors are not policy violations; they are treated as
		// "quota check unavailable" and the gate passes to avoid blocking
		// compilations when the counter backend is temporarily unreachable.
		return nil
	}

	if active >= quota.MaxConcurrent {
		return &PolicyGateError{
			Kind: PolicyViolationQuota,
			Message: fmt.Sprintf(
				"capability quota exceeded for namespace %q: %d active invocations, limit is %d",
				namespace, active, quota.MaxConcurrent,
			),
		}
	}
	return nil
}
