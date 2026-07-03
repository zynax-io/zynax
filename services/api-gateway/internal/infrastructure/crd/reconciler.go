// SPDX-License-Identifier: Apache-2.0

// Package crd is the Kubernetes controller adapter for the thin Workflow CRD
// front-end (ADR-043, M8.E). A controller-runtime manager watches Workflow
// custom resources (zynax.io/v1alpha1) and reconciles each by calling the
// existing compile->submit path (domain.ApplyService) — the CRD is an authoring
// surface only; execution and run state stay in the engine, never in the CR
// status or etcd (ADR-040 §3).
//
// This file is the reconciler skeleton (canvas step 2): it establishes the
// generation gate that keeps GitOps resync / controller restart / leader change
// from re-triggering work. The compile->submit call and the thin status write
// land in step 3 (#1612).
package crd

import (
	"context"
	"fmt"
	"log/slog"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// WorkflowGVK identifies the Workflow custom resource this controller reconciles.
var WorkflowGVK = schema.GroupVersionKind{Group: "zynax.io", Version: "v1alpha1", Kind: "Workflow"}

// WorkflowReconciler reconciles a Workflow CR through the existing apply path.
// In this step it establishes the generation gate; the submit and status write
// arrive in step 3. It is the single status writer and so runs only on the
// Lease-elected leader (the manager keeps the default NeedLeaderElection=true).
type WorkflowReconciler struct {
	Client client.Client
}

// Reconcile implements the controller-runtime Reconciler.
func (r *WorkflowReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(WorkflowGVK)
	if err := r.Client.Get(ctx, req.NamespacedName, u); err != nil {
		// A deleted CR needs no action: run state lives in the engine, not here,
		// so there is nothing to unwind (deletion policy is future scope).
		if residual := client.IgnoreNotFound(err); residual != nil {
			return reconcile.Result{}, fmt.Errorf("crd: get workflow %s: %w", req.NamespacedName, residual)
		}
		return reconcile.Result{}, nil
	}

	if !needsReconcile(u) {
		return reconcile.Result{}, nil // spec unchanged since last observed generation: no-op
	}

	// Step 2 skeleton: the generation gate is open (spec changed). The
	// compile->submit call and the thin status write are wired in step 3.
	slog.Info("workflow reconcile: spec changed, submit deferred to step 3",
		"workflow", req.String(), "generation", u.GetGeneration())
	return reconcile.Result{}, nil
}

// needsReconcile is the re-submit gate (ADR-043 §4): reconcile only when the
// spec generation has advanced past the generation the controller last
// observed. status.observedGeneration is absent/0 on a never-reconciled CR, so
// a freshly applied CR (generation >= 1) always reconciles exactly once. This
// is what makes a GitOps resync, a controller restart, or a leader change of an
// already-dispatched, spec-unchanged workflow a no-op — no duplicate run.
func needsReconcile(u *unstructured.Unstructured) bool {
	observed, _, _ := unstructured.NestedInt64(u.Object, "status", "observedGeneration")
	return u.GetGeneration() != observed
}
