// SPDX-License-Identifier: Apache-2.0

package domain

import "errors"

// Sentinel errors returned by domain operations and mapped to HTTP/gRPC status
// codes at the api boundary. Callers must use errors.Is to test these.
var (
	ErrCompilationFailed  = errors.New("api-gateway: compilation failed")
	ErrEngineUnavailable  = errors.New("api-gateway: engine unavailable")
	ErrNotFound           = errors.New("api-gateway: not found")
	ErrAgentAlreadyExists = errors.New("api-gateway: agent already registered")

	// ErrAgentDefRetired is returned for kind: AgentDef manifests now that the
	// Agent custom resource is the single source of truth (ADR-039): apply the
	// Agent CR with kubectl instead — see docs/patterns/agent-crd-migration.md.
	ErrAgentDefRetired = errors.New("api-gateway: AgentDef push registration retired (ADR-039) — apply a zynax.io/v1alpha1 Agent custom resource instead (docs/patterns/agent-crd-migration.md)")

	// ErrInvalidEvent is returned when an injected event is missing a required
	// field (run id or event type) before it reaches the event bus.
	ErrInvalidEvent = errors.New("api-gateway: invalid event")

	// ErrUnknownKind is returned when the manifest kind: field is absent or not
	// in the allowlist {Workflow, AgentDef}.
	ErrUnknownKind = errors.New("api-gateway: unsupported manifest kind")
)
