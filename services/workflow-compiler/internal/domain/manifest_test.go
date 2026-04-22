package domain

import (
	"fmt"
	"testing"
	"time"
)

// minimalValid returns a minimal valid workflow YAML for use in tests.
func minimalValid() []byte {
	return []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: test-wf
spec:
  initial_state: start
  states:
    start:
      type: terminal
`)
}

// fullValid returns a multi-state workflow YAML covering all field types.
func fullValid() []byte {
	return []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: full-wf
  namespace: platform
  labels:
    team: core
  annotations:
    description: "full example"
spec:
  initial_state: work
  states:
    work:
      type: normal
      actions:
        - capability: summarize
          timeout: 30s
          input:
            text: "{{ .ctx.doc }}"
          output:
            ctx.summary: "{{ .result.summary }}"
        - capability: notify
          async: true
      on:
        - event: work.done
          goto: review
          guard: "{{ .ctx.ok }}"
          set:
            ctx.count: 1
        - event: work.failed
          goto: done
    review:
      type: human_in_the_loop
      on:
        - event: human.approved
          goto: done
        - event: human.rejected
          goto: work
    done:
      type: terminal
`)
}

func TestParseManifest_MinimalValid(t *testing.T) {
	m, errs := ParseManifest(minimalValid())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if m.Name != "test-wf" {
		t.Errorf("name = %q, want %q", m.Name, "test-wf")
	}
	if m.Namespace != "default" {
		t.Errorf("namespace = %q, want %q", m.Namespace, "default")
	}
	if m.InitialState != "start" {
		t.Errorf("initial_state = %q, want %q", m.InitialState, "start")
	}
	if len(m.States) != 1 {
		t.Errorf("len(states) = %d, want 1", len(m.States))
	}
	st := m.States["start"]
	if st == nil || st.Type != StateTypeTerminal {
		t.Errorf("start state type = %v, want terminal", st)
	}
}

func TestParseManifest_FullValid(t *testing.T) {
	m, errs := ParseManifest(fullValid())
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if m.Namespace != "platform" {
		t.Errorf("namespace = %q, want platform", m.Namespace)
	}
	if m.Labels["team"] != "core" {
		t.Errorf("labels[team] = %q, want core", m.Labels["team"])
	}
	if len(m.States) != 3 {
		t.Errorf("len(states) = %d, want 3", len(m.States))
	}

	work := m.States["work"]
	if work.Type != StateTypeNormal {
		t.Errorf("work.type = %v, want normal", work.Type)
	}
	if len(work.Actions) != 2 {
		t.Errorf("work.actions len = %d, want 2", len(work.Actions))
	}
	if work.Actions[0].Capability != "summarize" {
		t.Errorf("action[0].capability = %q, want summarize", work.Actions[0].Capability)
	}
	if work.Actions[0].Timeout != 30*time.Second {
		t.Errorf("action[0].timeout = %v, want 30s", work.Actions[0].Timeout)
	}
	if !work.Actions[1].Async {
		t.Errorf("action[1].async = false, want true")
	}
	if len(work.Transitions) != 2 {
		t.Errorf("work.transitions len = %d, want 2", len(work.Transitions))
	}
	if work.Transitions[0].Guard == "" {
		t.Errorf("transition[0].guard should be non-empty")
	}
	if work.Transitions[0].Set == nil {
		t.Errorf("transition[0].set should be non-nil")
	}

	review := m.States["review"]
	if review.Type != StateTypeHumanInTheLoop {
		t.Errorf("review.type = %v, want human_in_the_loop", review.Type)
	}

	done := m.States["done"]
	if done.Type != StateTypeTerminal {
		t.Errorf("done.type = %v, want terminal", done.Type)
	}
}

func TestParseManifest_InvalidYAML(t *testing.T) {
	bad := []byte("key: [unterminated")
	_, errs := ParseManifest(bad)
	if len(errs) == 0 {
		t.Fatal("expected parse errors for invalid YAML")
	}
	if errs[0].Code != ErrorCodeYAMLParseError {
		t.Errorf("code = %v, want ErrorCodeYAMLParseError", errs[0].Code)
	}
}

func TestParseManifest_Empty(t *testing.T) {
	_, errs := ParseManifest([]byte(""))
	if len(errs) == 0 {
		t.Fatal("expected error for empty manifest")
	}
	if errs[0].Code != ErrorCodeMissingRequiredField {
		t.Errorf("code = %v, want ErrorCodeMissingRequiredField", errs[0].Code)
	}
}

func TestParseManifest_WrongKind(t *testing.T) {
	data := []byte(`
kind: AgentDef
apiVersion: zynax.io/v1
metadata:
  name: test
spec:
  initial_state: s
  states:
    s:
      type: terminal
`)
	_, errs := ParseManifest(data)
	if len(errs) == 0 {
		t.Fatal("expected error for wrong kind")
	}
	hasKindErr := false
	for _, e := range errs {
		if e.Code == ErrorCodeMissingRequiredField {
			hasKindErr = true
		}
	}
	if !hasKindErr {
		t.Errorf("expected MissingRequiredField error, got %v", errs)
	}
}

