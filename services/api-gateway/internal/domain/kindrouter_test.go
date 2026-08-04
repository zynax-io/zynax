// SPDX-License-Identifier: Apache-2.0

package domain_test

import (
	"errors"
	"testing"

	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
)

func TestDetectKind_Workflow(t *testing.T) {
	yaml := []byte("kind: Workflow\napiVersion: zynax.io/v1alpha1\n")
	got, err := domain.DetectKind(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != domain.KindWorkflow {
		t.Errorf("got %q, want %q", got, domain.KindWorkflow)
	}
}

// TestDetectKind_AgentDefRemoved pins the M9.A contract (ADR-039): kind:
// AgentDef left the allowlist with the push path, so it is now an unknown kind.
func TestDetectKind_AgentDefRemoved(t *testing.T) {
	yaml := []byte("kind: AgentDef\napiVersion: zynax.io/v1alpha1\n")
	_, err := domain.DetectKind(yaml)
	if !errors.Is(err, domain.ErrUnknownKind) {
		t.Errorf("got %v, want ErrUnknownKind", err)
	}
}

func TestDetectKind_Unknown(t *testing.T) {
	yaml := []byte("kind: SomethingElse\n")
	_, err := domain.DetectKind(yaml)
	if !errors.Is(err, domain.ErrUnknownKind) {
		t.Errorf("got %v, want ErrUnknownKind", err)
	}
}

func TestDetectKind_MissingField(t *testing.T) {
	yaml := []byte("apiVersion: zynax.io/v1alpha1\nspec: {}\n")
	_, err := domain.DetectKind(yaml)
	if !errors.Is(err, domain.ErrUnknownKind) {
		t.Errorf("got %v, want ErrUnknownKind", err)
	}
}

func TestDetectKind_EmptyKind(t *testing.T) {
	yaml := []byte("kind: \"\"\n")
	_, err := domain.DetectKind(yaml)
	if !errors.Is(err, domain.ErrUnknownKind) {
		t.Errorf("got %v, want ErrUnknownKind", err)
	}
}

func TestDetectKind_InvalidYAML(t *testing.T) {
	yaml := []byte(":\t invalid: [[[")
	_, err := domain.DetectKind(yaml)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
	if errors.Is(err, domain.ErrUnknownKind) {
		t.Error("invalid YAML should not return ErrUnknownKind")
	}
}
