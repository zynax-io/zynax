// SPDX-License-Identifier: Apache-2.0

package scorer

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrNotFound is returned when no ready agent matches the capability + constraints.
// Mirrors task-broker domain.ErrNoEligibleAgent.
var ErrNotFound = errors.New("no eligible agent")

// Constraints are hard filters; a candidate failing any is dropped before scoring.
type Constraints struct {
	RequiredTags      []string
	RequiredLanguage  string
	RequiredModel     string
	RequireGPU        bool
	RequiredProtocols []string
}

// Policy tunes the weighted score. Zero value = balanced default.
type Policy struct {
	WeightLoad    float64
	WeightLatency float64
	WeightCost    float64
	Objective     string // "" | balanced | lowest_latency | lowest_cost
}

func (p Policy) resolved() Policy {
	switch p.Objective {
	case "lowest_latency":
		return Policy{WeightLatency: 1, Objective: p.Objective}
	case "lowest_cost":
		return Policy{WeightCost: 1, Objective: p.Objective}
	}
	if p.WeightLoad == 0 && p.WeightLatency == 0 && p.WeightCost == 0 {
		return Policy{WeightLoad: 0.4, WeightLatency: 0.4, WeightCost: 0.2, Objective: "balanced"}
	}
	return p
}

// Request is one SelectAgent call.
type Request struct {
	Capability   string
	Constraints  Constraints
	Policy       Policy
	ExpertTarget string // ADR-028: when set, MUST resolve to this agent or ErrNotFound
}

// Rationale explains the pick (maps to proto SelectionRationale).
type Rationale struct {
	CandidatesConsidered int
	Score                float64
	Reason               string
	PrometheusConsulted  bool
}

// Result is the chosen agent + the matching capability's input schema (for ADR-028 binding).
type Result struct {
	Chosen                Candidate
	CapabilityInputSchema string
	Rationale             Rationale
}

// Scorer runs the ordered, short-circuiting selection pipeline.
type Scorer struct {
	Metrics MetricsSource
}

// Select implements the ADR-039 pipeline:
// capability match -> hard constraints -> readiness -> expert target ->
// Prometheus-weighted score -> cost/gpu/model tie-break. Degrades (never fails)
// when the metrics source is unavailable.
func (s *Scorer) Select(ctx context.Context, idx *Index, req Request) (Result, error) {
	// 1. Capability match (O(1) index lookup).
	cands := idx.Candidates(req.Capability)
	if len(cands) == 0 {
		return Result{}, fmt.Errorf("%w: capability %q", ErrNotFound, req.Capability)
	}
	considered := len(cands)

	// 2. Hard constraints (evaluated against the matching capability).
	cands = filter(cands, func(c Candidate) bool {
		cap, ok := c.capability(req.Capability)
		return ok && satisfies(cap, req.Constraints)
	})

	// 3. Readiness — the stale-liveness fix: a crashed agent has ready=false.
	cands = filter(cands, func(c Candidate) bool { return c.Ready })

	// 4. Expert target — strict isolation, no fallback (ADR-028).
	if req.ExpertTarget != "" {
		cands = filter(cands, func(c Candidate) bool {
			return c.Name == req.ExpertTarget || c.Key == req.ExpertTarget
		})
	}

	if len(cands) == 0 {
		return Result{}, fmt.Errorf("%w: capability %q (after constraints/readiness/expert)", ErrNotFound, req.Capability)
	}

	// 5. Prometheus snapshot. On error: degrade to readiness-filtered round-robin
	//    (here: deterministic first-by-key, since cands is already key-sorted).
	keys := make([]string, len(cands))
	for i, c := range cands {
		keys[i] = c.Key
	}
	snap, err := s.Metrics.Snapshot(ctx, keys)
	if err != nil {
		chosen := cands[0]
		cap, _ := chosen.capability(req.Capability)
		return Result{
			Chosen:                chosen,
			CapabilityInputSchema: cap.InputSchema,
			Rationale: Rationale{
				CandidatesConsidered: considered,
				PrometheusConsulted:  false,
				Reason:               "degraded: metrics unavailable; readiness-filtered fallback",
			},
		}, nil
	}

	// 6. Weighted score (lower is better) + cost/gpu/model tie-break.
	pol := req.Policy.resolved()
	best := cands[0]
	bestCap, _ := best.capability(req.Capability)
	bestScore := score(bestCap, snap[best.Key], pol)
	for _, c := range cands[1:] {
		cap, _ := c.capability(req.Capability)
		sc := score(cap, snap[c.Key], pol)
		if sc < bestScore {
			best, bestCap, bestScore = c, cap, sc
		}
	}

	m := snap[best.Key]
	return Result{
		Chosen:                best,
		CapabilityInputSchema: bestCap.InputSchema,
		Rationale: Rationale{
			CandidatesConsidered: considered,
			Score:                bestScore,
			PrometheusConsulted:  true,
			Reason: fmt.Sprintf("ready=true load=%.2f latency_p50=%.0fms cost=%s",
				m.Load, m.LatencyP50Ms, bestCap.Cost.LatencyClass),
		},
	}, nil
}

// score: weighted sum, lower is better. Latency normalized to seconds; cost from latencyClass.
func score(cap Capability, m Metrics, p Policy) float64 {
	return p.WeightLoad*m.Load +
		p.WeightLatency*(m.LatencyP50Ms/1000.0) +
		p.WeightCost*costScore(cap.Cost)
}

func costScore(c Cost) float64 {
	switch c.LatencyClass {
	case "low":
		return 0.1
	case "medium":
		return 0.5
	case "high":
		return 1.0
	default:
		return c.TokenPrice // fall back to relative token price when class absent
	}
}

// satisfies reports whether a capability meets every hard constraint.
func satisfies(cap Capability, k Constraints) bool {
	if k.RequiredLanguage != "" && !contains(cap.Selectors.Language, k.RequiredLanguage) {
		return false
	}
	for _, t := range k.RequiredTags {
		if !contains(cap.Selectors.Tags, t) {
			return false
		}
	}
	if k.RequiredModel != "" && !contains(cap.Models, k.RequiredModel) {
		return false
	}
	if k.RequireGPU && cap.Resources.GPU <= 0 {
		return false
	}
	for _, p := range k.RequiredProtocols {
		if !contains(cap.Protocols, p) {
			return false
		}
	}
	return true
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.EqualFold(h, needle) {
			return true
		}
	}
	return false
}

func filter(in []Candidate, keep func(Candidate) bool) []Candidate {
	out := in[:0:0]
	for _, c := range in {
		if keep(c) {
			out = append(out, c)
		}
	}
	return out
}
