// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "adk-adapter.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const validConfig = `
agent_id: adk-adapter-1
name: adk-adapter
description: ADK agents as capabilities
endpoint: "adk-adapter:50080"
registry_endpoint: "agent-registry:50052"
capabilities:
  - name: triage
    description: Classify a support ticket
    timeout_seconds: 60
    input_schema_json: '{"type":"object"}'
    output_schema_json: '{"type":"object"}'
`

func TestLoad_Valid(t *testing.T) {
	cfg, err := Load(writeConfig(t, validConfig))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AgentID != "adk-adapter-1" || cfg.Endpoint != "adk-adapter:50080" {
		t.Errorf("cfg = %+v", cfg)
	}
	if len(cfg.Capabilities) != 1 || cfg.Capabilities[0].Name != "triage" {
		t.Errorf("capabilities = %+v", cfg.Capabilities)
	}
}

func TestLoad_DefaultsEndpoint(t *testing.T) {
	cfg, err := Load(writeConfig(t, "agent_id: a\nregistry_endpoint: r:1\ncapabilities:\n  - {name: c}\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Endpoint != DefaultEndpoint {
		t.Errorf("endpoint default = %q, want %q", cfg.Endpoint, DefaultEndpoint)
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"missing agent_id", "registry_endpoint: r:1\ncapabilities:\n  - {name: c}\n", "agent_id is required"},
		{"missing registry_endpoint", "agent_id: a\ncapabilities:\n  - {name: c}\n", "registry_endpoint is required"},
		{"no capabilities", "agent_id: a\nregistry_endpoint: r:1\n", "at least one capability"},
		{"capability missing name", "agent_id: a\nregistry_endpoint: r:1\ncapabilities:\n  - {description: d}\n", "name is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Load(writeConfig(t, tc.body)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want containing %q", err, tc.want)
			}
		})
	}
}

func TestLoad_FileAndParseErrors(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("expected error for missing file")
	}
	if _, err := Load(writeConfig(t, "{{not valid yaml")); err == nil {
		t.Error("expected error for malformed YAML")
	}
}
