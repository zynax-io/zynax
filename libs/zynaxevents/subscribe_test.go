// SPDX-License-Identifier: Apache-2.0

package zynaxevents_test

import (
	"testing"

	"github.com/zynax-io/zynax/libs/zynaxevents"
)

func TestMatchesGlob_ExactMatch(t *testing.T) {
	cases := []struct {
		pattern   string
		eventType string
		want      bool
	}{
		{"zynax.v1.workflow.completed", "zynax.v1.workflow.completed", true},
		{"zynax.v1.workflow.completed", "zynax.v1.workflow.failed", false},
		{"zynax.v1.workflow.completed", "zynax.v1.task.completed", false},
	}
	for _, tc := range cases {
		got := zynaxevents.MatchesGlob(tc.pattern, tc.eventType)
		if got != tc.want {
			t.Errorf("MatchesGlob(%q, %q) = %v, want %v", tc.pattern, tc.eventType, got, tc.want)
		}
	}
}

func TestMatchesGlob_SingleSegmentWildcard(t *testing.T) {
	cases := []struct {
		pattern   string
		eventType string
		want      bool
	}{
		// "*" matches exactly one segment
		{"zynax.v1.workflow.*", "zynax.v1.workflow.completed", true},
		{"zynax.v1.workflow.*", "zynax.v1.workflow.failed", true},
		// "*" does NOT match across multiple segments
		{"zynax.v1.*", "zynax.v1.workflow.completed", false},
		// "*" in the middle
		{"zynax.*.workflow.completed", "zynax.v1.workflow.completed", true},
		{"zynax.*.workflow.completed", "zynax.v2.workflow.completed", true},
		{"zynax.*.workflow.completed", "zynax.v1.task.completed", false},
	}
	for _, tc := range cases {
		got := zynaxevents.MatchesGlob(tc.pattern, tc.eventType)
		if got != tc.want {
			t.Errorf("MatchesGlob(%q, %q) = %v, want %v", tc.pattern, tc.eventType, got, tc.want)
		}
	}
}

func TestMatchesGlob_MultiSegmentWildcard(t *testing.T) {
	cases := []struct {
		pattern   string
		eventType string
		want      bool
	}{
		// "**" matches zero or more segments
		{"zynax.v1.**", "zynax.v1.workflow.completed", true},
		{"zynax.v1.**", "zynax.v1.task.broker.submitted", true},
		{"zynax.v1.**", "zynax.v1", true}, // zero segments
		{"zynax.**", "zynax.v1", true},
		{"zynax.**", "zynax.v1.workflow.completed", true},
		// "**" at end matches anything remaining
		{"**.completed", "zynax.v1.workflow.completed", true},
		{"**.completed", "completed", true},
		// "**" sandwiched
		{"zynax.**.completed", "zynax.v1.workflow.completed", true},
		{"zynax.**.completed", "zynax.completed", true},
		// Mismatch
		{"zynax.v1.**", "other.v1.workflow", false},
	}
	for _, tc := range cases {
		got := zynaxevents.MatchesGlob(tc.pattern, tc.eventType)
		if got != tc.want {
			t.Errorf("MatchesGlob(%q, %q) = %v, want %v", tc.pattern, tc.eventType, got, tc.want)
		}
	}
}

func TestMatchesGlob_TopicIsolation(t *testing.T) {
	// A subscriber on one topic must NOT receive events from another topic.
	cases := []struct {
		subscriberPattern string
		publishedType     string
		want              bool
	}{
		{"zynax.v1.workflow.*", "zynax.v1.task.submitted", false},
		{"zynax.v1.task.*", "zynax.v1.workflow.completed", false},
		{"zynax.v1.workflow.*", "zynax.v1.workflow.completed", true},
		{"zynax.v1.task.*", "zynax.v1.task.submitted", true},
	}
	for _, tc := range cases {
		got := zynaxevents.MatchesGlob(tc.subscriberPattern, tc.publishedType)
		if got != tc.want {
			t.Errorf("TopicIsolation: MatchesGlob(%q, %q) = %v, want %v",
				tc.subscriberPattern, tc.publishedType, got, tc.want)
		}
	}
}

func TestDurableConsumerName_Sanitizes(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"service-A", "service-A"},
		{"service.B", "service_B"},
		{"my subscriber 1", "my_subscriber_1"},
		{"sub:99/test", "sub_99_test"},
		{"valid-sub-123", "valid-sub-123"},
	}
	for _, tc := range cases {
		got := zynaxevents.DurableConsumerName(tc.input)
		if got != tc.want {
			t.Errorf("DurableConsumerName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDurableConsumerName_Truncates(t *testing.T) {
	long := make([]byte, 250)
	for i := range long {
		long[i] = 'a'
	}
	got := zynaxevents.DurableConsumerName(string(long))
	if len(got) != 200 {
		t.Errorf("DurableConsumerName truncation: got len %d, want 200", len(got))
	}
}

func TestStreamSubjectFromPattern(t *testing.T) {
	cases := []struct {
		pattern string
		want    string
	}{
		{"zynax.v1.workflow.*", "zynax.v1.workflow.x"},
		{"zynax.v1.**", "zynax.v1.x"},
		{"zynax.v1.workflow.completed", "zynax.v1.workflow.completed"},
		{"zynax.**", "zynax.x"},
		{"*", "x"},
	}
	for _, tc := range cases {
		got := zynaxevents.StreamSubjectFromPattern(tc.pattern)
		if got != tc.want {
			t.Errorf("StreamSubjectFromPattern(%q) = %q, want %q", tc.pattern, got, tc.want)
		}
	}
}

func TestRetryBackoffLength(t *testing.T) {
	// The exported RetryBackoff slice must have exactly 5 entries to align with MaxDeliver=5.
	if len(zynaxevents.RetryBackoff) != 5 {
		t.Errorf("RetryBackoff: got %d entries, want 5", len(zynaxevents.RetryBackoff))
	}
}

func TestRetryBackoffOrdered(t *testing.T) {
	// Each entry must be strictly greater than the previous (ascending backoff).
	for i := 1; i < len(zynaxevents.RetryBackoff); i++ {
		if zynaxevents.RetryBackoff[i] <= zynaxevents.RetryBackoff[i-1] {
			t.Errorf("RetryBackoff[%d]=%v <= RetryBackoff[%d]=%v — must be ascending",
				i, zynaxevents.RetryBackoff[i], i-1, zynaxevents.RetryBackoff[i-1])
		}
	}
}
