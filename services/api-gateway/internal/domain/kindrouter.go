// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Kind represents the manifest resource type read from the YAML kind: field.
type Kind string

const (
	KindWorkflow Kind = "Workflow"
	KindAgentDef Kind = "AgentDef"
)

// ErrUnknownKind is returned when the manifest kind: field is absent or not
// in the allowlist {Workflow, AgentDef}.
var ErrUnknownKind = errors.New("api-gateway: unsupported manifest kind")

// DetectKind reads only the top-level kind: field from manifestYAML.
// Full manifest parsing and validation is intentionally delegated to
// WorkflowCompilerService (ADR-011).
func DetectKind(manifestYAML []byte) (Kind, error) {
	var envelope struct {
		Kind string `yaml:"kind"`
	}
	if err := yaml.Unmarshal(manifestYAML, &envelope); err != nil {
		return "", fmt.Errorf("api-gateway: yaml: %w", err)
	}
	switch Kind(envelope.Kind) {
	case KindWorkflow:
		return KindWorkflow, nil
	case KindAgentDef:
		return KindAgentDef, nil
	default:
		return "", fmt.Errorf("api-gateway: kind %q: %w", envelope.Kind, ErrUnknownKind)
	}
}
