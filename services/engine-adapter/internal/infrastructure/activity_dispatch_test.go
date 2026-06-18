// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
)

// fakeDispatcher returns a fixed (result, err) pair, standing in for the domain
// CapabilityDispatcher without a live task-broker.
type fakeDispatcher struct {
	result *domain.ActivityResult
	err    error
}

func (f *fakeDispatcher) DispatchCapabilityActivity(_ context.Context, _ domain.ActivityInput) (*domain.ActivityResult, error) {
	return f.result, f.err
}

// asNonRetryable extracts a *temporal.ApplicationError from err and reports its
// non-retryable flag and Type, or fails the test if err is not one.
func asApplicationError(t *testing.T, err error) *temporal.ApplicationError {
	t.Helper()
	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("error %v is not a *temporal.ApplicationError", err)
	}
	return appErr
}

func TestDispatchActivity_NotFound_IsNonRetryable(t *testing.T) {
	// The domain layer wraps the broker gRPC error with fmt.Errorf("...: %w").
	grpcErr := status.Error(codes.NotFound, `no agent registered for capability "request_review"`)
	wrapped := fmt.Errorf("engine-adapter: dispatch capability %q: %w", "request_review", grpcErr)

	act := NewDispatchActivity(&fakeDispatcher{err: wrapped})

	_, err := act.DispatchCapabilityActivity(context.Background(), domain.ActivityInput{
		CapabilityName: "request_review",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr := asApplicationError(t, err)
	if !appErr.NonRetryable() {
		t.Error("NotFound dispatch error must be marked non-retryable")
	}
	if appErr.Type() != capabilityNotFoundErrorType {
		t.Errorf("ApplicationError Type = %q; want %q", appErr.Type(), capabilityNotFoundErrorType)
	}
	// The non-retryable Type must be in the workflow's RetryPolicy block list so
	// Temporal actually stops retrying.
	found := false
	for _, v := range nonRetryableActivityErrors {
		if v == appErr.Type() {
			found = true
		}
	}
	if !found {
		t.Errorf("Type %q not in nonRetryableActivityErrors; retries would not stop", appErr.Type())
	}
}

func TestDispatchActivity_Unavailable_StaysRetryable(t *testing.T) {
	grpcErr := status.Error(codes.Unavailable, "task-broker connection refused")
	wrapped := fmt.Errorf("engine-adapter: dispatch capability %q: %w", "search_web", grpcErr)

	act := NewDispatchActivity(&fakeDispatcher{err: wrapped})

	_, err := act.DispatchCapabilityActivity(context.Background(), domain.ActivityInput{
		CapabilityName: "search_web",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// A transient error must NOT be reclassified as a non-retryable
	// ApplicationError; it is returned verbatim so the RetryPolicy retries it
	// up to MaximumAttempts.
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.NonRetryable() {
		t.Error("Unavailable dispatch error must stay retryable, not become non-retryable")
	}
	if !errors.Is(err, grpcErr) {
		t.Error("transient error should be returned unchanged (wrapping preserved)")
	}
}

func TestDispatchActivity_DeadlineExceeded_StaysRetryable(t *testing.T) {
	grpcErr := status.Error(codes.DeadlineExceeded, "broker call timed out")
	act := NewDispatchActivity(&fakeDispatcher{err: grpcErr})

	_, err := act.DispatchCapabilityActivity(context.Background(), domain.ActivityInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.NonRetryable() {
		t.Error("DeadlineExceeded must stay retryable")
	}
}

func TestDispatchActivity_Success_PassesThrough(t *testing.T) {
	want := &domain.ActivityResult{EventType: "review.approved"}
	act := NewDispatchActivity(&fakeDispatcher{result: want})

	got, err := act.DispatchCapabilityActivity(context.Background(), domain.ActivityInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.EventType != want.EventType {
		t.Errorf("EventType = %q; want %q", got.EventType, want.EventType)
	}
}

func TestIsCapabilityNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("boom"), false},
		{"bare NotFound", status.Error(codes.NotFound, "missing"), true},
		{"wrapped NotFound", fmt.Errorf("ctx: %w", status.Error(codes.NotFound, "missing")), true},
		{"Unavailable", status.Error(codes.Unavailable, "down"), false},
		{"wrapped Unavailable", fmt.Errorf("ctx: %w", status.Error(codes.Unavailable, "down")), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCapabilityNotFound(tt.err); got != tt.want {
				t.Errorf("isCapabilityNotFound(%v) = %v; want %v", tt.err, got, tt.want)
			}
		})
	}
}
