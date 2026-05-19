// SPDX-License-Identifier: Apache-2.0

package domain

import "errors"

// Sentinel errors returned by TaskService. All are safe to wrap with fmt.Errorf("%w", ...).
var (
	// ErrTaskNotFound is returned when a task_id is unknown.
	ErrTaskNotFound = errors.New("task not found")
	// ErrNoEligibleAgent is returned when no active agent declares the requested capability.
	ErrNoEligibleAgent = errors.New("no eligible agent for capability")
	// ErrTaskTerminal is returned when an operation is attempted on a terminal task.
	ErrTaskTerminal = errors.New("task is in a terminal state")
	// ErrInvalidArgument is returned when a request field fails validation.
	ErrInvalidArgument = errors.New("invalid argument")
)
