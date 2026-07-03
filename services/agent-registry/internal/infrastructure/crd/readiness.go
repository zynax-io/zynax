// SPDX-License-Identifier: Apache-2.0

package crd

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// serviceNameLabel is the well-known EndpointSlice label naming the owning
// Service — the join key between an Agent's spec.endpointRef.serviceName and
// the dataplane truth about its endpoints.
const serviceNameLabel = "kubernetes.io/service-name"

// Condition constants for the Agent status.conditions entry this reconciler owns.
const (
	conditionReady        = "Ready"
	reasonServing         = "EndpointsServing"
	reasonNoEndpoints     = "NoServingEndpoints"
	messageServingFmt     = "%d serving endpoint(s) behind Service %q"
	messageNoEndpointsFmt = "no serving endpoints behind Service %q"
)

// ReadinessReconciler derives each Agent CR's status.{ready,replicas,
// conditions} from the EndpointSlices of the Service named by endpointRef —
// the ADR-039 §3 stale-liveness fix: liveness is reconciled from the
// dataplane, never self-asserted by the agent. It is the single status
// writer: the manager runs it only on the Lease-elected leader
// (NeedLeaderElection defaults to true for controllers), while the
// index/select path serves on every replica.
type ReadinessReconciler struct {
	Client client.Client
}

// Reconcile recomputes one Agent's readiness from its EndpointSlices and
// writes status only when something actually changed (low-churn guard —
// steady state produces zero etcd writes, ADR-039 §3).
func (r *ReadinessReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(AgentGVK)
	if err := r.Client.Get(ctx, req.NamespacedName, u); err != nil {
		// Deleted CRs need no status; residual errors requeue.
		if residual := client.IgnoreNotFound(err); residual != nil {
			return reconcile.Result{}, fmt.Errorf("crd: get agent %s: %w", req.NamespacedName, residual)
		}
		return reconcile.Result{}, nil
	}

	svc, _, _ := unstructured.NestedString(u.Object, "spec", "endpointRef", "serviceName")
	serving, err := r.countServingEndpoints(ctx, req.Namespace, svc)
	if err != nil {
		return reconcile.Result{}, err
	}
	ready := serving > 0

	curReady, _, _ := unstructured.NestedBool(u.Object, "status", "ready")
	curReplicas, _, _ := unstructured.NestedInt64(u.Object, "status", "replicas")
	curGen, _, _ := unstructured.NestedInt64(u.Object, "status", "observedGeneration")
	if curReady == ready && curReplicas == int64(serving) && curGen == u.GetGeneration() {
		return reconcile.Result{}, nil // steady state: no write
	}

	if err := r.writeStatus(ctx, u, svc, ready, serving); err != nil {
		return reconcile.Result{}, err
	}
	slog.Info("scheduler readiness: status reconciled",
		"agent", req.String(), "ready", ready, "serving_endpoints", serving)
	return reconcile.Result{}, nil
}

// countServingEndpoints sums ready endpoints across the Service's slices.
func (r *ReadinessReconciler) countServingEndpoints(ctx context.Context, ns, svc string) (int, error) {
	if svc == "" {
		return 0, nil
	}
	var slices discoveryv1.EndpointSliceList
	if err := r.Client.List(ctx, &slices,
		client.InNamespace(ns), client.MatchingLabels{serviceNameLabel: svc}); err != nil {
		return 0, fmt.Errorf("crd: list endpointslices for %s/%s: %w", ns, svc, err)
	}
	serving := 0
	for _, slice := range slices.Items {
		for _, ep := range slice.Endpoints {
			if ep.Conditions.Ready != nil && *ep.Conditions.Ready {
				serving++
			}
		}
	}
	return serving, nil
}

// writeStatus updates the reconciler-owned status subresource: ready,
// replicas, observedGeneration, and the single Ready condition. Dynamic
// metrics are never written here (ADR-039 §3).
func (r *ReadinessReconciler) writeStatus(ctx context.Context, u *unstructured.Unstructured, svc string, ready bool, serving int) error {
	condStatus, reason, message := "False", reasonNoEndpoints, fmt.Sprintf(messageNoEndpointsFmt, svc)
	if ready {
		condStatus, reason, message = "True", reasonServing, fmt.Sprintf(messageServingFmt, serving, svc)
	}

	// lastTransitionTime moves only when the Ready condition flips.
	transition := time.Now().UTC().Format(time.RFC3339)
	if conds, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions"); len(conds) > 0 {
		if prev, ok := conds[0].(map[string]any); ok {
			if prevStatus, _ := prev["status"].(string); prevStatus == condStatus {
				if prevTime, _ := prev["lastTransitionTime"].(string); prevTime != "" {
					transition = prevTime
				}
			}
		}
	}

	status := map[string]any{
		"ready":              ready,
		"replicas":           int64(serving),
		"observedGeneration": u.GetGeneration(),
		"conditions": []any{map[string]any{
			"type":               conditionReady,
			"status":             condStatus,
			"reason":             reason,
			"message":            message,
			"lastTransitionTime": transition,
		}},
	}
	if err := unstructured.SetNestedMap(u.Object, status, "status"); err != nil {
		return fmt.Errorf("crd: set status: %w", err)
	}
	if err := r.Client.Status().Update(ctx, u); err != nil {
		return fmt.Errorf("crd: update agent status: %w", err)
	}
	return nil
}

// SetupReadiness registers the readiness controller on the manager: it
// reconciles Agents and additionally wakes on EndpointSlice events, mapped
// back to the Agents whose endpointRef names the slice's Service. The
// controller keeps the default NeedLeaderElection=true — only the elected
// leader writes status (ADR-039 Consequences).
func SetupReadiness(mgr manager.Manager) error {
	watched := &unstructured.Unstructured{}
	watched.SetGroupVersionKind(AgentGVK)
	r := &ReadinessReconciler{Client: mgr.GetClient()}
	if err := ctrl.NewControllerManagedBy(mgr).
		Named("agent-readiness").
		For(watched).
		Watches(&discoveryv1.EndpointSlice{}, handler.EnqueueRequestsFromMapFunc(mapSliceToAgents(mgr.GetClient()))).
		Complete(r); err != nil {
		return fmt.Errorf("crd: build readiness controller: %w", err)
	}
	return nil
}

// mapSliceToAgents resolves an EndpointSlice event to reconcile requests for
// every Agent in the slice's namespace whose endpointRef.serviceName matches
// the slice's owning Service.
func mapSliceToAgents(c client.Client) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		svc := obj.GetLabels()[serviceNameLabel]
		if svc == "" {
			return nil
		}
		agents := &unstructured.UnstructuredList{}
		agents.SetGroupVersionKind(AgentGVK.GroupVersion().WithKind("AgentList"))
		if err := c.List(ctx, agents, client.InNamespace(obj.GetNamespace())); err != nil {
			slog.Error("scheduler readiness: list agents for slice mapping", "err", err)
			return nil
		}
		var reqs []reconcile.Request
		for _, a := range agents.Items {
			name, _, _ := unstructured.NestedString(a.Object, "spec", "endpointRef", "serviceName")
			if name == svc {
				reqs = append(reqs, reconcile.Request{
					NamespacedName: client.ObjectKeyFromObject(&a),
				})
			}
		}
		return reqs
	}
}
