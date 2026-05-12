// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"testing"
)

func TestRun_MissingEnvVar(t *testing.T) {
	t.Setenv("ADAPTER_CONFIG", "")
	if err := run(); err == nil {
		t.Fatal("expected error when ADAPTER_CONFIG is unset")
	}
}

func TestRun_MissingFile(t *testing.T) {
	t.Setenv("ADAPTER_CONFIG", "/nonexistent/path/adapter.yaml")
	if err := run(); err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestRun_InvalidConfig(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("{{invalid yaml")
	f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	if err := run(); err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestRun_MissingRequiredFields(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "adapter-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// valid YAML but missing agent_id
	_, _ = f.WriteString("endpoint: \"0.0.0.0:8080\"\nregistry_endpoint: \"localhost:9090\"\ncapabilities:\n  - name: x\n    method: POST\n    url: http://x\n")
	f.Close()
	t.Setenv("ADAPTER_CONFIG", f.Name())
	if err := run(); err == nil {
		t.Fatal("expected error for config missing agent_id")
	}
}
