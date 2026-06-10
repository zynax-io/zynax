// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CapabilityQuotaConfig describes the concurrent invocation limit for a
// namespace. MaxConcurrent == 0 means "no quota configured" (unbounded).
// This is the engine-adapter representation of the proto CapabilityQuota
// message; the wire-up layer maps the proto into this struct.
type CapabilityQuotaConfig struct {
	// Namespace this quota applies to. Empty disables enforcement.
	Namespace string
	// MaxConcurrent is the hard ceiling for concurrent capability invocations.
	// Zero means "no quota configured" (unbounded).
	MaxConcurrent int32
}

// ActiveInvocationCounter is the port the quota checker uses to query the
// current number of in-flight capability invocations for a namespace. The
// wire-up layer provides the concrete implementation (e.g. backed by the
// task-broker); unit tests supply a simple stub.
type ActiveInvocationCounter interface {
	// ActiveCount returns the number of in-flight capability invocations for
	// the given namespace.
	ActiveCount(ctx context.Context, namespace string) (int32, error)
}

// QuotaChecker is the engine-adapter's execution-time enforcement gate. It is
// the second quota gate in the policy chain — the workflow-compiler enforces
// the same quota at compile time (#803), and this checker re-validates at
// dispatch time so a workflow that was admitted before its namespace filled up
// is still rejected before DispatchCapabilityActivity submits a task.
//
// It is safe for concurrent use once constructed. Use NewQuotaChecker.
type QuotaChecker struct {
	quotaConfigs map[string]CapabilityQuotaConfig // keyed by namespace
	counter      ActiveInvocationCounter
}

// NewQuotaChecker constructs a QuotaChecker from the given per-namespace quota
// configs. Quotas with an empty Namespace are silently ignored. counter may be
// nil — when nil the checker skips all quota checks (treats every namespace as
// unconstrained), so the gate is fail-open by configuration.
func NewQuotaChecker(quotas []CapabilityQuotaConfig, counter ActiveInvocationCounter) *QuotaChecker {
	qc := &QuotaChecker{
		quotaConfigs: make(map[string]CapabilityQuotaConfig, len(quotas)),
		counter:      counter,
	}
	for _, q := range quotas {
		if q.Namespace != "" {
			qc.quotaConfigs[q.Namespace] = q
		}
	}
	return qc
}

// Check is the pre-dispatch quota gate. Callers MUST invoke it before
// DispatchCapabilityActivity submits a task to the broker. It returns a
// codes.ResourceExhausted gRPC status error if the namespace has reached its
// concurrent-invocation ceiling, and nil when the dispatch is allowed.
//
// It returns nil (allows the dispatch) when:
//   - the counter is nil (quota enforcement is disabled)
//   - no quota is configured for the namespace
//   - the namespace's MaxConcurrent is zero (unbounded)
//   - the active count is below the ceiling
//   - the counter backend returns an error (fail-open: a transient counter
//     outage must not block all dispatches)
func (qc *QuotaChecker) Check(ctx context.Context, namespace string) error {
	if qc.counter == nil {
		return nil
	}
	quota, ok := qc.quotaConfigs[namespace]
	if !ok {
		return nil // no quota for this namespace
	}
	if quota.MaxConcurrent == 0 {
		return nil // unbounded
	}

	active, err := qc.counter.ActiveCount(ctx, namespace)
	if err != nil {
		// Counter errors are not quota violations; treat as "check
		// unavailable" and allow the dispatch so a transient counter outage
		// never blocks execution.
		return nil
	}

	if active >= quota.MaxConcurrent {
		return status.Errorf(
			codes.ResourceExhausted,
			"capability quota exceeded for namespace %q: %d active invocations, limit is %d",
			namespace, active, quota.MaxConcurrent,
		)
	}
	return nil
}
