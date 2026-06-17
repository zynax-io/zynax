// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"testing"
)

func TestRequestIDFromContext_Present(t *testing.T) {
	ctx := WithRequestID(context.Background(), "trace-123")
	if got := RequestIDFromContext(ctx); got != "trace-123" {
		t.Errorf("RequestIDFromContext = %q; want %q", got, "trace-123")
	}
}

func TestRequestIDFromContext_Absent(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Errorf("RequestIDFromContext on empty ctx = %q; want empty string", got)
	}
}

func TestNamespaceContextRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		set  bool
		ns   string
		want string
	}{
		{"present", true, "team-a", "team-a"},
		{"empty stored", true, "", ""},
		{"absent", false, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.set {
				ctx = WithNamespace(ctx, tt.ns)
			}
			if got := NamespaceFromContext(ctx); got != tt.want {
				t.Errorf("NamespaceFromContext = %q; want %q", got, tt.want)
			}
		})
	}
}

// TestCorrelationKeysDistinct verifies request ID and namespace use separate
// context keys so attaching one never overwrites the other.
func TestCorrelationKeysDistinct(t *testing.T) {
	ctx := WithNamespace(WithRequestID(context.Background(), "req-1"), "ns-1")
	if got := RequestIDFromContext(ctx); got != "req-1" {
		t.Errorf("RequestIDFromContext = %q; want %q", got, "req-1")
	}
	if got := NamespaceFromContext(ctx); got != "ns-1" {
		t.Errorf("NamespaceFromContext = %q; want %q", got, "ns-1")
	}
}
