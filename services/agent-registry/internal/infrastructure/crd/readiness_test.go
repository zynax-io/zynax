// SPDX-License-Identifier: Apache-2.0

package crd

import (
	"context"
	"sync/atomic"
	"testing"

	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// slice builds an EndpointSlice for svc with the given endpoint readiness flags.
func slice(ns, name, svc string, readyFlags ...bool) *discoveryv1.EndpointSlice {
	eps := make([]discoveryv1.Endpoint, 0, len(readyFlags))
	for _, rf := range readyFlags {
		eps = append(eps, discoveryv1.Endpoint{Conditions: discoveryv1.EndpointConditions{Ready: ptr.To(rf)}})
	}
	return &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: ns,
			Labels: map[string]string{serviceNameLabel: svc},
		},
		Endpoints: eps,
	}
}

// newReadinessFixture builds a fake client with the Agent CRD scheme, status
// subresource support, and an update counter for the churn-guard assertions.
func newReadinessFixture(t *testing.T, updates *atomic.Int32, objs ...client.Object) *ReadinessReconciler {
	t.Helper()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(AgentGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(AgentGVK.GroupVersion().WithKind("AgentList"), &unstructured.UnstructuredList{})
	if err := discoveryv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add discovery scheme: %v", err)
	}
	statusObj := &unstructured.Unstructured{}
	statusObj.SetGroupVersionKind(AgentGVK)
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(statusObj).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, cl client.Client, sub string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				updates.Add(1)
				return cl.SubResource(sub).Update(ctx, obj, opts...)
			},
		}).
		Build()
	return &ReadinessReconciler{Client: c}
}

func agentReq(ns, name string) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Namespace: ns, Name: name}}
}

func TestReadiness_ServingEndpointsFlipReady(t *testing.T) {
	var updates atomic.Int32
	fresh := agentCR("default", "reviewer")
	unstructured.RemoveNestedField(fresh.Object, "status") // freshly applied CR: no status yet
	r := newReadinessFixture(t, &updates,
		fresh,
		slice("default", "reviewer-abc", "reviewer", true, true, false),
	)

	if _, err := r.Reconcile(context.Background(), agentReq("default", "reviewer")); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	got := &unstructured.Unstructured{}
	got.SetGroupVersionKind(AgentGVK)
	if err := r.Client.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "reviewer"}, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	ready, _, _ := unstructured.NestedBool(got.Object, "status", "ready")
	replicas, _, _ := unstructured.NestedInt64(got.Object, "status", "replicas")
	if !ready || replicas != 2 {
		t.Fatalf("status = ready=%v replicas=%d, want true/2 (two serving endpoints)", ready, replicas)
	}
	conds, _, _ := unstructured.NestedSlice(got.Object, "status", "conditions")
	if len(conds) != 1 {
		t.Fatalf("conditions = %v", conds)
	}
	cond := conds[0].(map[string]any)
	if cond["type"] != conditionReady || cond["status"] != "True" || cond["reason"] != reasonServing {
		t.Errorf("condition = %+v", cond)
	}
}

// TestReadiness_NoServingEndpoints covers the stale-liveness fix: a crashed
// agent (no serving endpoints) is authoritatively not ready.
func TestReadiness_NoServingEndpoints(t *testing.T) {
	var updates atomic.Int32
	cases := map[string]client.Object{
		"slice with only unready endpoints": slice("default", "reviewer-abc", "reviewer", false),
		"no slice at all":                   slice("default", "other-abc", "other-svc", true),
	}
	for name, extra := range cases {
		t.Run(name, func(t *testing.T) {
			r := newReadinessFixture(t, &updates, agentCR("default", "reviewer"), extra)
			if _, err := r.Reconcile(context.Background(), agentReq("default", "reviewer")); err != nil {
				t.Fatalf("reconcile: %v", err)
			}
			got := &unstructured.Unstructured{}
			got.SetGroupVersionKind(AgentGVK)
			_ = r.Client.Get(context.Background(), types.NamespacedName{Namespace: "default", Name: "reviewer"}, got)
			ready, _, _ := unstructured.NestedBool(got.Object, "status", "ready")
			if ready {
				t.Fatal("agent without serving endpoints must be not-ready")
			}
		})
	}
}

// TestReadiness_SteadyStateWritesNothing is the low-churn guard (ADR-039 §3):
// repeated reconciles of an unchanged world must not touch etcd.
func TestReadiness_SteadyStateWritesNothing(t *testing.T) {
	var updates atomic.Int32
	r := newReadinessFixture(t, &updates,
		agentCR("default", "reviewer"),
		slice("default", "reviewer-abc", "reviewer", true),
	)

	for range 3 {
		if _, err := r.Reconcile(context.Background(), agentReq("default", "reviewer")); err != nil {
			t.Fatalf("reconcile: %v", err)
		}
	}
	if n := updates.Load(); n != 1 {
		t.Fatalf("status updates = %d, want exactly 1 (first write, then steady state)", n)
	}
}

func TestReadiness_DeletedAgentIsNoop(t *testing.T) {
	var updates atomic.Int32
	r := newReadinessFixture(t, &updates)
	if _, err := r.Reconcile(context.Background(), agentReq("default", "ghost")); err != nil {
		t.Fatalf("reconcile of deleted agent: %v", err)
	}
	if updates.Load() != 0 {
		t.Fatal("deleted agent must not trigger a status write")
	}
}

// TestMapSliceToAgents resolves slice events to the owning Agent(s) only.
func TestMapSliceToAgents(t *testing.T) {
	var updates atomic.Int32
	r := newReadinessFixture(t, &updates,
		agentCR("default", "reviewer"),  // endpointRef.serviceName == "reviewer"
		agentCR("default", "other"),     // endpointRef.serviceName == "other"
		agentCR("team-x", "reviewer-x"), // different namespace
	)
	mapFn := mapSliceToAgents(r.Client)

	reqs := mapFn(context.Background(), slice("default", "reviewer-abc", "reviewer", true))
	if len(reqs) != 1 || reqs[0].Name != "reviewer" || reqs[0].Namespace != "default" {
		t.Fatalf("requests = %+v, want exactly default/reviewer", reqs)
	}

	// A slice without the service-name label maps to nothing.
	anon := slice("default", "anon", "", true)
	delete(anon.Labels, serviceNameLabel)
	if got := mapFn(context.Background(), anon); got != nil {
		t.Fatalf("unlabeled slice mapped to %+v, want nil", got)
	}
}
