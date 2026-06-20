// SPDX-License-Identifier: Apache-2.0

// Package domain contains the core port definitions and value types for the
// engine-adapter service. It has zero imports from api or infrastructure layers.
package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"text/template"

	"github.com/google/cel-go/cel"
	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
)

// ActivityExecutor abstracts Temporal's workflow.ExecuteActivity.
// The infrastructure layer provides a concrete implementation using the Temporal SDK.
// Domain code depends only on this interface (ADR-015).
type ActivityExecutor interface {
	DispatchCapability(ctx context.Context, in ActivityInput) (*ActivityResult, error)
}

// EventPublisher abstracts CloudEvent publication via Temporal activities.
// Domain code depends only on this interface (ADR-015).
type EventPublisher interface {
	Publish(ctx context.Context, eventType, workflowID, stateID string) error
}

// ExecutionContext tracks the state machine's mutable state across activity executions.
// It is not persisted externally — Temporal's event log is the source of truth.
type ExecutionContext struct {
	WorkflowID   string
	CurrentState string
	Ctx          map[string]string
}

// IRInterpreter drives the state machine for a single workflow execution.
// It is a plain Go struct; the infrastructure layer registers it as a Temporal
// workflow function and provides concrete ActivityExecutor and EventPublisher
// implementations backed by the Temporal SDK (ADR-015).
type IRInterpreter struct{}

// Run drives the IR state machine until a terminal state or ctx cancellation.
func (i *IRInterpreter) Run(
	ctx context.Context,
	ir *zynaxv1.WorkflowIR,
	exec ActivityExecutor,
	pub EventPublisher,
) error {
	ec := &ExecutionContext{
		WorkflowID:   ir.GetWorkflowId(),
		CurrentState: ir.GetInitialState(),
		Ctx:          make(map[string]string),
	}
	// data is the run-scoped data context (ADR-029): written by output_bindings,
	// read by input_bindings. It lives for this Run only and is never persisted.
	// It is bound to this run's scope (run id + namespace); reads/writes that
	// present any other scope are denied so data never leaks across runs
	// (canvas C.3).
	scope := DataContextScope{RunID: ir.GetWorkflowId(), Namespace: ir.GetNamespace()}
	data := NewScopedWorkflowDataContext(scope)
	for {
		state := findState(ir, ec.CurrentState)
		if state == nil {
			return fmt.Errorf("engine-adapter: state %q not found in IR", ec.CurrentState)
		}
		if state.GetType() == zynaxv1.StateType_STATE_TYPE_TERMINAL {
			if err := pub.Publish(ctx, "zynax.workflow.completed", ec.WorkflowID, ec.CurrentState); err != nil {
				slog.Warn("lifecycle event publish failed", "event", "zynax.workflow.completed", "workflow_id", ec.WorkflowID, "err", err)
			}
			return nil
		}
		if err := pub.Publish(ctx, "zynax.workflow.state.entered", ec.WorkflowID, ec.CurrentState); err != nil {
			return fmt.Errorf("engine-adapter: publish state.entered: %w", err)
		}
		result, err := executeActions(ctx, state, ec, exec, data, scope)
		if err != nil {
			if perr := pub.Publish(ctx, "zynax.workflow.failed", ec.WorkflowID, ec.CurrentState); perr != nil {
				slog.Warn("lifecycle event publish failed", "event", "zynax.workflow.failed", "workflow_id", ec.WorkflowID, "err", perr)
			}
			return err
		}
		transition, err := resolveTransition(state.GetTransitions(), result, ec.Ctx)
		if err != nil {
			if perr := pub.Publish(ctx, "zynax.workflow.failed", ec.WorkflowID, ec.CurrentState); perr != nil {
				slog.Warn("lifecycle event publish failed", "event", "zynax.workflow.failed", "workflow_id", ec.WorkflowID, "err", perr)
			}
			return err
		}
		if perr := pub.Publish(ctx, "zynax.workflow.state.exited", ec.WorkflowID, ec.CurrentState); perr != nil {
			slog.Warn("lifecycle event publish failed", "event", "zynax.workflow.state.exited", "workflow_id", ec.WorkflowID, "err", perr)
		}
		mergePayload(ec.Ctx, result.Payload)
		ec.CurrentState = transition.GetTargetState()
	}
}

