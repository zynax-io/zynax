// SPDX-License-Identifier: Apache-2.0

package api

import (
	"crypto/subtle"
	"net/http"
)

// requireBearer wraps next with bearer-token authentication when key is non-empty.
// If key is empty the gate is disabled and all requests pass through unchanged.
// Callers should log a warning at startup when key is empty (see main.go).
func requireBearer(key string, next http.HandlerFunc) http.HandlerFunc {
	if key == "" {
		return next
	}
	want := []byte("Bearer " + key)
	return func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, want) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
			return
		}
		next(w, r)
	}
}
