// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v67/github"
)

// GitHubProbe issues the scope probe through a go-github client. It performs one
// authenticated GET to the API root and returns only the response headers; the
// body is discarded and the token is never surfaced.
type GitHubProbe struct {
	client *github.Client
}

// NewGitHubProbe builds a probe for the given token. baseURL overrides the GitHub
// API endpoint when non-empty (used by tests against an httptest server); the
// production default is the public api.github.com.
func NewGitHubProbe(token, baseURL string) (*GitHubProbe, error) {
	client := github.NewClient(nil).WithAuthToken(token)
	if baseURL != "" {
		parsed, err := client.BaseURL.Parse(baseURL + "/")
		if err != nil {
			return nil, fmt.Errorf("auth: parse base URL: %w", err)
		}
		client.BaseURL = parsed
	}
	return &GitHubProbe{client: client}, nil
}

// Probe satisfies scopeProbe. The root API endpoint is authenticated and echoes
// the token's X-OAuth-Scopes header for classic PATs while costing no rate-limit
// quota beyond the request itself.
func (g *GitHubProbe) Probe(ctx context.Context) (http.Header, error) {
	req, err := g.client.NewRequest(http.MethodGet, "", nil)
	if err != nil {
		return nil, fmt.Errorf("auth: build probe request: %w", err)
	}
	resp, err := g.client.BareDo(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("auth: probe request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.Header, nil
}
