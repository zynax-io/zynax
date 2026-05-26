// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"testing"
)

func TestDefaultActivityMaxAttempts_Default(t *testing.T) {
	if DefaultActivityMaxAttempts != 3 {
		t.Errorf("DefaultActivityMaxAttempts = %d; want 3", DefaultActivityMaxAttempts)
	}
}

func TestNonRetryableActivityErrors_Contents(t *testing.T) {
	want := map[string]bool{
		"ErrCapabilityNotFound": true,
		"ErrTaskTerminal":       true,
		"ErrInvalidArgument":    true,
	}
	if len(nonRetryableActivityErrors) != len(want) {
		t.Fatalf("nonRetryableActivityErrors len = %d; want %d", len(nonRetryableActivityErrors), len(want))
	}
	for _, v := range nonRetryableActivityErrors {
		if !want[v] {
			t.Errorf("unexpected entry in nonRetryableActivityErrors: %q", v)
		}
	}
}
