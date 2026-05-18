// SPDX-License-Identifier: Apache-2.0

package api

import "net/http"

// requireBearer wraps next with bearer-token authentication when key is non-empty.
// If key is empty the gate is disabled and all requests pass through unchanged.
// Callers should log a warning at startup when key is empty (see main.go).
func requireBearer(key string, next http.HandlerFunc) http.HandlerFunc {
	if key == "" {
		return next
	}
	want := "Bearer " + key
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != want {
			writeError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
			return
		}
		next(w, r)
	}
}
