<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Architecture

> This document explains the current architecture **as it is today** and **why** it
> is designed this way. For HOW to code within it, see `AGENTS.md`. For per-service
> internals, see `services/<service>/AGENTS.md`. For open decisions, see `docs/adr/`.
>
> **Authoritative tie-breaker:** `docs/architecture/2026-05-20-principal-architect-review.md`
> wins over this document on any conflict; reconcile both when updating.

---

## Milestone Status

| Milestone | Status | Version | What was built |
|-----------|--------|---------|---------------|
| M1 — Contracts Foundation | ✅ **Complete** | v0.1.0 | 8 proto contracts, AsyncAPI spec (11 channels), JSON schemas, Go + Python stubs, 140+ BDD scenarios, 5 CI gates |
| M2 — Workflow IR | ✅ **Complete** | v0.1.0 | YAML parser, WorkflowGraph builder, structural + semantic validators, WorkflowGraph→WorkflowIR serialisation, `CompileWorkflow`/`ValidateManifest`/`GetCompiledWorkflow` gRPC, JSON schemas for Workflow/AgentDef/Policy |
| M3 — Temporal Execution | ⚠ **Partial** | v0.2.0 | `WorkflowEngine` interface, `TemporalEngine`, `IRInterpreterWorkflow` state machine, `DispatchCapabilityActivity`, cel-go guard evaluation, 5 `EngineAdapterService` gRPC methods. **Not delivered in M3:** task-broker + agent-registry (delivered M5.C #479/#480). CloudEvents publish is a log stub. |
| M4 — YAML System + CLI | ⚠ **Partial** | v0.3.0 | api-gateway REST layer, `zynax` CLI, Docker Compose runner, GitOps watch. **Not delivered in M4:** agent-registry routing — delivered M5.C (#480); capability dispatch was unblocked by compose wiring (#481). |
| M5 — Adapter Library | ✅ **Complete** | v0.4.0 | task-broker MVP, agent-registry MVP, compose wiring, all five adapters (http ✅ git ✅ ci ✅ llm ✅ langgraph ✅), cel-go guard, Python SDK `Agent` base class, unified release pipeline, CI runner, distroless images, gRPC deadlines, e2e-demo wired. Released 2026-05-29. See `docs/milestones/M5-plan.md`. |
| M6 — K8s Production | 📅 **Planned** | v0.5.0 | TLS/mTLS (ADR-020), SBOM+cosign, persistent stores, rate limiting, OTel baseline, Helm charts |
| M7 — Full Observability | 📅 **Planned** | v0.6.0 | Benchmarks, load tests, SLOs, Watch polling fix |
| M8 — CNCF Sandbox | 📅 **Planned** | v1.0.0 | Community traction, second maintainer, trademark policy |

See `ROADMAP.md` for the narrative roadmap and `docs/milestones/M5-plan.md` for active execution details.

---

## 1. Core Philosophy

### The Control Plane Analogy

Kubernetes didn't build a new container runtime. It built a control plane
that abstracts container runtimes (Docker, containerd, CRI-O) behind a
declarative API.

Zynax does the same for AI workflows:

| Kubernetes concept | Zynax equivalent |
|---|---|
| Container | Capability |
| Pod spec | AgentDef YAML |
| Deployment | Workflow YAML |
| kubelet | Engine Adapter |
| etcd | Agent Registry |
| kube-scheduler | Task Broker |

Zynax does NOT build workflow engines. It builds the **control plane** that orchestrates them.

### Three-Layer Separation (Non-Negotiable)

```
┌──────────────────────────────────────────────────────────┐
│  LAYER 1 — INTENT (YAML)                                 │
│  Declarative · Versionable · No code                     │
│  spec/workflows/ · spec/schemas/                         │
│  Inspired by: Kubernetes CRDs, Helm, GitOps              │
├──────────────────────────────────────────────────────────┤
│  LAYER 2 — COMMUNICATION (Contracts)                     │
│  Typed · Multi-language · Source of truth                │
│  protos/zynax/v1/ · spec/asyncapi/                       │
│  Sync: gRPC   Async: NATS JetStream (AsyncAPI spec)     │
├──────────────────────────────────────────────────────────┤
│  LAYER 3 — EXECUTION (Engines + Adapters)                │
│  Pluggable · Swappable · Never a hard dependency         │
│  services/engine-adapter/ · agents/adapters/             │
│  Temporal (default) · LangGraph · Argo (planned)         │
└──────────────────────────────────────────────────────────┘
```

**Layer violations are hard CI failures** (`layer-boundaries` gate).  
Layer 1 YAML is never imported by Go services.  
Layer 2 contracts contain no business logic.  
Layer 3 engines are always behind the `WorkflowEngine` interface.

---

## 2. Runtime Architecture (Current State)

```
            ┌─────────────────────────┐
            │  zynax CLI (Go)          │
            │  apply / get / delete    │
            └──────────┬──────────────┘
                       │ HTTP/REST :7080
            ┌──────────▼──────────────┐
            │  api-gateway  ✅         │  bearer-token auth (constant-time ✅ #567)
            │  POST /api/v1/apply      │  X-Request-ID propagation
            │  GET  /api/v1/workflows  │  ReadHeaderTimeout ✅ #568
            └──────┬──────────┬───────┘
                   │          │ gRPC (insecure ⚠ — no TLS yet)
          ┌────────▼──────┐ ┌─▼─────────────────┐
          │ workflow-     │ │ engine-adapter ✅   │
          │ compiler ✅   │ │ TemporalEngine      │
          │ ⚠ unbounded   │ │ IRInterpreterWorkflow│
          │   IR map #466 │ │ ⚠ CloudEvents stub  │
          └───────────────┘ └─────────┬───────────┘
                                      │ gRPC: DispatchCapabilityActivity
                                      ▼
                            ┌─────────────────────┐
                            │ task-broker 🟡        │  in-memory only (Postgres in M6)
                            │ round-robin dispatch  │  wired in compose (#481 ✅)
                            └──────────┬────────────┘
                                       │ FindByCapability + ExecuteCapability gRPC
                                       ▼
                  ┌────────────────────────────────────────┐
                  │ agent-registry  🟡 In-memory MVP ✅       │
                  │ event-bus       ❌ 0 LoC (M6+)          │
                  │ memory-service  ❌ 0 LoC (M6+)          │
                  └────────────────────────────────────────┘
                                       │
                  ┌────────────────────────────────────────┐
                  │ Adapters (Layer 4 — Python + Go)        │
                  │ http-adapter ✅ (Go)                     │
                  │ git-adapter  ✅ (Go)                     │
                  │ ci-adapter   ✅ (Go)                     │
                  │ llm-adapter  ✅ (Python)                 │
                  │ langgraph    ✅ (Python)                 │
                  └────────────────────────────────────────┘
```

**Legend:** ✅ Implemented · 🟡 Partial / in-progress · ❌ Not yet implemented · ⚠ Known issue with issue link

> **M5.C complete:** End-to-end capability dispatch is fully wired. task-broker and
> agent-registry are in the compose stack (#481 ✅). Run `make run-local && zynax apply
> spec/workflows/examples/e2e-demo.yaml` to observe a full dispatch round-trip.

---

## 3. Workflow Intermediate Representation (IR)

### The Problem

Different engines speak different languages:
- Temporal: Activities + Workflows in Go/Python
- LangGraph: StateGraph + Nodes in Python
- Argo: YAML DAGs in Kubernetes

Without IR, every engine requires a different workflow definition format.

### The Solution: Canonical Workflow IR

```
YAML (user intent)
      ↓
  Workflow Compiler
      ↓
  Canonical IR         ← engine-agnostic protobuf struct
      ↓
Engine Adapter         ← translates IR → engine-native format
      ↓
Temporal / LangGraph / Argo
```

### IR Proto Schema (M2, complete)

```protobuf
// protos/zynax/v1/workflow_compiler.proto
message WorkflowIR {
    string workflow_id   = 1;
    string version       = 2;
    string target_engine = 3;
    bytes  ir_payload    = 4;  // kept for backward compat (M1); prefer structured fields

    // M2 structured fields:
    string           initial_state = 5;
    repeated StateIR states        = 6;
}

message StateIR {
    string               name            = 1;
    StateType            type            = 2;  // ACTIVE | TERMINAL | WAITING
    repeated ActionIR    actions         = 3;
    repeated TransitionIR transitions    = 4;
    int32                timeout_seconds = 5;
}

message ActionIR {
    string            capability = 1;
    map<string,string> input_map = 2;
    map<string,string> output_map = 3;  // ⚠ parsed but not yet consumed (#581)
    bool              async      = 4;
}

message TransitionIR {
    string on_event   = 1;
    string guard      = 2;  // CEL expression (cel-go, fail-closed — M5.B #538)
    string goto_state = 3;
}
```

> `ir_payload` (field 4) is kept for backward compatibility. Engine adapters should use
> structured fields (5–6) when `ir_version` is present. Planned for removal by v1.0
> (see ADR-012 proposed update).

---

## 4. Event-Driven State Machine Model

### Why State Machines, Not DAGs?

| Property | DAG | State Machine |
|---------|-----|--------------|
| Loops | ❌ Requires workarounds | ✅ Native |
| Human-in-the-loop | ❌ Breaks the graph | ✅ WAITING state |
| Long-running (days) | ⚠ Timeout issues | ✅ Event-driven |
| Async events | ❌ Complex | ✅ First-class transitions |
| Error recovery | ❌ Manual | ✅ Via transitions |

The code-review workflow naturally loops: `review → fix → review → fix → merge`.
This cannot be cleanly expressed as a DAG.

### Event Pattern Classification (Fowler taxonomy)

| Flow | Fowler Pattern | Current status |
|---|---|---|
| `zynax.workflow.state.entered/exited/completed/failed` | **Event Notification** | Log stub (#460 / M5.C) |
| `task.completed` with result payload | **Event-Carried State Transfer** | Log stub (event-bus pending) |
| Temporal activity history | **Event Sourcing** (Temporal-internal) | ✅ Temporal provides this |
| `DispatchCapabilityActivity` → task-broker | **Command** (gRPC) | ✅ Correct — not an event |

The system uses Event Notification for lifecycle events (minimal coupling, consumers call back for
state) and Event-Carried State Transfer for task results (consumers get payload without calling back).
This is the right design for a control plane. See `docs/architecture/2026-05-20-principal-architect-review.md §B1`.

---

## 5. Capability Model

### Why Capabilities, Not Named Agents?

```
Named routing (tight):      task → agent:analyst-01
Capability routing (loose): task → capability:summarize
```

Named routing breaks when an agent is replaced. Capability routing decouples the workflow
definition from any specific executor — swap a summarizer, zero workflow changes.

### Capability Resolution Flow (implemented in M5.C)

```
Workflow YAML:
  actions:
    - capability: summarize

Workflow Compiler → IR:
  ActionIR{Capability: "summarize", ...}

Task Broker:
  1. Query agent-registry: FindByCapability("summarize")
  2. Apply routing policy (round-robin for M5; least-loaded in M6)
  3. Dispatch → selected agent/adapter via ExecuteCapability gRPC

Event Bus (when implemented):
  emit: task.completed {capability: "summarize", result: ...}
```

---

## 6. Engine Adapter Architecture

### The WorkflowEngine Interface

```go
// services/engine-adapter/internal/domain/engine.go
type WorkflowEngine interface {
    Submit(ctx context.Context, ir WorkflowIR, input map[string]any) (ExecutionID, error)
    Signal(ctx context.Context, id ExecutionID, event WorkflowEvent) error
    GetWorkflowStatus(ctx context.Context, id ExecutionID) (*ExecutionState, error)
    Cancel(ctx context.Context, id ExecutionID, reason string) error
    Watch(ctx context.Context, id ExecutionID) (<-chan ExecutionEvent, error)
    Name() string
}
```

This 6-method interface is a crown jewel of the architecture — it genuinely decouples the IR
execution from any engine. Adding `ArgoEngine` or `LangGraphEngine` is ~500 LoC, not a rewrite.
**Never change the interface shape without an ADR** (ADR-015).

### Current Engines

| Engine | Status | Notes |
|---|---|---|
| `TemporalEngine` | ✅ Implemented | Only production engine today |
| `LangGraphEngine` | 📅 Planned | canvas: `docs/spdd/384-langgraph-adapter/canvas.md` |
| `ArgoEngine` | 📅 Planned | Not yet scoped |

### Activity RetryPolicy Note

Temporal's default Activity retry is exponential backoff with no max attempts. An explicit
`RetryPolicy` is set on `DispatchCapabilityActivity` (3 max attempts, non-retryable on
`ErrCapabilityNotFound` and gRPC `NOT_FOUND` — fixed #569).

---

## 7. Adapter-Based Integration (No SDK Required)

Any system becomes a capability by deploying an adapter that:
1. Implements the `AgentService` gRPC contract (`protos/zynax/v1/agent.proto`)
2. Registers capabilities in `agent-registry` via heartbeat
3. Handles `ExecuteCapability` RPCs

```
Existing system    Adapter          Capability name
───────────────    ──────────────   ──────────────
Bedrock API    →   llm-adapter   →  chat_completion
GitHub API     →   git-adapter   →  open_pr / get_diff
Jenkins API    →   ci-adapter    →  trigger_workflow
Internal API   →   http-adapter  →  (any name)
LangGraph app  →   langgraph-adapter → (graph node names)
```

The Python SDK (`agents/sdk/`) provides an optional `Agent` base class and `@capability`
decorator for Python adapters. Go and non-Python adapters implement `AgentService` directly.

---

## 8. Communication Architecture

### Synchronous Path (gRPC — implemented)

```
zynax CLI → api-gateway → workflow-compiler  (compile)
zynax CLI → api-gateway → engine-adapter    (submit/status/cancel)
engine-adapter → task-broker                (dispatch capability)
task-broker → agent-registry               (find capability)
task-broker → adapter                       (execute capability)
```

All gRPC currently uses `insecure.NewCredentials()`. TLS-by-default is planned for M6
(ADR-020 / #488). The `ZYNAX_DEV_INSECURE=1` environment variable will gate plain-text
in development once TLS is implemented.

### Asynchronous Path (NATS JetStream — stub)

```
engine-adapter → NATS (event-bus) → subscribers
```

11 AsyncAPI event channels are defined in `spec/asyncapi/`. NATS is in the compose stack.
`PublishLifecycleEventActivity` currently emits a WARN log and returns nil — no events
are actually published. Event bus implementation is planned for M6.

---

## 9. Hexagonal Service Architecture (Internal)

Every implemented service follows the same internal structure:

```
services/<service>/
  internal/
    api/           ← gRPC handler layer (receives calls, delegates to domain)
    domain/        ← business logic (ZERO gRPC/proto imports)
    infrastructure/ ← concrete implementations (DB, gRPC clients, Temporal SDK)
  cmd/<service>/   ← main.go — wire everything up
```

**The domain package has zero proto imports.** This is verified by the `layer-boundaries` CI gate
and confirmed by the 2026-05-20 review. The `IRInterpreter`'s `Run()` method depends only on
two domain interfaces (`ActivityExecutor`, `EventPublisher`). The Temporal SDK appears only in
`internal/infrastructure/`. This is textbook hexagonal architecture.

---

## 10. Service LoC Inventory (as of 2026-05-29)

| Service | Code LoC | Test LoC | Status |
|---|---:|---:|---|
| workflow-compiler | ~1,390 | ~1,855 | ✅ Implemented |
| engine-adapter | ~1,143 | ~1,209 | ✅ Implemented |
| api-gateway | ~1,047 | ~1,071 | ✅ Implemented |
| task-broker | ~905 | ~455 | 🟡 In-memory MVP |
| agent-registry | ~565 | ~691 | 🟡 In-memory MVP |
| event-bus | 0 | 0 | ❌ Stub |
| memory-service | 0 | 0 | ❌ Stub |
| cmd/zynax | ~1,470 | ~1,200 | ✅ Implemented |
| cmd/zynax-ci | ~854 | ~729 | ✅ Implemented |

Test-to-code ratio ~2:1 — excellent. Coverage gate ≥ 90% on all `internal/domain/` packages.

---

## 11. Language Interoperability

### Language Roles

**Go owns the platform** (Layers 1–3). All platform services handle control-plane concerns:
state management, routing, scheduling, contract validation.

**Python owns execution** (Layer 4). The AI/ML ecosystem (LangGraph, AutoGen, Transformers)
is Python-native. Python adapters are where intelligence is applied.

**Any language can participate in Layer 4.** The `AgentService` gRPC contract is language-neutral.
Go, TypeScript, Java, and Rust adapters are equal participants.

### The Import Hierarchy

| Consumer | How it uses the proto contracts |
|---|---|
| Go platform services (internal) | `gen/go/zynax/v1/` via `go.work` workspace |
| External Go consumers | `github.com/zynax-io/zynax/gen/go/zynax/v1` via `go.mod` |
| Python SDK agents | `agents/sdk/` `Agent` base class + `@capability`; uses `protos/generated/python/` |
| Python raw-stub callers | `protos/generated/python/` directly |
| Other languages | `buf generate` against `protos/zynax/v1/` |

Generated stubs in `gen/go/` and `protos/generated/python/` are committed. Regenerated by
`make generate-protos`. Never edited manually.

---

## 12. Contract Test Strategy

### How It Works

- Each proto service has `protos/tests/<service>/` with godog steps and an in-process
  `testserver.NewBufconnServer` — no network ports, no teardown races, parallel-safe
- Feature files in `protos/tests/features/` — **written before any implementation** (ADR-016)
- CI runs all 140+ scenarios on every PR that touches `protos/`

### GOWORK=off Requirement

**All `go test` and `go build` commands inside `services/*/`, `cmd/zynax/`,
`cmd/zynax-ci/`, and `protos/tests/` require `GOWORK=off`.**

```bash
GOWORK=off go test ./... -race -timeout 60s
```

`go.work` references modules that interact unexpectedly with standalone module directories.
See ADR-017 and `docs/decisions/004-gowork-off-isolation.md`.

---

## 13. Known Architectural Limitations (Open Issues)

| Limitation | Severity | Tracked by | Target milestone |
|---|---|---|---|
| workflow-compiler IR store unbounded in-memory | High | #466 | M6 |
| All inter-service gRPC insecure (no TLS) | High | #488 / ADR-020 | M6 |
| No OTel tracing on api-gateway + engine-adapter | High | #491 | M6 |
| CloudEvents publishing is a log stub | High | #460 | M6 |
| No rate limiting on POST /apply | Medium | #580 | M6 |
| No SBOM / cosign / SLSA provenance | High | #489 | M6 |

See `docs/reviews/04-architecture-gaps.md` for the full ranked gap list.

---

## 14. Key ADR References

| Decision | ADR |
|---|---|
| gRPC as inter-service protocol | ADR-001 |
| Language strategy (Go/Python) | ADR-009 |
| No shared databases | ADR-008 |
| Declarative YAML control plane (Layer 1 isolation) | ADR-011 |
| Workflow IR as canonical representation | ADR-012 |
| Pluggable workflow engines (WorkflowEngine interface) | ADR-015 |
| Layered testing strategy (BDD + unit + buf breaking) | ADR-016 |
| GOWORK=off contract test isolation | ADR-017 |
| SPDD prompt governance (Canvas before code) | ADR-019 |
| Zero-trust intra-service security (proposed) | ADR-020 (pending #240) |
| Horizontal scale + multi-tenancy (proposed) | ADR-021 (pending #578) |

Full ADR register: `docs/adr/INDEX.md`.

---

## 15. Architecture Review References

| Document | Date | Summary |
|---|---|---|
| `docs/architecture/2026-04-30-competitive-analysis.md` | 2026-04-30 | Competitive landscape (Temporal, Dapr, Argo, LangGraph, Kagent) |
| `docs/architecture/2026-04-30-execution-architecture.md` | 2026-04-30 | Execution architecture deep-dive (engine-adapter + Temporal) |
| `docs/architecture/2026-05-18-external-architectural-review.md` | 2026-05-18 | External architectural review |
| `docs/architecture/2026-05-20-principal-architect-review.md` | 2026-05-20 | **Authoritative** — 6.5/10 overall, G1-G24 gap list, 30-day plan |
