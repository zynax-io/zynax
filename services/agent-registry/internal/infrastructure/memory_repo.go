// SPDX-License-Identifier: Apache-2.0

// Package infrastructure contains adapters that implement domain ports.
package infrastructure

import (
	"context"
	"fmt"
	"sync"

	"github.com/zynax-io/zynax/services/agent-registry/internal/domain"
)

type memoryRepo struct {
	mu       sync.RWMutex
	agents   map[string]domain.Agent        // primary: agent ID → agent
	capIndex map[string]map[string]struct{} // secondary: capability name → set of registered agent IDs
}

// NewMemoryRepo creates an in-memory AgentRepository backed by sync.RWMutex.
// FindByCapability uses a secondary index for O(1) capability lookups.
func NewMemoryRepo() domain.AgentRepository {
	return &memoryRepo{
		agents:   make(map[string]domain.Agent),
		capIndex: make(map[string]map[string]struct{}),
	}
}

func (r *memoryRepo) Save(_ context.Context, agent domain.Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if old, ok := r.agents[agent.ID]; ok && old.Status == domain.AgentStatusRegistered {
		r.removeFromCapIndex(old)
	}

	r.agents[agent.ID] = agent

	if agent.Status == domain.AgentStatusRegistered {
		r.addToCapIndex(agent)
	}
	return nil
}

func (r *memoryRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	old, ok := r.agents[id]
	if !ok {
		return fmt.Errorf("%w: %q", domain.ErrAgentNotFound, id)
	}
	if old.Status == domain.AgentStatusRegistered {
		r.removeFromCapIndex(old)
	}
	delete(r.agents, id)
	return nil
}

func (r *memoryRepo) FindByID(_ context.Context, id string) (domain.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.agents[id]
	if !ok {
		return domain.Agent{}, fmt.Errorf("%w: %q", domain.ErrAgentNotFound, id)
	}
	return a, nil
}

func (r *memoryRepo) FindAll(_ context.Context) ([]domain.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]domain.Agent, 0, len(r.agents))
	for _, a := range r.agents {
		out = append(out, a)
	}
	return out, nil
}

func (r *memoryRepo) FindByCapability(_ context.Context, name string) ([]domain.Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids, ok := r.capIndex[name]
	if !ok {
		return nil, nil
	}
	out := make([]domain.Agent, 0, len(ids))
	for id := range ids {
		if a, ok := r.agents[id]; ok {
			out = append(out, a)
		}
	}
	return out, nil
}

// addToCapIndex adds the agent's capabilities to the secondary index.
// Caller must hold mu.Lock().
func (r *memoryRepo) addToCapIndex(agent domain.Agent) {
	for _, c := range agent.Capabilities {
		if r.capIndex[c.Name] == nil {
			r.capIndex[c.Name] = make(map[string]struct{})
		}
		r.capIndex[c.Name][agent.ID] = struct{}{}
	}
}

// removeFromCapIndex removes the agent's capabilities from the secondary index.
// Caller must hold mu.Lock().
func (r *memoryRepo) removeFromCapIndex(agent domain.Agent) {
	for _, c := range agent.Capabilities {
		if s, ok := r.capIndex[c.Name]; ok {
			delete(s, agent.ID)
			if len(s) == 0 {
				delete(r.capIndex, c.Name)
			}
		}
	}
}
