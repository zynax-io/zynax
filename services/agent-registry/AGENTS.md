# services/agent-registry — AGENTS.md

> Go toolchain pinned in the workspace [`go.work`](../../go.work). Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M9 — CRD-native Scheduler (ADR-039, EPICs #1571 + #1674).** The `Agent`
> custom resource (zynax.io/v1alpha1) is the single source of truth; this
> service is the STATELESS scheduler over it: informer-fed capability index,
> `SchedulerService.SelectAgent` (readiness- and metrics-aware, structured
> rationale), and a Lease-elected readiness reconciler deriving Agent status
> from EndpointSlices. The push-era `AgentRegistryService` surface, its
> repository adapters, and the database wiring were **removed** in M9 (#1698 +
> #1598, ADR-039 removal clause) — the RPCs no longer exist in the contract
> (migration: docs/patterns/agent-crd-migration.md). No database.

---

## Purpose

The agent-registry deployment hosts the **CRD-native scheduler** (ADR-039):
the `Agent` custom resource is the source of truth for identity and
capabilities; this service watches it and answers "which ONE agent should
take this capability now?"

- Watches `Agent` CRs in its namespace and maintains the capability index.
- Answers `SchedulerService.SelectAgent` with exactly one agent + a structured
  rationale (readiness-filtered; metrics-weighted when `PROM_URL` is set).
- Reconciles `Agent` readiness from EndpointSlices under a Lease election.

Does NOT: register agents (apply an `Agent` CR instead) · persist anything ·
assign tasks · store agent memory · authenticate external callers.

---

## Internal Layout

```
services/agent-registry/
├── cmd/agent-registry/
│   └── main.go                  ← composition root (envconfig, informer, gRPC server, graceful shutdown)
├── internal/
│   ├── api/
│   │   └── scheduler_handler.go ← SchedulerService.SelectAgent over the index + scorer
│   ├── domain/
│   │   └── scheduler/
│   │       ├── index.go         ← informer-fed capability index (capability → candidates)
│   │       └── scorer.go        ← filter → readiness → score pipeline + rationale
│   └── infrastructure/
│       ├── crd/                 ← controller-runtime manager: Agent informer + readiness reconciler
│       ├── promql/              ← Prometheus HTTP API metrics source (short-TTL cache)
│       └── tlscreds.go          ← mTLS transport credentials (ADR-020)
├── go.mod                       ← module github.com/zynax-io/zynax/services/agent-registry
└── go.sum
```

There is no `internal/domain/repository.go`, no repository adapter, and no
`tests/` BDD suite: the push surface they served was removed in #1698. The
`SchedulerService` contract is covered by `protos/tests/scheduler_service/`.

---

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `ZYNAX_REGISTRY_GRPC_PORT` | `50052` | gRPC listener port |
| `ZYNAX_REGISTRY_METRICS_PORT` | `9090` | Prometheus `/metrics` port |
| `ZYNAX_REGISTRY_LOG_LEVEL` | `info` | Log level: debug, info, warn, error |
| `ZYNAX_REGISTRY_CRD_INFORMER_ENABLED` | `false` | Watch `Agent` CRs and serve `SelectAgent` |
| `ZYNAX_REGISTRY_WATCH_NAMESPACE` | pod namespace | Namespace scope for the informer + Lease |
| `ZYNAX_REGISTRY_PROM_URL` | *(empty)* | Prometheus HTTP API; empty ⇒ degraded readiness-filtered selection |
| `ZYNAX_REGISTRY_TLS_CERT` / `_KEY` / `_CA` | *(empty)* | mTLS material (ADR-020) |

Config prefix: `ZYNAX_REGISTRY_` (via `kelseyhightower/envconfig`).
There is **no** `ZYNAX_REGISTRY_DB_*` variable — the service is stateless.

---

## gRPC RPCs

| RPC | Request | Response | Notes |
|-----|---------|----------|-------|
| `SelectAgent` | `SelectAgentRequest` | `SelectAgentResponse` | Exactly one agent + structured rationale; registered only when the CRD informer is enabled |

Proto source: `protos/zynax/v1/scheduler.proto`

The push-era `AgentRegistryService` is gone: unregistered from this server in
#1698 and deleted from `protos/zynax/v1/agent_registry.proto` in #1598 under a
file-scoped `buf breaking` exception (`protos/buf.yaml`, ADR-048 §4). That file
now carries only the `AgentDef` / `CapabilityDef` messages, which stay
permanently — `scheduler.proto` reuses them.

---

## Stateless Invariants (ADR-039)

- **No persistence.** The `Agent` CR in the API server is the only store; the
  informer cache is a derived, rebuildable view.
- **Restart = resync.** A killed pod rebuilds the capability index from the
  API server; nothing is written back except `Agent` `.status` by the
  Lease-elected readiness reconciler (single writer).
- **`SelectAgent` never fails on metrics.** No Prometheus ⇒ degraded
  readiness-filtered round-robin, not an error (ADR-039 §3).
- **Namespaced RBAC.** The informer watches one namespace; `WATCH_NAMESPACE`
  is mandatory when the informer is enabled.

---

## Running Tests

```bash
cd services/agent-registry
GOWORK=off go test ./... -race -timeout 60s

# BDD contract tests (proto-level, separate module)
cd protos/tests
GOWORK=off go test ./scheduler_service/... -v -timeout 60s
```

## Known Limitations

- Selection metrics come from Prometheus only; no direct agent heartbeat.
- Readiness is derived from EndpointSlices, so an agent outside the cluster
  network cannot be marked Ready.
- No authentication middleware — mTLS at the transport only (ADR-020).

See `docs/spdd/480-agent-registry/canvas.md` for the original REASONS Canvas
and `docs/spdd/1674-agent-registry-push-removal/canvas.md` for the removal.
