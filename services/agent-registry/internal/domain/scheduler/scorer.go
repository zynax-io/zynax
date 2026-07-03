// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
)

// ErrNoCapability is returned when no agent declares the capability at all
// (pipeline stage 1). Maps to gRPC NOT_FOUND (contract invariant 3).
var ErrNoCapability = errors.New("no agent declares capability")

// FilteredError reports that candidates existed but a pipeline stage
// eliminated all of them. Stage names the first eliminating filter (contract
// invariant 4). Maps to gRPC FAILED_PRECONDITION.
type FilteredError struct {
	Capability string
	Stage      string // "tags" | "language" | "model" | "gpu" | "protocols" | "ready" | "expert:<target>"
}

func (e *FilteredError) Error() string {
	return fmt.Sprintf("no candidate for %q survives filter: %s", e.Capability, e.Stage)
}

// ErrMetricsUnavailable signals the metrics backend is down or slow; selection
// must degrade to readiness-filtered round-robin, never fail (ADR-039 §3).
var ErrMetricsUnavailable = errors.New("metrics source unavailable")

// Metrics is the live, dynamic snapshot for one agent — pulled from the
// metrics backend at selection time, NEVER stored in CRD status (ADR-039 §3).
type Metrics struct {
	Load         float64 // 0..1 utilization
	LatencyP50Ms float64
	QueueDepth   float64
}

// MetricsSource returns a live snapshot for the given agent keys. An error
// means the telemetry backend is unavailable; the scorer degrades gracefully.
type MetricsSource interface {
	Snapshot(ctx context.Context, keys []string) (map[string]Metrics, error)
}

// Constraints are hard filters; a candidate failing any is dropped before
// scoring (ADR-039 §4 stage 2). Empty fields are unconstrained.
type Constraints struct {
	RequiredTags      []string
	RequiredLanguage  string
	RequiredModel     string
	RequireGPU        bool
	RequiredProtocols []string
}

// Mode reports which scoring path produced the decision (proto SelectionMode).
type Mode int

// Scoring modes, mirroring zynax.v1.SelectionMode.
const (
	ModeMetricsWeighted Mode = iota + 1
	ModeRoundRobin
	ModeDegradedRoundRobin
)

// Request is one SelectAgent call in domain terms.
type Request struct {
	Capability   string
	Constraints  Constraints
	RoundRobin   bool   // caller explicitly requested rotation over scoring
	ExpertTarget string // ADR-028: strict scope filter, no fallback
}

// Rationale explains the pick, structurally (maps 1:1 to proto
// SelectionRationale): per-stage counts trace where candidates were
// eliminated without reading scheduler logs.
type Rationale struct {
	CandidatesMatched           int
	CandidatesAfterConstraints  int
	CandidatesReady             int
	CandidatesAfterExpertFilter int
	Mode                        Mode
	WinningFactors              []string
	Summary                     string
}

// Result is the chosen agent plus the matching capability (the broker reads
// its InputSchema for the ADR-028 context-slice binding) and the rationale.
type Result struct {
	Chosen     Candidate
	Capability Capability
	Rationale  Rationale
}

// Scorer runs the ordered, short-circuiting ADR-039 §4 pipeline. Promoted
// from the KIND-verified spike (internal/scorer/scorer.go), with the rotation
// counter added so round-robin modes genuinely rotate across calls.
type Scorer struct {
	Metrics MetricsSource
	// weights of the balanced default policy (lower total score wins).
	rr atomic.Uint64
}

// factorRotation is the stable winning-factor token for rotation modes.
const factorRotation = "rotation"

// Balanced default weights: load and latency dominate, cost tie-breaks.
const (
	weightLoad    = 0.4
	weightLatency = 0.4
	weightCost    = 0.2
)

