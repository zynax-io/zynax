<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax M1 — Engineering Review
**Milestone:** Contracts Foundation · **Target version:** v0.1.0  
**Date:** 2026-04-21 · **Author:** Engineering

---

## Executive Summary

Milestone 1 establishes the **contractual foundation** of the Zynax distributed
AI workflow control plane. No service is implemented yet; this is intentional.
The strategic bet is that defining and machine-verifying all communication
contracts *before* any implementation begins eliminates the most expensive
class of distributed system failure: silent contract drift between services.

Every byte of production code written in M2+ has a verified contract waiting
for it. No engineer needs to read a wiki or ask what a service is supposed to
return — the answer is in a `.feature` file, enforced by a CI gate.

---

## What Zynax Is

Zynax is a **declarative control plane for AI agent workflows**. Operators
describe multi-step AI workflows in YAML. Zynax compiles those manifests,
orchestrates their execution across any runtime (Temporal, LangGraph, Argo
Workflows), and dispatches capability invocations to AI agents — without the
control plane knowing or caring which engine or agent is underneath.

The core design invariant: **the control plane is swappable at every boundary.**
Swap the execution engine without touching the compiler. Swap an AI agent
without touching the broker. Every swap point is a gRPC contract.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Layer 1 — Intent                                               │
│                                                                 │
│   YAML Manifest  ──►  API Gateway                               │
└─────────────────────────────┬───────────────────────────────────┘
                              │ CompileWorkflow(manifest_yaml)
┌─────────────────────────────▼───────────────────────────────────┐
│  Layer 2 — Control Plane (Go services)                          │
│                                                                 │
│  WorkflowCompilerService                                        │
│    YAML → WorkflowIR (engine-agnostic compiled representation)  │
│                      │ SubmitWorkflow(WorkflowIR)               │
│  EngineAdapterService ◄────────────────────────────────────     │
│    Single contract all execution engines implement              │
│    Submit · Signal · Cancel · Watch                             │
│                      │ publishes CloudEvents                    │
│  EventBusService ◄───┘                                          │
│    Async pub/sub backbone (NATS JetStream)                      │
│    Glob pattern routing · per-workflow scoping                  │
│                      │ fan-out on task.* events                 │
│  TaskBrokerService ◄─┘                                          │
│    Routes capability requests to the right agent                │
│    Dispatch · Acknowledge · Cancel · Retry state machine        │
│                      │ queries capability catalogue             │
│  AgentRegistryService ◄──────────────────────────────────────── │
│    Registry of agent capabilities                               │
│    Register · Deregister · FindByCapability · ListAgents        │
│                                                                 │
│  MemoryService                                                  │
│    Persistent KV store + vector store                           │
│    Scoped by workflow_id — agents are stateless by design       │
└──────────────────────────────────────────────────────┬──────────┘
                                                       │
┌──────────────────────────────────────────────────────▼──────────┐
│  Layer 3 — Execution                                            │
│                                                                 │
│  AgentService (Python adapters)                                 │
│    ExecuteCapability  ──►  streaming TaskEvents                 │
│    Any system implementing this RPC is a first-class agent      │
│    No SDK required · No framework requirement                   │
│                                                                 │
│  Execution Engines (via EngineAdapterService contract)          │
│    Temporal · LangGraph · Argo Workflows                        │
└─────────────────────────────────────────────────────────────────┘
```

### Request lifecycle

```
User YAML
  │
  ▼
API Gateway  ──►  WorkflowCompilerService.CompileWorkflow
                    validates structure, returns WorkflowIR
  │
  ▼
EngineAdapterService.SubmitWorkflow(WorkflowIR)
  │  run_id returned immediately, status = RUNNING
  │
  ▼  (engine executes state machine)
EngineAdapterService  ──►  EventBus.Publish(zynax.workflow.state.entered)
                              │
                              ▼
                           TaskBrokerService.DispatchTask(capability, payload)
                              │  routes via AgentRegistryService.FindByCapability
                              ▼
                           AgentService.ExecuteCapability(request)
                              │  streams TaskEvents: PROGRESS* → COMPLETED|FAILED
                              ▼
                           TaskBrokerService.AcknowledgeTask(result)
                              │
                              ▼
                           EventBus.Publish(zynax.task.completed)
                              │
                              ▼
                           EngineAdapterService (advances state machine)
