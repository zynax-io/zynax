package domain_test

import (
	"testing"

	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
)

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
	)
	// The restriction should not apply — the empty-namespace entry is ignored.
	err := gate.Check(minimalGraph("default"), map[string]string{
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
	)
	err := gate.Check(minimalGraph("prod"), map[string]string{
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
	)
	err := gate.Check(minimalGraph("prod"), map[string]string{
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
	)
	err := gate.Check(minimalGraph("prod"), map[string]string{
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
	)
	err := gate.Check(minimalGraph("prod"), map[string]string{})
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
	)
	err := gate.Check(minimalGraph("staging"), map[string]string{
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
	)
	gateErr := gate.Check(minimalGraph("prod"), map[string]string{
		domain.AnnotationEngineHint: "argo",
	})
	if gateErr == nil {
		t.Fatal("expected error")
	}
	if gateErr.Error() == "" {
		t.Error("error message must not be empty")
	}
}
