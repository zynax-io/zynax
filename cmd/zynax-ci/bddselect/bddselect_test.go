// SPDX-License-Identifier: Apache-2.0

package bddselect

import "testing"

func TestSelect(t *testing.T) {
	tests := []struct {
		name    string
		changed []string
		want    string
	}{
		{"empty_no_change", nil, ""},
		{"blank_lines_ignored", []string{"", "  "}, ""},
		{
			"single_proto_maps",
			[]string{"protos/zynax/v1/agent.proto"},
			"agent_service",
		},
		{
			"unknown_proto_ignored",
			[]string{"protos/zynax/v1/unknown.proto"},
			"",
		},
		{
			"two_protos_sorted_unique",
			[]string{"protos/zynax/v1/memory.proto", "protos/zynax/v1/agent.proto", "protos/zynax/v1/agent.proto"},
			"agent_service memory_service",
		},
		{
			"go_mod_forces_all",
			[]string{"protos/tests/go.mod", "protos/zynax/v1/agent.proto"},
			All,
		},
		{
			"go_sum_forces_all",
			[]string{"protos/tests/go.sum"},
			All,
		},
		{
			"testserver_forces_all",
			[]string{"protos/tests/testserver/server.go"},
			All,
		},
		{
			"features_forces_all",
			[]string{"protos/tests/features/agent.feature"},
			All,
		},
		{
			"tests_pkg_dir_selects",
			[]string{"protos/tests/agent_service/steps_test.go"},
			"agent_service",
		},
		{
			"tests_features_dir_not_treated_as_pkg",
			[]string{"protos/tests/features/x.feature", "protos/zynax/v1/memory.proto"},
			All,
		},
		{
			"mixed_proto_and_tests_dir",
			[]string{"protos/zynax/v1/agent.proto", "protos/tests/memory_service/foo_test.go"},
			"agent_service memory_service",
		},
		{
			"tests_single_segment_ignored",
			[]string{"protos/tests/README.md"},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Select(tt.changed); got != tt.want {
				t.Fatalf("Select(%v)=%q want %q", tt.changed, got, tt.want)
			}
		})
	}
}
