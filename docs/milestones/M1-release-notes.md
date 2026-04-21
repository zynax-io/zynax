<!-- SPDX-License-Identifier: Apache-2.0 -->

# Release Notes — v0.1.0 (Milestone 1: Contracts Foundation)

**Release date:** pending M1 closure  
**Branch:** `main` · **Milestone:** [Contracts Foundation (M1)](https://github.com/zynax-io/zynax/milestone/1)

---

## Overview

v0.1.0 is the **Contracts Foundation** release. It establishes every
communication contract in the Zynax distributed AI workflow control plane
before any service implementation begins.

This release contains no running services. It contains something more
fundamental: the machine-verified specifications that all future services
must honour. Every contract is expressed in protobuf, backed by BDD
scenarios, and protected by CI gates that prevent backward-incompatible
changes from reaching `main`.

---

## What's New

### gRPC Service Contracts (8 services)

All inter-service APIs are defined as versioned protobuf contracts under
`protos/zynax/v1/`:

| Service | Description |
|---------|-------------|
| `AgentService` | Universal capability provider — any system implementing `ExecuteCapability` is a first-class Zynax agent. No SDK required. |
| `AgentRegistryService` | Capability catalogue. Agents register on startup; the task broker queries at dispatch time via `FindByCapability`. |
| `WorkflowCompilerService` | Transforms user-authored YAML manifests into engine-agnostic `WorkflowIR`. Exposes all 7 `CompilationErrorCode` variants with structured error detail. |
| `EngineAdapterService` | The single contract all execution engines (Temporal, LangGraph, Argo) must implement. Submit, Signal, Cancel, Watch. |
| `EventBusService` | Async pub/sub backbone over NATS JetStream. Glob pattern routing (`zynax.*`, `zynax.**`), per-workflow scoping, server-side streaming. |
| `TaskBrokerService` | Capability routing with a full task lifecycle state machine: PENDING → DISPATCHED → COMPLETED / FAILED / RETRYING / CANCELLED. |
| `MemoryService` | Externalised agent state: per-workflow-scoped KV store (with TTL) and vector store (cosine similarity search). Agents are stateless by design. |
| `CloudEventsEnvelope` | CNCF CloudEvents v1.0 wire format with Zynax extensions (`workflow_id`, `run_id`, `namespace`, `capability_name`). |

### Async Event Contracts (AsyncAPI 2.6)

`spec/asyncapi/zynax-events.yaml` documents all 11 event channels on the
platform event bus:

```
zynax.workflow.started      zynax.task.dispatched
zynax.workflow.completed    zynax.task.completed
zynax.workflow.failed       zynax.task.failed
zynax.workflow.cancelled    zynax.task.retrying
zynax.agent.registered      zynax.agent.deregistered
zynax.agent.capability.invoked
```

### JSON Schema Layer

- `spec/schemas/cloudevent.schema.json` — CloudEvents v1.0 envelope with Zynax extensions
- `spec/schemas/capability.schema.json` — Agent capability declaration schema (name, input_schema, output_schema, timeout_seconds, max_retries)

### Generated Stubs

`make generate-protos` (via `buf generate`) produces committed stubs for both
language targets:

- **Go:** `protos/generated/go/zynax/v1/*.pb.go` and `*_grpc.pb.go`
- **Python:** `protos/generated/python/zynax/v1/*_pb2.py` and `*_pb2_grpc.py`

Stubs are committed to the repository. CI detects and rejects drift between
`.proto` sources and committed stubs on every PR.

### BDD Contract Test Suite

One `godog` test suite per gRPC service, running against in-memory stubs
via `bufconn` (in-process gRPC — no Docker, no network, <100ms total):

| Service | Scenarios |
|---------|-----------|
| WorkflowCompilerService | 17 — all 7 error codes, dry_run, ValidateManifest |
| EngineAdapterService | 24 — lifecycle (15) + signals/watch (9) |
| EventBusService | 23 — pub/sub fan-out, glob patterns, unsubscribe |
| TaskBrokerService | 24 — retry state machine, cancel guards, filters |
| MemoryService | — KV TTL, vector similarity, namespace scoping |
| AgentService | — streaming terminal semantics, timeout, task_id echo |
| AgentRegistryService | — registration lifecycle, FindByCapability |
| CloudEventsEnvelope | — required fields, extension validation |

### CI Quality Gates

Five automated checks run on every pull request:

| Check | Enforces |
|-------|----------|
| `conventional-commit` | PR title follows Conventional Commits (enables semantic versioning) |
| `pr-size` | ≤900 lines changed (forces reviewable decomposition) |
| `proto-breaking` | No backward-incompatible proto changes via `buf breaking` |
| `proto-stubs-fresh` | Generated stubs are regenerated when `.proto` files change |
| `layer-boundaries` | Three-layer separation: `domain/` never imports `api/` or `infrastructure/` |

### Testing Strategy — ADR-016

The project adopted a layered testing pyramid (replaces BDD-first ADR-004):

- **BDD/contract tests** at system boundaries (this milestone)
- **Unit/property tests** for domain logic (M2+ service implementations)
- **buf breaking** as always-on contract CI
- **Simulation/chaos** for end-to-end confidence (M3+)

---

## Breaking Changes

None. This is the first release.

---

## Migration Guide

Not applicable. This is the first release.

---

## Known Limitations

- No running services — M1 is contracts only
- No pagination on `ListAgents`, `ListTasks` list RPCs
- No gRPC health checking protocol (`grpc.health.v1`) — planned M6
- Python pytest-bdd contract harness pending (issue #31)
- AsyncAPI spec missing `zynax.workflow.state.entered` / `zynax.workflow.state.exited` events
- No selective contract test execution (all 8 suites run on every proto change)
- Generated stubs not published to buf.build BSR or package registries

---

## Issues Closed

### Proto Contracts
- #2 — AgentService proto: ExecuteCapability streaming RPC
- #3 — AgentRegistryService proto: agent registration and capability discovery
- #4 — TaskBrokerService proto: capability routing and task lifecycle
- #5 — WorkflowCompilerService proto: YAML manifest compilation contract
- #6 — EngineAdapterService proto: workflow execution lifecycle
- #7 — MemoryService proto: shared KV and vector storage contract
- #8 — EventBusService proto: async event publish/subscribe contract
- #9 — CloudEvents event envelope: proto definition and JSON Schema

### Specifications
- #10 — AsyncAPI spec: all platform async event types documented
- #11 — Capability schema: JSON Schema for capability input/output declarations
- #22 — CloudEvents envelope: proto and JSON Schema

### Code Generation
- #12 — buf generate pipeline: Go and Python proto stubs from a single make target
- #30 — Commit initial generated proto stubs

### CI Infrastructure
- #15 — GitHub Actions pipeline: lint, BDD execution, security, PR checks
- #27 — Dockerfile.tools on Alpine: proto and BDD tools

### Contract Tests (godog BDD)
- #42 — Test module bootstrap: godog harness, bufconn testserver, go.mod
- #43 — AgentService contract tests: ExecuteCapability streaming
- #44 — AgentRegistryService contract tests: registration and discovery
- #45 — CloudEventsEnvelope contract tests: field validation
- #46 — MemoryService contract tests: KV store, vector search, TTL
- #47 — WorkflowCompilerService contract tests: YAML compilation
- #48 — EventBusService contract tests: publish, subscribe, unsubscribe
- #49 — TaskBrokerService contract tests: dispatch, retry, lifecycle
- #50 — EngineAdapterService contract tests: lifecycle (submission, status, cancellation)
- #51 — EngineAdapterService contract tests: signals, watch stream, validation

---

## What Comes Next — M2 (Workflow IR)

M2 implements the `WorkflowCompilerService` in Go, replacing the BDD in-memory
stub with a real YAML parser and IR compiler. It also:

- Adds JSON Schema for `Workflow`, `AgentDef`, and `Policy` manifest kinds
- Implements IR serialisation (protobuf) for transmission to engine adapters
- Adds reference workflow YAML examples
- Connects the `make validate-spec` and `make dry-run` targets to real logic

See [ROADMAP.md §M2](../../ROADMAP.md) for the full checklist.

---

## Architecture Decision Records

All ADRs referenced in this release:
[ADR-001](../adr/ADR-001-grpc-inter-service-protocol.md) ·
[ADR-008](../adr/ADR-008-no-shared-databases.md) ·
[ADR-009](../adr/ADR-009-language-strategy.md) ·
[ADR-010](../adr/ADR-010-pluggable-agent-runtime.md) ·
[ADR-011](../adr/ADR-011-declarative-yaml-control-plane.md) ·
[ADR-012](../adr/ADR-012-workflow-ir.md) ·
[ADR-013](../adr/ADR-013-adapter-first-no-sdk.md) ·
[ADR-014](../adr/ADR-014-event-driven-state-machine.md) ·
[ADR-015](../adr/ADR-015-pluggable-workflow-engines.md) ·
[ADR-016](../adr/ADR-016-layered-testing-strategy.md)
