// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
)

// wantJoined is the concatenation of the two-chunk fixtures used across the
// per-provider streaming success tests.
const wantJoined = "Hello"

func TestNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cfg      config.ProviderConfig
		wantType string
		wantErr  bool
	}{
		{"openai", config.ProviderConfig{Name: "openai", Model: "gpt-4o"}, "*provider.openAIProvider", false},
		{"bedrock", config.ProviderConfig{Name: "bedrock", Model: "m", Region: "us-east-1"}, "*provider.bedrockProvider", false},
		{"ollama", config.ProviderConfig{Name: "ollama", Model: "llama3", OllamaBaseURL: "http://x:11434"}, "*provider.ollamaProvider", false},
		{"unknown", config.ProviderConfig{Name: "anthropic", Model: "m"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := New(tt.cfg, config.Secret{})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("New(%q) = nil error, want error", tt.cfg.Name)
				}
				return
			}
			if err != nil {
				t.Fatalf("New(%q) unexpected error: %v", tt.cfg.Name, err)
			}
			if got := typeName(p); got != tt.wantType {
				t.Fatalf("New(%q) type = %s, want %s", tt.cfg.Name, got, tt.wantType)
			}
		})
	}
}

func TestSanitiseErr(t *testing.T) {
	t.Parallel()
	short := "boom"
	if got := sanitiseErr(short); got != short {
		t.Fatalf("sanitiseErr(short) = %q, want unchanged", got)
	}
	long := strings.Repeat("x", maxErrMsgLen+50)
	if got := sanitiseErr(long); len([]rune(got)) != maxErrMsgLen {
		t.Fatalf("sanitiseErr(long) len = %d, want %d", len([]rune(got)), maxErrMsgLen)
	}
}

func TestEffectiveMaxTokens(t *testing.T) {
	t.Parallel()
	if got := effectiveMaxTokens(config.ProviderConfig{MaxTokens: 42}); got != 42 {
		t.Fatalf("effectiveMaxTokens(42) = %d, want 42", got)
	}
	if got := effectiveMaxTokens(config.ProviderConfig{MaxTokens: 0}); got != 1024 {
		t.Fatalf("effectiveMaxTokens(0) = %d, want default 1024", got)
	}
}

func TestSendRespectsCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := make(chan Chunk) // unbuffered, no reader
	if send(ctx, out, Chunk{Text: "x"}) {
		t.Fatal("send on cancelled ctx = true, want false")
	}
}

// typeName returns the concrete type name of a Provider for factory assertions.
func typeName(p Provider) string {
	switch p.(type) {
	case *openAIProvider:
		return "*provider.openAIProvider"
	case *bedrockProvider:
		return "*provider.bedrockProvider"
	case *ollamaProvider:
		return "*provider.ollamaProvider"
	default:
		return "unknown"
	}
}

// collect drains a Chunk channel into texts and a terminal error (if any).
func collect(t *testing.T, ch <-chan Chunk) ([]string, error) {
	t.Helper()
	var texts []string
	var streamErr error
	for c := range ch {
		if c.Err != nil {
			streamErr = c.Err
			continue
		}
		texts = append(texts, c.Text)
	}
	return texts, streamErr
}
