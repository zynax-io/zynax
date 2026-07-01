// SPDX-License-Identifier: Apache-2.0

// Package domain contains the core port definitions and value types for the
// engine-adapter service. It has zero imports from api or infrastructure layers.
package domain

import (
	"context"
	"errors"
	"testing"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

func TestWorkflowDataContext_WriteOutputs(t *testing.T) {
	cases := []struct {
		name     string
		stateID  string
		bindings map[string]string
		payload  string
		wantKey  string
		wantVal  string
	}{
		{"string output", "search", map[string]string{"results": "results"}, `{"results":"found-it"}`, "states.search.output.results", "found-it"},
		{"integral number drops trailing zeros", "score", map[string]string{"n": "value"}, `{"value":42}`, "states.score.output.n", "42"},
		{"fractional number keeps decimals", "score", map[string]string{"n": "value"}, `{"value":3.5}`, "states.score.output.n", "3.5"},
		{"bool output", "gate", map[string]string{"ok": "passed"}, `{"passed":true}`, "states.gate.output.ok", "true"},
		{"false bool output", "gate", map[string]string{"ok": "passed"}, `{"passed":false}`, "states.gate.output.ok", "false"},
		{"nested source path", "build", map[string]string{"sha": "meta.sha"}, `{"meta":{"sha":"abc123"}}`, "states.build.output.sha", "abc123"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dc := NewWorkflowDataContext()
			if err := dc.WriteOutputs(DataContextScope{}, tc.stateID, tc.bindings, []byte(tc.payload)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := dc.store[tc.wantKey]; got != tc.wantVal {
				t.Errorf("store[%q] = %q; want %q", tc.wantKey, got, tc.wantVal)
			}
		})
	}
}

func TestWorkflowDataContext_WriteOutputs_Errors(t *testing.T) {
	cases := []struct {
		name     string
		bindings map[string]string
		payload  string
	}{
		{"missing source path", map[string]string{"results": "absent"}, `{"results":"x"}`},
		{"object value is a type mismatch", map[string]string{"results": "obj"}, `{"obj":{"k":"v"}}`},
		{"array value is a type mismatch", map[string]string{"results": "arr"}, `{"arr":[1,2,3]}`},
		{"null value is a type mismatch", map[string]string{"results": "n"}, `{"n":null}`},
		{"non-object payload", map[string]string{"results": "x"}, `not-json`},
		{"traversing a non-object segment", map[string]string{"sha": "meta.sha"}, `{"meta":"flat"}`},
		{"empty source path", map[string]string{"results": ""}, `{"results":"x"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dc := NewWorkflowDataContext()
			err := dc.WriteOutputs(DataContextScope{}, "search", tc.bindings, []byte(tc.payload))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var dre *DataReferenceError
			if !errors.As(err, &dre) {
				t.Fatalf("expected *DataReferenceError, got %T: %v", err, err)
			}
		})
	}
}

func TestWorkflowDataContext_WriteOutputs_NoBindings(t *testing.T) {
	dc := NewWorkflowDataContext()
	if err := dc.WriteOutputs(DataContextScope{}, "s", nil, []byte(`{"x":"y"}`)); err != nil {
		t.Fatalf("nil bindings should be a no-op, got: %v", err)
	}
	if len(dc.store) != 0 {
		t.Errorf("store should be empty, got %d entries", len(dc.store))
	}
}

func TestWorkflowDataContext_WriteOutputs_EmptyPayload(t *testing.T) {
	dc := NewWorkflowDataContext()
	err := dc.WriteOutputs(DataContextScope{}, "s", map[string]string{"k": "v"}, nil)
	if err == nil {
		t.Fatal("expected error extracting from empty payload")
	}
}

func TestWorkflowDataContext_ResolveInputs(t *testing.T) {
	dc := NewWorkflowDataContext()
	if err := dc.WriteOutputs(DataContextScope{}, "search", map[string]string{"results": "results"}, []byte(`{"results":"data"}`)); err != nil {
		t.Fatalf("seed write failed: %v", err)
	}

	cases := []struct {
		name     string
		bindings map[string]string
		wantKey  string
		wantVal  string
		wantErr  bool
	}{
		{
			name:     "reference resolves",
			bindings: map[string]string{"query": "$.states.search.output.results"},
			wantKey:  "query",
			wantVal:  "data",
		},
		{
			name:     "literal passes through",
			bindings: map[string]string{"mode": "fast"},
			wantKey:  "mode",
			wantVal:  "fast",
		},
		{
			name:     "missing reference is an error",
			bindings: map[string]string{"q": "$.states.search.output.absent"},
			wantErr:  true,
		},
		{
			name:     "reference to unknown state is an error",
			bindings: map[string]string{"q": "$.states.nope.output.results"},
			wantErr:  true,
		},
		{
			name:     "malformed reference is an error",
			bindings: map[string]string{"q": "$.states.search.results"},
			wantErr:  true,
		},
		{
			name:     "empty-segment reference is an error",
			bindings: map[string]string{"q": "$.states..output.results"},
			wantErr:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := dc.ResolveInputs(DataContextScope{}, tc.bindings)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (resolved=%v)", got)
				}
				var dre *DataReferenceError
				if !errors.As(err, &dre) {
					t.Fatalf("expected *DataReferenceError, got %T: %v", err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got[tc.wantKey] != tc.wantVal {
				t.Errorf("resolved[%q] = %q; want %q", tc.wantKey, got[tc.wantKey], tc.wantVal)
			}
		})
	}
}

func TestWorkflowDataContext_ResolveInputs_NoBindings(t *testing.T) {
	dc := NewWorkflowDataContext()
	got, err := dc.ResolveInputs(DataContextScope{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil result for nil bindings, got %v", got)
	}
}

func TestDataReferenceError_Error(t *testing.T) {
	e := &DataReferenceError{InputKey: "q", Reference: "$.states.s.output.k", Reason: "not found"}
	want := `engine-adapter: input "q" reference "$.states.s.output.k": not found`
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestMergeInputs(t *testing.T) {
	base := map[string]string{"a": "1", "b": "2"}
	if got := mergeInputs(base, nil); got["a"] != "1" || got["b"] != "2" {
		t.Fatalf("nil inputs should return base unchanged, got %v", got)
	}
	merged := mergeInputs(base, map[string]string{"b": "override", "c": "3"})
	if merged["a"] != "1" || merged["b"] != "override" || merged["c"] != "3" {
		t.Errorf("merge = %v; want a=1 b=override c=3", merged)
	}
	// base must not be mutated.
	if base["b"] != "2" {
		t.Errorf("base was mutated: %v", base)
	}
}

// TestIRInterpreter_DataFlowHappyPath proves an upstream output is stored and a
// downstream state's input binding resolves it (acceptance criterion 1).
func TestIRInterpreter_DataFlowHappyPath(t *testing.T) {
	exec := &stubExecutor{
		results: map[string]*ActivityResult{
			"search": {
				EventType: "search.done",
				Payload:   []byte(`{"_event":"search.done","results":"the-answer"}`),
			},
			"summarize": {EventType: "summarize.done"},
		},
	}
	var captured ActivityInput
	capExec := &captureNamed{inner: exec, name: "summarize", capture: &captured}
	ir := buildIR("wf-dataflow", "search",
		normal("search",
			[]*zynaxv1.ActionIR{{
				Capability:     "search",
				OutputBindings: map[string]string{"results": "results"},
			}},
			[]*zynaxv1.TransitionIR{transition("search.done", "summarize", nil)},
		),
		normal("summarize",
			[]*zynaxv1.ActionIR{{
				Capability:        "summarize",
				InputBindings:     map[string]string{"text": "$.states.search.output.results"},
				InputTemplateJson: `{"in":"{{ .ctx.text }}"}`,
			}},
			[]*zynaxv1.TransitionIR{transition("summarize.done", "done", nil)},
		),
		terminal("done"),
	)
	if _, err := (&IRInterpreter{}).Run(context.Background(), ir, capExec, &stubPublisher{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(captured.InputPayload) != `{"in":"the-answer"}` {
		t.Errorf("downstream input = %s; want {\"in\":\"the-answer\"}", captured.InputPayload)
	}
}

// TestIRInterpreter_DataFlowMissingReferenceFailsRun proves an unresolved input
// reference fails the run with a structured error and emits the failed event
// (acceptance criterion 2).
func TestIRInterpreter_DataFlowMissingReferenceFailsRun(t *testing.T) {
	ir := buildIR("wf-dataflow-missing", "summarize",
		normal("summarize",
			[]*zynaxv1.ActionIR{{
				Capability:    "summarize",
				InputBindings: map[string]string{"text": "$.states.search.output.results"},
			}},
			[]*zynaxv1.TransitionIR{transition("summarize.done", "done", nil)},
		),
		terminal("done"),
	)
	pub := &stubPublisher{}
	_, err := (&IRInterpreter{}).Run(context.Background(), ir, &stubExecutor{}, pub)
	if err == nil {
		t.Fatal("expected run to fail on unresolved reference")
	}
	var dre *DataReferenceError
	if !errors.As(err, &dre) {
		t.Fatalf("expected *DataReferenceError, got %T: %v", err, err)
	}
	found := false
	for _, e := range pub.events {
		if e == "zynax.workflow.failed" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected zynax.workflow.failed event, got: %v", pub.events)
	}
}

// TestIRInterpreter_DataFlowTypeMismatchFailsRun proves a typed-mismatch output
// (a non-scalar value at the source path) fails the run (acceptance criterion 2).
func TestIRInterpreter_DataFlowTypeMismatchFailsRun(t *testing.T) {
	exec := &stubExecutor{
		results: map[string]*ActivityResult{
			"search": {
				EventType: "search.done",
				Payload:   []byte(`{"_event":"search.done","results":{"nested":"obj"}}`),
			},
		},
	}
	ir := buildIR("wf-dataflow-mismatch", "search",
		normal("search",
			[]*zynaxv1.ActionIR{{
				Capability:     "search",
				OutputBindings: map[string]string{"results": "results"},
			}},
			[]*zynaxv1.TransitionIR{transition("search.done", "done", nil)},
		),
		terminal("done"),
	)
	_, err := (&IRInterpreter{}).Run(context.Background(), ir, exec, &stubPublisher{})
	if err == nil {
		t.Fatal("expected run to fail on typed-mismatch output")
	}
	var dre *DataReferenceError
	if !errors.As(err, &dre) {
		t.Fatalf("expected *DataReferenceError, got %T: %v", err, err)
	}
}

// TestDataContextScope_Accessors proves equals/String behave on the value type.
func TestDataContextScope_Accessors(t *testing.T) {
	a := DataContextScope{RunID: "run-1", Namespace: "ns"}
	if !a.equals(DataContextScope{RunID: "run-1", Namespace: "ns"}) {
		t.Error("identical scopes should be equal")
	}
	if a.equals(DataContextScope{RunID: "run-2", Namespace: "ns"}) {
		t.Error("different run ids must not be equal")
	}
	if a.equals(DataContextScope{RunID: "run-1", Namespace: "other"}) {
		t.Error("different namespaces must not be equal")
	}
	if got := a.String(); got != "ns/run-1" {
		t.Errorf("String() = %q; want ns/run-1", got)
	}
}

// TestWorkflowDataContext_Scope proves the constructor binds the owning scope.
func TestWorkflowDataContext_Scope(t *testing.T) {
	owner := DataContextScope{RunID: "run-A", Namespace: "team"}
	dc := NewScopedWorkflowDataContext(owner)
	if got := dc.Scope(); !got.equals(owner) {
		t.Errorf("Scope() = %v; want %v", got, owner)
	}
}

// TestWorkflowDataContext_CrossRunWriteDenied proves a write presenting another
// run's scope is denied with a ScopeError (canvas C.3 acceptance criterion 2).
func TestWorkflowDataContext_CrossRunWriteDenied(t *testing.T) {
	dc := NewScopedWorkflowDataContext(DataContextScope{RunID: "run-A", Namespace: "ns"})
	other := DataContextScope{RunID: "run-B", Namespace: "ns"}
	err := dc.WriteOutputs(other, "search", map[string]string{"results": "results"}, []byte(`{"results":"x"}`))
	var se *ScopeError
	if !errors.As(err, &se) {
		t.Fatalf("expected *ScopeError, got %T: %v", err, err)
	}
	if se.Op != "write" {
		t.Errorf("ScopeError.Op = %q; want write", se.Op)
	}
	if len(dc.store) != 0 {
		t.Errorf("denied write must not mutate store, got %d entries", len(dc.store))
	}
}

// TestWorkflowDataContext_CrossRunReadDenied proves a read presenting another
// run's scope is denied even when the key exists (canvas C.3, criterion 2).
func TestWorkflowDataContext_CrossRunReadDenied(t *testing.T) {
	owner := DataContextScope{RunID: "run-A", Namespace: "ns"}
	dc := NewScopedWorkflowDataContext(owner)
	if err := dc.WriteOutputs(owner, "search", map[string]string{"results": "results"}, []byte(`{"results":"secret"}`)); err != nil {
		t.Fatalf("seed write failed: %v", err)
	}
	// A different run that knows the exact key still cannot read it.
	other := DataContextScope{RunID: "run-B", Namespace: "ns"}
	got, err := dc.ResolveInputs(other, map[string]string{"q": "$.states.search.output.results"})
	var se *ScopeError
	if !errors.As(err, &se) {
		t.Fatalf("expected *ScopeError, got %T: %v (resolved=%v)", err, err, got)
	}
	if se.Op != "read" {
		t.Errorf("ScopeError.Op = %q; want read", se.Op)
	}
}

// TestWorkflowDataContext_CrossNamespaceDenied proves the same run id in a
// different namespace is a distinct scope and is denied (ADR-008).
func TestWorkflowDataContext_CrossNamespaceDenied(t *testing.T) {
	dc := NewScopedWorkflowDataContext(DataContextScope{RunID: "run-A", Namespace: "ns-1"})
	other := DataContextScope{RunID: "run-A", Namespace: "ns-2"}
	err := dc.WriteOutputs(other, "search", map[string]string{"results": "results"}, []byte(`{"results":"x"}`))
	var se *ScopeError
	if !errors.As(err, &se) {
		t.Fatalf("expected *ScopeError across namespaces, got %T: %v", err, err)
	}
}

// TestWorkflowDataContext_SameScopeAllowed proves the owning run reads/writes
// its own data context normally (canvas C.3 acceptance criterion 1).
func TestWorkflowDataContext_SameScopeAllowed(t *testing.T) {
	owner := DataContextScope{RunID: "run-A", Namespace: "ns"}
	dc := NewScopedWorkflowDataContext(owner)
	if err := dc.WriteOutputs(owner, "search", map[string]string{"results": "results"}, []byte(`{"results":"data"}`)); err != nil {
		t.Fatalf("same-scope write failed: %v", err)
	}
	got, err := dc.ResolveInputs(owner, map[string]string{"q": "$.states.search.output.results"})
	if err != nil {
		t.Fatalf("same-scope read failed: %v", err)
	}
	if got["q"] != "data" {
		t.Errorf("resolved[q] = %q; want data", got["q"])
	}
}

// TestScopeError_Error proves the diagnostic message names both scopes and op.
func TestScopeError_Error(t *testing.T) {
	e := &ScopeError{
		Owner:     DataContextScope{RunID: "run-A", Namespace: "ns"},
		Requester: DataContextScope{RunID: "run-B", Namespace: "ns"},
		Op:        "read",
	}
	want := "engine-adapter: cross-run data-context read denied: context owned by ns/run-A, requested by ns/run-B"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

// captureNamed wraps an executor and records the ActivityInput for a named capability.
type captureNamed struct {
	inner   *stubExecutor
	name    string
	capture *ActivityInput
}

func (c *captureNamed) DispatchCapability(ctx context.Context, in ActivityInput) (*ActivityResult, error) {
	if in.CapabilityName == c.name {
		*c.capture = in
	}
	return c.inner.DispatchCapability(ctx, in)
}
