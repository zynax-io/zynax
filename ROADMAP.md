<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax Roadmap

> **Write your agent workflow once — run it on Temporal or Argo without a rewrite.**
> Zynax is the engine-portability layer for agentic automation: the best declarative
> YAML manifest layer over interchangeable workflow engines, with portability proved
> by a conformance suite — not claimed (the "honest YAML layer", 2026-05 architectural
> review Fork A; canonical framing in [docs/product/positioning.md](docs/product/positioning.md)).
> This document is the **narrative roadmap** — it explains the goals and sequence.
> The **execution roadmap** (issue tracking, progress, and assignments) lives in
> [GitHub Milestones](https://github.com/zynax-io/zynax/milestones), the per-milestone
> plans under `docs/milestones/`, and [state/current-milestone.md](state/current-milestone.md).

---

## How to Read This Roadmap

- **Milestones** are goals, not dates. Each milestone is "done" when every item
  on its checklist has a merged PR.
- **Issues** on the Project board map to individual checklist items below. Each
  checklist item should correspond to a `type: feature` or `type: task` issue.
- **Versions** are cut when a milestone is complete (see `GOVERNANCE.md §6`).
- **Contributing to the roadmap**: see §9 — Propose a Roadmap Addition.

---

## Execution Tracking

> **Note (2026-07-08):** execution tracking moved from GitHub Projects to **GitHub
> Milestones + `docs/milestones/<name>-planning.md` + `state/milestone.yaml`** during M6.
> The M5-era board ([org project #2](https://github.com/orgs/zynax-io/projects/2)) is
> historical and no longer maintained.

### Issue → Roadmap Mapping

Every actionable item in this file must have a corresponding GitHub Issue:

1. Create the issue using the **Feature Request** template.
2. Set the `milestone:` label matching the milestone below (e.g., `milestone: M9`)
   AND assign the GitHub milestone.
3. Set the `area:` label for the service or layer.
4. When implementation begins, the delivery flow sets `status: in-progress` + assignee.

### Milestones on GitHub

Each roadmap milestone maps to a GitHub Milestone:

| GitHub Milestone | Roadmap Milestone | Version shipped/target |
|-----------------|------------------|---------------|
| Contracts Foundation (#1) | M1 | v0.1.0 |
| Workflow IR (#2) | M2 | v0.1.0 |
| Temporal Execution (#3) | M3 | v0.2.0 |
| YAML System + CLI (#4) | M4 | v0.3.0 |
| Adapter Library (#5) | M5 | v0.4.0 |
| K8s Production (#6) | M6 | v0.5.0 |
| Usable Workflows + Observability (#7) | M7 (reframed — first-run UX closeout) | v0.7.0¹ |
| CNCF Sandbox (#8) | M8 (+ thin-Zynax reduction) | v0.7.0¹ |
| Hard Removals + Conformance (#11) | **M9 — active** | v0.8.0 |
| Developer Experience (#9) | M-dx (contributor / SDK / AI tooling) | unversioned program bucket² |
| User Experience (#10) | M-UX (forward UX program) | unversioned program bucket² |

> ¹ **M7 and M8 shipped together as one signed v0.7.0 release on 2026-07-10** — M7 closed at
> 0 open issues; its v0.6.0 target was skipped to keep tags monotonic. v1.0.0 is reserved
> for CNCF **acceptance** (the M8 milestone covers submission *prep*; filing is a maintainer
> action). GitHub milestone numbers ≠ M-numbers from M9 on (M9 = milestone **#11**) —
> `state/milestone.yaml` `github_milestone_number` is the source of truth.
>
> ² **UX program (2026-06-18, re-sequenced by events):** the plan was M7 → M-UX → M-dx → M8;
> in practice M8's thin-Zynax reduction and M9's hard removals were executed immediately after
> M7. M-dx/M-UX remain planned program buckets (skeletons: `docs/milestones/M-dx-planning.md`,
> `docs/milestones/M-UX-planning.md`); their work ships inside whichever numbered release is
> active when it lands. See [docs/product/2026-06-18-ux-roadmap-realignment.md](docs/product/2026-06-18-ux-roadmap-realignment.md).

---

## Milestone 1 — Contracts Foundation (v0.1.0)

**Goal:** All communication contracts defined. Nothing builds on sand.

> Released 2026-04-21. See [Engineering Review](docs/milestones/M1-engineering-review.md)
> and [Release Notes](docs/milestones/M1-release-notes.md).

- [x] gRPC proto definitions: `AgentService`, `AgentRegistryService`, `TaskBrokerService`, `MemoryService`, `EventBusService`, `WorkflowCompilerService`, `EngineAdapterService`
- [x] AsyncAPI spec: all async events documented (workflow events, task events, agent events)
- [x] Event envelope schema: `CloudEvents`-compatible wrapper for all events
- [x] Capability schema: JSON Schema for capability input/output declarations
- [x] `buf generate` produces Go stubs + Python stubs in one command
- [x] Contract tests: every proto method has a BDD scenario (140+ scenarios across all services)

---

## Milestone 2 — Workflow IR (v0.1.0)

**Goal:** YAML manifests compile to a canonical, engine-agnostic Intermediate Representation.

- [x] JSON Schema for all manifest kinds: `Workflow`, `AgentDef`, `Policy`
- [x] `workflow-compiler` service: Go implementation
- [x] YAML → IR compilation: states, transitions, actions, triggers
- [x] Schema validation: invalid YAML rejected with clear error messages and line numbers
- [x] Semantic validation: no orphan states, terminal state required, valid capability refs
- [x] IR serialisation (protobuf) for transmission to engine adapters
- [x] `make validate-spec` and `make dry-run` targets
- [x] BDD scenarios for all compiler error cases
- [x] Reference workflow YAML examples: `code-review.yaml`, `ci-pipeline.yaml`, `research-task.yaml`

---

## Milestone 3 — Engine Adapters (v0.2.0)

**Goal:** Workflow IR executes on Temporal. Engine abstraction proven.

**Delivered:**
- [x] `engine-adapter` service: `WorkflowEngine` interface, `TemporalEngine`, `IRInterpreterWorkflow`
- [x] `DispatchCapabilityActivity`: Temporal Activity → task-broker gRPC
- [x] All 5 `EngineAdapterService` RPCs (Submit/Signal/Cancel/GetWorkflowStatus/WatchWorkflow)
- [x] cel-go guard evaluation (bespoke evaluator replaced in M5.B)

**Not delivered in M3** (completed later):
- `task-broker` service (delivered in M5.C)
- End-to-end capability dispatch (required agent-registry; delivered in M5.C)
- CloudEvents publishing was a log stub (event-bus delivered in M6)

---

## Milestone 4 — YAML System + CLI (v0.3.0)

**Goal:** Users can `zynax apply workflow.yaml` and see it run.

**Delivered:**
- [x] `api-gateway`: `POST /api/v1/apply`, `GET /api/v1/workflows/{id}`, `DELETE`, SSE logs
- [x] `zynax` CLI: `apply`, `get`, `delete`, `status`, `logs`
- [x] Local Docker Compose runner (`make run-local`)
- [x] GitOps watch mode (`zynax apply --watch`)

**Not delivered in M4** (completed later):
- `agent-registry` service, required for `kind: AgentDef` routing (delivered in M5.C)
- End-to-end capability dispatch (delivered in M5.C)

---

## Milestone 5 — Adapter Library (v0.4.0)

**Goal:** Existing systems become capabilities without SDK adoption. First green E2E demo.

> Released 2026-05-29. Label: `milestone: M5`
> Full plan and per-track delivery detail: [docs/milestones/M5-plan.md](docs/milestones/M5-plan.md)

### M5 Definition of Done (7/7 criteria met)

1. [x] `make run-local && zynax apply spec/workflows/examples/e2e-demo.yaml` → `WORKFLOW_STATUS_COMPLETED`
2. [x] v0.4.0 tag with downloadable CLI + GHCR images (released 2026-05-29)
3. [x] All 5 adapters merged (http, git, ci, llm, langgraph)
4. [x] Python SDK `Agent` base class implemented
5. [x] cel-go replaces bespoke guard evaluator
6. [x] SECURITY.md matches shipped reality
7. [x] CI < 10 minutes per PR

### M5 tracks delivered

- **M5.A — Truth Pass**: docs aligned with shipped reality; per-service status table in README
- **M5.B — Engine Correctness Hardening**: deterministic template resolution, cel-go fail-closed guards, full `CompilationError` lists, SSE timeout fix
- **M5.C — Capability Dispatch End-to-End**: `task-broker` + `agent-registry` MVPs, Docker Compose wiring
- **M5.D — Control Plane Security Baseline**: bearer-token auth, X-Request-ID propagation, idempotent `zynax apply`
- **M5.E — Developer Experience Polish**: idempotent apply, compose consolidation
- **M5.F — CI/CD Performance Sprint**: concurrency cancellation, unified release workflow, ci-runner container, per-service change detection
- **Adapter Library**: `http-adapter`, `git-adapter`, `ci-adapter`, `llm-adapter`, `langgraph-adapter`

---

## Milestone 6 — Kubernetes Production-Ready (v0.5.0 — ✅ released 2026-06-12)

**Goal:** Production deployment on Kubernetes. Argo engine support.

> Label: `milestone: M6` · Target: v0.5.0 · Plan: [docs/milestones/M6-planning.md](docs/milestones/M6-planning.md)
> Live per-EPIC status: [state/current-milestone.md](state/current-milestone.md)

**Scope — process health:**
- K8s startup/readiness/liveness probe semantics across services
- Stateless workflow-compiler (no in-memory IR store)
- Inter-service mTLS — env-var cert paths + gRPC credential wiring
- Supply chain hardening — cosign signing, SPDX SBOM, multi-arch release images
- Merge discipline (ADR-023) + ci-runner bump tooling + merge policy docs

**Scope — feature EPICs:**
- EventBusService — NATS JetStream gRPC wrapper (ADR-022)
- Helm charts for all services + subcharts
- Postgres-backed repositories — horizontal scale
- Config convergence — env-var canonicalisation
- Container image source-of-truth — `images.yaml` + drift gate (ADR-024)
- Self-hosting dev-automation — orchestrator + expert mesh
- Orchestrator concurrency hardening — worktree isolation + idempotent dispatch
- `ArgoEngine` adapter + multi-engine dispatch
- Multi-namespace support in workflow-compiler
- Policy enforcement: routing policies, rate limits, capability quotas
- Prometheus `/metrics` per-request counters in all services (OTel traces → M7)
- `zynax-sdk` Python package published to PyPI
- Memory service — Redis KV + pgvector context
- End-to-end harness — kind + Helm + reference workflows
- Native multi-arch build pipeline — QEMU eliminated
- gRPC Health Checking Protocol in all services
- e2e-green: e2e-smoke gate executes a workflow end-to-end
- Postgres off deprecated Bitnami images (ADR-026)
- CI-E2E: e2e smoke + upgrade gate on infra/services changes
- Self-hosted issue-delivery engine (DevAuto Wave 4)

---

## Milestone 7 — Usable Workflows + Observability ✅ Complete (v0.7.0, released 2026-07-10)

**Goal:** A developer authors a real multi-step workflow, runs it locally (`docker compose up`
incl. Uptrace), and watches it execute state→state with data-flow, streamed logs, and a connected
distributed trace in the Uptrace UI — with green `make ci`. First of the M7 → M-dx → M8 program.

> Label: `milestone: M7` · GitHub milestone #7: **0 open / 172 closed, closed** · Shipped in **v0.7.0**
> (v0.6.0 skipped — see the version footnote above) · Plan: [docs/milestones/M7-planning.md](docs/milestones/M7-planning.md)
> Live status: [state/current-milestone.md](state/current-milestone.md)

- [x] Workflow data-flow — output/input bindings across steps (keystone, #1167)
- [x] Execution log/event streaming — `/logs` + `zynax logs --follow` (#468)
- [x] Observability — OTEL + Uptrace traces/metrics/logs/APM, compose & Helm (#467)
- [x] Context propagation — trace, data, and correlation across all hops (#1168)
- [x] Git MCP shim over git-adapter capabilities (#1169)
- [x] Expert-agent substrate + `agents/examples` reference agents (#1170)
- [x] Reusable templates + first real runnable workflows (#1171)
- [x] Quality & supply-chain fixes — audit closeout (#1172)
- [x] Test rigor — benchmarks, fuzz tests, request correlation (#469)
- [x] Quick-start, authoring, and observability docs (#1173)

**First-run UX closeout (reframed 2026-06-18).** M7's remaining work is the **first-run User
Experience**, owned by the canonical epic **#1370** — *clone (or no-clone) → one command → meaningful
result → configure your own scenario declaratively (workflow + AgentDef + context injection)* on a
local model (**Qwen2.5-Coder 3B** default). Absorbs #1359 (Day-0 engine) and #1360 (`make demo`)
from M-dx. Stories: #1371–#1381, #1385–#1388. Map: [docs/product/2026-06-18-ux-roadmap-realignment.md](docs/product/2026-06-18-ux-roadmap-realignment.md).

---

## Milestone M-UX — User Experience 🆕

**Goal:** The **forward** User-Experience program — experience Zynax's value with **no clone**, with
intelligent context-loading at scale and a discoverable Documentation Portal.

> Label: GitHub milestone "User Experience (M-UX)" (#10) · Planning skeleton:
> [docs/milestones/M-UX-planning.md](docs/milestones/M-UX-planning.md)

- [ ] No-clone try-it / hosted playground path (epic #1389)
- [ ] Intelligent context-loading architecture — metadata + required/optional/lazy policy (#1389)
- [ ] Documentation Portal — Diátaxis restructure (epic #1390)

---

## Milestone M-dx — Developer Experience

**Goal:** Make contributing and building on Zynax delightful — distinct from end-user UX.

> Label: GitHub milestone "Developer Experience (M-dx)" (#9) · Planning skeleton:
> [docs/milestones/M-dx-planning.md](docs/milestones/M-dx-planning.md)

- [ ] Contributor Experience — fast-lane, PR ergonomics, automation (epic #1391)
- [ ] SDK & Adapter-Author Experience (epic #1392)
- [ ] AI methodology / KB / technical-excellence (existing #205, #173, #148)

---

## Milestone 8 — CNCF Sandbox Submission ✅ Complete (v0.7.0, released 2026-07-10)

**Goal:** Community, governance, and technical maturity for CNCF Sandbox — and the
thin-Zynax K8s-native reduction (ADR-040: scheduler on CRDs, Workflow CRD front-end,
edge auth/rate-limit on Gateway API, admission-policy allow-list, direct JetStream
eventing).

> Label: `milestone: M8` · Shipped in **v0.7.0** (jointly with M7; milestone #8 closed) · Plan:
> [docs/milestones/M8-planning.md](docs/milestones/M8-planning.md)

- [x] Governance honesty: `MAINTAINERS.md` + single-maintainer operating mode (#494)
- [x] Troubleshooting guide + curated `good first issue` entry points (#495)
- [x] Thin-Zynax reduction epics (M8.C–M8.H: ADR-039/041/043/044/045/046)
- [x] M8.I — GitHub merge queue live since 2026-07-08 (epic #1680, stories #1681–#1685,
      ADR-047); epic closed 2026-07-10 under milestone M9 with fork-canary evidence (#1685)
- [ ] CNCF Sandbox application prepared and filed (submission is a maintainer action;
      prep: [docs/cncf/sandbox-submission.md](docs/cncf/sandbox-submission.md))
- [ ] ≥ 2 maintainers from different organisations *(Sandbox nice-to-have, required
      for Incubation — community building continues past M8)*

### Engine-portability conformance (Fork A)

The portability claim is **proved, not asserted**: the e2e matrix runs the same
workflow manifests (`spec/workflows/examples/`) through the identical
compile→IR→dispatch path on **Temporal and Argo** on every infra/service PR
(`e2e-smoke.yml`), including the Workflow CRD GitOps path. Formalising this into a
named, versioned **conformance suite** (published pass/fail matrix per engine per
release) is the follow-on milestone item below.

## Milestone 9 — Hard removals + conformance suite 🚧 Active

**Goal:** Delete the paths deprecated across M8 (agent-registry push registration,
the EventBusService facade) per each ADR's removal clause, and formalise the
dual-engine e2e into a named conformance suite.

> Label: `milestone: M9` · GitHub milestone **#11** · Target **v0.8.0** · Plan:
> [docs/milestones/M9-planning.md](docs/milestones/M9-planning.md)

- [ ] agent-registry push path hard-removal (ADR-039) — epic #1674
- [ ] `services/event-bus/` facade hard-removal (ADR-046) — epic #1675
- [ ] Named engine-conformance suite over the existing dual-engine e2e — epic #1692

---

## Version Plan

| Version | Milestone(s) | Key Capability |
|---------|-------------|---------------|
| v0.1.0 | M1, M2 | Contracts + Workflow IR |
| v0.2.0 | M3 | Temporal execution |
| v0.3.0 | M4 | `zynax apply` + CLI |
| v0.4.0 | M5 | Adapter library |
| v0.5.0 | M6 | K8s production-ready |
| ~~v0.6.0~~ | — | Skipped — M7 was never tagged at v0.6.0; kept tags monotonic |
| v0.7.0 | **M7 + M8** | Usable workflows + observability · thin-Zynax reduction + CNCF Sandbox prep (one joint signed release — released 2026-07-10) |
| v0.8.0 | M9 | Hard removals + conformance suite |
| v1.0.0 | CNCF acceptance | Cut when the Sandbox application is accepted |

> M-dx and M-UX are unversioned program buckets — their work ships inside whichever
> numbered release is active when it lands.

---

## Proposing a Roadmap Addition

1. Open a `Feature Request` issue.
2. Set `milestone:` label for the appropriate milestone, or "unscheduled".
3. The maintainers will evaluate it at the next triage cycle.
4. Accepted items are added to this document and the GitHub Project board.

Large additions (new milestones, reprioritisation of existing milestones) require
a GitHub Discussion tagged `roadmap` with 3-day lazy consensus among maintainers.
See `GOVERNANCE.md §7`.

---

## What Is NOT on the Roadmap

Items that have been considered and explicitly not included:

| Item | Decision | ADR |
|------|----------|-----|
| Zynax as an LLM framework | Out of scope — Zynax is the control plane, not the intelligence | ADR-011 |
| DAG-based workflows | Event-driven state machines chosen over DAGs | ADR-014 |
| SDK required for agents | Adapter-first — no SDK required | ADR-013 |
| Single workflow engine lock-in | Pluggable engine architecture | ADR-015 |
