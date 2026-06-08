// SPDX-License-Identifier: Apache-2.0

package domain

import "fmt"

// KVKey returns the Redis key for a given namespace and user-supplied key.
// Format: {ns}:{key} — all keys are scoped to their namespace at the storage layer.
func KVKey(ns, key string) string {
	return fmt.Sprintf("%s:%s", ns, key)
}

// KVKeyPrefix returns the key prefix used to enumerate all keys in a namespace.
func KVKeyPrefix(ns string) string {
	return fmt.Sprintf("%s:", ns)
}
