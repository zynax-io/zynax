<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M6.H: Postgres-Backed Repositories for task-broker + agent-registry

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #626
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-02
**Status:** Implemented

**Child issues:** #793 (J.0 task-broker Postgres repo) · #794 (J.1 agent-registry Postgres repo)

---

## R — Requirements

**Problem:** task-broker and agent-registry use in-memory repositories (`sync.RWMutex` + Go maps) as their only persistence layer. Multiple replicas hold disjoint views of tasks and agents; a restart loses all in-flight state. This blocks horizontal scaling and durable task history — both required before any production K8s workload. ADR-021 (accepted 2026-05-21) mandates Postgres-backed repos in M6.

**Definition of done:**
- `task-broker` uses `postgres.TaskRepository` when `ZYNAX_DB_ENABLED=true`; falls back to `memory.TaskRepository` when unset/false.
- `agent-registry` uses `postgres.AgentRepository` when `ZYNAX_DB_ENABLED=true`; falls back to `memory.AgentRepository`.
- Schema migrations run at service startup via `golang-migrate`; `m.Up()` is a no-op on second boot (`ErrNoChange`).
- Integration tests pass against a real Postgres container via `testcontainers-go` with `//go:build integration` tag.
- Both services can run 2+ replicas sharing the same Postgres without data loss or split-brain.
- In-memory adapters are retained (not deleted) as unit-test doubles.

---

## E — Entities

- **`domain.TaskRepository`** — existing port interface (unchanged): `Save`, `GetByID`, `Update`, `List(ListFilter) ListResult`. The domain layer has zero knowledge of Postgres.
- **`domain.AgentRepository`** — existing port interface (unchanged): `Save`, `Delete`, `FindByID`, `FindAll`, `FindByCapability`.
- **`infrastructure/memory/` (task-broker, agent-registry)** — existing in-memory adapters; retained as test doubles. Files are NOT deleted.
- **`infrastructure/postgres/repository.go`** (task-broker) — NEW: implements `domain.TaskRepository` via `pgx/v5`. Atomic `Save`/`Update` within a `BEGIN`/`COMMIT` transaction; keyset-cursor pagination on `List`.
- **`infrastructure/postgres/repository.go`** (agent-registry) — NEW: implements `domain.AgentRepository` via `pgx/v5`. Maintains `agents` + `heartbeats` tables; `FindByCapability` uses a JOIN with a capabilities column or join table.
- **`migrations/001_initial.sql`** (task-broker) — NEW: `CREATE TABLE tasks (...)`. Owned exclusively by task-broker; no other service may access this table (ADR-008).
- **`migrations/001_initial.sql`** (agent-registry) — NEW: `CREATE TABLE agents (...), CREATE TABLE heartbeats (...)`. Owned exclusively by agent-registry.
- **`ZYNAX_DB_DSN`** — env var: Postgres connection string. Injected at runtime; never hardcoded.
- **`ZYNAX_DB_ENABLED`** — env var: `"true"` in K8s and compose; unset/`"false"` in unit tests. Selects adapter at startup.
- **`golang-migrate/migrate`** — startup migration runner. Each service calls `m.Up()` before serving gRPC; `ErrNoChange` is not an error.
- **`testcontainers-go`** — integration-test infrastructure. Spins up `postgres:16-alpine` container in process for `//go:build integration` tests; no shared or external DB.

---

## A — Approach

