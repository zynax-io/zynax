// SPDX-License-Identifier: Apache-2.0

//go:build integration

package api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	zynaxv1 "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
	"github.com/zynax-io/zynax/services/memory-service/internal/api"
	"github.com/zynax-io/zynax/services/memory-service/internal/infrastructure"
	"github.com/zynax-io/zynax/services/memory-service/internal/infrastructure/postgres"
)

// ─── container helpers ───────────────────────────────────────────────────────

func setupRedisContainer(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start redis container: %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	host, err := ctr.Host(ctx)
	if err != nil {
		t.Fatalf("redis host: %v", err)
	}
	port, err := ctr.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("redis port: %v", err)
	}

	return redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port.Port()),
	})
}

func setupPostgresContainer(t *testing.T) *postgres.VectorStore {
	t.Helper()
	ctx := context.Background()

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
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	store, err := postgres.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open vector store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	return store
}

func newIntegrationHandler(t *testing.T) *api.Handler {
	t.Helper()
	redisClient := setupRedisContainer(t)
	vecStore := setupPostgresContainer(t)
	kvStore := infrastructure.NewRedisKV(redisClient)
	return api.NewHandler(kvStore, vecStore)
}

// ─── KV integration tests ────────────────────────────────────────────────────

func TestHandler_Integration_Set_Get(t *testing.T) {
	h := newIntegrationHandler(t)
	ctx := context.Background()

	_, err := h.Set(ctx, &zynaxv1.SetRequest{
		WorkflowId: "wf-integ-01",
		Key:        "greeting",
		Value:      []byte("hello from redis"),
	})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	resp, err := h.Get(ctx, &zynaxv1.GetRequest{
		WorkflowId: "wf-integ-01",
		Key:        "greeting",
	})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(resp.GetValue()) != "hello from redis" {
		t.Errorf("got %q, want %q", resp.GetValue(), "hello from redis")
	}
}

func TestHandler_Integration_Delete(t *testing.T) {
	h := newIntegrationHandler(t)
	ctx := context.Background()

	_, err := h.Set(ctx, &zynaxv1.SetRequest{
		WorkflowId: "wf-del",
		Key:        "to-delete",
		Value:      []byte("ephemeral"),
	})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	_, err = h.Delete(ctx, &zynaxv1.DeleteRequest{
		WorkflowId: "wf-del",
		Key:        "to-delete",
	})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = h.Get(ctx, &zynaxv1.GetRequest{
		WorkflowId: "wf-del",
		Key:        "to-delete",
	})
	if err == nil {
		t.Fatal("expected NotFound after delete, got nil error")
	}
}

func TestHandler_Integration_ListKeys(t *testing.T) {
	h := newIntegrationHandler(t)
	ctx := context.Background()

	ns := "wf-list"
	keys := []string{"alpha", "beta", "gamma"}
	for _, k := range keys {
		if _, err := h.Set(ctx, &zynaxv1.SetRequest{
			WorkflowId: ns,
			Key:        k,
			Value:      []byte(k),
		}); err != nil {
			t.Fatalf("Set %q: %v", k, err)
		}
	}

	resp, err := h.ListKeys(ctx, &zynaxv1.ListKeysRequest{
		WorkflowId: ns,
		Prefix:     "*",
	})
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	if len(resp.GetKeys()) != 3 {
		t.Errorf("expected 3 keys, got %d: %v", len(resp.GetKeys()), resp.GetKeys())
	}
}

func TestHandler_Integration_MGetMSet(t *testing.T) {
	h := newIntegrationHandler(t)
	ctx := context.Background()

	ns := "wf-mset"
	_, err := h.MSet(ctx, &zynaxv1.MSetRequest{
		WorkflowId: ns,
		Entries: []*zynaxv1.MSetEntry{
			{Key: "k1", Value: []byte("v1")},
			{Key: "k2", Value: []byte("v2")},
		},
	})
	if err != nil {
		t.Fatalf("MSet: %v", err)
	}

	mresp, err := h.MGet(ctx, &zynaxv1.MGetRequest{
		WorkflowId: ns,
		Keys:       []string{"k1", "k2"},
	})
	if err != nil {
		t.Fatalf("MGet: %v", err)
	}
	if len(mresp.GetEntries()) != 2 {
		t.Errorf("expected 2 entries, got %d", len(mresp.GetEntries()))
	}
}

