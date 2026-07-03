// SPDX-License-Identifier: Apache-2.0

// Package crd is the Kubernetes controller adapter for the thin Workflow CRD
// front-end (ADR-043, M8.E). A controller-runtime manager watches Workflow
// custom resources (zynax.io/v1alpha1) and reconciles each by calling the
// existing compile->submit path (domain.ApplyService) — the CRD is an authoring
// surface only; execution and run state stay in the engine, never in the CR
// status or etcd (ADR-040 §3).
package crd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
)

// WorkflowGVK identifies the Workflow custom resource this controller reconciles.
var WorkflowGVK = schema.GroupVersionKind{Group: "zynax.io", Version: "v1alpha1", Kind: "Workflow"}

// manifestAPIVersion is the apiVersion the compiler expects. The CRD serves
// v1alpha1; the manifest schema pins zynax.io/v1, so the reconciler maps
// v1alpha1 -> v1 when it reconstructs the manifest (ADR-043 §5).
const manifestAPIVersion = "zynax.io/v1"

// Condition types on the Workflow status this reconciler owns.
const (
	conditionCompiled   = "Compiled"
	conditionDispatched = "Dispatched"
)

// WorkflowApplier is the slice of domain.ApplyService the controller needs:
// compile the manifest and (unless dry-run) submit it to the engine. Satisfied
// by *domain.ApplyService — the controller reuses the exact path the REST
// handler uses, so it re-implements neither compilation nor execution.
type WorkflowApplier interface {
	ApplyWorkflow(ctx context.Context, req domain.ApplyRequest) (domain.ApplyResult, error)
}

// WorkflowReconciler reconciles a Workflow CR through the existing apply path
// and writes a deliberately thin status. It is the single status writer and so
// runs only on the Lease-elected leader (the manager keeps the default
// NeedLeaderElection=true).
type WorkflowReconciler struct {
	Client  client.Client
	Applier WorkflowApplier
}

// Reconcile compiles and submits the Workflow through domain.ApplyService when
// the spec generation has advanced, then mirrors the outcome into a thin status.
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

	manifest, engineHint, err := buildManifest(u)
	if err != nil {
		// The CRD OpenAPI schema is the structural gate; a build failure here is
		// unexpected. Record it and advance the generation so it does not
		// crash-loop — a corrected spec (new generation) reopens the gate.
		return r.writeStatus(ctx, u, nil, []conditionSpec{
			{conditionCompiled, "False", "BuildFailed", err.Error()},
		})
	}

	result, err := r.Applier.ApplyWorkflow(ctx, domain.ApplyRequest{
		ManifestYAML: manifest,
		Namespace:    u.GetNamespace(),
		EngineHint:   engineHint,
	})
	if err != nil {
		if errors.Is(err, domain.ErrCompilationFailed) {
			// Structural error: surface it as a condition, advance the
			// generation, and do NOT requeue — the user must fix the spec.
			return r.writeStatus(ctx, u, nil, []conditionSpec{
				{conditionCompiled, "False", "CompilationFailed", compileErrorMessage(result.Errors)},
			})
		}
		// Transient (engine unavailable, etc.): requeue with backoff and leave
		// observedGeneration untouched so the next attempt retries.
		return reconcile.Result{}, fmt.Errorf("crd: apply workflow %s: %w", req.NamespacedName, err)
	}

	workflowID := domain.ManifestWorkflowID(manifest)
	slog.Info("workflow reconcile: dispatched",
		"workflow", req.String(), "workflow_id", workflowID, "run_id", result.RunID, "engine", engineHint)
	return r.writeStatus(ctx, u,
		map[string]any{"workflowID": workflowID, "runID": result.RunID, "engine": engineHint},
		[]conditionSpec{
			{conditionCompiled, "True", "Compiled", "manifest compiled to the engine-agnostic IR"},
			{conditionDispatched, "True", "Dispatched", fmt.Sprintf("submitted as run %q", result.RunID)},
		})
}

// needsReconcile is the re-submit gate (ADR-043 §4): reconcile only when the
// spec generation has advanced past the generation the controller last
// observed. status.observedGeneration is absent/0 on a never-reconciled CR, so
// a freshly applied CR (generation >= 1) reconciles exactly once. This is what
// makes a GitOps resync, a controller restart, or a leader change of an
// already-dispatched, spec-unchanged workflow a no-op — no duplicate run.
func needsReconcile(u *unstructured.Unstructured) bool {
	observed, _, _ := unstructured.NestedInt64(u.Object, "status", "observedGeneration")
	return u.GetGeneration() != observed
}

