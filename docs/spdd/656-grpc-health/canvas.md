# REASONS Canvas — gRPC Health Checking Protocol for K8s-Native Probes

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #656
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-10
**Status:** Draft

---

## R — Requirements

> Problem statement: what breaks or is missing without this feature?

The Zynax gRPC services should expose the standard **gRPC Health Checking Protocol**
(`grpc.health.v1.Health`) so Kubernetes 1.24+ native gRPC probes (`livenessProbe.grpc` /
`readinessProbe.grpc`) and `grpc-health-probe` work with no sidecar, and so rolling restarts
drain in-flight requests gracefully.

**Current state (verified in-repo, 2026-06-10):** the 6 gRPC services
(agent-registry, task-broker, workflow-compiler, engine-adapter, event-bus, memory-service)
**already** register `grpc_health_v1.RegisterHealthServer` and set the overall status to
`SERVING` on startup. api-gateway is an HTTP REST gateway (no external gRPC server) and uses
HTTP `/healthz` probes — out of scope here.

**The remaining gaps** this issue closes:
1. **Graceful-shutdown signalling is absent** — no service sets `NOT_SERVING` before
   `GracefulStop()`, so clients/load-balancers can receive in-flight errors during rolling
   restarts. (verified: `NOT_SERVING` appears in 0 of the service `main.go` files.)
2. **No per-service named serving status** — only the empty `""` overall key is set; probes
   cannot target a specific service name.
3. **No BDD `.feature` scenario** for the Health service (ADR-016 requires it).
4. **No Kubernetes gRPC probe config** in the Helm charts.
5. **No probe documentation** in the service `AGENTS.md` files.

> Definition of done: observable outcomes that confirm delivery.

- Each of the 6 gRPC services sets `NOT_SERVING` (overall + named key) **before** `GracefulStop()`.
- Each service sets a **per-service named** serving status (`zynax.<svc>.v1.<Svc>Service`) to
  `SERVING` on startup.
- A committed BDD `.feature` scenario asserts `Check` returns `SERVING` on a running service
  (ADR-016 — committed before implementation).
- Helm charts expose `livenessProbe.grpc` / `readinessProbe.grpc` for the gRPC services.
- Each service `AGENTS.md` documents the Kubernetes gRPC probe YAML.
- No new external dependency (`google.golang.org/grpc/health` is already transitively present);
  the #655 custom healthcheck binary is **retained** (coexists for docker-compose TCP/HTTP checks).

## E — Entities

> Domain entities and their relationships. Tier 1 only.

```
HealthServer  (google.golang.org/grpc/health.Server)
  ├── OverallStatus      key ""              → SERVING | NOT_SERVING
  ├── NamedStatus        key "zynax.<svc>.v1.<Svc>Service" → SERVING | NOT_SERVING
  └── lifecycle:
        startup   → SetServingStatus(SERVING)   [already done]
        shutdown  → SetServingStatus(NOT_SERVING) → GracefulStop()   [the gap]

Consumers (no Zynax code — external):
  ├── Kubernetes native gRPC probe (1.24+)
  └── grpc-health-probe CLI
```

This is a wiring-layer concern only (`cmd/<svc>/main.go`); no domain logic changes.

## A — Approach

> Solution strategy. What we WILL do and what we WON'T do. Reference governing ADRs.

**WILL:**
- Complete the protocol on the **6 gRPC services**: add graceful `NOT_SERVING` drain before
  `GracefulStop()`, and add the per-service **named** serving status alongside the existing `""`.
- Add a `health.feature` BDD scenario (ADR-016, committed first) and step bindings against the
  gRPC boundary in `protos/tests/`.
- Add `livenessProbe.grpc` / `readinessProbe.grpc` to the gRPC services' Helm charts.
- Document the probe YAML in each service `AGENTS.md`.

