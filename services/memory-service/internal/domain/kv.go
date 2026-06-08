// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"time"
)

// KVStore is the domain interface for namespaced key-value storage.
// All operations are scoped to a namespace (workflow_id); cross-namespace
// access is rejected at the handler layer before reaching this interface.
type KVStore interface {
	// Set stores or overwrites value under key in ns with an optional TTL.
	// ttl == 0 means no expiry (best-effort per proto invariant 3).
	Set(ctx context.Context, ns, key string, value []byte, ttl time.Duration) error

	// Get returns the value for key in ns, or ErrKeyNotFound if absent.
	Get(ctx context.Context, ns, key string) ([]byte, error)

	// Delete removes key from ns. No-op if the key does not exist.
	Delete(ctx context.Context, ns, key string) error

	// ListKeys returns all keys in ns matching pattern (glob syntax).
	ListKeys(ctx context.Context, ns, pattern string) ([]string, error)

	// MGet returns values for a batch of keys in ns. Missing keys map to nil.
	MGet(ctx context.Context, ns string, keys []string) (map[string][]byte, error)

	// MSet stores multiple key-value pairs in ns with an optional TTL.
	MSet(ctx context.Context, ns string, pairs map[string][]byte, ttl time.Duration) error

	// DeleteNamespace removes all keys in ns.
	DeleteNamespace(ctx context.Context, ns string) error
}
