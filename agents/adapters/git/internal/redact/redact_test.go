// SPDX-License-Identifier: Apache-2.0

package redact_test

import (
	"strings"
	"testing"

	"github.com/zynax-io/zynax/agents/adapters/git/internal/redact"
)

// fakeToken is a syntactically PAT-shaped placeholder — never a real credential.
const fakeToken = "ghp_FAKE0000000000000000000000000000fake" //nolint:gosec // test fixture, not a real secret

func TestString_RedactsToken(t *testing.T) {
	t.Parallel()
	r := redact.New(fakeToken)
	// Simulate the one real leak path: an authenticated remote URL echoed inside
	// an upstream error message.
	in := "fatal: unable to access 'https://x-access-token:" + fakeToken + "@github.com/o/r.git/': 403"
	got := r.String(in)
	if strings.Contains(got, fakeToken) {
		t.Fatalf("token leaked through redaction: %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Errorf("expected [REDACTED] placeholder, got %q", got)
	}
}

func TestString_NoSecretsIsNoOp(t *testing.T) {
	t.Parallel()
	r := redact.New()
	in := "no secrets here"
	if got := r.String(in); got != in {
		t.Errorf("empty redactor mutated input: got %q want %q", got, in)
	}
}

func TestString_EmptyInput(t *testing.T) {
	t.Parallel()
	r := redact.New(fakeToken)
	if got := r.String(""); got != "" {
		t.Errorf("empty input should return empty, got %q", got)
	}
}

func TestZeroRedactorIsNoOp(t *testing.T) {
	t.Parallel()
	var r redact.Redactor // zero value
	in := "anything " + fakeToken
	if got := r.String(in); got != in {
		t.Errorf("zero redactor must be a no-op, got %q", got)
	}
}

func TestNew_IgnoresShortAndEmptyAndDuplicates(t *testing.T) {
	t.Parallel()
	// "short" (< minSecretLen) and "" must be ignored; the long token deduped.
	r := redact.New("", "short", fakeToken, fakeToken)
	out := r.String("x short y " + fakeToken)
	if strings.Contains(out, fakeToken) {
		t.Fatalf("token leaked: %q", out)
	}
	// "short" must survive — redacting it would scrub innocuous substrings.
	if !strings.Contains(out, "short") {
		t.Errorf("a sub-minimum-length value must not be redacted, got %q", out)
	}
}

func TestBytes_RedactsAndDoesNotMutateInput(t *testing.T) {
	t.Parallel()
	r := redact.New(fakeToken)
	in := []byte(`{"diff":"token=` + fakeToken + `"}`)
	orig := string(in)
	out := r.Bytes(in)
	if strings.Contains(string(out), fakeToken) {
		t.Fatalf("token leaked through Bytes: %q", out)
	}
	if string(in) != orig {
		t.Errorf("Bytes mutated its input: %q", in)
	}
}

func TestBytes_NilAndNoSecrets(t *testing.T) {
	t.Parallel()
	if got := redact.New(fakeToken).Bytes(nil); got != nil {
		t.Errorf("nil input must return nil, got %v", got)
	}
	plain := []byte("plain")
	if got := redact.New().Bytes(plain); string(got) != "plain" {
		t.Errorf("no-secret redactor must pass bytes through, got %q", got)
	}
}