**WON'T:**
- Touch api-gateway (HTTP REST gateway — keeps HTTP `/healthz`).
- Remove or replace the #655 custom healthcheck binary (it stays for docker-compose checks).
- Add any new external dependency.
- Implement deep/dependency-aware readiness (e.g. gating SERVING on DB/Temporal reachability) —
  the overall+named SERVING/NOT_SERVING lifecycle is the scope; richer readiness is future work.

Governing ADRs: ADR-016 (BDD at gRPC boundaries, `.feature` before implementation),
ADR-019 (this Canvas before implementation), and the service-layer wiring norms in
`services/AGENTS.md` (ctx-first, wiring-only in `cmd/`).

## S — Structure

> System placement. Services, packages, files, contracts touched.

- `services/<svc>/cmd/<svc>/main.go` for the **6 gRPC services** (agent-registry, task-broker,
  workflow-compiler, engine-adapter, event-bus, memory-service) — named status on startup +
  `NOT_SERVING` drain on shutdown. Wiring layer only; no `internal/domain` changes.
- `protos/tests/features/health.feature` (or per-service) + step bindings in `protos/tests/`.
- `helm/zynax-<svc>/values.yaml` + `templates/deployment.yaml` — gRPC probe config for the 6
  services.
- `services/<svc>/AGENTS.md` — probe YAML examples.
- No `.proto` change: `grpc.health.v1.Health` is provided by the gRPC library, not a Zynax proto.

## O — Operations

> Ordered, concrete, testable implementation steps. Each = one reviewable PR/commit.

1. **BDD first (ADR-016):** add `health.feature` — `Check` returns `SERVING` on a running
   service; commit before any implementation. (`test:`)
2. **Graceful-shutdown `NOT_SERVING` + named status** across the 6 gRPC services: set the
   per-service named key to `SERVING` on startup; set overall + named to `NOT_SERVING` before
   `GracefulStop()`. Wiring-layer edit in each `cmd/<svc>/main.go`. (`feat:`)
3. **BDD step bindings** for `health.feature` at the gRPC boundary (assert SERVING; assert
   NOT_SERVING after shutdown signal where feasible). (`test:`)
4. **Helm gRPC probes:** add `livenessProbe.grpc` / `readinessProbe.grpc` to the 6 gRPC services'
   charts; `helm lint` + `ct lint` clean. (`feat(infra):`)
5. **Docs:** add the Kubernetes gRPC probe YAML to each service `AGENTS.md`. (`docs:`)

## N — Norms

> Cross-cutting standards. Pull from AGENTS.md Hard Constraints + layer norms.

- Commit hygiene: `Signed-off-by` (DCO) + `Assisted-by` trailer; never `Co-Authored-By` for AI.
- Conventional commit types only (feat/fix/refactor/docs/test/ci/chore); scope = directory.
- `GOWORK=off` for every `go` command inside `services/*/` and `protos/tests/` (ADR-017).
- Service layer: `cmd/` is wiring-only; no business logic; ctx threaded first
  (`services/AGENTS.md`). Domain unit coverage ≥ 90% maintained (this change is wiring, so no
  domain regression).
- BDD `.feature` committed before implementation (ADR-016); `buf breaking` stays a CI gate.
- Helm: `helm lint` + `ct lint` clean; chart changes trigger the gated `e2e smoke` workflow.

## S — Safeguards (second S)

> Non-negotiable constraints. Things that MUST NEVER happen.

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file (2026-06-10 — verdict WARN: clean content, Status Draft pending human alignment)

### Feature Safeguards
- Never remove or disable the #655 custom healthcheck binary — docker-compose TCP/HTTP checks
  depend on it; the two health mechanisms coexist.
- Never add a new external dependency — only `google.golang.org/grpc/health` (already transitive).
- Never put health/shutdown logic in `internal/domain` — it is a `cmd/` wiring concern.
- Never gate startup `SERVING` on deep dependency reachability in this issue (out of scope;
  avoids flapping readiness).
- Never call `GracefulStop()` before setting `NOT_SERVING` — the ordering is the whole point of
  the graceful-drain requirement.
