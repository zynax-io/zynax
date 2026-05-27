// SPDX-License-Identifier: Apache-2.0

// Package registry (whitebox test) exercises the unexported isTransient helper.
// In Go, a _test.go file with package registry (no _test suffix) has access to
// unexported symbols. Closes #717 — part of the git-adapter coverage epic (#713).
package registry

import (
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestIsTransient_TransientCodes verifies that Unavailable, Internal, and
// DeadlineExceeded are classified as transient (retry-eligible).
func TestIsTransient_TransientCodes(t *testing.T) {
	t.Parallel()
	transient := []codes.Code{codes.Unavailable, codes.Internal, codes.DeadlineExceeded}
	for _, code := range transient {
		code := code
		t.Run(code.String(), func(t *testing.T) {
			t.Parallel()
			err := status.Error(code, "test error")
			if !isTransient(err) {
				t.Errorf("isTransient(%v) = false; want true", code)
			}
		})
	}
}

// TestIsTransient_PermanentCodes verifies that NotFound, AlreadyExists,
// InvalidArgument, and OK are classified as permanent (no retry).
func TestIsTransient_PermanentCodes(t *testing.T) {
	t.Parallel()
	permanent := []codes.Code{
		codes.NotFound,
		codes.AlreadyExists,
		codes.InvalidArgument,
		codes.OK,
	}
	for _, code := range permanent {
		code := code
		t.Run(code.String(), func(t *testing.T) {
			t.Parallel()
			err := status.Error(code, "test error")
			if isTransient(err) {
				t.Errorf("isTransient(%v) = true; want false", code)
			}
		})
	}
}

// TestIsTransient_NonGRPCError verifies that a plain (non-gRPC) error is not
// transient — status.FromError returns (Unknown, false) for non-status errors.
func TestIsTransient_NonGRPCError(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("plain network error: connection refused")
	if isTransient(err) {
		t.Errorf("isTransient(non-grpc) = true; want false")
	}
}
