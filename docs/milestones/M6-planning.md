<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax M6 — K8s Production-Ready Planning

> Generated: 2026-06-02 · Last updated: 2026-06-03 (GHCR hygiene issues #865–#869 + M6.Build #837 wired)  
> Based on live repo state at commit `994efb7` (main).  
> GitHub Milestone: **"K8s Production-Ready (M6)"** (milestone #6, 30 open issues / 2 closed).  
> All `gh` commands, file reads, and live issue data were gathered in this session — nothing assumed from memory.

## Delivery Progress

| Story | Issue | EPIC | PR | Status |
|-------|-------|------|----|--------|
| A.1 feat(api-gateway): split startup/readiness/liveness probes | #487 | M6.A #463 | #821 | ✅ Merged |
| D.1 refactor(workflow-compiler): drop in-memory IR store — stateless compiler | #490 | M6.D #466 | #774 | ✅ Merged |
| I.0 docs: ADR-022 EventBus architecture decision | #764 | M6.I #772 | #822 | ✅ Merged |
| B.1 feat(infra): mTLS env-var cert paths + gRPC credential wiring for all services | #488 | M6.B #464 | #831 | ✅ Merged |
| C.1 ci: cosign + SBOM + multi-arch for all release workflows | #489 | M6.C #465 | #833 | ✅ Merged |
| I.1 feat(event-bus): service scaffold | #823 | M6.I #772 | — | ⬜ Pending canvas |
| I.2 feat(event-bus): Publish path | #824 | M6.I #772 | — | ⬜ Pending I.1 |
| I.3 feat(event-bus): Subscribe path | #825 | M6.I #772 | — | ⬜ Pending I.2 |
| I.4 feat(event-bus): Unsubscribe + DLQ + retry | #826 | M6.I #772 | — | ⬜ Pending I.3 |
| I.5 feat(engine-adapter): wire lifecycle activity | #827 | M6.I #772 | — | ⬜ Pending I.4 |
| I.6 test: BDD steps for event_bus.feature | #828 | M6.I #772 | — | ⬜ Pending I.4 |

**M6 Infra / Tooling** (process health — not feature EPICs; exempt from SPDD)

| Story | Issue | Area | PR | Status |
|-------|-------|------|-----|--------|
| chore(ci): add bump-ci-runner script and make target | [#843](https://github.com/zynax-io/zynax/issues/843) | CI tooling | #848 | ⬜ Open |
| ci(infra): open ci-runner bump issue after tools-image build | [#844](https://github.com/zynax-io/zynax/issues/844) | CI automation | #849 | ⬜ Open |
| chore(claude): rewrite /resume-m6 — FF discipline, doc-PR path, branch cleanup | [#845](https://github.com/zynax-io/zynax/issues/845) | Slash commands | #850 | ⬜ Open |
| docs(contributing): record rebase-merge / branch-delete / no-reopen policy | [#846](https://github.com/zynax-io/zynax/issues/846) | Docs | #851 | ⬜ Open |
| docs(adr): ADR-023 — restrict direct pushes to main; rebase-merge only | — | Docs/ADR | #847 | ⬜ Open |

**M6.Images — Single source of truth for container-image references** (EPIC #855; canvas `docs/spdd/855-images-sot/canvas.md` — Status: **Aligned**)

| Story | Issue | Area | PR | Status |
|-------|-------|------|-----|--------|
| O1 chore(ci): images/images.yaml — schema + initial population | [#856](https://github.com/zynax-io/zynax/issues/856) | CI tooling | — | ⬜ Open |
| O2 feat(zynax-ci): images sync + check subcommands | [#857](https://github.com/zynax-io/zynax/issues/857) | CI tooling | — | ⬜ Open |
| O3 ci: wire drift-check into pr-checks.yml + ci.yml | [#858](https://github.com/zynax-io/zynax/issues/858) | CI | — | ⬜ Open (ships with O2) |
| O4 chore(infra): Dockerfile ARG migration | [#859](https://github.com/zynax-io/zynax/issues/859) | Infra | — | ⬜ Open |
| O5 chore(ci): bump flow rewrite | [#860](https://github.com/zynax-io/zynax/issues/860) | CI tooling | — | ⬜ Open |
| O6 docs: single source of truth propagation | [#861](https://github.com/zynax-io/zynax/issues/861) | Docs | — | ⬜ Open |
| O7 docs: ADR-024 | [#862](https://github.com/zynax-io/zynax/issues/862) | Docs/ADR | — | ⬜ Open |

**M6.Images — GHCR Package Hygiene** (attached to EPIC #855; SPDD-exempt; delivery order: #868 → #865 → #866 → #867 → #869)

| Story | Issue | Area | PR | Status |
|-------|-------|------|-----|--------|
| docs(adr): ADR-025 — SLSA provenance attestation keep vs disable | [#868](https://github.com/zynax-io/zynax/issues/868) | Docs/ADR | — | ⬜ Open |
| ci: add OCI manifest annotations — fix "no description" on GHCR | [#865](https://github.com/zynax-io/zynax/issues/865) | CI/Infra | — | ⬜ Open |
| ci: description-present gate + size-budget check in publish steps | [#866](https://github.com/zynax-io/zynax/issues/866) | CI | — | ⬜ Open |
| chore(ci): GHCR retention cap — keep last 5 builds per image | [#867](https://github.com/zynax-io/zynax/issues/867) | CI | — | ⬜ Open |
| docs: document unknown/unknown — expected SLSA provenance | [#869](https://github.com/zynax-io/zynax/issues/869) | Docs/Infra | — | ⬜ Open |

**M6.Build — Native multi-arch build pipeline** (EPIC [#837](https://github.com/zynax-io/zynax/issues/837); SPDD-exempt; no canvas required)

| Story | Issue | Area | PR | Status |
|-------|-------|------|-----|--------|
| ci(infra): migrate release.yml service builds to native arm64 — no QEMU | — | CI | — | ⬜ Open |
| ci(infra): migrate tools-image.yml to native arm64 — no QEMU | — | CI | — | ⬜ Open |
| ci(infra): add Python adapter images to multi-arch release pipeline | — | CI/Infra | — | ⬜ Open |
| ci(infra): audit and minimize final image sizes | [#841](https://github.com/zynax-io/zynax/issues/841) | CI/Infra | — | ⬜ Open |

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

## Section 6 — SPDD Command Runbook (Deliverable 4)

> Commands use the **exact signatures** read from `.claude/commands/`:
>
> - `/spdd-analysis <issue-number | issue-URL | feature-description>`
> - `/spdd-story <issue-number | issue-URL | raw feature description>`
> - `/spdd-reasons-canvas <issue-number | feature-description>` → writes `docs/spdd/<issue>-<slug>/canvas.md`
> - `/spdd-security-review <path/to/canvas.md>` → PASS required before commit
> - `/spdd-generate <path/to/aligned-canvas.md>` → Canvas status must be `Aligned`; executes ONE Operations step
> - `/spdd-api-test <path/to/canvas.md>` → generates BDD .feature + grpcurl scenarios
> - `/spdd-sync <path/to/canvas.md>` → syncs Canvas after refactoring (sets status: Synced)
> - `/spdd-prompt-update <path/to/canvas.md>` → requirements changed → update Canvas first (resets to Draft)
>
> `ci:`, `test:`, `refactor:`, `docs:`, `chore:` stories are **SPDD-exempt** (no Canvas required).

---

### EPIC B (ArgoEngine) — Full SPDD Command Sequence

```
# --- B.1: Argo proto + .feature (feat:, new gRPC boundary) ---
/spdd-analysis <ARGO_PROTO_ISSUE>
  # Reads: services/engine-adapter/AGENTS.md, protos/AGENTS.md, ADR-015, ADR-016
  # Surfaces: WorkflowEngine interface, EngineAdapterService RPCs, Argo Workflows CRD shape
  # Tier 2 flags: any internal Argo cluster hostnames → canvas.private.md

/spdd-story <ARGO_PROTO_ISSUE>
  # Produces: INVEST stories B.1–B.4 with conventional-commit titles
  # Creates GitHub issues via gh issue create; reports issue URLs

/spdd-reasons-canvas <ARGO_PROTO_ISSUE>
  # Writes: docs/spdd/<ARGO_PROTO_ISSUE>-argo-engine-proto/canvas.md (Status: Draft)
  # Immediately runs /spdd-security-review on the output

/spdd-security-review docs/spdd/<ARGO_PROTO_ISSUE>-argo-engine-proto/canvas.md
  # Must PASS before canvas is committed
  # [human reviews and sets status: Aligned]

/spdd-api-test docs/spdd/<ARGO_PROTO_ISSUE>-argo-engine-proto/canvas.md
  # Generates protos/tests/engine_adapter_service/features/argo_engine.feature
  # (new gRPC boundary: ArgoEngine config in EngineConfig oneof)

/spdd-generate docs/spdd/<ARGO_PROTO_ISSUE>-argo-engine-proto/canvas.md
  # Executes O step 1 (proto changes), stops, waits for review
  # [review → PR B.1 merged]

# --- B.2: ArgoEngine Submit/Signal ---
/spdd-analysis <ARGO_SUBMIT_ISSUE>
/spdd-story <ARGO_SUBMIT_ISSUE>
/spdd-reasons-canvas <ARGO_SUBMIT_ISSUE>
/spdd-security-review docs/spdd/<ARGO_SUBMIT_ISSUE>-argo-submit/canvas.md
  # [human sets: Aligned]
/spdd-generate docs/spdd/<ARGO_SUBMIT_ISSUE>-argo-submit/canvas.md
  # O step 1: ArgoClient wrapper + SubmitWorkflow
  # [review] → /spdd-generate (O step 2: SignalWorkflow) → [review → PR B.2 merged]

# --- B.3: ArgoEngine Query/Cancel/Watch ---
/spdd-analysis <ARGO_QUERY_ISSUE>
/spdd-story <ARGO_QUERY_ISSUE>
/spdd-reasons-canvas <ARGO_QUERY_ISSUE>
/spdd-security-review docs/spdd/<ARGO_QUERY_ISSUE>-argo-query/canvas.md
  # [human sets: Aligned]
/spdd-generate docs/spdd/<ARGO_QUERY_ISSUE>-argo-query/canvas.md
  # [repeat per O step] → [PR B.3 merged]

# --- B.4: ArgoEngine wiring + multi-engine dispatch ---
/spdd-analysis <ARGO_WIRING_ISSUE>
/spdd-reasons-canvas <ARGO_WIRING_ISSUE>
/spdd-security-review docs/spdd/<ARGO_WIRING_ISSUE>-argo-wiring/canvas.md
  # [human sets: Aligned]
/spdd-generate docs/spdd/<ARGO_WIRING_ISSUE>-argo-wiring/canvas.md
  # [PR B.4 merged]
```

---

### EPIC G (e2e Harness) — SPDD Command Sequence

```
# --- G.1: kind cluster bootstrap (feat:) ---
/spdd-analysis <E2E_CLUSTER_ISSUE>
  # Reads: infra/AGENTS.md, docs/patterns/helm-charts.md, all EPIC A canvases
  # Surfaces: cluster config, helmfile or helm umbrella, kind/k3d choice

/spdd-story <E2E_CLUSTER_ISSUE>
/spdd-reasons-canvas <E2E_CLUSTER_ISSUE>
  # Canvas S-Structure: scripts/e2e/, .github/workflows/e2e-smoke.yml (skeleton)
  # Canvas S-Safeguards: no real cluster credentials in canvas; CI kubeconfig ephemeral

/spdd-security-review docs/spdd/<E2E_CLUSTER_ISSUE>-e2e-cluster/canvas.md
  # [Aligned]
/spdd-generate docs/spdd/<E2E_CLUSTER_ISSUE>-e2e-cluster/canvas.md
  # O step 1: kind config + helmfile
  # [review] → O step 2: deploy script → [PR G.1]

# --- G.2: happy-path e2e (feat:) ---
/spdd-analysis <E2E_HAPPY_ISSUE>
/spdd-reasons-canvas <E2E_HAPPY_ISSUE>
  # R: YAML → api-gateway → compiler → engine-adapter → Temporal → task-broker
  #    → http-adapter → EventBusService (JetStream) → memory-service → assert
  # O steps: (1) workflow YAML fixture, (2) deploy stack, (3) trigger, (4) assert CloudEvents, (5) assert memory reads
/spdd-security-review docs/spdd/<E2E_HAPPY_ISSUE>-e2e-happy/canvas.md
  # [Aligned]
/spdd-generate docs/spdd/<E2E_HAPPY_ISSUE>-e2e-happy/canvas.md
  # [repeat per step → PR G.2]

# --- G.3–G.5: Argo path, failure path, upgrade/rollback (all feat:) ---
# Same pattern as G.2: analysis → canvas → security-review → [Aligned] → generate (per step) → PR

# NOTE: G.3–G.5 are SPDD-eligible (feat:) but smaller; canvas is brief (2–3 O steps each)
```

---

### EPIC J (memory-service — stateful service) — Full SPDD Command Sequence

```
# --- J.0: Postgres-backed task-broker repo (prereq, feat:) ---
/spdd-analysis <POSTGRES_TASKBROKER_ISSUE>
  # Reads: services/task-broker/AGENTS.md, docs/adr/ADR-021-horizontal-scale.md,
  #        docs/adr/ADR-008-no-shared-databases.md
  # Surfaces: TaskRepository interface, pgx/v5 adapter, golang-migrate, testcontainers-go

/spdd-story <POSTGRES_TASKBROKER_ISSUE>
/spdd-reasons-canvas <POSTGRES_TASKBROKER_ISSUE>
  # Canvas S-Structure: services/task-broker/internal/infrastructure/postgres/repository.go,
  #   migrations/001_initial.sql, go.mod deps: pgx/v5, golang-migrate
  # Canvas S-Safeguards:
  #   - Never share the tasks table with any other service (ADR-008)
  #   - In-memory adapter RETAINED as test double (not deleted)
  #   - Integration tests use testcontainers-go — no shared/external DB (ADR-016)
  #   - ZYNAX_DB_ENABLED=false in unit tests; true in K8s and compose (ADR-021)

/spdd-security-review docs/spdd/<POSTGRES_TASKBROKER_ISSUE>-postgres-taskbroker/canvas.md
  # Tier 2 check: no DSN literals, no real DB credentials, no internal hostnames → PASS
  # [human sets: Aligned]

/spdd-generate docs/spdd/<POSTGRES_TASKBROKER_ISSUE>-postgres-taskbroker/canvas.md
  # O step 1: go.mod + pgx/v5 + golang-migrate dependency
  # [review] → O step 2: 001_initial.sql migration → [review]
  # → O step 3: postgres/repository.go (TaskRepository impl) → [review]
  # → O step 4: main.go wiring (env-flag switch: memory vs postgres) → [PR J.0]

# --- J.1: Postgres-backed agent-registry repo (same pattern as J.0) ---
/spdd-analysis <POSTGRES_REGISTRY_ISSUE>
/spdd-reasons-canvas <POSTGRES_REGISTRY_ISSUE>
/spdd-security-review docs/spdd/<POSTGRES_REGISTRY_ISSUE>-postgres-registry/canvas.md
  # [Aligned]
/spdd-generate docs/spdd/<POSTGRES_REGISTRY_ISSUE>-postgres-registry/canvas.md
  # [per O step → PR J.1]

# --- J.2: memory-service scaffold (feat:) ---
/spdd-analysis <MEMORY_SCAFFOLD_ISSUE>
/spdd-story <MEMORY_SCAFFOLD_ISSUE>
/spdd-reasons-canvas <MEMORY_SCAFFOLD_ISSUE>
  # Canvas S-Structure:
  #   services/memory-service/cmd/memory-service/main.go
  #   internal/domain/kv.go, vector.go, namespace.go, errors.go
  #   internal/api/handler.go
  #   internal/infrastructure/redis_kv.go, pgvector.go
  # Canvas S-Safeguards (MANDATORY content per state-minimization):
  #   - Single-store evaluation: Redis Stack vs Redis+pgvector — RECORD decision and rationale
  #   - memory-service is the ONLY consumer of its Redis instance; no sharing (ADR-008)
  #   - Postgres schema is exclusively `memory` — no shared tables with task-broker/agent-registry
  #   - Why JetStream cannot substitute: JetStream is message delivery, not synchronous KV
  #     with TTL and similarity search — these are fundamentally different access patterns
  #   - K8s topology: memory-service Go pod = stateless Deployment; Redis = StatefulSet/PVC
  #   - KV TTL is best-effort (per proto invariant 3); callers must not depend on sub-second precision
  #   - workflow_id isolation: service MUST reject any cross-namespace access (proto invariant 2)

/spdd-security-review docs/spdd/<MEMORY_SCAFFOLD_ISSUE>-memory-scaffold/canvas.md
  # Tier 2 flags: no real Redis/Postgres hostnames, no credentials → must PASS
  # [human sets: Aligned]

/spdd-api-test docs/spdd/<MEMORY_SCAFFOLD_ISSUE>-memory-scaffold/canvas.md
  # Generates BDD grpcurl scenarios for MemoryService 10 RPCs
  # (BDD feature file already committed — this generates test script and verifies coverage)

/spdd-generate docs/spdd/<MEMORY_SCAFFOLD_ISSUE>-memory-scaffold/canvas.md
  # O step 1: go.mod + domain interfaces
  # [review → PR J.2]

# --- J.3: Redis KV adapter ---
/spdd-analysis <REDIS_KV_ISSUE>
/spdd-reasons-canvas <REDIS_KV_ISSUE>
  # Inherits single-store decision from J.2 Canvas; links [[memory-scaffold]]
/spdd-security-review docs/spdd/<REDIS_KV_ISSUE>-redis-kv/canvas.md
  # [Aligned]
/spdd-generate docs/spdd/<REDIS_KV_ISSUE>-redis-kv/canvas.md
  # O step 1: go-redis v9 dep → O step 2: redis_kv.go Set/Get/Delete/ListKeys
  # → O step 3: MGet/MSet/DeleteNamespace + TTL → [PR J.3]

# --- J.4: pgvector adapter ---
/spdd-analysis <PGVECTOR_ISSUE>
/spdd-reasons-canvas <PGVECTOR_ISSUE>
  # Safeguards: pgvector schema is exclusively memory-service; no JOIN across service schemas
/spdd-security-review docs/spdd/<PGVECTOR_ISSUE>-pgvector/canvas.md
  # [Aligned]
/spdd-generate docs/spdd/<PGVECTOR_ISSUE>-pgvector/canvas.md
  # O step 1: pgvector migration + pgxpool → O step 2: StoreVector/QueryVector/DeleteVector
  # → [PR J.4]

# --- J.5: namespace TTL + isolation --- (follow same pattern)
# --- J.6: gRPC handler wiring --- (follow same pattern)

# --- J.7: BDD steps (test: — SPDD-exempt) ---
# No Canvas needed. Directly implement testcontainers-go BDD steps for memory_service.feature.
# test: PR → no /spdd-* commands required
```

**When requirements change mid-implementation (e.g., single-store evaluation in J.2 changes the store choice):**
```
/spdd-prompt-update docs/spdd/<MEMORY_SCAFFOLD_ISSUE>-memory-scaffold/canvas.md
  # Describe the change (e.g., "switching to Redis Stack for both KV and vector planes")
  # Updates R, E, A, S sections; resets Status to Draft; runs /spdd-security-review
  # [human re-aligns] → /spdd-generate continues from affected O step
```

**After any refactoring that doesn't change logic (e.g., renaming files):**
```
/spdd-sync docs/spdd/<MEMORY_SCAFFOLD_ISSUE>-memory-scaffold/canvas.md
  # Updates S-Structure file paths; sets Status: Synced
```

**SPDD-exempt story types (skip Canvas):**
```
# test: J.7  — BDD step implementations → no /spdd-* commands
# ci:   H.1  — e2e smoke job → no /spdd-* commands
# ci:   A.12 — helm lint gate → no /spdd-* commands
# refactor: D.1 — stateless compiler (canvas already exists at #466) → use existing canvas
# docs: A.13, F.4 → no /spdd-* commands
```

---

## Section 7 — GitHub Bootstrap Commands (Deliverable 5)

> **For human review and execution only — do NOT run these automatically.**
> Labels and milestone title verified against live repo:
>   - Milestone: `"K8s Production-Ready (M6)"` (confirmed from milestones API, #6)
>   - `type: adr-proposal`, `type: feature`, `type: epic`, `type: docs`, `type: test`, `type: ci`, `type: chore`
>   - `area: event-bus`, `area: memory-service`, `area: engine-adapter`, `area: infra`, `area: workflow-compiler`, `area: api-gateway`
>   - `priority: high`, `priority: medium`
>   - `milestone: M6` label (separate from GitHub Milestone — use both)
>
> Paste in order: (0) EBUS-DECISION → (1) epics → (2) stories

---

### (0) EBUS-DECISION Issue — Paste First (gates EPIC I)

```bash
gh issue create \
  --title "decision(event-bus): ADR-022 — EventBusService gRPC vs CloudEvents library + direct NATS" \
  --label "type: adr-proposal,type: docs,area: event-bus,priority: high,milestone: M6" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Decision Needed

ADR-022: Should EventBusService remain a gRPC wrapper over JetStream (Option 1),
or should services use NATS/JetStream directly with a shared CloudEvents library (Option 2/3)?

## Why Now

M6 implements event-bus for real. This is a one-way door — ADR required per CLAUDE.md.
It gates EPIC I scope and the e2e harness (EPIC G).

## Pre-Commitment Evidence

The repo has substantially pre-committed to Option 1:
- `protos/zynax/v1/event_bus.proto` — gRPC EventBusService committed
- `services/event-bus/tests/features/event_bus.feature` — 6 BDD scenarios committed
- `services/event-bus/AGENTS.md` — describes gRPC service with NATSEventBus infrastructure adapter
- ADR-001: all inter-service calls are gRPC; ADR-013: Python adapters use gRPC stubs only

## Options

1. **Full gRPC EventBusService** wrapping JetStream (recommended — repo pre-committed)
2. **Shared CloudEvents library + direct NATS** (would require reverting proto + BDD)
3. **Hybrid**: thin control-plane service + direct NATS for internal (partially violates ADR-001)

## Required Output

- `docs/adr/ADR-022-event-bus-architecture.md` recording the choice
- EPIC I (#<EPIC_I_ISSUE>) unblocked after merge

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"
```

---

### (1) New Epics — Paste After EBUS-DECISION

```bash
# EPIC A — Helm Charts + K8s Runtime Provider
gh issue create \
  --title "epic(infra): M6.Helm — Helm charts for all 7 services + cluster dependencies" \
  --label "type: epic,area: infra,priority: high,milestone: M6" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Scope

Helm charts (Deployment, Service, ServiceAccount, HPA, PDB, NetworkPolicy) for all
7 Go services: api-gateway, workflow-compiler, engine-adapter, task-broker,
agent-registry, event-bus, memory-service.

Cluster dependencies: Temporal (existing subchart), NATS JetStream (new), Postgres 16
(Bitnami, for task-broker + agent-registry + memory-service schemas), Redis (dedicated
to memory-service KV plane), cert-manager ClusterIssuer.

memory-service chart uses StatefulSet/PVC for Redis. All Go service pods are
stateless Deployments.

## Stories

- A.0: Shared _helpers.tpl library chart
- A.1–A.7: One chart PR per service
- A.8: NATS JetStream subchart
- A.9: Postgres subchart
- A.10: Temporal dependency
- A.11: cert-manager certificates
- A.12: helm lint CI gate (#242)
- A.13: Environment parity manifest (#243)

## References

infra/AGENTS.md · docs/patterns/helm-charts.md · ADR-020 (mTLS) · ADR-021 (Postgres)

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

# EPIC B — ArgoEngine Adapter
gh issue create \
  --title "epic(engine-adapter): M6.Argo — ArgoEngine WorkflowEngine implementation" \
  --label "type: epic,area: engine-adapter,priority: medium,milestone: M6" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Scope

Implement ArgoEngine as a second WorkflowEngine behind the existing WorkflowEngine
interface (ADR-015). No engine name hardcoding.

Stories:
- B.1: Proto config message for Argo + .feature BDD scenarios
- B.2: ArgoEngine Submit + Signal
- B.3: ArgoEngine Query + Cancel + Watch
- B.4: Multi-engine dispatch wiring + config flag

## ADRs

ADR-015 (pluggable engines) · ADR-016 (.feature before implementation)

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

# EPIC D — Multi-Namespace + Stateless Compiler (extends #466)
gh issue create \
  --title "epic(workflow-compiler): M6.NS — Multi-namespace support in workflow-compiler" \
  --label "type: epic,area: workflow-compiler,priority: medium,milestone: M6" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Scope

Extends M6.D (#466 stateless compiler) with multi-namespace routing.

Stories:
- D.1: Drop in-memory IR store (#490) — use existing #466 canvas
- D.2: Add namespace field to WorkflowIR proto + .feature
- D.3: Namespace-scoped compilation in workflow-compiler
- D.4: Propagate namespace from api-gateway request

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

# EPIC E — Policy Enforcement
gh issue create \
  --title "epic(api-gateway): M6.Policy — routing policies, rate limits, capability quotas" \
  --label "type: epic,area: api-gateway,area: workflow-compiler,priority: medium,milestone: M6" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Scope

Policy enforcement layer: routing policies, per-IP rate limits, capability quotas.

Stories:
- E.1: Policy enforcement proto messages (RoutingPolicy, RateLimit, CapabilityQuota) + .feature
- E.2: Per-IP token-bucket rate limit on POST /api/v1/apply (#580)
- E.3: Routing policy + capability quota in workflow-compiler
- E.4: Quota check in engine-adapter before DispatchCapabilityActivity

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

# EPIC F — zynax-sdk PyPI + Supply Chain
gh issue create \
  --title "epic(agents): M6.SDK — zynax-sdk PyPI publish + supply chain hardening" \
  --label "type: epic,area: agents/sdk,area: ci,priority: medium,milestone: M6" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Scope

PyPI publish for zynax-sdk (pyproject.toml already has name + hatchling build backend).
Plus supply chain hardening (#465 canvas exists).

Stories:
- F.1: Trusted publisher + TestPyPI dry-run
- F.2: CI release-triggered PyPI publish workflow
- F.3: cosign signing + SBOM (closes #489, #235, #239)
- F.4: SDK docstrings step 2 (closes #376)

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

# EPIC G — e2e Harness
gh issue create \
  --title "epic(infra): M6.E2E — real end-to-end harness (kind/k3d + Helm + reference workflows)" \
  --label "type: epic,area: infra,priority: high,milestone: M6" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Scope

kind/k3d cluster + Helm full-stack deployment running the reference workflows
(code-review.yaml, ci-pipeline.yaml, research-task.yaml) end-to-end on both
Temporal and Argo engines.

Assert: emitted CloudEvents consumed off JetStream (not mocked) + memory-service
reads returning what was written.

Stories:
- G.1: kind cluster bootstrap + helmfile
- G.2: happy-path e2e via Temporal (assert CloudEvents off JetStream, memory reads)
- G.3: Argo path e2e
- G.4: failure-path e2e (timeout, guard-rejected, 5xx → workflow.failed)
- G.5: Helm upgrade/rollback test (--atomic, Temporal↔Argo values matrix)

## Scope NOT in M6

Full observability (distributed traces, dashboards) — explicitly M7.

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

# EPIC H — e2e CI Gate
gh issue create \
  --title "epic(ci): M6.CI-E2E — e2e smoke + upgrade CI gate on infra/services changes" \
  --label "type: epic,area: ci,priority: medium,milestone: M6" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Scope

GitHub Actions jobs running the kind+Helm e2e stack. Gated — not on every PR.
Triggers on changes to infra/ or engine-adapter/.

Stories:
- H.1: e2e smoke job (kind + helm, gated on infra/services/ changes)
- H.2: matrix values test (Temporal-backed vs Argo-backed helm values)

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

# EPIC I — event-bus Implementation (BLOCKED-ON EBUS-DECISION)
gh issue create \
  --title "epic(event-bus): M6.I — event-bus NATS JetStream implementation" \
  --label "type: epic,area: event-bus,priority: high,milestone: M6" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## BLOCKED-ON: EBUS-DECISION

This epic is blocked until ADR-022 is merged (see EBUS-DECISION issue).
Do NOT write implementation-story REASONS Canvases until ADR-022 is resolved.

## Assumed Scope (if Option 1 / gRPC EventBusService chosen)

Stateless Go service wrapping NATS JetStream. Durability lives entirely in
JetStream streams/consumers (retention, replay, DLQ). No separate database.

Stories:
- I.0: docs: ADR-022 (decision record)
- I.1: Service scaffold (go.mod, cmd/, domain/, NATS client bootstrap)
- I.2: Publish path (JetStream stream create + event publish)
- I.3: Subscribe path (durable consumer group + gRPC server-streaming)
- I.4: Unsubscribe + DLQ + retry-backoff
- I.5: Wire engine-adapter PublishLifecycleEventActivity to EventBusService gRPC
- I.6: BDD step implementations for event_bus.feature (6 scenarios)

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

# EPIC J — memory-service Implementation
gh issue create \
  --title "epic(memory-service): M6.J — memory-service KV + vector implementation" \
  --label "type: epic,area: memory-service,priority: high,milestone: M6" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Scope

Implement MemoryService (10 RPCs) with Redis KV plane and pgvector vector plane.
The ONLY stateful Go service in the control plane: Redis = dedicated StatefulSet/PVC.
Postgres (pgvector schema) is shared at cluster level with task-broker/agent-registry
schemas per ADR-008 (each service owns its schema exclusively).

## State-Minimization Justification

Agents must not hold shared state internally (twelve-factor principle).
JetStream cannot substitute: it is message-passing, not a synchronous KV store
with TTL enforcement and ANN similarity search. Redis+pgvector is the minimum
viable store combination for the MemoryService contract.

Before committing the J.2 Canvas, evaluate Redis Stack (vector module) as a
single-store alternative to Redis+pgvector.

## Stories

- J.0: Postgres-backed task-broker repo (pgx/v5 + migrations) — prereq
- J.1: Postgres-backed agent-registry repo — prereq
- J.2: Service scaffold (go.mod, cmd/, domain interfaces)
- J.3: Redis KV adapter (Set/Get/Delete/ListKeys/MGet/MSet/DeleteNamespace)
- J.4: pgvector adapter (StoreVector/QueryVector/DeleteVector)
- J.5: Namespace TTL enforcement + workflow_id isolation
- J.6: gRPC handler wiring (all 10 RPCs, integration tests via testcontainers-go)
- J.7: BDD step implementations for memory_service.feature (6 scenarios)

## Dependencies

#626 (M6.H Postgres-backed repos) is prerequisite for J.0–J.1 + J.4.
EPIC A (A.9 Postgres chart) is prerequisite for K8s deployment.

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"
```

---

### (2) Key Stories — Paste After Epics

> Only the highest-priority and untracked stories are shown below.
> Existing tracked stories (#487, #488, #489, #490, #491, #580, etc.) already exist — do NOT recreate.
> Replace `<EPIC_A_ISSUE>`, `<EPIC_B_ISSUE>`, etc. with the issue numbers from step (1).

```bash
# A.0 — Shared helpers library chart (prerequisite for all service charts)
gh issue create \
  --title "feat(infra): shared Helm _helpers.tpl library chart for all Zynax services" \
  --label "type: feature,area: infra,priority: high,milestone: M6,size/S" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Story

As a platform operator, I want a shared Helm library chart with canonical
_helpers.tpl macros so that all service charts have consistent labels,
selectors, and security contexts without duplication.

## Acceptance Criteria

- [ ] `helm/zynax-lib/Chart.yaml` with `type: library`
- [ ] `_helpers.tpl` includes: `zynax.fullname`, `zynax.labels`, `zynax.selectorLabels`, `zynax.serviceAccountName`
- [ ] Security context macro: `runAsNonRoot: true`, `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, `capabilities.drop: ["ALL"]`, `seccompProfile.type: RuntimeDefault`
- [ ] `ct lint` passes on the library chart
- [ ] `docs/patterns/helm-charts.md` references the library chart

## Out of Scope

Individual service charts (A.1–A.7 each add their own chart depending on this library).

## Size estimate: S (≤200 lines)
## Dependencies: None — this is the first chart story

Closes #<EPIC_A_ISSUE> (step A.0)
Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

# B.1 — Argo proto + BDD feature (contracts first)
gh issue create \
  --title "feat(protos): ArgoEngine config message + BDD scenarios for Argo engine dispatch" \
  --label "type: feature,area: engine-adapter,area: protos,priority: medium,milestone: M6,size/S,spdd: canvas-step" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Story

As a workflow author, I want to specify Argo Workflows as the execution engine in my
WorkflowIR so that the engine-adapter can dispatch to Argo instead of Temporal.

## Acceptance Criteria

- [ ] `protos/zynax/v1/engine_adapter.proto`: `EngineConfig` oneof extended with `ArgoConfig` message
- [ ] `ArgoConfig` includes: `argo_server_url`, `namespace`, `service_account_name`, `workflow_template_ref`
- [ ] `.feature` file committed: `protos/tests/engine_adapter_service/features/argo_engine.feature` with ≥3 Gherkin scenarios (submit, query, cancel via Argo)
- [ ] `make generate-protos` succeeds (Go + Python stubs regenerated)
- [ ] `buf breaking` CI gate passes

## Size estimate: S (≤200 lines, stub exclusions apply to *.pb.go)
## Dependencies: None

Closes #<EPIC_B_ISSUE> (step B.1)
Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

# I.0 — ADR-022 (must be first I story; unblocks all EPIC I)
gh issue create \
  --title "docs: ADR-022 — EventBusService gRPC architecture decision record" \
  --label "type: docs,type: adr-proposal,area: event-bus,priority: high,milestone: M6,size/S" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Story

Record the EBUS-DECISION outcome as ADR-022 in docs/adr/.

## Acceptance Criteria

- [ ] `docs/adr/ADR-022-event-bus-architecture.md` merged
- [ ] `docs/adr/INDEX.md` updated with ADR-022 entry
- [ ] ADR references: ADR-001 (gRPC), ADR-013 (adapter-first), ADR-014 (event-driven), ADR-016 (contracts)
- [ ] Implementation scope for EPIC I confirmed in ADR (Option 1/2/3)
- [ ] services/event-bus/AGENTS.md updated to reference ADR-022

## Size estimate: S (≤200 lines — docs)
## Depends on: EBUS-DECISION issue resolved by human

Closes #<EPIC_I_ISSUE> (step I.0)
Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

# J.2 — memory-service scaffold (first implementation story for EPIC J)
gh issue create \
  --title "feat(memory-service): service scaffold — go.mod, domain KV+Vector interfaces, cmd/" \
  --label "type: feature,area: memory-service,priority: high,milestone: M6,size/S,spdd: canvas-step" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Story

As a platform developer, I want the memory-service Go module scaffolded with
domain interfaces so that subsequent stories (Redis KV, pgvector) can implement
them independently.

## Acceptance Criteria

- [ ] `services/memory-service/go.mod` with correct module path
- [ ] `internal/domain/kv.go`: `KVStore` interface (Set/Get/Delete/ListKeys/MGet/MSet/DeleteNamespace)
- [ ] `internal/domain/vector.go`: `VectorStore` interface (StoreVector/QueryVector/DeleteVector)
- [ ] `internal/domain/errors.go`: `ErrKeyNotFound`, `ErrNamespaceNotFound`, `ErrDimensionMismatch`
- [ ] `cmd/memory-service/main.go`: wiring-only skeleton (compiles; returns UNIMPLEMENTED on all RPCs)
- [ ] `GOWORK=off go build ./...` succeeds
- [ ] Canvas S-Safeguards records single-store evaluation (Redis Stack vs Redis+pgvector)
- [ ] Canvas status: Aligned before PR opened

## Size estimate: S (≤200 lines)
## Dependencies: None (scaffold is independent)

Closes #<EPIC_J_ISSUE> (step J.2)
Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

# G.1 — e2e cluster bootstrap
gh issue create \
  --title "feat(infra): kind cluster bootstrap + helmfile for full Zynax stack e2e" \
  --label "type: feature,area: infra,priority: high,milestone: M6,size/M,spdd: canvas-step" \
  --milestone "K8s Production-Ready (M6)" \
  --body "$(cat <<'EOF'
## Story

As a CI engineer, I want a reproducible script that spins up a kind/k3d cluster
and deploys the full Zynax stack via Helm so that e2e tests can run against a
real cluster in CI and locally.

## Acceptance Criteria

- [ ] `scripts/e2e/cluster-up.sh`: creates kind cluster with appropriate node config
- [ ] `scripts/e2e/deploy-stack.sh`: runs helmfile/helm install for all charts (or umbrella)
- [ ] All 7 Go services + NATS + Temporal + Postgres + Redis start healthy
- [ ] `scripts/e2e/cluster-down.sh`: tears down cluster
- [ ] Script works on CI Ubuntu-latest runner with Docker-in-Docker
- [ ] Documented minimum resource requirements (RAM, CPU)

## Size estimate: M (201–400 lines; scripts justified by e2e complexity)
## Dependencies: All EPIC A charts merged (A.0–A.11)

Closes #<EPIC_G_ISSUE> (step G.1)
Assisted-by: Claude/claude-sonnet-4-6
EOF
)"
```

---

## Section 8 — Open Questions for the Human

> Ordered by urgency. Lead with the EBUS-DECISION.

1. **EBUS-DECISION (blocks EPIC I):** The repo pre-commits to Option 1 (gRPC EventBusService wrapping JetStream), but confirm this as ADR-022 before any EPIC I implementation starts. Run the `gh issue create` from Deliverable 0 to open the decision issue; align on ADR-022; then EPIC I stories can proceed.

2. **memory-service single-store question:** Should the J.2 Canvas evaluation recommend Redis Stack (single store for KV + vector) over the pre-decided Redis+pgvector (two stores)? This is an operational simplicity vs known-patterns tradeoff. Flag your preference before J.2 Canvas is written.

3. **State-minimization vs ADR-021 conflict:** The prompt says "only memory-service is stateful." ADR-021 (accepted) adds Postgres for task-broker + agent-registry. Reconcile: all Go service Deployments ARE stateless; Postgres is a shared cluster StatefulSet with separate schemas (ADR-008 compliant). Confirm this interpretation is correct before starting EPIC A Postgres subchart (A.9).

4. **Multi-namespace protocol design:** How should namespace context flow from the CLI user to the workflow-compiler? (a) HTTP header in api-gateway request → gRPC metadata, or (b) new field in YAML manifest → WorkflowIR namespace? Option (b) requires a proto change and new spec field. Decide before D.2 starts.

5. **Policy enforcement priority:** EPIC E (policy enforcement) is on the roadmap but has no design. The per-IP rate limit (#580) is the most tractable story. Should the full routing-policy/capability-quota scope be M6 or deferred to M6.1? If M6, a proto design sprint (E.1) is needed early.

6. **e2e harness timing:** EPIC G requires all EPIC A charts, EPIC I (event-bus), and EPIC J (memory-service) to be merged first. Given EPIC I is blocked on EBUS-DECISION, do you want EPIC G to start with a mock event bus stub and wire the real one in a later G iteration? Recommend: G.1–G.2 with Temporal + mocked events first; G.3 adds real JetStream after EPIC I.

7. **ArgoEngine chart dependency:** EPIC B (ArgoEngine) requires the Argo Workflows cluster dependency, which has large CRD footprint. Should Argo Workflows be an optional/feature-flag chart dependency in M6, or mandatory for all deployments?

8. **zynax-sdk version:** Should the first PyPI release be `0.1.0` (current pyproject.toml value) or `0.1.0.dev0` / `1.0.0a1`? This affects the PyPI stable channel vs pre-release distinction.

9. **Platform config convergence ordering (#670):** Should #667 (shared config lib) and #668 (Dockerfile template) go BEFORE the Helm charts (since charts embed env var names) or in parallel? Recommend: #667 first (shared config defines env var grammar for all charts).

---

*Zynax — M6 Planning Document · Apache 2.0*
*Generated 2026-06-02 from live repo state at commit 994efb7*
