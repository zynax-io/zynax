package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
)

// stubCounter is a test double for ActiveInvocationCounter.
type stubCounter struct {
	count int32
	err   error
}

func (s *stubCounter) ActiveCount(_ context.Context, _ string) (int32, error) {
	return s.count, s.err
}

// minimalGraph returns a WorkflowGraph with a given namespace for gate tests.
func minimalGraph(ns string) *domain.WorkflowGraph {
	return &domain.WorkflowGraph{
		Name:         "test-workflow",
		Namespace:    ns,
		InitialState: "start",
		States: map[string]*domain.State{
			"start": {ID: "start", Type: domain.StateTypeTerminal},
		},
	}
}

// ─── NewPolicyGate ────────────────────────────────────────────────────────────

func TestNewPolicyGate_IgnoresEmptyNamespacePolicies(t *testing.T) {
	gate := domain.NewPolicyGate(
		[]domain.RoutingPolicyConfig{{Namespace: "", AllowedEngines: []string{"temporal"}}},
		[]domain.CapabilityQuotaConfig{{Namespace: "", MaxConcurrent: 1}},
		nil,
	)
	// Neither restriction should apply — the empty-namespace entries are ignored.
	err := gate.Check(context.Background(), minimalGraph("default"), map[string]string{
		domain.AnnotationEngineHint: "argo",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// ─── Routing policy gate ──────────────────────────────────────────────────────

func TestRoutingPolicy_AllowedEngine_Pass(t *testing.T) {
	gate := domain.NewPolicyGate(
		[]domain.RoutingPolicyConfig{{
			Namespace:      "prod",
			AllowedEngines: []string{"temporal", "argo"},
		}},
		nil, nil,
	)
	err := gate.Check(context.Background(), minimalGraph("prod"), map[string]string{
		domain.AnnotationEngineHint: "temporal",
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRoutingPolicy_DeniedEngine_ReturnsPermissionDenied(t *testing.T) {
	gate := domain.NewPolicyGate(
		[]domain.RoutingPolicyConfig{{
			Namespace:      "prod",
			AllowedEngines: []string{"temporal"},
		}},
		nil, nil,
	)
	err := gate.Check(context.Background(), minimalGraph("prod"), map[string]string{
		domain.AnnotationEngineHint: "argo",
	})
	if err == nil {
		t.Fatal("expected PolicyGateError, got nil")
	}
	if err.Kind != domain.PolicyViolationRouting {
		t.Errorf("expected PolicyViolationRouting, got %v", err.Kind)
	}
}

func TestRoutingPolicy_EmptyAllowList_Pass(t *testing.T) {
	// Empty allowed_engines → no restriction.
	gate := domain.NewPolicyGate(
		[]domain.RoutingPolicyConfig{{
			Namespace:      "prod",
			AllowedEngines: []string{},
		}},
		nil, nil,
	)
	err := gate.Check(context.Background(), minimalGraph("prod"), map[string]string{
		domain.AnnotationEngineHint: "any-engine",
	})
	if err != nil {
		t.Fatalf("expected nil for empty allow-list, got %v", err)
	}
}

func TestRoutingPolicy_NoAnnotation_Pass(t *testing.T) {
	// No engine hint in manifest → nothing to enforce.
	gate := domain.NewPolicyGate(
		[]domain.RoutingPolicyConfig{{
			Namespace:      "prod",
			AllowedEngines: []string{"temporal"},
		}},
		nil, nil,
	)
	err := gate.Check(context.Background(), minimalGraph("prod"), map[string]string{})
	if err != nil {
		t.Fatalf("expected nil when no engine hint, got %v", err)
	}
}

func TestRoutingPolicy_NoPolicyForNamespace_Pass(t *testing.T) {
	// Policy defined for "prod" but graph is in "staging".
	gate := domain.NewPolicyGate(
		[]domain.RoutingPolicyConfig{{
			Namespace:      "prod",
			AllowedEngines: []string{"temporal"},
		}},
		nil, nil,
	)
	err := gate.Check(context.Background(), minimalGraph("staging"), map[string]string{
		domain.AnnotationEngineHint: "argo",
	})
	if err != nil {
		t.Fatalf("expected nil for unconstrained namespace, got %v", err)
	}
}

func TestRoutingPolicy_ErrorMessageContainsEngine(t *testing.T) {
	gate := domain.NewPolicyGate(
		[]domain.RoutingPolicyConfig{{
			Namespace:      "prod",
			AllowedEngines: []string{"temporal"},
		}},
		nil, nil,
	)
	gateErr := gate.Check(context.Background(), minimalGraph("prod"), map[string]string{
		domain.AnnotationEngineHint: "argo",
	})
	if gateErr == nil {
		t.Fatal("expected error")
	}
	if gateErr.Error() == "" {
		t.Error("error message must not be empty")
	}
}

// ─── Capability quota gate ────────────────────────────────────────────────────

func TestCapabilityQuota_UnderLimit_Pass(t *testing.T) {
	gate := domain.NewPolicyGate(
		nil,
		[]domain.CapabilityQuotaConfig{{Namespace: "prod", MaxConcurrent: 5}},
		&stubCounter{count: 4},
	)
	err := gate.Check(context.Background(), minimalGraph("prod"), map[string]string{})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCapabilityQuota_AtLimit_ReturnsResourceExhausted(t *testing.T) {
	gate := domain.NewPolicyGate(
		nil,
		[]domain.CapabilityQuotaConfig{{Namespace: "prod", MaxConcurrent: 5}},
		&stubCounter{count: 5},
	)
	err := gate.Check(context.Background(), minimalGraph("prod"), nil)
	if err == nil {
		t.Fatal("expected PolicyGateError, got nil")
	}
	if err.Kind != domain.PolicyViolationQuota {
		t.Errorf("expected PolicyViolationQuota, got %v", err.Kind)
	}
}

func TestCapabilityQuota_OverLimit_ReturnsResourceExhausted(t *testing.T) {
	gate := domain.NewPolicyGate(
		nil,
		[]domain.CapabilityQuotaConfig{{Namespace: "prod", MaxConcurrent: 3}},
		&stubCounter{count: 10},
	)
	err := gate.Check(context.Background(), minimalGraph("prod"), nil)
	if err == nil {
		t.Fatal("expected PolicyGateError for over-limit")
	}
	if err.Kind != domain.PolicyViolationQuota {
		t.Errorf("expected PolicyViolationQuota, got %v", err.Kind)
	}
}

func TestCapabilityQuota_ZeroMaxConcurrent_Pass(t *testing.T) {
	// max_concurrent == 0 means unbounded.
	gate := domain.NewPolicyGate(
		nil,
		[]domain.CapabilityQuotaConfig{{Namespace: "prod", MaxConcurrent: 0}},
		&stubCounter{count: 9999},
	)
	err := gate.Check(context.Background(), minimalGraph("prod"), nil)
	if err != nil {
		t.Fatalf("expected nil for unbounded quota, got %v", err)
	}
}

func TestCapabilityQuota_NoQuotaForNamespace_Pass(t *testing.T) {
	// Quota defined for "prod" but graph is in "dev".
	gate := domain.NewPolicyGate(
		nil,
		[]domain.CapabilityQuotaConfig{{Namespace: "prod", MaxConcurrent: 1}},
		&stubCounter{count: 999},
	)
	err := gate.Check(context.Background(), minimalGraph("dev"), nil)
	if err != nil {
		t.Fatalf("expected nil for unconstrained namespace, got %v", err)
	}
}

func TestCapabilityQuota_NilCounter_Pass(t *testing.T) {
	// nil counter → quota enforcement disabled.
	gate := domain.NewPolicyGate(
		nil,
		[]domain.CapabilityQuotaConfig{{Namespace: "prod", MaxConcurrent: 1}},
		nil, // no counter
	)
	err := gate.Check(context.Background(), minimalGraph("prod"), nil)
	if err != nil {
		t.Fatalf("expected nil when counter is nil, got %v", err)
	}
}

func TestCapabilityQuota_CounterError_PassThrough(t *testing.T) {
	// Counter errors are treated as "quota unavailable" — gate passes.
	gate := domain.NewPolicyGate(
		nil,
		[]domain.CapabilityQuotaConfig{{Namespace: "prod", MaxConcurrent: 1}},
		&stubCounter{count: 0, err: errors.New("counter unavailable")},
	)
	err := gate.Check(context.Background(), minimalGraph("prod"), nil)
	if err != nil {
		t.Fatalf("expected nil when counter returns error, got %v", err)
	}
}

func TestCapabilityQuota_ErrorMessageContainsNamespace(t *testing.T) {
	gate := domain.NewPolicyGate(
		nil,
		[]domain.CapabilityQuotaConfig{{Namespace: "prod", MaxConcurrent: 2}},
		&stubCounter{count: 3},
	)
	gateErr := gate.Check(context.Background(), minimalGraph("prod"), nil)
	if gateErr == nil {
		t.Fatal("expected error")
	}
	if gateErr.Error() == "" {
		t.Error("error message must not be empty")
	}
}

// ─── Combined gate ────────────────────────────────────────────────────────────

func TestPolicyGate_RoutingViolationTakesPrecedenceOverQuota(t *testing.T) {
	// Both routing and quota are violated — routing check runs first and returns.
	gate := domain.NewPolicyGate(
		[]domain.RoutingPolicyConfig{{
			Namespace:      "prod",
			AllowedEngines: []string{"temporal"},
		}},
		[]domain.CapabilityQuotaConfig{{Namespace: "prod", MaxConcurrent: 1}},
		&stubCounter{count: 99},
	)
	err := gate.Check(context.Background(), minimalGraph("prod"), map[string]string{
		domain.AnnotationEngineHint: "argo",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Kind != domain.PolicyViolationRouting {
		t.Errorf("expected routing violation first, got %v", err.Kind)
	}
}

func TestPolicyGate_BothPoliciesPass(t *testing.T) {
	gate := domain.NewPolicyGate(
		[]domain.RoutingPolicyConfig{{
			Namespace:      "prod",
			AllowedEngines: []string{"temporal"},
		}},
		[]domain.CapabilityQuotaConfig{{Namespace: "prod", MaxConcurrent: 10}},
		&stubCounter{count: 5},
	)
	err := gate.Check(context.Background(), minimalGraph("prod"), map[string]string{
		domain.AnnotationEngineHint: "temporal",
	})
	if err != nil {
		t.Fatalf("expected nil when all policies pass, got %v", err)
	}
}
