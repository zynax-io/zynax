// SPDX-License-Identifier: Apache-2.0

package adapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/adapter"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/credential"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// rotatingSource hands out a fresh token on each call, modelling an App source
// that has re-minted between requests.
type rotatingSource struct{ n int }

func (r *rotatingSource) Token(context.Context) (string, error) {
	r.n++
	return tokenN(r.n), nil
}

func tokenN(n int) string { return "ghs_rotated_v" + string(rune('0'+n)) } //nolint:gosec // test helper: n is a small bounded loop index

func sourceTestCfg() *config.AdapterConfig {
	return &config.AdapterConfig{
		AgentID:          "git-test",
		Endpoint:         ":50060",
		RegistryEndpoint: "localhost:50052",
		Git:              config.GitConfig{Provider: "github"},
		Capabilities: []config.GitCapabilityConfig{
			{Name: "open_pr", Owner: "test-owner", Repo: "test-repo", TimeoutSeconds: 5},
		},
	}
}

// TestSourceBackedServer_UsesRefreshedToken proves the source-backed server sends
// the credential.Source's current token as a Bearer header — and that a rotated
// token (App refresh) is picked up on the next request without a rebuild.
func TestSourceBackedServer_UsesRefreshedToken(t *testing.T) {
	var auths []string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		auths = append(auths, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"number": 7, "html_url": "https://github.com/test-owner/test-repo/pull/7",
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	srv := adapter.NewAgentServerWithSourceURL(sourceTestCfg(), &rotatingSource{}, ts.URL)
	payload, _ := json.Marshal(map[string]string{"title": "T", "head": "h", "base": "main"})

	for i := 0; i < 2; i++ {
		stream := &stubStream{ctx: context.Background()}
		req := &zynaxv1.ExecuteCapabilityRequest{TaskId: "t", CapabilityName: "open_pr", InputPayload: payload}
		if err := srv.ExecuteCapability(req, stream); err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
	}
	if len(auths) != 2 {
		t.Fatalf("expected 2 upstream requests, got %d", len(auths))
	}
	if auths[0] == auths[1] {
		t.Fatalf("expected a rotated Bearer token between requests, both were %q", auths[0])
	}
	if auths[0] != "Bearer "+tokenN(1) {
		t.Fatalf("first request Authorization = %q, want Bearer + first token", auths[0])
	}
}

// TestNewAgentServerWithSource_Constructs covers the production constructor path.
func TestNewAgentServerWithSource_Constructs(t *testing.T) {
	srv := adapter.NewAgentServerWithSource(sourceTestCfg(), credential.NewStaticSource("ghp_seed_value"), "ghp_seed_value")
	if srv == nil {
		t.Fatal("expected a non-nil server")
	}
}
