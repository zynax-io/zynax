// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"
)

const validConfig = `
agent_id: llm-adapter-test
endpoint: :50070
registry_endpoint: localhost:50052
capabilities:
  - name: chat_completion
provider:
  name: openai
  model: gpt-4o
  api_key_env: OPENAI_API_KEY
`

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestRun_MissingEnv(t *testing.T) {
	t.Setenv(configEnvVar, "")
	if err := run(); err == nil {
		t.Fatal("expected error when config env var unset")
	}
}

func TestRun_BadConfigPath(t *testing.T) {
	t.Setenv(configEnvVar, "/nonexistent/config.yaml")
	if err := run(); err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestRun_UnsetSecret(t *testing.T) {
	t.Setenv(configEnvVar, writeConfig(t, validConfig))
	t.Setenv("OPENAI_API_KEY", "")
	if err := run(); err == nil {
		t.Fatal("expected error when api key env unset")
	}
}

func TestRun_Success(t *testing.T) {
	t.Setenv(configEnvVar, writeConfig(t, validConfig))
	t.Setenv("OPENAI_API_KEY", "sk-test-value")
	if err := run(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
