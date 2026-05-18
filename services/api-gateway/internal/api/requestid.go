// SPDX-License-Identifier: Apache-2.0

package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
)

const requestIDHeader = "X-Request-ID"

// RequestIDMiddleware reads X-Request-ID from the incoming request; if absent,
// generates a random 16-byte hex ID. The ID is stored in the request context
// and echoed as X-Request-ID on the response.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		ctx := domain.WithRequestID(r.Context(), id)
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
