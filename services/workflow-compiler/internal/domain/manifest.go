package domain

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// defaultNamespace is used when metadata.namespace is omitted from the manifest.
const defaultNamespace = "default"

// StateType classifies the behaviour of a state in the workflow state machine.
type StateType int

// State type constants.
const (
	StateTypeNormal         StateType = 0 // standard execution state
	StateTypeTerminal       StateType = 1 // end state; no outbound transitions
	StateTypeHumanInTheLoop StateType = 2 // pauses for human input
)

// Action is a single capability invocation within a state. It holds no proto
// types — the api layer maps Action to ActionIR when building WorkflowIR.
//
// OutputBindings declares the named outputs this action publishes into the
// workflow-scoped data context (ADR-029): context key → source path within the
// action's result payload. InputBindings declares the inputs this action
// consumes: context key → a JSON-path reference rooted at
// "$.states.<state>.output.<key>". Literal (non-reference) inputs remain in
// Input. Both binding maps are nil when the action declares none — additive and
// backward-compatible (M7 EPIC W, issue #1177).
type Action struct {
	Capability     string
	Timeout        time.Duration // zero means no timeout
	Input          map[string]interface{}
	OutputBindings map[string]string
	InputBindings  map[string]string
	Async          bool
}

// Transition is an outbound edge in the workflow state machine.
type Transition struct {
	EventType   string
	TargetState string
	Guard       string                 // optional CEL expression
	Set         map[string]interface{} // context writes on fire
	Conditions  map[string]string      // labelled CEL conditions (maps to TransitionIR.conditions)
}

// State is a single node in the compiled workflow state machine.
type State struct {
	ID          string
	Type        StateType
	Actions     []Action
	Transitions []Transition
	// Outputs declares the workflow-level result of a TERMINAL state (ADR-042,
	// M7.U): result name → a literal or an ADR-029 data reference
	// "$.states.<state>.output.<key>". Nil when the state declares none. Valid
	// only on terminal states; the graph builder rejects it elsewhere.
	Outputs map[string]string
	Line    int // 1-based source line in the YAML manifest
}

// Manifest is the domain-level representation of a parsed workflow YAML manifest.
// It holds no proto types — proto mapping is the responsibility of the api layer.
type Manifest struct {
	APIVersion  string
	Kind        string
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string

	InitialState string
	States       map[string]*State
}

// ── YAML intermediate structs ─────────────────────────────────────────────────
// These types mirror the YAML shape and are private to this file.

type yamlManifest struct {
	Kind       string       `yaml:"kind"`
	APIVersion string       `yaml:"apiVersion"`
	Metadata   yamlMetadata `yaml:"metadata"`
	Spec       yamlSpec     `yaml:"spec"`
}

type yamlMetadata struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

type yamlSpec struct {
	InitialState string               `yaml:"initial_state"`
	States       map[string]yamlState `yaml:"states"`
}

type yamlState struct {
	Type    string                 `yaml:"type"`
	Actions []yamlAction           `yaml:"actions"`
	On      []yamlTransition       `yaml:"on"`
	Outputs map[string]interface{} `yaml:"outputs"`
}

type yamlAction struct {
	Capability string                 `yaml:"capability"`
	Timeout    string                 `yaml:"timeout"`
	Input      map[string]interface{} `yaml:"input"`
	Output     map[string]interface{} `yaml:"output"`
	Async      bool                   `yaml:"async"`
}

type yamlTransition struct {
	Event string                 `yaml:"event"`
	Goto  string                 `yaml:"goto"`
	Guard string                 `yaml:"guard"`
	Set   map[string]interface{} `yaml:"set"`
}

// ── ParseManifest ─────────────────────────────────────────────────────────────

