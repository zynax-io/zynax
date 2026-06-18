// SPDX-License-Identifier: Apache-2.0

// Package config parses and validates the ci-adapter YAML configuration at startup.
// Auth token values are never stored — only the env-var name is kept; the token is
// resolved at runtime from the environment by the bootstrap layer.
package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ErrTokenMissing is returned by ResolveToken when the configured auth-token env
// var is unset or empty. The bootstrap layer distinguishes this from a malformed
// config so it can degrade gracefully (start, warn, skip registration) instead of
// crash-looping when no secret is provided (issue #1375).
var ErrTokenMissing = errors.New("config: auth token env var is not set")

// AdapterConfig is the top-level YAML struct parsed from the file at startup.
// Path is read from the ADAPTER_CONFIG env var by the bootstrap layer.
type AdapterConfig struct {
	AgentID          string               `yaml:"agent_id"`
	Name             string               `yaml:"name"`
	Description      string               `yaml:"description"`
	Endpoint         string               `yaml:"endpoint"`
	RegistryEndpoint string               `yaml:"registry_endpoint"`
	CI               CIConfig             `yaml:"ci"`
	Capabilities     []CICapabilityConfig `yaml:"capabilities"`
}

// CIConfig holds provider-level settings shared across all capabilities.
type CIConfig struct {
	// Provider is the CI provider: "github-actions" or "jenkins-stub" (stub only in M5).
	Provider string `yaml:"provider"`
	// TokenEnv is the name of the environment variable that holds the API token.
	// The token value is never stored in config — it is resolved from the environment.
	TokenEnv string `yaml:"token_env"`
	// PollIntervalSeconds is the initial polling interval for run-status polling.
	// Default: 2 seconds.
	PollIntervalSeconds int `yaml:"poll_interval_seconds"`
	// MaxPollIntervalSeconds is the backoff ceiling for the exponential poll loop.
	// Default: 30 seconds.
	MaxPollIntervalSeconds int `yaml:"max_poll_interval_seconds"`
	// TriggerPollTimeoutSeconds is the maximum time to wait for a run ID to appear
	// after a workflow_dispatch event is sent. Default: 10 seconds.
	TriggerPollTimeoutSeconds int `yaml:"trigger_poll_timeout_seconds"`
}

// CICapabilityConfig maps one capability name to a static repository target.
// Owner and repo are always declared in config — never derived from input_payload
// (SSRF prevention: no attacker-controlled URL construction).
type CICapabilityConfig struct {
	// Name is the capability identifier in snake_case (1–64 chars).
	Name string `yaml:"name"`
	// Description is surfaced via GetCapabilitySchema.
	Description string `yaml:"description"`
	// Owner is the GitHub org or user. Static — never from input_payload.
	Owner string `yaml:"owner"`
	// Repo is the repository name. Static — never from input_payload.
	Repo string `yaml:"repo"`
	// WorkflowID is the GitHub Actions workflow file name or numeric ID.
	WorkflowID string `yaml:"workflow_id"`
	// TimeoutSeconds caps the overall capability execution duration.
	TimeoutSeconds   int    `yaml:"timeout_seconds"`
	InputSchemaJSON  string `yaml:"input_schema_json"`
	OutputSchemaJSON string `yaml:"output_schema_json"`
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

	applyDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyDefaults sets zero-value fields to their documented defaults.
func applyDefaults(cfg *AdapterConfig) {
	if cfg.CI.PollIntervalSeconds == 0 {
		cfg.CI.PollIntervalSeconds = 2
	}
	if cfg.CI.MaxPollIntervalSeconds == 0 {
		cfg.CI.MaxPollIntervalSeconds = 30
	}
	if cfg.CI.TriggerPollTimeoutSeconds == 0 {
		cfg.CI.TriggerPollTimeoutSeconds = 10
	}
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
	if cfg.CI.Provider == "" {
		return fmt.Errorf("config: ci.provider is required")
	}
	if cfg.CI.TokenEnv == "" {
		return fmt.Errorf("config: ci.token_env is required")
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
		if c.WorkflowID == "" {
			return fmt.Errorf("config: capabilities[%d].workflow_id is required", i)
		}
	}
	return nil
}

// ResolveToken reads the auth token from the env var named in cfg.CI.TokenEnv.
// Returns an error if the env var is unset or empty.
func ResolveToken(cfg *AdapterConfig) (string, error) {
	token := os.Getenv(cfg.CI.TokenEnv)
	if token == "" {
		return "", fmt.Errorf("env var %s is required but not set: %w", cfg.CI.TokenEnv, ErrTokenMissing)
	}
	return token, nil
}