**What we WILL do:**
- Add `internal/infrastructure/postgres/` sub-package per service; implement the domain repository port.
- Add `internal/infrastructure/postgres/migrations/001_initial.sql` per service.
- Wire the adapter selection in `cmd/<service>/main.go` via `ZYNAX_DB_ENABLED` flag.
- Add `golang-migrate/migrate` and `pgx/v5` to each service's `go.mod`.
- Add `testcontainers-go` as a test-only dependency; integration tests under `//go:build integration`.
- Ship task-broker (J.0 in EPIC J planning) first, then agent-registry (J.1) — same pattern applied twice.
- Update `infra/docker-compose/docker-compose.yml`: add a dedicated Postgres service for task-broker + agent-registry (separate database from Temporal's Postgres).

**What we WON'T do:**
- Delete the in-memory adapters — they are the unit-test doubles for all existing and future unit tests.
- Add a Redis layer for caching (Redis belongs to memory-service only, per ADR-008 and M6.J scope).
- Share Postgres tables between task-broker and agent-registry (ADR-008: each service owns its schema exclusively).
- Use an ORM — raw `pgx/v5` queries for predictable SQL.
- Add a new gRPC RPC — the domain port interface is unchanged; no proto changes required.

**ADR references:**
- ADR-008: No shared databases — task-broker owns `tasks`; agent-registry owns `agents`/`heartbeats`. Separate databases or separate schemas; no cross-service JOINs.
- ADR-009: Go services only — all implementation in `services/`.
- ADR-016: Layered testing — unit tests use memory adapter; integration tests use testcontainers-go.
- ADR-017: GOWORK=off — mandatory for all `go test` invocations inside `services/task-broker/` and `services/agent-registry/`.
- ADR-021: Postgres-backed repos — directly governs this epic; defines env vars, tool choices, migration strategy.

---

## S — Structure

**New files (task-broker):**
```
services/task-broker/
├── go.mod                                 ← add pgx/v5, golang-migrate, testcontainers-go
├── internal/
│   └── infrastructure/
│       ├── memory/
│       │   └── repository.go             ← existing file, renamed from memory_repo.go
│       └── postgres/
│           ├── repository.go             ← NEW: pgx/v5 TaskRepository impl
│           └── migrations/
│               └── 001_initial.sql       ← NEW: tasks table schema
└── cmd/task-broker/main.go               ← modified: ZYNAX_DB_ENABLED wiring
```

**New files (agent-registry):**
```
services/agent-registry/
├── go.mod                                 ← add pgx/v5, golang-migrate, testcontainers-go
├── internal/
│   └── infrastructure/
│       ├── memory/
│       │   └── repository.go             ← existing file, renamed from memory_repo.go
│       └── postgres/
│           ├── repository.go             ← NEW: pgx/v5 AgentRepository impl
│           └── migrations/
│               └── 001_initial.sql       ← NEW: agents + heartbeats tables
└── cmd/agent-registry/main.go            ← modified: ZYNAX_DB_ENABLED wiring
```

**Modified files:**
- `infra/docker-compose/docker-compose.yml` — add `postgres:16-alpine` service for task-broker + agent-registry (separate from Temporal's Postgres; separate databases `task_broker` and `agent_registry`).
- `infra/docker-compose/docker-compose.services.yml` — add `ZYNAX_DB_DSN` + `ZYNAX_DB_ENABLED=true` env vars to task-broker and agent-registry service blocks.

**No proto changes. No API contract changes. No Python changes.**

---

## O — Operations

1. **[J.0 — task-broker Postgres repo]** Add `pgx/v5` + `golang-migrate` to `services/task-broker/go.mod`; create `internal/infrastructure/postgres/migrations/001_initial.sql` (tasks table schema); implement `postgres.TaskRepository`; wire `ZYNAX_DB_ENABLED` in `cmd/task-broker/main.go`; add integration test with `testcontainers-go`; update `docker-compose.yml` with Postgres service; `GOWORK=off go test ./... -race` passes.

2. **[J.1 — agent-registry Postgres repo]** Same pattern as J.0 for agent-registry: add deps to `go.mod`; `migrations/001_initial.sql` (agents + heartbeats tables); `postgres.AgentRepository` implementing `FindByCapability` via JOIN; wire `ZYNAX_DB_ENABLED`; integration test; `GOWORK=off go test ./... -race` passes.

---

## N — Norms

- `feat:` PR type (new observable behaviour: durable state, multi-replica correctness).
- Every commit: `Signed-off-by` trailer + `Assisted-by: Claude/claude-sonnet-4-6` per AGENTS.md §Hard Constraints. Never `Co-Authored-By:` for AI.
- `GOWORK=off` required for every `go test` and `go build` invocation inside `services/task-broker/` and `services/agent-registry/` (ADR-017).
- Unit tests use `memory/` adapter — no Docker, no Postgres, no `testcontainers-go`.
- Integration tests use `//go:build integration` tag; run with `go test -tags=integration`.
- Domain coverage ≥ 90% on `internal/domain/` — enforced by CI (ADR-016).
- Never delete `memory/` adapters — they are test doubles; their deletion breaks all unit tests.
- `ZYNAX_DB_DSN` must never be hardcoded, printed to logs, or written to any tracked file.
- Migration `001_initial.sql` must be idempotent (`CREATE TABLE IF NOT EXISTS`) or use `golang-migrate` versioning to ensure `m.Up()` is safe on restart.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII
- [x] No prompt injection
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards
- **Never** delete `infrastructure/memory/` adapters — they are unit-test doubles; removing them breaks all unit tests that do not require a running Postgres.
- **Never** share the `tasks` table between task-broker and any other service (ADR-008).
- **Never** share the `agents` or `heartbeats` tables between agent-registry and any other service (ADR-008).
- **Never** hardcode a DSN string in source code or any tracked file — `ZYNAX_DB_DSN` is always read from the environment.
- **Never** run `go test` without `GOWORK=off` inside `services/task-broker/` or `services/agent-registry/` (ADR-017).
- **Never** use an ORM that generates SQL behind the scenes — raw `pgx/v5` only for predictable query plans.
- **Never** add a proto change or new gRPC RPC in this epic — port interfaces are unchanged; this is a pure infrastructure swap.