func TestParseManifest_MissingName(t *testing.T) {
	data := []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  namespace: ns
spec:
  initial_state: s
  states:
    s:
      type: terminal
`)
	_, errs := ParseManifest(data)
	if len(errs) == 0 {
		t.Fatal("expected error for missing name")
	}
	if errs[0].Code != ErrorCodeMissingRequiredField {
		t.Errorf("code = %v, want ErrorCodeMissingRequiredField", errs[0].Code)
	}
}

func TestParseManifest_MissingInitialState(t *testing.T) {
	data := []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: wf
spec:
  states:
    s:
      type: terminal
`)
	_, errs := ParseManifest(data)
	if len(errs) == 0 {
		t.Fatal("expected error for missing initial_state")
	}
	hasCode := false
	for _, e := range errs {
		if e.Code == ErrorCodeNoInitialState {
			hasCode = true
		}
	}
	if !hasCode {
		t.Errorf("expected ErrorCodeNoInitialState, got %v", errs)
	}
}

func TestParseManifest_NoStates(t *testing.T) {
	data := []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: wf
spec:
  initial_state: s
`)
	_, errs := ParseManifest(data)
	if len(errs) == 0 {
		t.Fatal("expected error for empty states")
	}
}

func TestParseManifest_InvalidStateType(t *testing.T) {
	data := []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: wf
spec:
  initial_state: s
  states:
    s:
      type: bogus_type
`)
	_, errs := ParseManifest(data)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid state type")
	}
	if errs[0].Code != ErrorCodeInvalidFieldValue {
		t.Errorf("code = %v, want ErrorCodeInvalidFieldValue", errs[0].Code)
	}
}

func TestParseManifest_MissingActionCapability(t *testing.T) {
	data := []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: wf
spec:
  initial_state: s
  states:
    s:
      type: terminal
      actions:
        - timeout: 5s
`)
	_, errs := ParseManifest(data)
	if len(errs) == 0 {
		t.Fatal("expected error for missing action capability")
	}
	if errs[0].Code != ErrorCodeMissingRequiredField {
		t.Errorf("code = %v, want ErrorCodeMissingRequiredField", errs[0].Code)
	}
}

func TestParseManifest_InvalidTimeout(t *testing.T) {
	data := []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: wf
spec:
  initial_state: s
  states:
    s:
      type: terminal
      actions:
        - capability: ping
          timeout: "not-a-duration"
`)
	_, errs := ParseManifest(data)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid timeout")
	}
	if errs[0].Code != ErrorCodeInvalidFieldValue {
		t.Errorf("code = %v, want ErrorCodeInvalidFieldValue", errs[0].Code)
	}
}

func TestParseManifest_MissingTransitionEvent(t *testing.T) {
	data := []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: wf
spec:
  initial_state: s
  states:
    s:
      type: normal
      on:
        - goto: done
    done:
      type: terminal
`)
	_, errs := ParseManifest(data)
	if len(errs) == 0 {
		t.Fatal("expected error for missing transition event")
	}
	if errs[0].Code != ErrorCodeMissingRequiredField {
		t.Errorf("code = %v, want ErrorCodeMissingRequiredField", errs[0].Code)
	}
}

func TestParseManifest_MissingTransitionGoto(t *testing.T) {
	data := []byte(`
kind: Workflow
apiVersion: zynax.io/v1
metadata:
  name: wf
spec:
  initial_state: s
  states:
    s:
      type: normal
      on:
        - event: something.happened
    done:
      type: terminal
`)
	_, errs := ParseManifest(data)
	if len(errs) == 0 {
		t.Fatal("expected error for missing transition goto")
	}
	if errs[0].Code != ErrorCodeMissingRequiredField {
		t.Errorf("code = %v, want ErrorCodeMissingRequiredField", errs[0].Code)
	}
}

func TestParseManifest_DefaultNamespace(t *testing.T) {
	m, errs := ParseManifest(minimalValid())
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if m.Namespace != "default" {
		t.Errorf("namespace = %q, want default", m.Namespace)
	}
}

func TestParseManifest_StateLineNumbers(t *testing.T) {
	m, errs := ParseManifest(fullValid())
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	for name, st := range m.States {
		if st.Line == 0 {
			t.Errorf("state %q has line number 0, expected non-zero", name)
		}
	}
}

func TestParseError_ErrorWithLine(t *testing.T) {
	e := ParseError{Code: ErrorCodeYAMLParseError, Message: "syntax error", Line: 5}
	if e.Error() != "line 5: syntax error" {
		t.Errorf("Error() = %q", e.Error())
	}
}

func TestParseError_ErrorWithoutLine(t *testing.T) {
	e := ParseError{Code: ErrorCodeNoInitialState, Message: "no initial state"}
	if e.Error() != "no initial state" {
		t.Errorf("Error() = %q", e.Error())
	}
}

func TestParseErrors_Error(t *testing.T) {
	pe := ParseErrors{{Code: ErrorCodeYAMLParseError, Message: "first"}}
	if pe.Error() != "first" {
		t.Errorf("ParseErrors.Error() = %q", pe.Error())
	}
}

func TestParseErrors_ErrorEmpty(t *testing.T) {
	var pe ParseErrors
	if pe.Error() != "no errors" {
		t.Errorf("empty ParseErrors.Error() = %q", pe.Error())
	}
}

func TestExtractYAMLErrorLine_NoLine(t *testing.T) {
	// Error without a line number in the message
	n := extractYAMLErrorLine(fmt.Errorf("some error without line info"))
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestExtractYAMLErrorLine_Nil(t *testing.T) {
	if extractYAMLErrorLine(nil) != 0 {
		t.Error("expected 0 for nil error")
	}
}
