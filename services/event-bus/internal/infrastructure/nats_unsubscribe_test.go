// SPDX-License-Identifier: Apache-2.0

package infrastructure_test

import (
	"testing"

	"github.com/zynax-io/zynax/services/event-bus/internal/infrastructure"
)

func TestRetryBackoffLength(t *testing.T) {
	// The exported RetryBackoff slice must have exactly 5 entries to align with MaxDeliver=5.
	if len(infrastructure.RetryBackoff) != 5 {
		t.Errorf("RetryBackoff: got %d entries, want 5", len(infrastructure.RetryBackoff))
	}
}

func TestRetryBackoffOrdered(t *testing.T) {
	// Each entry must be strictly greater than the previous (ascending backoff).
	for i := 1; i < len(infrastructure.RetryBackoff); i++ {
		if infrastructure.RetryBackoff[i] <= infrastructure.RetryBackoff[i-1] {
			t.Errorf("RetryBackoff[%d]=%v <= RetryBackoff[%d]=%v — must be ascending",
				i, infrastructure.RetryBackoff[i], i-1, infrastructure.RetryBackoff[i-1])
		}
	}
}
