// SPDX-License-Identifier: Apache-2.0

package domain_test

import (
	"errors"
	"testing"

	"github.com/zynax-io/zynax/services/memory-service/internal/domain"
)

func TestValidateAccess_SameNamespace(t *testing.T) {
	t.Parallel()
	if err := domain.ValidateAccess("wf-123", "wf-123"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateAccess_CrossNamespace(t *testing.T) {
	t.Parallel()
	err := domain.ValidateAccess("wf-123", "wf-456")
	if err == nil {
		t.Fatal("expected ErrCrossNamespaceAccess, got nil")
	}
	if !errors.Is(err, domain.ErrCrossNamespaceAccess) {
		t.Fatalf("expected ErrCrossNamespaceAccess, got %v", err)
	}
}

func TestValidateAccess_EmptyRequestNamespace(t *testing.T) {
	t.Parallel()
	err := domain.ValidateAccess("", "wf-123")
	if err == nil {
		t.Fatal("expected ErrEmptyNamespace, got nil")
	}
	if !errors.Is(err, domain.ErrEmptyNamespace) {
		t.Fatalf("expected ErrEmptyNamespace, got %v", err)
	}
}

func TestValidateAccess_EmptyResourceNamespace(t *testing.T) {
	t.Parallel()
	err := domain.ValidateAccess("wf-123", "")
	if err == nil {
		t.Fatal("expected ErrEmptyNamespace, got nil")
	}
	if !errors.Is(err, domain.ErrEmptyNamespace) {
		t.Fatalf("expected ErrEmptyNamespace, got %v", err)
	}
}

func TestValidateAccess_BothEmpty(t *testing.T) {
	t.Parallel()
	err := domain.ValidateAccess("", "")
	if err == nil {
		t.Fatal("expected ErrEmptyNamespace, got nil")
	}
	if !errors.Is(err, domain.ErrEmptyNamespace) {
		t.Fatalf("expected ErrEmptyNamespace, got %v", err)
	}
}

func TestKVKey_Format(t *testing.T) {
	t.Parallel()
	got := domain.KVKey("wf-123", "my-key")
	want := "wf-123:my-key"
	if got != want {
		t.Fatalf("KVKey = %q; want %q", got, want)
	}
}

func TestKVKeyPrefix_Format(t *testing.T) {
	t.Parallel()
	got := domain.KVKeyPrefix("wf-123")
	want := "wf-123:"
	if got != want {
		t.Fatalf("KVKeyPrefix = %q; want %q", got, want)
	}
}
