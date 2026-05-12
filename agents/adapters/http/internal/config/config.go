// SPDX-License-Identifier: Apache-2.0

// Package config parses and validates the adapter YAML configuration at startup.
// Credential values are never stored — only env-var name references for auth headers.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AdapterConfig is the top-level YAML struct parsed from the file at startup.
// Path is read from the ADAPTER_CONFIG env var by the bootstrap layer.
type AdapterConfig struct {
	AgentID          string        `yaml:"agent_id"`
	Name             string        `yaml:"name"`
	Description      string        `yaml:"description"`
	Endpoint         string        `yaml:"endpoint"`
	RegistryEndpoint string        `yaml:"registry_endpoint"`
	Capabilities     []RouteConfig `yaml:"capabilities"`
}

// RouteConfig maps one capability name to a static HTTP route.
// URL and headers are always declared in config — never derived from input_payload (SSRF prevention).
type RouteConfig struct {
	Name             string            `yaml:"name"`
	Method           string            `yaml:"method"`
	URL              string            `yaml:"url"`
	Headers          map[string]string `yaml:"headers"`
	TimeoutSeconds   int               `yaml:"timeout_seconds"`
	InputSchemaJSON  string            `yaml:"input_schema_json"`
	OutputSchemaJSON string            `yaml:"output_schema_json"`
	Description      string            `yaml:"description"`
}

// Load reads, parses, and validates the YAML config at path.
// Returns a descriptive error for missing required fields or malformed YAML.
func Load(path string) (*AdapterConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg AdapterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *AdapterConfig) error {
	if cfg.AgentID == "" {
		return fmt.Errorf("config: agent_id is required")
	}
	if cfg.Endpoint == "" {
		return fmt.Errorf("config: endpoint is required")
	}
	if cfg.RegistryEndpoint == "" {
		return fmt.Errorf("config: registry_endpoint is required")
	}
	if len(cfg.Capabilities) == 0 {
		return fmt.Errorf("config: at least one capability is required")
	}
	for i, c := range cfg.Capabilities {
		if c.Name == "" {
			return fmt.Errorf("config: capabilities[%d].name is required", i)
		}
		if c.Method == "" {
			return fmt.Errorf("config: capabilities[%d].method is required", i)
		}
		if c.URL == "" {
			return fmt.Errorf("config: capabilities[%d].url is required", i)
		}
	}
	return nil
}
