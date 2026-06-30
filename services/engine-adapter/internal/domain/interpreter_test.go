// SPDX-License-Identifier: Apache-2.0

// Package domain contains the core port definitions and value types for the
// engine-adapter service. It has zero imports from api or infrastructure layers.
package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// stubExecutor implements ActivityExecutor for tests.
type stubExecutor struct {
	results map[string]*ActivityResult
	err     error
}

func (s *stubExecutor) DispatchCapability(_ context.Context, in ActivityInput) (*ActivityResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if r, ok := s.results[in.CapabilityName]; ok {
		return r, nil
	}
	return &ActivityResult{EventType: in.CapabilityName + ".completed"}, nil
}

// stubPublisher implements EventPublisher for tests; records events in order.
type stubPublisher struct {
	events []string
	err    error
}

func (p *stubPublisher) Publish(_ context.Context, eventType, _, _ string, _ []byte) error {
	p.events = append(p.events, eventType)
	return p.err
}

// buildIR constructs a minimal WorkflowIR for tests.
func buildIR(workflowID, initial string, states ...*zynaxv1.StateIR) *zynaxv1.WorkflowIR {
	return &zynaxv1.WorkflowIR{
		WorkflowId:   workflowID,
		InitialState: initial,
		States:       states,
	}
}

func terminal(id string) *zynaxv1.StateIR {
	return &zynaxv1.StateIR{Id: id, Type: zynaxv1.StateType_STATE_TYPE_TERMINAL}
}

func normal(id string, actions []*zynaxv1.ActionIR, transitions []*zynaxv1.TransitionIR) *zynaxv1.StateIR {
	return &zynaxv1.StateIR{
		Id:          id,
		Type:        zynaxv1.StateType_STATE_TYPE_NORMAL,
		Actions:     actions,
		Transitions: transitions,
	}
}

func action(capability string) *zynaxv1.ActionIR {
	return &zynaxv1.ActionIR{Capability: capability}
}

func transition(eventType, target string, conditions map[string]string) *zynaxv1.TransitionIR {
	return &zynaxv1.TransitionIR{
		EventType:   eventType,
		TargetState: target,
		Conditions:  conditions,
	}
}

func TestIRInterpreter_TerminalInitialState(t *testing.T) {
	ir := buildIR("wf-1", "done", terminal("done"))
	pub := &stubPublisher{}
	_, err := (&IRInterpreter{}).Run(context.Background(), ir, &stubExecutor{}, pub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pub.events) != 1 || pub.events[0] != "zynax.workflow.completed" {
		t.Errorf("events = %v; want [zynax.workflow.completed]", pub.events)
	}
}

