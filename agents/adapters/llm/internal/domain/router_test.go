// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
)

func testConfig() *config.AdapterConfig {
	return &config.AdapterConfig{
		Capabilities: []config.CapabilityConfig{{
			Name:             "chat_completion",
			Description:      "Stream a chat completion.",
			InputSchemaJSON:  chatSchema,
			OutputSchemaJSON: `{"type":"object","properties":{"completion":{"type":"string"}}}`,
		}},
	}
}

func TestNewRouterDispatchAndSchema(t *testing.T) {
	r, err := NewRouter(testConfig(), &fakeProvider{})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	if _, ok := r.Dispatch("chat_completion"); !ok {
		t.Error("Dispatch(chat_completion): want hit")
	}
	if _, ok := r.Dispatch("nonexistent"); ok {
		t.Error("Dispatch(nonexistent): want miss")
	}

	desc, in, out, ok := r.Schema("chat_completion")
	if !ok {
		t.Fatal("Schema(chat_completion): want hit")
	}
	if desc == "" || in == "" || out == "" {
		t.Errorf("schema fields must be non-empty: desc=%q in=%q out=%q", desc, in, out)
	}
	if _, _, _, ok := r.Schema("nonexistent"); ok {
		t.Error("Schema(nonexistent): want miss")
	}
}

func TestNewRouterBadSchemaFails(t *testing.T) {
	cfg := testConfig()
	cfg.Capabilities[0].InputSchemaJSON = `{not a schema`
	if _, err := NewRouter(cfg, &fakeProvider{}); err == nil {
		t.Fatal("NewRouter: want error for malformed capability schema")
	}
}
