// SPDX-License-Identifier: Apache-2.0

// Package config parses and validates the git-adapter YAML configuration at startup.
// Auth token values are never stored — only the env-var name is kept; the token is
// resolved at runtime from the environment by the bootstrap layer.
package config

import (
	"fmt"
	"os"
	"strconv"

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
//
// Two credential modes are supported (G.7 / #1262):
//   - PAT mode (default): AuthEnv names an env var holding a classic / fine-grained
//     personal-access token. The token does not expire and is read once at startup.
//   - GitHub App mode: App.* fields name the env vars holding the App id,
//     installation id, and private key. The adapter mints a short-lived (~1 h)
//     installation token and refreshes it before expiry, with no process restart.
//
// No secret value is ever stored in config — only the names of the env vars that
// hold them. App mode takes precedence when App is configured.
type GitConfig struct {
	// Provider is the Git hosting provider: "github" or "gitlab" (stub only in M5).
	Provider string `yaml:"provider"`
	// AuthEnv is the name of the environment variable that holds the PAT or app token.
	// The token value is never stored in config — it is resolved from the environment.
	AuthEnv string `yaml:"auth_env"`
	// App configures GitHub App installation-token mode. When set, it takes
	// precedence over AuthEnv and credentials refresh automatically before expiry.
	App *GitHubAppConfig `yaml:"app,omitempty"`
}

// GitHubAppConfig names the environment variables that hold GitHub App identity
// inputs. As with AuthEnv, only env-var names live in config — never the App id,
// installation id, or private key themselves.
type GitHubAppConfig struct {
	// AppIDEnv names the env var holding the numeric GitHub App id.
	AppIDEnv string `yaml:"app_id_env"`
	// InstallationIDEnv names the env var holding the numeric installation id.
	InstallationIDEnv string `yaml:"installation_id_env"`
	// PrivateKeyEnv names the env var holding the PEM-encoded RSA private key.
	PrivateKeyEnv string `yaml:"private_key_env"`
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
	if err := validateAuth(&cfg.Git); err != nil {
		return err
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

// validateAuth checks that exactly one credential mode is fully configured.
// App mode requires all three env-var names; otherwise AuthEnv (PAT mode) is
// required. Only env-var names are validated here — secret values are resolved
// later from the environment, never stored.
func validateAuth(g *GitConfig) error {
	if g.App != nil {
		if g.App.AppIDEnv == "" {
			return fmt.Errorf("config: git.app.app_id_env is required in App mode")
		}
		if g.App.InstallationIDEnv == "" {
			return fmt.Errorf("config: git.app.installation_id_env is required in App mode")
		}
		if g.App.PrivateKeyEnv == "" {
			return fmt.Errorf("config: git.app.private_key_env is required in App mode")
		}
		return nil
	}
	if g.AuthEnv == "" {
		return fmt.Errorf("config: git.auth_env is required (or configure git.app)")
	}
	return nil
}

// UsesApp reports whether the config selects GitHub App installation-token mode.
func (c *AdapterConfig) UsesApp() bool {
	return c.Git.App != nil
}

// ResolveToken reads the auth token from the env var named in cfg.Git.AuthEnv.
// Returns an error if the env var is unset or empty. This is the PAT (non-expiring)
// path; App mode resolves credentials via ResolveAppCredentials instead.
func ResolveToken(cfg *AdapterConfig) (string, error) {
	token := os.Getenv(cfg.Git.AuthEnv)
	if token == "" {
		return "", fmt.Errorf("config: env var %s is required but not set", cfg.Git.AuthEnv)
	}
	return token, nil
}

// AppCredentialInputs are the resolved GitHub App identity values, read from the
// env vars named in GitHubAppConfig. The private key bytes are held only in
// memory and never re-stored in config.
type AppCredentialInputs struct {
	AppID          int64
	InstallationID int64
	PrivateKeyPEM  []byte
}

// ResolveAppCredentials reads the GitHub App identity values from the env vars
// named in cfg.Git.App. It returns a descriptive error (without any secret value)
// for a missing var or a non-numeric id.
func ResolveAppCredentials(cfg *AdapterConfig) (AppCredentialInputs, error) {
	app := cfg.Git.App
	if app == nil {
		return AppCredentialInputs{}, fmt.Errorf("config: git.app is not configured")
	}
	appID, err := envInt64(app.AppIDEnv)
	if err != nil {
		return AppCredentialInputs{}, err
	}
	instID, err := envInt64(app.InstallationIDEnv)
	if err != nil {
		return AppCredentialInputs{}, err
	}
	keyPEM := os.Getenv(app.PrivateKeyEnv)
	if keyPEM == "" {
		return AppCredentialInputs{}, fmt.Errorf("config: env var %s is required but not set", app.PrivateKeyEnv)
	}
	return AppCredentialInputs{
		AppID:          appID,
		InstallationID: instID,
		PrivateKeyPEM:  []byte(keyPEM),
	}, nil
}

// envInt64 reads a positive int64 from the named env var.
func envInt64(name string) (int64, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return 0, fmt.Errorf("config: env var %s is required but not set", name)
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("config: env var %s must be an integer", name)
	}
	if v <= 0 {
		return 0, fmt.Errorf("config: env var %s must be positive", name)
	}
	return v, nil
}
