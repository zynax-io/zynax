// SPDX-License-Identifier: Apache-2.0

// Package redact scrubs credential material from any string that may reach a
// log line, gRPC trace, or — most sensitively — an MCP tool result that becomes
// model prompt content (EPIC G, story G.3 / #1199, ADR-032).
//
// The credential model is inject-at-start: the token is resolved once from an
// env var (see internal/config.ResolveToken) and held only in process memory; it
// is never a config field, never a tool argument, and never serialized. This
// package is the egress backstop for the one path a token could still slip
// through — a transport or git error message that embeds the token (for example
// an authenticated remote URL of the form https://x-access-token:TOKEN@host/...
// echoed back inside an upstream error string).
//
// A Redactor is value-type, immutable after construction, and safe for
// concurrent use. The empty (zero) Redactor is a valid no-op.
package redact

import "strings"

// placeholder replaces every occurrence of a registered secret.
const placeholder = "[REDACTED]"

// minSecretLen guards against redacting trivially short or empty tokens, which
// would scrub innocuous substrings out of every message. A real GitHub PAT or
// App token is well above this length; anything shorter is treated as absent.
const minSecretLen = 8

// Redactor replaces known secret values with a fixed placeholder. It never holds
// the env-var name or any metadata — only the literal secret values to scrub —
// and it exposes no accessor that returns them, so the secret cannot be read
// back out once registered.
type Redactor struct {
	secrets []string
}

// New builds a Redactor for the given secret values. Empty values and values
// shorter than minSecretLen are ignored (a short or absent token cannot be
// meaningfully scrubbed and would over-redact). Duplicate values are collapsed.
func New(secrets ...string) Redactor {
	seen := make(map[string]struct{}, len(secrets))
	kept := make([]string, 0, len(secrets))
	for _, s := range secrets {
		if len(s) < minSecretLen {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		kept = append(kept, s)
	}
	return Redactor{secrets: kept}
}

// String replaces every occurrence of every registered secret in s with
// placeholder. With no registered secrets it returns s unchanged. The result
// never contains a registered secret value.
func (r Redactor) String(s string) string {
	if s == "" || len(r.secrets) == 0 {
		return s
	}
	for _, secret := range r.secrets {
		s = strings.ReplaceAll(s, secret, placeholder)
	}
	return s
}

// Bytes redacts a byte slice (e.g. a JSON output payload that becomes a tool
// result) and returns a freshly allocated, scrubbed slice. The input is never
// mutated. A nil input returns nil.
func (r Redactor) Bytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	if len(r.secrets) == 0 {
		return b
	}
	return []byte(r.String(string(b)))
}
