// SPDX-License-Identifier: Apache-2.0

package crd

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// workflowCR builds a minimal valid Workflow CR at the given spec generation,
// optionally carrying status.observedGeneration.
func workflowCR(ns, name string, generation, observedGen int64) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(WorkflowGVK)
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetGeneration(generation)
	_ = unstructured.SetNestedMap(u.Object, map[string]any{
		"initial_state": "start",
		"states": map[string]any{
			"start": map[string]any{"type": "terminal"},
		},
	}, "spec")
	if observedGen > 0 {
		_ = unstructured.SetNestedField(u.Object, observedGen, "status", "observedGeneration")
	}
	return u
}

func newReconcilerFixture(objs ...client.Object) *WorkflowReconciler {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(WorkflowGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(WorkflowGVK.GroupVersion().WithKind("WorkflowList"), &unstructured.UnstructuredList{})
	statusObj := &unstructured.Unstructured{}
	statusObj.SetGroupVersionKind(WorkflowGVK)
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(statusObj).
		Build()
	return &WorkflowReconciler{Client: c}
}

func wfReq(ns, name string) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Namespace: ns, Name: name}}
}

func TestNeedsReconcile_Gate(t *testing.T) {
	cases := []struct {
		name        string
		generation  int64
		observedGen int64
		want        bool
	}{
		{"fresh CR, no status", 1, 0, true},
		{"spec advanced past observed", 3, 2, true},
		{"already observed at current generation", 2, 2, false},
		{"observed ahead (should not happen) still equal-gated", 5, 5, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := workflowCR("default", "wf", tc.generation, tc.observedGen)
			if got := needsReconcile(u); got != tc.want {
				t.Fatalf("needsReconcile(gen=%d, observed=%d) = %v, want %v",
					tc.generation, tc.observedGen, got, tc.want)
			}
		})
	}
}

// A CR whose spec has not changed since the last observed generation must be a
// no-op — this is the guarantee that GitOps resync / restart / leader change
// never re-triggers a run.
func TestReconcile_UnchangedSpecIsNoop(t *testing.T) {
	cr := workflowCR("default", "reviewer", 4, 4)
	r := newReconcilerFixture(cr)
	res, err := r.Reconcile(context.Background(), wfReq("default", "reviewer"))
	if err != nil {
		t.Fatalf("reconcile: unexpected error: %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Fatalf("reconcile: expected no requeue for an unchanged CR, got %+v", res)
	}
}

// A freshly applied CR (generation 1, no status) passes the gate and reconciles
// without error.
func TestReconcile_NewCRProceeds(t *testing.T) {
	cr := workflowCR("default", "reviewer", 1, 0)
	r := newReconcilerFixture(cr)
	if _, err := r.Reconcile(context.Background(), wfReq("default", "reviewer")); err != nil {
		t.Fatalf("reconcile: unexpected error: %v", err)
	}
}

// A reconcile for a CR that no longer exists is a clean no-op (no run state to
// unwind — it lives in the engine).
func TestReconcile_DeletedCRIsNoop(t *testing.T) {
	r := newReconcilerFixture() // empty client
	res, err := r.Reconcile(context.Background(), wfReq("default", "gone"))
	if err != nil {
		t.Fatalf("reconcile of missing CR: unexpected error: %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Fatalf("reconcile of missing CR: expected no requeue, got %+v", res)
	}
}

// NewManager must reject an empty namespace: namespaced RBAC forbids a
// cluster-scope watch, so starting one would fail at runtime with a less clear
// error.
func TestNewManager_RejectsEmptyNamespace(t *testing.T) {
	if _, err := NewManager(nil, ""); err == nil {
		t.Fatal("NewManager(\"\"): expected an error for an empty namespace, got nil")
	}
}
