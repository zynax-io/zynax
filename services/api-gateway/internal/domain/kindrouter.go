// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Kind represents the manifest resource type read from the YAML kind: field.
type Kind string

// Allowlisted manifest kinds. Unknown values are rejected at the gateway.
// kind: AgentDef was removed in M9.A (ADR-039): agent identity is declared by
// the zynax.io/v1alpha1 Agent custom resource, applied with kubectl.
const (
	KindWorkflow Kind = "Workflow"
)

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
	default:
		return "", fmt.Errorf("api-gateway: kind %q: %w", envelope.Kind, ErrUnknownKind)
	}
}
