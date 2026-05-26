// SPDX-License-Identifier: Apache-2.0

// Package config parses and validates the git-adapter YAML configuration at startup.
// Auth token values are never stored — only the env-var name is kept; the token is
// resolved at runtime from the environment by the bootstrap layer.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AdapterConfig is the top-level YAML struct parsed from the file at startup.
// Path is read from the ADAPTER_CONFIG env var by the bootstrap layer.
type AdapterConfig struct {
	AgentID          string                `yaml:"agent_id"`
	Name             string                `yaml:"name"`
	Description      string                `yaml:"description"`
	Endpoint         string                `yaml:"endpoint"`
	RegistryEndpoint string                `yaml:"registry_endpoint"`
	Git              GitConfig             `yaml:"git"`
	Capabilities     []GitCapabilityConfig `yaml:"capabilities"`
}

// GitConfig holds provider-level settings shared across all capabilities.
type GitConfig struct {
	// Provider is the Git hosting provider: "github" or "gitlab" (stub only in M5).
	Provider string `yaml:"provider"`
	// AuthEnv is the name of the environment variable that holds the PAT or app token.
	// The token value is never stored in config — it is resolved from the environment.
	AuthEnv string `yaml:"auth_env"`
}

// GitCapabilityConfig maps one capability name to a static repository target.
// Owner and repo are always declared in config — never derived from input_payload (SSRF prevention).
type GitCapabilityConfig struct {
	Name             string `yaml:"name"`
	Owner            string `yaml:"owner"`
	Repo             string `yaml:"repo"`
	TimeoutSeconds   int    `yaml:"timeout_seconds"`
	InputSchemaJSON  string `yaml:"input_schema_json"`
	OutputSchemaJSON string `yaml:"output_schema_json"`
	Description      string `yaml:"description"`
}

// Load reads, parses, and validates the YAML config at path.
// Returns a descriptive error for missing required fields or malformed YAML.
func Load(path string) (*AdapterConfig, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path sourced from ADAPTER_CONFIG env var (operator-controlled)
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
	if cfg.Git.Provider == "" {
		return fmt.Errorf("config: git.provider is required")
	}
	if cfg.Git.AuthEnv == "" {
		return fmt.Errorf("config: git.auth_env is required")
	}
	if len(cfg.Capabilities) == 0 {
		return fmt.Errorf("config: at least one capability is required")
	}
	for i, c := range cfg.Capabilities {
		if c.Name == "" {
			return fmt.Errorf("config: capabilities[%d].name is required", i)
		}
		if c.Owner == "" {
			return fmt.Errorf("config: capabilities[%d].owner is required", i)
		}
		if c.Repo == "" {
			return fmt.Errorf("config: capabilities[%d].repo is required", i)
		}
	}
	return nil
}

// ResolveToken reads the auth token from the env var named in cfg.Git.AuthEnv.
// Returns an error if the env var is unset or empty.
func ResolveToken(cfg *AdapterConfig) (string, error) {
	token := os.Getenv(cfg.Git.AuthEnv)
	if token == "" {
		return "", fmt.Errorf("config: env var %s is required but not set", cfg.Git.AuthEnv)
	}
	return token, nil
}
