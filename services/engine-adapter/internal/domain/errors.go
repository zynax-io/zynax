// SPDX-License-Identifier: Apache-2.0

// Package domain contains the core port definitions and value types for the
// engine-adapter service. It has zero imports from api or infrastructure layers.
package domain

import "errors"

var (
	// ErrEngineUnavailable is returned when the execution engine cannot be reached.
	ErrEngineUnavailable = errors.New("engine-adapter: engine unavailable")

	// ErrExecutionNotFound is returned when the run_id does not exist.
	ErrExecutionNotFound = errors.New("engine-adapter: execution not found")

	// ErrTerminalState is returned when an operation is attempted on a terminal run.
	ErrTerminalState = errors.New("engine-adapter: workflow is in a terminal state")
)
