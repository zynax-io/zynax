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

// newTestOpenAI returns a provider pointed at srv with the secret applied.
func newTestOpenAI(srv *httptest.Server, secret config.Secret) *openAIProvider {
	p := newOpenAIProvider(config.ProviderConfig{Name: "openai", Model: "gpt-4o", MaxTokens: 16}, secret)
	p.baseURL = srv.URL
	p.client = srv.Client()
	return p
}

func TestOpenAIStreamSuccess(t *testing.T) {
	t.Parallel()
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n")
		_, _ = io.WriteString(w, "\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n")
		_, _ = io.WriteString(w, "data: [DONE]\n")
	}))
	defer srv.Close()

	p := newTestOpenAI(srv, config.NewSecret("sk-test-key"))
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
	if len(texts) < 1 {
		t.Fatal("want at least one progress chunk before terminal")
	}
	if gotAuth != "Bearer sk-test-key" {
		t.Fatalf("Authorization = %q, want bearer with key", gotAuth)
	}
}

func TestOpenAIStreamUpstreamError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "{\"error\":{\"message\":\"secret-leak-token\"}}")
	}))
	defer srv.Close()

	p := newTestOpenAI(srv, config.NewSecret("sk-test-key"))
	ch, err := p.Stream(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	_, streamErr := collect(t, ch)
	if streamErr == nil {
		t.Fatal("want terminal error on HTTP 500")
	}
	if strings.Contains(streamErr.Error(), "secret-leak-token") {
		t.Fatalf("error leaks response body: %v", streamErr)
	}
	if strings.Contains(streamErr.Error(), "sk-test-key") {
		t.Fatalf("error leaks credential: %v", streamErr)
	}
}

func TestOpenAIStreamTimeout(t *testing.T) {
	t.Parallel()
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-block // hang until the client deadline fires
	}))
	defer srv.Close()
	defer close(block)

	p := newTestOpenAI(srv, config.Secret{})
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

func TestOpenAIStreamBadFrame(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {not-json\n")
	}))
	defer srv.Close()

	p := newTestOpenAI(srv, config.Secret{})
	ch, err := p.Stream(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	_, streamErr := collect(t, ch)
	if streamErr == nil {
		t.Fatal("want terminal error on malformed frame")
	}
}
