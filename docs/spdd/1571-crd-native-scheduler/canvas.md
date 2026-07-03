# REASONS Canvas — EPIC M8.C: CRD-native Scheduler (`Agent` CRD as source of truth)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content belongs in `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #1571 · **Milestone:** M8 (v1.0.0)
**Author:** M8 program plan
**Date:** 2026-07-03
**Status:** Implemented

---

## R — Requirements

- **Problem (ADR-039 Context):** agent identity is a push registry with three structural defects —
  (1) **two sources of truth**: the declarative `AgentDef` an operator applies and the runtime
  record an agent pushes can drift, nothing reconciles them; (2) **stale liveness**: an agent that
  crashes without calling `DeregisterAgent` leaves a live-looking record, and task-broker
  round-robins requests into dead endpoints; (3) **off-idiom for Kubernetes**: a bespoke push
  registry re-implements lifecycle, health, watches, and caching the control plane already
  provides — an HA liability and a credibility gap for the M8 CNCF Sandbox submission.
- Dispatch selection is blind round-robin (`idx % len(agents)`) — ignores readiness, load, cost,
  GPU/model fit (rejected as an end-state by ADR-039's rationale table).
- **Definition of done (observable):**
  1. `kubectl apply` of an `Agent` CR is the only registration path; no `RegisterAgent` call
     occurs anywhere in the e2e path.
  2. Crashing an agent pod stops it being selected within one EndpointSlice reconcile
     (stale-liveness BDD green).
  3. Killing the scheduler pod loses no state — the capability index rebuilds from an API-server
     List (resync BDD green).
  4. Stopping Prometheus degrades selection to readiness-filtered round-robin; `SelectAgent`
     never errors from metrics unavailability (degradation BDD green).
  5. `buf breaking` green throughout (additive `scheduler.proto`; registry RPCs deprecated, not
     deleted — hard removal booked for M9).
  6. agent-registry runs with **no database** (Postgres dependency removed from its chart).
  7. First-run flow green on CRD-only discovery, twice back-to-back, both engine legs — the
     ADR-041/#1572 gate discharged before any push-path removal.

## E — Entities

```
Agent CRD (zynax.io/v1alpha1, namespaced, plural agents, short ag)
├── spec: identity + capabilities (today's AgentDef) + scoring hints
│         (selectors, cost, resources, models, protocols) + endpointRef → Service
└── status (subresource, reconciler-owned, low-churn ONLY):
          ready, replicas, conditions          ← derived from EndpointSlice

Scheduler (evolved agent-registry service — stateless)
├── controller-runtime Manager
│   ├── Informer cache over Agent CRs  → CapabilityIndex (capability → {agents})
│   └── ReadinessReconciler (EndpointSlice → Agent.status; Lease leader-elected)
├── ScoringPipeline (ordered, short-circuiting):
│   capability match → hard constraints → readiness → expert target (ADR-028, no fallback)
│   → Prometheus-weighted load/latency (TTL cache) → cost/gpu/model tie-break → locality
└── gRPC: SchedulerService.SelectAgent(capability, policy, constraints)
          → exactly one agent + structured SelectionRationale

Consumers / neighbors
├── task-broker: calls SelectAgent (replaces FindByCapability + round-robin);
│                keeps ADR-028 context-slice binding; TaskRepository (Postgres) untouched
├── api-gateway: kind:AgentDef → RegisterAgent forward removed (CRD-era response)
├── Prometheus (M6): queried at selection time — never stored in CRD status
└── AgentRegistryService (5 RPCs): deprecated → UNIMPLEMENTED (M8), removed (M9);
    AgentDef/CapabilityDef messages reused by scheduler.proto
```

## A — Approach

- **WILL:** make the `Agent` CRD the single source of truth and evolve agent-registry into a
  stateless, informer-backed scheduler, exactly as decided by ADR-039 (Accepted 2026-06-22) and
  de-risked by the KIND-verified spike (`spike/adr-039-crd-scheduler-proof`, 7/7 checks — CRD,
  controller-runtime manager, scorer promoted from `spike/crd-scheduler/`). Additive-first
  sequencing: contract → CRD → informer → scorer/RPC → reconciler → broker cutover → deprecation,
  so `main` stays shippable at every step.
- **WILL:** keep readiness in CRD `status` (reconciler-owned, low-churn) and query live
  load/latency from Prometheus at selection time behind a short-TTL cache, degrading to
  readiness-filtered round-robin on error (ADR-039 §3).
- **WON'T:** store load/latency/queueDepth in CRD status (etcd churn — rejected option); keep a
  push fallback alongside the CRD (dual source of truth — rejected option); delegate capability
  matching to Kubernetes (protected Zynax core, ADR-040 §6); ship admission/conversion webhooks
  (deferred to `v1beta1`, ADR-040 §5); hard-remove the registry RPCs in M8 (deprecation only —
  removal is M9); touch task-broker's Postgres `TaskRepository` (ADR-021 amendment is
  registry-half only).
