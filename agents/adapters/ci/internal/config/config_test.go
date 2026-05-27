// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/ci/internal/config"
)

// minimalConfig returns a valid YAML config for use in happy-path tests.
func minimalConfig(t *testing.T) string {
	t.Helper()
	return `
agent_id: test-ci-adapter
name: CI Adapter
description: Triggers and monitors CI pipelines
endpoint: "localhost:50060"
registry_endpoint: "localhost:50052"
ci:
  provider: github-actions
  token_env: GITHUB_TOKEN
capabilities:
  - name: trigger_workflow
    owner: my-org
    repo: my-repo
    workflow_id: ci.yml
`
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "ci-adapter-*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer f.Close() //nolint:errcheck
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return f.Name()
}

func TestLoad_HappyPath(t *testing.T) {
	path := writeTemp(t, minimalConfig(t))
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.AgentID != "test-ci-adapter" {
		t.Errorf("AgentID = %q, want %q", cfg.AgentID, "test-ci-adapter")
	}
	if cfg.CI.Provider != "github-actions" {
		t.Errorf("CI.Provider = %q, want %q", cfg.CI.Provider, "github-actions")
	}
	if cfg.CI.TokenEnv != "GITHUB_TOKEN" {
		t.Errorf("CI.TokenEnv = %q, want %q", cfg.CI.TokenEnv, "GITHUB_TOKEN")
	}
}

func TestLoad_Defaults(t *testing.T) {
	path := writeTemp(t, minimalConfig(t))
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.CI.PollIntervalSeconds != 2 {
		t.Errorf("PollIntervalSeconds = %d, want 2", cfg.CI.PollIntervalSeconds)
	}
	if cfg.CI.MaxPollIntervalSeconds != 30 {
		t.Errorf("MaxPollIntervalSeconds = %d, want 30", cfg.CI.MaxPollIntervalSeconds)
	}
	if cfg.CI.TriggerPollTimeoutSeconds != 10 {
		t.Errorf("TriggerPollTimeoutSeconds = %d, want 10", cfg.CI.TriggerPollTimeoutSeconds)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Error("Load() expected error for missing file, got nil")
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	path := writeTemp(t, ":::not valid yaml:::")
	_, err := config.Load(path)
	if err == nil {
		t.Error("Load() expected error for malformed YAML, got nil")
	}
}

func TestLoad_MissingAgentID(t *testing.T) {
	yaml := `
name: CI Adapter
endpoint: "localhost:50060"
registry_endpoint: "localhost:50052"
ci:
  provider: github-actions
  token_env: GITHUB_TOKEN
capabilities:
  - name: trigger_workflow
    owner: my-org
    repo: my-repo
    workflow_id: ci.yml
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Error("Load() expected error for missing agent_id, got nil")
	}
}

func TestLoad_MissingEndpoint(t *testing.T) {
	yaml := `
agent_id: test-ci-adapter
name: CI Adapter
registry_endpoint: "localhost:50052"
ci:
  provider: github-actions
  token_env: GITHUB_TOKEN
capabilities:
  - name: trigger_workflow
    owner: my-org
    repo: my-repo
    workflow_id: ci.yml
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Error("Load() expected error for missing endpoint, got nil")
	}
}

func TestLoad_MissingRegistryEndpoint(t *testing.T) {
	yaml := `
agent_id: test-ci-adapter
name: CI Adapter
endpoint: "localhost:50060"
ci:
  provider: github-actions
  token_env: GITHUB_TOKEN
capabilities:
  - name: trigger_workflow
    owner: my-org
    repo: my-repo
    workflow_id: ci.yml
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Error("Load() expected error for missing registry_endpoint, got nil")
	}
}

func TestLoad_MissingProvider(t *testing.T) {
	yaml := `
agent_id: test-ci-adapter
name: CI Adapter
endpoint: "localhost:50060"
registry_endpoint: "localhost:50052"
ci:
  token_env: GITHUB_TOKEN
capabilities:
  - name: trigger_workflow
    owner: my-org
    repo: my-repo
    workflow_id: ci.yml
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Error("Load() expected error for missing ci.provider, got nil")
	}
}

func TestLoad_MissingTokenEnv(t *testing.T) {
	yaml := `
agent_id: test-ci-adapter
name: CI Adapter
endpoint: "localhost:50060"
registry_endpoint: "localhost:50052"
ci:
  provider: github-actions
capabilities:
  - name: trigger_workflow
    owner: my-org
    repo: my-repo
    workflow_id: ci.yml
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Error("Load() expected error for missing ci.token_env, got nil")
	}
}

func TestLoad_NoCapabilities(t *testing.T) {
	yaml := `
agent_id: test-ci-adapter
name: CI Adapter
endpoint: "localhost:50060"
registry_endpoint: "localhost:50052"
ci:
  provider: github-actions
  token_env: GITHUB_TOKEN
capabilities: []
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Error("Load() expected error for empty capabilities, got nil")
	}
}

func TestLoad_CapabilityMissingOwner(t *testing.T) {
	yaml := `
agent_id: test-ci-adapter
name: CI Adapter
endpoint: "localhost:50060"
registry_endpoint: "localhost:50052"
ci:
  provider: github-actions
  token_env: GITHUB_TOKEN
capabilities:
  - name: trigger_workflow
    repo: my-repo
    workflow_id: ci.yml
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Error("Load() expected error for capability missing owner, got nil")
	}
}

func TestLoad_CapabilityMissingRepo(t *testing.T) {
	yaml := `
agent_id: test-ci-adapter
name: CI Adapter
endpoint: "localhost:50060"
registry_endpoint: "localhost:50052"
ci:
  provider: github-actions
  token_env: GITHUB_TOKEN
capabilities:
  - name: trigger_workflow
    owner: my-org
    workflow_id: ci.yml
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Error("Load() expected error for capability missing repo, got nil")
	}
}

func TestLoad_CapabilityMissingWorkflowID(t *testing.T) {
	yaml := `
agent_id: test-ci-adapter
name: CI Adapter
endpoint: "localhost:50060"
registry_endpoint: "localhost:50052"
ci:
  provider: github-actions
  token_env: GITHUB_TOKEN
capabilities:
  - name: trigger_workflow
    owner: my-org
    repo: my-repo
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Error("Load() expected error for capability missing workflow_id, got nil")
	}
}

func TestResolveToken_HappyPath(t *testing.T) {
	// Use an obviously non-secret test value — this is a unit test, not real credentials.
	wantToken := "test-token-value-for-unit-test" //nolint:gosec // test credential placeholder
	t.Setenv("CI_TEST_TOKEN", wantToken)
	cfg := &config.AdapterConfig{
		CI: config.CIConfig{TokenEnv: "CI_TEST_TOKEN"},
	}
	token, err := config.ResolveToken(cfg)
	if err != nil {
		t.Fatalf("ResolveToken() unexpected error: %v", err)
	}
	if token != wantToken {
		t.Errorf("ResolveToken() = %q, want %q", token, wantToken)
	}
}

func TestResolveToken_MissingEnvVar(t *testing.T) {
	cfg := &config.AdapterConfig{
		CI: config.CIConfig{TokenEnv: "CI_TEST_MISSING_TOKEN_XYZ"},
	}
	_, err := config.ResolveToken(cfg)
	if err == nil {
		t.Error("ResolveToken() expected error for unset env var, got nil")
	}
}
