// SPDX-License-Identifier: Apache-2.0

// Package credential provides a refreshable token abstraction for the
// git-adapter (EPIC G, story G.7 / #1262, ADR-032).
//
// Before G.7 the adapter read its token once at startup and the go-github client
// held it for the whole process lifetime (see internal/config.ResolveToken). That
// is correct for a classic or fine-grained personal-access token, which does not
// expire — but a GitHub App installation token lives only ~1 h, so a long-running
// adapter would start returning 401s mid-process and need a restart.
//
// A Source returns a currently-valid token on demand and transparently refreshes
// it before expiry. Two implementations are provided:
//
//   - StaticSource wraps a never-expiring PAT — Token always returns the same
//     value, so the classic / fine-grained path is unchanged (G.7 AC: PAT path
//     unchanged).
//   - AppSource mints and refreshes GitHub App installation tokens from an
//     app-id + private-key + installation-id, minting a fresh token whenever the
//     held one is within refreshSkew of expiry.
//
// The token value and the App private key are never logged, returned in an
// error, or otherwise serialized — Source surfaces only the token to its single
// consumer (the authenticated HTTP transport) and metadata to logs. Pair a Source
// with redact.Redactor at every caller-visible egress (handler.sanitise, MCP tool
// results) so a token that leaks into an upstream error string is still scrubbed.
package credential

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// refreshSkew is how long before a minted token's expiry AppSource refreshes it.
// A generous margin (relative to the ~1 h installation-token TTL) absorbs clock
// skew and in-flight request latency so a token never expires mid-request.
const refreshSkew = 5 * time.Minute

// Source yields a currently-valid Git credential. Implementations must be safe
// for concurrent use: the adapter calls Token once per outbound API request from
// multiple goroutines. Token must never return the App private key or any secret
// material other than the token the transport needs.
type Source interface {
	// Token returns a valid token, refreshing it first if the held one is near
	// expiry. It returns an error only when a refresh is required and fails; a
	// non-expiring source never errors.
	Token(ctx context.Context) (string, error)
}

// StaticSource is a Source for a non-expiring credential (classic or fine-grained
// PAT). Token always returns the injected value and never errors, preserving the
// pre-G.7 read-once behaviour for the PAT path.
type StaticSource struct {
	token string
}

// NewStaticSource wraps a never-expiring token.
func NewStaticSource(token string) *StaticSource {
	return &StaticSource{token: token}
}

// Token returns the static token. ctx is accepted for interface symmetry and is
// unused — there is nothing to refresh.
func (s *StaticSource) Token(_ context.Context) (string, error) {
	return s.token, nil
}

// installationToken is one minted GitHub App installation token and its expiry.
// It carries no secret beyond the token value itself.
type installationToken struct {
	token   string
	expires time.Time
}

// Minter mints a fresh GitHub App installation token. The production
// implementation signs an App JWT with the private key and calls the GitHub
// installation-token endpoint; tests substitute a fake that returns a token with
// a controllable expiry and no network access. A Minter must never log or return
// the private key — only the minted token and its expiry.
type Minter interface {
	Mint(ctx context.Context) (token string, expires time.Time, err error)
}

// AppSource is a Source backed by GitHub App installation tokens. It holds the
// most recently minted token and re-mints through its Minter once the token is
// within refreshSkew of expiry. It is safe for concurrent use.
type AppSource struct {
	minter Minter
	now    func() time.Time // injectable clock; defaults to time.Now in tests via NewAppSource

	mu  sync.Mutex
	cur installationToken
}

// NewAppSource builds an AppSource over the given Minter. now is the clock used
// for expiry decisions; pass nil for time.Now (tests inject a fake clock to drive
// refresh deterministically without sleeping).
func NewAppSource(minter Minter, now func() time.Time) *AppSource {
	if now == nil {
		now = time.Now
	}
	return &AppSource{minter: minter, now: now}
}

// Token returns a valid installation token, minting a fresh one when none is held
// yet or the current one is within refreshSkew of expiry. On a refresh failure the
// previously held (still-unexpired) token is returned if usable, so a transient
// minting error does not break a request that an existing token can still serve.
func (s *AppSource) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.valid() {
		return s.cur.token, nil
	}

	tok, exp, err := s.minter.Mint(ctx)
	if err != nil {
		// Fall back to the held token if it has not yet hard-expired — a refresh
		// hiccup should not fail a request a valid token can still serve.
		if s.cur.token != "" && s.now().Before(s.cur.expires) {
			return s.cur.token, nil
		}
		return "", fmt.Errorf("mint installation token: %w", err)
	}
	s.cur = installationToken{token: tok, expires: exp}
	return s.cur.token, nil
}

// valid reports whether the held token exists and is not yet within refreshSkew
// of expiry. Caller must hold s.mu.
func (s *AppSource) valid() bool {
	if s.cur.token == "" {
		return false
	}
	return s.now().Add(refreshSkew).Before(s.cur.expires)
}