```

---

## What Was Built

### 1. gRPC Contract Layer — 8 Services

| Service | RPCs | Purpose |
|---------|------|---------|
| `AgentService` | `ExecuteCapability` (streaming) | Universal capability provider contract |
| `AgentRegistryService` | `Register`, `Deregister`, `Get`, `List`, `FindByCapability` | Capability catalogue |
| `WorkflowCompilerService` | `CompileWorkflow`, `ValidateManifest` | YAML → WorkflowIR |
| `EngineAdapterService` | `Submit`, `Signal`, `Cancel`, `GetStatus`, `Watch` | Execution engine boundary |
| `EventBusService` | `Publish`, `Subscribe`, `Unsubscribe` | Async pub/sub backbone |
| `TaskBrokerService` | `Dispatch`, `Acknowledge`, `Cancel`, `GetTask`, `ListTasks` | Capability routing |
| `MemoryService` | `Set/Get/Delete/List` (KV) + `Upsert/Search/Delete` (vector) | Externalised agent state |
| `CloudEventsEnvelope` | (message type only) | CNCF CloudEvents v1.0 wire format |

All proto field numbers are permanent (ADR-001 §backward-compat). Ordinal values
on every `enum` are stable and documented. `buf breaking` in CI enforces this on
every PR.

### 2. Event Contract Layer — AsyncAPI Spec

`spec/asyncapi/zynax-events.yaml` documents all async events on the NATS
JetStream bus:

| Channel | Publisher | Consumers |
|---------|-----------|-----------|
| `zynax.workflow.started` | EngineAdapter | Observability, UI |
| `zynax.workflow.completed` | EngineAdapter | Orchestrator, billing |
| `zynax.workflow.failed` | EngineAdapter | Alerting, retry policy |
| `zynax.workflow.cancelled` | EngineAdapter | Orchestrator |
| `zynax.task.dispatched` | TaskBroker | Observability |
| `zynax.task.completed` | TaskBroker | EngineAdapter (state advance) |
| `zynax.task.failed` | TaskBroker | EngineAdapter, alerting |
| `zynax.task.retrying` | TaskBroker | Observability |
| `zynax.agent.registered` | AgentRegistry | Service mesh, UI |
| `zynax.agent.deregistered` | AgentRegistry | Service mesh |
| `zynax.agent.capability.invoked` | TaskBroker | Audit trail, billing |

Every payload is a CloudEvents v1.0 envelope (`spec/schemas/cloudevent.schema.json`)
with Zynax-specific extensions: `workflow_id`, `run_id`, `namespace`, `capability_name`.

### 3. Schema Layer

| Schema | Purpose |
|--------|---------|
| `spec/schemas/cloudevent.schema.json` | Wire format for all async events |
| `spec/schemas/capability.schema.json` | Agent capability declarations (name, input_schema, output_schema, timeout, retries) |

### 4. Generated Stubs

`buf generate` produces committed, fresness-checked stubs:

- `protos/generated/go/` — Go `*.pb.go` + `*_grpc.pb.go` for all 8 services
- `protos/generated/python/` — Python `*_pb2.py` + `*_pb2_grpc.py` for all 8 services

CI (`ci.yml`) regenerates stubs and diffs — any drift fails the build before
review begins.

### 5. BDD Contract Test Suite — 140+ Scenarios

One `godog` test suite per service, running against in-memory stubs via
`bufconn` (in-process gRPC, no Docker, <100ms total):

| Package | Scenarios | Key contracts verified |
|---------|-----------|----------------------|
| `agent_service` | — | Streaming terminal semantics, timeout, task_id echo |
| `agent_registry_service` | — | Register/deregister lifecycle, FindByCapability routing |
| `cloudevents_envelope` | — | Required fields, extension validation |
| `workflow_compiler_service` | 17 | All 7 CompilationErrorCodes, dry_run, ValidateManifest |
| `engine_adapter_service` | 24 | Full lifecycle (15) + signals/watch (9) |
| `event_bus_service` | 23 | Pub/sub fan-out, glob patterns, unsubscribe |
| `task_broker_service` | 24 | Retry state machine, cancel guards, ListTasks filters |
| `memory_service` | — | KV TTL, vector cosine similarity, namespace scoping |

These are not unit tests. They verify the **observable protocol behaviour**:
given this wire input, the server must produce this exact response code,
this payload shape, this error message. When the real Go service is implemented,
the scenarios do not change — only the stub is swapped for the real server.

### 6. CI Quality Gates

Every PR is blocked by 5 automated checks:

| Gate | Blocks on |
|------|-----------|
| `conventional-commit` | Non-standard PR title (blocks changelog generation) |
| `pr-size` | >900 lines changed (forces decomposition) |
| `proto-breaking` | Backward-incompatible proto change (field removal/rename) |
| `proto-stubs-fresh` | `.proto` changed but generated stubs not updated |
| `layer-boundaries` | `domain/` importing `api/` or `infrastructure/` |

### 7. Testing Strategy — ADR-016

The project moved from BDD-first-for-everything (ADR-004) to a layered pyramid:

```
              ▲
         BDD / E2E         10–15%   system boundaries (this sprint)
        ─────────────────────────
         Contract CI       always   buf breaking (every PR)
        ─────────────────────────
         Unit / Property    40%     domain logic (M2+ service implementations)
        ─────────────────────────
         Simulation                 chaos / load (M3+)
              ▼
