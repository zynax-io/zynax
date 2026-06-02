// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zynax-io/zynax/services/api-gateway/internal/api"
)

func alwaysReady() bool { return true }
func neverReady() bool  { return false }

func get(handler func(http.ResponseWriter, *http.Request)) int {
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	return rec.Code
}

// ── /startupz ────────────────────────────────────────────────────────────────

func TestProbes_Startupz_Before_Ready(t *testing.T) {
	p := api.NewProbes(60, alwaysReady)
	if got := get(p.HandleStartupz); got != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", got)
	}
}

func TestProbes_Startupz_After_Ready(t *testing.T) {
	p := api.NewProbes(60, alwaysReady)
	p.MarkStarted()
	if got := get(p.HandleStartupz); got != http.StatusOK {
		t.Fatalf("want 200, got %d", got)
	}
}

// ── /readyz ───────────────────────────────────────────────────────────────────

func TestProbes_Readyz_Deps_Ready(t *testing.T) {
	p := api.NewProbes(60, alwaysReady)
	if got := get(p.HandleReadyz); got != http.StatusOK {
		t.Fatalf("want 200, got %d", got)
	}
}

func TestProbes_Readyz_Deps_Fail(t *testing.T) {
	p := api.NewProbes(60, neverReady)
	if got := get(p.HandleReadyz); got != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", got)
	}
}

// ── /livez ────────────────────────────────────────────────────────────────────

func TestProbes_Livez_No_Work_Done(t *testing.T) {
	// No RecordWork call → 200 (startup grace — initialDelaySeconds handles this).
	p := api.NewProbes(60, alwaysReady)
	if got := get(p.HandleLivez); got != http.StatusOK {
		t.Fatalf("want 200, got %d", got)
	}
}

func TestProbes_Livez_Recent_Work(t *testing.T) {
	p := api.NewProbes(60, alwaysReady)
	p.RecordWork()
	if got := get(p.HandleLivez); got != http.StatusOK {
		t.Fatalf("want 200, got %d", got)
	}
}

func TestProbes_Livez_Stale_Work(t *testing.T) {
	// threshold 0 forces any previous timestamp to be stale.
	p := api.NewProbes(0, alwaysReady)
	p.RecordWork()
	// Sleep 1 ms so Now()-last > 0 threshold.
	time.Sleep(time.Millisecond)
	if got := get(p.HandleLivez); got != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", got)
	}
}
