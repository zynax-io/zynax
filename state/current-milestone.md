<!-- Canonical status file. Updated by /milestone-close and /repo-clean. Do not edit by hand. -->

# Current Milestone State

> This file tracks the active execution state. Update it when milestones close,
> blockers change, or active work shifts. Do NOT use this file for architecture
> decisions — those belong in `docs/adr/`. Do NOT accumulate history here.

---

## Status Summary

| Milestone | Status | Version |
|-----------|--------|---------|
| M1 — Contracts Foundation | ✅ Complete | v0.1.0 |
| M2 — Workflow IR | ✅ Complete | v0.1.0 |
| M3 — Temporal Execution | ⚠ Partial | v0.2.0 |
| M4 — YAML System + CLI | ⚠ Partial | v0.3.0 |
| **M5 — Adapter Library** | ✅ **Complete** | **v0.4.0** |
| **M6 — K8s Production-Ready** | ✅ **Complete** | **v0.5.0** |
| **M7 — Usable Workflows + Observability** | 🚧 **Active** | **v0.6.0 (target)** |

M3/M4 are partial because task-broker and agent-registry were not delivered in those milestones.
Both completed under M5.C (#460). CloudEvents publishing is log-only (not wired to NATS).
v0.4.0 tag pushed 2026-05-29; GitHub Release live at https://github.com/zynax-io/zynax/releases/tag/v0.4.0
GitHub milestone "Adapter Library (M5)" closed 2026-05-29; 5 deferred issues (#235 #239 #376 #466 #656) moved to M6.
See [docs/milestones/M5-plan.md](../docs/milestones/M5-plan.md).

---

## M7 — Active (🚧 target v0.6.0, opened 2026-06-15)

GitHub milestone #7 ("Full Observability"); scope broadened to **Usable Workflows + Observability**
— first of the M7 → M-dx → M8 program. Full plan: **[docs/milestones/M7-planning.md](../docs/milestones/M7-planning.md)**.

Goal: a developer authors a real multi-step workflow, runs it locally (`docker compose up` incl.
**Uptrace**), watches it execute state→state with **data-flow**, **streamed logs**, and a connected
**distributed trace** in the Uptrace login UI — with green `make ci`.

### EPICs (10) — one REASONS Canvas each under `docs/spdd/<letter>-<slug>/`
| EPIC | Title | Type | Canvas |
|------|-------|------|--------|
| W | Workflow data-flow (output/input bindings) — **keystone** | feat | [W](../docs/spdd/W-workflow-data-flow/canvas.md) |
| L | Execution log/event streaming (`/logs`) | feat | [L](../docs/spdd/L-log-streaming/canvas.md) |
| O | Observability — OTEL + **Uptrace** (traces/metrics/logs/APM + login UI; compose **and** Helm) | feat | [O](../docs/spdd/O-observability-otel-uptrace/canvas.md) |
| C | Context propagation (trace · data · correlation) | feat | [C](../docs/spdd/1168-context-propagation/canvas.md) |
| G | Git MCP shim over git-adapter | feat | [G](../docs/spdd/G-git-mcp-shim/canvas.md) |
| X | Expert-agent substrate + `agents/examples/` | feat | [X](../docs/spdd/1170-expert-substrate/canvas.md) |
| T | Reusable templates + first real workflows | feat | [T](../docs/spdd/1171-templates-real-workflows/canvas.md) |
| R | Test rigor (absorbs #469; #493 #553 #1103) | test | [R](../docs/spdd/469-test-rigor/canvas.md) |
| Q | Quality & supply-chain fixes (pip CVE · coverage gate · PyPI Trusted Publisher) | chore/ci | [Q](../docs/spdd/1172-quality-supply-chain/canvas.md) |
| D | Docs — quick-start · authoring · observability | docs | [D](../docs/spdd/1173-docs/canvas.md) |

Pre-existing M7 EPICs #467 (OTel) / #468 (history streaming) / #469 (test rigor) are **absorbed** into
O / L / R respectively. New ADRs: ADR-029…034 (Proposed).

---

## M6 — Complete (✅ released v0.5.0, 2026-06-12)

GitHub milestone #6 closed; all 8 EPICs and every story delivered (one deferral: Wave 4 O8 → M7 #1103).
Release: https://github.com/zynax-io/zynax/releases/tag/v0.5.0 · **Next milestone: M7 — open (active) as of 2026-06-15.**

Full plan + per-EPIC status: **[docs/milestones/M6-planning.md](../docs/milestones/M6-planning.md)**.
As of 2026-06-11: **143 issues closed / 17 open** (CI-overhaul stories #1110–#1122 added 2026-06-10).

### EPICs delivered ✅
| EPIC | Issue | Notes |
|------|-------|-------|
| Postgres-backed repositories | #626 | task-broker + agent-registry on pgx/v5 |
| Helm charts | #765 | all 7 services + NATS/Postgres/Temporal subcharts |
| EventBus over NATS JetStream | #772 | Publish/Subscribe/Unsubscribe + DLQ (ADR-022) |
| Config convergence | #670 | libs/zynaxconfig shared package |
| Container image source-of-truth | #855 | `images.yaml` + drift gate + ADR-024 |
| Self-hosting dev-automation | #873 | Waves 0–3 superseded (#1129); Wave 4 delivered to boundary via #881; closing |
| DevAuto Wave 4 — self-hosted issue-delivery | #881 | O1–O7 + O9 delivered; O8 → M7 #1103; canvas Implemented-at-boundary; closing |
| Orchestrator concurrency hardening | #1001 | worktree isolation + idempotent dispatch |
| Health probes · mTLS · supply-chain | #463 #464 #465 | startup/readiness/liveness, cert-manager, cosign+SBOM |
| memory-service KV + vector | #773 | Redis KV + pgvector adapters, namespace TTL isolation, all 10 RPCs + BDD |
| ArgoEngine | #766 | ArgoEngine WorkflowEngine + multi-engine dispatch (#798) |
| Multi-namespace support | #767 | namespace isolation in workflow-compiler |
| Policy enforcement | #768 | routing policies, rate limits (#802), capability quotas (#803 #804) |
| SDK PyPI publish | #769 | zynax-sdk on PyPI + supply-chain hardening |
| e2e harness | #770 | kind + Helm + reference workflows (#811 #812 #813) |
| Native multi-arch build | #837 | QEMU eliminated, image sizes audited (#841) |
| gRPC health protocol | #74 #656 | grpc.health.v1 in all services, K8s-native probes |
| Prometheus /metrics | #491 | per-request counters in all services (OTel rest → M7 #467) |
| DevAuto Wave 3 | #880 | post-merge completeness mesh |

### In progress / remaining
**e2e-green execution path (#1086 — O1 #1087 ✅ / O2 #1088 ✅ merged via PR #1095; O3 #1089 ✅ satisfied by build-images gate PR #1132; O4 #1090 ✅ event-bus + memory-service enabled with required assertions; O5 #1091 ✅ runner sizing verified on full stack; bug #1149 ✅ fixed — JetStream subject overlap, completed/failed CloudEvent assertions now required — O6 #1092 ✅ temporal leg required + skip-shim; argo advisory until stable — EPIC #1086 complete)**,
Postgres off Bitnami (#1073 — ✅ complete: O1 ADR-026, O2–O3 #1076, O4–O5 #1077, O6 #1078, O7–O8 #1079; canvas Implemented, EPIC ready to close),
CI-E2E gate (#771 — #1070 ✅, #1071 ✅ merged via PR #1155: engine matrix temporal/argo in e2e-smoke; bug #1157 ✅ fixed — ArgoEngine submit now sends the WorkflowCreateRequest envelope, argo leg unblocked; #1092 ✅ — EPIC #771 complete),
DevAuto Wave 4 (#881 — ✅ delivered to the platform-readiness boundary: O1 #1096 ADR-028; O2 #1097 9 expert AgentDefs; O3 #1098 orchestrator Workflow; O4 #1099 issue-delivery intake→plan→route; O5 #1100 context-slice injection binding; O6 #1101 delivery leg inject→implement→verify→decide; O7 #1102 learning-synthesizer AgentDef; O9 #1104 status reconcile. O8 #1103 deferred to M7 — operator decision 2026-06-12, four code-verified platform gaps (compiler `output:` rejection, Go-template vs CEL guards, missing orchestration-capability providers, no gateway outputs/decision-log read path; analysis in #1103 comments); the strict xfail in `automation/tests/test_platform_readiness.py` remains the honest gate. Canvas Implemented-at-boundary; EPICs #881 and #873 close at the boundary).

---

### M6.E2E-Green — make the e2e-smoke gate execute a workflow end-to-end (#1086) — ✅ Complete

Canvas: `docs/spdd/1086-e2e-green/canvas.md` — Status: **Implemented**. The gate brings the cluster
up but fails at the happy-path assertion; this epic closes the execution-path gaps discovered
2026-06-10. Delivery order (parallelizable where noted):

| Step | Story | Type | Size | Depends on |
|------|-------|------|------|-----------|
| O1 | [#1087](https://github.com/zynax-io/zynax/issues/1087) expose api-gateway on host (NodePort 30080) | feat(infra) | S | ✅ merged (PR #1095) |
| O2 | [#1088](https://github.com/zynax-io/zynax/issues/1088) minimal capability worker + reference workflow → succeeded | feat(infra) | M | ✅ merged (PR #1095) |
| O3 | [#1089](https://github.com/zynax-io/zynax/issues/1089) event-bus + memory-service in pre-merge build-images matrix | ci(infra) | M | ✅ satisfied by PR #1132 (shift-left, ADR-027) — closed with evidence |
| O4 | [#1090](https://github.com/zynax-io/zynax/issues/1090) enable event-bus + memory-service in e2e + assertions | test(infra) | M | ✅ merged — lifecycle CloudEvent + memory-Get required, no skip path; completed-event check now also required (#1149 fixed) |
| O5 | [#1091](https://github.com/zynax-io/zynax/issues/1091) right-size e2e-smoke runner / pod resources | ci(infra) | S | ✅ verified 2026-06-12 — full 7-service stack GREEN on ubuntu-latest (2 CPU/7 GB), zero Evicted/OOMKilled/Pending; 3 consecutive green runs (PRs #1148, #1150) |
| O6 | [#1092](https://github.com/zynax-io/zynax/issues/1092) promote gate advisory → stable/required | ci | S | #1090 #1091 #1071 · unblocked: #1149 fixed (JetStream subject overlap; completed/failed event assertions now required in e2e scripts) · #1157 fixed (argo leg 422 — WorkflowCreateRequest envelope) |

**Ready now for `/milestone-orchestrate`:** #1092 (O6 — promote gate to required; depends #1071. #1149 fixed: completed/failed CloudEvent assertions already required in the e2e scripts).

## Recently Closed (last updated 2026-06-11)

**M6 CI/CD overhaul (EPIC #1109) — 2026-06-11 session:**
- **#1118** ci: pre-merge build-images gate — staging lane, Hadolint, Trivy, SBOM (PR #1132)
- **#1120** ci: release.yml retag model — workflow_run promotion + atomic images.yaml digest sync (ADR-027)
- **#1122** docs(contributing): engineering manifesto — 15 enforced principles, DORA targets, CNCF patterns
  (capstone of EPIC #1107 — **all child issues done; #1107 ready to close**)

**M6 batch — 2026-06-08 session:**
- **#799** feat(workflow-compiler): namespace-scoped capability routing (PR #977)
- **#805** feat(agents): PyPI trusted publisher + TestPyPI dry-run (PR #973)
- **#838** ci(infra): migrate release.yml service builds to native arm64 (PR #974)
- **#839** ci(infra): migrate tools-image.yml to native arm64 (PR #975)
- **#861** docs: propagation — README/CONTRIBUTING/AGENTS.md/CLAUDE.md for images SoT
- **#862** docs: ADR-024 container image reference management
- **#866** ci: description-present gate + size-budget check (⚠ regression fixed by PR #979)
- **#869** docs: document unknown/unknown attestation manifests
- **#875** chore(automation): expert mesh YAML configs
- **#876** chore(automation): orchestrator config
- **#877** ci(automation): Wave 0 advisory workflow (PR #969)

**M6 memory-service chain — 2026-06-08:**
- **#815** Redis KV adapter · **#816** pgvector adapter · **#817** namespace TTL enforcement
- **#818** gRPC handler wiring · **#819** BDD step implementations

**M6 event-bus chain — 2026-06-08:**
- **#823** service scaffold · **#824** Publish path · **#825** Subscribe path
- **#826** Unsubscribe + DLQ + retry · **#827** engine-adapter wiring · **#828** BDD steps

**Earlier sessions:**
- **#474** #459 #479 #480 #543 #556 — epics closed; all children had merged prior sessions
- **#577** #574 #576 #712 #714 — phantom agent cleanup, summarizer removal, git adapter
- **#865** ci: OCI manifest annotations · **#868** ADR-025 SLSA provenance

---

## M6 — Final Work Log (historical)

### M6.A Health Probe Semantics (#463) — ✅ COMPLETE

| Story | Issue | Status |
|-------|-------|--------|
| A.1 split probes in api-gateway | [#487](https://github.com/zynax-io/zynax/issues/487) | ✅ Merged (#821) |

Canvas: `docs/spdd/463-health-probes/canvas.md` — Status: Implemented

### M6.D Stateless Compiler (#466) — ✅ COMPLETE

| Story | Issue | Status |
|-------|-------|--------|
| D.1 drop in-memory IR store | [#490](https://github.com/zynax-io/zynax/issues/490) | ✅ Merged (#774) |

Canvas: `docs/spdd/466-stateless-compiler/canvas.md` — Status: Implemented

### EBUS-DECISION (#764) — ✅ RESOLVED

**ADR-022 accepted — Option 1: Full gRPC EventBusService wrapping NATS JetStream.** (PR #822)
EPIC I (#772) unblocked. Stories created: #823 #824 #825 #826 #827 #828.
Canvas: `docs/spdd/772-event-bus/canvas.md` — Status: Aligned ✅

### M6.I Event Bus NATS JetStream (#772) — ✅ COMPLETE

All 6 stories merged 2026-06-08.

| Story | Issue | Status |
|-------|-------|--------|
| I.1 feat(event-bus): service scaffold | [#823](https://github.com/zynax-io/zynax/issues/823) | ✅ Merged |
| I.2 feat(event-bus): Publish path | [#824](https://github.com/zynax-io/zynax/issues/824) | ✅ Merged |
| I.3 feat(event-bus): Subscribe path | [#825](https://github.com/zynax-io/zynax/issues/825) | ✅ Merged |
| I.4 feat(event-bus): Unsubscribe + DLQ + retry | [#826](https://github.com/zynax-io/zynax/issues/826) | ✅ Merged |
| I.5 feat(engine-adapter): wire PublishLifecycleEventActivity | [#827](https://github.com/zynax-io/zynax/issues/827) | ✅ Merged |
| I.6 test: BDD steps for event_bus.feature | [#828](https://github.com/zynax-io/zynax/issues/828) | ✅ Merged |

### M6.B Inter-Service mTLS (#464) — ✅ COMPLETE

| Story | Issue | Status |
|-------|-------|--------|
| O1 mTLS env-var cert paths + credential wiring | [#488](https://github.com/zynax-io/zynax/issues/488) | ✅ Merged (#831) |

Canvas: `docs/spdd/464-mtls/canvas.md` — Status: Implemented

### M6.C Supply Chain Hardening (#465) — ✅ COMPLETE

| Story | Issue | Status |
|-------|-------|--------|
| O1 cosign signing + SBOM (SPDX) + multi-arch in release workflows | [#489](https://github.com/zynax-io/zynax/issues/489) | ✅ Merged (#833) |

Canvas: `docs/spdd/465-supply-chain/canvas.md` — Status: Implemented

### M6.Helm — Helm Charts for All Services (#765) — ✅ COMPLETE

Canvas: `docs/spdd/765-helm-charts/canvas.md` — Status: **Implemented** ✅

All 14 O-steps merged. ✅ EPIC COMPLETE.

### M6.H — Postgres-Backed Repositories (#626) — ✅ COMPLETE

Canvas: `docs/spdd/626-postgres-repos/canvas.md` — Status: Implemented

| Story | Issue | Status |
|-------|-------|--------|
| O1 feat(task-broker): Postgres TaskRepository | [#793](https://github.com/zynax-io/zynax/issues/793) | ✅ Merged (#900) |
| O2 feat(agent-registry): Postgres AgentRepository | [#794](https://github.com/zynax-io/zynax/issues/794) | ✅ Merged (#901) |

### M6.Images — Single Source of Truth for Container Image References (#855) — ✅ COMPLETE

Canvas: `docs/spdd/855-images-sot/canvas.md` — Status: **Implemented** ✅

All 7 stories merged. Date: 2026-06-08.

| Story | Issue | Status |
|-------|-------|--------|
| O1 chore(ci): images/images.yaml schema + initial population | [#856](https://github.com/zynax-io/zynax/issues/856) | ✅ Merged (PR #913) |
| O2 feat(zynax-ci): images sync + check subcommands | [#857](https://github.com/zynax-io/zynax/issues/857) | ✅ Merged (PR #915) |
| O3 ci: wire drift-check into CI | [#858](https://github.com/zynax-io/zynax/issues/858) | ✅ Merged (PR #916) |
| O4 chore(infra): Dockerfile ARG migration | [#859](https://github.com/zynax-io/zynax/issues/859) | ✅ Merged |
| O5 chore(ci): bump flow rewrite (closes #844) | [#860](https://github.com/zynax-io/zynax/issues/860) | ✅ Merged |
| O6 docs: propagation | [#861](https://github.com/zynax-io/zynax/issues/861) | ✅ Merged |
| O7 docs: ADR-024 | [#862](https://github.com/zynax-io/zynax/issues/862) | ✅ Merged |

### M6.Images — GHCR Package Hygiene — 🔄 In Progress

Investigation confirmed (2026-06-03): all 8 GHCR images have `"annotations": null` on their OCI index manifests → "No description" in GHCR UI. Two `unknown/unknown` rows per image are SLSA provenance attestations (expected). No retention policy exists.

⚠ **Note (2026-06-08):** PR #979 (`fix(ci)`) corrects a regression where release.yml built images
without OCI description annotation — existing images in GHCR are unsigned/no SBOM. Next service
image push will include all annotations and trigger signing + SBOM again.

Delivery order (each is its own PR):

| Story | Issue | Status |
|-------|-------|--------|
| docs(adr): ADR-025 — keep vs disable SLSA attestations | [#868](https://github.com/zynax-io/zynax/issues/868) | ✅ Done |
| ci: OCI manifest annotations (fix "no description") | [#865](https://github.com/zynax-io/zynax/issues/865) | ✅ Done |
| ci: description-present gate + size-budget check | [#866](https://github.com/zynax-io/zynax/issues/866) | ✅ Done — regression fixed in PR #979 |
| docs: document unknown/unknown attestation manifests | [#869](https://github.com/zynax-io/zynax/issues/869) | ✅ Done |
| chore(ci): GHCR retention cap (last 5 builds) | [#867](https://github.com/zynax-io/zynax/issues/867) | ⬜ Open |

### M6.Build — Native Multi-Arch Build Pipeline (#837) — 🔄 In Progress

Epic: `epic(ci): M6.Build — native multi-arch build pipeline (eliminate QEMU, minimize image sizes)`

| Story | Issue | Status |
|-------|-------|--------|
| B.1 ci(infra): migrate release.yml service builds to native arm64 | [#838](https://github.com/zynax-io/zynax/issues/838) | ✅ Merged (PR #974) |
| B.2 ci(infra): migrate tools-image.yml to native arm64 | [#839](https://github.com/zynax-io/zynax/issues/839) | ✅ Merged (PR #975) |
| B.3 ci(infra): add Python adapter images to multi-arch release pipeline | [#840](https://github.com/zynax-io/zynax/issues/840) | ⬜ Open |
| B.4 ci(infra): audit and minimize final image sizes | [#841](https://github.com/zynax-io/zynax/issues/841) | ⬜ Open |

### M6.Argo — Argo Workflows Engine Adapter (#766) — 🔄 In Progress

Canvas: `docs/spdd/766-argo-engine/canvas.md` — Status: **Aligned** ✅

| Story | Issue | Status |
|-------|-------|--------|
| O1 feat(protos): ArgoConfig message + argo_engine BDD scenarios | [#795](https://github.com/zynax-io/zynax/issues/795) | 🔄 In Review (PR #976) |

### M6.NS — Namespace Routing (#767) — 🔄 In Progress

Canvas: `docs/spdd/767-namespace-routing/canvas.md` — Status: **Aligned** ✅

| Story | Issue | Status |
|-------|-------|--------|
| O1 feat(workflow-compiler): namespace-scoped capability routing | [#799](https://github.com/zynax-io/zynax/issues/799) | ✅ Merged (PR #977) |

### M6.PyPI — SDK PyPI Distribution (#769) — 🔄 In Progress

Canvas: `docs/spdd/769-pypi/canvas.md` — Status: **Aligned** ✅

| Story | Issue | Status |
|-------|-------|--------|
| O1 feat(agents): PyPI trusted publisher + TestPyPI dry-run | [#805](https://github.com/zynax-io/zynax/issues/805) | ✅ Merged (PR #973) |

### M6 Infra / Tooling — ✅ Complete

Process-health work (ADR-023, merge-policy, image-bump tooling, /resume-m6 fix).
All PRs merged 2026-06-03.

| Work Item | Issue | PR | Status |
|-----------|-------|-----|--------|
| ADR-023 — restrict direct pushes to main | — | #847 | ✅ Merged |
| chore(ci): bump-ci-runner script + make target | [#843](https://github.com/zynax-io/zynax/issues/843) | #848 | ✅ Merged |
| ci(infra): post-build bump issue from tools-image.yml | [#844](https://github.com/zynax-io/zynax/issues/844) | #849 | ✅ Merged |
| chore(claude): /resume-m6 rewrite — FF discipline | [#845](https://github.com/zynax-io/zynax/issues/845) | #850 | ✅ Merged |
| docs(contributing): merge policy | [#846](https://github.com/zynax-io/zynax/issues/846) | #851 | ✅ Merged |
| docs(milestones): tracking + ROADMAP expansion | — | #852 | ✅ Merged |

### M6.F — Platform Configuration Convergence (#670) — ✅ Complete

| Story | Issue | Status |
|-------|-------|--------|
| refactor: libs/zynaxconfig shared config + task-broker migration | [#667](https://github.com/zynax-io/zynax/issues/667) | ✅ Merged (PR #907) |
| chore(infra): Dockerfile template consolidation | [#668](https://github.com/zynax-io/zynax/issues/668) | ✅ Merged (PR #909) |
| ci: go.mod version-alignment gate | [#669](https://github.com/zynax-io/zynax/issues/669) | ✅ Merged (PR #910) |

### M6.J — memory-service KV + vector implementation (#773) — ✅ COMPLETE

Canvas: `docs/spdd/773-memory-service/canvas.md` — Status: **Implemented** ✅

All 6 stories merged 2026-06-08.

| Story | Issue | Status |
|-------|-------|--------|
| J.2 feat(memory-service): service scaffold — go.mod, domain KV+Vector interfaces, cmd/ | [#814](https://github.com/zynax-io/zynax/issues/814) | ✅ Merged (#932) |
| J.3 feat(memory-service): Redis KV adapter | [#815](https://github.com/zynax-io/zynax/issues/815) | ✅ Merged |
| J.4 feat(memory-service): pgvector adapter | [#816](https://github.com/zynax-io/zynax/issues/816) | ✅ Merged |
| J.5 feat(memory-service): namespace TTL enforcement + workflow_id isolation | [#817](https://github.com/zynax-io/zynax/issues/817) | ✅ Merged |
| J.6 feat(memory-service): gRPC handler wiring — all 10 RPCs, integration tests | [#818](https://github.com/zynax-io/zynax/issues/818) | ✅ Merged |
| J.7 test: BDD step implementations for memory_service.feature | [#819](https://github.com/zynax-io/zynax/issues/819) | ✅ Merged |

### M6.DevAuto — Self-hosting dev-automation (#873) — ✅ DELIVERED TO BOUNDARY (EPIC closing)

Canvas (Wave 4): `docs/spdd/881-self-hosted-issue-delivery/canvas.md` — Status: **Implemented**
(delivered through O7 + O9; O8 → M7 [#1103](https://github.com/zynax-io/zynax/issues/1103)).
Waves 0–3 were superseded by the generalized delivery commands and retired in #1129 (configs
archived under `docs/archive/dev-advisory/`; learnings kept in `docs/ai-learnings/`).
EPICs #873 and #881 close at the platform-readiness boundary; the strict xfail in
`automation/tests/test_platform_readiness.py` remains the honest gate, continued under M7 (#1103).

| Story | Issue | Status |
|-------|-------|--------|
| DevAuto.1 docs(automation): STATUS-AND-DIRECTION.md | [#874](https://github.com/zynax-io/zynax/issues/874) | ✅ Merged (#884) |
| DevAuto.2 chore(automation): expert mesh YAML configs | [#875](https://github.com/zynax-io/zynax/issues/875) | ✅ Merged |
| DevAuto.3 chore(automation): orchestrator config | [#876](https://github.com/zynax-io/zynax/issues/876) | ✅ Merged |
| DevAuto.4 ci: Wave 0 advisory workflow | [#877](https://github.com/zynax-io/zynax/issues/877) | ✅ Merged (PR #969) — superseded, retired in #1129 |
| DevAuto.5 ci: Wave 1 orchestrator aggregation | [#878](https://github.com/zynax-io/zynax/issues/878) | ✅ Closed — superseded (#1129) |
| DevAuto.6 ci: Wave 2 | [#879](https://github.com/zynax-io/zynax/issues/879) | ✅ Closed — superseded (#1129) |
| DevAuto.7 ci: Wave 3 | [#880](https://github.com/zynax-io/zynax/issues/880) | ✅ Closed — demoted to weekly audit (#1107/#1113) |
| DevAuto.8 feat: Wave 4 self-hosted issue-delivery | [#881](https://github.com/zynax-io/zynax/issues/881) | ✅ Delivered to boundary — O1 [#1096](https://github.com/zynax-io/zynax/issues/1096) ADR-028 · O2 [#1097](https://github.com/zynax-io/zynax/issues/1097) 9 expert AgentDefs · O3 [#1098](https://github.com/zynax-io/zynax/issues/1098) orchestrator Workflow · O4 [#1099](https://github.com/zynax-io/zynax/issues/1099) intake→plan→route · O5 [#1100](https://github.com/zynax-io/zynax/issues/1100) context-slice binding · O6 [#1101](https://github.com/zynax-io/zynax/issues/1101) delivery leg · O7 [#1102](https://github.com/zynax-io/zynax/issues/1102) learning-synthesizer · O9 [#1104](https://github.com/zynax-io/zynax/issues/1104) reconcile · O8 [#1103](https://github.com/zynax-io/zynax/issues/1103) ⏸ deferred to M7 (platform gaps — see #1103 comments) |
| DevAuto.9 test: xfail gate | [#882](https://github.com/zynax-io/zynax/issues/882) | ✅ Closed — xfail gate in place; flip continues under M7 (#1103) |
| DevAuto.10 docs: AGENTS.md pointer + README | [#883](https://github.com/zynax-io/zynax/issues/883) | ✅ Closed |

---

## Next Session Queue (priority order)

**Immediate (unblocked):**
- **#1087** feat(infra): expose api-gateway on host port for e2e (NodePort 30080) — #1086 O1 ✅ ready
- **#1071** ci(infra): Temporal-vs-Argo engine matrix in e2e-smoke — #771 O2 ✅ delivered-pending-merge (branch `ci/1071-engine-matrix-e2e`)
- **#867** chore(ci): GHCR retention cap — last 5 builds only
- **#840** ci(infra): Python adapters in multi-arch release pipeline
- **#841** ci(infra): audit and minimize image sizes

**M6.E2E-Green chain (O1 #1087 / O2 #1088 / O3 #1089 / O4 #1090 / O5 #1091 all ✅):** #1092 (O6, needs #1071 + #1149 fix for `E2E_REQUIRE_COMPLETED_EVENT`).
**M6.Postgres chain — ✅ complete:** #1076 ✅, #1077 ✅, #1078 ✅, #1079 ✅ (O1–O8 all delivered; canvas Implemented; EPIC #1073 ready to close).

**M6.Argo continuation (after #795 merges):**
- **#796** feat(engine-adapter): Argo engine adapter implementation (#766, O2)
- **#797** feat(api-gateway): engine config routing (#766, O3)

**M6.DevAuto — ✅ complete (delivered to boundary):** Wave 4 (#881) delivered through O7 + O9;
O8 #1103 deferred to M7; EPICs #881/#873 closing at the platform-readiness boundary.

**SDK/docs:**
- **#808** docs(agents): SDK docstrings step 2 (#769, O4)
- **#376** docs: SDK docstrings step 2 — BLOCKED on SDK modules (M6+ scope)
