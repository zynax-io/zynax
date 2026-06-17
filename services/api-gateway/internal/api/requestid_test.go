// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
)

func TestRequestIDMiddleware_GeneratesAndEchoes(t *testing.T) {
	var gotID, gotNS string
	h := RequestIDMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotID = domain.RequestIDFromContext(r.Context())
		gotNS = domain.NamespaceFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if gotID == "" {
		t.Fatal("expected a generated request ID on the context")
	}
	if rec.Header().Get(requestIDHeader) != gotID {
		t.Errorf("response %s = %q; want %q", requestIDHeader, rec.Header().Get(requestIDHeader), gotID)
	}
	if gotNS != "" {
		t.Errorf("namespace = %q; want empty when no header sent", gotNS)
	}
}

func TestRequestIDMiddleware_HonoursIncomingHeaders(t *testing.T) {
	var gotID, gotNS string
	h := RequestIDMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotID = domain.RequestIDFromContext(r.Context())
		gotNS = domain.NamespaceFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/x", nil)
	req.Header.Set(requestIDHeader, "req-abc")
	req.Header.Set(namespaceHeader, "team-a")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if gotID != "req-abc" {
		t.Errorf("request ID = %q; want %q", gotID, "req-abc")
	}
	if gotNS != "team-a" {
		t.Errorf("namespace = %q; want %q", gotNS, "team-a")
	}
	if rec.Header().Get(requestIDHeader) != "req-abc" {
		t.Errorf("echoed %s = %q; want %q", requestIDHeader, rec.Header().Get(requestIDHeader), "req-abc")
	}
}
