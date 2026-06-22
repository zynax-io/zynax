// SPDX-License-Identifier: Apache-2.0

// Command poc is the ADR-039 M7 spike: a controller-runtime manager that watches
// Agent CRs (zynax.io/v1alpha1) via an informer-backed cache and maintains the
// capIndex-shaped in-memory index from reconcile events. It exposes a tiny HTTP
// surface to drive the scorer:
//
//	GET /index                      -> indexed agent count (resync proof)
//	GET /select?cap=<id>[&expert=][&fail=1]  -> runs the scorer, returns the chosen agent
//
// `fail=1` simulates Prometheus being unavailable (degradation proof). Live fake
// metrics (load/latency) are read from CR annotations so the harness can drive
// scoring without a real Prometheus:
//
//	spike.zynax.io/load        (float)
//	spike.zynax.io/latency-ms  (float)
//
// This binary is throwaway. The only durable artifact is config/crd/agents.zynax.io.yaml.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/zynax-io/zynax/spike/crd-scheduler/internal/scorer"
)

var agentGVK = schema.GroupVersionKind{Group: "zynax.io", Version: "v1alpha1", Kind: "Agent"}

// liveMetrics holds annotation-derived fake metrics, keyed by "namespace/name".
// In M8 this is replaced by a real Prometheus query client.
type liveMetrics struct {
	mu   sync.RWMutex
	data map[string]scorer.Metrics
	fail bool // set per-request via ?fail=1
}

func (l *liveMetrics) set(key string, m scorer.Metrics) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.data[key] = m
}

func (l *liveMetrics) del(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.data, key)
}

// Snapshot implements scorer.MetricsSource.
func (l *liveMetrics) Snapshot(_ context.Context, keys []string) (map[string]scorer.Metrics, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.fail {
		return nil, scorer.ErrMetricsUnavailable
	}
	out := make(map[string]scorer.Metrics, len(keys))
	for _, k := range keys {
		out[k] = l.data[k]
	}
	return out, nil
}

// reconciler maintains the index + live metrics from Agent CR reconcile events.
type reconciler struct {
	client  client.Client
	index   *scorer.Index
	metrics *liveMetrics
}

func (r *reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	key := req.Namespace + "/" + req.Name
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(agentGVK)
	if err := r.client.Get(ctx, req.NamespacedName, u); err != nil {
		// NotFound => the CR was deleted; drop it from the derived state.
		r.index.Delete(key)
		r.metrics.del(key)
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	cand, m := toCandidate(key, u)
	r.index.Upsert(cand)
	r.metrics.set(key, m)
	return reconcile.Result{}, nil
}

// toCandidate converts an Agent CR into a scorer.Candidate + its annotation-derived fake metrics.
func toCandidate(key string, u *unstructured.Unstructured) (scorer.Candidate, scorer.Metrics) {
	name := u.GetName()
	ns := u.GetNamespace()

	ready, _, _ := unstructured.NestedBool(u.Object, "status", "ready")
	svc, _, _ := unstructured.NestedString(u.Object, "spec", "endpointRef", "serviceName")
	port, _, _ := unstructured.NestedInt64(u.Object, "spec", "endpointRef", "port")
	endpoint := fmt.Sprintf("%s.%s.svc:%d", svc, ns, port)

	caps := parseCapabilities(u)

	ann := u.GetAnnotations()
	m := scorer.Metrics{
		Load:         parseFloat(ann["spike.zynax.io/load"]),
		LatencyP50Ms: parseFloat(ann["spike.zynax.io/latency-ms"]),
	}
	return scorer.Candidate{Key: key, Name: name, Endpoint: endpoint, Ready: ready, Capabilities: caps}, m
}

func parseCapabilities(u *unstructured.Unstructured) []scorer.Capability {
	raw, _, _ := unstructured.NestedSlice(u.Object, "spec", "capabilities")
	out := make([]scorer.Capability, 0, len(raw))
	for _, item := range raw {
		cm, ok := item.(map[string]any)
		if !ok {
			continue
		}
		c := scorer.Capability{}
		c.ID, _ = cm["id"].(string)
		c.InputSchema, _ = cm["inputSchema"].(string)
		if sel, ok := cm["selectors"].(map[string]any); ok {
			c.Selectors.Language = toStrings(sel["language"])
			c.Selectors.Tags = toStrings(sel["tags"])
		}
		if cost, ok := cm["cost"].(map[string]any); ok {
			c.Cost.LatencyClass, _ = cost["latencyClass"].(string)
			c.Cost.TokenPrice = parseFloat(asString(cost["tokenPrice"]))
		}
		if res, ok := cm["resources"].(map[string]any); ok {
			c.Resources.GPU = int(parseFloat(asString(res["gpu"])))
		}
		c.Models = toStrings(cm["models"])
		c.Protocols = toStrings(cm["protocols"])
		out = append(out, c)
	}
	return out
}

func toStrings(v any) []string {
	s, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(s))
	for _, e := range s {
		if str, ok := e.(string); ok {
			out = append(out, str)
		}
	}
	return out
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func main() {
	addr := flag.String("addr", ":8088", "HTTP listen address for the select API")
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	idx := scorer.NewIndex()
	metrics := &liveMetrics{data: map[string]scorer.Metrics{}}
	sc := &scorer.Scorer{Metrics: metrics}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), manager.Options{})
	if err != nil {
		panic(fmt.Errorf("create manager: %w", err))
	}

	watched := &unstructured.Unstructured{}
	watched.SetGroupVersionKind(agentGVK)
	if err := ctrl.NewControllerManagedBy(mgr).For(watched).
		Complete(&reconciler{client: mgr.GetClient(), index: idx, metrics: metrics}); err != nil {
		panic(fmt.Errorf("build controller: %w", err))
	}

	// HTTP select surface, started as a managed runnable (after cache sync).
	if err := mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
		mux.HandleFunc("/index", func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]int{"agents": idx.Len()})
		})
		mux.HandleFunc("/select", func(w http.ResponseWriter, req *http.Request) {
			metrics.fail = req.URL.Query().Get("fail") == "1"
			res, err := sc.Select(req.Context(), idx, scorer.Request{
				Capability:   req.URL.Query().Get("cap"),
				ExpertTarget: req.URL.Query().Get("expert"),
			})
			metrics.fail = false
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"chosen":               res.Chosen.Name,
				"endpoint":             res.Chosen.Endpoint,
				"prometheus_consulted": res.Rationale.PrometheusConsulted,
				"reason":               res.Rationale.Reason,
				"candidates":           res.Rationale.CandidatesConsidered,
			})
		})
		srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
		go func() { <-ctx.Done(); _ = srv.Close() }()
		return srv.ListenAndServe()
	})); err != nil {
		panic(fmt.Errorf("add http runnable: %w", err))
	}

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		panic(fmt.Errorf("manager exited: %w", err))
	}
}
