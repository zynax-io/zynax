// SPDX-License-Identifier: Apache-2.0

// Package domain contains the core port definitions and value types for the
// engine-adapter service. It has zero imports from api or infrastructure layers.
package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// dataRefPrefix marks an input binding value as a JSON-path reference into the
// workflow-scoped data context rather than a literal value (ADR-029 §1).
const dataRefPrefix = "$."

// DataContextScope identifies the single workflow run that owns a
// WorkflowDataContext (canvas C.3). A data context is bound to exactly one
// scope at construction; every read and write must present a scope that equals
// the owning scope, so one run can never read or write another run's data even
// if it somehow obtains a reference to the instance.
//
// RunID is the per-run identity (the workflow run / envelope workflow_id).
// Namespace is the run's namespace (ADR-008 — never share state across
// namespaces); it may be empty for single-namespace deployments.
type DataContextScope struct {
	RunID     string
	Namespace string
}

// equals reports whether two scopes denote the same run in the same namespace.
func (s DataContextScope) equals(other DataContextScope) bool {
	return s.RunID == other.RunID && s.Namespace == other.Namespace
}

// String renders the scope for diagnostics as "<namespace>/<run-id>".
func (s DataContextScope) String() string {
	return s.Namespace + "/" + s.RunID
}

// ScopeError is returned when a read or write presents a scope that does not
// match the data context's owning scope (canvas C.3 — cross-run access denied).
// It fails the access loudly rather than silently returning or dropping data.
type ScopeError struct {
	// Owner is the scope the data context is bound to.
	Owner DataContextScope
	// Requester is the scope that attempted the access.
	Requester DataContextScope
	// Op is the attempted operation ("read" or "write").
	Op string
}

func (e *ScopeError) Error() string {
	return fmt.Sprintf(
		"engine-adapter: cross-run data-context %s denied: context owned by %s, requested by %s",
		e.Op, e.Owner, e.Requester,
	)
}

// DataReferenceError is a structured error returned when an input binding
// reference cannot be resolved against the workflow-scoped data context, or
// when a referenced/extracted value is not a coercible scalar (a typed
// mismatch). It fails the run loudly per ADR-029 — there is no implicit
// fallback to an empty value.
type DataReferenceError struct {
	// InputKey is the input_bindings key whose value failed to resolve.
	InputKey string
	// Reference is the raw binding value (the `$.states.<state>.output.<key>`
	// path or the offending literal).
	Reference string
	// Reason is a human-readable explanation ("not found", "type mismatch", ...).
	Reason string
}

func (e *DataReferenceError) Error() string {
	return fmt.Sprintf(
		"engine-adapter: input %q reference %q: %s",
		e.InputKey, e.Reference, e.Reason,
	)
}

// WorkflowDataContext is the run-scoped key/value store that backs data-flow
// between states (ADR-029 §2). It is written only by an action's
// output_bindings and read only by a downstream action's input_bindings —
// there is no implicit/global mutable state. It lives for a single workflow
// run and is never persisted beyond it.
//
// Keys are canonical dotted paths of the form "states.<stateID>.output.<key>";
// values are the scalar string form of the extracted output. The interpreter
// owns exactly one instance per Run (behind the WorkflowEngine interface).
//
// The context is bound to a single DataContextScope at construction. Every read
// (ResolveInputs) and write (WriteOutputs) must present a matching scope, so a
// run can never leak data into or out of another run (canvas C.3).
type WorkflowDataContext struct {
	scope DataContextScope
	store map[string]string
}

// NewWorkflowDataContext returns an empty data context bound to the zero scope.
// It is retained for back-compat with callers that do not yet thread a run
// scope; reads and writes must present the same (zero) scope. Prefer
// NewScopedWorkflowDataContext so cross-run access is denied (canvas C.3).
func NewWorkflowDataContext() *WorkflowDataContext {
	return NewScopedWorkflowDataContext(DataContextScope{})
}

// NewScopedWorkflowDataContext returns an empty data context owned by scope.
// Every subsequent read/write must present a scope equal to this one, so the
// context cannot serve another run's data (canvas C.3 — cross-run denied).
func NewScopedWorkflowDataContext(scope DataContextScope) *WorkflowDataContext {
	return &WorkflowDataContext{scope: scope, store: make(map[string]string)}
}

// Scope returns the run scope that owns this data context.
func (c *WorkflowDataContext) Scope() DataContextScope {
	return c.scope
}

// assertScope returns a ScopeError when requester does not match the owning
// scope, enforcing strict run+namespace isolation on every read and write.
func (c *WorkflowDataContext) assertScope(requester DataContextScope, op string) error {
	if !c.scope.equals(requester) {
		return &ScopeError{Owner: c.scope, Requester: requester, Op: op}
	}
	return nil
}

