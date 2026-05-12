// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/http/internal/config"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return f.Name()
}

const validYAML = `
agent_id: test-agent
name: Test Agent
endpoint: "0.0.0.0:8080"
registry_endpoint: "registry:9090"
capabilities:
  - name: call_api
    method: POST
    url: "https://api.example.com/v1/action"
`

func TestLoad_Valid(t *testing.T) {
	path := writeTemp(t, validYAML)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AgentID != "test-agent" {
		t.Errorf("agent_id = %q, want %q", cfg.AgentID, "test-agent")
	}
	if len(cfg.Capabilities) != 1 {
		t.Fatalf("capabilities len = %d, want 1", len(cfg.Capabilities))
	}
	if cfg.Capabilities[0].Name != "call_api" {
		t.Errorf("capability name = %q, want %q", cfg.Capabilities[0].Name, "call_api")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	path := writeTemp(t, "{{not: valid: yaml")
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

var missingFieldCases = []struct {
	name string
	yaml string
}{
	{
		name: "missing agent_id",
		yaml: `
endpoint: "0.0.0.0:8080"
registry_endpoint: "registry:9090"
capabilities:
  - name: call_api
    method: POST
    url: "https://api.example.com"
`,
	},
	{
		name: "missing endpoint",
		yaml: `
agent_id: test-agent
registry_endpoint: "registry:9090"
capabilities:
  - name: call_api
    method: POST
    url: "https://api.example.com"
`,
	},
	{
		name: "missing registry_endpoint",
		yaml: `
agent_id: test-agent
endpoint: "0.0.0.0:8080"
capabilities:
  - name: call_api
    method: POST
    url: "https://api.example.com"
`,
	},
	{
		name: "no capabilities",
		yaml: `
agent_id: test-agent
endpoint: "0.0.0.0:8080"
registry_endpoint: "registry:9090"
`,
	},
	{
		name: "capability missing name",
		yaml: `
agent_id: test-agent
endpoint: "0.0.0.0:8080"
registry_endpoint: "registry:9090"
capabilities:
  - method: POST
    url: "https://api.example.com"
`,
	},
	{
		name: "capability missing method",
		yaml: `
agent_id: test-agent
endpoint: "0.0.0.0:8080"
registry_endpoint: "registry:9090"
capabilities:
  - name: call_api
    url: "https://api.example.com"
`,
	},
	{
		name: "capability missing url",
		yaml: `
agent_id: test-agent
endpoint: "0.0.0.0:8080"
registry_endpoint: "registry:9090"
capabilities:
  - name: call_api
    method: POST
`,
	},
}

func TestLoad_MissingRequiredFields(t *testing.T) {
	for _, tc := range missingFieldCases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, tc.yaml)
			_, err := config.Load(path)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tc.name)
			}
		})
	}
}