func TestHandler_Integration_DeleteNamespace(t *testing.T) {
	h := newIntegrationHandler(t)
	ctx := context.Background()

	ns := "wf-delns"
	for i := 0; i < 3; i++ {
		if _, err := h.Set(ctx, &zynaxv1.SetRequest{
			WorkflowId: ns,
			Key:        fmt.Sprintf("key-%d", i),
			Value:      []byte("data"),
		}); err != nil {
			t.Fatalf("Set key-%d: %v", i, err)
		}
	}

	_, err := h.DeleteNamespace(ctx, &zynaxv1.DeleteNamespaceRequest{
		WorkflowId: ns,
	})
	if err != nil {
		t.Fatalf("DeleteNamespace: %v", err)
	}

	resp, err := h.ListKeys(ctx, &zynaxv1.ListKeysRequest{
		WorkflowId: ns,
		Prefix:     "*",
	})
	if err != nil {
		t.Fatalf("ListKeys after DeleteNamespace: %v", err)
	}
	if len(resp.GetKeys()) != 0 {
		t.Errorf("expected 0 keys after DeleteNamespace, got %d", len(resp.GetKeys()))
	}
}

// ─── Vector integration tests ─────────────────────────────────────────────────

func TestHandler_Integration_StoreVector_QueryVector(t *testing.T) {
	h := newIntegrationHandler(t)
	ctx := context.Background()

	ns := "wf-vec"

	// Store a vector.
	sresp, err := h.StoreVector(ctx, &zynaxv1.StoreVectorRequest{
		WorkflowId: ns,
		Embedding:  []float32{1, 0, 0},
		Text:       "unit vector along x",
		Metadata:   map[string]string{"tag": "x-axis"},
	})
	if err != nil {
		t.Fatalf("StoreVector: %v", err)
	}
	if sresp.GetVectorId() == "" {
		t.Fatal("expected non-empty vector_id")
	}

	// Store another vector.
	_, err = h.StoreVector(ctx, &zynaxv1.StoreVectorRequest{
		WorkflowId: ns,
		Embedding:  []float32{0, 1, 0},
		Text:       "unit vector along y",
	})
	if err != nil {
		t.Fatalf("StoreVector y: %v", err)
	}

	// Query closest to [1, 0, 0] — should return the first vector.
	qresp, err := h.QueryVector(ctx, &zynaxv1.QueryVectorRequest{
		WorkflowId: ns,
		Embedding:  []float32{1, 0, 0},
		TopK:       1,
	})
	if err != nil {
		t.Fatalf("QueryVector: %v", err)
	}
	if len(qresp.GetResults()) == 0 {
		t.Fatal("expected at least 1 query result")
	}
	if qresp.GetResults()[0].GetVectorId() != sresp.GetVectorId() {
		t.Errorf("expected vector_id %q as top result, got %q",
			sresp.GetVectorId(), qresp.GetResults()[0].GetVectorId())
	}
}

func TestHandler_Integration_DeleteVector(t *testing.T) {
	h := newIntegrationHandler(t)
	ctx := context.Background()

	ns := "wf-delvec"

	sresp, err := h.StoreVector(ctx, &zynaxv1.StoreVectorRequest{
		WorkflowId: ns,
		Embedding:  []float32{1, 0, 0},
	})
	if err != nil {
		t.Fatalf("StoreVector: %v", err)
	}

	_, err = h.DeleteVector(ctx, &zynaxv1.DeleteVectorRequest{
		WorkflowId: ns,
		VectorId:   sresp.GetVectorId(),
	})
	if err != nil {
		t.Fatalf("DeleteVector: %v", err)
	}

	qresp, err := h.QueryVector(ctx, &zynaxv1.QueryVectorRequest{
		WorkflowId: ns,
		Embedding:  []float32{1, 0, 0},
		TopK:       5,
	})
	if err != nil {
		t.Fatalf("QueryVector after delete: %v", err)
	}
	if len(qresp.GetResults()) != 0 {
		t.Errorf("expected 0 results after delete, got %d", len(qresp.GetResults()))
	}
}
