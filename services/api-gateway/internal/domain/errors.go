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

	// ErrUnknownKind is returned when the manifest kind: field is absent or not
	// in the allowlist {Workflow, AgentDef}.
	ErrUnknownKind = errors.New("api-gateway: unsupported manifest kind")
)