// ParseManifest parses raw YAML bytes into a domain Manifest.
// Returns all errors found — not just the first — to let callers surface all
// problems in a single response. Returns (nil, errs) on any error.
func ParseManifest(_ context.Context, data []byte) (*Manifest, ParseErrors) { //nolint:funlen // four sequential validation phases (YAML→decode→top-level→states) are one concern
	// Phase 1: YAML syntax — parse into yaml.Node for source position info.
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, ParseErrors{{
			Code:    ErrorCodeYAMLParseError,
			Message: err.Error(),
			Line:    extractYAMLErrorLine(err),
		}}
	}
	if root.Kind == 0 {
		return nil, ParseErrors{{
			Code:    ErrorCodeMissingRequiredField,
			Message: "manifest is empty",
		}}
	}

	// Phase 2: Decode yaml.Node into intermediate struct.
	var raw yamlManifest
	if err := root.Decode(&raw); err != nil {
		return nil, ParseErrors{{
			Code:    ErrorCodeYAMLParseError,
			Message: err.Error(),
			Line:    extractYAMLErrorLine(err),
		}}
	}

	// Phase 3: Validate required top-level fields.
	var errs ParseErrors
	if raw.Kind != "Workflow" {
		errs = append(errs, ParseError{
			Code:    ErrorCodeMissingRequiredField,
			Message: fmt.Sprintf("kind must be 'Workflow', got %q", raw.Kind),
		})
	}
	if raw.Metadata.Name == "" {
		errs = append(errs, ParseError{
			Code:    ErrorCodeMissingRequiredField,
			Message: "metadata.name is required",
		})
	}
	if raw.Spec.InitialState == "" {
		errs = append(errs, ParseError{
			Code:    ErrorCodeNoInitialState,
			Message: "spec.initial_state is required",
		})
	}
	if len(raw.Spec.States) == 0 {
		errs = append(errs, ParseError{
			Code:    ErrorCodeMissingRequiredField,
			Message: "spec.states must contain at least one state",
		})
	}
	if len(errs) > 0 {
		return nil, errs
	}

	// Phase 4: Convert each YAML state, collecting per-state line numbers.
	stateLines := extractStateLines(&root)
	states := make(map[string]*State, len(raw.Spec.States))
	for name, ys := range raw.Spec.States {
		st, stateErrs := convertState(name, ys, stateLines[name])
		errs = append(errs, stateErrs...)
		if st != nil {
			states[name] = st
		}
	}
	if len(errs) > 0 {
		return nil, errs
	}

	ns := raw.Metadata.Namespace
	if ns == "" {
		ns = defaultNamespace
	}

	return &Manifest{
		APIVersion:   raw.APIVersion,
		Kind:         raw.Kind,
		Name:         raw.Metadata.Name,
		Namespace:    ns,
		Labels:       raw.Metadata.Labels,
		Annotations:  raw.Metadata.Annotations,
		InitialState: raw.Spec.InitialState,
		States:       states,
	}, nil
}

// convertState converts a yamlState into a domain State, validating all fields.
func convertState( //nolint:funlen // validates type + actions + transitions in one pass; splitting adds indirection without clarity
	name string,
	ys yamlState,
	line int,
) (*State, ParseErrors) {
	var errs ParseErrors
	st := &State{ID: name, Line: line}

	switch strings.ToLower(ys.Type) {
	case "", "normal":
		st.Type = StateTypeNormal
	case "terminal":
		st.Type = StateTypeTerminal
	case "human_in_the_loop":
		st.Type = StateTypeHumanInTheLoop
	default:
		errs = append(errs, ParseError{
			Code:      ErrorCodeInvalidFieldValue,
			Message:   fmt.Sprintf("state %q: unknown type %q", name, ys.Type),
			Line:      line,
			StateName: name,
		})
	}

	for i, ya := range ys.Actions {
		if ya.Capability == "" {
			errs = append(errs, ParseError{
				Code:      ErrorCodeMissingRequiredField,
				Message:   fmt.Sprintf("state %q: actions[%d].capability is required", name, i),
				Line:      line,
				StateName: name,
			})
			continue
		}
		var d time.Duration
		if ya.Timeout != "" {
			var parseErr error
			d, parseErr = time.ParseDuration(ya.Timeout)
			if parseErr != nil {
				errs = append(errs, ParseError{
					Code:      ErrorCodeInvalidFieldValue,
					Message:   fmt.Sprintf("state %q: actions[%d].timeout %q is not a valid duration", name, i, ya.Timeout),
					Line:      line,
					StateName: name,
				})
			}
		}
		outputs, outErrs := convertOutputBindings(name, i, ya.Output, line)
		errs = append(errs, outErrs...)
		literals, inputs := splitInputBindings(ya.Input)
		st.Actions = append(st.Actions, Action{
			Capability:     ya.Capability,
			Timeout:        d,
			Input:          literals,
			OutputBindings: outputs,
			InputBindings:  inputs,
			Async:          ya.Async,
		})
	}

	for i, yt := range ys.On {
		if yt.Event == "" {
			errs = append(errs, ParseError{
				Code:      ErrorCodeMissingRequiredField,
				Message:   fmt.Sprintf("state %q: on[%d].event is required", name, i),
				Line:      line,
				StateName: name,
			})
			continue
		}
		if yt.Goto == "" {
			errs = append(errs, ParseError{
				Code:      ErrorCodeMissingRequiredField,
				Message:   fmt.Sprintf("state %q: on[%d].goto is required", name, i),
				Line:      line,
				StateName: name,
			})
			continue
		}
		st.Transitions = append(st.Transitions, Transition{
			EventType:   yt.Event,
			TargetState: yt.Goto,
			Guard:       yt.Guard,
			Set:         yt.Set,
		})
	}

	outputs, outErrs := convertWorkflowOutputs(name, ys.Outputs, line)
	errs = append(errs, outErrs...)
	st.Outputs = outputs

	if len(errs) > 0 {
		return nil, errs
	}
	return st, nil
}

