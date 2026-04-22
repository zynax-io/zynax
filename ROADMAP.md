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

## Milestone 2 — Workflow IR

**Goal:** YAML manifests compile to a canonical, engine-agnostic Intermediate Representation.

> Label: `milestone: M2`

- [ ] JSON Schema for all manifest kinds: `Workflow`, `AgentDef`, `Policy`
- [ ] `workflow-compiler` service: Go implementation
- [ ] YAML → IR compilation: states, transitions, actions, triggers
- [ ] Schema validation: invalid YAML rejected with clear error messages and line numbers
- [ ] Semantic validation: no orphan states, terminal state required, valid capability refs
- [ ] IR serialisation (protobuf) for transmission to engine adapters
- [ ] `make validate-spec` and `make dry-run` targets
- [ ] BDD scenarios for all compiler error cases
- [ ] Reference workflow YAML examples: `code-review.yaml`, `ci-pipeline.yaml`, `research-task.yaml`

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

> Label: `milestone: M5`

- [ ] `http-adapter`: wraps any REST API — config-only, no code
- [ ] `llm-adapter`: Bedrock, Ollama, OpenAI — provider configurable
- [ ] `git-adapter`: GitHub/GitLab capabilities + webhook → event-bus
- [ ] `langgraph-adapter`: runs a LangGraph app as a capability
- [ ] `LangGraphEngine` adapter in `engine-adapter` service
- [ ] Adapter cookbook: documented patterns for CI, databases, messaging

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
