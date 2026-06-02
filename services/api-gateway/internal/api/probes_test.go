// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zynax-io/zynax/services/api-gateway/internal/api"
)

func alwaysReady() func() bool   { return func() bool { return true } }
func alwaysUnready() func() bool { return func() bool { return false } }

// Startupz ─────────────────────────────────────────────────────────────────

func TestStartupz_BeforeMarkStarted_Returns503(t *testing.T) {
	p := api.NewProbes(alwaysReady())
	w := httptest.NewRecorder()
	p.Startupz(w, httptest.NewRequest(http.MethodGet, "/startupz", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("before MarkStarted: got %d, want 503", w.Code)
	}
}

func TestStartupz_AfterMarkStarted_Returns200(t *testing.T) {
	p := api.NewProbes(alwaysReady())
	p.MarkStarted()
	w := httptest.NewRecorder()
	p.Startupz(w, httptest.NewRequest(http.MethodGet, "/startupz", nil))
	if w.Code != http.StatusOK {
		t.Errorf("after MarkStarted: got %d, want 200", w.Code)
	}
}

// Readyz ───────────────────────────────────────────────────────────────────

func TestReadyz_WhenReady_Returns200(t *testing.T) {
	p := api.NewProbes(alwaysReady())
	w := httptest.NewRecorder()
	p.Readyz(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if w.Code != http.StatusOK {
		t.Errorf("when ready: got %d, want 200", w.Code)
	}
}

func TestReadyz_WhenUnready_Returns503(t *testing.T) {
	p := api.NewProbes(alwaysUnready())
	w := httptest.NewRecorder()
	p.Readyz(w, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("when unready: got %d, want 503", w.Code)
	}
}

// Livez ────────────────────────────────────────────────────────────────────

func TestLivez_WithRecentWork_Returns200(t *testing.T) {
	p := api.NewProbes(alwaysReady())
	p.RecordWork()
	w := httptest.NewRecorder()
	p.Livez(w, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if w.Code != http.StatusOK {
		t.Errorf("with recent work: got %d, want 200", w.Code)
	}
}

func TestLivez_WithStaleWork_Returns503(t *testing.T) {
	p := api.NewProbesWithThreshold(alwaysReady(), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	w := httptest.NewRecorder()
	p.Livez(w, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("with stale work: got %d, want 503", w.Code)
	}
}

func TestLivez_RecordWorkResetsTimer(t *testing.T) {
	p := api.NewProbesWithThreshold(alwaysReady(), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	p.RecordWork()
	w := httptest.NewRecorder()
	p.Livez(w, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if w.Code != http.StatusOK {
		t.Errorf("after RecordWork: got %d, want 200", w.Code)
	}
}
