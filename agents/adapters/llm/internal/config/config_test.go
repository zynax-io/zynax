// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
)

func writeYAML(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

const validOpenAI = `
agent_id: llm-adapter-test
name: LLM Adapter Test
description: test
endpoint: :50070
advertise_endpoint: llm-adapter:50070
registry_endpoint: localhost:50052
capabilities:
  - name: chat_completion
    timeout_seconds: 60
provider:
  name: openai
  model: gpt-4o
  api_key_env: OPENAI_API_KEY
  max_tokens: 4096
`

func TestLoad_Valid(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(writeYAML(t, validOpenAI))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AgentID != "llm-adapter-test" {
		t.Errorf("agent_id mismatch: got %q", cfg.AgentID)
	}
	if cfg.Provider.Name != "openai" {
		t.Errorf("provider.name mismatch: got %q", cfg.Provider.Name)
	}
	if cfg.Provider.Model != "gpt-4o" {
		t.Errorf("provider.model mismatch: got %q", cfg.Provider.Model)
	}
	if cfg.Provider.APIKeyEnv != "OPENAI_API_KEY" {
		t.Errorf("provider.api_key_env mismatch: got %q", cfg.Provider.APIKeyEnv)
	}
	if len(cfg.Capabilities) != 1 || cfg.Capabilities[0].Name != "chat_completion" {
		t.Fatalf("capabilities mismatch: %+v", cfg.Capabilities)
	}
}

func TestLoad_ValidOllama(t *testing.T) {
	t.Parallel()
	body := `
agent_id: llm-ollama
endpoint: :50070
advertise_endpoint: llm-adapter:50070
registry_endpoint: localhost:50052
capabilities:
  - name: chat_completion
provider:
  name: ollama
  model: llama3.2
  ollama_base_url: http://localhost:11434
`
	cfg, err := config.Load(writeYAML(t, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Provider.OllamaBaseURL != "http://localhost:11434" {
		t.Errorf("ollama_base_url mismatch: got %q", cfg.Provider.OllamaBaseURL)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	t.Parallel()
	if _, err := config.Load("/nonexistent/path.yaml"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	t.Parallel()
	if _, err := config.Load(writeYAML(t, "agent_id: [unterminated")); err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestLoad_MissingField(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want string
	}{
		{"no agent_id", strings.Replace(validOpenAI, "agent_id: llm-adapter-test", "", 1), "agent_id"},
		{"no endpoint", strings.Replace(validOpenAI, "endpoint: :50070", "", 1), "endpoint"},
		{"no registry", strings.Replace(validOpenAI, "registry_endpoint: localhost:50052", "", 1), "registry_endpoint"},
		{"no capabilities", strings.Split(validOpenAI, "capabilities:")[0] + "provider:\n  name: openai\n  model: gpt-4o\n  api_key_env: OPENAI_API_KEY\n", "capability"},
		{"no model", strings.Replace(validOpenAI, "model: gpt-4o", "", 1), "model"},
		{"no api_key_env openai", strings.Replace(validOpenAI, "api_key_env: OPENAI_API_KEY", "", 1), "api_key_env"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := config.Load(writeYAML(t, tt.body))
			if err == nil {
				t.Fatalf("expected error mentioning %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not mention %q", err.Error(), tt.want)
			}
		})
	}
}

func TestLoad_UnknownProvider(t *testing.T) {
	t.Parallel()
	body := strings.Replace(validOpenAI, "name: openai", "name: anthropic", 1)
	_, err := config.Load(writeYAML(t, body))
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "openai|bedrock|ollama") {
		t.Errorf("error %q should list valid providers", err.Error())
	}
}

func TestLoad_BedrockRequiresRegion(t *testing.T) {
	t.Parallel()
	body := `
agent_id: llm-bedrock
endpoint: :50070
advertise_endpoint: llm-adapter:50070
registry_endpoint: localhost:50052
capabilities:
  - name: chat_completion
provider:
  name: bedrock
  model: anthropic.claude-3-5-sonnet
  api_key_env: AWS_KEY
`
	_, err := config.Load(writeYAML(t, body))
	if err == nil || !strings.Contains(err.Error(), "region") {
		t.Fatalf("expected region-required error, got: %v", err)
	}
}

// TestLoad_HostlessEndpointRequiresAdvertise asserts a hostless bind endpoint
// with no advertise_endpoint is rejected at load time — the regression guard for
// issue #1371, where a verbatim ":50070" made the broker dial localhost.
func TestLoad_HostlessEndpointRequiresAdvertise(t *testing.T) {
	t.Parallel()
	body := strings.Replace(validOpenAI, "advertise_endpoint: llm-adapter:50070\n", "", 1)
	_, err := config.Load(writeYAML(t, body))
	if err == nil {
		t.Fatal("expected error for hostless endpoint without advertise_endpoint")
	}
	if !strings.Contains(err.Error(), "advertise_endpoint") {
		t.Errorf("error %q should mention advertise_endpoint", err.Error())
	}
}

// TestLoad_ExplicitHostEndpointNeedsNoAdvertise asserts the fallback path: when
// the bind endpoint already carries an explicit host, advertise_endpoint may be
// omitted and AdvertisedEndpoint() returns the bind endpoint verbatim.
func TestLoad_ExplicitHostEndpointNeedsNoAdvertise(t *testing.T) {
	t.Parallel()
	body := strings.Replace(validOpenAI,
		"endpoint: :50070\nadvertise_endpoint: llm-adapter:50070\n",
		"endpoint: 0.0.0.0:50070\n", 1)
	cfg, err := config.Load(writeYAML(t, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cfg.AdvertisedEndpoint(); got != "0.0.0.0:50070" {
		t.Errorf("AdvertisedEndpoint() = %q, want 0.0.0.0:50070", got)
	}
}

// TestAdvertisedEndpoint asserts the resolver never returns a hostless address:
// it prefers advertise_endpoint and otherwise falls back to a host-bearing bind
// endpoint. This is the core invariant guarding issue #1371.
func TestAdvertisedEndpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		endpoint  string
		advertise string
		want      string
	}{
		{"explicit advertise wins over hostless bind", ":50070", "llm-adapter:50070", "llm-adapter:50070"},
		{"explicit advertise wins over host bind", "0.0.0.0:50070", "llm-adapter:50070", "llm-adapter:50070"},
		{"falls back to host-bearing bind", "0.0.0.0:50070", "", "0.0.0.0:50070"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &config.AdapterConfig{Endpoint: tt.endpoint, AdvertiseEndpoint: tt.advertise}
			if got := cfg.AdvertisedEndpoint(); got != tt.want {
				t.Errorf("AdvertisedEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveSecret(t *testing.T) {
	cfg, err := config.Load(writeYAML(t, validOpenAI))
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	t.Setenv("OPENAI_API_KEY", "sk-supersecretvalue123")
	secret, err := cfg.ResolveSecret()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if secret.Reveal() != "sk-supersecretvalue123" {
		t.Errorf("reveal mismatch: got %q", secret.Reveal())
	}
}

func TestResolveSecret_Unset(t *testing.T) {
	cfg, err := config.Load(writeYAML(t, validOpenAI))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	t.Setenv("OPENAI_API_KEY", "")
	if _, err := cfg.ResolveSecret(); err == nil {
		t.Fatal("expected error for unset env var")
	}
}

func TestResolveSecret_NoEnvDeclared(t *testing.T) {
	cfg, err := config.Load(writeYAML(t, `
agent_id: llm-ollama
endpoint: :50070
advertise_endpoint: llm-adapter:50070
registry_endpoint: localhost:50052
capabilities:
  - name: chat_completion
provider:
  name: ollama
  model: llama3.2
  ollama_base_url: http://localhost:11434
`))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	secret, err := cfg.ResolveSecret()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !secret.IsZero() {
		t.Error("expected zero secret when no api_key_env declared")
	}
}

func TestSecret_NeverLeaks(t *testing.T) {
	t.Parallel()
	const raw = "sk-this-must-never-appear"
	s := config.NewSecret(raw)

	// Render the Secret through every verb a log line or error might use.
	// A struct wrapper forces the %v/%s family through Secret's Stringer
	// rather than tripping the staticcheck S1025 single-arg shortcut.
	wrap := struct{ S config.Secret }{S: s}
	renderings := []string{
		s.String(),
		s.GoString(),
		fmt.Sprintf("%v", wrap),
		fmt.Sprintf("%s", wrap),
		fmt.Sprintf("%#v", wrap),
		fmt.Sprintf("%+v", wrap),
	}
	for _, r := range renderings {
		if strings.Contains(r, raw) {
			t.Errorf("secret value leaked in rendering: %q", r)
		}
		if !strings.Contains(r, "[REDACTED]") {
			t.Errorf("expected [REDACTED] marker, got %q", r)
		}
	}
	if s.String() != "[REDACTED]" || s.GoString() != "[REDACTED]" {
		t.Errorf("direct redaction mismatch: String=%q GoString=%q", s.String(), s.GoString())
	}

	text, err := s.MarshalText()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(text), raw) {
		t.Errorf("secret leaked via MarshalText: %q", text)
	}
}
