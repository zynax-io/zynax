// SPDX-License-Identifier: Apache-2.0

// Package promql is the Prometheus-backed MetricsSource of the CRD-native
// scheduler (ADR-039 §3): live load/latency are queried from the existing
// (M6) Prometheus at selection time through a short-TTL cache — never stored
// in CRD status. Any transport or decode failure surfaces as
// scheduler.ErrMetricsUnavailable so the scorer degrades instead of failing.
package promql

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/zynax-io/zynax/services/agent-registry/internal/domain/scheduler"
)

// Default instant queries. Each must return a vector labeled with the agent
// key label; series whose label matches an indexed agent feed its snapshot.
// Deployments override them (and the label) via the ZYNAX_REGISTRY_PROM_*
// env vars when their agent metrics use different names.
const (
	DefaultKeyLabel     = "zynax_agent"
	DefaultLoadQuery    = `avg by (zynax_agent) (zynax_agent_load)`
	DefaultLatencyQuery = `avg by (zynax_agent) (zynax_agent_latency_p50_ms)`
	DefaultTTL          = 5 * time.Second
	requestTimeout      = 2 * time.Second
)

// Unavailable is the MetricsSource used when no Prometheus endpoint is
// configured: every snapshot reports the backend unavailable, so selection
// runs permanently in the (correct) degraded readiness-filtered mode.
type Unavailable struct{}

// Snapshot implements scheduler.MetricsSource.
func (Unavailable) Snapshot(context.Context, []string) (map[string]scheduler.Metrics, error) {
	return nil, scheduler.ErrMetricsUnavailable
}

// Client queries a Prometheus HTTP API (/api/v1/query) and caches the merged
// snapshot for TTL — protecting the dispatch hot path from one round-trip per
// selection (ADR-039 Consequences).
type Client struct {
	BaseURL      string
	KeyLabel     string
	LoadQuery    string
	LatencyQuery string
	TTL          time.Duration
	HTTPC        *http.Client

	mu      sync.Mutex
	cached  map[string]scheduler.Metrics
	fetched time.Time
	now     func() time.Time // test seam
}

// New builds a Client with defaults applied for zero-value fields.
func New(baseURL string) *Client {
	return &Client{
		BaseURL:      baseURL,
		KeyLabel:     DefaultKeyLabel,
		LoadQuery:    DefaultLoadQuery,
		LatencyQuery: DefaultLatencyQuery,
		TTL:          DefaultTTL,
		HTTPC:        &http.Client{Timeout: requestTimeout},
		now:          time.Now,
	}
}

// Snapshot implements scheduler.MetricsSource. The full label-keyed snapshot
// is cached; per-call filtering to the requested keys is free.
func (c *Client) Snapshot(ctx context.Context, keys []string) (map[string]scheduler.Metrics, error) {
	all, err := c.snapshotAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]scheduler.Metrics, len(keys))
	for _, k := range keys {
		out[k] = all[k] // zero Metrics when the backend has no series for k
	}
	return out, nil
}

func (c *Client) snapshotAll(ctx context.Context) (map[string]scheduler.Metrics, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cached != nil && c.now().Sub(c.fetched) < c.TTL {
		return c.cached, nil
	}

	loads, err := c.instantQuery(ctx, c.LoadQuery)
	if err != nil {
		return nil, err
	}
	lats, err := c.instantQuery(ctx, c.LatencyQuery)
	if err != nil {
		return nil, err
	}

	merged := make(map[string]scheduler.Metrics, len(loads))
	for k, v := range loads {
		m := merged[k]
		m.Load = v
		merged[k] = m
	}
	for k, v := range lats {
		m := merged[k]
		m.LatencyP50Ms = v
		merged[k] = m
	}
	c.cached = merged
	c.fetched = c.now()
	return merged, nil
}

// promResponse mirrors the subset of the Prometheus HTTP API response used.
type promResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"` // [ts, "value"]
		} `json:"result"`
	} `json:"data"`
}

// instantQuery runs one /api/v1/query and returns keyLabel -> value.
func (c *Client) instantQuery(ctx context.Context, q string) (map[string]float64, error) {
	u := fmt.Sprintf("%s/api/v1/query?query=%s", c.BaseURL, url.QueryEscape(q))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build query: %w", scheduler.ErrMetricsUnavailable, err)
	}
	resp, err := c.HTTPC.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", scheduler.ErrMetricsUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("%w: status %d", scheduler.ErrMetricsUnavailable, resp.StatusCode)
	}
	var pr promResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("%w: decode: %w", scheduler.ErrMetricsUnavailable, err)
	}
	if pr.Status != "success" {
		return nil, fmt.Errorf("%w: prometheus status %q", scheduler.ErrMetricsUnavailable, pr.Status)
	}

	out := make(map[string]float64, len(pr.Data.Result))
	for _, r := range pr.Data.Result {
		key := r.Metric[c.KeyLabel]
		if key == "" || len(r.Value) != 2 {
			continue
		}
		s, ok := r.Value[1].(string)
		if !ok {
			continue
		}
		var f float64
		if _, err := fmt.Sscanf(s, "%g", &f); err != nil {
			continue
		}
		out[key] = f
	}
	return out, nil
}
