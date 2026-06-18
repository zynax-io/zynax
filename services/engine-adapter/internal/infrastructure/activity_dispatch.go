// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zynax-io/zynax/services/engine-adapter/internal/domain"
)

// capabilityNotFoundErrorType is the Temporal ApplicationError Type assigned to a
// dispatch failure whose underlying gRPC status is codes.NotFound (no agent is
// registered for the requested capability). It MUST match an entry in
// nonRetryableActivityErrors so the activity RetryPolicy stops retrying.
const capabilityNotFoundErrorType = "ErrCapabilityNotFound"

// capabilityDispatcher is the subset of domain.CapabilityDispatcher this wrapper
// drives. Keeping it as an interface lets unit tests exercise the error
// classification without a live task-broker.
type capabilityDispatcher interface {
	DispatchCapabilityActivity(ctx context.Context, in domain.ActivityInput) (*domain.ActivityResult, error)
}

// DispatchActivity is the Temporal activity boundary in front of the
// domain.CapabilityDispatcher. The domain layer is Temporal-free (ADR-015), so
// the translation of a permanent gRPC failure into a non-retryable Temporal
// ApplicationError happens here rather than in domain.
type DispatchActivity struct {
	dispatcher capabilityDispatcher
}

// NewDispatchActivity wraps a domain dispatcher for registration as a Temporal activity.
func NewDispatchActivity(d capabilityDispatcher) *DispatchActivity {
	return &DispatchActivity{dispatcher: d}
}

// DispatchCapabilityActivity runs the domain dispatch and reclassifies a
// codes.NotFound result — "no agent registered for capability" — as a
// non-retryable Temporal ApplicationError of type capabilityNotFoundErrorType.
// Without this, a structurally unbacked capability would retry until
// MaximumAttempts is reached on every workflow forever (#1381); marking it
// non-retryable fails the workflow fast. Transient codes (Unavailable, deadline)
// are returned unchanged and stay retryable/bounded by the RetryPolicy.
func (a *DispatchActivity) DispatchCapabilityActivity(ctx context.Context, in domain.ActivityInput) (*domain.ActivityResult, error) {
	result, err := a.dispatcher.DispatchCapabilityActivity(ctx, in)
	if err == nil {
		return result, nil
	}
	if isCapabilityNotFound(err) {
		// Temporal sentinel: must be returned without an outer fmt.Errorf wrap so
		// the worker classifies it via errors.As against nonRetryableActivityErrors.
		return nil, temporal.NewNonRetryableApplicationError( //nolint:wrapcheck // sentinel error; re-wrapping defeats Temporal retry classification
			err.Error(), capabilityNotFoundErrorType, err,
		)
	}
	// Transient (Unavailable, deadline, ...) — keep retryable; wrap for wrapcheck
	// while preserving the chain so errors.Is still matches the underlying status.
	return nil, fmt.Errorf("engine-adapter: dispatch capability: %w", err)
}

// isCapabilityNotFound reports whether err carries a gRPC codes.NotFound status
// anywhere in its chain. status.FromError unwraps wrapped status errors, so the
// domain layer's fmt.Errorf("...: %w", grpcErr) wrapping is preserved.
func isCapabilityNotFound(err error) bool {
	if err == nil {
		return false
	}
	if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
		return true
	}
	// Defensive: walk the chain in case an intermediate wrapper hides the
	// status interface from status.FromError.
	next := err
	for next != nil {
		if st, ok := status.FromError(next); ok && st.Code() == codes.NotFound {
			return true
		}
		next = errors.Unwrap(next)
	}
	return false
}
