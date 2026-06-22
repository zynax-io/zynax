// SPDX-License-Identifier: Apache-2.0

package scorer

import (
	"context"
	"errors"
	"testing"
)

// cap is a small helper for building a capability in tests.
func cap(id string, opts ...func(*Capability)) Capability {
	c := Capability{ID: id}
	for _, o := range opts {
		o(&c)
	}
	return c
}

func withLang(langs ...string) func(*Capability) {
	return func(c *Capability) { c.Selectors.Language = langs }
}
func withTags(tags ...string) func(*Capability) {
	return func(c *Capability) { c.Selectors.Tags = tags }
}
func withModels(m ...string) func(*Capability) { return func(c *Capability) { c.Models = m } }
func withGPU(n int) func(*Capability)          { return func(c *Capability) { c.Resources.GPU = n } }
func withProtocols(p ...string) func(*Capability) {
	return func(c *Capability) { c.Protocols = p }
}
func withClass(cls string) func(*Capability) {
	return func(c *Capability) { c.Cost.LatencyClass = cls }
}

func idxWith(cs ...Candidate) *Index {
	i := NewIndex()
	for _, c := range cs {
		i.Upsert(c)
	}
	return i
}

func mustSelect(t *testing.T, s *Scorer, idx *Index, req Request) Result {
	t.Helper()
	r, err := s.Select(context.Background(), idx, req)
	if err != nil {
		t.Fatalf("Select(%q): unexpected error: %v", req.Capability, err)
	}
	return r
}