// convertWorkflowOutputs flattens a terminal state's outputs: mapping into a
// string→string map of result name → source path (a literal or an ADR-029
// "$.states.<state>.output.<key>" reference). Non-string values are a
// compile-time error: M7 has no transform language, so each value must be a
// literal string (ADR-042). Terminal-only placement and reference resolution are
// enforced later by the graph builder, which has the full state set.
func convertWorkflowOutputs(
	state string,
	raw map[string]interface{},
	line int,
) (map[string]string, ParseErrors) {
	if len(raw) == 0 {
		return nil, nil
	}
	var errs ParseErrors
	out := make(map[string]string, len(raw))
	for key, val := range raw {
		s, ok := val.(string)
		if !ok {
			errs = append(errs, ParseError{
				Code:      ErrorCodeInvalidFieldValue,
				Message:   fmt.Sprintf("state %q: outputs[%q] must be a string source path, got %T", state, key, val),
				Line:      line,
				StateName: state,
			})
			continue
		}
		out[key] = s
	}
	if len(out) == 0 {
		return nil, errs
	}
	return out, errs
}

// inputBindingPrefix is the JSON-path root that marks a string input value as a
// reference into the workflow data context rather than a literal. The full form
// is "$.states.<state>.output.<key>" (ADR-029, M7 EPIC W). There is no
// expression/transform language in M7 — a value resolves verbatim or is a
// compile-time error.
const inputBindingPrefix = "$.states."

// convertOutputBindings flattens a YAML output: mapping into a string→string
// map of context key → source path. Non-string source paths are a compile-time
// error: M7 has no transform language, so each binding value must be a literal
// path string (ADR-029).
func convertOutputBindings(
	state string,
	actionIdx int,
	raw map[string]interface{},
	line int,
) (map[string]string, ParseErrors) {
	if len(raw) == 0 {
		return nil, nil
	}
	var errs ParseErrors
	out := make(map[string]string, len(raw))
	for key, val := range raw {
		s, ok := val.(string)
		if !ok {
			errs = append(errs, ParseError{
				Code:      ErrorCodeInvalidFieldValue,
				Message:   fmt.Sprintf("state %q: actions[%d].output[%q] must be a string source path, got %T", state, actionIdx, key, val),
				Line:      line,
				StateName: state,
			})
			continue
		}
		out[key] = s
	}
	if len(out) == 0 {
		return nil, errs
	}
	return out, errs
}

// splitInputBindings partitions a YAML input: mapping into literal inputs (which
// stay in the input template, preserving their original types) and binding
// references (string values rooted at inputBindingPrefix). Either returned map
// is nil when it would be empty.
func splitInputBindings(raw map[string]interface{}) (literals map[string]interface{}, bindings map[string]string) {
	if len(raw) == 0 {
		return nil, nil
	}
	for key, val := range raw {
		if s, ok := val.(string); ok && strings.HasPrefix(s, inputBindingPrefix) {
			if bindings == nil {
				bindings = make(map[string]string)
			}
			bindings[key] = s
			continue
		}
		if literals == nil {
			literals = make(map[string]interface{})
		}
		literals[key] = val
	}
	return literals, bindings
}

// extractStateLines walks the yaml.Node tree to find the source line of each
// state name under spec.states.
func extractStateLines(root *yaml.Node) map[string]int {
	lines := make(map[string]int)
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return lines
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return lines
	}
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value != "spec" {
			continue
		}
		specNode := doc.Content[i+1]
		if specNode.Kind != yaml.MappingNode {
			break
		}
		for j := 0; j+1 < len(specNode.Content); j += 2 {
			if specNode.Content[j].Value != "states" {
				continue
			}
			statesNode := specNode.Content[j+1]
			if statesNode.Kind != yaml.MappingNode {
				break
			}
			for k := 0; k+1 < len(statesNode.Content); k += 2 {
				lines[statesNode.Content[k].Value] = statesNode.Content[k].Line
			}
		}
	}
	return lines
}

// yamlLineRe extracts a line number from gopkg.in/yaml.v3 error messages.
var yamlLineRe = regexp.MustCompile(`line (\d+)`)

func extractYAMLErrorLine(err error) int {
	if err == nil {
		return 0
	}
	matches := yamlLineRe.FindStringSubmatch(err.Error())
	if len(matches) < 2 {
		return 0
	}
	lineNum, _ := strconv.Atoi(matches[1])
	return lineNum
}
