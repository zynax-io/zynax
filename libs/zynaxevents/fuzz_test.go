// SPDX-License-Identifier: Apache-2.0

package zynaxevents

// Fuzz target for the subscription glob matcher (#1659 — M7 test-rigor house
// style for parser-ish code). The golden pins in testdata/golden/
// glob_matching.json remain the byte-compat spec (ADR-046); this target seeds
// from them and asserts the invariants that must hold for EVERY input:
//
//   - never panics or loops (a panic/hang fails the run)
//   - deterministic: the same inputs give the same answer twice
//   - identity: a pattern equal to the event type always matches
//   - universality: the "**" pattern matches every event type

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func FuzzMatchesGlob(f *testing.F) {
	var g struct {
		Cases []struct {
			Pattern   string `json:"pattern"`
			EventType string `json:"eventType"`
		} `json:"cases"`
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "golden", "glob_matching.json")) //nolint:gosec // fixture path built from constants
	if err != nil {
		f.Fatalf("reading golden glob_matching.json: %v", err)
	}
	if err := json.Unmarshal(raw, &g); err != nil {
		f.Fatalf("parsing golden glob_matching.json: %v", err)
	}
	for _, c := range g.Cases {
		f.Add(c.Pattern, c.EventType)
	}

	f.Fuzz(func(t *testing.T, pattern, eventType string) {
		got := MatchesGlob(pattern, eventType)
		if again := MatchesGlob(pattern, eventType); again != got {
			t.Errorf("non-deterministic: MatchesGlob(%q, %q) = %v, then %v",
				pattern, eventType, got, again)
		}
		if !MatchesGlob(eventType, eventType) {
			t.Errorf("identity violated: MatchesGlob(%q, %q) = false", eventType, eventType)
		}
		if !MatchesGlob("**", eventType) {
			t.Errorf(`universality violated: MatchesGlob("**", %q) = false`, eventType)
		}
	})
}
