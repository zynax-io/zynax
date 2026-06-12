<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-021: Postgres-Backed Repositories for Horizontal Scaling (M6)

**Status:** Accepted  **Date:** 2026-05-21  
**Related:** ADR-001 (gRPC-first), ADR-009 (Go services), ADR-016 (Layered Testing)  
**Issues:** [#578](https://github.com/zynax-io/zynax/issues/578) (ADR authoring) · [#626](https://github.com/zynax-io/zynax/issues/626) (M6.H epic)  
**Milestone:** M6 — K8s Production-Ready

---

## Context

Zynax's task-broker and agent-registry use **in-memory repositories** for M5. Each service
holds state in a Go struct with a mutex. This is acceptable for a single-replica MVP but
blocks horizontal scaling:

- Multiple replicas of task-broker would each hold a disjoint view of tasks.
- Multiple replicas of agent-registry would each hold a disjoint agent list.
- A replica restart loses all in-flight state.

The 2026-05-20 principal architect review rated this as risk **R4** (High) and
gap **H1** in `docs/reviews/04-architecture-gaps.md`.

Three options were considered for M6:
- **Option A — Single replica, defer persistence** (acceptable for M5, not M6)
- **Option B — Redis-backed repositories**
- **Option C — Postgres-backed repositories** (selected)

The decision was made 2026-05-21 (`docs/reviews/DECISIONS-NEEDED.md §D3`).

---

## Decision

**Switch task-broker and agent-registry from in-memory to Postgres-backed repositories in M6.**

### Rationale

1. **ACID guarantees** — task state transitions require atomicity (claim a task,
   update status). Postgres transactions provide this; Redis requires Lua scripts.
2. **Queryable history** — completed tasks and agent heartbeat history are queryable
   for debugging and analytics. Redis has limited query capabilities.
3. **Front-loaded work** — Postgres is required before CNCF Sandbox (M8) regardless.
   Shipping in M6 alongside K8s deployment avoids a mid-Sandbox migration.
4. **Existing patterns** — Zynax uses Temporal (Postgres-backed) and NATS JetStream
   (file-backed) already. Adding Postgres is an operational pattern engineers already
   manage.

### Hexagonal architecture (ports & adapters)

The repository port is **unchanged**. Only the infrastructure adapter changes:

```
services/<service>/internal/
  domain/
    repository.go          ← port interface (unchanged)
  infrastructure/
    memory/
      repository.go        ← retained as test double (not removed)
    postgres/
      repository.go        ← NEW — implements domain port via pgx/v5
      migrations/
        001_initial.sql    ← schema
```

**The domain layer has zero knowledge of Postgres.** The adapter is injected at startup:

```go
// main.go (simplified)
var repo domain.TaskRepository
if cfg.DBEnabled {
    repo = postgres.NewTaskRepository(db)
} else {
    repo = memory.NewTaskRepository() // test / local mode
}
```

### Services affected

| Service | Repository | Table |
|---------|-----------|-------|
| task-broker | `TaskRepository` | `tasks` |
| agent-registry | `AgentRepository` | `agents`, `heartbeats` |

### Postgres provisioning

**Docker Compose:** Add `postgres:16-alpine` container alongside Temporal/NATS.
**Kubernetes:** project-owned Postgres subchart on the Docker Official `postgres:17` image
(M6 — distribution decided in ADR-026, which supersedes the original community-chart plan);
migrate to Postgres Operator (M7+).

Environment variables:
- `ZYNAX_DB_DSN` — Postgres connection string (e.g. `postgres://zynax:pwd@localhost:5432/zynax`)
- `ZYNAX_DB_ENABLED` — `true` in K8s and compose, unset/`false` in unit tests

### Migration strategy

Migrations run at service startup via `golang-migrate/migrate` (same tool Temporal uses).
Each service owns its own schema — no shared tables (ADR-001).

```go
// Startup migration (each service)
m, _ := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
if err := m.Up(); err != nil && err != migrate.ErrNoChange {
    log.Fatal("migration failed", err)
}
```

### Testing

- **Unit tests:** inject `memory.TaskRepository` — no DB required (existing pattern)
- **Integration tests:** spin up Postgres via `testcontainers-go` with `//go:build integration` tag
- **CI:** integration tests run in `test-integration` job (already exists, currently skipped — see #227)

---

## Consequences

### Positive
- Horizontal scaling: task-broker and agent-registry can run N replicas
- Durable state across restarts (no lost in-flight tasks)
- Queryable task history for debugging
- Satisfies CNCF Sandbox data durability requirement
- In-memory adapter retained as fast test double

### Negative / Trade-offs
- Postgres is a new runtime dependency (docker-compose and K8s Helm chart)
- Migration management adds ~50 LoC per service
- Integration test infrastructure requires testcontainers (Docker in CI)
- Adds ~4–5 weeks to M6 scope

### Scale target (M6)

| Service | M6 replicas | Horizontal scale? |
|---------|-------------|-------------------|
| api-gateway | 2+ | ✅ Stateless already |
| workflow-compiler | 2+ | ✅ After #466 (stateless compiler) |
| engine-adapter | 1 | ⚠ Temporal handles fan-out |
| task-broker | 2+ | ✅ With Postgres repo |
| agent-registry | 2+ | ✅ With Postgres repo |

---

## Rejected Alternatives

### Option A — Single replica, defer persistence (M7)

Deferred persistence would block horizontal scaling until M7, adding risk to CNCF
Sandbox (M8). In-memory state under load is risk R4 from the 2026-05-20 review.
Acceptable for M5 (already shipped); not acceptable for a production K8s release.

### Option B — Redis-backed repositories

Redis provides fast key-value storage with TTL support. However:
- No ACID transactions without Lua scripts (error-prone for task state machines)
- Limited query capability for task history
- Adds a second caching layer alongside the in-memory adapter (confusing to operators)
- Postgres is required in M8 regardless; Redis would be a stepping stone only

Redis remains an option for **caching** (e.g. compiled workflow IR cache in
workflow-compiler — see #466) but not for primary task/agent state.

---

*See also: `docs/reviews/04-architecture-gaps.md §H1/R4` · `docs/reviews/DECISIONS-NEEDED.md §D3`*
