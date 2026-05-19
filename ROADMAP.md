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

## Milestone 3 — Engine Adapters (Temporal First)

**Goal:** Workflow IR executes on Temporal. Engine abstraction proven.

> Label: `milestone: M3`

- [ ] `engine-adapter` service: Go implementation
- [ ] `WorkflowEngine` interface defined and documented
- [ ] `TemporalEngine` adapter: Submit, Signal, Query, Cancel, Watch
- [ ] Generic Temporal "state machine worker" that interprets IR at runtime
- [ ] `DispatchCapabilityActivity`: Temporal Activity that calls task-broker
- [ ] End-to-end test: YAML → IR → Temporal → capability dispatch → result

---

## Milestone 4 — YAML System + CLI

**Goal:** Users can `zynax apply workflow.yaml` and see it run.

> Label: `milestone: M4`

- [ ] `api-gateway` extended with `/api/v1/apply` endpoint (accepts YAML)
- [ ] `zynax` CLI: `apply`, `get`, `delete`, `logs`, `status` commands
- [ ] Local runner: Docker Compose-based (no Kubernetes required)
- [ ] GitOps integration: watch a git repo, apply changes on push
- [ ] `kind: AgentDef` apply: registers agent in registry + deploys adapter
- [ ] Validation feedback: clear error messages, line numbers, fix suggestions

---

## Milestone 5 — Adapter Library

**Goal:** Existing systems become capabilities without SDK adoption.

> Label: `milestone: M5` · Epic: [#377](https://github.com/zynax-io/zynax/issues/377)
> Execution plan: [docs/milestones/M5-plan.md](docs/milestones/M5-plan.md)

M5 is structured into five parallel tracks, each with a REASONS Canvas and child issues.

### M5.A — Truth Pass ([#458](https://github.com/zynax-io/zynax/issues/458))

- [x] Remove CNCF Sandbox Candidate badge (#472)
- [x] Audit CHANGELOG for phantom entries (#473)
- [ ] Python SDK decision (#474)

### M5.B — Engine Correctness Hardening ([#459](https://github.com/zynax-io/zynax/issues/459))

- [x] Fix `resolveTemplate` map-iteration non-determinism (#475)
- [ ] Replace bespoke guard parser with `cel-go` (#476)
- [x] Return full `CompilationError` list from `CompileWorkflow` (#477)
- [x] Fix SSE `WriteTimeout` breaking `zynax logs` at 30 s (#478)

### M5.C — Capability Dispatch End-to-End ([#460](https://github.com/zynax-io/zynax/issues/460))

- [x] `task-broker` MVP: in-memory `TaskBrokerService`, 5 RPCs, hexagonal layout (#479 / PRs #520 #522 #523)
- [ ] `agent-registry` MVP: in-memory `AgentRegistryService`, 5 RPCs (#480 — BDD trim → domain → wiring)
- [ ] Docker Compose wiring: task-broker + agent-registry in `make run-local` (#481)

### M5.D — Control Plane Security Baseline ([#461](https://github.com/zynax-io/zynax/issues/461)) ✅

- [x] Bearer-token auth middleware for api-gateway (#482)
- [x] Log event publish failures instead of discarding (#483)
- [x] X-Request-ID propagation across all services (#484)
- [x] Idempotent `zynax apply` — manifest hash derives stable workflow ID (#485)
- [x] Consolidate Docker Compose files (#486)

### M5.E — Developer Experience Polish ([#462](https://github.com/zynax-io/zynax/issues/462)) ✅

- [x] Idempotent apply and compose consolidation (shared with M5.D: #485 #486)

### Adapter Library ([#377](https://github.com/zynax-io/zynax/issues/377))

- [x] `http-adapter`: REST API proxy — config-only, no code (#380)
- [ ] `git-adapter`: GitHub/GitLab operations (`open_pr`, `request_review`, `get_diff`) (#381)
- [ ] `ci-adapter`: CI pipeline triggers (`trigger_workflow`, `get_run_status`) (#382)
- [ ] `llm-adapter`: OpenAI / Bedrock / Ollama inference (#383) — Python
- [ ] `langgraph-adapter`: any LangGraph graph as a named capability (#384) — Python

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