// WriteOutputs extracts each declared output from the action's result payload
// and publishes it into the context under "states.<stateID>.output.<key>".
//
// bindings maps a context key to a source path within the JSON payload (a
// dotted path, e.g. "results" or "data.score"). A binding whose source path
// does not resolve to a scalar value is reported as a typed mismatch so the
// run fails loudly rather than storing an unusable value. A nil/empty bindings
// map is a no-op (actions that publish nothing).
//
// requester is the scope of the run performing the write; it must equal the
// context's owning scope or the write is denied with a ScopeError (canvas C.3).
func (c *WorkflowDataContext) WriteOutputs(requester DataContextScope, stateID string, bindings map[string]string, payload []byte) error {
	if err := c.assertScope(requester, "write"); err != nil {
		return err
	}
	if len(bindings) == 0 {
		return nil
	}
	var doc map[string]json.RawMessage
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &doc); err != nil {
			return &DataReferenceError{
				InputKey:  stateID,
				Reference: string(payload),
				Reason:    fmt.Sprintf("output payload is not a JSON object: %v", err),
			}
		}
	}
	for key, sourcePath := range bindings {
		val, err := extractScalar(doc, sourcePath)
		if err != nil {
			return &DataReferenceError{
				InputKey:  key,
				Reference: sourcePath,
				Reason:    err.Error(),
			}
		}
		c.store[outputKey(stateID, key)] = val
	}
	return nil
}

// ResolveInputs resolves an action's input_bindings into a flat key/value map
// ready to merge into the interpreter's template context. Each binding value
// is either a literal (returned verbatim) or a "$.states.<state>.output.<key>"
// reference resolved against the store. A reference that does not resolve is a
// DataReferenceError — the run fails rather than substituting an empty value.
// A nil/empty bindings map yields a nil result (actions that consume nothing).
//
// requester is the scope of the run performing the read; it must equal the
// context's owning scope or the read is denied with a ScopeError (canvas C.3).
func (c *WorkflowDataContext) ResolveInputs(requester DataContextScope, bindings map[string]string) (map[string]string, error) {
	if err := c.assertScope(requester, "read"); err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		return nil, nil
	}
	resolved := make(map[string]string, len(bindings))
	for inputKey, ref := range bindings {
		if !strings.HasPrefix(ref, dataRefPrefix) {
			// Literal value — consumed verbatim (ADR-029 §1).
			resolved[inputKey] = ref
			continue
		}
		key, err := canonicalRefKey(ref)
		if err != nil {
			return nil, &DataReferenceError{InputKey: inputKey, Reference: ref, Reason: err.Error()}
		}
		val, ok := c.store[key]
		if !ok {
			return nil, &DataReferenceError{
				InputKey:  inputKey,
				Reference: ref,
				Reason:    "not found in workflow data context",
			}
		}
		resolved[inputKey] = val
	}
	return resolved, nil
}

// mergeInputs overlays resolved input bindings onto the base template context
// without mutating base. Resolved inputs win on key collision (an explicit
// input binding overrides ambient ctx). When inputs is empty the base map is
// returned unchanged to avoid an allocation on the common no-bindings path.
func mergeInputs(base, inputs map[string]string) map[string]string {
	if len(inputs) == 0 {
		return base
	}
	merged := make(map[string]string, len(base)+len(inputs))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range inputs {
		merged[k] = v
	}
	return merged
}

// outputKey returns the canonical store key for a state's named output.
func outputKey(stateID, key string) string {
	return "states." + stateID + ".output." + key
}

// canonicalRefKey converts a "$.states.<state>.output.<key>" reference into the
// canonical store key "states.<state>.output.<key>". A reference that does not
// match the only supported shape is rejected — there is no expression language
// in M7 (ADR-029 §1).
func canonicalRefKey(ref string) (string, error) {
	path := strings.TrimPrefix(ref, dataRefPrefix)
	parts := strings.Split(path, ".")
	// Expected shape: states.<state>.output.<key> (4 segments, non-empty).
	if len(parts) != 4 || parts[0] != "states" || parts[2] != "output" ||
		parts[1] == "" || parts[3] == "" {
		return "", fmt.Errorf("malformed reference; expected %sstates.<state>.output.<key>", dataRefPrefix)
	}
	return path, nil
}

// extractScalar reads sourcePath (a dotted path) from a JSON object and returns
// the value as its scalar string form. Strings, numbers, and booleans are
// supported; an object, array, or null at the target path is a type mismatch.
// A path segment that is absent is reported as "not found".
func extractScalar(doc map[string]json.RawMessage, sourcePath string) (string, error) {
	if sourcePath == "" {
		return "", fmt.Errorf("empty source path")
	}
	segments := strings.Split(sourcePath, ".")
	cur := doc
	for i, seg := range segments {
		raw, ok := cur[seg]
		if !ok {
			return "", fmt.Errorf("source path %q not found in output payload", sourcePath)
		}
		if i == len(segments)-1 {
			return scalarString(raw)
		}
		var next map[string]json.RawMessage
		if err := json.Unmarshal(raw, &next); err != nil {
			return "", fmt.Errorf("source path %q traverses a non-object at %q", sourcePath, seg)
		}
		cur = next
	}
	return "", fmt.Errorf("source path %q not found in output payload", sourcePath)
}

// scalarString decodes a JSON value into its string form. Objects, arrays, and
// null are rejected as typed mismatches (the data context stores scalars only;
// ADR-029 §3 — no typed schema, stringly-typed paths).
func scalarString(raw json.RawMessage) (string, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("type mismatch: undecodable value")
	}
	switch t := v.(type) {
	case string:
		return t, nil
	case bool:
		if t {
			return "true", nil
		}
		return "false", nil
	case float64:
		// Render without a trailing ".0" for integral values.
		return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%f", t), "0"), "."), nil
	default:
		return "", fmt.Errorf("type mismatch: expected scalar, got %T", v)
	}
}
