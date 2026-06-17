// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
)

// writeAppYAML writes a config file for the App-mode tests (its own helper so it
// does not depend on declaration order with the existing config_test.go helper).
func writeAppYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeAppYAML: %v", err)
	}
	return path
}

const appModeYAML = `
agent_id: git-adapter
endpoint: :50060
registry_endpoint: localhost:50052
git:
  provider: github
  app:
    app_id_env: GH_APP_ID
    installation_id_env: GH_INSTALL_ID
    private_key_env: GH_APP_KEY
capabilities:
  - name: open_pr
    owner: zynax-io
    repo: zynax
`

func TestLoad_AppMode_NoAuthEnvRequired(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(writeAppYAML(t, appModeYAML))
	if err != nil {
		t.Fatalf("App-mode config should load without auth_env: %v", err)
	}
	if !cfg.UsesApp() {
		t.Fatal("UsesApp should be true when git.app is set")
	}
	if cfg.Git.App.AppIDEnv != "GH_APP_ID" {
		t.Errorf("app_id_env mismatch: got %q", cfg.Git.App.AppIDEnv)
	}
}

func TestLoad_AppMode_MissingFieldRejected(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		yaml string
	}{
		{"missing app_id_env", `
agent_id: a
endpoint: :1
registry_endpoint: r:1
git:
  provider: github
  app:
    installation_id_env: I
    private_key_env: K
capabilities:
  - {name: open_pr, owner: o, repo: r}
`},
		{"missing installation_id_env", `
agent_id: a
endpoint: :1
registry_endpoint: r:1
git:
  provider: github
  app:
    app_id_env: A
    private_key_env: K
capabilities:
  - {name: open_pr, owner: o, repo: r}
`},
		{"missing private_key_env", `
agent_id: a
endpoint: :1
registry_endpoint: r:1
git:
  provider: github
  app:
    app_id_env: A
    installation_id_env: I
capabilities:
  - {name: open_pr, owner: o, repo: r}
`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := config.Load(writeAppYAML(t, tt.yaml)); err == nil {
				t.Fatal("expected validation error for incomplete App config")
			}
		})
	}
}

func TestUsesApp_FalseForPATMode(t *testing.T) {
	t.Parallel()
	cfg := &config.AdapterConfig{Git: config.GitConfig{AuthEnv: "X"}}
	if cfg.UsesApp() {
		t.Fatal("UsesApp should be false in PAT mode")
	}
}

func TestResolveAppCredentials_Set(t *testing.T) {
	t.Setenv("GH_APP_ID", "12345")
	t.Setenv("GH_INSTALL_ID", "67890")
	t.Setenv("GH_APP_KEY", "test-private-key-placeholder") // not a real key; ResolveAppCredentials only passes the env value through
	cfg := &config.AdapterConfig{Git: config.GitConfig{App: &config.GitHubAppConfig{
		AppIDEnv: "GH_APP_ID", InstallationIDEnv: "GH_INSTALL_ID", PrivateKeyEnv: "GH_APP_KEY",
	}}}
	got, err := config.ResolveAppCredentials(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AppID != 12345 || got.InstallationID != 67890 {
		t.Fatalf("ids mismatch: %+v", got)
	}
	if len(got.PrivateKeyPEM) == 0 {
		t.Fatal("expected private key bytes to be resolved")
	}
}

func TestResolveAppCredentials_NotConfigured(t *testing.T) {
	t.Parallel()
	cfg := &config.AdapterConfig{Git: config.GitConfig{AuthEnv: "X"}}
	if _, err := config.ResolveAppCredentials(cfg); err == nil {
		t.Fatal("expected error when git.app is not configured")
	}
}

func TestResolveAppCredentials_BadValues(t *testing.T) {
	base := &config.GitHubAppConfig{
		AppIDEnv: "BAD_APP_ID", InstallationIDEnv: "BAD_INSTALL_ID", PrivateKeyEnv: "BAD_KEY",
	}
	tests := []struct {
		name               string
		appID, instID, key string
	}{
		{"app id unset", "", "1", "k"},
		{"app id non-numeric", "abc", "1", "k"},
		{"app id non-positive", "0", "1", "k"},
		{"installation id unset", "1", "", "k"},
		{"installation id non-numeric", "1", "xyz", "k"},
		{"key unset", "1", "1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BAD_APP_ID", tt.appID)
			t.Setenv("BAD_INSTALL_ID", tt.instID)
			t.Setenv("BAD_KEY", tt.key)
			cfg := &config.AdapterConfig{Git: config.GitConfig{App: base}}
			if _, err := config.ResolveAppCredentials(cfg); err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}
