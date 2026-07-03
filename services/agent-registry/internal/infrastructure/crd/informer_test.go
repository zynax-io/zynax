// SPDX-License-Identifier: Apache-2.0

package crd

import (
	"context"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/zynax-io/zynax/services/agent-registry/internal/domain/scheduler"
)

// agentCR builds an unstructured Agent CR mirroring the CRD schema
// (helm/zynax-agent-registry/crds/agents.zynax.io.yaml).
func agentCR(ns, name string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "zynax.io/v1alpha1",
		"kind":       "Agent",
		"metadata": map[string]any{
			"name":      name,
			"namespace": ns,
			"labels":    map[string]any{"zynax.io/expert-scope": "security-reviewer"},
		},
		"spec": map[string]any{
			"endpointRef": map[string]any{"serviceName": name, "port": int64(50061)},
			"replicas":    int64(2),
			"capabilities": []any{
				map[string]any{
					"id":           "review",
					"description":  "reviews code",
					"inputSchema":  `{"type":"object"}`,
					"outputSchema": `{"type":"object"}`,
					"selectors": map[string]any{
						"language": []any{"go"},
						"tags":     []any{"fast"},
					},
					"cost":      map[string]any{"latencyClass": "low", "tokenPrice": "0.002"},
					"resources": map[string]any{"gpu": "1"},
					"models":    []any{"qwen2.5-coder:3b"},
					"protocols": []any{"A2A", "HTTP"},
				},
			},
		},
		"status": map[string]any{"ready": true, "replicas": int64(2)},
	}}
	return u
}

func TestToCandidate_FullSpec(t *testing.T) {
	got := ToCandidate("default/reviewer", agentCR("default", "reviewer"))
	want := scheduler.Candidate{
		Key:         "default/reviewer",
		Name:        "reviewer",
		Endpoint:    "reviewer.default.svc:50061",
		Ready:       true,
		Replicas:    2,
		ExpertScope: "security-reviewer",
		Capabilities: []scheduler.Capability{{
			ID:           "review",
			Description:  "reviews code",
			InputSchema:  `{"type":"object"}`,
			OutputSchema: `{"type":"object"}`,
			Selectors:    scheduler.Selectors{Language: []string{"go"}, Tags: []string{"fast"}},
			Cost:         scheduler.Cost{TokenPrice: 0.002, LatencyClass: "low"},
			Resources:    scheduler.Resources{GPU: 1},
			Models:       []string{"qwen2.5-coder:3b"},
			Protocols:    []string{"A2A", "HTTP"},
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ToCandidate mismatch:\n got  %+v\n want %+v", got, want)
	}
}

// TestToCandidate_Defensive covers absent status, malformed capability entries,
// and missing optional blocks — decode must skip, never panic.
func TestToCandidate_Defensive(t *testing.T) {
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "zynax.io/v1alpha1",
		"kind":       "Agent",
		"metadata":   map[string]any{"name": "bare", "namespace": "default"},
		"spec": map[string]any{
			"endpointRef": map[string]any{"serviceName": "bare", "port": int64(1)},
			"capabilities": []any{
				"not-a-map", // malformed entry: skipped
				map[string]any{"id": "ping"},
			},
		},
	}}
	got := ToCandidate("default/bare", u)
	if got.Ready {
		t.Error("absent status must decode as not ready")
	}
	if got.ExpertScope != "" {
		t.Errorf("no label => empty expert scope, got %q", got.ExpertScope)
	}
	if len(got.Capabilities) != 1 || got.Capabilities[0].ID != "ping" {
		t.Fatalf("capabilities = %+v, want the single well-formed entry", got.Capabilities)
	}
}

func newFakeReconciler(t *testing.T, objs ...*unstructured.Unstructured) *Reconciler {
	t.Helper()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(AgentGVK, &unstructured.Unstructured{})
	scheme.AddKnownTypeWithName(AgentGVK.GroupVersion().WithKind("AgentList"), &unstructured.UnstructuredList{})
	builder := fake.NewClientBuilder().WithScheme(scheme)
	for _, o := range objs {
		builder = builder.WithObjects(o)
	}
	return &Reconciler{Client: builder.Build(), Index: scheduler.NewIndex()}
}

func TestReconcile_UpsertAndDelete(t *testing.T) {
	r := newFakeReconciler(t, agentCR("default", "reviewer"))
	req := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "reviewer"}}

	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if got := r.Index.Candidates("review"); len(got) != 1 || got[0].Key != "default/reviewer" {
		t.Fatalf("index after upsert = %+v", got)
	}

	// Delete the CR; the next reconcile must drop it from the index.
	obj := agentCR("default", "reviewer")
	if err := r.Client.Delete(context.Background(), obj); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile after delete: %v", err)
	}
	if r.Index.Candidates("review") != nil {
		t.Fatal("index must not return a deleted agent")
	}
	if r.Index.Len() != 0 {
		t.Fatalf("Len = %d, want 0", r.Index.Len())
	}
}

// TestReconcile_ResyncRebuildsIdenticalIndex is the stateless-restart proof at
// the adapter level (ADR-039 §2): a brand-new Reconciler + Index fed the same
// API-server objects converges to the identical projection — no persisted
// state is read or written anywhere in the path.
func TestReconcile_ResyncRebuildsIdenticalIndex(t *testing.T) {
	objs := []*unstructured.Unstructured{
		agentCR("default", "reviewer-a"),
		agentCR("default", "reviewer-b"),
		agentCR("team-x", "compiler"),
	}
	reqs := []reconcile.Request{
		{NamespacedName: types.NamespacedName{Namespace: "default", Name: "reviewer-a"}},
		{NamespacedName: types.NamespacedName{Namespace: "default", Name: "reviewer-b"}},
		{NamespacedName: types.NamespacedName{Namespace: "team-x", Name: "compiler"}},
	}

	first := newFakeReconciler(t, objs...)
	for _, req := range reqs {
		if _, err := first.Reconcile(context.Background(), req); err != nil {
			t.Fatalf("first pass: %v", err)
		}
	}

	// "Restart": fresh reconciler + fresh index over the same stored objects.
	second := newFakeReconciler(t, objs...)
	for _, req := range reqs {
		if _, err := second.Reconcile(context.Background(), req); err != nil {
			t.Fatalf("second pass: %v", err)
		}
	}

	if !reflect.DeepEqual(first.Index.Snapshot(), second.Index.Snapshot()) {
		t.Fatal("restart resync produced a different index projection")
	}
	if first.Index.Len() != 3 {
		t.Fatalf("Len = %d, want 3", first.Index.Len())
	}
}
