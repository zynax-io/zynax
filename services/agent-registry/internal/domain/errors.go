// SPDX-License-Identifier: Apache-2.0

package domain

import "errors"

// Sentinel errors returned by AgentRegistryService. All are safe to wrap with fmt.Errorf("%w", ...).
var (
	// ErrAgentNotFound is returned when an agent_id is unknown.
	ErrAgentNotFound = errors.New("agent not found")
	// ErrInvalidArgument is returned when a request field fails validation.
	ErrInvalidArgument = errors.New("invalid argument")
)
