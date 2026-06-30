// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// capturingPublisher records each published event with its payload so tests can
// assert the terminal "completed" event carries the typed outputs payload.
type capturingPublisher struct {
	events   []string
	payloads map[string][]byte
}

func (p *capturingPublisher) Publish(_ context.Context, eventType, _, _ string, payload []byte) error {
	p.events = append(p.events, eventType)
	if p.payloads == nil {
		p.payloads = make(map[string][]byte)
	}
	p.payloads[eventType] = payload
	return nil
}

// searchToTerminal builds a run where "search" produces output "results" and a
// terminal "done" state declares the given outputs map.
func searchToTerminal(workflowID string, outputs map[string]string) *zynaxv1.WorkflowIR {
	done := &zynaxv1.StateIR{
		Id:      "done",
		Type:    zynaxv1.StateType_STATE_TYPE_TERMINAL,
		Outputs: outputs,
	}
	return buildIR(workflowID, "search",
		normal("search",
			[]*zynaxv1.ActionIR{{
				Capability:     "search",
				OutputBindings: map[string]string{"results": "results"},
			}},
			[]*zynaxv1.TransitionIR{transition("search.done", "done", nil)},
		),
		done,
	)
}

func searchExec() *stubExecutor {
	return &stubExecutor{results: map[string]*ActivityResult{
		"search": {EventType: "search.done", Payload: []byte(`{"_event":"search.done","results":"the-answer"}`)},
	}}
}

// TestIRInterpreter_TerminalOutputsResolved proves AC1: declared outputs (a ref
// and a literal) are resolved at the terminal state, returned as the run result,
// and carried on the terminal completed event payload.
func TestIRInterpreter_TerminalOutputsResolved(t *testing.T) {
	ir := searchToTerminal("wf-out", map[string]string{
		"answer": "$.states.search.output.results",
		"label":  "static-literal",
	})
	pub := &capturingPublisher{}
	out, err := (&IRInterpreter{}).Run(context.Background(), ir, searchExec(), pub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["answer"] != "the-answer" || out["label"] != "static-literal" {
		t.Errorf("resolved outputs = %v, want answer=the-answer label=static-literal", out)
	}
	// The completed event payload carries {"outputs": {...}}.
	raw, ok := pub.payloads["zynax.workflow.completed"]
	if !ok || len(raw) == 0 {
		t.Fatalf("expected a payload on the completed event, got: %v", pub.payloads)
	}
	var decoded struct {
		Outputs map[string]string `json:"outputs"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("completed payload is not valid JSON: %v", err)
	}
	if decoded.Outputs["answer"] != "the-answer" {
		t.Errorf("payload outputs = %v, want answer=the-answer", decoded.Outputs)
	}
}

// TestIRInterpreter_NoDeclaredOutputsEmptyMap proves AC2: a COMPLETED run that
// declares no outputs returns a non-nil empty map (no regression, no error).
func TestIRInterpreter_NoDeclaredOutputsEmptyMap(t *testing.T) {
	ir := searchToTerminal("wf-empty", nil)
	out, err := (&IRInterpreter{}).Run(context.Background(), ir, searchExec(), &capturingPublisher{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("expected a non-nil empty map for a run with no declared outputs")
	}
	if len(out) != 0 {
		t.Errorf("expected empty outputs, got %v", out)
	}
}

// TestIRInterpreter_TerminalOutputDanglingRefFails proves AC3: an output that
// references an output no upstream state produced fails the run with a
// DataReferenceError and emits the failed event.
func TestIRInterpreter_TerminalOutputDanglingRefFails(t *testing.T) {
	ir := searchToTerminal("wf-dangling", map[string]string{
		"answer": "$.states.search.output.missing",
	})
	pub := &capturingPublisher{}
	_, err := (&IRInterpreter{}).Run(context.Background(), ir, searchExec(), pub)
	if err == nil {
		t.Fatal("expected run to fail on an unresolved output reference")
	}
	var dre *DataReferenceError
	if !errors.As(err, &dre) {
		t.Fatalf("expected *DataReferenceError, got %T: %v", err, err)
	}
	if !contains(pub.events, "zynax.workflow.failed") {
		t.Errorf("expected a failed event, got: %v", pub.events)
	}
}

// TestIRInterpreter_TerminalOutputSizeBound proves AC4: a captured output that
// exceeds the per-value size bound fails the run with a typed OutputSizeError —
// never silently truncated.
func TestIRInterpreter_TerminalOutputSizeBound(t *testing.T) {
	big := strings.Repeat("x", MaxOutputValueBytes+1)
	ir := searchToTerminal("wf-big", map[string]string{"blob": big})
	pub := &capturingPublisher{}
	_, err := (&IRInterpreter{}).Run(context.Background(), ir, searchExec(), pub)
	if err == nil {
		t.Fatal("expected run to fail on oversized output")
	}
	var se *OutputSizeError
	if !errors.As(err, &se) {
		t.Fatalf("expected *OutputSizeError, got %T: %v", err, err)
	}
	if se.Key != "blob" {
		t.Errorf("OutputSizeError.Key = %q, want blob", se.Key)
	}
	if contains(pub.events, "zynax.workflow.completed") {
		t.Error("a run that overflows the output bound must not emit completed")
	}
}

func TestEnforceOutputBounds(t *testing.T) {
	if err := enforceOutputBounds(nil); err != nil {
		t.Errorf("nil outputs should pass, got %v", err)
	}
	if err := enforceOutputBounds(map[string]string{"a": "ok"}); err != nil {
		t.Errorf("small outputs should pass, got %v", err)
	}
	// Per-value overflow.
	err := enforceOutputBounds(map[string]string{"k": strings.Repeat("x", MaxOutputValueBytes+1)})
	var se *OutputSizeError
	if !errors.As(err, &se) || se.Key != "k" {
		t.Errorf("expected per-value OutputSizeError for k, got %v", err)
	}
	// Total overflow across several in-bound values.
	many := make(map[string]string, 8)
	chunk := strings.Repeat("y", MaxOutputValueBytes-1)
	for i := 0; i < 6; i++ {
		many[string(rune('a'+i))] = chunk
	}
	err = enforceOutputBounds(many)
	if !errors.As(err, &se) || se.Key != "" {
		t.Errorf("expected total-size OutputSizeError, got %v", err)
	}
}

func TestOutputSizeError_Error(t *testing.T) {
	perKey := (&OutputSizeError{Key: "blob", Size: 100, Limit: 64}).Error()
	if !strings.Contains(perKey, "blob") || !strings.Contains(perKey, "per-value") {
		t.Errorf("per-value message unexpected: %q", perKey)
	}
	total := (&OutputSizeError{Key: "", Size: 1000, Limit: 256}).Error()
	if !strings.Contains(total, "total") {
		t.Errorf("total message unexpected: %q", total)
	}
}

func contains(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}
