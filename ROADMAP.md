<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax Roadmap

> Zynax is the declarative control plane for AI agent workflows.
> This document is the **narrative roadmap** — it explains the goals and sequence.
> The **execution roadmap** (issue tracking, progress, and assignments) lives in the
> [GitHub Project board](https://github.com/orgs/zynax-io/projects/1).

---

## How to Read This Roadmap

- **Milestones** are goals, not dates. Each milestone is "done" when every item
  on its checklist has a merged PR.
- **Issues** on the Project board map to individual checklist items below. Each
  checklist item should correspond to a `type: feature` or `type: task` issue.
- **Versions** are cut when a milestone is complete (see `GOVERNANCE.md §6`).
- **Contributing to the roadmap**: see §9 — Propose a Roadmap Addition.

---

## GitHub Projects Setup

The board is at: **https://github.com/orgs/zynax-io/projects/1**

### Board Views

| View | Purpose |
|------|---------|
| **Kanban** | Day-to-day work: `Backlog → Ready → In Progress → In Review → Done` |
| **Milestone Table** | All open issues grouped by milestone with status |
| **Roadmap Timeline** | Milestone swimlanes for release planning |

### Issue → Roadmap Mapping

Every actionable item in this file must have a corresponding GitHub Issue:

1. Create the issue using the **Feature Request** template.
2. Set the `milestone:` label matching the milestone below (e.g., `milestone: M1`).
3. Set the `area:` label for the service or layer.
4. When implementation begins, assign to a contributor and move to "In Progress".

### Milestones on GitHub

Each roadmap milestone maps to a GitHub Milestone:

| GitHub Milestone | Roadmap Milestone | Target version |
|-----------------|------------------|---------------|
| Contracts Foundation | M1 | v0.1.0 |
| Workflow IR | M2 | v0.1.0 |
| Temporal Execution | M3 | v0.2.0 |
| YAML System + CLI | M4 | v0.3.0 |
| Adapter Library | M5 | v0.4.0 |
| K8s Production | M6 | v0.5.0 |
| Full Observability | M7 | v0.6.0 |
| CNCF Sandbox | M8 | v1.0.0 |

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

## Milestone 6 — Kubernetes Production-Ready (v0.5.0 target)

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

## Milestone 7 — Full Observability

**Goal:** End-to-end observability across all workflow execution layers.

> Label: `milestone: M7`

- [ ] Distributed traces: workflow execution → capability dispatch → adapter → LLM call
- [ ] Grafana dashboards: pre-built for all services
- [ ] Workflow execution timeline view (Gantt-style)
- [ ] Alert rules: workflow stuck, capability error rate, task queue depth
- [ ] Structured audit log: all `apply` operations, all capability invocations
- [ ] OpenCost integration: cost per workflow execution

---

## Milestone 8 — CNCF Sandbox Submission

**Goal:** Community, governance, and technical maturity for CNCF Sandbox.

> Label: `milestone: M8`

- [ ] ≥ 2 maintainers from different organisations
- [ ] External security audit (CNCF security framework)
- [ ] Production reference deployment documented and tested
- [ ] Multi-cloud E2E validation (GKE + EKS + AKS)
- [ ] CNCF TOC application filed

---

## Version Plan

| Version | Milestone(s) | Key Capability |
|---------|-------------|---------------|
| v0.1.0 | M1, M2 | Contracts + Workflow IR |
| v0.2.0 | M3 | Temporal execution |
| v0.3.0 | M4 | `zynax apply` + CLI |
| v0.4.0 | M5 | Adapter library |
| v0.5.0 | M6 | K8s production-ready |
| v0.6.0 | M7 | Full observability |
| v1.0.0 | M8 | CNCF Sandbox submission |

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
