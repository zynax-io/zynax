// SPDX-License-Identifier: Apache-2.0

package infrastructure_test

import (
	"strings"
	"testing"

	"github.com/zynax-io/zynax/services/event-bus/internal/infrastructure"
)

func TestStreamName(t *testing.T) {
	cases := []struct {
		eventType string
		want      string
	}{
		{
			eventType: "zynax.v1.engine-adapter.workflow.completed",
			want:      "ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW",
		},
		{
			eventType: "zynax.v1.agent-registry.agent.registered",
			want:      "ZYNAX_V1_AGENT_REGISTRY_AGENT",
		},
		{
			eventType: "zynax.v1.task-broker.task.submitted",
			want:      "ZYNAX_V1_TASK_BROKER_TASK",
		},
		{
			eventType: "single",
			want:      "SINGLE",
		},
		// Event types at or below the taxonomy depth are used verbatim.
		{
			eventType: "zynax.v1.workflow.completed",
			want:      "ZYNAX_V1_WORKFLOW_COMPLETED",
		},
		// Multi-segment verbs share the entity stream (#1149 regression).
		{
			eventType: "zynax.v1.engine-adapter.workflow.state.entered",
			want:      "ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW",
		},
	}

	for _, tc := range cases {
		got := infrastructure.StreamName(tc.eventType)
		if got != tc.want {
			t.Errorf("StreamName(%q): got %q, want %q", tc.eventType, got, tc.want)
		}
	}
}

func TestSubjectFilter(t *testing.T) {
	cases := []struct {
		eventType string
		want      string
	}{
		{
			eventType: "zynax.v1.engine-adapter.workflow.completed",
			want:      "zynax.v1.engine-adapter.workflow.>",
		},
		{
			eventType: "zynax.v1.agent-registry.agent.registered",
			want:      "zynax.v1.agent-registry.agent.>",
		},
		{
			eventType: "zynax.v1.engine-adapter.workflow.state.entered",
			want:      "zynax.v1.engine-adapter.workflow.>",
		},
		// At or below the taxonomy depth the literal type is the filter —
		// literal subjects can never overlap a fixed-depth wildcard filter.
		{
			eventType: "zynax.v1.workflow.completed",
			want:      "zynax.v1.workflow.completed",
		},
		{
			eventType: "single",
			want:      "single",
		},
	}

	for _, tc := range cases {
		got := infrastructure.SubjectFilter(tc.eventType)
		if got != tc.want {
			t.Errorf("SubjectFilter(%q): got %q, want %q", tc.eventType, got, tc.want)
		}
	}
}

// subjectsOverlap reports whether two NATS subject filters can match a common
// subject. Token semantics: "*" matches exactly one token, ">" matches one or
// more remaining tokens.
func subjectsOverlap(a, b string) bool {
	at := strings.Split(a, ".")
	bt := strings.Split(b, ".")
	for i := 0; i < len(at) && i < len(bt); i++ {
		if at[i] == ">" || bt[i] == ">" {
			return true
		}
		if at[i] == "*" || bt[i] == "*" || at[i] == bt[i] {
			continue
		}
		return false
	}
	return len(at) == len(bt)
}

// TestStreamDerivation_NoOverlap_AcrossEventTypeSet is the regression test for
// #1149: deriving streams for the full platform event-type set (including the
// engine-adapter workflow lifecycle family, whose verbs have different segment
// counts, and the historical double-prefixed forms) must yield subject filters
// that are either identical (same stream) or pairwise disjoint. The previous
// "drop the last segment" derivation produced a filter for "….workflow.completed"
// that was a superset of the "….workflow.state.>" filter, and NATS rejected the
// second stream with "subjects overlap with an existing stream" (err 10065),
// silently making workflow.completed/failed undeliverable platform-wide.
func TestStreamDerivation_NoOverlap_AcrossEventTypeSet(t *testing.T) {
	eventTypes := []string{
		// Engine-adapter workflow lifecycle family (interpreter event types
		// mapped onto the topic taxonomy by lifecycleTopic in engine-adapter).
		"zynax.v1.engine-adapter.workflow.state.entered",
		"zynax.v1.engine-adapter.workflow.state.exited",
		"zynax.v1.engine-adapter.workflow.completed",
		"zynax.v1.engine-adapter.workflow.failed",
		// Historical double-prefixed forms (pre-#1149 engine-adapter topics):
		// even malformed deep types must not derive overlapping streams.
		"zynax.v1.engine-adapter.workflow.zynax.workflow.state.entered",
		"zynax.v1.engine-adapter.workflow.zynax.workflow.completed",
		// Other entity families per the taxonomy.
		"zynax.v1.agent-registry.agent.registered",
		"zynax.v1.task-broker.task.submitted",
		// Below taxonomy depth.
		"zynax.v1.workflow.completed",
		"single",
	}

	for i, a := range eventTypes {
		for _, b := range eventTypes[i+1:] {
			nameA, nameB := infrastructure.StreamName(a), infrastructure.StreamName(b)
			filterA, filterB := infrastructure.SubjectFilter(a), infrastructure.SubjectFilter(b)
			if nameA == nameB {
				if filterA != filterB {
					t.Errorf("same stream %q for %q and %q but different filters %q vs %q",
						nameA, a, b, filterA, filterB)
				}
				continue
			}
			if subjectsOverlap(filterA, filterB) {
				t.Errorf("streams %q (%q) and %q (%q) have overlapping subject filters %q vs %q — JetStream would reject the second stream (err 10065)",
					nameA, a, nameB, b, filterA, filterB)
			}
		}
	}
}

// TestStreamDerivation_LifecycleFamilySharesOneStream pins the #1149 fix shape:
// every workflow lifecycle event type — regardless of verb segment count —
// lands on the single ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW entity stream.
func TestStreamDerivation_LifecycleFamilySharesOneStream(t *testing.T) {
	const wantStream = "ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW"
	const wantFilter = "zynax.v1.engine-adapter.workflow.>"
	for _, et := range []string{
		"zynax.v1.engine-adapter.workflow.state.entered",
		"zynax.v1.engine-adapter.workflow.state.exited",
		"zynax.v1.engine-adapter.workflow.completed",
		"zynax.v1.engine-adapter.workflow.failed",
	} {
		if got := infrastructure.StreamName(et); got != wantStream {
			t.Errorf("StreamName(%q) = %q, want %q", et, got, wantStream)
		}
		if got := infrastructure.SubjectFilter(et); got != wantFilter {
			t.Errorf("SubjectFilter(%q) = %q, want %q", et, got, wantFilter)
		}
	}
}
