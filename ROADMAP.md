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

## Milestone 1 — Contracts Foundation ✅ Complete (v0.1.0)

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

## Milestone 2 — Workflow IR ✅ Complete (v0.1.0)

**Goal:** YAML manifests compile to a canonical, engine-agnostic Intermediate Representation.

> See [Epic #101](https://github.com/zynax-io/zynax/issues/101) for the full issue list.

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

## Milestone 3 — Engine Adapters ⚠ Partial (v0.2.0)

**Goal:** Workflow IR executes on Temporal. Engine abstraction proven.

**Delivered:**
- [x] `engine-adapter` service: `WorkflowEngine` interface, `TemporalEngine`, `IRInterpreterWorkflow`
- [x] `DispatchCapabilityActivity`: Temporal Activity → task-broker gRPC
- [x] All 5 `EngineAdapterService` RPCs (Submit/Signal/Cancel/GetWorkflowStatus/WatchWorkflow)
- [x] cel-go guard evaluation (bespoke evaluator replaced by #538 in M5.B)

**Not delivered (moved to M5.C):**
- [ ] `task-broker` service (delivered M5.C #479)
- [ ] End-to-end capability dispatch (requires agent-registry; pending #480)
- CloudEvents publishing is a log stub (pending event-bus implementation, M6+)

---

## Milestone 4 — YAML System + CLI ⚠ Partial (v0.3.0)

**Goal:** Users can `zynax apply workflow.yaml` and see it run.

**Delivered:**
- [x] `api-gateway`: `POST /api/v1/apply`, `GET /api/v1/workflows/{id}`, `DELETE`, SSE logs
- [x] `zynax` CLI: `apply`, `get`, `delete`, `status`, `logs`
- [x] Local Docker Compose runner (`make run-local`)
- [x] GitOps watch mode (`zynax apply --watch`)

**Not delivered (moved to M5.C):**
- [ ] `agent-registry` service (#480 — required for `kind: AgentDef` routing)
- Capability dispatch: workflows submit but actions fail (no registry)

---

## Milestone 5 — Adapter Library 🔄 In Progress (v0.4.0)

**Goal:** Existing systems become capabilities without SDK adoption. First green E2E demo.

> Label: `milestone: M5` · Epic: [#377](https://github.com/zynax-io/zynax/issues/377)
> Execution plan: [docs/milestones/M5-plan.md](docs/milestones/M5-plan.md)

### M5 Definition of Done (7 criteria)
1. `make run-local && zynax apply code-review.yaml` → real state transitions + ≥1 dispatch
2. v0.4.0 tag with downloadable CLI + GHCR images
3. All 5 adapters (http ✅ + git + ci + llm + langgraph) merged
4. Python SDK `Agent` base class implemented ✅
5. cel-go replaces bespoke guard evaluator ✅
6. SECURITY.md matches shipped reality ✅
7. CI < 10 minutes per PR 🟡

### M5.A — Truth Pass ([#458](https://github.com/zynax-io/zynax/issues/458))

- [x] Remove CNCF Sandbox Candidate badge (#472)
- [x] Audit CHANGELOG for phantom entries (#473)
- [x] Python SDK Agent base class (#474 / #535 #536 #537)
- [x] Fix SECURITY.md — remove mTLS/SBOM/cosign false claims
- [ ] Add per-service status table to README (#579)

### M5.B — Engine Correctness Hardening ([#459](https://github.com/zynax-io/zynax/issues/459)) ✅

- [x] Fix `resolveTemplate` map-iteration non-determinism (#475)
- [x] Replace bespoke guard evaluator with `cel-go`, fail-closed (#476 / #538 #539 #540)
- [x] Return full `CompilationError` list from `CompileWorkflow` (#477)
- [x] Fix SSE `WriteTimeout` breaking `zynax logs` at 30 s (#478)

### M5.C — Capability Dispatch End-to-End ([#460](https://github.com/zynax-io/zynax/issues/460)) 🔴 Critical path

- [x] `task-broker` MVP: in-memory `TaskBrokerService`, 5 RPCs, 92.7% coverage (#479 / #520 #522 #523)
- [ ] `agent-registry` MVP: BDD trim (#526) → domain (#527) → gRPC wiring (#528)
- [ ] Docker Compose wiring: task-broker + agent-registry in `make run-local` (#481)

### M5.D — Control Plane Security Baseline ([#461](https://github.com/zynax-io/zynax/issues/461)) ✅

- [x] Bearer-token auth middleware (#482)
- [x] Log event publish failures (#483)
- [x] X-Request-ID propagation (#484)
- [x] Idempotent `zynax apply` — manifest hash (#485)
- [x] Consolidate Docker Compose files (#486)

### M5.E — Developer Experience Polish ([#462](https://github.com/zynax-io/zynax/issues/462)) ✅

- [x] Idempotent apply and compose consolidation (#485 #486)

### M5.F — CI/CD Performance Sprint ([#542](https://github.com/zynax-io/zynax/issues/542)) 🟡

- [x] Concurrency + stale-run cancellation (#545)
- [x] Unified release workflow — fix race condition (#557)
- [x] v0.4.0 CHANGELOG promoted; tag push pending
- [x] All service/adapter images public on GHCR (#562)
- [x] CI runner container image (#551 #552)
- [ ] Force-full-pipeline trigger (#554)
- [ ] Per-service change detection (#549 #550)

### Adapter Library ([#377](https://github.com/zynax-io/zynax/issues/377))

- [x] `http-adapter`: REST API proxy — all step issues merged (#380)
- [ ] `git-adapter`: `open_pr`, `request_review`, `get_diff` (#381 — BDD done, impl pending #481)
- [ ] `ci-adapter`: `trigger_workflow`, `get_run_status` (#382 — BDD done, impl pending #481)
- [ ] `llm-adapter`: OpenAI / Bedrock / Ollama `chat_completion` (#383 — BDD done, impl pending #481)
- [ ] `langgraph-adapter`: LangGraph StateGraph as Zynax capabilities (#384 — BDD done, impl pending #481)

---

## Milestone 6 — Kubernetes Production-Ready

**Goal:** Production deployment on Kubernetes. Argo engine support.

> Label: `milestone: M6`

- [ ] Helm charts for all services (including Temporal dependency)
- [ ] `ArgoEngine` adapter
- [ ] Kubernetes Runtime Provider (HPA, PDB, NetworkPolicy for all services)
- [ ] Multi-namespace support in workflow-compiler
- [ ] Policy enforcement: routing policies, rate limits, capability quotas
- [ ] `zynax-sdk` Python package published to PyPI

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
