<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax M6 — K8s Production-Ready Planning

> Generated: 2026-06-02  
> Based on live repo state at commit `994efb7` (main).  
> GitHub Milestone: **"K8s Production-Ready (M6)"** (milestone #6, 30 open issues / 2 closed).  
> All `gh` commands, file reads, and live issue data were gathered in this session — nothing assumed from memory.

---

## gh Commands Run (Evidence)

```bash
gh api repos/zynax-io/zynax/milestones --paginate
gh issue list --milestone "K8s Production-Ready (M6)" --state all --limit 200
gh issue list --label "milestone: M6" --state all --limit 200
gh issue list --label "type: epic" --state all --limit 100
gh pr list --state open --limit 50
gh pr list --state merged --limit 20
gh label list --limit 100
```

---

## Section 1 — Rules I Must Obey

> Cited from live repo files; read in this session.

1. **Three-layer separation** — YAML (L1) never imports from L3; contracts (L2) contain no business logic; execution (L3) always behind an interface. Layer violations are hard blockers at review. *(AGENTS.md §Three-Layer Separation)*
2. **PR size** — ≤200 lines ideal; 201–400 acceptable; 401–900 must be justified in PR description; >900 BLOCKED. Exclusions: `*.pb.go`, `*_pb2.py`, lock files, `.github/workflows/`, schema fixtures. One PR per issue; one commit per logical change. *(CLAUDE.md §PR size)*
3. **feat: PRs require a REASONS Canvas** — `docs/spdd/<issue>-<slug>/canvas.md` committed *before* any implementation code; `/spdd-security-review` must PASS before committing. *(AGENTS.md §Five Non-Negotiable Mandates, ADR-019)*
4. **Contracts before code** — `.feature` BDD file committed and CI-green before any implementation touching a gRPC boundary. *(ADR-016, AGENTS.md §Definition of Done)*
5. **GOWORK=off** — mandatory for every `go test` and `go` command inside `services/*/`, `cmd/zynax/`, `protos/tests/`. *(CLAUDE.md, ADR-017)*
6. **No shared databases** — cross-service data access is gRPC only; each service owns its own schema. *(ADR-008, AGENTS.md §API Mandate)*
7. **Language strategy** — Go in `services/`; Python in `agents/`. No Python in `services/`. *(ADR-009)*
8. **Pluggable engines** — never hardcode engine names; always route through `WorkflowEngine` interface. *(ADR-015)*
9. **Commit trailers** — `Signed-off-by: <author> <DCO-email>` + `Assisted-by: Claude/<model>`. Never `Co-Authored-By:` for AI. No `🤖 Generated` lines. *(AGENTS.md §Hard Constraints)*
10. **PR type is CI-enforced** — valid types: `feat`, `fix`, `refactor`, `docs`, `test`, `ci`, `chore`. Rejected: `spec:`, `proto:`, `adr:`, `service:`, `make:`, `security:`. *(AGENTS.md §Hard Constraints)*
11. **ADR for one-way doors** — any decision another engineer would reverse without knowing the rationale gets an ADR. *(CLAUDE.md §Decision-Making Guide)*
12. **State-minimization** — default to stateless; introduce a datastore only where persistence is genuinely unavoidable; only the service that owns the data may hold it. Event-bus durability lives in JetStream (Go service stateless). Memory-service owns Redis (KV) + pgvector exclusively. *(prompt ground rules, ADR-008, ADR-021)*
13. **mTLS for all inter-service gRPC in K8s** — cert-manager issues per-service certificates; Docker Compose stays insecure (dev convenience). *(ADR-020)*
14. **Postgres-backed repos for task-broker + agent-registry** in M6 — in-memory adapters retained as test doubles; each service owns its own schema. *(ADR-021)*
15. **Every Helm chart must include**: `deployment.yaml`, `service.yaml`, `serviceaccount.yaml`, `hpa.yaml`, `pdb.yaml`, `networkpolicy.yaml`. Security context: `runAsNonRoot: true`, `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`. *(infra/AGENTS.md)*

---

## Section 2 — EBUS-DECISION (Deliverable 0)

### 2a. Existing ADR Check

Read `docs/adr/INDEX.md` (21 ADRs, highest is ADR-021). No existing ADR settles "EventBusService gRPC wrapper vs CloudEvents library + direct NATS." The decision is NOT already recorded.

**However**, the repo has substantially pre-committed to Option 1 through:
- `protos/zynax/v1/event_bus.proto` — gRPC `EventBusService` with `Publish`, `Subscribe`, `Unsubscribe` RPCs committed and stable.
- `services/event-bus/tests/features/event_bus.feature` — 6 BDD scenarios committed (ADR-016: contract before code; these must be green before any implementation).
- `services/event-bus/AGENTS.md` — explicitly describes a gRPC service with a `NATSEventBus` infrastructure adapter (`infrastructure/nats.go`).
- ADR-001 — "All services expose capabilities only via versioned gRPC. No shared databases. No cross-service imports." This prohibits engine-adapter calling NATS directly instead of calling EventBusService.
- ADR-013 — Python adapters "Never call platform services via HTTP — only gRPC stubs." They cannot take a NATS client library.

### 2b. Options Analysis

| Dimension | Option 1: Full gRPC EventBusService wrapping JetStream | Option 2: Shared CloudEvents library + direct NATS (no service) | Option 3: Hybrid — thin control-plane service + direct NATS data plane |
|-----------|-------------------------------------------------------|------------------------------------------------------------------|------------------------------------------------------------------------|
| **ADR-001 compliance** | ✅ All inter-service calls via gRPC | ❌ Engine-adapter bypasses the gRPC contract | ⚠ Internal Go services use direct NATS; violates spirit of ADR-001 |
| **ADR-013 compliance** | ✅ Python adapters use gRPC stubs | ❌ Python adapters need NATS client + library | ⚠ Partial; Python still needs gRPC for external ingress |
| **Extra sync hop** | ⚠ One gRPC call before JetStream Publish; latency ~0.1–1 ms on localhost | ✅ No extra hop | ⚠ Policy-path still has a hop |
| **Policy/quota enforcement** | ✅ Natural chokepoint; add gRPC middleware in M7 | ❌ No natural enforcement point; requires sidecar or service mesh | ✅ Service handles external/policy; direct for internal |
| **Multi-namespace isolation** | ✅ Service enforces namespace checks per subscriber_id | ❌ Subject conventions only; no enforcement | ⚠ Enforcement only on external path |
| **DLQ + consumer groups** | ✅ JetStream consumer groups behind service abstraction | ✅ JetStream native (but each consumer manages its own durable) | ⚠ Split responsibility |
| **Proto/BDD already committed** | ✅ All committed (ADR-016 compliant) | ❌ Would require reverting committed contracts | ❌ Would require significant contract revision |
| **Operational cost** | ⚠ One more service to deploy/scale | ✅ No new service | ⚠ One service, but smaller |
| **CNCF Sandbox (M8) alignment** | ✅ Clean gRPC contract = CNCF-friendly | ⚠ Library coupling | ✅ Clean external contract |

### 2c. Recommendation

**Option 1 — Full gRPC EventBusService wrapping JetStream.**

Rationale:
- The proto contract and BDD feature file are already committed. Reversing to a library approach requires reverting accepted contracts, which would violate ADR-016's "contracts before code" invariant in reverse.
- ADR-001 and ADR-013 require gRPC for ALL inter-service and service-to-agent communication. There is no carve-out for "internal" direct NATS.
- The event-bus Go service is effectively stateless — it holds no state of its own. Durability is entirely in JetStream streams and consumer groups. The extra gRPC hop cost (~0.1–1 ms per publish) is acceptable for async event delivery.
- Policy/quota enforcement (EPIC E) and multi-namespace isolation fit naturally as gRPC middleware on EventBusService in M6/M7.

**Implementation shape** (for ADR-022):
- `services/event-bus/internal/infrastructure/nats.go` wraps NATS JetStream publish + consumer.
- All producers (engine-adapter, task-broker, agent-registry) call `EventBusService.Publish` via gRPC.
- All consumers (future observability, memory-service event triggers) call `EventBusService.Subscribe` via gRPC streaming.
- The Go service is a stateless `Deployment` in K8s. NATS JetStream is the StatefulSet.
- The `PublishLifecycleEventActivity` stub in `services/engine-adapter/internal/infrastructure/activities.go` will be wired to the EventBusService gRPC stub when event-bus is implemented.

**The final call is the human's.** The above represents a strong pre-commitment by the repo; deviating would require reverting committed proto + BDD contracts.

### 2d. DECISION Issue Command (for human to run — do NOT execute)

```bash
# Verify label: type: adr-proposal exists (confirmed from gh label list)
# Milestone title: "K8s Production-Ready (M6)" (confirmed from gh milestones API)
gh issue create \
  --title "decision(event-bus): ADR-022 — EventBusService gRPC wrapper vs CloudEvents library + direct NATS" \
  --label "type: adr-proposal,type: docs,area: event-bus,priority: high,milestone: M6" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Decision Needed

Should the event bus be a full gRPC EventBusService wrapping JetStream, or should
services use NATS/JetStream directly with a shared CloudEvents envelope library?

## Why This Is a One-Way Door

The proto contract (`event_bus.proto`) and BDD feature file are already committed
(ADR-016: contracts before code). Reversing to Option 2 or 3 would require reverting
accepted contracts. This decision gates EPIC I scope and the e2e harness (EPIC G).

## Pre-Commitment Evidence

The repo has substantially pre-committed to Option 1:
- `protos/zynax/v1/event_bus.proto` — gRPC `EventBusService` committed
- `services/event-bus/tests/features/event_bus.feature` — 6 BDD scenarios committed
- `services/event-bus/AGENTS.md` — describes gRPC service with `NATSEventBus` adapter
- ADR-001 + ADR-013 — prohibit direct NATS calls from services and Python adapters

## Options

1. **Full gRPC EventBusService** (recommended — repo pre-committed to this)
   Stateless Go service wrapping JetStream. All producers/consumers via gRPC.
2. **Shared CloudEvents library + direct NATS** (no service)
   Would require reverting proto + BDD contracts already committed.
3. **Hybrid**: thin control-plane service for policy/external + direct NATS for internal
   Partially violates ADR-001 (internal direct NATS bypasses gRPC contract).

## Decision Criteria

- ADR-001 (gRPC for all inter-service) · ADR-013 (adapter-first, Python uses gRPC stubs)
- Extra sync hop cost vs policy enforcement benefit
- Multi-namespace isolation in M6 (EPIC D)
- Proto/BDD contracts already committed

## Required Output

An ADR under `docs/adr/ADR-022-event-bus-architecture.md` recording the choice.
EPIC I implementation scope is defined by this outcome.
Mark EPIC I as BLOCKED until this ADR is merged.

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"
```

### 2e. EPIC I Status Until Decision

**EPIC I (event-bus implementation) is BLOCKED-ON: EBUS-DECISION.** Do not write implementation-story REASONS Canvases for EPIC I until ADR-022 is merged. The chart story (A.6: Helm chart for event-bus) may proceed after Option 1 is confirmed, as it depends only on whether a service exists.

---

## Section 3 — Project Status Analysis (Deliverable 1)

### 3a. Milestone Ledger

| Milestone | Status | Evidence |
|-----------|--------|----------|
| M1 — Contracts Foundation | ✅ Complete (v0.1.0) | GitHub milestone "Contracts Foundation" closed; all protos + BDD scenarios merged |
| M2 — Workflow IR | ✅ Complete (v0.1.0) | GitHub milestone "Workflow IR" closed; workflow-compiler shipped |
| M3 — Temporal Execution | ⚠ Partial (v0.2.0) | task-broker + agent-registry delivered under M5.C (#460), NOT M3. CloudEvents is log-only stub. |
| M4 — YAML System + CLI | ⚠ Partial (v0.3.0) | agent-registry (#480) delivered under M5.C, NOT M4. Capability dispatch still incomplete in M4 boundary. |
| **M5 — Adapter Library** | ✅ Complete (v0.4.0) | GitHub milestone "Adapter Library (M5)" closed 2026-05-29. All 7 DoD criteria met. v0.4.0 tag pushed. |
| **M6 — K8s Production-Ready** | 🔜 Next | GitHub milestone open: 30 open / 2 closed. Not started. |
| M7 — Full Observability | Not started | Epics #467/#468/#469 in backlog; canvases drafted |
| M8 — CNCF Sandbox | Not started | |

**Reconciliation:** ROADMAP.md says M5 "Complete ✅" — confirmed by live milestone data (closed). M3/M4 flagged as "Partial" in CLAUDE.md and ROADMAP.md — confirmed: task-broker and agent-registry slipped from M3/M4 to M5.C. README divergence noted in M5.A truth pass (PR#760). No active conflict between sources as of 2026-06-02.

### 3b. M5 Carry-Over Into M6

The following M5 items were explicitly deferred to M6 (from `state/current-milestone.md` and PR#759):

| Issue | Title | M6 Dependency |
|-------|-------|---------------|
| #235 | SBOM generation with syft | EPIC M6.C supply chain hardening |
| #239 | SLSA provenance — cosign signing | EPIC M6.C supply chain hardening |
| #376 | SDK docstrings step 2 | EPIC M6.SDK (blocked on SDK modules) |
| #466 | Stateless Compiler (M6.D epic) | EPIC D; canvas exists |
| #656 | gRPC Health Checking for K8s probes | EPIC A (health probes in charts) |

**Dependency risk:** The M5 e2e demo (`make run-local && zynax apply e2e-demo.yaml`) works via Docker Compose with in-memory task-broker/agent-registry. A **real** Kubernetes e2e (EPIC G) requires:
- All 7 Helm charts (EPIC A) — none exist yet
- Postgres-backed repos (EPIC M6.H #626) — no implementation yet
- mTLS (EPIC M6.B #464) — implementation pending
- event-bus real implementation (EPIC I) — contract-only stub
- memory-service real implementation (EPIC J) — contract-only stub

M6 cannot be "genuinely production-ready" without all of the above. The dependency chain is: EPIC A → EPIC I/J → EPIC G.

### 3c. Codebase Reality Check (per-area)

| Area | Status | Evidence |
|------|--------|----------|
| **Helm charts** | ❌ None | `infra/` contains only `docker-compose/`, `docker/`, `tests/`. No `helm/` directory. Issue #241 (skeleton) is open. |
| **ArgoEngine adapter** | ❌ None | `services/engine-adapter/` has `TemporalEngine` only. No Argo-related Go files found. |
| **gRPC health checking** | ❌ None | Issues #656 + #74 open. `services/AGENTS.md` anti-pattern table says "Implementing gRPC server without `grpc_health_v1.RegisterHealthServer`" is a mistake. Not yet fixed. |
| **Health probes (K8s semantics)** | 🟡 Scaffold | Issue #487 open; canvas at `docs/spdd/463-health-probes/canvas.md`; no implementation. |
| **mTLS (inter-service)** | 🟡 Scaffold | ADR-020 written; canvas at `docs/spdd/464-mtls/canvas.md`; issue #488 open; no implementation. |
| **Supply chain hardening** | 🟡 Scaffold | Canvas at `docs/spdd/465-supply-chain/canvas.md`; issues #489/#235/#239 open; no implementation. |
| **Stateless compiler** | 🟡 Scaffold | Canvas at `docs/spdd/466-stateless-compiler/canvas.md`; issue #490 open; no implementation. |
| **Multi-namespace (compiler)** | ❌ None | No issue, no design, not tracked. Roadmap item only. |
| **Policy enforcement** | ❌ None | Issue #580 (per-IP rate limit) is the only related item. No routing-policy/capability-quota design. |
| **zynax-sdk PyPI** | 🟡 Partial | `pyproject.toml` has `name = "zynax-sdk"`, `version = "0.1.0"`, hatchling build backend. No trusted publisher, no GitHub Actions publish workflow, no `[tool.hatch.publish]` section. |
| **event-bus** | 🟡 Contract-only | `protos/zynax/v1/event_bus.proto` ✅; `services/event-bus/tests/features/event_bus.feature` ✅ (6 scenarios); `services/event-bus/AGENTS.md` ✅. No `cmd/`, `internal/`, `go.mod` in `services/event-bus/`. `PublishLifecycleEventActivity` in engine-adapter is a `slog.Debug` log-only stub. |
| **memory-service** | 🟡 Contract-only | `protos/zynax/v1/memory.proto` ✅ (10 RPCs); `services/memory-service/tests/features/memory_service.feature` ✅ (6 scenarios); `services/memory-service/AGENTS.md` ✅. No `cmd/`, `internal/`, `go.mod` in `services/memory-service/`. |
| **Postgres-backed repos** | 🟡 Designed | ADR-021 written; epic #626 open; no implementation. In-memory repos still in use. |
| **Platform config convergence** | 🟡 Designed | Epic #670 open; issues #667/#668/#669 open; no implementation. |
| **AsyncAPI channels** | ✅ Complete | 13 channels defined in `spec/asyncapi/zynax-events.yaml`: 6 workflow events, 4 task events, 3 agent events. |
| **Prometheus metrics** | ❌ None | Issue #491 open. No `/metrics` endpoint in any service. |
| **e2e harness (Helm/kind)** | ❌ None | No issue exists. Proposed in this document. |

### 3d. Risk Table

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|-----------|--------|------------|
| R1 | EPIC A (Helm charts) balloons: 7 service charts + 4 subcharts (Temporal, NATS, Postgres, Redis) + cert-manager + umbrella. Could push to 30+ PRs. | High | High | One chart per PR; reusable `_helpers.tpl` library chart first to share templates |
| R2 | memory-service store choice (Redis KV + pgvector) adds two new cluster dependencies alongside Postgres (ADR-021). Operators must manage Redis + Postgres. | Medium | Medium | Evaluate Redis Stack (vector module) to consolidate to one store; flag in EPIC J Canvas. If not adopted, scope note in charts. |
| R3 | EBUS-DECISION delays EPIC I. If decision takes >1 sprint, e2e harness (EPIC G) is blocked on real event flow. | Medium | High | Stage EPIC G to use mock/log event bus for first iteration; wire real event-bus in second iteration |
| R4 | Postgres-backed repos (ADR-021/#626) for task-broker + agent-registry have no implementation. This is a prerequisite for EPIC J (memory-service pgvector needs Postgres) and M6.H Helm charts. | High | High | Start M6.H (#626) implementation immediately (first sprint) before EPIC J |
| R5 | ArgoEngine adapter (EPIC B) requires Argo Workflows cluster dependency + Helm chart. Argo's CRD footprint is large. | Low | Medium | Pin Argo chart version; use `argo-workflows` subchart; validate against kind cluster in CI |
| R6 | Policy enforcement (EPIC E) has no proto contract. Needs new messages in proto before implementation — risks proto churn mid-milestone. | Medium | High | Design proto first (ADR for policy schema); commit `.feature` before any implementation |
| R7 | Tier-2 security flags for EPIC I (event-bus): subscriber authentication, topic authorization, DLQ retention are sensitive config details. Canvas must not expose these. | Medium | High | `/spdd-security-review` mandatory before committing any EPIC I canvas; move auth/retention config to `canvas.private.md` |
| R8 | e2e harness (EPIC G) with kind + Helm is complex CI infrastructure. Local developers may not be able to run it on their machines without Docker resources. | Medium | Medium | Gate it as a separate CI job (not on every PR); document minimum kind config; use ephemeral k3d as alternative |

---

## Section 4 — Proposed M6 Scope (Deliverable 2)

### 4a. Roadmap Items (must ship)

| Item | In/Out | Rationale |
|------|--------|-----------|
| Helm charts for all 7 services + Temporal/NATS/Postgres/Redis dependencies | ✅ **IN** | Core M6 definition |
| ArgoEngine adapter | ✅ **IN** | Core M6 definition |
| K8s Runtime Provider (HPA, PDB, NetworkPolicy) | ✅ **IN** (folded into each chart) | infra/AGENTS.md mandates these in every chart; not a separate service |
| Multi-namespace support in workflow-compiler | ✅ **IN** | Core M6 definition |
| Policy enforcement (routing policies, rate limits, capability quotas) | ✅ **IN** | Core M6 definition; needs proto design first |
| zynax-sdk PyPI publish | ✅ **IN** | Core M6 definition; pyproject.toml ready; needs trusted publisher + CI workflow |

### 4b. Service Implementations (human directive)

| Service | Decision | Rationale |
|---------|----------|-----------|
| event-bus (NATS JetStream) | ✅ **IN** — CONDITIONAL on EBUS-DECISION (Deliverable 0) | Contract-only today; real e2e requires live events |
| memory-service (Redis KV + pgvector) | ✅ **IN** | Contract-only today; real e2e requires persisted context |

**State-minimization note (vs ADR-021 conflict):**
The prompt says "only memory-service (and its store) is a StatefulSet/PVC; everything else is a stateless Deployment." ADR-021 (accepted) requires Postgres-backed repositories for task-broker and agent-registry in M6. The repo wins: Postgres is a cluster-level StatefulSet (Bitnami subchart) shared by task-broker, agent-registry, and memory-service's vector plane — each with its own schema (ADR-008). The Go service Deployments for task-broker/agent-registry ARE stateless; only the backing Postgres (and Redis) are StatefulSets. The correct M6 K8s topology is:

```
StatefulSets/PVCs:
  - Temporal (PostgreSQL + Cassandra or bundled) — existing
  - NATS JetStream                               — new (event-bus backend)
  - Postgres 16-alpine (Bitnami)                 — new (task-broker schema + agent-registry schema + memory-service pgvector schema)
  - Redis (memory-service KV plane only)          — new, dedicated Redis instance

Stateless Deployments (all Go services + adapters):
  - api-gateway, workflow-compiler, engine-adapter, task-broker, agent-registry, event-bus, memory-service
  - http-adapter, git-adapter, ci-adapter, llm-adapter, langgraph-adapter
```

### 4c. Proposed Additions (evaluate each)

| Addition | Decision | Rationale |
|----------|----------|-----------|
| **Real e2e harness** (kind/k3d + Helm, reference workflows on Temporal AND Argo, assert CloudEvents off JetStream, assert memory-service reads) | ✅ **IN M6** | Without this, M6 is "charts exist" not "K8s production-ready." This is the DoD gate. |
| **Smoke/e2e GitHub Actions job** (gated, not on every PR) | ✅ **IN M6** | Validates charts + services together; triggers on `infra/`, `services/`, `engine-adapter/` changes |
| **Upgrade/rollback test** (helm upgrade --atomic, values matrix Temporal-backed vs Argo-backed) | ✅ **IN M6** | Production readiness requires upgrade confidence |
| **Failure-path e2e** (capability timeout, guard-rejected, adapter 5xx → workflow.failed CloudEvent asserted) | ✅ **IN M6** | Critical for operator confidence; included in EPIC G scope |
| **Operational baseline** (liveness/readiness/startup probes + grpc_health_probe in all charts) | ✅ **IN M6** | Required for K8s-native health; already tracked as #656/#463; folded into EPIC A/C |
| **Full observability** (distributed traces, Grafana dashboards, OTel) | ❌ **DEFER to M7** | ROADMAP.md §M7 explicitly owns this. Only liveness/readiness probes + Prometheus annotations on Deployment go into M6 charts (already required by chart template). |
| **Postgres-backed repos for task-broker + agent-registry** | ✅ **IN M6** | ADR-021 accepted; prerequisite for horizontal scale and EPIC J |
| **gRPC Health Checking Protocol** | ✅ **IN M6** | Required for K8s-native `grpc_health_probe`; tracked as #656; folded into EPIC A/C |
| **Platform config convergence** (#670) | ✅ **IN M6** | Prerequisite for clean chart values and Dockerfile parameterization |
| **mTLS inter-service** (#464) | ✅ **IN M6** | ADR-020 accepted; required for production security; cert-manager in EPIC A charts |
| **Supply chain hardening** (#465) | ✅ **IN M6** | SBOM + cosign already tracked; deferred from M5 |
| **StatefulSet/PVC**: only Postgres + Redis + NATS are StatefulSets; all Go services are Deployments | ✅ **IN M6** (architecture constraint) | State-minimization principle + ADR-021 |

---

## Section 5 — Epic / Story Decomposition (Deliverable 3)

> Epic lettering follows the prompt's A–J scheme. Existing M6 repo epics are mapped and referenced.
> Canvas required = yes for all `feat:` type stories.
> `.feature` required = yes for any story introducing a new gRPC boundary.
> PR size bucket: S = ≤200 lines; M = 201–400 (acceptable); L = 401–900 (justify in description).

---

### EPIC A — Helm Charts + K8s Runtime Provider (NEW EPIC)

> Covers roadmap items: Helm charts for all services, K8s Runtime Provider (HPA/PDB/NetworkPolicy folded into each chart), Temporal + NATS + Postgres + Redis + cert-manager dependencies.
> Absorbs: #241 (skeleton story), #242 (CI lint gate), #487 (health probes), #656 (gRPC health), infra/AGENTS.md requirements.

| Story | PR Title | Type | Size | Canvas? | .feature? | Deps |
|-------|----------|------|------|---------|-----------|------|
| A.0 | `feat(infra): shared Helm _helpers.tpl library chart` | feat | S | ✅ | — | None (first) |
| A.1 | `feat(infra): Helm chart for api-gateway` | feat | S | ✅ | — | A.0 |
| A.2 | `feat(infra): Helm chart for workflow-compiler` | feat | S | ✅ | — | A.0 |
| A.3 | `feat(infra): Helm chart for engine-adapter` | feat | S | ✅ | — | A.0 |
| A.4 | `feat(infra): Helm chart for task-broker` | feat | S | ✅ | — | A.0, M6.H |
| A.5 | `feat(infra): Helm chart for agent-registry` | feat | S | ✅ | — | A.0, M6.H |
| A.6 | `feat(infra): Helm chart for event-bus` | feat | S | ✅ | — | A.0, EBUS-DECISION |
| A.7 | `feat(infra): Helm chart for memory-service (StatefulSet + Redis + pgvector)` | feat | M | ✅ | — | A.0, EPIC J |
| A.8 | `feat(infra): NATS JetStream subchart + stream config` | feat | S | ✅ | — | A.0 |
| A.9 | `feat(infra): Postgres 16 subchart (task-broker + agent-registry + memory schemas)` | feat | S | ✅ | — | A.0, M6.H |
| A.10 | `feat(infra): Temporal Helm dependency in umbrella chart` | feat | S | ✅ | — | A.0 |
| A.11 | `feat(infra): cert-manager ClusterIssuer + per-service Certificate resources` | feat | S | ✅ | — | A.0, EPIC C |
| A.12 | `ci: helm lint gate on infra/ changes in CI` (#242) | ci | S | — | — | A.0 |
| A.13 | `docs(infra): environment parity manifest — dev/staging/prod differences` (#243) | docs | S | — | — | A.1–A.7 |

**EPIC A note on prompt's EPIC C:** The "Kubernetes Runtime Provider (HPA/PDB/NetworkPolicy)" is FOLDED into each chart story (A.1–A.7) per infra/AGENTS.md, which mandates these resources in every chart. There is no separate K8s Runtime Provider service. Stories A.1–A.7 each include HPA + PDB + NetworkPolicy templates sourced from docs/patterns/helm-charts.md.

---

### EPIC B — ArgoEngine Adapter (NEW)

> Roadmap: "ArgoEngine adapter (WorkflowEngine impl behind the existing interface)"
> ADR-015 (pluggable engines): never hardcode engine names; implement behind WorkflowEngine interface.
> Proto/config + .feature before implementation.

| Story | PR Title | Type | Size | Canvas? | .feature? | Deps |
|-------|----------|------|------|---------|-----------|------|
| B.1 | `feat(protos): ArgoEngine config message + Argo workflow options in EngineAdapterService` | feat | S | ✅ | ✅ | EPIC A (A.3 chart) |
| B.2 | `feat(engine-adapter): ArgoEngine — Submit + Signal implementation` | feat | M | ✅ | — | B.1 |
| B.3 | `feat(engine-adapter): ArgoEngine — Query + Cancel + Watch implementation` | feat | M | ✅ | — | B.2 |
| B.4 | `feat(engine-adapter): ArgoEngine — wiring, multi-engine dispatch, config flag` | feat | S | ✅ | — | B.3 |

**PR split rationale for EPIC B:** Mirroring the prompt's suggested split: (1) proto/config + .feature, (2) Submit/Signal, (3) Query/Cancel/Watch, (4) wiring + e2e. Each stays ≤400 lines excluding generated stubs.

---

### EPIC C — Health Checking + mTLS Security (repo M6.A #463 + M6.B #464)

> Canvases already exist: `docs/spdd/463-health-probes/canvas.md` and `docs/spdd/464-mtls/canvas.md`.
> Stories can proceed directly to `/spdd-generate` once canvas status is `Aligned`.

| Story | PR Title | Type | Area | Size | Canvas? | Deps |
|-------|----------|------|------|------|---------|------|
| C.1 | `feat(services): implement grpc.health.v1 in all 7 Go services` (#656/#74) | feat | all services | M | ✅ | None |
| C.2 | `feat(api-gateway): split startup/readiness/liveness probes` (#487) | feat | api-gateway | S | ✅ (existing #463) | C.1 |
| C.3 | `feat(engine-adapter): split startup/readiness/liveness probes` | feat | engine-adapter | S | ✅ | C.1 |
| C.4 | `feat(infra): mTLS env-var cert paths in all Go service configs` (#488) | feat | infra | M | ✅ (existing #464) | A.11 |
| C.5 | `feat(infra): mTLS gRPC client/server TLS credential wiring` | feat | infra | M | ✅ | C.4 |

---

### EPIC D — Stateless Compiler + Multi-Namespace (repo M6.D #466 + roadmap multi-namespace)

> Canvas exists: `docs/spdd/466-stateless-compiler/canvas.md`. Multi-namespace is new.

| Story | PR Title | Type | Area | Size | Canvas? | .feature? | Deps |
|-------|----------|------|------|------|---------|-----------|------|
| D.1 | `refactor(workflow-compiler): drop in-memory IR store, make CompileWorkflow stateless` (#490) | refactor | workflow-compiler | S | ✅ (existing #466) | — | None |
| D.2 | `feat(protos): add namespace field to WorkflowIR + CompileWorkflow request` | feat | protos | S | ✅ | ✅ | D.1 |
| D.3 | `feat(workflow-compiler): multi-namespace routing — compile to namespace-scoped IR` | feat | workflow-compiler | M | ✅ | — | D.2 |
| D.4 | `feat(api-gateway): propagate namespace from request to compiler + engine` | feat | api-gateway | S | ✅ | — | D.3 |

---

### EPIC E — Policy Enforcement (NEW)

> Roadmap: "routing policies, rate limits, capability quotas"
> Issue #580 (per-IP rate limit on api-gateway) is the only existing related story.
> Needs proto design first (new Policy enforcement messages).

| Story | PR Title | Type | Area | Size | Canvas? | .feature? | Deps |
|-------|----------|------|------|------|---------|-----------|------|
| E.1 | `feat(protos): Policy enforcement messages — RoutingPolicy, RateLimit, CapabilityQuota` | feat | protos | S | ✅ | ✅ | None |
| E.2 | `feat(api-gateway): per-IP token-bucket rate limit on POST /api/v1/apply` (#580) | feat | api-gateway | S | ✅ | — | E.1 |
| E.3 | `feat(workflow-compiler): routing policy enforcement — capability quota gate` | feat | workflow-compiler | M | ✅ | — | E.1, D.1 |
| E.4 | `feat(engine-adapter): capability quota check before DispatchCapabilityActivity` | feat | engine-adapter | S | ✅ | — | E.3 |

---

### EPIC F — zynax-sdk PyPI + Supply Chain (roadmap SDK + repo M6.C #465)

> Canvas exists: `docs/spdd/465-supply-chain/canvas.md`.
> SDK pyproject.toml has name + version + hatchling. Needs: trusted publisher setup, publish CI workflow, version bump to 0.1.1 or 1.0.0a1.

| Story | PR Title | Type | Area | Size | Canvas? | Deps |
|-------|----------|------|------|------|---------|------|
| F.1 | `feat(agents): zynax-sdk PyPI packaging — trusted publisher + TestPyPI dry-run` | feat | agents/sdk | S | ✅ | None |
| F.2 | `ci: zynax-sdk publish workflow — release-triggered PyPI publish` | ci | agents/sdk | S | — | F.1 |
| F.3 | `ci: cosign signing + SBOM generation for release artifacts` (#489/#235/#239) | ci | ci | M | — (M6.C canvas exists) | None |
| F.4 | `docs(agents): SDK docstrings step 2 — remaining public modules` (#376) | docs | agents/sdk | S | — | F.1 |

---

### EPIC G — Real e2e Harness (PROPOSED — IN M6)

> kind/k3d + Helm + reference workflows on Temporal AND Argo.
> Assert CloudEvents off JetStream (not mocked) and memory-service reads.
> Failure-path and upgrade/rollback tests.

| Story | PR Title | Type | Area | Size | Canvas? | Deps |
|-------|----------|------|------|------|---------|------|
| G.1 | `test: kind cluster bootstrap script + helmfile for full stack deploy` | test | infra | M | ✅ | All EPIC A charts |
| G.2 | `test: e2e happy-path — code-review.yaml via Temporal, assert CloudEvents off JetStream` | test | infra | M | ✅ | G.1, EPIC I wired |
| G.3 | `test: e2e Argo path — code-review.yaml via ArgoEngine, assert CloudEvents` | test | infra | S | ✅ | G.2, EPIC B wired |
| G.4 | `test: e2e failure-path — capability timeout → workflow.failed CloudEvent asserted` | test | infra | S | ✅ | G.2 |
| G.5 | `test: Helm upgrade/rollback test (helm upgrade --atomic, Temporal→Argo values matrix)` | test | infra | S | ✅ | G.1 |

---

### EPIC H — e2e CI Gate (PROPOSED — IN M6)

> Smoke job in GitHub Actions on PRs touching `infra/`, `services/`, `engine-adapter/`.
> Gated (not on every PR) to respect CI time.

| Story | PR Title | Type | Area | Size | Canvas? | Deps |
|-------|----------|------|------|------|---------|------|
| H.1 | `ci: e2e smoke job — kind + helm stack, gated on infra/services/ changes` | ci | ci | M | — | G.1–G.4 |
| H.2 | `ci: matrix values test — Temporal-backed vs Argo-backed helm values` | ci | ci | S | — | H.1 |

---

### EPIC I — event-bus Implementation (**BLOCKED-ON: EBUS-DECISION**)

> **Do not write implementation-story REASONS Canvases until ADR-022 is merged.**
> Scope and even whether this is a "service" depends on the EBUS-DECISION outcome.
> Assuming Option 1 (full gRPC EventBusService — recommended), the split below applies.
> The service stays stateless Go; durability lives in JetStream streams/consumers. No DB.

| Story | PR Title | Type | Area | Size | Canvas? | .feature? | Deps |
|-------|----------|------|------|------|---------|-----------|------|
| I.0 | `docs: ADR-022 — EventBusService gRPC wrapper vs CloudEvents library + direct NATS` | docs | event-bus | S | — | — | EBUS-DECISION issue |
| I.1 | `feat(event-bus): service scaffold — go.mod, cmd/, domain/, NATS JetStream client bootstrap` | feat | event-bus | S | ✅ | — | I.0, A.8 (NATS chart) |
| I.2 | `feat(event-bus): Publish path — JetStream stream create + event publish` | feat | event-bus | M | ✅ | — | I.1 |
| I.3 | `feat(event-bus): Subscribe path — durable consumer group + server-streaming gRPC` | feat | event-bus | M | ✅ | — | I.2 |
| I.4 | `feat(event-bus): Unsubscribe + DLQ + retry-backoff wiring` | feat | event-bus | M | ✅ | — | I.3 |
| I.5 | `feat(engine-adapter): wire PublishLifecycleEventActivity to EventBusService gRPC` | feat | engine-adapter | S | ✅ | — | I.4 |
| I.6 | `test: BDD step implementations for event_bus.feature (6 scenarios)` | test | event-bus | M | — | — | I.4 |

**If Option 2 is chosen:** I.1–I.4 become a shared `libs/zynax-events` CloudEvents library PR. I.5 changes to wiring that library directly. EPIC A does NOT get an event-bus Deployment chart. Mark these as TBD until ADR-022 merged.

---

### EPIC J — memory-service Implementation (NEW)

> The ONLY stateful Go service (but the Go service itself is a stateless Deployment;
> Redis and pgvector are the StatefulSets).
> Redis for KV plane; Postgres+pgvector for vector plane (per AGENTS.md pre-decision).
> State-minimization note: Redis is dedicated to memory-service. Postgres is shared
> with task-broker + agent-registry schemas (ADR-008: separate schemas, same cluster Postgres).
> Prerequisite: EPIC M6.H (#626) — Postgres-backed repos — provides the Postgres instance.

| Story | PR Title | Type | Area | Size | Canvas? | .feature? | Deps |
|-------|----------|------|------|------|---------|-----------|------|
| J.0 | `feat(infra): Postgres-backed task-broker repository (pgx/v5 + migrations)` (M6.H #626, step 1) | feat | task-broker | M | ✅ | — | A.9 (Postgres chart) |
| J.1 | `feat(infra): Postgres-backed agent-registry repository (pgx/v5 + migrations)` (M6.H #626, step 2) | feat | agent-registry | M | ✅ | — | J.0 |
| J.2 | `feat(memory-service): service scaffold — go.mod, cmd/, domain KV+Vector interfaces` | feat | memory-service | S | ✅ | — | None |
| J.3 | `feat(memory-service): Redis KV infrastructure adapter — Set/Get/Delete/ListKeys/MGet/MSet/DeleteNamespace` | feat | memory-service | M | ✅ | — | J.2 |
| J.4 | `feat(memory-service): pgvector infrastructure adapter — StoreVector/QueryVector/DeleteVector` | feat | memory-service | M | ✅ | — | J.2, J.0 (Postgres) |
| J.5 | `feat(memory-service): namespace TTL enforcement + workflow_id isolation` | feat | memory-service | S | ✅ | — | J.3, J.4 |
| J.6 | `feat(memory-service): gRPC handler wiring — all 10 RPCs, integration tests` | feat | memory-service | M | ✅ | — | J.5 |
| J.7 | `test: BDD step implementations for memory_service.feature (6 scenarios)` | test | memory-service | M | — | — | J.6 |

**Single-store evaluation note:** `services/memory-service/AGENTS.md` pre-decided Redis (KV) + Postgres/pgvector (vector). Before committing the J.3/J.4 Canvas, evaluate whether Redis Stack (with vector module) can serve both planes from one store. If yes, J.3 and J.4 merge to one Redis-Stack adapter and one fewer cluster dependency. Document the evaluation in the J.2 Canvas S-Safeguards section.

**EPIC J Canvas Safeguards section MUST record:**
- Single-store evaluation result (Redis Stack vs Redis+pgvector)
- State-minimization justification: memory-service needs persistence for cross-invocation context (twelve-factor principle: agents must not hold shared state internally)
- Why JetStream cannot meet the KV/vector contract (JetStream is message-passing, not a synchronous KV store with similarity search)
- memory-service Redis is a **dedicated** Redis instance — no other service may use it
- Postgres instance is shared at cluster level, but memory-service uses its own schema (ADR-008)
- K8s Structure: memory-service Go pod = stateless Deployment; Redis = StatefulSet with PVC

---

### Existing M6 Epics (referenced, not new)

| Repo Epic | Status | Canvas | Stories |
|-----------|--------|--------|---------|
| M6.F #670 Platform Config Convergence | 🔜 Not started | — | #667 (shared config lib), #668 (Dockerfile template), #669 (go.mod version gate) |
| M6.H #626 Postgres-backed repos | 🔜 Not started | — | Become J.0–J.1 above |
| M6.A #463 Health Probe Semantics | 🟡 Canvas exists | `docs/spdd/463-health-probes/canvas.md` | Becomes C.2–C.3 above |
| M6.B #464 mTLS | 🟡 Canvas exists | `docs/spdd/464-mtls/canvas.md` | Becomes C.4–C.5 above |
| M6.C #465 Supply Chain | 🟡 Canvas exists | `docs/spdd/465-supply-chain/canvas.md` | Becomes F.3 above |
| M6.D #466 Stateless Compiler | 🟡 Canvas exists | `docs/spdd/466-stateless-compiler/canvas.md` | Becomes D.1 above |

---