// executeActions runs each synchronous action in the state and returns the result
// of the last action. Async actions (action.GetAsync() == true) are skipped in M3.
// If no synchronous actions are present, a sentinel result with empty EventType is
// returned so the caller can match the first unconditional transition.
func executeActions(
	ctx context.Context,
	state *zynaxv1.StateIR,
	ec *ExecutionContext,
	exec ActivityExecutor,
	data *WorkflowDataContext,
	scope DataContextScope,
) (*ActivityResult, error) {
	var last *ActivityResult
	for _, action := range state.GetActions() {
		if action.GetAsync() {
			continue
		}
		var timeoutSec int32
		if t := action.GetTimeout(); t != nil {
			sec := t.GetSeconds()
			if sec > math.MaxInt32 {
				sec = math.MaxInt32
			}
			timeoutSec = int32(sec)
		}
		// Resolve input_bindings from the run-scoped data context (ADR-029) and
		// merge them into the template context. A missing or typed-mismatch
		// reference fails the run with a structured DataReferenceError.
		inputs, err := data.ResolveInputs(scope, action.GetInputBindings())
		if err != nil {
			return nil, err
		}
		tmplCtx := mergeInputs(ec.Ctx, inputs)
		resolved, err := resolveTemplate(action.GetInputTemplateJson(), tmplCtx)
		if err != nil {
			return nil, fmt.Errorf("engine-adapter: action %q template error: %w", action.GetCapability(), err)
		}
		in := ActivityInput{
			CapabilityName: action.GetCapability(),
			InputPayload:   resolved,
			WorkflowID:     ec.WorkflowID,
			TimeoutSeconds: timeoutSec,
		}
		result, err := exec.DispatchCapability(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("engine-adapter: action %q: %w", action.GetCapability(), err)
		}
		// Publish this action's declared output_bindings into the data context
		// so downstream states can consume them (ADR-029).
		if err := data.WriteOutputs(scope, state.GetId(), action.GetOutputBindings(), result.Payload); err != nil {
			return nil, err
		}
		last = result
	}
	if last == nil {
		return &ActivityResult{EventType: ""}, nil
	}
	return last, nil
}

// resolveTransition finds the first transition that matches the result event type
// and whose CEL guards all pass against ctx.
// An empty EventType matches any transition (used when a state has no sync actions).
func resolveTransition(
	transitions []*zynaxv1.TransitionIR,
	result *ActivityResult,
	ctx map[string]string,
) (*zynaxv1.TransitionIR, error) {
	for _, t := range transitions {
		if result.EventType != "" && t.GetEventType() != result.EventType {
			continue
		}
		if guardsMatch(t.GetConditions(), ctx) {
			return t, nil
		}
	}
	return nil, fmt.Errorf("engine-adapter: no transition matched event %q", result.EventType)
}

// guardsMatch returns true when every condition expression evaluates to true.
func guardsMatch(conditions map[string]string, ctx map[string]string) bool {
	for _, expr := range conditions {
		if !evalGuard(expr, ctx) {
			return false
		}
	}
	return true
}

// cel-go environment and program cache — created once, shared across all evalGuard calls.
// Programs are deterministic pure functions; caching is safe for Temporal workflow replays.
var (
	celEnvOnce sync.Once
	celEnv     *cel.Env
	celEnvErr  error
	progCache  sync.Map // map[string]cel.Program
)

func celEnvironment() (*cel.Env, error) {
	celEnvOnce.Do(func() {
		env, err := cel.NewEnv(
			cel.Variable("ctx", cel.MapType(cel.StringType, cel.StringType)),
		)
		if err != nil {
			celEnvErr = fmt.Errorf("cel.NewEnv: %w", err)
			return
		}
		celEnv = env
	})
	return celEnv, celEnvErr
}

