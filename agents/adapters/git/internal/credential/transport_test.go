// SPDX-License-Identifier: Apache-2.0

package credential

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// erroringSource always fails to resolve a token.
type erroringSource struct{}

func (erroringSource) Token(context.Context) (string, error) {
	return "", errors.New("no token")
}

// rotatingSource returns a different token on each call to prove the transport
// resolves per request rather than caching a value.
type rotatingSource struct{ n int }

func (r *rotatingSource) Token(context.Context) (string, error) {
	r.n++
	return tokenFor(r.n), nil
}

func TestTransport_InjectsBearerPerRequest(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Transport: NewTransport(&rotatingSource{}, nil)}
	for i := 0; i < 2; i++ {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
	if len(seen) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(seen))
	}
	if seen[0] == seen[1] {
		t.Fatalf("expected a fresh token per request, both were %q", seen[0])
	}
	if seen[0] != "Bearer "+tokenFor(1) {
		t.Fatalf("first header = %q, want Bearer + first token", seen[0])
	}
}

func TestTransport_DoesNotMutateCallerHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	client := &http.Client{Transport: NewTransport(NewStaticSource("ghp_static_token_value"), nil)}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	_ = resp.Body.Close()
	if req.Header.Get("Authorization") != "" {
		t.Fatal("transport must not mutate the caller's request header")
	}
}

func TestTransport_TokenResolutionFailureAborts(t *testing.T) {
	client := &http.Client{Transport: NewTransport(erroringSource{}, nil)}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.invalid", nil)
	if _, err := client.Do(req); err == nil {
		t.Fatal("expected the request to fail when token resolution errors")
	}
}

func TestNewTransport_NilBaseDefaultsToDefaultTransport(t *testing.T) {
	tr := NewTransport(NewStaticSource("x"), nil)
	if tr.base == nil {
		t.Fatal("expected a non-nil base RoundTripper")
	}
}
