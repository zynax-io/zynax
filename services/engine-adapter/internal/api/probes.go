// SPDX-License-Identifier: Apache-2.0

// Package api — probe handlers for K8s startup / readiness / liveness probes.
package api

import (
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

// Probes implements the three K8s HTTP probe handlers for a Zynax service.
// All state reads are non-blocking (atomics only) per canvas Safeguards.
type Probes struct {
	started      atomic.Bool
	lastWork     atomic.Int64 // Unix nanoseconds of last successful request
	readyChecker func() bool
	threshold    time.Duration
}

// NewProbes creates Probes using readyChecker to assess downstream gRPC health.
// ZYNAX_LIVENESS_THRESHOLD (integer seconds, default 60) sets the livez window.
func NewProbes(readyChecker func() bool) *Probes {
	p := &Probes{
		readyChecker: readyChecker,
		threshold:    livenessThreshold(),
	}
	p.lastWork.Store(time.Now().UnixNano())
	return p
}

// NewProbesWithThreshold creates Probes with an explicit liveness threshold.
// Intended for tests only; production code should use NewProbes.
func NewProbesWithThreshold(readyChecker func() bool, threshold time.Duration) *Probes {
	p := &Probes{
		readyChecker: readyChecker,
		threshold:    threshold,
	}
	p.lastWork.Store(time.Now().UnixNano())
	return p
}

// MarkStarted transitions the startup probe from 503 to 200.
// Call once initial configuration, listener binding, and gRPC dial have completed.
func (p *Probes) MarkStarted() {
	p.started.Store(true)
}

// RecordWork updates the liveness timestamp. Call from gRPC interceptor on every
// successfully completed request.
func (p *Probes) RecordWork() {
	p.lastWork.Store(time.Now().UnixNano())
}

// Startupz returns 200 after MarkStarted; 503 before.
func (p *Probes) Startupz(w http.ResponseWriter, _ *http.Request) {
	if !p.started.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Readyz returns 200 when all downstream gRPC dependencies are healthy; 503 otherwise.
func (p *Probes) Readyz(w http.ResponseWriter, _ *http.Request) {
	if !p.readyChecker() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Livez returns 200 when a successful request was recorded within the threshold window.
// Returns 503 when the service appears deadlocked (no work within threshold).
func (p *Probes) Livez(w http.ResponseWriter, _ *http.Request) {
	last := time.Unix(0, p.lastWork.Load())
	if time.Since(last) > p.threshold {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func livenessThreshold() time.Duration {
	if v := os.Getenv("ZYNAX_LIVENESS_THRESHOLD"); v != "" {
		if s, err := strconv.Atoi(v); err == nil && s > 0 {
			return time.Duration(s) * time.Second
		}
	}
	return 60 * time.Second
}
