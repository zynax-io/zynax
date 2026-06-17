// SPDX-License-Identifier: Apache-2.0

package credential

import (
	"fmt"
	"net/http"
)

// Transport is an http.RoundTripper that injects the current token from a Source
// as a Bearer Authorization header on every outbound request. Resolving the token
// per request (rather than baking it into the client once) is what makes refresh
// transparent: when an AppSource re-mints, the next request automatically carries
// the new token with no client rebuild.
//
// The token is set on a shallow clone of the request so the caller's Header map is
// never mutated, and it is written only to the Authorization header — never logged.
type Transport struct {
	src  Source
	base http.RoundTripper
}

// NewTransport wraps base (or http.DefaultTransport when nil) so each request is
// authenticated with src's current token.
func NewTransport(src Source, base http.RoundTripper) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &Transport{src: src, base: base}
}

// RoundTrip resolves the current token and attaches it as a Bearer header. A token
// resolution failure aborts the request with a non-secret error.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.src.Token(req.Context())
	if err != nil {
		return nil, fmt.Errorf("credential: resolve token for request: %w", err)
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+token)
	return t.base.RoundTrip(clone) //nolint:wrapcheck // transport error surfaced as-is to the http client
}