- **Positioning fit (engine-portability wedge):** only the *transport of identity* changes
  (push → declarative apply; ADR-039's ADR-028 amendment preserves the AgentDef-vs-Workflow split
  and context-slice contract verbatim). Execution stays engine-agnostic WorkflowIR on Temporal or
  Argo (ADR-012/015, reaffirmed by ADR-040 §3) — the scheduler picks *agents*, never engines, and
  the step-6 e2e proves the same flow on both engine legs.
- Governing ADRs: ADR-039 (decision), ADR-040 (boundary), ADR-041 (runtime + first-run gate),
  ADR-021/028 (amended as noted), ADR-001/008 (API server = external infra; etcd ≠ shared app DB),
  ADR-011 (declarative surface), ADR-016/017 (BDD-first · `GOWORK=off`).

## S — Structure

- `protos/zynax/v1/scheduler.proto` — **new file** (additive): `SchedulerService.SelectAgent`;
  reuses `AgentDef`/`CapabilityDef` from `agent_registry.proto`.
- `protos/zynax/v1/agent_registry.proto` — `deprecated` markers on all five RPCs (step 7; no
  field/RPC removal).
- `protos/tests/features/scheduler_service.feature` — new BDD suite (committed before
  implementation, ADR-016).
- `services/agent-registry/` — controller-runtime manager, informer-backed capability index,
  scoring pipeline, readiness reconciler, `SelectAgent` handler; memory/Postgres `AgentRepository`
  adapters leave the production wiring in step 7. Heavy deps (`controller-runtime`, `client-go`)
  enter this module (spike-proven under `GOWORK=off`; fallback: standalone module like
  `cmd/zynax`).
- `services/task-broker/` — `internal/infrastructure/registry_client.go` → scheduler client;
  round-robin removed from `internal/domain/service.go`; context-slice binding
  (`internal/domain/contextslice.go`) untouched.
- `services/api-gateway/` — `kind: AgentDef` push-forward removed (step 7).
- `agents/` — in-repo adapters/fixtures stop self-registering (step 7).
- `helm/` — `Agent` CRD + least-privilege RBAC (Agent CRs get/list/watch, EndpointSlice
  get/list/watch, Lease get/create/update); agent-registry chart drops Postgres.
- `spec/schemas/agent-def.schema.json` — constraint source ported into the CRD's OpenAPI v3
  structural schema.

## O — Operations

> Each step = one story issue = one reviewable PR. 1:1 with the child issues of #1571.

1. **Contract first** — `protos/zynax/v1/scheduler.proto` (`SelectAgent`) + committed BDD
   `.feature`; stubs regenerated; `buf breaking` green. → #1578 ✅
2. **`Agent` CRD + Helm RBAC** — promote the spike CRD with the full OpenAPI v3 schema (ported
   from `spec/schemas/agent-def.schema.json` + scoring hints + `endpointRef`); umbrella ships CRD
   + least-privilege Role; sample CRs green on kind. → #1579 ✅
3. **Informer + capability index** — controller-runtime manager with informer cache over Agent
   CRs; index mirrors today's `capIndex`; resync-on-restart integration test; existing gRPC
   surface untouched. → #1580 ✅
4. **Scoring pipeline + `SelectAgent` live** — promote the spike scorer (ADR-039 §4 order,
   expert-strict); Prometheus at selection time via TTL cache; degradation path; structured
   rationale; step-1 `.feature` passes on kind. → #1581 ✅