func TestIRInterpreter_TwoStateWorkflow(t *testing.T) {
	ir := buildIR("wf-2", "review",
		normal("review",
			[]*zynaxv1.ActionIR{action("summarize")},
			[]*zynaxv1.TransitionIR{transition("summarize.completed", "done", nil)},
		),
		terminal("done"),
	)
	pub := &stubPublisher{}
	_, err := (&IRInterpreter{}).Run(context.Background(), ir, &stubExecutor{}, pub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{
		"zynax.workflow.state.entered",
		"zynax.workflow.state.exited",
		"zynax.workflow.completed",
	}
	if !equalSlice(pub.events, want) {
		t.Errorf("events = %v; want %v", pub.events, want)
	}
}

func TestIRInterpreter_GuardBranching(t *testing.T) {
	exec := &stubExecutor{
		results: map[string]*ActivityResult{
			"review": {
				EventType: "review.done",
				Payload:   []byte(`{"_event":"review.done","score":"90"}`),
			},
		},
	}
	ir := buildIR("wf-3", "review",
		normal("review",
			[]*zynaxv1.ActionIR{action("review")},
			[]*zynaxv1.TransitionIR{
				transition("review.done", "rejected", map[string]string{"low": `ctx.score == "low"`}),
				transition("review.done", "approved", nil),
			},
		),
		terminal("approved"),
		terminal("rejected"),
	)
	pub := &stubPublisher{}
	_, err := (&IRInterpreter{}).Run(context.Background(), ir, exec, pub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Score "90" does not equal "low", so the second (unconditional) transition fires.
	// Final state should be "approved".
	last := pub.events[len(pub.events)-1]
	if last != "zynax.workflow.completed" {
		t.Errorf("last event = %q; want zynax.workflow.completed", last)
	}
}

func TestIRInterpreter_StateNotFound(t *testing.T) {
	ir := buildIR("wf-4", "missing")
	_, err := (&IRInterpreter{}).Run(context.Background(), ir, &stubExecutor{}, &stubPublisher{})
	if err == nil || !containsMsg(err, "state \"missing\" not found") {
		t.Errorf("expected state-not-found error, got: %v", err)
	}
}

func TestIRInterpreter_NoMatchingTransition(t *testing.T) {
	ir := buildIR("wf-5", "s1",
		normal("s1",
			[]*zynaxv1.ActionIR{action("cap")},
			[]*zynaxv1.TransitionIR{transition("other.event", "done", nil)},
		),
		terminal("done"),
	)
	_, err := (&IRInterpreter{}).Run(context.Background(), ir, &stubExecutor{}, &stubPublisher{})
	if err == nil {
		t.Fatal("expected error for no matching transition")
	}
}

func TestIRInterpreter_ActivityError(t *testing.T) {
	ir := buildIR("wf-6", "s1",
		normal("s1", []*zynaxv1.ActionIR{action("cap")}, nil),
	)
	exec := &stubExecutor{err: errors.New("broker down")}
	pub := &stubPublisher{}
	_, err := (&IRInterpreter{}).Run(context.Background(), ir, exec, pub)
	if err == nil {
		t.Fatal("expected error from activity")
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

func TestIRInterpreter_CtxMergedIntoNextAction(t *testing.T) {
	exec := &stubExecutor{
		results: map[string]*ActivityResult{
			"step1": {
				EventType: "step1.done",
				Payload:   []byte(`{"_event":"step1.done","key":"hello"}`),
			},
			"step2": {EventType: "step2.done"},
		},
	}
	var capturedInput ActivityInput
	captureExec := &captureExecutor{inner: exec, capture: &capturedInput}
	ir := buildIR("wf-7", "s1",
		normal("s1",
			[]*zynaxv1.ActionIR{action("step1")},
			[]*zynaxv1.TransitionIR{transition("step1.done", "s2", nil)},
		),
		normal("s2",
			[]*zynaxv1.ActionIR{{Capability: "step2", InputTemplateJson: `{"k":"{{ .ctx.key }}"}`}},
			[]*zynaxv1.TransitionIR{transition("step2.done", "done", nil)},
		),
		terminal("done"),
	)
	_, err := (&IRInterpreter{}).Run(context.Background(), ir, captureExec, &stubPublisher{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(capturedInput.InputPayload) != `{"k":"hello"}` {
		t.Errorf("template not resolved: %s", capturedInput.InputPayload)
	}
}

// captureExecutor wraps stubExecutor and records the last ActivityInput for step2.
type captureExecutor struct {
	inner   *stubExecutor
	capture *ActivityInput
}

func (c *captureExecutor) DispatchCapability(ctx context.Context, in ActivityInput) (*ActivityResult, error) {
	if in.CapabilityName == "step2" {
		*c.capture = in
	}
	return c.inner.DispatchCapability(ctx, in)
}

// --- unit tests for pure helper functions ---

func TestWorkflowStatus_IsTerminal(t *testing.T) {
	cases := []struct {
		name string
		s    WorkflowStatus
		want bool
	}{
		{"completed is terminal", WorkflowStatusCompleted, true},
		{"failed is terminal", WorkflowStatusFailed, true},
		{"cancelled is terminal", WorkflowStatusCancelled, true},
		{"running is not terminal", WorkflowStatusRunning, false},
		{"pending is not terminal", WorkflowStatusPending, false},
		{"unspecified is not terminal", WorkflowStatusUnspecified, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.IsTerminal(); got != tc.want {
				t.Errorf("IsTerminal(%v) = %v; want %v", tc.s, got, tc.want)
			}
		})
	}
}

func TestIRInterpreter_AsyncActionsSkipped(t *testing.T) {
	exec := &stubExecutor{}
	ir := buildIR("wf-async", "s1",
		&zynaxv1.StateIR{
			Id:   "s1",
			Type: zynaxv1.StateType_STATE_TYPE_NORMAL,
			Actions: []*zynaxv1.ActionIR{
				{Capability: "fire-and-forget", Async: true},
			},
			Transitions: []*zynaxv1.TransitionIR{
				{EventType: "", TargetState: "done"},
			},
		},
		terminal("done"),
	)
	pub := &stubPublisher{}
	if _, err := (&IRInterpreter{}).Run(context.Background(), ir, exec, pub); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvalGuard_Equality(t *testing.T) {
	ctx := map[string]string{"status": "approved"}
	if !evalGuard(`ctx.status == "approved"`, ctx) {
		t.Error("expected guard to pass")
	}
	if evalGuard(`ctx.status == "rejected"`, ctx) {
		t.Error("expected guard to fail")
	}
}

func TestEvalGuard_Inequality(t *testing.T) {
	ctx := map[string]string{"status": "pending"}
	if !evalGuard(`ctx.status != "approved"`, ctx) {
		t.Error("expected guard to pass")
	}
	if evalGuard(`ctx.status != "pending"`, ctx) {
		t.Error("expected guard to fail")
	}
}

func TestEvalGuard_LiteralLhsEquality(t *testing.T) {
	ctx := map[string]string{}
	if !evalGuard(`"approved" == "approved"`, ctx) {
		t.Error("literal == literal should pass")
	}
}

func TestEvalGuard_FailClosed_EmptyExpr(t *testing.T) {
	if evalGuard("", map[string]string{}) {
		t.Error("empty expression should be fail-closed (false)")
	}
}

func TestEvalGuard_FailClosed_ParseError(t *testing.T) {
	// Garbage expression cannot compile — fail-closed.
	if evalGuard("unknown garbage !@#", map[string]string{}) {
		t.Error("unparseable expression should be fail-closed (false)")
	}
}

func TestEvalGuard_FailClosed_TypeMismatch(t *testing.T) {
	// ctx.score is a string; 80 is an int — CEL type error at compile — fail-closed.
	if evalGuard("ctx.score >= 80", map[string]string{"score": "90"}) {
		t.Error("type-mismatched expression should be fail-closed (false)")
	}
}

func TestEvalGuard_FailClosed_MissingKey(t *testing.T) {
	// Key not in map — CEL eval error — fail-closed.
	if evalGuard(`ctx.missing_key == "x"`, map[string]string{}) {
		t.Error("missing ctx key should be fail-closed (false)")
	}
}

func TestEvalGuard_ProgCacheHit(t *testing.T) {
	// Evaluate the same expression twice to exercise the sync.Map cache path.
	ctx := map[string]string{"status": "ok"}
	expr := `ctx.status == "ok"`
	if !evalGuard(expr, ctx) {
		t.Error("first eval: expected true")
	}
	if !evalGuard(expr, ctx) {
		t.Error("second eval (cache hit): expected true")
	}
}

func TestEvalGuard_FailClosed_NonBoolResult(t *testing.T) {
	// ctx.status selects a string from the map; string is not bool — fail-closed.
	if evalGuard("ctx.status", map[string]string{"status": "ok"}) {
		t.Error("non-bool CEL result should be fail-closed (false)")
	}
}

func TestMergePayload_InvalidJSON(t *testing.T) {
	ctx := map[string]string{}
	mergePayload(ctx, []byte("not json"))
	if len(ctx) != 0 {
		t.Error("invalid JSON payload should leave ctx unchanged")
	}
}

// errorPublisher fails with an error for a specific event type; all others succeed.
type errorPublisher struct{ failOn string }

func (p *errorPublisher) Publish(_ context.Context, eventType, _, _ string, _ []byte) error {
	if eventType == p.failOn {
		return errors.New("publish failed")
	}
	return nil
}

func TestIRInterpreter_StateEnteredPublishError(t *testing.T) {
	ir := buildIR("wf-entered-err", "s1",
		normal("s1", []*zynaxv1.ActionIR{action("cap")},
			[]*zynaxv1.TransitionIR{transition("", "done", nil)}),
		terminal("done"),
	)
	pub := &errorPublisher{failOn: "zynax.workflow.state.entered"}
	_, err := (&IRInterpreter{}).Run(context.Background(), ir, &stubExecutor{}, pub)
	if err == nil || !containsMsg(err, "publish state.entered") {
		t.Errorf("expected state.entered error, got: %v", err)
	}
}

func TestMergePayload_StringValues(t *testing.T) {
	ctx := map[string]string{}
	mergePayload(ctx, []byte(`{"_event":"done","key":"val","num":42}`))
	if ctx["key"] != "val" {
		t.Errorf("key = %q; want %q", ctx["key"], "val")
	}
	if _, ok := ctx["_event"]; ok {
		t.Error("_event should not be merged into ctx")
	}
	if _, ok := ctx["num"]; ok {
		t.Error("non-string value should not be merged into ctx")
	}
}

func TestResolveTemplate(t *testing.T) {
	ctx := map[string]string{"name": "alice"}
	got, err := resolveTemplate(`{"user":"{{ .ctx.name }}"}`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"user":"alice"}` {
		t.Errorf("resolveTemplate = %s; want %s", got, `{"user":"alice"}`)
	}
}

// TestResolveTemplate_EscapesJSON guards EPIC-W data-flow: a ctx value carrying
// newlines and quotes (e.g. an LLM review passed between steps) must be
// JSON-escaped on substitution so the rendered input_payload stays valid JSON,
// rather than failing dispatch with "input_payload must be valid JSON".
func TestResolveTemplate_EscapesJSON(t *testing.T) {
	ctx := map[string]string{"review": "line 1\n\"quoted\" line 2\tend"}
	got, err := resolveTemplate(`{"prompt":"Rank:\n{{ .ctx.review }}"}`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]string
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("rendered payload is not valid JSON: %v\npayload=%s", err, got)
	}
	if out["prompt"] != "Rank:\nline 1\n\"quoted\" line 2\tend" {
		t.Errorf("decoded prompt = %q; want the original review verbatim", out["prompt"])
	}
}

// TestResolveTemplate_Deterministic asserts that resolveTemplate produces
// byte-identical output across repeated calls with the same multi-key ctx,
// guarding against non-determinism from map iteration order.
func TestResolveTemplate_Deterministic(t *testing.T) {
	ctx := map[string]string{
		"alpha":   "A",
		"beta":    "B",
		"gamma":   "G",
		"delta":   "D",
		"epsilon": "E",
	}
	tmpl := `{"a":"{{ .ctx.alpha }}","b":"{{ .ctx.beta }}","g":"{{ .ctx.gamma }}","d":"{{ .ctx.delta }}","e":"{{ .ctx.epsilon }}"}`
	first, err := resolveTemplate(tmpl, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 50; i++ {
		got, err := resolveTemplate(tmpl, ctx)
		if err != nil {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}
		if string(got) != string(first) {
			t.Fatalf("non-deterministic output on iteration %d:\n got  %s\n want %s", i, got, first)
		}
	}
}

// TestResolveTemplate_DefaultValue verifies the "default" FuncMap helper.
// {{ index .ctx "key" | default "fallback" }} returns "fallback" when "key" is absent.
func TestResolveTemplate_DefaultValue(t *testing.T) {
	ctx := map[string]string{}
	// Compose the template string to avoid escaping issues with nested quotes.
	tmplStr := `{"val":"` + `{{ index .ctx "missing" | default "fallback" }}` + `"}`
	got, err := resolveTemplate(tmplStr, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"val":"fallback"}` {
		t.Errorf("resolveTemplate = %s; want %s", got, `{"val":"fallback"}`)
	}
}

// TestResolveTemplate_DefaultValue_KeyPresent verifies that "default" is a no-op
// when the key is present and non-empty.
func TestResolveTemplate_DefaultValue_KeyPresent(t *testing.T) {
	ctx := map[string]string{"env": "prod"}
	tmplStr := `{"env":"` + `{{ index .ctx "env" | default "dev" }}` + `"}`
	got, err := resolveTemplate(tmplStr, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"env":"prod"}` {
		t.Errorf("resolveTemplate = %s; want %s", got, `{"env":"prod"}`)
	}
}

// TestResolveTemplate_MissingKey verifies that a missing key resolves to an empty
// string (missingkey=zero option) rather than returning an error.
func TestResolveTemplate_MissingKey(t *testing.T) {
	ctx := map[string]string{}
	got, err := resolveTemplate(`{"x":"{{ .ctx.absent }}"}`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"x":""}` {
		t.Errorf("resolveTemplate = %s; want empty string for missing key", got)
	}
}

// TestResolveTemplate_InvalidTemplate verifies that a malformed template returns an error.
func TestResolveTemplate_InvalidTemplate(t *testing.T) {
	ctx := map[string]string{}
	_, err := resolveTemplate(`{{ .ctx.key`, ctx)
	if err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}
}

// TestResolveTemplate_NoInjection verifies that a ctx value containing template
// syntax is not re-evaluated — text/template renders the substituted value verbatim
// rather than re-parsing it as a template action.
func TestResolveTemplate_NoInjection(t *testing.T) {
	ctx := map[string]string{
		"evil":  "{{ .ctx.other }}",
		"other": "secret",
	}
	got, err := resolveTemplate(`{"v":"{{ .ctx.evil }}"}`, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The output must NOT contain "secret" — proving the value was not re-executed.
	result := string(got)
	if strings.Contains(result, "secret") {
		t.Errorf("template injection: ctx value was re-executed as template; got: %s", result)
	}
}

func TestIRInterpreter_PublishErrorLogged(t *testing.T) {
	// Redirect the default slog logger to a buffer so we can assert the Warn line.
	var buf bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(old)

	// Terminal-only IR: only the "completed" publish fires; publisher returns error.
	ir := buildIR("wf-pub-err", "done", terminal("done"))
	pub := &stubPublisher{err: errors.New("event bus down")}

	// Run should succeed — publish failure is logged, not propagated.
	if _, err := (&IRInterpreter{}).Run(context.Background(), ir, &stubExecutor{}, pub); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "lifecycle event publish failed") {
		t.Errorf("expected slog.Warn log line, got: %s", buf.String())
	}
}

// TestEvalGuard_FailClosed_InvalidExpressions verifies fail-closed for specific
// expressions called out in the issue-#539 acceptance criteria.
func TestEvalGuard_FailClosed_InvalidExpressions(t *testing.T) {
	cases := []struct {
		name string
		expr string
	}{
		{"invalid-keywords", "not a valid expression !!!"},
		{"malformed-parenthesis", `ctx.x == ("y"`},
	}
	ctx := map[string]string{"x": "y"}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if evalGuard(tc.expr, ctx) {
				t.Errorf("expression %q should be fail-closed (false)", tc.expr)
			}
		})
	}
}

func TestEvalGuard_BooleanLiteral(t *testing.T) {
	if !evalGuard("true", map[string]string{}) {
		t.Error(`evalGuard("true", {}) should return true`)
	}
	if evalGuard("false", map[string]string{}) {
		t.Error(`evalGuard("false", {}) should return false`)
	}
}

// FuzzEvalGuard verifies that evalGuard never panics for any input combination.
// Full fuzz execution is deferred to M7.C (#469); this seed enables the corpus.
func FuzzEvalGuard(f *testing.F) {
	f.Add(`ctx.status == "approved"`, "status", "approved")
	f.Add(`ctx.status != "rejected"`, "status", "ok")
	f.Add("true", "key", "val")
	f.Add("false", "key", "val")
	f.Add("", "key", "val")
	f.Add("not a valid expression !!!", "key", "val")
	f.Fuzz(func(_ *testing.T, expr, key, val string) {
		_ = evalGuard(expr, map[string]string{key: val})
	})
}

// --- helpers ---

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsMsg(err error, sub string) bool {
	return err != nil && len(err.Error()) >= len(sub) && contains2(err.Error(), sub)
}

func contains2(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
