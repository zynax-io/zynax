// SPDX-License-Identifier: Apache-2.0

package credential

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeMinter is a controllable Minter: it returns a token derived from a counter
// and an expiry the test sets, never touching the network.
type fakeMinter struct {
	calls   int
	ttl     time.Duration
	now     func() time.Time
	failNow bool
	err     error
}

func (f *fakeMinter) Mint(_ context.Context) (string, time.Time, error) {
	f.calls++
	if f.failNow {
		if f.err != nil {
			return "", time.Time{}, f.err
		}
		return "", time.Time{}, errors.New("mint failed")
	}
	exp := f.now().Add(f.ttl)
	return tokenFor(f.calls), exp, nil
}

func tokenFor(n int) string {
	return "ghs_installation_token_v" + string(rune('0'+n)) //nolint:gosec // test helper: n is a small bounded loop index
}

func TestStaticSource_AlwaysReturnsSameTokenNoError(t *testing.T) {
	s := NewStaticSource("ghp_classic_pat_value")
	for i := 0; i < 3; i++ {
		got, err := s.Token(context.Background())
		if err != nil {
			t.Fatalf("StaticSource.Token returned error: %v", err)
		}
		if got != "ghp_classic_pat_value" {
			t.Fatalf("StaticSource.Token = %q, want the injected PAT", got)
		}
	}
}

func TestAppSource_MintsLazilyAndCaches(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	m := &fakeMinter{ttl: time.Hour, now: clock}
	s := NewAppSource(m, clock)

	first, err := s.Token(context.Background())
	if err != nil {
		t.Fatalf("first Token: %v", err)
	}
	if m.calls != 1 {
		t.Fatalf("expected lazy mint on first call, got calls=%d", m.calls)
	}

	// A second call well before expiry must reuse the cached token (no new mint).
	second, err := s.Token(context.Background())
	if err != nil {
		t.Fatalf("second Token: %v", err)
	}
	if second != first {
		t.Fatalf("expected cached token reuse, got %q then %q", first, second)
	}
	if m.calls != 1 {
		t.Fatalf("expected no re-mint within TTL, got calls=%d", m.calls)
	}
}

func TestAppSource_RefreshesBeforeExpiry(t *testing.T) {
	base := time.Now()
	cur := base
	clock := func() time.Time { return cur }
	m := &fakeMinter{ttl: time.Hour, now: clock}
	s := NewAppSource(m, clock)

	first, err := s.Token(context.Background())
	if err != nil {
		t.Fatalf("first Token: %v", err)
	}

	// Advance the clock to within refreshSkew of expiry — the next call must re-mint.
	cur = base.Add(time.Hour - (refreshSkew / 2))
	second, err := s.Token(context.Background())
	if err != nil {
		t.Fatalf("second Token: %v", err)
	}
	if second == first {
		t.Fatalf("expected a refreshed token near expiry, got the same value %q", first)
	}
	if m.calls != 2 {
		t.Fatalf("expected exactly one re-mint near expiry, got calls=%d", m.calls)
	}
}

// AC: requests after the original token's TTL succeed without a restart.
func TestAppSource_RequestAfterOriginalTTLSucceeds(t *testing.T) {
	base := time.Now()
	cur := base
	clock := func() time.Time { return cur }
	m := &fakeMinter{ttl: time.Hour, now: clock}
	s := NewAppSource(m, clock)

	if _, err := s.Token(context.Background()); err != nil {
		t.Fatalf("initial Token: %v", err)
	}
	// Move the clock past the original token's full TTL — a static read-once PAT
	// would now be expired; the App source must mint a fresh, valid one.
	cur = base.Add(2 * time.Hour)
	got, err := s.Token(context.Background())
	if err != nil {
		t.Fatalf("Token after original TTL: %v", err)
	}
	if got == "" {
		t.Fatal("expected a valid refreshed token after the original TTL")
	}
	if m.calls != 2 {
		t.Fatalf("expected a refresh after TTL, got calls=%d", m.calls)
	}
}

func TestAppSource_MintFailureWithNoHeldTokenErrors(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	m := &fakeMinter{ttl: time.Hour, now: clock, failNow: true}
	s := NewAppSource(m, clock)

	_, err := s.Token(context.Background())
	if err == nil {
		t.Fatal("expected error when first mint fails and no token is held")
	}
}

func TestAppSource_MintFailureFallsBackToUnexpiredToken(t *testing.T) {
	base := time.Now()
	cur := base
	clock := func() time.Time { return cur }
	m := &fakeMinter{ttl: time.Hour, now: clock}
	s := NewAppSource(m, clock)

	first, err := s.Token(context.Background())
	if err != nil {
		t.Fatalf("first Token: %v", err)
	}

	// Near expiry the source tries to refresh; make minting fail. The held token
	// is still before hard expiry, so it should be returned rather than erroring.
	cur = base.Add(time.Hour - (refreshSkew / 2))
	m.failNow = true
	got, err := s.Token(context.Background())
	if err != nil {
		t.Fatalf("expected fallback to held token, got error: %v", err)
	}
	if got != first {
		t.Fatalf("expected the still-valid held token %q, got %q", first, got)
	}
}

func TestAppSource_MintFailureAfterHardExpiryErrors(t *testing.T) {
	base := time.Now()
	cur := base
	clock := func() time.Time { return cur }
	m := &fakeMinter{ttl: time.Hour, now: clock}
	s := NewAppSource(m, clock)

	if _, err := s.Token(context.Background()); err != nil {
		t.Fatalf("first Token: %v", err)
	}
	// Past hard expiry with minting down: no usable token remains.
	cur = base.Add(2 * time.Hour)
	m.failNow = true
	if _, err := s.Token(context.Background()); err == nil {
		t.Fatal("expected error when held token is hard-expired and mint fails")
	}
}

func TestNewAppSource_NilClockDefaultsToTimeNow(t *testing.T) {
	m := &fakeMinter{ttl: time.Hour, now: time.Now}
	s := NewAppSource(m, nil)
	if s.now == nil {
		t.Fatal("expected a default clock when nil is passed")
	}
	if _, err := s.Token(context.Background()); err != nil {
		t.Fatalf("Token with default clock: %v", err)
	}
}