5. **Readiness reconciler** — EndpointSlice → `status.{ready,replicas,conditions}`; Lease leader
   election (single writer, select path always-serving); stale-liveness scenario green. → #1582 ✅
6. **task-broker cutover** — `SelectAgent` replaces `FindByCapability` + round-robin; ADR-028
   binding regression green; rationale on the task record; **joint first-run-on-CRD-path e2e
   (named CI scenario, both engine legs, run twice) — the #1572/ADR-041 gate**. → #1583 ✅
7. **Push-path deprecation + stateless registry** — five registry RPCs `deprecated` →
   `UNIMPLEMENTED` with migration pointer; memory/Postgres adapters + api-gateway forward + Helm
   Postgres removed; adapters stop self-registering; migration guide + AGENTS/ARCHITECTURE
   truth-pass; M9 hard-removal issue filed. **Hard gate: step 6's joint e2e green first.** → #1584 ✅

## N — Norms

- Commit hygiene: DCO `Signed-off-by` + `Assisted-by: Claude/<model>` (never `Co-Authored-By` for
  AI); SSH-signed commits; conventional commit types only (`feat|fix|refactor|docs|test|ci|chore`);
  one commit per logical change; PR ≤ 400 lines (excluding generated stubs — `*.pb.go`/`*.pb.py`
  exempt), squash-merge.
- Go services (`docs/patterns/go-service-patterns.md`, `services/AGENTS.md`): hexagonal layout
  (`internal/domain` pure, `internal/api` gRPC, `internal/infrastructure` adapters);
  **`GOWORK=off` for every `go` command** (ADR-017); `go test ./... -race -timeout 60s`; unit
  ≥ 90% on `internal/domain/`; run local golangci-lint pre-commit (stricter than the tools image —
  gosec/goconst deltas).
- Contracts (`protos/AGENTS.md`): PascalCase verb+noun RPCs with full request/response wrappers;
  never remove/renumber/retype a field — deprecate instead; BDD at the gRPC boundary in
  `protos/tests/` **before** implementation (ADR-016); `make generate-protos`, commit the output.
- Config: envconfig with the service prefix (`ZYNAX_REGISTRY_*`) via `libs/zynaxconfig`; no new
  config service (ADR-040 §1).
- Observability: Prometheus `/metrics` + OTel via `libs/zynaxobs`; structured logs to stdout —
  no log aggregation service (ADR-040 §5).
- Runtime smoke before claiming done: exercise the changed flow on kind; run stateful paths twice.

## S — Safeguards (second S)

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /lib:spdd-security-review passed on this file (PASS, 2026-07-03)

### Feature Safeguards
- Never store dynamic metrics (`load`/`latency`/`queueDepth`) in CRD `status` — etcd churn
  (ADR-039 §3, rejected option).
- Never fail `SelectAgent` because Prometheus is unavailable — degrade to readiness-filtered
  round-robin (ADR-039 §3).
- Never fall back on expert-targeted selection — strict isolation, no fallback (ADR-028/ADR-039 §4).
- Never keep a push-registration fallback alongside the CRD — the dual source of truth is the
  disease being cured (ADR-039 rationale, rejected option).
- Never merge step 7 (push-path removal) before step 6's first-run-on-CRD-path e2e is a green
  named CI scenario (ADR-041 / #1572 gate — the ADR-039 load-bearing trade-off).
- Never remove/renumber the deprecated registry RPCs in M8 — `deprecated` + `UNIMPLEMENTED` only;
  hard removal is M9 (`buf breaking` stays green).
- Never let the CRD `spec` subsume the agent's Deployment — identity + capabilities +
  `endpointRef` only (ADR-039 §1).
- Never write CRD status from more than one elected writer; never make the select path depend on
  holding the Lease (ADR-039 Consequences).
- Never touch task-broker's Postgres `TaskRepository` under this epic (ADR-021 amendment is
  registry-half only).
- Never couple selection to a specific engine — the scheduler picks agents, never engines
  (ADR-012/015/040 §3; the wedge is the differentiator).
