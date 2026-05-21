// SPDX-License-Identifier: Apache-2.0

package domain

import "time"

// AgentStatus mirrors the proto AgentStatus enum; values are stable (ADR-001).
type AgentStatus int32

// AgentStatus values; ordinal values are permanent — never reorder or reassign (ADR-001).
const (
	AgentStatusUnspecified  AgentStatus = 0
	AgentStatusRegistered   AgentStatus = 1
	AgentStatusDeregistered AgentStatus = 2
)

// String returns the proto-style name of the status.
func (s AgentStatus) String() string {
	switch s {
	case AgentStatusRegistered:
		return "REGISTERED"
	case AgentStatusDeregistered:
		return "DEREGISTERED"
	default:
		return "UNSPECIFIED"
	}
}

// Capability describes a single named function an agent can execute.
type Capability struct {
	Name         string
	Description  string
	InputSchema  []byte // optional JSON Schema (draft-07)
	OutputSchema []byte // optional JSON Schema (draft-07)
}

// Agent is the registry's canonical agent record.
type Agent struct {
	ID           string
	Name         string
	Description  string
	Endpoint     string // host:port used by the task broker for capability routing
	Capabilities []Capability
	Labels       map[string]string
	Status       AgentStatus
	RegisteredAt time.Time
	UpdatedAt    time.Time
}

// ListFilter specifies criteria for a List call.
type ListFilter struct {
	LabelSelector       string // comma-separated key=value pairs; empty matches all
	IncludeDeregistered bool
	PageToken           string
	PageSize            int32
}

// ListResult carries one page of matching agents.
type ListResult struct {
	Agents        []Agent
	NextPageToken string
}
