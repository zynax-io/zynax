// SPDX-License-Identifier: Apache-2.0

// Package auth validates, at startup, that the git-adapter's GitHub token grants
// no more access than the configured owner/repo capability set requires
// (least-privilege; EPIC G, story G.5 / #1260, ADR-032).
//
// The check is a single authenticated probe to the GitHub API root. GitHub
// echoes the granted scopes of a classic personal-access token in the
// `X-OAuth-Scopes` response header. A token carrying the broad `repo`
// (or other account-wide admin/delete) scope can reach every repository the
// account can, regardless of the adapter's static owner/repo pinning — so it is
// rejected (or warned on, when configured) before the adapter ever uses it.
//
// Fine-grained PATs and GitHub App installation tokens carry no
// `X-OAuth-Scopes` header (their access is bounded server-side to selected
// repositories); they are treated as already least-privilege and pass.
//
// The token value is never logged, returned, or embedded in any error. Only
// scope/visibility metadata is surfaced.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Token-class labels surfaced in the (non-secret) Result.
const (
	tokenClassClassic     = "classic"
	tokenClassFineGrained = "fine-grained-or-app" //nolint:gosec // G101: a token-class label, not a credential
)

// scopesHeaderKey is the canonical (textproto-canonicalised) form of the
// GitHub X-OAuth-Scopes response header. http.Header keys are stored canonical,
// so the lookup must use this exact casing.
const scopesHeaderKey = "X-Oauth-Scopes"

// Mode selects what happens when a token's scope exceeds the configured set.
type Mode int

const (
	// ModeEnforce fails startup (fail-fast) when the token is over-broad. Default.
	ModeEnforce Mode = iota
	// ModeWarn logs a loud structured warning but allows startup to continue.
	ModeWarn
)

// ParseMode maps an operator-supplied string (e.g. the GIT_ADAPTER_SCOPE_MODE
// env var) to a Mode. Empty or unrecognised values fall back to ModeEnforce so
// the secure default holds when the setting is absent or mistyped.
func ParseMode(s string) Mode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "warn":
		return ModeWarn
	default:
		return ModeEnforce
	}
}

// String renders the Mode for structured log fields.
func (m Mode) String() string {
	if m == ModeWarn {
		return "warn"
	}
	return "enforce"
}

// overBroadScopes are classic-PAT OAuth scopes that grant access beyond a single
// configured repository: account-wide repository write/admin, repo deletion, and
// organisation/user administration. Any of these on the token means it is not
// least-privilege for an adapter pinned to one owner/repo.
var overBroadScopes = map[string]struct{}{
	"repo":             {}, // full control of all private repositories
	"write:org":        {},
	"admin:org":        {},
	"delete_repo":      {},
	"admin:repo_hook":  {},
	"admin:org_hook":   {},
	"admin:public_key": {},
	"admin:gpg_key":    {},
	"user":             {}, // full account profile write
	"delete:packages":  {},
	"site_admin":       {},
}

// Result is the non-secret outcome of a scope probe — safe to log in full.
type Result struct {
	// TokenClass is "classic" when the probe saw an X-OAuth-Scopes header,
	// "fine-grained-or-app" otherwise.
	TokenClass string
	// Scopes are the granted classic-PAT scopes (empty for fine-grained/App).
	Scopes []string
	// OverBroad lists the granted scopes that exceed a single-repo least-privilege
	// posture. Non-empty means the token is over-privileged.
	OverBroad []string
}

// scopeProbe is the minimal surface the validator needs from a GitHub client —
// a single GET to the API root whose response headers are inspected. Satisfied
// by the production *github.Client wrapper (see GitHubProbe) and by test fakes.
type scopeProbe interface {
	// Probe issues an authenticated GET to the API root and returns the response
	// headers (notably X-OAuth-Scopes). It must never return the token.
	Probe(ctx context.Context) (http.Header, error)
}

// ErrOverBroadScope is returned by Validate in ModeEnforce when the token grants
// access beyond the configured owner/repo set. It carries only scope metadata,
// never the token value.
var ErrOverBroadScope = errors.New("git token scope exceeds configured owner/repo (least-privilege)")

// Inspect probes the token and classifies its scope without applying a policy.
// It never logs or returns the token value.
func Inspect(ctx context.Context, p scopeProbe) (Result, error) {
	hdr, err := p.Probe(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("auth: scope probe failed: %w", err)
	}

	raw, hasHeader := scopeHeader(hdr)
	if !hasHeader {
		// Fine-grained PAT or App installation token: access is bounded
		// server-side to selected repositories; least-privilege by construction.
		return Result{TokenClass: tokenClassFineGrained}, nil
	}

	scopes := parseScopes(raw)
	over := make([]string, 0)
	for _, s := range scopes {
		if _, bad := overBroadScopes[s]; bad {
			over = append(over, s)
		}
	}
	sort.Strings(over)
	return Result{TokenClass: tokenClassClassic, Scopes: scopes, OverBroad: over}, nil
}

// Validate inspects the token and applies the policy: in ModeEnforce an
// over-broad token yields ErrOverBroadScope; in ModeWarn it returns the Result
// with a non-nil error replaced by nil so the caller can log a warning and
// continue. The returned Result is always safe to log.
func Validate(ctx context.Context, p scopeProbe, mode Mode) (Result, error) {
	res, err := Inspect(ctx, p)
	if err != nil {
		return Result{}, err
	}
	if len(res.OverBroad) > 0 && mode == ModeEnforce {
		return res, fmt.Errorf("%w: granted [%s]", ErrOverBroadScope, strings.Join(res.OverBroad, ", "))
	}
	return res, nil
}

// scopeHeader returns the X-OAuth-Scopes header value and whether it was present.
// A present-but-empty header (the token grants no classic scopes) still counts as
// present — it tells us this is a classic token with an empty scope set.
func scopeHeader(h http.Header) (string, bool) {
	if h == nil {
		return "", false
	}
	vals, ok := h[scopesHeaderKey]
	if !ok {
		return "", false
	}
	if len(vals) == 0 {
		return "", true
	}
	return vals[0], true
}

// parseScopes splits the comma-separated X-OAuth-Scopes value into a normalised,
// de-duplicated, sorted slice. Empty entries are dropped.
func parseScopes(raw string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
