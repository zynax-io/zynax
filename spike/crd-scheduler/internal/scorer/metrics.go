// SPDX-License-Identifier: Apache-2.0

package scorer

import (
	"context"
	"errors"
)

// Metrics is the live, dynamic snapshot for one agent — pulled from Prometheus at
// selection time, NEVER stored in CRD status.
type Metrics struct {
	Load         float64 // 0..1 utilization
	LatencyP50Ms float64
	QueueDepth   float64
}

// MetricsSource returns a live snapshot for the given agent keys. An error means
// the telemetry backend is unavailable; the scorer then degrades gracefully.
type MetricsSource interface {
	Snapshot(ctx context.Context, keys []string) (map[string]Metrics, error)
}

// ErrMetricsUnavailable signals Prometheus is down/slow; selection must degrade, not fail.
var ErrMetricsUnavailable = errors.New("metrics source unavailable")

// FakeMetrics is the spike's in-process Prometheus stub. Fail=true simulates an outage.
type FakeMetrics struct {
	Data map[string]Metrics
	Fail bool
}

// Snapshot implements MetricsSource.
func (f *FakeMetrics) Snapshot(_ context.Context, keys []string) (map[string]Metrics, error) {
	if f.Fail {
		return nil, ErrMetricsUnavailable
	}
	out := make(map[string]Metrics, len(keys))
	for _, k := range keys {
		if m, ok := f.Data[k]; ok {
			out[k] = m
		}
	}
	return out, nil
}
