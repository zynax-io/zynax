// SPDX-License-Identifier: Apache-2.0

// Package scorer holds the ADR-039 spike's pure selection logic: an in-memory
// capability index (mirroring the production memory_repo capIndex) and the
// ordered, short-circuiting scoring pipeline. No I/O, no Kubernetes, no proto —
// this is the code the M8 domain layer will lift behind the SchedulerService.
package scorer

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

// Capability mirrors AgentDef.capabilities[] plus the new scheduler-scoring fields.
type Capability struct {
	ID          string
	InputSchema string // JSON Schema string; carries the context_slice default (ADR-028)
	Selectors   Selectors
	Cost        Cost
	Resources   Resources
	Models      []string
	Protocols   []string
}

// Candidate is the scheduler's view of one Agent CR (identity + readiness + capabilities).
// Key is "namespace/name" and doubles as the agent_id.
type Candidate struct {
	Key          string
	Name         string
	Endpoint     string // resolved "<svc>.<ns>.svc:<port>"
	Ready        bool
	Capabilities []Capability
}

// capability returns the candidate's capability matching id, or false.
func (c Candidate) capability(id string) (Capability, bool) {
	for _, cap := range c.Capabilities {
		if cap.ID == id {
			return cap, true
		}
	}
	return Capability{}, false
}

// Index is the informer-backed cache. Add/Update/Delete are driven by informer
// events in the PoC; here they are plain method calls so the logic is unit-testable.
// Shape mirrors services/agent-registry/internal/infrastructure/memory_repo.go:
// agents map + capIndex (capability -> set of agent keys).
type Index struct {
	mu       sync.RWMutex
	agents   map[string]Candidate
	capIndex map[string]map[string]struct{}
}

// NewIndex returns an empty index.
func NewIndex() *Index {
	return &Index{
		agents:   map[string]Candidate{},
		capIndex: map[string]map[string]struct{}{},
	}
}

// Upsert inserts or replaces a candidate and rebuilds its capability index entries.
// Mirrors memory_repo addToCapIndex/removeFromCapIndex semantics.
func (i *Index) Upsert(c Candidate) {
	i.mu.Lock()
	defer i.mu.Unlock()
	// Remove any stale capability links from a previous version of this candidate.
	i.removeLinks(c.Key)
	i.agents[c.Key] = c
	for _, cap := range c.Capabilities {
		set := i.capIndex[cap.ID]
		if set == nil {
			set = map[string]struct{}{}
			i.capIndex[cap.ID] = set
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
	for _, cap := range old.Capabilities {
		if set := i.capIndex[cap.ID]; set != nil {
			delete(set, key)
			if len(set) == 0 {
				delete(i.capIndex, cap.ID)
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
