// SPDX-License-Identifier: Apache-2.0

package promql

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zynax-io/zynax/services/agent-registry/internal/domain/scheduler"
)

func promBody(label string, series map[string]float64) string {
	out := `{"status":"success","data":{"result":[`
	first := true
	for k, v := range series {
		if !first {
			out += ","
		}
		first = false
		out += fmt.Sprintf(`{"metric":{"%s":"%s"},"value":[1700000000,"%g"]}`, label, k, v)
	}
	return out + `]}}`
}

func TestSnapshot_MergesLoadAndLatency(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		q := r.URL.Query().Get("query")
		if q == DefaultLoadQuery {
			_, _ = w.Write([]byte(promBody(DefaultKeyLabel, map[string]float64{"default/a": 0.4})))
			return
		}
		_, _ = w.Write([]byte(promBody(DefaultKeyLabel, map[string]float64{"default/a": 120})))
	}))
	defer srv.Close()

	c := New(srv.URL)
	got, err := c.Snapshot(context.Background(), []string{"default/a", "default/ghost"})
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if got["default/a"].Load != 0.4 || got["default/a"].LatencyP50Ms != 120 {
		t.Errorf("merged = %+v", got["default/a"])
	}
	// Unknown keys resolve to the zero snapshot, not an error.
	if got["default/ghost"] != (scheduler.Metrics{}) {
		t.Errorf("ghost = %+v, want zero", got["default/ghost"])
	}
}

func TestSnapshot_TTLCacheAvoidsHotPathQueries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(promBody(DefaultKeyLabel, nil)))
	}))
	defer srv.Close()

	c := New(srv.URL)
	now := time.Unix(0, 0)
	c.now = func() time.Time { return now }

	for range 5 {
		if _, err := c.Snapshot(context.Background(), []string{"default/a"}); err != nil {
			t.Fatalf("Snapshot: %v", err)
		}
	}
	if n := calls.Load(); n != 2 { // one load + one latency query, then cache
		t.Errorf("backend calls = %d, want 2 (TTL cache)", n)
	}

	// Past the TTL the cache refreshes.
	now = now.Add(DefaultTTL + time.Second)
	if _, err := c.Snapshot(context.Background(), []string{"default/a"}); err != nil {
		t.Fatalf("Snapshot after TTL: %v", err)
	}
	if n := calls.Load(); n != 4 {
		t.Errorf("backend calls = %d, want 4 after TTL expiry", n)
	}
}

func TestSnapshot_UnavailableVariants(t *testing.T) {
	// Transport down.
	down := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	down.Close()
	if _, err := New(down.URL).Snapshot(context.Background(), []string{"k"}); !errors.Is(err, scheduler.ErrMetricsUnavailable) {
		t.Errorf("transport: err = %v, want ErrMetricsUnavailable", err)
	}

	// Non-200.
	e500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer e500.Close()
	if _, err := New(e500.URL).Snapshot(context.Background(), []string{"k"}); !errors.Is(err, scheduler.ErrMetricsUnavailable) {
		t.Errorf("status: err = %v, want ErrMetricsUnavailable", err)
	}

	// Malformed body.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{not json"))
	}))
	defer bad.Close()
	if _, err := New(bad.URL).Snapshot(context.Background(), []string{"k"}); !errors.Is(err, scheduler.ErrMetricsUnavailable) {
		t.Errorf("decode: err = %v, want ErrMetricsUnavailable", err)
	}

	// Prometheus-level error status.
	perr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"error","data":{"result":[]}}`))
	}))
	defer perr.Close()
	if _, err := New(perr.URL).Snapshot(context.Background(), []string{"k"}); !errors.Is(err, scheduler.ErrMetricsUnavailable) {
		t.Errorf("prom status: err = %v, want ErrMetricsUnavailable", err)
	}
}

func TestUnavailable_AlwaysDegrades(t *testing.T) {
	if _, err := (Unavailable{}).Snapshot(context.Background(), []string{"k"}); !errors.Is(err, scheduler.ErrMetricsUnavailable) {
		t.Errorf("err = %v, want ErrMetricsUnavailable", err)
	}
}