```

The key insight: BDD at the *boundary* produces stable, high-value contracts.
BDD at the *domain level* produces fragile tests that couple to implementation.

---

## What This Unlocks

M1 creates the **contractual moat** that makes parallel development safe in M2:

- Any engineer implementing `WorkflowCompilerService` runs `go test ./workflow_compiler_service/...` and gets instant contract compliance feedback
- Any engineer writing an engine adapter implements `EngineAdapterService` and the 24 BDD scenarios tell them exactly what the contract requires
- Any engineer building the Python task broker runs the pytest-bdd suite (issue #31) against their implementation
- The AsyncAPI spec is the canonical source of truth for all async event consumers — no guessing event field names

---

## What Is Not Yet Built (Intentional)

| Item | Milestone | Reason deferred |
|------|-----------|-----------------|
| Service implementations (Go) | M2+ | Contracts must precede implementations |
| Python pytest-bdd harness | M1 open (#31) | Same pattern as Go, needs Python stubs |
| Docker Compose integration profile | M2 | Meaningful only when services are real |
| Coverage gate (90%) | M2 | Meaningful only when domain packages exist |
| YAML workflow manifest JSON Schema | M2 | WorkflowCompiler M2 concern |
| gRPC health checking protocol | M2/M6 | K8s readiness probes: M6 concern |
| Pagination on list RPCs | TBD | See open questions §below |

---

## Open Questions for M1 Closure

The following design decisions need answers before M1 can be formally closed
and v0.1.0 tagged:

1. **Missing AsyncAPI events**: The EventBus spec has workflow lifecycle and
   task lifecycle events, but is missing `zynax.workflow.state.entered` and
   `zynax.workflow.state.exited`. The `EngineAdapterService.WatchWorkflow`
   stream carries these over gRPC, but downstream consumers on the EventBus
   (e.g., the TaskBroker deciding when to dispatch a task) need them as
   published events. Are these M1 or M2?

2. **Stub distribution mechanism**: Generated stubs are committed to the repo
   (current approach). For M2 multi-repo consumers, should stubs also be
   published to `buf.build/zynax-io/zynax` (BSR) and/or as a Go module
   tagged release? Or is commit-and-reference sufficient for M1?

3. **Pagination on list RPCs**: `ListAgents`, `ListTasks` have no `page_token`
   / `page_size` fields. Adding these after M1 is a backward-compatible change
   (new fields), but the absence means early consumers may load unbounded
   result sets. Cap for M1 or add pagination now?

4. **gRPC health checking**: Standard production gRPC (`grpc.health.v1`) is not
   in any service contract. K8s liveness/readiness probes will need it in M6.
   Adding it in M1 is zero implementation cost (it's a proto import). Worth doing?

5. **Memory bulk operations**: `MemoryService` has per-key `Set/Get/Delete` but
   no `MGet`/`MSet`/`DeleteNamespace`. Agents running in a workflow frequently
   need to load all their context in one call. M1 addition or M2?

6. **WorkflowIR retrieval**: `WorkflowCompilerService` compiles a manifest and
   returns the IR synchronously. There is no `GetCompiledWorkflow(workflow_id)`
   RPC to retrieve a previously compiled IR. Is the IR ephemeral (caller must
   store it) or should the compiler service persist and expose it?

7. **AgentService capability introspection**: The task broker validates
   `input_schema` at dispatch time using the schema registered in
   `AgentRegistryService`. Should `AgentService` also expose a
   `GetCapabilitySchema` RPC so the broker can re-fetch live schemas without
   going through the registry?

8. **Event type versioning**: Event types are `zynax.workflow.started` (no
   version segment). Should they be `zynax.v1.workflow.started` to allow
   non-breaking evolution? Or is the CloudEvent `specversion` field sufficient?

---

## Architecture Decisions Referenced

| ADR | Title | Impact on M1 |
|-----|-------|-------------|
| ADR-001 | gRPC as inter-service protocol | All 8 service contracts are gRPC |
| ADR-008 | No shared databases | MemoryService is the only shared state store |
| ADR-009 | Language strategy | Go services + Python agents |
| ADR-010 | Pluggable agent runtime | AgentService = the single adapter contract |
| ADR-011 | Declarative YAML control plane | WorkflowCompilerService contract |
| ADR-012 | Workflow IR | WorkflowIR is the compiled, engine-agnostic representation |
| ADR-013 | Adapter-first, no SDK required | AgentService RPC is the only requirement |
| ADR-014 | Event-driven state machine | EventBusService + CloudEvents |
| ADR-015 | Pluggable workflow engines | EngineAdapterService contract |
| ADR-016 | Layered testing strategy | BDD at boundaries only |
