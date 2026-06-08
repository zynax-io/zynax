// SPDX-License-Identifier: Apache-2.0

//go:build integration

package postgres_test

import (
	"context"
	"testing"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/zynax-io/zynax/services/memory-service/internal/infrastructure/postgres"
)

func setupVectorContainer(t *testing.T) (store *postgres.VectorStore, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	// pgvector is bundled in pgvector/pgvector:pg16 image.
	ctr, err := tcpostgres.Run(ctx,
		"pgvector/pgvector:pg16",
		tcpostgres.WithDatabase("memory_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start pgvector container: %v", err)
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = ctr.Terminate(ctx)
		t.Fatalf("connection string: %v", err)
	}

	s, err := postgres.New(ctx, dsn)
	if err != nil {
		_ = ctr.Terminate(ctx)
		t.Fatalf("open vector store: %v", err)
	}

	return s, func() {
		s.Close()
		_ = ctr.Terminate(ctx)
	}
}

func TestVectorStore_StoreAndQuery(t *testing.T) {
	store, cleanup := setupVectorContainer(t)
	defer cleanup()

	ctx := context.Background()
	ns := "test-ns"

	// Store three vectors in the namespace.
	if err := store.StoreVector(ctx, ns, "v1", []float32{1, 0, 0}, map[string]string{"tag": "alpha"}); err != nil {
		t.Fatalf("StoreVector v1: %v", err)
	}
	if err := store.StoreVector(ctx, ns, "v2", []float32{0, 1, 0}, map[string]string{"tag": "beta"}); err != nil {
		t.Fatalf("StoreVector v2: %v", err)
	}
	if err := store.StoreVector(ctx, ns, "v3", []float32{0, 0, 1}, map[string]string{"tag": "gamma"}); err != nil {
		t.Fatalf("StoreVector v3: %v", err)
	}

	// Query closest to [1, 0, 0] — should return v1 first.
	results, err := store.QueryVector(ctx, ns, []float32{1, 0, 0}, 3, "")
	if err != nil {
		t.Fatalf("QueryVector: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].ID != "v1" {
		t.Errorf("expected v1 as top result, got %q", results[0].ID)
	}
}

func TestVectorStore_NamespaceIsolation(t *testing.T) {
	store, cleanup := setupVectorContainer(t)
	defer cleanup()

	ctx := context.Background()

	// Store a vector in namespace A.
	if err := store.StoreVector(ctx, "ns-a", "shared-id", []float32{1, 0, 0}, nil); err != nil {
		t.Fatalf("StoreVector ns-a: %v", err)
	}

	// Query namespace B — must return no results.
	results, err := store.QueryVector(ctx, "ns-b", []float32{1, 0, 0}, 5, "")
	if err != nil {
		t.Fatalf("QueryVector ns-b: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("namespace isolation violated: got %d results for ns-b, expected 0", len(results))
	}
}

func TestVectorStore_Upsert(t *testing.T) {
	store, cleanup := setupVectorContainer(t)
	defer cleanup()

	ctx := context.Background()
	ns := "upsert-ns"

	// Store initial vector.
	if err := store.StoreVector(ctx, ns, "u1", []float32{1, 0, 0}, map[string]string{"v": "1"}); err != nil {
		t.Fatalf("StoreVector initial: %v", err)
	}

	// Upsert with a different embedding.
	if err := store.StoreVector(ctx, ns, "u1", []float32{0, 1, 0}, map[string]string{"v": "2"}); err != nil {
		t.Fatalf("StoreVector upsert: %v", err)
	}

	// Query: the updated vector [0,1,0] should now be the best match for [0,1,0].
	results, err := store.QueryVector(ctx, ns, []float32{0, 1, 0}, 1, "")
	if err != nil {
		t.Fatalf("QueryVector after upsert: %v", err)
	}
	if len(results) == 0 || results[0].ID != "u1" {
		t.Errorf("expected u1 after upsert, got %v", results)
	}
	if results[0].Metadata["v"] != "2" {
		t.Errorf("expected metadata v=2 after upsert, got %q", results[0].Metadata["v"])
	}
}

func TestVectorStore_DeleteVector(t *testing.T) {
	store, cleanup := setupVectorContainer(t)
	defer cleanup()

	ctx := context.Background()
	ns := "delete-ns"

	if err := store.StoreVector(ctx, ns, "d1", []float32{1, 0, 0}, nil); err != nil {
		t.Fatalf("StoreVector: %v", err)
	}

	if err := store.DeleteVector(ctx, ns, "d1"); err != nil {
		t.Fatalf("DeleteVector: %v", err)
	}

	// After deletion, querying should return empty.
	results, err := store.QueryVector(ctx, ns, []float32{1, 0, 0}, 5, "")
	if err != nil {
		t.Fatalf("QueryVector after delete: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after delete, got %d", len(results))
	}
}

func TestVectorStore_DeleteVector_NoOp(t *testing.T) {
	store, cleanup := setupVectorContainer(t)
	defer cleanup()

	ctx := context.Background()
	// Deleting a non-existent vector must not return an error.
	if err := store.DeleteVector(ctx, "noop-ns", "nonexistent"); err != nil {
		t.Fatalf("DeleteVector non-existent: %v", err)
	}
}

func TestVectorStore_QueryVector_FilterByMetadata(t *testing.T) {
	store, cleanup := setupVectorContainer(t)
	defer cleanup()

	ctx := context.Background()
	ns := "filter-ns"

	if err := store.StoreVector(ctx, ns, "f1", []float32{1, 0, 0}, map[string]string{"category": "A"}); err != nil {
		t.Fatalf("StoreVector f1: %v", err)
	}
	if err := store.StoreVector(ctx, ns, "f2", []float32{1, 0, 0}, map[string]string{"category": "B"}); err != nil {
		t.Fatalf("StoreVector f2: %v", err)
	}

	// Filter to category=A only.
	results, err := store.QueryVector(ctx, ns, []float32{1, 0, 0}, 5, "category=A")
	if err != nil {
		t.Fatalf("QueryVector with filter: %v", err)
	}
	for _, r := range results {
		if r.Metadata["category"] != "A" {
			t.Errorf("filter leak: got result with category=%q", r.Metadata["category"])
		}
	}
}
