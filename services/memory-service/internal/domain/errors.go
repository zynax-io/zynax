// SPDX-License-Identifier: Apache-2.0

// Package domain defines the core interfaces and sentinel errors for memory-service.
package domain

import "errors"

var (
	// ErrKeyNotFound is returned when a requested key does not exist in the store.
	ErrKeyNotFound = errors.New("key not found")
	// ErrNamespaceNotFound is returned when a namespace does not exist.
	ErrNamespaceNotFound = errors.New("namespace not found")
	// ErrDimensionMismatch is returned when a stored vector's dimension differs from the query.
	ErrDimensionMismatch = errors.New("vector dimension mismatch")
)
