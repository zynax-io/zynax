// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// newProbeServer returns an httptest server that answers the API-root probe with
// the given X-OAuth-Scopes header (omitted when emit is false) and records the
// Authorization header it received so a test can assert the token was sent
// without that token ever appearing in adapter logs.
func newProbeServer(t *testing.T, scopes string, emit bool, gotAuth *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotAuth = r.Header.Get("Authorization")
		if emit {
			w.Header().Set("X-OAuth-Scopes", scopes)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGitHubProbe_ClassicOverBroad(t *testing.T) {
	var auth string
	srv := newProbeServer(t, "repo, gist", true, &auth)

	p, err := NewGitHubProbe("classic-token-value-1234", srv.URL)
	if err != nil {
		t.Fatalf("NewGitHubProbe: %v", err)
	}
	res, err := Validate(context.Background(), p, ModeEnforce)
	if err == nil {
		t.Fatal("expected over-broad classic token to be rejected")
	}
	if !reflect.DeepEqual(res.OverBroad, []string{"repo"}) {
		t.Errorf("OverBroad = %v, want [repo]", res.OverBroad)
	}
	if auth == "" {
		t.Error("probe must send the Authorization header")
	}
}

func TestGitHubProbe_FineGrainedPasses(t *testing.T) {
	var auth string
	srv := newProbeServer(t, "", false, &auth)

	p, err := NewGitHubProbe("github_pat_fine_grained_value", srv.URL)
	if err != nil {
		t.Fatalf("NewGitHubProbe: %v", err)
	}
	res, err := Validate(context.Background(), p, ModeEnforce)
	if err != nil {
		t.Fatalf("fine-grained token must pass: %v", err)
	}
	if res.TokenClass != "fine-grained-or-app" {
		t.Errorf("TokenClass = %q", res.TokenClass)
	}
}

func TestGitHubProbe_NarrowClassicPasses(t *testing.T) {
	var auth string
	srv := newProbeServer(t, "public_repo, read:user", true, &auth)

	p, err := NewGitHubProbe("classic-narrow-token-value", srv.URL)
	if err != nil {
		t.Fatalf("NewGitHubProbe: %v", err)
	}
	if _, err := Validate(context.Background(), p, ModeEnforce); err != nil {
		t.Fatalf("narrow classic token must pass: %v", err)
	}
}

func TestGitHubProbe_TransportError(t *testing.T) {
	// Point at an unroutable base URL so BareDo fails at transport level.
	p, err := NewGitHubProbe("tok-value-abcdef", "http://127.0.0.1:0")
	if err != nil {
		t.Fatalf("NewGitHubProbe: %v", err)
	}
	if _, err := p.Probe(context.Background()); err == nil {
		t.Fatal("expected transport error from unroutable endpoint")
	}
}

func TestNewGitHubProbe_BadBaseURL(t *testing.T) {
	if _, err := NewGitHubProbe("tok-value-abcdef", "://bad url"); err == nil {
		t.Fatal("expected error for malformed base URL")
	}
}
