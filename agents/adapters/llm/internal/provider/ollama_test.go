// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zynax-io/zynax/agents/adapters/llm/internal/config"
)

// newTestOllama returns a provider pointed at srv.
func newTestOllama(srv *httptest.Server) *ollamaProvider {
	p := newOllamaProvider(config.ProviderConfig{Name: "ollama", Model: "llama3", OllamaBaseURL: srv.URL, MaxTokens: 16})
	p.client = srv.Client()
	return p
}

func TestOllamaStreamSuccess(t *testing.T) {
	t.Parallel()
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.WriteString(w, "{\"message\":{\"content\":\"Hel\"},\"done\":false}\n")
		_, _ = io.WriteString(w, "{\"message\":{\"content\":\"lo\"},\"done\":false}\n")
		_, _ = io.WriteString(w, "{\"message\":{\"content\":\"\"},\"done\":true}\n")
	}))
	defer srv.Close()

	p := newTestOllama(srv)
	ch, err := p.Stream(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	texts, streamErr := collect(t, ch)
	if streamErr != nil {
		t.Fatalf("unexpected stream error: %v", streamErr)
	}
	if got := strings.Join(texts, ""); got != wantJoined {
		t.Fatalf("joined chunks = %q, want %q", got, wantJoined)
	}
	if gotPath != ollamaChatPath {
		t.Fatalf("request path = %q, want %q", gotPath, ollamaChatPath)
	}
}

func TestOllamaStreamUpstreamError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "raw-body-should-not-leak")
	}))
	defer srv.Close()

	p := newTestOllama(srv)
	ch, err := p.Stream(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	_, streamErr := collect(t, ch)
	if streamErr == nil {
		t.Fatal("want terminal error on HTTP 502")
	}
	if strings.Contains(streamErr.Error(), "raw-body-should-not-leak") {
		t.Fatalf("error leaks response body: %v", streamErr)
	}
}

func TestOllamaStreamBadFrame(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "{not-json\n")
	}))
	defer srv.Close()

	p := newTestOllama(srv)
	ch, err := p.Stream(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	_, streamErr := collect(t, ch)
	if streamErr == nil {
		t.Fatal("want terminal error on malformed frame")
	}
}

func TestOllamaStreamTimeout(t *testing.T) {
	t.Parallel()
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-block
	}))
	defer srv.Close()
	defer close(block)

	p := newTestOllama(srv)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	ch, err := p.Stream(ctx, "hi")
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	_, streamErr := collect(t, ch)
	if streamErr == nil {
		t.Fatal("want terminal error on deadline")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("timeout path too slow: %v", elapsed)
	}
}
