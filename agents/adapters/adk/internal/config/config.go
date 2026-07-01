// SPDX-License-Identifier: Apache-2.0

// Package config parses and validates the adk-adapter YAML configuration at
// startup. The path is read from the ADAPTER_CONFIG env var by the bootstrap
// layer. This is the S2 skeleton surface (#1478): identity, endpoint, registry,
// and the capability list. The model backend (provider/name) is added in S3
// (#1479) alongside the ADK Runner bridge; unknown YAML keys are ignored, so a
// richer config file remains parseable here.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DefaultEndpoint is the gRPC bind address used when none is configured.
const DefaultEndpoint = ":50080"

// ProviderOllama is the only model backend wired in S3 (#1479): a zero-secret
// local Ollama endpoint (ADR-038 §3). It is the default when none is declared.
const ProviderOllama = "ollama"

// AdapterConfig is the top-level YAML struct parsed from the file at startup.
type AdapterConfig struct {
	AgentID          string             `yaml:"agent_id"`
	Name             string             `yaml:"name"`
	Description      string             `yaml:"description"`
	Endpoint         string             `yaml:"endpoint"`
	RegistryEndpoint string             `yaml:"registry_endpoint"`
	Model            ModelConfig        `yaml:"model"`
	Capabilities     []CapabilityConfig `yaml:"capabilities"`
}

// ModelConfig selects the LLM backend shared by every ADK agent in this adapter.
// All fields are optional: an omitted block yields the zero-secret Ollama default
// (ADR-038 §3), with host resolved from OLLAMA_HOST at model-construction time.
type ModelConfig struct {
	Provider string `yaml:"provider"` // "ollama" (default); the only value wired in S3
	Name     string `yaml:"name"`     // model tag, e.g. "qwen2.5-coder:0.5b"
	Host     string `yaml:"host"`     // base URL; falls back to OLLAMA_HOST then localhost
}

// CapabilityConfig declares one capability the adapter exposes: its JSON Schemas
// and the ADK agent Instruction that drives its reasoning. Each maps to one ADK
// llmagent wired to a Runner by the S3 bridge (#1479).
type CapabilityConfig struct {
	Name             string `yaml:"name"`
	Description      string `yaml:"description"`
	Instruction      string `yaml:"instruction"`
	TimeoutSeconds   int    `yaml:"timeout_seconds"`
	InputSchemaJSON  string `yaml:"input_schema_json"`
	OutputSchemaJSON string `yaml:"output_schema_json"`
}

// Load reads, parses, and validates the YAML config at path. It returns a
// descriptive error for a missing file, malformed YAML, or any invalid field.
func Load(path string) (*AdapterConfig, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path sourced from ADAPTER_CONFIG env var (operator-controlled)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg AdapterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultEndpoint
	}
	if cfg.Model.Provider == "" {
		cfg.Model.Provider = ProviderOllama
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// validate fails fast on any missing or invalid required field.
func (c *AdapterConfig) validate() error {
	if c.AgentID == "" {
		return fmt.Errorf("config: agent_id is required")
	}
	if c.RegistryEndpoint == "" {
		return fmt.Errorf("config: registry_endpoint is required")
	}
	if c.Model.Provider != ProviderOllama {
		return fmt.Errorf("config: model.provider %q unsupported (only %q is wired)", c.Model.Provider, ProviderOllama)
	}
	if len(c.Capabilities) == 0 {
		return fmt.Errorf("config: at least one capability is required")
	}
	for i, capCfg := range c.Capabilities {
		if capCfg.Name == "" {
			return fmt.Errorf("config: capabilities[%d].name is required", i)
		}
	}
	return nil
}
