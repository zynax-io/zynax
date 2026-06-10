// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// stubCounter is a test ActiveInvocationCounter returning a fixed count or error.
type stubCounter struct {
	count int32
	err   error
}

func (s stubCounter) ActiveCount(_ context.Context, _ string) (int32, error) {
	return s.count, s.err
}

func TestQuotaChecker_Check(t *testing.T) {
	const ns = "team-a"

	tests := []struct {
		name     string
		quotas   []CapabilityQuotaConfig
		counter  ActiveInvocationCounter
		ns       string
		wantCode codes.Code // codes.OK means "expect nil error"
	}{
		{
			name:     "quota exceeded returns RESOURCE_EXHAUSTED",
			quotas:   []CapabilityQuotaConfig{{Namespace: ns, MaxConcurrent: 2}},
			counter:  stubCounter{count: 2},
			ns:       ns,
			wantCode: codes.ResourceExhausted,
		},
		{
			name:     "active above ceiling returns RESOURCE_EXHAUSTED",
			quotas:   []CapabilityQuotaConfig{{Namespace: ns, MaxConcurrent: 2}},
			counter:  stubCounter{count: 5},
			ns:       ns,
			wantCode: codes.ResourceExhausted,
		},
		{
			name:     "active below ceiling allows dispatch",
			quotas:   []CapabilityQuotaConfig{{Namespace: ns, MaxConcurrent: 3}},
			counter:  stubCounter{count: 2},
			ns:       ns,
			wantCode: codes.OK,
		},
		{
			name:     "max concurrent zero is unbounded",
			quotas:   []CapabilityQuotaConfig{{Namespace: ns, MaxConcurrent: 0}},
			counter:  stubCounter{count: 100},
			ns:       ns,
			wantCode: codes.OK,
		},
		{
			name:     "no quota configured for namespace allows dispatch",
			quotas:   []CapabilityQuotaConfig{{Namespace: "other", MaxConcurrent: 1}},
			counter:  stubCounter{count: 100},
			ns:       ns,
			wantCode: codes.OK,
		},
		{
			name:     "nil counter disables enforcement",
			quotas:   []CapabilityQuotaConfig{{Namespace: ns, MaxConcurrent: 1}},
			counter:  nil,
			ns:       ns,
			wantCode: codes.OK,
		},
		{
			name:     "counter error is fail-open",
			quotas:   []CapabilityQuotaConfig{{Namespace: ns, MaxConcurrent: 1}},
			counter:  stubCounter{err: errors.New("backend down")},
			ns:       ns,
			wantCode: codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qc := NewQuotaChecker(tt.quotas, tt.counter)
			err := qc.Check(context.Background(), tt.ns)

			if tt.wantCode == codes.OK {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error with code %v, got nil", tt.wantCode)
			}
			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("expected code %v, got %v (err=%v)", tt.wantCode, got, err)
			}
		})
	}
}

// TestNewQuotaChecker_IgnoresEmptyNamespace verifies that quotas with an empty
// namespace are dropped during construction and therefore never enforced.
func TestNewQuotaChecker_IgnoresEmptyNamespace(t *testing.T) {
	qc := NewQuotaChecker(
		[]CapabilityQuotaConfig{{Namespace: "", MaxConcurrent: 1}},
		stubCounter{count: 100},
	)
	if err := qc.Check(context.Background(), ""); err != nil {
		t.Fatalf("empty-namespace quota must be ignored, got %v", err)
	}
}
