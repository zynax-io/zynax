// SPDX-License-Identifier: Apache-2.0

package infrastructure_test

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/zynax-io/zynax/services/memory-service/internal/domain"
	"github.com/zynax-io/zynax/services/memory-service/internal/infrastructure"
)

func newTestKV(t *testing.T) (*infrastructure.RedisKV, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return infrastructure.NewRedisKV(client), mr
}

func TestRedisKV_SetGet(t *testing.T) {
	kv, _ := newTestKV(t)
	ctx := context.Background()

	if err := kv.Set(ctx, "ns1", "k1", []byte("hello"), 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	val, err := kv.Get(ctx, "ns1", "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(val) != "hello" {
		t.Fatalf("got %q, want %q", string(val), "hello")
	}
}

func TestRedisKV_GetNotFound(t *testing.T) {
	kv, _ := newTestKV(t)
	_, err := kv.Get(context.Background(), "ns1", "missing")
	if !errors.Is(err, domain.ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestRedisKV_Delete(t *testing.T) {
	kv, _ := newTestKV(t)
	ctx := context.Background()
	_ = kv.Set(ctx, "ns1", "k1", []byte("v"), 0)
	if err := kv.Delete(ctx, "ns1", "k1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := kv.Get(ctx, "ns1", "k1")
	if !errors.Is(err, domain.ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound after delete, got %v", err)
	}
}

func TestRedisKV_DeleteNoOp(t *testing.T) {
	kv, _ := newTestKV(t)
	if err := kv.Delete(context.Background(), "ns1", "nonexistent"); err != nil {
		t.Fatalf("Delete of nonexistent key should be no-op, got: %v", err)
	}
}

func TestRedisKV_TTL(t *testing.T) {
	kv, mr := newTestKV(t)
	ctx := context.Background()
	if err := kv.Set(ctx, "ns1", "ttlkey", []byte("v"), 5*time.Second); err != nil {
		t.Fatalf("Set: %v", err)
	}

	mr.FastForward(6 * time.Second)

	_, err := kv.Get(ctx, "ns1", "ttlkey")
	if !errors.Is(err, domain.ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound after TTL expiry, got %v", err)
	}
}

func TestRedisKV_TTLZeroMeansNoExpiry(t *testing.T) {
	kv, mr := newTestKV(t)
	ctx := context.Background()
	if err := kv.Set(ctx, "ns1", "permanent", []byte("v"), 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	mr.FastForward(9999 * time.Second)

	val, err := kv.Get(ctx, "ns1", "permanent")
	if err != nil {
		t.Fatalf("expected key to persist with ttl=0, got: %v", err)
	}
	if string(val) != "v" {
		t.Fatalf("got %q, want %q", string(val), "v")
	}
}

func TestRedisKV_ListKeys(t *testing.T) {
	kv, _ := newTestKV(t)
	ctx := context.Background()
	_ = kv.Set(ctx, "ns1", "a", []byte("1"), 0)
	_ = kv.Set(ctx, "ns1", "b", []byte("2"), 0)
	_ = kv.Set(ctx, "ns2", "c", []byte("3"), 0)

	keys, err := kv.ListKeys(ctx, "ns1", "*")
	if err != nil {
		t.Fatalf("ListKeys: %v", err)
	}
	sort.Strings(keys)
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys in ns1, got %d: %v", len(keys), keys)
	}
	if keys[0] != "a" || keys[1] != "b" {
		t.Fatalf("expected [a b], got %v", keys)
	}
}

func TestRedisKV_ListKeysEmpty(t *testing.T) {
	kv, _ := newTestKV(t)
	keys, err := kv.ListKeys(context.Background(), "empty-ns", "*")
	if err != nil {
		t.Fatalf("ListKeys on empty namespace: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys, got %d", len(keys))
	}
}

func TestRedisKV_MGetMSet(t *testing.T) {
	kv, _ := newTestKV(t)
	ctx := context.Background()

	pairs := map[string][]byte{"k1": []byte("v1"), "k2": []byte("v2")}
	if err := kv.MSet(ctx, "ns1", pairs, 0); err != nil {
		t.Fatalf("MSet: %v", err)
	}

	got, err := kv.MGet(ctx, "ns1", []string{"k1", "k2", "missing"})
	if err != nil {
		t.Fatalf("MGet: %v", err)
	}
	if string(got["k1"]) != "v1" {
		t.Fatalf("k1: got %q, want %q", got["k1"], "v1")
	}
	if string(got["k2"]) != "v2" {
		t.Fatalf("k2: got %q, want %q", got["k2"], "v2")
	}
	if got["missing"] != nil {
		t.Fatalf("missing: expected nil, got %v", got["missing"])
	}
}

func TestRedisKV_MGetEmpty(t *testing.T) {
	kv, _ := newTestKV(t)
	got, err := kv.MGet(context.Background(), "ns1", []string{})
	if err != nil {
		t.Fatalf("MGet with empty keys: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

func TestRedisKV_MSetEmpty(t *testing.T) {
	kv, _ := newTestKV(t)
	if err := kv.MSet(context.Background(), "ns1", map[string][]byte{}, 0); err != nil {
		t.Fatalf("MSet with empty pairs should be no-op, got: %v", err)
	}
}

func TestRedisKV_DeleteNamespace(t *testing.T) {
	kv, _ := newTestKV(t)
	ctx := context.Background()
	_ = kv.Set(ctx, "ns1", "a", []byte("1"), 0)
	_ = kv.Set(ctx, "ns1", "b", []byte("2"), 0)
	_ = kv.Set(ctx, "ns2", "c", []byte("3"), 0)

	if err := kv.DeleteNamespace(ctx, "ns1"); err != nil {
		t.Fatalf("DeleteNamespace: %v", err)
	}

	keys, _ := kv.ListKeys(ctx, "ns1", "*")
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys in ns1 after DeleteNamespace, got %d", len(keys))
	}

	// ns2 must be unaffected
	keys, _ = kv.ListKeys(ctx, "ns2", "*")
	if len(keys) != 1 {
		t.Fatalf("expected 1 key in ns2, got %d", len(keys))
	}
}

func TestRedisKV_NamespaceIsolation(t *testing.T) {
	kv, _ := newTestKV(t)
	ctx := context.Background()
	_ = kv.Set(ctx, "ns1", "shared-key", []byte("ns1-value"), 0)

	// Same user key in a different namespace should not be found
	_, err := kv.Get(ctx, "ns2", "shared-key")
	if !errors.Is(err, domain.ErrKeyNotFound) {
		t.Fatalf("expected ErrKeyNotFound in ns2 (isolation), got %v", err)
	}
}
