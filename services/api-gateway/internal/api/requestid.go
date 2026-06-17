// SPDX-License-Identifier: Apache-2.0

package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
)

const (
	requestIDHeader = "X-Request-ID"
	namespaceHeader = "X-Namespace"
)

// RequestIDMiddleware reads X-Request-ID from the incoming request; if absent,
// generates a random 16-byte hex ID. It also reads the optional X-Namespace
// header. Both correlation identifiers are stored in the request context so the
// downstream gRPC interceptors can attach them as metadata on every hop, and the
// request ID is echoed as X-Request-ID on the response. The W3C trace context
// (traceparent) is propagated separately by zynaxobs.HTTPMiddleware.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		ctx := domain.WithRequestID(r.Context(), id)
		if ns := r.Header.Get(namespaceHeader); ns != "" {
			ctx = domain.WithNamespace(ctx, ns)
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
