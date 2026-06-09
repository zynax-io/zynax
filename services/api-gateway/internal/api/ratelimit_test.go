// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── clientIP tests ────────────────────────────────────────────────────────

func TestClientIP(t *testing.T) {
	t.Parallel()
	// Use RFC 5737 documentation IPs (TEST-NET-3: 203.0.113.x) — safe for tests.
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		want       string
	}{
		{
			name:       "uses RemoteAddr when no XFF",
			remoteAddr: "203.0.113.1:54321",
			want:       "203.0.113.1",
		},
		{
			name:       "XFF single entry takes precedence",
			remoteAddr: "203.0.113.1:80",
			xff:        "203.0.113.5",
			want:       "203.0.113.5",
		},
		{
			name:       "XFF leftmost entry used when multiple",
			remoteAddr: "203.0.113.1:80",
			xff:        "203.0.113.5, 203.0.113.6, 203.0.113.7",
			want:       "203.0.113.5",
		},
		{
			name:       "XFF with leading whitespace trimmed",
			remoteAddr: "203.0.113.1:80",
			xff:        " 203.0.113.9 , 203.0.113.2",
			want:       "203.0.113.9",
		},
		{
			name:       "RemoteAddr without port passes through",
			remoteAddr: "203.0.113.2",
			want:       "203.0.113.2",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				r.Header.Set("X-Forwarded-For", tc.xff)
			}
			got := clientIP(r)
			if got != tc.want {
				t.Errorf("clientIP() = %q; want %q", got, tc.want)
			}
		})
	}
}

// ── env var config tests ──────────────────────────────────────────────────

func TestNewIPRateLimiterDefaults(t *testing.T) {
	t.Setenv("RATE_LIMIT_RPS", "")
	t.Setenv("RATE_LIMIT_BURST", "")
	rl := newIPRateLimiter()
	if float64(rl.rps) != 10.0 {
		t.Errorf("default rps = %v; want 10.0", rl.rps)
	}
	if rl.burst != 20 {
		t.Errorf("default burst = %d; want 20", rl.burst)
	}
}

func TestNewIPRateLimiterEnvOverride(t *testing.T) {
	t.Setenv("RATE_LIMIT_RPS", "50.5")
	t.Setenv("RATE_LIMIT_BURST", "100")
	rl := newIPRateLimiter()
	if float64(rl.rps) != 50.5 {
		t.Errorf("rps = %v; want 50.5", rl.rps)
	}
	if rl.burst != 100 {
		t.Errorf("burst = %d; want 100", rl.burst)
	}
}

func TestNewIPRateLimiterInvalidEnvIgnored(t *testing.T) {
	t.Setenv("RATE_LIMIT_RPS", "notanumber")
	t.Setenv("RATE_LIMIT_BURST", "alsowrong")
	rl := newIPRateLimiter()
	if float64(rl.rps) != 10.0 {
		t.Errorf("invalid rps env should use default 10.0; got %v", rl.rps)
	}
	if rl.burst != 20 {
		t.Errorf("invalid burst env should use default 20; got %d", rl.burst)
	}
}

func TestNewIPRateLimiterZeroEnvIgnored(t *testing.T) {
	t.Setenv("RATE_LIMIT_RPS", "0")
	t.Setenv("RATE_LIMIT_BURST", "0")
	rl := newIPRateLimiter()
	if float64(rl.rps) != 10.0 {
		t.Errorf("zero rps env should use default 10.0; got %v", rl.rps)
	}
	if rl.burst != 20 {
		t.Errorf("zero burst env should use default 20; got %d", rl.burst)
	}
}

// ── middleware accept / reject tests ─────────────────────────────────────

// okHandler is a trivial next handler that writes 200.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// newRequestWithIP creates a POST /api/v1/apply test request from the given IP.
// Uses RFC 5737 TEST-NET IPs (203.0.113.x) by convention.
func newRequestWithIP(ip string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/apply", strings.NewReader("{}"))
	r.RemoteAddr = ip + ":12345"
	return r
}

func TestRateLimitMiddlewareAllows(t *testing.T) {
	t.Setenv("RATE_LIMIT_RPS", "1000")
	t.Setenv("RATE_LIMIT_BURST", "1000")
	rl := newIPRateLimiter()
	handler := rl.Middleware(okHandler)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, newRequestWithIP("203.0.113.10"))
	if rr.Code != http.StatusOK {
		t.Errorf("first request: got %d; want 200", rr.Code)
	}
}

func TestRateLimitMiddlewareRejects(t *testing.T) {
	// burst=1 so the second request from the same IP must be rejected
	t.Setenv("RATE_LIMIT_RPS", "0.001")
	t.Setenv("RATE_LIMIT_BURST", "1")
	rl := newIPRateLimiter()
	handler := rl.Middleware(okHandler)

	ip := "203.0.113.20"
	// First request consumes the single token.
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, newRequestWithIP(ip))
	if rr1.Code != http.StatusOK {
		t.Fatalf("first request should pass; got %d", rr1.Code)
	}

	// Second request from same IP must be rate-limited.
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, newRequestWithIP(ip))
	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("second request: got %d; want 429", rr2.Code)
	}

	// Verify JSON body contains expected code.
	var body map[string]string
	if err := json.NewDecoder(rr2.Body).Decode(&body); err != nil {
		t.Fatalf("decode 429 body: %v", err)
	}
	if body["code"] != "RATE_LIMITED" {
		t.Errorf("body.code = %q; want RATE_LIMITED", body["code"])
	}
}

func TestRateLimitMiddlewarePerIPIsolation(t *testing.T) {
	// burst=1: each IP gets exactly one token; different IPs don't share state.
	t.Setenv("RATE_LIMIT_RPS", "0.001")
	t.Setenv("RATE_LIMIT_BURST", "1")
	rl := newIPRateLimiter()
	handler := rl.Middleware(okHandler)

	// Three distinct RFC 5737 documentation IPs — each must have its own bucket.
	ips := []string{"203.0.113.30", "203.0.113.31", "203.0.113.32"}
	for _, ip := range ips {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, newRequestWithIP(ip))
		if rr.Code != http.StatusOK {
			t.Errorf("IP %s first request: got %d; want 200", ip, rr.Code)
		}
	}
}

func TestRateLimitMiddlewareContentType(t *testing.T) {
	// burst=1 so second request triggers 429.
	t.Setenv("RATE_LIMIT_RPS", "0.001")
	t.Setenv("RATE_LIMIT_BURST", "1")
	rl := newIPRateLimiter()
	handler := rl.Middleware(okHandler)

	ip := "203.0.113.40"
	// Exhaust the single token.
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, newRequestWithIP(ip))

	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, newRequestWithIP(ip))
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429; got %d", rr2.Code)
	}
	ct := rr2.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q; want application/json", ct)
	}
}
