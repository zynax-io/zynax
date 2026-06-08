// SPDX-License-Identifier: Apache-2.0

package domain

import "context"

// VectorStore is the domain interface for namespaced vector embedding storage.
// Backed by pgvector (HNSW index). Namespace isolation is enforced via
// a WHERE namespace = $1 clause in all queries.
type VectorStore interface {
	// StoreVector upserts an embedding with the given id in ns.
	StoreVector(ctx context.Context, ns, id string, vector []float32, metadata map[string]string) error

	// QueryVector returns the k nearest embeddings in ns to the given vector.
	// filter is an optional metadata predicate (empty string means no filter).
	QueryVector(ctx context.Context, ns string, vector []float32, k int, filter string) ([]VectorResult, error)

	// DeleteVector removes the embedding identified by id in ns.
	// No-op if the id does not exist.
	DeleteVector(ctx context.Context, ns, id string) error
}

// VectorResult is a single match returned by QueryVector.
type VectorResult struct {
	ID       string
	Score    float32
	Metadata map[string]string
}
