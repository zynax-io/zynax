// SPDX-License-Identifier: Apache-2.0

// Package scheduler holds the pure domain state of the CRD-native scheduler
// (ADR-039): the capability index over the scheduler's view of Agent custom
// resources. No I/O, no Kubernetes, no proto imports — the informer adapter
// (internal/infrastructure/crd) feeds it from watch events, exactly as the
// memory adapter's capIndex was fed from RegisterAgent pushes (ADR-021: only
// the infrastructure adapter changes behind the port).
//
// Promoted from the KIND-verified ADR-039 spike
// (spike/adr-039-crd-scheduler-proof, internal/scorer/index.go).
package scheduler

import (
	"sort"
	"sync"
)

// Selectors are the hard-constraint match candidates declared on a capability.
type Selectors struct {
	Language []string
	Tags     []string
}

// Cost carries the static cost hints used as a scoring tie-break.
type Cost struct {
	TokenPrice   float64 // relative per-1k-token cost; 0 = unknown
	LatencyClass string  // low|medium|high
}

// Resources carries placement/fit hints.
type Resources struct {
	GPU int // GPU count; 0 = CPU-only
}

// Capability mirrors Agent CR spec.capabilities[] plus the scheduler-scoring hints.
type Capability struct {
	ID           string
	Description  string
	InputSchema  string // JSON Schema string; carries the context_slice default (ADR-028)
	OutputSchema string
	Selectors    Selectors
	Cost         Cost
	Resources    Resources
	Models       []string
	Protocols    []string
}

// Candidate is the scheduler's view of one Agent CR (identity + readiness +
// capabilities). Key is "namespace/name" and doubles as the agent_id.
type Candidate struct {
	Key          string
	Name         string
	Endpoint     string // resolved "<svc>.<ns>.svc:<port>"
	Ready        bool
	Replicas     int
	ExpertScope  string
	Capabilities []Capability
}

// CapabilityByID returns the candidate's capability matching id, or false.
func (c Candidate) CapabilityByID(id string) (Capability, bool) {
	for _, capability := range c.Capabilities {
		if capability.ID == id {
			return capability, true
		}
	}
	return Capability{}, false
}

// Index is the informer-backed cache: agents keyed by "namespace/name" plus a
// secondary capability index (capability -> set of agent keys). Shape mirrors
// services/agent-registry/internal/infrastructure/memory_repo.go's capIndex —
// "exactly the shape a Kubernetes informer cache would maintain from watch
// events" (ADR-039). Safe for concurrent use: watch events write, the select
// path reads.
type Index struct {
	mu       sync.RWMutex
	agents   map[string]Candidate
	capIndex map[string]map[string]struct{}
}

// NewIndex returns an empty index. The scheduler is stateless: on restart the
// informer Lists from the API server and replays Upserts — nothing persisted.
func NewIndex() *Index {
	return &Index{
		agents:   map[string]Candidate{},
		capIndex: map[string]map[string]struct{}{},
	}
}

// Upsert inserts or replaces a candidate and rebuilds its capability links.
// Mirrors memory_repo addToCapIndex/removeFromCapIndex semantics.
func (i *Index) Upsert(c Candidate) {
	i.mu.Lock()
	defer i.mu.Unlock()
	// Remove any stale capability links from a previous version of this candidate.
	i.removeLinks(c.Key)
	i.agents[c.Key] = c
	for _, capability := range c.Capabilities {
		set := i.capIndex[capability.ID]
		if set == nil {
			set = map[string]struct{}{}
			i.capIndex[capability.ID] = set
		}
		set[c.Key] = struct{}{}
	}
}

// Delete removes a candidate and its capability links.
func (i *Index) Delete(key string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.removeLinks(key)
	delete(i.agents, key)
}

// removeLinks drops every capIndex entry pointing at key. Caller holds the lock.
func (i *Index) removeLinks(key string) {
	old, ok := i.agents[key]
	if !ok {
		return
	}
	for _, capability := range old.Capabilities {
		if set := i.capIndex[capability.ID]; set != nil {
			delete(set, key)
			if len(set) == 0 {
				delete(i.capIndex, capability.ID)
			}
		}
	}
}

// Candidates returns every agent declaring capID, in deterministic key order.
// O(1) index lookup mirrors FindByCapability in the memory adapter.
func (i *Index) Candidates(capID string) []Candidate {
	i.mu.RLock()
	defer i.mu.RUnlock()
	set := i.capIndex[capID]
	if len(set) == 0 {
		return nil
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Candidate, 0, len(keys))
	for _, k := range keys {
		out = append(out, i.agents[k])
	}
	return out
}

// Len reports how many agents are indexed (used by the resync proof).
func (i *Index) Len() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return len(i.agents)
}

// Snapshot returns every indexed candidate in deterministic key order. Used by
// tests and the resync proof; the hot path uses Candidates.
func (i *Index) Snapshot() []Candidate {
	i.mu.RLock()
	defer i.mu.RUnlock()
	keys := make([]string, 0, len(i.agents))
	for k := range i.agents {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Candidate, 0, len(keys))
	for _, k := range keys {
		out = append(out, i.agents[k])
	}
	return out
}
