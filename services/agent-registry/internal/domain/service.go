// SPDX-License-Identifier: Apache-2.0

// Package domain contains the pure business logic for the agent-registry service.
// It has zero imports from the api or infrastructure layers (ADR-001).
package domain

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const maxCapabilities = 50

// capNameRE matches valid capability names: lowercase letters, digits, underscores; 1–64 chars.
var capNameRE = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)

// AgentRegistryService implements agent registration, deregistration, and discovery.
type AgentRegistryService struct {
	repo AgentRepository
}

// NewAgentRegistryService constructs an AgentRegistryService backed by the given repository.
func NewAgentRegistryService(repo AgentRepository) *AgentRegistryService {
	return &AgentRegistryService{repo: repo}
}

// Register validates and persists a new agent.
// Returns ErrAgentAlreadyExists if a REGISTERED agent with the same ID already exists.
// Returns ErrInvalidArgument if required fields are missing or invalid.
func (s *AgentRegistryService) Register(ctx context.Context, agent Agent) (Agent, error) {
	if err := validateAgent(agent); err != nil {
		return Agent{}, err
	}

	existing, err := s.repo.FindByID(ctx, agent.ID)
	if err == nil && existing.Status == AgentStatusRegistered {
		return Agent{}, fmt.Errorf("%w: %q", ErrAgentAlreadyExists, agent.ID)
	}

	now := time.Now()
	agent.Status = AgentStatusRegistered
	agent.RegisteredAt = now
	agent.UpdatedAt = now

	if err := s.repo.Save(ctx, agent); err != nil {
		return Agent{}, fmt.Errorf("agent-registry: save: %w", err)
	}
	return agent, nil
}

// Deregister marks the agent as DEREGISTERED. The record is retained for audit.
// Returns ErrAgentNotFound if no agent with the given ID is known.
// Returns ErrInvalidArgument if id is empty.
func (s *AgentRegistryService) Deregister(ctx context.Context, id string) (time.Time, error) {
	if id == "" {
		return time.Time{}, fmt.Errorf("%w: agent_id is required", ErrInvalidArgument)
	}

	agent, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s", ErrAgentNotFound, id)
	}

	now := time.Now()
	agent.Status = AgentStatusDeregistered
	agent.UpdatedAt = now

	if err := s.repo.Save(ctx, agent); err != nil {
		return time.Time{}, fmt.Errorf("agent-registry: save: %w", err)
	}
	return now, nil
}

// GetByID returns the agent record regardless of status.
// Returns ErrAgentNotFound if the ID is unknown.
// Returns ErrInvalidArgument if id is empty.
func (s *AgentRegistryService) GetByID(ctx context.Context, id string) (Agent, error) {
	if id == "" {
		return Agent{}, fmt.Errorf("%w: agent_id is required", ErrInvalidArgument)
	}

	agent, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return Agent{}, fmt.Errorf("%w: %s", ErrAgentNotFound, id)
	}
	return agent, nil
}

// FindByCapability returns all REGISTERED agents that declare the named capability.
// Returns an empty slice (not an error) when no agents match.
// Returns ErrInvalidArgument if capability_name is empty.
func (s *AgentRegistryService) FindByCapability(ctx context.Context, capabilityName string) ([]Agent, error) {
	if capabilityName == "" {
		return nil, fmt.Errorf("%w: capability_name is required", ErrInvalidArgument)
	}

	agents, err := s.repo.FindByCapability(ctx, capabilityName)
	if err != nil {
		return nil, fmt.Errorf("agent-registry: find by capability: %w", err)
	}
	return agents, nil
}

// List returns a filtered page of agents. DEREGISTERED agents are excluded unless
// ListFilter.IncludeDeregistered is true. Label filtering uses equality-based matching.
func (s *AgentRegistryService) List(ctx context.Context, filter ListFilter) (ListResult, error) {
	all, err := s.repo.FindAll(ctx)
	if err != nil {
		return ListResult{}, fmt.Errorf("agent-registry: list: %w", err)
	}

	labelReqs := parseLabelSelector(filter.LabelSelector)

	var matched []Agent
	for _, a := range all {
		if !filter.IncludeDeregistered && a.Status == AgentStatusDeregistered {
			continue
		}
		if !matchesLabels(a.Labels, labelReqs) {
			continue
		}
		matched = append(matched, a)
	}

	return ListResult{Agents: matched}, nil
}

// validateAgent checks required fields and capability constraints.
func validateAgent(agent Agent) error {
	if agent.ID == "" {
		return fmt.Errorf("%w: agent_id is required", ErrInvalidArgument)
	}
	if agent.Endpoint == "" {
		return fmt.Errorf("%w: endpoint is required", ErrInvalidArgument)
	}
	if len(agent.Capabilities) == 0 {
		return fmt.Errorf("%w: at least one capability is required", ErrInvalidArgument)
	}
	if len(agent.Capabilities) > maxCapabilities {
		return fmt.Errorf("%w: capability limit is %d", ErrInvalidArgument, maxCapabilities)
	}
	seen := make(map[string]struct{}, len(agent.Capabilities))
	for _, c := range agent.Capabilities {
		if !capNameRE.MatchString(c.Name) {
			return fmt.Errorf("%w: capability name %q must match %s", ErrInvalidArgument, c.Name, capNameRE)
		}
		if _, dup := seen[c.Name]; dup {
			return fmt.Errorf("%w: duplicate capability name %q", ErrInvalidArgument, c.Name)
		}
		seen[c.Name] = struct{}{}
	}
	return nil
}

// parseLabelSelector parses "key=value,key2=value2" into a map. Empty string → empty map.
func parseLabelSelector(selector string) map[string]string {
	if selector == "" {
		return nil
	}
	reqs := make(map[string]string)
	for _, part := range strings.Split(selector, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			reqs[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return reqs
}

// matchesLabels returns true if the agent's labels satisfy all equality requirements.
func matchesLabels(labels map[string]string, reqs map[string]string) bool {
	for k, v := range reqs {
		if labels[k] != v {
			return false
		}
	}
	return true
}