func TestSelect_NotFoundWhenNoAgentDeclaresCapability(t *testing.T) {
	s := &Scorer{Metrics: &FakeMetrics{}}
	idx := idxWith(Candidate{Key: "default/a", Name: "a", Ready: true, Capabilities: []Capability{cap("review")}})
	_, err := s.Select(context.Background(), idx, Request{Capability: "deploy"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// The stale-liveness fix: an agent that declares the capability but is not Ready
// (e.g. crashed without deregistering) must never be selected. The old push
// registry had no such signal and would round-robin into the dead endpoint.
func TestSelect_SkipsNotReadyAgent(t *testing.T) {
	s := &Scorer{Metrics: &FakeMetrics{Data: map[string]Metrics{"default/live": {Load: 0.5}}}}
	idx := idxWith(
		Candidate{Key: "default/dead", Name: "dead", Ready: false, Capabilities: []Capability{cap("review")}},
		Candidate{Key: "default/live", Name: "live", Ready: true, Capabilities: []Capability{cap("review")}},
	)
	r := mustSelect(t, s, idx, Request{Capability: "review"})
	if r.Chosen.Name != "live" {
		t.Fatalf("want live agent, got %q", r.Chosen.Name)
	}
}

func TestSelect_NotFoundWhenAllUnready(t *testing.T) {
	s := &Scorer{Metrics: &FakeMetrics{}}
	idx := idxWith(Candidate{Key: "default/dead", Ready: false, Capabilities: []Capability{cap("review")}})
	if _, err := s.Select(context.Background(), idx, Request{Capability: "review"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound when all unready, got %v", err)
	}
}

func TestSelect_HardConstraints(t *testing.T) {
	s := &Scorer{Metrics: &FakeMetrics{Data: map[string]Metrics{
		"default/go": {}, "default/py": {},
	}}}
	idx := idxWith(
		Candidate{Key: "default/go", Name: "go", Ready: true, Capabilities: []Capability{
			cap("review", withLang("go"), withTags("fast"), withModels("qwen"), withProtocols("A2A")),
		}},
		Candidate{Key: "default/py", Name: "py", Ready: true, Capabilities: []Capability{
			cap("review", withLang("python"), withProtocols("HTTP")),
		}},
	)

	r := mustSelect(t, s, idx, Request{Capability: "review", Constraints: Constraints{RequiredLanguage: "go"}})
	if r.Chosen.Name != "go" {
		t.Fatalf("language constraint: want go, got %q", r.Chosen.Name)
	}
	r = mustSelect(t, s, idx, Request{Capability: "review", Constraints: Constraints{RequiredProtocols: []string{"HTTP"}}})
	if r.Chosen.Name != "py" {
		t.Fatalf("protocol constraint: want py, got %q", r.Chosen.Name)
	}
	if _, err := s.Select(context.Background(), idx, Request{
		Capability: "review", Constraints: Constraints{RequiredModel: "gpt-9"},
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("impossible model constraint: want ErrNotFound, got %v", err)
	}
}

func TestSelect_RequireGPU(t *testing.T) {
	s := &Scorer{Metrics: &FakeMetrics{Data: map[string]Metrics{"default/gpu": {}}}}
	idx := idxWith(
		Candidate{Key: "default/cpu", Name: "cpu", Ready: true, Capabilities: []Capability{cap("train")}},
		Candidate{Key: "default/gpu", Name: "gpu", Ready: true, Capabilities: []Capability{cap("train", withGPU(1))}},
	)
	r := mustSelect(t, s, idx, Request{Capability: "train", Constraints: Constraints{RequireGPU: true}})
	if r.Chosen.Name != "gpu" {
		t.Fatalf("want gpu agent, got %q", r.Chosen.Name)
	}
}

// ADR-028 strict isolation: an expert target routes ONLY to that agent; no fallback
// to a sibling provider even when a sibling could serve the capability.
func TestSelect_ExpertStrictIsolation(t *testing.T) {
	s := &Scorer{Metrics: &FakeMetrics{Data: map[string]Metrics{
		"default/expert-a": {}, "default/expert-b": {},
	}}}
	idx := idxWith(
		Candidate{Key: "default/expert-a", Name: "expert-a", Ready: true, Capabilities: []Capability{
			cap("review", func(c *Capability) { c.InputSchema = `{"context_slice":{"files":["a.go"]}}` }),
		}},
		Candidate{Key: "default/expert-b", Name: "expert-b", Ready: true, Capabilities: []Capability{cap("review")}},
	)
	r := mustSelect(t, s, idx, Request{Capability: "review", ExpertTarget: "expert-a"})
	if r.Chosen.Name != "expert-a" {
		t.Fatalf("want expert-a, got %q", r.Chosen.Name)
	}
	// The matching capability's input schema is carried back for the broker's context-slice binding.
	if r.CapabilityInputSchema == "" {
		t.Fatal("want input schema carried back for ADR-028 binding, got empty")
	}
	// Unknown expert => no fallback, ErrNotFound.
	if _, err := s.Select(context.Background(), idx, Request{Capability: "review", ExpertTarget: "ghost"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown expert: want ErrNotFound (no fallback), got %v", err)
	}
}

func TestSelect_PrometheusScoringPicksLowestLatency(t *testing.T) {
	s := &Scorer{Metrics: &FakeMetrics{Data: map[string]Metrics{
		"default/slow": {Load: 0.2, LatencyP50Ms: 400},
		"default/fast": {Load: 0.2, LatencyP50Ms: 80},
	}}}
	idx := idxWith(
		Candidate{Key: "default/slow", Name: "slow", Ready: true, Capabilities: []Capability{cap("review", withClass("medium"))}},
		Candidate{Key: "default/fast", Name: "fast", Ready: true, Capabilities: []Capability{cap("review", withClass("medium"))}},
	)
	r := mustSelect(t, s, idx, Request{Capability: "review", Policy: Policy{Objective: "lowest_latency"}})
	if r.Chosen.Name != "fast" {
		t.Fatalf("want fast (lower latency), got %q", r.Chosen.Name)
	}
	if !r.Rationale.PrometheusConsulted {
		t.Fatal("want PrometheusConsulted=true")
	}
}

// Prometheus down: selection must DEGRADE (return a ready agent) not FAIL.
func TestSelect_DegradesWhenMetricsUnavailable(t *testing.T) {
	s := &Scorer{Metrics: &FakeMetrics{Fail: true}}
	idx := idxWith(
		Candidate{Key: "default/a", Name: "a", Ready: true, Capabilities: []Capability{cap("review")}},
		Candidate{Key: "default/b", Name: "b", Ready: true, Capabilities: []Capability{cap("review")}},
	)
	r := mustSelect(t, s, idx, Request{Capability: "review"})
	if r.Rationale.PrometheusConsulted {
		t.Fatal("want PrometheusConsulted=false in degraded mode")
	}
	if !r.Chosen.Ready {
		t.Fatal("degraded fallback must still return a Ready agent")
	}
}

// Index resync proof at the data-structure level: deleting an agent drops its
// capability links, and the index reflects exactly what was Upserted.
func TestIndex_UpsertDeleteMaintainsCapIndex(t *testing.T) {
	idx := NewIndex()
	idx.Upsert(Candidate{Key: "default/a", Capabilities: []Capability{cap("review"), cap("scan")}})
	idx.Upsert(Candidate{Key: "default/b", Capabilities: []Capability{cap("review")}})
	if got := len(idx.Candidates("review")); got != 2 {
		t.Fatalf("review candidates: want 2, got %d", got)
	}
	if got := len(idx.Candidates("scan")); got != 1 {
		t.Fatalf("scan candidates: want 1, got %d", got)
	}
	idx.Delete("default/a")
	if got := len(idx.Candidates("review")); got != 1 {
		t.Fatalf("after delete, review candidates: want 1, got %d", got)
	}
	if got := idx.Candidates("scan"); got != nil {
		t.Fatalf("after delete, scan must be empty, got %v", got)
	}
	if idx.Len() != 1 {
		t.Fatalf("index len: want 1, got %d", idx.Len())
	}
}

// Upsert replacing a candidate must not leave stale capability links.
func TestIndex_UpsertReplaceDropsStaleLinks(t *testing.T) {
	idx := NewIndex()
	idx.Upsert(Candidate{Key: "default/a", Capabilities: []Capability{cap("review")}})
	idx.Upsert(Candidate{Key: "default/a", Capabilities: []Capability{cap("scan")}}) // capability changed
	if got := idx.Candidates("review"); got != nil {
		t.Fatalf("stale 'review' link must be gone, got %v", got)
	}
	if got := len(idx.Candidates("scan")); got != 1 {
		t.Fatalf("new 'scan' link: want 1, got %d", got)
	}
}
