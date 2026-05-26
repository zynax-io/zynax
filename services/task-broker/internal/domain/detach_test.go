// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"testing"
)

// ── detach ────────────────────────────────────────────────────────────────

func TestDetach_CancelledParentDoesNotCancelChild(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	child := detach(ctx)
	if child.Err() != nil {
		t.Errorf("detached context should not inherit cancellation error, got %v", child.Err())
	}
	_, hasDeadline := child.Deadline()
	if hasDeadline {
		t.Error("detached context should report no deadline")
	}
	select {
	case <-child.Done():
		t.Error("detached context Done() should never close")
	default:
		// correct — nil channel never receives
	}
}

func TestDetach_PreservesContextValues(t *testing.T) {
	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "req-id-42")

	child := detach(ctx)
	if got := child.Value(key{}); got != "req-id-42" {
		t.Errorf("expected value %q preserved in detached ctx, got %v", "req-id-42", got)
	}
}

func TestDetach_NilValueForAbsentKey(t *testing.T) {
	type key struct{}
	child := detach(context.Background())
	if got := child.Value(key{}); got != nil {
		t.Errorf("expected nil for absent key, got %v", got)
	}
}
