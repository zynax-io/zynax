// SPDX-License-Identifier: Apache-2.0

// Package domain implements the capability routing and request handling for the
// llm-adapter: a CapabilityRouter maps capability names to a ChatCompletionHandler
// and the declared JSON Schemas, and the handler validates input, streams provider
// tokens as PROGRESS events, and emits exactly one terminal event (canvas M7.P.4).
package domain

import (
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/llm/internal/provider"
)

// capabilitySchema holds the declared JSON Schemas and description for a single
// capability, returned verbatim by GetCapabilitySchema.
type capabilitySchema struct {
	description  string
	inputSchema  string
	outputSchema string
}

// CapabilityRouter maps capability names to a handler and the declared JSON
// Schemas. It is built once from AdapterConfig at startup and is immutable after
// construction — all fields are read-only after New (canvas: stateless adapter).
type CapabilityRouter struct {
	handlers map[string]*ChatCompletionHandler
	schemas  map[string]capabilitySchema
}

// NewRouter builds a CapabilityRouter from a validated AdapterConfig and a single
// shared Provider. Each declared capability is bound to a ChatCompletionHandler
// that validates against the capability's input schema before invoking p.
func NewRouter(cfg *config.AdapterConfig, p provider.Provider) (*CapabilityRouter, error) {
	handlers := make(map[string]*ChatCompletionHandler, len(cfg.Capabilities))
	schemas := make(map[string]capabilitySchema, len(cfg.Capabilities))
	for _, c := range cfg.Capabilities {
		h, err := newChatCompletionHandler(p, c.InputSchemaJSON)
		if err != nil {
			return nil, err
		}
		handlers[c.Name] = h
		schemas[c.Name] = capabilitySchema{
			description:  c.Description,
			inputSchema:  c.InputSchemaJSON,
			outputSchema: c.OutputSchemaJSON,
		}
	}
	return &CapabilityRouter{handlers: handlers, schemas: schemas}, nil
}

// Dispatch returns the handler registered for capability name and whether it
// exists. The boolean lets the caller map an unknown capability to NOT_FOUND.
func (r *CapabilityRouter) Dispatch(name string) (*ChatCompletionHandler, bool) {
	h, ok := r.handlers[name]
	return h, ok
}

// Schema returns the declared description and JSON Schemas for capability name
// and whether it exists.
func (r *CapabilityRouter) Schema(name string) (description, inputSchema, outputSchema string, ok bool) {
	s, found := r.schemas[name]
	if !found {
		return "", "", "", false
	}
	return s.description, s.inputSchema, s.outputSchema, true
}
