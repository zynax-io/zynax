// SPDX-License-Identifier: Apache-2.0

// Package config parses and validates the llm-adapter YAML configuration at
// startup. API-key values are never stored in config — only the name of the
// environment variable holding the key is kept; the value is resolved at
// startup into a redacting Secret type that never appears in logs, errors, or
// String()/repr output (ADR-035, canvas M7.P norms).
package config

// redactedPlaceholder is the only thing a Secret ever renders as.
const redactedPlaceholder = "[REDACTED]"

// Secret holds a credential value resolved from an environment variable at
// startup. It deliberately exposes no exported field and overrides String() and
// GoString() so the value never leaks into %s, %v, %#v, structured logs, or a
// CapabilityError.message. Retrieve the raw value only via Reveal at the
// provider call site.
type Secret struct {
	value string
}

// NewSecret wraps a raw credential value in a redacting Secret.
func NewSecret(value string) Secret {
	return Secret{value: value}
}

// Reveal returns the underlying credential value. Call this only at the point
// of use (the provider client constructor); never log or format the result.
func (s Secret) Reveal() string {
	return s.value
}

// IsZero reports whether the Secret holds no value.
func (s Secret) IsZero() bool {
	return s.value == ""
}

// String renders the redacted placeholder so a Secret is safe in any %s/%v.
func (s Secret) String() string {
	return redactedPlaceholder
}

// GoString renders the redacted placeholder so a Secret is safe in any %#v.
func (s Secret) GoString() string {
	return redactedPlaceholder
}

// MarshalText keeps the value out of any text encoder (JSON, YAML).
func (s Secret) MarshalText() ([]byte, error) {
	return []byte(redactedPlaceholder), nil
}
