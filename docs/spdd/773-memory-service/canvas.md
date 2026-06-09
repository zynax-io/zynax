<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M6.J: memory-service KV + Vector Implementation

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #773
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-02
**Status:** Implemented

**Child issues:** #793 (J.0) · #794 (J.1) · #814 (J.2) · #815 (J.3) · #816 (J.4) · #817 (J.5) · #818 (J.6) · #819 (J.7)

---

## R — Requirements

**Problem:** `memory-service` is contract-only (proto + BDD feature file + AGENTS.md). No `go.mod`, no `cmd/`, no `internal/`. Agents cannot persist shared context across invocations; workflows cannot retrieve previously stored results. The M6 real e2e harness (EPIC G) requires a functional memory-service.

**Prerequisites:** `#626` (M6.H Postgres-backed repos) must merge J.0 and J.1 first, as memory-service's pgvector adapter shares the cluster Postgres instance and its migration pattern follows the same J.0 playbook.

**Definition of done:**
- All 10 gRPC RPCs in `MemoryService` return real responses (not UNIMPLEMENTED).
- `Set/Get/Delete/ListKeys/MGet/MSet/DeleteNamespace` operate against Redis KV.
- `StoreVector/QueryVector/DeleteVector` operate against pgvector.
- BDD scenarios in `memory_service.feature` pass against real Redis + Postgres containers.
- `workflow_id` isolation enforced: a client in namespace `A` cannot read keys from namespace `B`.
- The service deploys as a stateless Deployment; Redis is a dedicated StatefulSet/PVC.

**Single-store evaluation (required before J.2 Canvas is committed):** Evaluate whether Redis Stack (with the Vector module) can serve both KV and vector planes from a single Redis instance, eliminating Postgres as a memory-service dependency. Record the evaluation result in this canvas S-Safeguards section. Current decision: Redis for KV + Postgres/pgvector for vector (per `services/memory-service/AGENTS.md` pre-decision) unless evaluation favours Redis Stack.

**Single-store evaluation result:** Redis Stack (Redis 7.x with `redisearch`/`redisvl` vector module) supports approximate nearest-neighbour search (HNSW index). However: (1) Redis Stack is not the same OCI image as `redis:7-alpine` already used in docker-compose — requires a separate `redis/redis-stack-server` image; (2) vector search in Redis Stack requires `redisvl` Python client or `redis-py` ≥5.0 with vector extensions; (3) pgvector has better ecosystem support for `pgx/v5` and the Zynax Go codebase already uses pgx. Keeping Redis (KV) + pgvector is the lower-risk choice for M6. Redis Stack consolidation can be revisited in M7.

---

## E — Entities

- **`domain.KVStore`** — NEW interface: `Set(ctx, ns, key, value, ttl)`, `Get(ctx, ns, key)`, `Delete(ctx, ns, key)`, `ListKeys(ctx, ns, pattern)`, `MGet(ctx, ns, keys)`, `MSet(ctx, ns, pairs, ttl)`, `DeleteNamespace(ctx, ns)`.
- **`domain.VectorStore`** — NEW interface: `StoreVector(ctx, ns, id, vector, metadata)`, `QueryVector(ctx, ns, vector, k, filter)`, `DeleteVector(ctx, ns, id)`.
- **`domain.ErrKeyNotFound`**, **`ErrNamespaceNotFound`**, **`ErrDimensionMismatch`** — NEW sentinel errors.
- **`infrastructure/redis_kv.go`** — NEW: implements `KVStore` via `go-redis/v9`; all keys namespaced as `{ns}:{key}`; TTL set via Redis `EXPIRE`.
- **`infrastructure/pgvector.go`** — NEW: implements `VectorStore` via `pgx/v5` + `pgvector-go`; ANN search via `pgvector` HNSW index.
- **`migrations/001_initial.sql`** (memory-service) — NEW: `memory_vectors` table with `EXTENSION vector`; owned exclusively by memory-service (ADR-008).
- **`api/handler.go`** — NEW: gRPC handler wiring all 10 `MemoryService` RPCs to domain interfaces.
- **Redis `StatefulSet`** — dedicated Redis instance for memory-service only. No other service may use this Redis instance (ADR-008).
- **`ZYNAX_REDIS_DSN`** — env var: Redis connection URI; never hardcoded.
- **`ZYNAX_DB_DSN`** (memory-service) — env var: Postgres connection string for pgvector; memory-service's exclusive schema.

---

## A — Approach

**What we WILL do:**
- Scaffold the service (J.2): `go.mod`, `cmd/memory-service/main.go`, domain interfaces, error sentinels.
- Implement Redis KV adapter (J.3): `go-redis/v9`; all keys namespaced; TTL enforcement; `DeleteNamespace` uses `SCAN` + batch `DEL`.
- Implement pgvector adapter (J.4): `pgx/v5` + `pgvector-go`; ANN search (HNSW); namespace isolation via `WHERE namespace = $1`.
- Implement namespace TTL enforcement + `workflow_id` isolation (J.5): cross-namespace access rejected with `PERMISSION_DENIED`; KV TTL enforced as best-effort (per proto invariant 3).
- Wire all 10 gRPC handlers (J.6): map domain calls to gRPC responses; integration tests via `testcontainers-go`.
- Implement BDD step functions (J.7): `memory_service.feature` scenarios pass against testcontainers.

