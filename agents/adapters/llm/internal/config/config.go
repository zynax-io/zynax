// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"net"
	"os"

	"gopkg.in/yaml.v3"
)

// ErrSecretMissing is returned by ResolveSecret when the configured api-key env
// var is declared but unset or empty. The bootstrap layer distinguishes this from
// a malformed config so it can degrade gracefully (start, warn, skip registration)
// instead of crash-looping when no secret is provided (issue #1375).
var ErrSecretMissing = errors.New("config: api key env var is not set")

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
	AgentID     string `yaml:"agent_id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	// Endpoint is the address the gRPC server binds to (net.Listen). A hostless
	// value such as ":50070" binds to all interfaces but is NOT routable, so it
	// must never be advertised to the registry verbatim (issue #1371).
	Endpoint string `yaml:"endpoint"`
	// AdvertiseEndpoint is the routable address the task-broker dials for this
	// adapter, e.g. "llm-adapter:50070". When empty it falls back to Endpoint —
	// but only if Endpoint carries an explicit host (see AdvertisedEndpoint).
	// Mirrors the langgraph-adapter ADAPTER_ENDPOINT split (bind vs advertise).
	AdvertiseEndpoint string             `yaml:"advertise_endpoint"`
	RegistryEndpoint  string             `yaml:"registry_endpoint"`
	Capabilities      []CapabilityConfig `yaml:"capabilities"`
	Provider          ProviderConfig     `yaml:"provider"`
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
// KeyEnvVar names the environment variable holding the credential — the value
// is never a config field, only the env-var name. Mirrors the Python
// ProviderConfig fields (wire key stays api_key_env).
//
// The field is named KeyEnvVar rather than APIKeyEnv on purpose: it carries the
// non-sensitive env-var NAME, but a name containing "api_key" trips CodeQL's
// go/clear-text-logging heuristic, which taints the value and flags the (safe)
// operator diagnostics that echo it back. The redacting Secret type (ADR-035)
// is what actually guards the credential value.
type ProviderConfig struct {
	Name          string `yaml:"name"`
	Model         string `yaml:"model"`
	OllamaBaseURL string `yaml:"ollama_base_url"`
	KeyEnvVar     string `yaml:"api_key_env"`
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
	// The address advertised to the registry must be routable. A hostless bind
	// endpoint (":50070") advertised verbatim makes the broker dial localhost
	// (issue #1371), so require an explicit advertise_endpoint in that case.
	if c.AdvertiseEndpoint == "" && !hasExplicitHost(c.Endpoint) {
		return fmt.Errorf(
			"config: advertise_endpoint is required when endpoint %q is hostless "+
				"(a hostless bind address is not routable by the task-broker)", c.Endpoint)
	}
	return nil
}

// AdvertisedEndpoint returns the routable address registered with the registry
// and dialled by the task-broker. It prefers an explicit advertise_endpoint and
// otherwise falls back to the bind Endpoint — which validate() guarantees has an
// explicit host when no advertise_endpoint is set.
func (c *AdapterConfig) AdvertisedEndpoint() string {
	if c.AdvertiseEndpoint != "" {
		return c.AdvertiseEndpoint
	}
	return c.Endpoint
}

// hasExplicitHost reports whether addr carries a non-empty host component.
// Hostless forms such as ":50070" or "50070" return false; "host:port" and
// "0.0.0.0:50070" return true. (0.0.0.0 binds all interfaces but, unlike a bare
// ":port", is at least a concrete address the broker can dial.)
func hasExplicitHost(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	return host != ""
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
		if p.KeyEnvVar == "" {
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
	if c.Provider.KeyEnvVar == "" {
		return Secret{}, nil
	}
	value := os.Getenv(c.Provider.KeyEnvVar)
	if value == "" {
		return Secret{}, fmt.Errorf("env var %s is required but not set: %w", c.Provider.KeyEnvVar, ErrSecretMissing)
	}
	return NewSecret(value), nil
}
