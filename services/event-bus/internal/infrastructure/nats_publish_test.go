// SPDX-License-Identifier: Apache-2.0

package infrastructure_test

import (
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
		{
			eventType: "zynax.v1.workflow.completed",
			want:      "ZYNAX_V1_WORKFLOW",
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
			eventType: "single",
			want:      "single.>",
		},
	}

	for _, tc := range cases {
		got := infrastructure.SubjectFilter(tc.eventType)
		if got != tc.want {
			t.Errorf("SubjectFilter(%q): got %q, want %q", tc.eventType, got, tc.want)
		}
	}
}
