// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Supported provider names — the closed set per ADR-035.
const (
	providerOpenAI  = "openai"
	providerBedrock = "bedrock"
	providerOllama  = "ollama"
)

// validProviders is the closed set of supported LLM providers (ADR-035).
var validProviders = map[string]struct{}{
	providerOpenAI:  {},
	providerBedrock: {},
	providerOllama:  {},
}

// AdapterConfig is the top-level YAML struct parsed from the file at startup.
// The path is read from the ZYNAX_LLM_CONFIG env var by the bootstrap layer.
// Fields mirror the Python AdapterConfig for behavioural parity.
type AdapterConfig struct {
	AgentID          string             `yaml:"agent_id"`
	Name             string             `yaml:"name"`
	Description      string             `yaml:"description"`
	Endpoint         string             `yaml:"endpoint"`
	RegistryEndpoint string             `yaml:"registry_endpoint"`
	Capabilities     []CapabilityConfig `yaml:"capabilities"`
	Provider         ProviderConfig     `yaml:"provider"`
}

// CapabilityConfig declares one capability the adapter exposes, with its JSON
// Schemas. Schemas are validated against input_payload at request time (P.4).
type CapabilityConfig struct {
	Name             string `yaml:"name"`
	Description      string `yaml:"description"`
	TimeoutSeconds   int    `yaml:"timeout_seconds"`
	InputSchemaJSON  string `yaml:"input_schema_json"`
	OutputSchemaJSON string `yaml:"output_schema_json"`
}

// ProviderConfig holds per-provider settings. Name selects the active provider;
// ApiKeyEnv names the environment variable holding the credential — the value
// is never a config field. Mirrors the Python ProviderConfig fields.
type ProviderConfig struct {
	Name          string `yaml:"name"`
	Model         string `yaml:"model"`
	OllamaBaseURL string `yaml:"ollama_base_url"`
	APIKeyEnv     string `yaml:"api_key_env"`
	MaxTokens     int    `yaml:"max_tokens"`
	Region        string `yaml:"region"`
}

// Load reads, parses, and validates the YAML config at path. It returns a
// descriptive error for a missing file, malformed YAML, or any invalid field.
func Load(path string) (*AdapterConfig, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path sourced from ZYNAX_LLM_CONFIG env var (operator-controlled)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg AdapterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate fails fast on any missing or invalid required field.
func (c *AdapterConfig) validate() error {
	if err := c.validateIdentity(); err != nil {
		return err
	}
	if len(c.Capabilities) == 0 {
		return fmt.Errorf("config: at least one capability is required")
	}
	for i, cap := range c.Capabilities {
		if cap.Name == "" {
			return fmt.Errorf("config: capabilities[%d].name is required", i)
		}
	}
	return c.Provider.validate()
}

// validateIdentity checks the adapter identity and endpoint fields.
func (c *AdapterConfig) validateIdentity() error {
	if c.AgentID == "" {
		return fmt.Errorf("config: agent_id is required")
	}
	if c.Endpoint == "" {
		return fmt.Errorf("config: endpoint is required")
	}
	if c.RegistryEndpoint == "" {
		return fmt.Errorf("config: registry_endpoint is required")
	}
	return nil
}

// validate checks the provider selection and its required fields.
func (p *ProviderConfig) validate() error {
	if _, ok := validProviders[p.Name]; !ok {
		return fmt.Errorf("config: provider.name %q is not one of openai|bedrock|ollama", p.Name)
	}
	if p.Model == "" {
		return fmt.Errorf("config: provider.model is required")
	}
	return p.validateRequiredByProvider()
}

// validateRequiredByProvider enforces the per-provider required fields.
func (p *ProviderConfig) validateRequiredByProvider() error {
	switch p.Name {
	case providerOpenAI, providerBedrock:
		if p.APIKeyEnv == "" {
			return fmt.Errorf("config: provider.api_key_env is required for %s", p.Name)
		}
	case providerOllama:
		if p.OllamaBaseURL == "" {
			return fmt.Errorf("config: provider.ollama_base_url is required for ollama")
		}
	}
	if p.Name == providerBedrock && p.Region == "" {
		return fmt.Errorf("config: provider.region is required for bedrock")
	}
	return nil
}

// ResolveSecret reads the API key from the env var named in provider.api_key_env
// and returns it wrapped in a redacting Secret. Returns an error if the named
// env var is unset or empty. Returns a zero Secret when no key env is declared
// (e.g. the ollama provider, which needs no credential).
func (c *AdapterConfig) ResolveSecret() (Secret, error) {
	if c.Provider.APIKeyEnv == "" {
		return Secret{}, nil
	}
	value := os.Getenv(c.Provider.APIKeyEnv)
	if value == "" {
		return Secret{}, fmt.Errorf("config: env var %s is required but not set", c.Provider.APIKeyEnv)
	}
	return NewSecret(value), nil
}
