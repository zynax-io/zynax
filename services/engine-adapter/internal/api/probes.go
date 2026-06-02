// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"
	"sync/atomic"
	"time"
)

// Probes holds shared atomic state for the three K8s probe endpoints.
// All methods are safe for concurrent use without additional locking.
type Probes struct {
	started     atomic.Bool
	lastWorkNs  atomic.Int64 // Unix nanoseconds of last successful request; 0 = never
	readyFn     func() bool
	thresholdNs int64 // liveness threshold in nanoseconds
}

// NewProbes constructs a Probes instance. thresholdS is the liveness threshold
// in seconds (ZYNAX_ENGINE_ADAPTER_LIVENESS_THRESHOLD_S). readyFn returns true
// when all downstream gRPC connections are in an acceptable state; it must not block.
func NewProbes(thresholdS int64, readyFn func() bool) *Probes {
	return &Probes{
		readyFn:     readyFn,
		thresholdNs: thresholdS * int64(time.Second),
	}
}

// MarkStarted signals that the service has finished startup (Temporal worker
// started, gRPC server listening). Call once from main after both are running.
func (p *Probes) MarkStarted() { p.started.Store(true) }

// RecordWork records the current time as the last successful work timestamp.
// Call this after each gRPC request that completes without error.
func (p *Probes) RecordWork() { p.lastWorkNs.Store(time.Now().UnixNano()) }

// HandleStartupz returns 503 until MarkStarted has been called, then 200.
func (p *Probes) HandleStartupz(w http.ResponseWriter, _ *http.Request) {
	if !p.started.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleReadyz returns 503 if any downstream gRPC dependency is not ready.
// It never performs blocking network calls — readyFn only reads connection state.
func (p *Probes) HandleReadyz(w http.ResponseWriter, _ *http.Request) {
	if !p.readyFn() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleLivez returns 503 if no successful request has been recorded within the
// liveness threshold. Returns 200 when no work has ever been done (K8s
// initialDelaySeconds provides the startup grace period for liveness).
func (p *Probes) HandleLivez(w http.ResponseWriter, _ *http.Request) {
	last := p.lastWorkNs.Load()
	if last != 0 && time.Now().UnixNano()-last > p.thresholdNs {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Register mounts the three probe handlers plus a backward-compatible /healthz
// alias onto mux.
func (p *Probes) Register(mux *http.ServeMux) {
	mux.HandleFunc("/startupz", p.HandleStartupz)
	mux.HandleFunc("/readyz", p.HandleReadyz)
	mux.HandleFunc("/livez", p.HandleLivez)
	// /healthz retained for backward compatibility with existing orchestration.
	mux.HandleFunc("/healthz", p.HandleLivez)
}
