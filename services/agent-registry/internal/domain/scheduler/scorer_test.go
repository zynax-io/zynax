// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"errors"
	"testing"
)

// fakeMetrics is the in-process metrics stub; Fail simulates an outage.
type fakeMetrics struct {
	data map[string]Metrics
	fail bool
}

func (f *fakeMetrics) Snapshot(_ context.Context, keys []string) (map[string]Metrics, error) {
	if f.fail {
		return nil, ErrMetricsUnavailable
	}
	out := make(map[string]Metrics, len(keys))
	for _, k := range keys {
		out[k] = f.data[k]
	}
	return out, nil
}

func scoredCand(key string, ready bool, capability Capability) Candidate {
	return Candidate{Key: key, Name: key, Endpoint: key + ".default.svc:50061", Ready: ready,
		Capabilities: []Capability{capability}}
}

func reviewCap() Capability {
	return Capability{
		ID:        "review",
		Selectors: Selectors{Language: []string{"go"}, Tags: []string{"fast"}},
		Cost:      Cost{LatencyClass: "low"},
		Resources: Resources{GPU: 0},
		Models:    []string{"qwen2.5-coder:3b"},
		Protocols: []string{"A2A", "HTTP"},
	}
}

func newIdx(cands ...Candidate) *Index {
	idx := NewIndex()
	for _, c := range cands {
		idx.Upsert(c)
	}
	return idx
}

const lessLoaded = "default/b"

