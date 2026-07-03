// SPDX-License-Identifier: Apache-2.0

package crd

import (
	"context"
	"errors"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/zynax-io/zynax/services/api-gateway/internal/domain"
)

// fakeApplier records ApplyWorkflow calls and returns a canned result/error —
// standing in for *domain.ApplyService so the reconciler is tested without a
// compiler or engine.
type fakeApplier struct {
	calls   int
	lastReq domain.ApplyRequest
	result  domain.ApplyResult
	err     error
}

func (f *fakeApplier) ApplyWorkflow(_ context.Context, req domain.ApplyRequest) (domain.ApplyResult, error) {
	f.calls++
	f.lastReq = req
	return f.result, f.err
}

// workflowCR builds a minimal valid Workflow CR at the given spec generation,
// optionally carrying status.observedGeneration.
func workflowCR(ns, name string, generation, observedGen int64) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(WorkflowGVK)
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetGeneration(generation)
	_ = unstructured.SetNestedMap(u.Object, map[string]any{
		"engine":        "temporal",
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

func newReconcilerFixture(applier WorkflowApplier, objs ...client.Object) *WorkflowReconciler {
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
	return &WorkflowReconciler{Client: c, Applier: applier}
}

func wfReq(ns, name string) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Namespace: ns, Name: name}}
}

// readStatus re-fetches the CR and returns its status map.
func readStatus(t *testing.T, r *WorkflowReconciler, ns, name string) map[string]any {
	t.Helper()
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(WorkflowGVK)
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: name}, u); err != nil {
		t.Fatalf("re-get %s/%s: %v", ns, name, err)
	}
	st, _, _ := unstructured.NestedMap(u.Object, "status")
	return st
}