// Select implements: capability match → hard constraints → readiness →
// expert target → metrics-weighted score (or rotation) → cost tie-break.
// Degrades (never fails) when the metrics source is unavailable.
func (s *Scorer) Select(ctx context.Context, idx *Index, req Request) (Result, error) {
	// Stage 1 — capability match (O(1) index lookup, key-sorted).
	matched := idx.Candidates(req.Capability)
	if len(matched) == 0 {
		return Result{}, fmt.Errorf("%w: %q", ErrNoCapability, req.Capability)
	}
	r := Rationale{CandidatesMatched: len(matched)}

	// Stage 2 — hard constraints, evaluated against the matching capability.
	afterConstraints, stage := filterConstraints(matched, req.Capability, req.Constraints)
	r.CandidatesAfterConstraints = len(afterConstraints)
	if len(afterConstraints) == 0 {
		return Result{}, &FilteredError{Capability: req.Capability, Stage: stage}
	}

	// Stage 3 — readiness: the stale-liveness fix (crashed agent => ready=false).
	ready := filter(afterConstraints, func(c Candidate) bool { return c.Ready })
	r.CandidatesReady = len(ready)
	if len(ready) == 0 {
		return Result{}, &FilteredError{Capability: req.Capability, Stage: "ready"}
	}

	// Stage 4 — expert target: strict isolation, no fallback (ADR-028).
	afterExpert := ready
	if req.ExpertTarget != "" {
		afterExpert = filter(ready, func(c Candidate) bool { return c.ExpertScope == req.ExpertTarget })
		if len(afterExpert) == 0 {
			return Result{}, &FilteredError{Capability: req.Capability, Stage: "expert:" + req.ExpertTarget}
		}
	}
	r.CandidatesAfterExpertFilter = len(afterExpert)

	// Stage 5 — explicit rotation bypasses the metrics stage on request.
	if req.RoundRobin {
		return s.rotate(afterExpert, req.Capability, r, ModeRoundRobin, []string{factorRotation}), nil
	}

	// Stage 5b — metrics snapshot; on error degrade, never fail (ADR-039 §3).
	keys := make([]string, len(afterExpert))
	for i, c := range afterExpert {
		keys[i] = c.Key
	}
	snap, err := s.Metrics.Snapshot(ctx, keys)
	if err != nil {
		return s.rotate(afterExpert, req.Capability, r, ModeDegradedRoundRobin, []string{"readiness", factorRotation}), nil
	}

	// Stage 6 — weighted score (lower is better) + cost/gpu/model tie-break
	// (cost rides the weighted sum; key order makes ties deterministic).
	best := afterExpert[0]
	bestCap, _ := best.CapabilityByID(req.Capability)
	bestScore := score(bestCap, snap[best.Key])
	for _, c := range afterExpert[1:] {
		capability, _ := c.CapabilityByID(req.Capability)
		if sc := score(capability, snap[c.Key]); sc < bestScore {
			best, bestCap, bestScore = c, capability, sc
		}
	}

	m := snap[best.Key]
	r.Mode = ModeMetricsWeighted
	r.WinningFactors = []string{"load", "latency", "cost"}
	r.Summary = fmt.Sprintf("selected %s for %s (load=%.2f latency_p50=%.0fms cost=%s)",
		best.Key, req.Capability, m.Load, m.LatencyP50Ms, bestCap.Cost.LatencyClass)
	return Result{Chosen: best, Capability: bestCap, Rationale: r}, nil
}

// rotate picks the next candidate in rotation order (candidates arrive
// key-sorted from the index, so rotation is stable across replicas).
func (s *Scorer) rotate(cands []Candidate, capID string, r Rationale, mode Mode, factors []string) Result {
	n := s.rr.Add(1) - 1
	chosen := cands[int(n%uint64(len(cands)))] //nolint:gosec // len > 0 guaranteed by caller
	capability, _ := chosen.CapabilityByID(capID)
	r.Mode = mode
	r.WinningFactors = factors
	verb := factorRotation
	if mode == ModeDegradedRoundRobin {
		verb = "degraded: metrics unavailable; readiness-filtered rotation"
	}
	r.Summary = fmt.Sprintf("selected %s for %s (%s)", chosen.Key, capID, verb)
	return Result{Chosen: chosen, Capability: capability, Rationale: r}
}

// filterConstraints drops candidates failing any populated constraint and
// reports the first eliminating filter name (contract invariant 4).
func filterConstraints(in []Candidate, capID string, k Constraints) ([]Candidate, string) {
	out := in[:0:0]
	stage := ""
	for _, c := range in {
		capability, ok := c.CapabilityByID(capID)
		if !ok {
			continue
		}
		if f := failedConstraint(capability, k); f != "" {
			stage = f
			continue
		}
		out = append(out, c)
	}
	return out, stage
}

// failedConstraint names the first unmet constraint, or "" when satisfied.
func failedConstraint(capability Capability, k Constraints) string {
	for _, t := range k.RequiredTags {
		if !containsFold(capability.Selectors.Tags, t) {
			return "tags"
		}
	}
	if k.RequiredLanguage != "" && !containsFold(capability.Selectors.Language, k.RequiredLanguage) {
		return "language"
	}
	if k.RequiredModel != "" && !containsFold(capability.Models, k.RequiredModel) {
		return "model"
	}
	if k.RequireGPU && capability.Resources.GPU <= 0 {
		return "gpu"
	}
	for _, p := range k.RequiredProtocols {
		if !containsFold(capability.Protocols, p) {
			return "protocols"
		}
	}
	return ""
}

// score: weighted sum, lower is better. Latency normalized to seconds; the
// static cost hint rides as the tie-break weight (ADR-039 §4 stage 6).
func score(capability Capability, m Metrics) float64 {
	return weightLoad*m.Load +
		weightLatency*(m.LatencyP50Ms/1000.0) +
		weightCost*costScore(capability.Cost)
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

func containsFold(haystack []string, needle string) bool {
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
