// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/zynax-io/zynax/services/memory-service/internal/domain"
)

// RedisKV implements domain.KVStore backed by a Redis instance.
// All keys are stored as {ns}:{key} using domain.KVKey for namespace isolation.
type RedisKV struct {
	client *redis.Client
}

// NewRedisKV creates a RedisKV wrapping an existing *redis.Client.
func NewRedisKV(client *redis.Client) *RedisKV {
	return &RedisKV{client: client}
}

// NewRedisKVFromDSN creates a RedisKV by parsing a Redis URL (e.g. redis://:pwd@host:6379/0).
func NewRedisKVFromDSN(dsn string) (*RedisKV, error) {
	opts, err := redis.ParseURL(dsn)
	if err != nil {
		return nil, fmt.Errorf("memory-service: parse redis dsn: %w", err)
	}
	return &RedisKV{client: redis.NewClient(opts)}, nil
}

// Set stores or overwrites value under key in ns with an optional TTL.
// ttl == 0 means no expiry (best-effort per proto invariant 3).
func (r *RedisKV) Set(ctx context.Context, ns, key string, value []byte, ttl time.Duration) error {
	if err := r.client.Set(ctx, domain.KVKey(ns, key), value, ttl).Err(); err != nil {
		return fmt.Errorf("memory-service: redis set: %w", err)
	}
	return nil
}

// Get returns the value for key in ns, or ErrKeyNotFound if absent.
func (r *RedisKV) Get(ctx context.Context, ns, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, domain.KVKey(ns, key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("%w: %q", domain.ErrKeyNotFound, key)
	}
	if err != nil {
		return nil, fmt.Errorf("memory-service: redis get: %w", err)
	}
	return val, nil
}

// Delete removes key from ns. No-op if the key does not exist.
func (r *RedisKV) Delete(ctx context.Context, ns, key string) error {
	if err := r.client.Del(ctx, domain.KVKey(ns, key)).Err(); err != nil {
		return fmt.Errorf("memory-service: redis del: %w", err)
	}
	return nil
}

// ListKeys returns all keys in ns matching pattern (glob syntax, applied after the namespace prefix).
// Uses SCAN to avoid blocking.
func (r *RedisKV) ListKeys(ctx context.Context, ns, pattern string) ([]string, error) {
	prefix := domain.KVKeyPrefix(ns)
	match := prefix + pattern

	var cursor uint64
	var keys []string
	for {
		var batch []string
		var err error
		batch, cursor, err = r.client.Scan(ctx, cursor, match, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("memory-service: redis scan: %w", err)
		}
		for _, k := range batch {
			keys = append(keys, strings.TrimPrefix(k, prefix))
		}
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

// MGet returns values for a batch of keys in ns. Missing keys map to nil.
func (r *RedisKV) MGet(ctx context.Context, ns string, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return map[string][]byte{}, nil
	}
	redisKeys := make([]string, len(keys))
	for i, k := range keys {
		redisKeys[i] = domain.KVKey(ns, k)
	}

	vals, err := r.client.MGet(ctx, redisKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("memory-service: redis mget: %w", err)
	}

	result := make(map[string][]byte, len(keys))
	for i, v := range vals {
		if v == nil {
			result[keys[i]] = nil
		} else {
			result[keys[i]] = []byte(v.(string))
		}
	}
	return result, nil
}

// MSet stores multiple key-value pairs in ns with an optional TTL via a pipeline.
func (r *RedisKV) MSet(ctx context.Context, ns string, pairs map[string][]byte, ttl time.Duration) error {
	if len(pairs) == 0 {
		return nil
	}
	pipe := r.client.Pipeline()
	for k, v := range pairs {
		pipe.Set(ctx, domain.KVKey(ns, k), v, ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("memory-service: redis mset pipeline: %w", err)
	}
	return nil
}

// DeleteNamespace removes all keys in ns using SCAN + batch DEL.
func (r *RedisKV) DeleteNamespace(ctx context.Context, ns string) error {
	prefix := domain.KVKeyPrefix(ns)
	var cursor uint64
	for {
		var batch []string
		var err error
		batch, cursor, err = r.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("memory-service: redis scan for delete: %w", err)
		}
		if len(batch) > 0 {
			if err := r.client.Del(ctx, batch...).Err(); err != nil {
				return fmt.Errorf("memory-service: redis del namespace: %w", err)
			}
		}
		if cursor == 0 {
			break
		}
	}
	return nil
}