// condition finds a status condition by type.
func condition(st map[string]any, typ string) map[string]any {
	conds, _, _ := unstructured.NestedSlice(st, "conditions")
	for _, c := range conds {
		if cm, ok := c.(map[string]any); ok {
			if t, _ := cm["type"].(string); t == typ {
				return cm
			}
		}
	}
	return nil
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

// A new CR is compiled+submitted once, and the outcome is mirrored into a thin
// status that carries NO run state.
func TestReconcile_NewCR_SubmitsAndWritesThinStatus(t *testing.T) {
	applier := &fakeApplier{result: domain.ApplyResult{RunID: "run-abc", Status: "new"}}
	cr := workflowCR("default", "reviewer", 1, 0)
	r := newReconcilerFixture(applier, cr)

	if _, err := r.Reconcile(context.Background(), wfReq("default", "reviewer")); err != nil {
		t.Fatalf("reconcile: unexpected error: %v", err)
	}
	if applier.calls != 1 {
		t.Fatalf("ApplyWorkflow calls = %d, want 1", applier.calls)
	}
	assertManifestAndHint(t, applier.lastReq)

	st := readStatus(t, r, "default", "reviewer")
	assertDispatchedStatus(t, st)
	assertNoRunState(t, st)
}

// assertManifestAndHint checks that the compiler received a v1 Workflow manifest
// with the engine lifted out to the hint (byte-identical to a `zynax apply`).
func assertManifestAndHint(t *testing.T, req domain.ApplyRequest) {
	t.Helper()
	man := string(req.ManifestYAML)
	for _, want := range []string{"apiVersion: zynax.io/v1", "kind: Workflow", "initial_state: start"} {
		if !strings.Contains(man, want) {
			t.Fatalf("manifest missing %q:\n%s", want, man)
		}
	}
	if strings.Contains(man, "engine:") {
		t.Fatalf("manifest must not carry the CR-only engine field:\n%s", man)
	}
	if req.EngineHint != "temporal" || req.Namespace != "default" {
		t.Fatalf("apply request = %+v, want engine=temporal namespace=default", req)
	}
}

// assertDispatchedStatus checks the thin status fields and conditions of a
// successfully dispatched Workflow.
func assertDispatchedStatus(t *testing.T, st map[string]any) {
	t.Helper()
	if og, _, _ := unstructured.NestedInt64(st, "observedGeneration"); og != 1 {
		t.Fatalf("observedGeneration = %d, want 1", og)
	}
	if run, _, _ := unstructured.NestedString(st, "runID"); run != "run-abc" {
		t.Fatalf("runID = %q, want run-abc", run)
	}
	if wid, _, _ := unstructured.NestedString(st, "workflowID"); !strings.HasPrefix(wid, "wf-") {
		t.Fatalf("workflowID = %q, want a wf- prefixed id", wid)
	}
	if eng, _, _ := unstructured.NestedString(st, "engine"); eng != "temporal" {
		t.Fatalf("engine = %q, want temporal", eng)
	}
	if c := condition(st, conditionCompiled); c == nil || c["status"] != "True" {
		t.Fatalf("Compiled condition = %v, want status True", c)
	}
	if c := condition(st, conditionDispatched); c == nil || c["status"] != "True" {
		t.Fatalf("Dispatched condition = %v, want status True", c)
	}
}

// assertNoRunState enforces the thin-status contract: only the mirror keys may
// appear — run state must stay in the engine.
func assertNoRunState(t *testing.T, st map[string]any) {
	t.Helper()
	allowed := map[string]bool{"observedGeneration": true, "workflowID": true, "runID": true, "engine": true, "conditions": true}
	for k := range st {
		if !allowed[k] {
			t.Fatalf("status carries a non-thin key %q (run state must stay in the engine): %v", k, st)
		}
	}
}

// A completed, spec-unchanged CR is not re-submitted on a subsequent reconcile
// (GitOps resync).
func TestReconcile_NoResubmitAfterDispatch(t *testing.T) {
	applier := &fakeApplier{result: domain.ApplyResult{RunID: "run-1"}}
	cr := workflowCR("default", "reviewer", 1, 0)
	r := newReconcilerFixture(applier, cr)

	if _, err := r.Reconcile(context.Background(), wfReq("default", "reviewer")); err != nil {
		t.Fatalf("first reconcile: %v", err)
	}
	if _, err := r.Reconcile(context.Background(), wfReq("default", "reviewer")); err != nil {
		t.Fatalf("second reconcile: %v", err)
	}
	if applier.calls != 1 {
		t.Fatalf("ApplyWorkflow calls = %d after two reconciles, want 1 (gate must hold)", applier.calls)
	}
}

// A controller restart re-Lists the CR with its persisted status; an
// already-dispatched, spec-unchanged CR (observedGeneration == generation) must
// not be re-submitted.
func TestReconcile_RestartNoResubmit(t *testing.T) {
	applier := &fakeApplier{result: domain.ApplyResult{RunID: "run-1"}}
	dispatched := workflowCR("default", "reviewer", 2, 2) // status already caught up
	r := newReconcilerFixture(applier, dispatched)

	if _, err := r.Reconcile(context.Background(), wfReq("default", "reviewer")); err != nil {
		t.Fatalf("reconcile after restart: %v", err)
	}
	if applier.calls != 0 {
		t.Fatalf("ApplyWorkflow calls = %d after restart, want 0", applier.calls)
	}
}

// A compilation failure is surfaced as a Compiled=False condition, advances the
// generation (so it does not crash-loop), and returns no error.
func TestReconcile_CompileError_ConditionNoCrash(t *testing.T) {
	applier := &fakeApplier{
		result: domain.ApplyResult{Errors: []domain.CompileError{{Code: "E_STATE", Message: "unknown state ref"}}},
		err:    domain.ErrCompilationFailed,
	}
	cr := workflowCR("default", "broken", 1, 0)
	r := newReconcilerFixture(applier, cr)

	if _, err := r.Reconcile(context.Background(), wfReq("default", "broken")); err != nil {
		t.Fatalf("reconcile of a bad manifest must not error (no crash-loop): %v", err)
	}
	st := readStatus(t, r, "default", "broken")
	if og, _, _ := unstructured.NestedInt64(st, "observedGeneration"); og != 1 {
		t.Fatalf("observedGeneration = %d, want 1 (gate must close on structural error)", og)
	}
	c := condition(st, conditionCompiled)
	if c == nil || c["status"] != "False" {
		t.Fatalf("Compiled condition = %v, want status False", c)
	}
	if msg, _ := c["message"].(string); !strings.Contains(msg, "unknown state ref") {
		t.Fatalf("Compiled message = %q, want the compile error", msg)
	}
	if _, ok := st["runID"]; ok {
		t.Fatalf("a failed compile must not write a runID: %v", st)
	}
}

// A transient (non-compilation) error requeues and leaves observedGeneration
// untouched so the next attempt retries.
func TestReconcile_TransientError_Requeues(t *testing.T) {
	applier := &fakeApplier{err: errors.New("engine adapter unavailable")}
	cr := workflowCR("default", "reviewer", 1, 0)
	r := newReconcilerFixture(applier, cr)

	_, err := r.Reconcile(context.Background(), wfReq("default", "reviewer"))
	if err == nil {
		t.Fatal("reconcile: expected a transient error to be returned (for requeue), got nil")
	}
	st := readStatus(t, r, "default", "reviewer")
	if og, found, _ := unstructured.NestedInt64(st, "observedGeneration"); found && og != 0 {
		t.Fatalf("observedGeneration = %d after a transient error, want unset (gate stays open)", og)
	}
}

// A reconcile for a CR that no longer exists is a clean no-op (no run state to
// unwind — it lives in the engine).
func TestReconcile_DeletedCRIsNoop(t *testing.T) {
	applier := &fakeApplier{}
	r := newReconcilerFixture(applier) // empty client
	res, err := r.Reconcile(context.Background(), wfReq("default", "gone"))
	if err != nil {
		t.Fatalf("reconcile of missing CR: unexpected error: %v", err)
	}
	if res.RequeueAfter != 0 || applier.calls != 0 {
		t.Fatalf("reconcile of missing CR: expected a pure no-op, got res=%+v calls=%d", res, applier.calls)
	}
}

// NewManager must reject an empty namespace: namespaced RBAC forbids a
// cluster-scope watch, so starting one would fail at runtime with a less clear
// error.
func TestNewManager_RejectsEmptyNamespace(t *testing.T) {
	if _, err := NewManager(nil, &fakeApplier{}, ""); err == nil {
		t.Fatal("NewManager(\"\"): expected an error for an empty namespace, got nil")
	}
}
