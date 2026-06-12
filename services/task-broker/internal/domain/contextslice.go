// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"encoding/json"
	"fmt"
)

// ContextSlice is the bounded {files[], max_tokens} context an expert
// AgentDef declares for a capability (ADR-028, EPIC #881 O5): the expert
// receives only its declared files, hard-capped at max_tokens.
type ContextSlice struct {
	Files     []string `json:"files"`
	MaxTokens int      `json:"max_tokens"`
}

// expertTarget returns the top-level "expert" field of an input payload when
// present and a non-empty string; anything else is not expert-targeted.
func expertTarget(payload []byte) (string, bool) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return "", false
	}
	raw, ok := fields["expert"]
	if !ok {
		return "", false
	}
	var expert string
	if err := json.Unmarshal(raw, &expert); err != nil || expert == "" {
		return "", false
	}
	return expert, true
}

// selectExpert returns the single agent whose Name (or AgentID) matches the
// requested expert. Strict isolation (ADR-028): an expert-targeted dispatch
// never falls back to a different provider of the same capability.
func selectExpert(agents []AgentInfo, expert string) (AgentInfo, bool) {
	for _, a := range agents {
		if a.Name == expert || a.AgentID == expert {
			return a, true
		}
	}
	return AgentInfo{}, false
}

// declaredSlice extracts the context-slice declaration from a registered
// capability input_schema (JSON Schema draft-07): the defaults of
// properties.context_slice.properties.{files,max_tokens}. The registered
// expert manifest is the single source of truth — slices are never inlined
// into workflow manifests (ADR-028). Returns nil when none is declared.
func declaredSlice(inputSchema []byte) (*ContextSlice, error) {
	if len(inputSchema) == 0 {
		return nil, nil
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(inputSchema, &schema); err != nil {
		return nil, fmt.Errorf("parse input_schema: %w", err)
	}
	raw, ok := schema.Properties["context_slice"]
	if !ok {
		return nil, nil
	}
	var decl struct {
		Properties struct {
			Files struct {
				Default []string `json:"default"`
			} `json:"files"`
			MaxTokens struct {
				Default int `json:"default"`
			} `json:"max_tokens"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &decl); err != nil {
		return nil, fmt.Errorf("parse context_slice declaration: %w", err)
	}
	if decl.Properties.Files.Default == nil {
		return nil, nil
	}
	return &ContextSlice{
		Files:     decl.Properties.Files.Default,
		MaxTokens: decl.Properties.MaxTokens.Default,
	}, nil
}

// bindContextSlice rewrites payload so its context_slice field is exactly the
// declared slice, or absent when the agent declares none. A caller-supplied
// context_slice is always discarded: the registered manifest is the only
// authority, so a caller can never plant another expert's slice into an
// invocation — the strict-isolation enforcement point (ADR-028).
func bindContextSlice(payload []byte, slice *ContextSlice) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return nil, fmt.Errorf("parse input_payload: %w", err)
	}
	delete(fields, "context_slice")
	if slice != nil {
		raw, err := json.Marshal(slice)
		if err != nil {
			return nil, fmt.Errorf("marshal context_slice: %w", err)
		}
		fields["context_slice"] = raw
	}
	bound, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("marshal input_payload: %w", err)
	}
	return bound, nil
}

// prepareExpertDispatch applies the context-slice injection binding (EPIC
// #881 O5): an "expert"-keyed payload narrows the agent set to exactly that
// expert and binds its declared slice into the payload, replacing anything
// the caller supplied. Non-expert dispatches pass through untouched.
func prepareExpertDispatch(payload []byte, agents []AgentInfo) ([]AgentInfo, []byte, error) {
	expert, ok := expertTarget(payload)
	if !ok {
		return agents, payload, nil
	}
	agent, found := selectExpert(agents, expert)
	if !found {
		return nil, nil, fmt.Errorf("%w: no agent %q provides the requested capability", ErrNoEligibleAgent, expert)
	}
	slice, err := declaredSlice(agent.InputSchema)
	if err != nil {
		return nil, nil, fmt.Errorf("task-broker: expert %q: %w", expert, err)
	}
	bound, err := bindContextSlice(payload, slice)
	if err != nil {
		return nil, nil, fmt.Errorf("task-broker: expert %q: %w", expert, err)
	}
	return []AgentInfo{agent}, bound, nil
}