// buildManifest reconstructs the Workflow manifest the compiler expects from the
// CR: metadata.name/namespace supply the manifest metadata, spec supplies the
// state machine. The CR-only fields (engine, version) are lifted out — engine
// becomes the EngineHint, version becomes manifest metadata.version — so what
// reaches the compiler is byte-identical to a `zynax apply` of the same body.
func buildManifest(u *unstructured.Unstructured) (manifest []byte, engineHint string, err error) {
	spec, found, err := unstructured.NestedMap(u.Object, "spec")
	if err != nil || !found {
		return nil, "", fmt.Errorf("workflow %s: missing spec", u.GetName())
	}
	engineHint, _ = spec["engine"].(string)
	version, _ := spec["version"].(string)

	manifestSpec := make(map[string]any, len(spec))
	for k, v := range spec {
		if k == "engine" || k == "version" {
			continue // CR-only fields, not part of the manifest spec
		}
		manifestSpec[k] = v
	}

	metadata := map[string]any{"name": u.GetName()}
	if ns := u.GetNamespace(); ns != "" {
		metadata["namespace"] = ns
	}
	if version != "" {
		metadata["version"] = version
	}

	out, err := yaml.Marshal(map[string]any{
		"apiVersion": manifestAPIVersion,
		"kind":       "Workflow",
		"metadata":   metadata,
		"spec":       manifestSpec,
	})
	if err != nil {
		return nil, "", fmt.Errorf("workflow %s: marshal manifest: %w", u.GetName(), err)
	}
	return out, engineHint, nil
}

// compileErrorMessage renders the first few compile errors into a condition message.
func compileErrorMessage(errs []domain.CompileError) string {
	if len(errs) == 0 {
		return "compilation failed"
	}
	parts := make([]string, 0, len(errs))
	for i, e := range errs {
		if i == 3 {
			parts = append(parts, fmt.Sprintf("(+%d more)", len(errs)-3))
			break
		}
		parts = append(parts, fmt.Sprintf("%s: %s", e.Code, e.Message))
	}
	return strings.Join(parts, "; ")
}

// conditionSpec is a desired status condition before lastTransitionTime is resolved.
type conditionSpec struct{ Type, Status, Reason, Message string }

// writeStatus commits the thin status: observedGeneration (closing the re-submit
// gate), the given extra mirror fields (workflowID/runID/engine — never run
// state), and the conditions. It replaces status wholesale, so a success write
// clears a prior failure's stale fields. lastTransitionTime moves only when a
// condition's status actually flips.
func (r *WorkflowReconciler) writeStatus(ctx context.Context, u *unstructured.Unstructured, extra map[string]any, conds []conditionSpec) (reconcile.Result, error) {
	prev, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions")

	status := map[string]any{"observedGeneration": u.GetGeneration()}
	for k, v := range extra {
		status[k] = v
	}
	status["conditions"] = mergeConditions(prev, conds)

	if err := unstructured.SetNestedMap(u.Object, status, "status"); err != nil {
		return reconcile.Result{}, fmt.Errorf("crd: set workflow status: %w", err)
	}
	if err := r.Client.Status().Update(ctx, u); err != nil {
		return reconcile.Result{}, fmt.Errorf("crd: update workflow status: %w", err)
	}
	return reconcile.Result{}, nil
}

// mergeConditions builds the conditions slice, preserving each condition's prior
// lastTransitionTime when its status is unchanged.
func mergeConditions(prev []any, specs []conditionSpec) []any {
	now := time.Now().UTC().Format(time.RFC3339)
	out := make([]any, 0, len(specs))
	for _, s := range specs {
		transition := now
		for _, p := range prev {
			pm, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if t, _ := pm["type"].(string); t != s.Type {
				continue
			}
			if st, _ := pm["status"].(string); st == s.Status {
				if lt, _ := pm["lastTransitionTime"].(string); lt != "" {
					transition = lt
				}
			}
		}
		out = append(out, map[string]any{
			"type":               s.Type,
			"status":             s.Status,
			"reason":             s.Reason,
			"message":            s.Message,
			"lastTransitionTime": transition,
		})
	}
	return out
}
