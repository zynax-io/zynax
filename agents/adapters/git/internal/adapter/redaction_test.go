// SPDX-License-Identifier: Apache-2.0

package adapter_test

// Redaction integration tests (G.3 / #1199, type: security): prove the injected
// token never reaches a caller-visible CapabilityError message or COMPLETED
// payload — even when an upstream GitHub error body echoes it back. The token
// here is a syntactically PAT-shaped placeholder, never a real credential.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/adapter"
	"github.com/zynax-io/zynax/agents/adapters/git/internal/config"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// fakeToken is long enough to be redacted (≥ minSecretLen) and is not a real secret.
const fakeToken = "ghp_FAKE0000000000000000000000000000fake" //nolint:gosec // test fixture

func newRedactionServer(t *testing.T, baseURL string) *adapter.AgentServer {
	t.Helper()
	cfg := &config.AdapterConfig{
		AgentID:          "git-redact-test",
		Name:             "Git Redact Test",
		Endpoint:         ":50062",
		RegistryEndpoint: "localhost:50052",
		Git:              config.GitConfig{Provider: "github", AuthEnv: "TEST_TOKEN"},
		Capabilities: []config.GitCapabilityConfig{
			{Name: "open_pr", Owner: "test-owner", Repo: "test-repo", TimeoutSeconds: 5},
			{Name: "get_diff", Owner: "test-owner", Repo: "test-repo", TimeoutSeconds: 5},
		},
	}
	return adapter.NewAgentServerWithURL(cfg, fakeToken, baseURL)
}

// TestRedaction_ErrorMessageScrubsToken: an upstream error body that embeds the
// token (e.g. an authenticated remote URL) must be redacted before the message
// reaches the caller.
func TestRedaction_ErrorMessageScrubsToken(t *testing.T) {
	t.Parallel()
	leaky := "remote rejected: https://x-access-token:" + fakeToken + "@github.com/o/r 403"
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": leaky})
		})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	srv := newRedactionServer(t, ts.URL)
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]string{"title": "t", "head": "feat", "base": "main"})
	req := &zynaxv1.ExecuteCapabilityRequest{TaskId: "t-redact", CapabilityName: "open_pr", InputPayload: payload}
	_ = srv.ExecuteCapability(req, stream)

	last := stream.last()
	if last.Error == nil {
		t.Fatalf("expected FAILED with error, got %+v", last)
	}
	if strings.Contains(last.Error.Message, fakeToken) {
		t.Fatalf("token leaked into error message: %q", last.Error.Message)
	}
	if !strings.Contains(last.Error.Message, "[REDACTED]") {
		t.Errorf("expected [REDACTED] placeholder in error message, got %q", last.Error.Message)
	}
}

// TestRedaction_CompletedPayloadScrubsToken: a COMPLETED payload that happens to
// echo the token (a diff body) must be redacted before it leaves the adapter.
func TestRedaction_CompletedPayloadScrubsToken(t *testing.T) {
	t.Parallel()
	leakyDiff := "+ url = https://x-access-token:" + fakeToken + "@github.com/o/r"
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/7",
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/vnd.github.diff")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(leakyDiff))
		})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	srv := newRedactionServer(t, ts.URL)
	stream := &stubStream{ctx: context.Background()}
	payload, _ := json.Marshal(map[string]int{"pr_number": 7})
	req := &zynaxv1.ExecuteCapabilityRequest{TaskId: "t-diff-redact", CapabilityName: "get_diff", InputPayload: payload}
	_ = srv.ExecuteCapability(req, stream)

	last := stream.last()
	if last.EventType != zynaxv1.TaskEventType_TASK_EVENT_TYPE_COMPLETED {
		t.Fatalf("expected COMPLETED, got %v (err=%+v)", last.EventType, last.Error)
	}
	if strings.Contains(string(last.Payload), fakeToken) {
		t.Fatalf("token leaked into COMPLETED payload: %q", last.Payload)
	}
	if !strings.Contains(string(last.Payload), "[REDACTED]") {
		t.Errorf("expected [REDACTED] in diff payload, got %q", last.Payload)
	}
}
