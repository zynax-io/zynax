// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"errors"
	"fmt"
)

// ErrCrossNamespaceAccess is returned when a request attempts to read or write
// a resource whose namespace differs from the caller's namespace.
// The gRPC handler translates this to codes.PermissionDenied.
var ErrCrossNamespaceAccess = errors.New("cross-namespace access denied")

// ErrEmptyNamespace is returned when either the request namespace or the
// resource namespace is the empty string.
var ErrEmptyNamespace = errors.New("namespace must not be empty")

// ValidateAccess checks that requestNamespace and resourceNamespace are both
// non-empty and identical. It returns:
//   - ErrEmptyNamespace if either argument is empty.
//   - ErrCrossNamespaceAccess if the two namespaces differ.
//   - nil when access is permitted.
//
// All KV and Vector handler methods must call ValidateAccess before delegating
// to the domain store. TTL enforcement is best-effort: Redis applies EXPIRE at
// the storage layer and callers must not depend on sub-second precision
// (proto invariant 3).
func ValidateAccess(requestNamespace, resourceNamespace string) error {
	if requestNamespace == "" || resourceNamespace == "" {
		return ErrEmptyNamespace
	}
	if requestNamespace != resourceNamespace {
		return ErrCrossNamespaceAccess
	}
	return nil
}

// KVKey returns the Redis key for a given namespace and user-supplied key.
// Format: {ns}:{key} — all keys are scoped to their namespace at the storage layer.
func KVKey(ns, key string) string {
	return fmt.Sprintf("%s:%s", ns, key)
}

// KVKeyPrefix returns the key prefix used to enumerate all keys in a namespace.
func KVKeyPrefix(ns string) string {
	return fmt.Sprintf("%s:", ns)
}