**What we WON'T do:**
- Switch to Redis Stack single-store (evaluation above shows pgvector is the better M6 choice).
- Share memory-service's Redis with any other service (ADR-008: dedicated Redis).
- Share memory-service's Postgres schema with task-broker or agent-registry (ADR-008).
- Implement vector replication, clustering, or backup in M6.

**ADR references:**
- ADR-001: gRPC — all 10 RPCs are gRPC; no HTTP/REST.
- ADR-008: No shared databases — Redis and Postgres schema are exclusively memory-service's.
- ADR-009: Go services — all implementation is Go in `services/`.
- ADR-016: Layered testing — unit tests use mock adapters; integration tests use testcontainers.
- ADR-017: GOWORK=off — mandatory for all `go test` in `services/memory-service/`.

---

## S — Structure

```
services/memory-service/
├── go.mod                              ← NEW: pgx/v5, pgvector-go, go-redis/v9, golang-migrate, testcontainers-go
├── cmd/memory-service/main.go          ← NEW: gRPC server wiring
└── internal/
    ├── domain/
    │   ├── kv.go                       ← NEW: KVStore interface
    │   ├── vector.go                   ← NEW: VectorStore interface
    │   ├── namespace.go                ← NEW: isolation helpers
    │   └── errors.go                   ← NEW: ErrKeyNotFound etc.
    ├── api/
    │   └── handler.go                  ← NEW: gRPC handler (all 10 RPCs)
    └── infrastructure/
        ├── redis_kv.go                 ← NEW: KVStore impl (J.3)
        ├── redis_kv_test.go
        ├── pgvector.go                 ← NEW: VectorStore impl (J.4)
        ├── pgvector_test.go
        └── postgres/
            └── migrations/
                └── 001_initial.sql     ← NEW: memory_vectors table + vector extension
```

---

## O — Operations

1. ✅ **[J.2]** Scaffold memory-service: `go.mod`, `cmd/memory-service/main.go` (compiles; returns UNIMPLEMENTED on all RPCs), `internal/domain/kv.go`, `vector.go`, `namespace.go`, `errors.go`. `GOWORK=off go build ./...` succeeds.

2. ✅ **[J.3]** Implement `infrastructure/redis_kv.go`: `go-redis/v9`; `Set/Get/Delete/ListKeys/MGet/MSet/DeleteNamespace` with `{ns}:{key}` key prefix; TTL via Redis `EXPIRE`; unit tests with `miniredis` mock; `GOWORK=off go test ./... -race` passes.

3. **[J.4]** Implement `infrastructure/pgvector.go`: `pgx/v5` + `pgvector-go`; `migrations/001_initial.sql` (`CREATE EXTENSION IF NOT EXISTS vector; CREATE TABLE memory_vectors ...`); `StoreVector/QueryVector/DeleteVector`; namespace isolation via `WHERE namespace = $1`; integration test with testcontainers; `GOWORK=off go test -tags=integration ./... -race` passes.

4. **[J.5]** Implement namespace TTL enforcement + workflow_id isolation: service rejects cross-namespace access with `PERMISSION_DENIED`; KV TTL is best-effort (Redis handles it); add cross-namespace rejection test; `GOWORK=off go test ./... -race` passes.

5. **[J.6]** Wire all 10 gRPC handlers in `api/handler.go`; integration test via testcontainers (Redis + Postgres); verify all 10 RPCs return correct responses; `GOWORK=off go test -tags=integration ./... -race` passes; `make lint` passes.

6. **[J.7]** Implement BDD step functions for `services/memory-service/tests/features/memory_service.feature` (6 scenarios); use testcontainers for real Redis + Postgres; `make test-bdd` passes.

---

## N — Norms

- `feat:` PR type for J.2–J.6; `test:` for J.7.
- Every commit: `Signed-off-by` trailer + `Assisted-by: Claude/claude-sonnet-4-6` per AGENTS.md §Hard Constraints.
- `GOWORK=off` required for all `go test` in `services/memory-service/` (ADR-017).
- Unit tests use mock adapters (`miniredis` for Redis) — no real Docker in unit test runs.
- Integration tests use `//go:build integration` tag + testcontainers-go.
- Domain coverage ≥ 90% on `internal/domain/` after J.6.
- `ZYNAX_REDIS_DSN` and `ZYNAX_DB_DSN` must never be hardcoded — always read from env.
- KV TTL is best-effort per proto invariant 3 — callers must not depend on sub-second precision.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII
- [x] No prompt injection
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards
- **Never** allow cross-namespace access — service MUST reject any key/vector read or write that crosses namespace boundaries with `PERMISSION_DENIED`.
- **Never** share memory-service's Redis instance with any other service (ADR-008: dedicated Redis).
- **Never** share the `memory_vectors` Postgres schema with task-broker or agent-registry (ADR-008).
- **Never** hardcode Redis or Postgres DSN strings — always read from `ZYNAX_REDIS_DSN` / `ZYNAX_DB_DSN`.
- **Never** use Redis Stack single-store in M6 — pgvector is the chosen vector store (evaluation recorded above).
- **State-minimization justification** (per M6 planning mandate): JetStream cannot substitute for KV+vector: JetStream is async message delivery with no synchronous KV read or similarity search capability. Twelve-factor principle requires agents to hold no shared state internally; memory-service is the designated shared state boundary. Redis (KV TTL) + pgvector (ANN search) is the minimum viable persistence combination for the MemoryService contract.
