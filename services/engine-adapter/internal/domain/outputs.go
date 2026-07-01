// SPDX-License-Identifier: Apache-2.0

package domain

import "fmt"

// Output-safety size bounds (ADR-042 §6). Workflow outputs are attacker-influenced
// and are rendered to terminal, SSE, logs, and gateway JSON, so they are bounded at
// capture. Oversized outputs are REJECTED with a typed error — never silently
// truncated — so a workflow that emits very large values fails loudly and must be
// redesigned to emit a reference/handle instead.
const (
	// MaxOutputValueBytes bounds a single resolved output value.
	MaxOutputValueBytes = 64 * 1024
	// MaxOutputsTotalBytes bounds the sum of all output names and values.
	MaxOutputsTotalBytes = 256 * 1024
)

// OutputSizeError is returned when a captured output exceeds a per-value or total
// size bound (ADR-042 §6). It carries the offending key (empty for a total
// overflow) so the failure is actionable.
type OutputSizeError struct {
	// Key is the output name that overflowed, or "" for a total-size overflow.
	Key string
	// Size is the measured byte size that exceeded the bound.
	Size int
	// Limit is the bound that was exceeded.
	Limit int
}

func (e *OutputSizeError) Error() string {
	if e.Key == "" {
		return fmt.Sprintf(
			"engine-adapter: workflow outputs total %d bytes exceed the %d-byte limit; emit a reference/handle instead",
			e.Size, e.Limit,
		)
	}
	return fmt.Sprintf(
		"engine-adapter: output %q is %d bytes, exceeding the %d-byte per-value limit; emit a reference/handle instead",
		e.Key, e.Size, e.Limit,
	)
}

// enforceOutputBounds rejects oversized captured outputs with a typed
// OutputSizeError (ADR-042 §6). A nil/empty map passes. The per-value bound is
// checked first so the error names the offending key; otherwise the total
// (names + values) is checked.
func enforceOutputBounds(outputs map[string]string) error {
	total := 0
	for k, v := range outputs {
		if len(v) > MaxOutputValueBytes {
			return &OutputSizeError{Key: k, Size: len(v), Limit: MaxOutputValueBytes}
		}
		total += len(k) + len(v)
	}
	if total > MaxOutputsTotalBytes {
		return &OutputSizeError{Key: "", Size: total, Limit: MaxOutputsTotalBytes}
	}
	return nil
}
