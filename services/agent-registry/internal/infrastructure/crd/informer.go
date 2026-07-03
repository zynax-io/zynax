// SPDX-License-Identifier: Apache-2.0

// Package crd is the informer-backed infrastructure adapter of the CRD-native
// scheduler (ADR-039): a controller-runtime manager watches Agent custom
// resources (zynax.io/v1alpha1) and maintains the pure domain capability index
// from reconcile events. Restart recovery is a free resync — the cache Lists
// from the API server and replays Upserts; nothing is persisted (ADR-039 §2).
//
// Promoted from the KIND-verified ADR-039 spike
// (spike/adr-039-crd-scheduler-proof, cmd/poc/main.go), with the spike's
// annotation-driven fake metrics dropped: live metrics come from Prometheus at
// selection time (canvas O-step 4), never from the CR.
package crd

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/zynax-io/zynax/services/agent-registry/internal/domain/scheduler"
)

// AgentGVK identifies the Agent custom resource this adapter watches.
var AgentGVK = schema.GroupVersionKind{Group: "zynax.io", Version: "v1alpha1", Kind: "Agent"}

// expertScopeLabel is the Agent CR label carrying the strict expert scope
// (ADR-028: expert targeting moves into the request; scope rides the CR).
const expertScopeLabel = "zynax.io/expert-scope"

// Reconciler maintains the domain capability index from Agent CR events.
// It never writes to the API server — status reconciliation is a separate,
// Lease-elected reconciler (canvas O-step 5).
type Reconciler struct {
	Client client.Client
	Index  *scheduler.Index
}

// Reconcile translates one Agent CR event into an index Upsert, or a Delete
// when the CR is gone.
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	key := req.Namespace + "/" + req.Name
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(AgentGVK)
	if err := r.Client.Get(ctx, req.NamespacedName, u); err != nil {
		// NotFound => the CR was deleted; drop it from the derived state.
		r.Index.Delete(key)
		slog.Info("scheduler index: agent removed", "agent", key, "indexed", r.Index.Len())
		if residual := client.IgnoreNotFound(err); residual != nil {
			return reconcile.Result{}, fmt.Errorf("crd: get agent %s: %w", key, residual)
		}
		return reconcile.Result{}, nil
	}
	cand := ToCandidate(key, u)
	r.Index.Upsert(cand)
	slog.Info("scheduler index: agent upserted",
		"agent", key, "ready", cand.Ready, "capabilities", len(cand.Capabilities), "indexed", r.Index.Len())
	return reconcile.Result{}, nil
}

// NewManager builds a controller-runtime manager whose informer cache watches
// Agent CRs and feeds idx. The manager's own metrics/probe servers are
// disabled — the service already serves Prometheus metrics via zynaxobs.
// Leader election is deliberately NOT enabled here: the index is a read-only
// projection every replica must maintain; only the status reconciler
// (O-step 5) is single-writer.
func NewManager(restCfg *rest.Config, idx *scheduler.Index) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(restCfg, manager.Options{
		Metrics:                metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		return nil, fmt.Errorf("crd: create manager: %w", err)
	}
	watched := &unstructured.Unstructured{}
	watched.SetGroupVersionKind(AgentGVK)
	if err := ctrl.NewControllerManagedBy(mgr).For(watched).
		Complete(&Reconciler{Client: mgr.GetClient(), Index: idx}); err != nil {
		return nil, fmt.Errorf("crd: build controller: %w", err)
	}
	return mgr, nil
}

// ToCandidate converts an Agent CR into the scheduler's domain view.
// Endpoint resolves endpointRef to "<serviceName>.<namespace>.svc:<port>";
// readiness comes from the reconciler-owned status subresource.
func ToCandidate(key string, u *unstructured.Unstructured) scheduler.Candidate {
	ns := u.GetNamespace()

	ready, _, _ := unstructured.NestedBool(u.Object, "status", "ready")
	replicas, _, _ := unstructured.NestedInt64(u.Object, "status", "replicas")
	svc, _, _ := unstructured.NestedString(u.Object, "spec", "endpointRef", "serviceName")
	port, _, _ := unstructured.NestedInt64(u.Object, "spec", "endpointRef", "port")

	return scheduler.Candidate{
		Key:          key,
		Name:         u.GetName(),
		Endpoint:     fmt.Sprintf("%s.%s.svc:%d", svc, ns, port),
		Ready:        ready,
		Replicas:     int(replicas),
		ExpertScope:  u.GetLabels()[expertScopeLabel],
		Capabilities: parseCapabilities(u),
	}
}

// parseCapabilities extracts spec.capabilities[] with the scoring hints.
// Malformed entries are skipped rather than failing the whole CR — the CRD's
// OpenAPI schema is the validation gate; this is defensive decoding only.
func parseCapabilities(u *unstructured.Unstructured) []scheduler.Capability {
	raw, _, _ := unstructured.NestedSlice(u.Object, "spec", "capabilities")
	out := make([]scheduler.Capability, 0, len(raw))
	for _, item := range raw {
		cm, ok := item.(map[string]any)
		if !ok {
			continue
		}
		c := scheduler.Capability{}
		c.ID, _ = cm["id"].(string)
		c.Description, _ = cm["description"].(string)
		c.InputSchema, _ = cm["inputSchema"].(string)
		c.OutputSchema, _ = cm["outputSchema"].(string)
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
