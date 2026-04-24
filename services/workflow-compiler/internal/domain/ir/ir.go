// Package ir converts domain WorkflowGraph values to proto WorkflowIR messages.
package ir

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/workflow-compiler/internal/domain"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const irVersion = "v1"

// ToIR converts a validated WorkflowGraph to a proto WorkflowIR.
// workflowID and apiVersion are caller-supplied envelope values.
// compiledAt defaults to time.Now() when zero.
func ToIR(g *domain.WorkflowGraph, workflowID, apiVersion string, compiledAt time.Time) (*zynaxv1.WorkflowIR, error) {
	if g == nil {
		return nil, fmt.Errorf("graph must not be nil")
	}
	if workflowID == "" {
		return nil, fmt.Errorf("workflowID must not be empty")
	}
	if compiledAt.IsZero() {
		compiledAt = time.Now().UTC()
	}

	states, err := convertStates(g.States)
	if err != nil {
		return nil, err
	}

	return &zynaxv1.WorkflowIR{
		WorkflowId:   workflowID,
		Name:         g.Name,
		Namespace:    g.Namespace,
		ApiVersion:   apiVersion,
		CompiledAt:   timestamppb.New(compiledAt),
		InitialState: g.InitialState,
		States:       states,
		IrVersion:    irVersion,
	}, nil
}

// convertStates maps the domain state map to a deterministically ordered slice
// of StateIR. States are sorted by ID so that the output is stable across runs.
func convertStates(states map[string]*domain.State) ([]*zynaxv1.StateIR, error) {
	ids := make([]string, 0, len(states))
	for id := range states {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]*zynaxv1.StateIR, 0, len(states))
	for _, id := range ids {
		s := states[id]
		actions, err := convertActions(id, s.Actions)
		if err != nil {
			return nil, err
		}
		out = append(out, &zynaxv1.StateIR{
			Id:          id,
			Type:        convertStateType(s.Type),
			Actions:     actions,
			Transitions: convertTransitions(s.Transitions),
		})
	}
	return out, nil
}

func convertStateType(t domain.StateType) zynaxv1.StateType {
	switch t {
	case domain.StateTypeTerminal:
		return zynaxv1.StateType_STATE_TYPE_TERMINAL
	case domain.StateTypeHumanInTheLoop:
		return zynaxv1.StateType_STATE_TYPE_HUMAN_IN_THE_LOOP
	default:
		return zynaxv1.StateType_STATE_TYPE_NORMAL
	}
}

// convertActions maps domain Actions to ActionIR slice.
// input_template_json is produced by JSON-encoding Action.Input.
func convertActions(stateID string, actions []domain.Action) ([]*zynaxv1.ActionIR, error) {
	out := make([]*zynaxv1.ActionIR, 0, len(actions))
	for i, a := range actions {
		inputJSON, err := marshalInputTemplate(a.Input)
		if err != nil {
			return nil, fmt.Errorf("state %q action[%d]: cannot marshal input: %w", stateID, i, err)
		}
		var pbDur *durationpb.Duration
		if a.Timeout > 0 {
			pbDur = durationpb.New(a.Timeout)
		}
		out = append(out, &zynaxv1.ActionIR{
			Capability:        a.Capability,
			Timeout:           pbDur,
			InputTemplateJson: inputJSON,
			Async:             a.Async,
		})
	}
	return out, nil
}

func marshalInputTemplate(input map[string]interface{}) (string, error) {
	if len(input) == 0 {
		return "", nil
	}
	b, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("marshaling input template: %w", err)
	}
	return string(b), nil
}

func convertTransitions(transitions []domain.Transition) []*zynaxv1.TransitionIR {
	out := make([]*zynaxv1.TransitionIR, 0, len(transitions))
	for _, t := range transitions {
		conditions := t.Conditions
		if conditions == nil {
			conditions = make(map[string]string)
		}
		out = append(out, &zynaxv1.TransitionIR{
			EventType:   t.EventType,
			TargetState: t.TargetState,
			Conditions:  conditions,
		})
	}
	return out
}
