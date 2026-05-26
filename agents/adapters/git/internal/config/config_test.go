// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
)

func TestLoad_Valid(t *testing.T) {
	t.Parallel()
	path := writeYAML(t, `
agent_id: git-adapter-test
name: Git Adapter Test
description: test
endpoint: :50060
registry_endpoint: localhost:50052
git:
  provider: github
  auth_env: GITHUB_TOKEN
capabilities:
  - name: open_pr
    owner: zynax-io
    repo: zynax
    timeout_seconds: 30
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AgentID != "git-adapter-test" {
		t.Errorf("agent_id mismatch: got %q", cfg.AgentID)
	}
	if cfg.Git.Provider != "github" {
		t.Errorf("git.provider mismatch: got %q", cfg.Git.Provider)
	}
	if cfg.Git.AuthEnv != "GITHUB_TOKEN" {
		t.Errorf("git.auth_env mismatch: got %q", cfg.Git.AuthEnv)
	}
	if len(cfg.Capabilities) != 1 {
		t.Fatalf("expected 1 capability, got %d", len(cfg.Capabilities))
	}
	if cfg.Capabilities[0].Owner != "zynax-io" {
		t.Errorf("capabilities[0].owner mismatch: got %q", cfg.Capabilities[0].Owner)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := config.Load("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	t.Parallel()
	path := writeYAML(t, ":::bad yaml:::")
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestValidate_MissingAgentID(t *testing.T) {
	t.Parallel()
	path := writeYAML(t, `
endpoint: :50060
registry_endpoint: localhost:50052
git:
  provider: github
  auth_env: GITHUB_TOKEN
capabilities:
  - name: open_pr
    owner: zynax-io
    repo: zynax
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
}

func TestValidate_MissingGitProvider(t *testing.T) {
	t.Parallel()
	path := writeYAML(t, `
agent_id: git-adapter
endpoint: :50060
registry_endpoint: localhost:50052
git:
  auth_env: GITHUB_TOKEN
capabilities:
  - name: open_pr
    owner: zynax-io
    repo: zynax
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing git.provider")
	}
}

func TestValidate_MissingAuthEnv(t *testing.T) {
	t.Parallel()
	path := writeYAML(t, `
agent_id: git-adapter
endpoint: :50060
registry_endpoint: localhost:50052
git:
  provider: github
capabilities:
  - name: open_pr
    owner: zynax-io
    repo: zynax
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing git.auth_env")
	}
}

func TestValidate_MissingCapabilityOwner(t *testing.T) {
	t.Parallel()
	path := writeYAML(t, `
agent_id: git-adapter
endpoint: :50060
registry_endpoint: localhost:50052
git:
  provider: github
  auth_env: GITHUB_TOKEN
capabilities:
  - name: open_pr
    repo: zynax
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing capabilities[0].owner")
	}
}

func TestResolveToken_Set(t *testing.T) {
	// t.Setenv requires no t.Parallel
	t.Setenv("TEST_TOKEN_123", "mytoken")
	cfg := &config.AdapterConfig{
		Git: config.GitConfig{AuthEnv: "TEST_TOKEN_123"},
	}
	tok, err := config.ResolveToken(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "mytoken" {
		t.Errorf("token mismatch: got %q", tok)
	}
}

func TestResolveToken_Unset(t *testing.T) {
	t.Parallel()
	cfg := &config.AdapterConfig{
		Git: config.GitConfig{AuthEnv: "TEST_TOKEN_UNSET_XYZ"},
	}
	_, err := config.ResolveToken(cfg)
	if err == nil {
		t.Fatal("expected error for unset env var")
	}
}

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writeYAML: %v", err)
	}
	return path
}
