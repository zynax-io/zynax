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

// AdapterConfig is the top-level YAML struct parsed from the file at startup.
type AdapterConfig struct {
	AgentID          string             `yaml:"agent_id"`
	Name             string             `yaml:"name"`
	Description      string             `yaml:"description"`
	Endpoint         string             `yaml:"endpoint"`
	RegistryEndpoint string             `yaml:"registry_endpoint"`
	Capabilities     []CapabilityConfig `yaml:"capabilities"`
}

// CapabilityConfig declares one capability the adapter exposes, with its JSON
// Schemas. Each maps to one ADK llmagent once the bridge lands in S3 (#1479).
type CapabilityConfig struct {
	Name             string `yaml:"name"`
	Description      string `yaml:"description"`
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