// evalGuard evaluates a CEL expression against ctx using cel-go (github.com/google/cel-go).
// Returns false (fail-closed) on empty expression, compile error, eval error, or non-bool result.
// The cel.Environment is initialised once at process startup; cel.Programs are cached per
// unique expression string in a sync.Map to avoid recompilation on repeated evaluations.
// ctx map field access uses CEL select syntax: ctx.key is equivalent to ctx["key"].
func evalGuard(expr string, ctx map[string]string) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false
	}

	env, err := celEnvironment()
	if err != nil {
		slog.Warn("evalGuard: cel env init failed", "err", err)
		return false
	}

	var prog cel.Program
	if cached, ok := progCache.Load(expr); ok {
		prog = cached.(cel.Program)
	} else {
		ast, issues := env.Compile(expr)
		if issues != nil && issues.Err() != nil {
			slog.Warn("evalGuard: compile error", "expr", expr, "err", issues.Err())
			return false
		}
		compiled, err := env.Program(ast)
		if err != nil {
			slog.Warn("evalGuard: program build failed", "expr", expr, "err", err)
			return false
		}
		progCache.Store(expr, compiled)
		prog = compiled
	}

	out, _, err := prog.Eval(map[string]interface{}{"ctx": ctx})
	if err != nil {
		slog.Warn("evalGuard: eval error", "expr", expr, "err", err)
		return false
	}
	b, ok := out.Value().(bool)
	if !ok {
		slog.Warn("evalGuard: non-bool result", "expr", expr)
		return false
	}
	return b
}

// defaultFuncs provides a FuncMap with a "default" function for use in templates.
// Usage: {{ index .ctx "key" | default "fallback" }}
var defaultFuncs = template.FuncMap{
	"default": func(fallback, val string) string {
		if val == "" {
			return fallback
		}
		return val
	},
}

// resolveTemplate substitutes {{ .ctx.<key> }} placeholders in the JSON template
// using Go's stdlib text/template. The data root is map[string]any{"ctx": ctx},
// so existing {{ .ctx.key }} syntax continues to work unchanged.
//
// Compared to the previous string-replace implementation this adds:
//   - Proper output fidelity (no re-injection of template syntax from ctx values)
//   - Conditional expressions ({{ if .ctx.key }}...{{ end }})
//   - Default values via the "default" func: {{ index .ctx "key" | default "fallback" }}
//
// Template parse errors and execution errors are returned as non-nil errors so
// callers can propagate them rather than silently producing malformed JSON.
func resolveTemplate(tmpl string, ctx map[string]string) ([]byte, error) {
	t, err := template.New("").Funcs(defaultFuncs).Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("invalid template %q: %w", tmpl, err)
	}
	// Each ctx value is substituted INTO the input's JSON template, almost always
	// inside a JSON string literal (e.g. {"prompt":"...{{ .ctx.review }}..."}).
	// Data-context values are arbitrary capability outputs — LLM text is routinely
	// multi-line and quoted — so substituting them raw produces invalid JSON
	// ("input_payload must be valid JSON"). JSON-escape each value to its
	// string-inner form (marshal, strip the surrounding quotes) so the rendered
	// payload stays valid; for simple values this is a no-op (backward compatible).
	esc := make(map[string]string, len(ctx))
	for k, v := range ctx {
		b, mErr := json.Marshal(v)
		if mErr != nil { // string marshalling never fails, but stay defensive
			esc[k] = v
			continue
		}
		esc[k] = string(b[1 : len(b)-1])
	}
	var buf strings.Builder
	if err := t.Execute(&buf, map[string]any{"ctx": esc}); err != nil {
		return nil, fmt.Errorf("template execution failed: %w", err)
	}
	return []byte(buf.String()), nil
}

// mergePayload unmarshals the JSON payload and merges top-level string values into ctx.
// The "_event" key is reserved and skipped.
func mergePayload(ctx map[string]string, payload []byte) {
	if len(payload) == 0 {
		return
	}
	var m map[string]interface{}
	if err := json.Unmarshal(payload, &m); err != nil {
		return
	}
	for k, v := range m {
		if k == "_event" {
			continue
		}
		if s, ok := v.(string); ok {
			ctx[k] = s
		}
	}
}

// findState returns the StateIR with the given id or nil if not found.
func findState(ir *zynaxv1.WorkflowIR, stateID string) *zynaxv1.StateIR {
	for _, s := range ir.GetStates() {
		if s.GetId() == stateID {
			return s
		}
	}
	return nil
}