func TestSelect_LeastLoadedWins(t *testing.T) {
	idx := newIdx(
		scoredCand("default/a", true, reviewCap()),
		scoredCand(lessLoaded, true, reviewCap()),
	)
	s := &Scorer{Metrics: &fakeMetrics{data: map[string]Metrics{
		"default/a": {Load: 0.9, LatencyP50Ms: 200},
		lessLoaded:  {Load: 0.1, LatencyP50Ms: 80},
	}}}

	res, err := s.Select(context.Background(), idx, Request{Capability: "review"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if res.Chosen.Key != lessLoaded {
		t.Errorf("chosen = %s, want the less-loaded default/b", res.Chosen.Key)
	}
	r := res.Rationale
	if r.CandidatesMatched != 2 || r.CandidatesReady != 2 || r.CandidatesAfterExpertFilter != 2 {
		t.Errorf("counts = %+v", r)
	}
	if r.Mode != ModeMetricsWeighted || len(r.WinningFactors) == 0 || r.Summary == "" {
		t.Errorf("rationale = %+v", r)
	}
	if res.Capability.ID != "review" {
		t.Errorf("capability = %q (ADR-028 binding needs the matching capability)", res.Capability.ID)
	}
}

func TestSelect_NoCapability(t *testing.T) {
	s := &Scorer{Metrics: &fakeMetrics{}}
	_, err := s.Select(context.Background(), newIdx(), Request{Capability: "ghost"})
	if !errors.Is(err, ErrNoCapability) {
		t.Fatalf("err = %v, want ErrNoCapability", err)
	}
}

func TestSelect_ConstraintsEliminateAll(t *testing.T) {
	idx := newIdx(scoredCand("default/a", true, reviewCap()))
	s := &Scorer{Metrics: &fakeMetrics{}}

	cases := []struct {
		name  string
		k     Constraints
		stage string
	}{
		{"gpu", Constraints{RequireGPU: true}, "gpu"},
		{"tags", Constraints{RequiredTags: []string{"thorough"}}, "tags"},
		{"language", Constraints{RequiredLanguage: "rust"}, "language"},
		{"model", Constraints{RequiredModel: "gpt-x"}, "model"},
		{"protocols", Constraints{RequiredProtocols: []string{"MCP"}}, "protocols"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.Select(context.Background(), idx, Request{Capability: "review", Constraints: tc.k})
			var fe *FilteredError
			if !errors.As(err, &fe) {
				t.Fatalf("err = %v, want FilteredError", err)
			}
			if fe.Stage != tc.stage {
				t.Errorf("stage = %q, want %q", fe.Stage, tc.stage)
			}
		})
	}
}

func TestSelect_ConstraintsMatchFold(t *testing.T) {
	idx := newIdx(scoredCand("default/a", true, reviewCap()))
	s := &Scorer{Metrics: &fakeMetrics{}}
	// Case-insensitive protocol + language + tag + model match must pass.
	res, err := s.Select(context.Background(), idx, Request{Capability: "review", Constraints: Constraints{
		RequiredTags:      []string{"FAST"},
		RequiredLanguage:  "GO",
		RequiredModel:     "QWEN2.5-CODER:3B",
		RequiredProtocols: []string{"http"},
	}})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if res.Rationale.CandidatesAfterConstraints != 1 {
		t.Errorf("after constraints = %d", res.Rationale.CandidatesAfterConstraints)
	}
}

func TestSelect_ReadinessFiltersDeadAgents(t *testing.T) {
	idx := newIdx(
		scoredCand("default/alive", true, reviewCap()),
		scoredCand("default/dead", false, reviewCap()),
	)
	s := &Scorer{Metrics: &fakeMetrics{}}
	res, err := s.Select(context.Background(), idx, Request{Capability: "review"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if res.Chosen.Key != "default/alive" {
		t.Errorf("chosen = %s", res.Chosen.Key)
	}
	if res.Rationale.CandidatesMatched != 2 || res.Rationale.CandidatesReady != 1 {
		t.Errorf("counts = %+v", res.Rationale)
	}
}

func TestSelect_AllNotReadyFails(t *testing.T) {
	idx := newIdx(scoredCand("default/dead", false, reviewCap()))
	s := &Scorer{Metrics: &fakeMetrics{}}
	_, err := s.Select(context.Background(), idx, Request{Capability: "review"})
	var fe *FilteredError
	if !errors.As(err, &fe) || fe.Stage != "ready" {
		t.Fatalf("err = %v, want FilteredError(ready)", err)
	}
}

func TestSelect_ExpertStrictNoFallback(t *testing.T) {
	sec := scoredCand("default/sec", true, reviewCap())
	sec.ExpertScope = "security-reviewer"
	idx := newIdx(sec, scoredCand("default/generalist", true, reviewCap()))
	s := &Scorer{Metrics: &fakeMetrics{}}

	res, err := s.Select(context.Background(), idx, Request{Capability: "review", ExpertTarget: "security-reviewer"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if res.Chosen.Key != "default/sec" || res.Rationale.CandidatesAfterExpertFilter != 1 {
		t.Errorf("chosen=%s counts=%+v", res.Chosen.Key, res.Rationale)
	}

	// No declared expert => FAILED_PRECONDITION-class error, never a fallback.
	_, err = s.Select(context.Background(), idx, Request{Capability: "review", ExpertTarget: "compliance-reviewer"})
	var fe *FilteredError
	if !errors.As(err, &fe) || fe.Stage != "expert:compliance-reviewer" {
		t.Fatalf("err = %v, want FilteredError(expert:compliance-reviewer)", err)
	}
}

func TestSelect_MetricsDownDegradesNeverFails(t *testing.T) {
	idx := newIdx(
		scoredCand("default/a", true, reviewCap()),
		scoredCand("default/b", true, reviewCap()),
	)
	s := &Scorer{Metrics: &fakeMetrics{fail: true}}

	seen := map[string]bool{}
	for range 4 {
		res, err := s.Select(context.Background(), idx, Request{Capability: "review"})
		if err != nil {
			t.Fatalf("degraded selection must not fail: %v", err)
		}
		if res.Rationale.Mode != ModeDegradedRoundRobin {
			t.Fatalf("mode = %v, want degraded", res.Rationale.Mode)
		}
		seen[res.Chosen.Key] = true
	}
	// Genuine rotation: both ready candidates receive work across calls.
	if !seen["default/a"] || !seen[lessLoaded] {
		t.Errorf("rotation did not spread: %v", seen)
	}
}

func TestSelect_ExplicitRoundRobinMode(t *testing.T) {
	idx := newIdx(scoredCand("default/a", true, reviewCap()))
	s := &Scorer{Metrics: &fakeMetrics{}}
	res, err := s.Select(context.Background(), idx, Request{Capability: "review", RoundRobin: true})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if res.Rationale.Mode != ModeRoundRobin {
		t.Errorf("mode = %v, want ModeRoundRobin", res.Rationale.Mode)
	}
	if res.Rationale.WinningFactors[0] != "rotation" {
		t.Errorf("factors = %v", res.Rationale.WinningFactors)
	}
}

func TestSelect_CostTieBreak(t *testing.T) {
	cheap := reviewCap()
	cheap.Cost = Cost{LatencyClass: "low"}
	pricey := reviewCap()
	pricey.Cost = Cost{LatencyClass: "high"}
	idx := newIdx(
		scoredCand("default/pricey", true, pricey),
		scoredCand("default/cheap", true, cheap),
	)
	// Identical live metrics => the static cost hint decides.
	s := &Scorer{Metrics: &fakeMetrics{data: map[string]Metrics{
		"default/pricey": {Load: 0.5, LatencyP50Ms: 100},
		"default/cheap":  {Load: 0.5, LatencyP50Ms: 100},
	}}}
	res, err := s.Select(context.Background(), idx, Request{Capability: "review"})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if res.Chosen.Key != "default/cheap" {
		t.Errorf("chosen = %s, want cost tie-break winner default/cheap", res.Chosen.Key)
	}
}

func TestCostScore_TokenPriceFallback(t *testing.T) {
	if got := costScore(Cost{TokenPrice: 0.7}); got != 0.7 {
		t.Errorf("costScore fallback = %v", got)
	}
	if got := costScore(Cost{LatencyClass: "medium"}); got != 0.5 {
		t.Errorf("costScore medium = %v", got)
	}
}
