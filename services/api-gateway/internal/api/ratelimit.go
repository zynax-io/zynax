// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

// ipRateLimiter maintains per-IP token-bucket limiters.
// Rate parameters are read once at construction from env vars:
//   - RATE_LIMIT_RPS  (float64, default 10.0) — sustained request rate
//   - RATE_LIMIT_BURST (int,     default 20)   — burst capacity
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

func newIPRateLimiter() *ipRateLimiter {
	rps := 10.0
	burst := 20
	if v := os.Getenv("RATE_LIMIT_RPS"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			rps = f
		}
	}
	if v := os.Getenv("RATE_LIMIT_BURST"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			burst = i
		}
	}
	return &ipRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

// getLimiter returns (creating if needed) the per-IP limiter for the given IP.
func (rl *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if l, ok := rl.limiters[ip]; ok {
		return l
	}
	l := rate.NewLimiter(rl.rps, rl.burst)
	rl.limiters[ip] = l
	return l
}

// Middleware returns an http.Handler that enforces the per-IP token-bucket rate
// limit. Requests that exceed the limit receive HTTP 429 with a JSON body
// {"code":"RATE_LIMITED"}. Requests within the limit are passed to next.
func (rl *ipRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.getLimiter(ip).Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{"code": "RATE_LIMITED"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the real client IP from the request.
// X-Forwarded-For takes precedence over RemoteAddr (leftmost entry is the
// original client when the gateway is behind a trusted reverse proxy).
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
