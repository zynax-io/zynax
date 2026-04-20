# Keel — Architecture

> This document explains WHY the architecture is designed this way.
> For WHAT to build and HOW to code it, see `AGENTS.md`.
> For each service's internal design, see `services/<service>/AGENTS.md`.

---

## 1. Core Philosophy

### The Control Plane Analogy

Kubernetes didn't build a new container runtime. It built a control plane
that abstracts container runtimes (Docker, containerd, CRI-O) behind a
declarative API.

Keel does the same for AI workflows:

| Kubernetes | Keel |
|-----------|-----------|
| Container | Capability |
| Pod spec | AgentDef YAML |
| Deployment | Workflow YAML |
| kubelet | Engine Adapter |
| etcd | Agent Registry |
| kube-scheduler | Task Broker |

We don't build workflow engines. We build the control plane that orchestrates them.

---

## 2. The Three-Layer Model

```
┌──────────────────────────────────────────────────────────┐
│  LAYER 1 — INTENT (YAML)                                 │
│  Declarative. Versionable. No code.                      │
│  Inspired by: Kubernetes CRDs, Helm, GitOps              │
│                                                          │
│  kind: Workflow | AgentDef | Policy | RoutingRule        │
├──────────────────────────────────────────────────────────┤
│  LAYER 2 — COMMUNICATION (Contracts)                     │
│  Typed. Multi-language. Source of truth.                 │
│  Inspired by: gRPC ecosystem, AsyncAPI, CloudEvents      │
│                                                          │
│  proto files → sync (gRPC)                              │
│  AsyncAPI spec → async (NATS JetStream)                 │
├──────────────────────────────────────────────────────────┤
│  LAYER 3 — EXECUTION (Engines + Adapters)                │
│  Pluggable. Swappable. Never a hard dependency.          │
│  Inspired by: Temporal, LangGraph, Argo Workflows        │
│                                                          │
│  Temporal adapter | LangGraph adapter | Argo adapter     │
│  HTTP adapter | LLM adapter | Git adapter                │
└──────────────────────────────────────────────────────────┘
```

---

## 3. Workflow Intermediate Representation (IR)

### The Problem

Different engines speak different languages:
- Temporal: Activities + Workflows in Go/Python
- LangGraph: StateGraph + Nodes in Python
- Argo: YAML DAGs in Kubernetes

A workflow defined for Temporal cannot run on LangGraph. Without IR,
every engine requires a different workflow definition format.

### The Solution: Canonical Workflow IR

```
YAML (user intent)
      ↓
  Workflow Compiler
      ↓
  Canonical IR         ← engine-agnostic representation
      ↓
Engine Adapter         ← translates IR → engine-native format
      ↓
Temporal / LangGraph / Argo
```

The IR is the "LLVM of workflows". It normalises semantics that differ
across engines: loops, conditional transitions, timeout handling,
human-in-the-loop signals, event subscriptions.

### IR Schema (conceptual)

```go
// services/workflow-compiler/internal/domain/ir.go

type WorkflowIR struct {
    ID           string
    Version      string
    InitialState string
    States       map[string]StateIR
    EventSchema  map[string]EventSchemaIR
}

type StateIR struct {
    Name        string
    Type        StateType  // ACTIVE | TERMINAL | HUMAN_IN_THE_LOOP | WAITING
    Actions     []ActionIR
    Transitions []TransitionIR
    Timeout     *time.Duration
}

type ActionIR struct {
    Capability  string            // what capability to invoke
    InputMap    map[string]string // maps workflow context → capability input
    OutputMap   map[string]string // maps capability output → workflow context
    Async       bool
}

type TransitionIR struct {
    OnEvent string
    Guard   *ExpressionIR // optional condition
    Goto    string
}
```

---

## 4. Event-Driven State Machine Model

### Why State Machines, Not DAGs?

Most workflow engines model workflows as DAGs (directed acyclic graphs).
DAGs work well for batch processing but fail for real AI workflows:

| Property | DAG | State Machine |
|---------|-----|--------------|
| Loops | ❌ Requires workarounds | ✅ Native |
| Human-in-the-loop | ❌ Breaks the graph | ✅ WAITING state |
| Long-running (days) | ⚠️ Timeout issues | ✅ Event-driven |
| Async events | ❌ Complex | ✅ First-class transitions |
| Error recovery | ❌ Manual | ✅ Via transitions |

Real AI workflows loop. A code review workflow is:
`review → fix → review → fix → review → merge`

This cannot be expressed cleanly as a DAG.

### State Machine Semantics

States have:
- **Actions**: capabilities to invoke when entering the state
- **Transitions**: event → next state mappings
- **Type**: ACTIVE (running) | WAITING (human) | TERMINAL (done)

Events are the only mechanism for state transitions. No polling.
No timer-based checks. Everything is event-driven.

---

## 5. Capability Model

### Why Capabilities, Not Named Agents?

Classical agent systems route tasks to named agents:
`task → agent:analyst-01`

Keel routes tasks to capabilities:
`task → capability:summarize`

**The difference:**
- Named routing: tight coupling. Swap an agent = rewrite the workflow.
- Capability routing: loose coupling. Add a better summarizer = zero workflow changes.

### Capability Resolution Flow

```
Workflow YAML:
  actions:
    - capability: summarize

Workflow Compiler → IR:
  ActionIR{Capability: "summarize", ...}

Task Broker:
  1. Query agent-registry: "Who has capability: summarize?"
  2. Apply routing policy (round-robin, least-loaded, affinity)
  3. Dispatch to selected agent/adapter
  4. Stream results back as events

Event Bus:
  emit: task.completed {capability: "summarize", result: ...}
```

---

## 6. Engine Adapter Architecture

### The Interface

Every engine adapter implements ONE Go interface:

```go
// services/engine-adapter/internal/domain/engine.go

type WorkflowEngine interface {
    // Submit a workflow IR for execution
    Submit(ctx context.Context, ir WorkflowIR, input map[string]any) (ExecutionID, error)

    // Signal a running workflow (inject an event)
    Signal(ctx context.Context, id ExecutionID, event WorkflowEvent) error

    // Query current state of a workflow execution
    Query(ctx context.Context, id ExecutionID) (*ExecutionState, error)

    // Cancel a running workflow
    Cancel(ctx context.Context, id ExecutionID, reason string) error

    // Watch for execution state changes (server-streaming)
    Watch(ctx context.Context, id ExecutionID) (<-chan ExecutionEvent, error)

    // Name of this engine (for routing decisions)
    Name() string
}
```

### Semantic Translation Challenge

Each engine has different semantics. The adapter is responsible for
translating IR to engine-native format AND back. Known mismatches:

| Semantic | Temporal | LangGraph | Argo |
|---------|---------|-----------|------|
| Loop | Recursive workflow | Graph cycle | Loop template |
| Human signal | `workflow.Signal()` | Interrupt + resume | Suspend + resume |
| Timeout | `workflow.WithTimeout()` | `interrupt_after` | `activeDeadlineSeconds` |
| Parallel | `workflow.Go()` | Parallel nodes | DAG tasks |

Each adapter in `services/engine-adapter/internal/adapters/` handles this translation.

---

## 7. Adapter-Based Integration (No SDK)

### The Problem with SDK-Required Architectures

If Keel requires an SDK, adoption is:
- Language-limited (only SDK languages work)
- Framework-coupled (upgrade SDK = upgrade all agents)
- High-friction (non-engineering teams can't participate)

### The Adapter Solution

Any system becomes a capability by deploying an adapter that:
1. Implements the `AgentService` gRPC contract
2. Registers capabilities in `agent-registry`
3. Handles `ExecuteCapability` RPCs

```
Existing system           Adapter                Keel
─────────────────         ──────────────         ─────────────
Bedrock API      →   llm-adapter        →   capability: summarize
GitHub API       →   git-adapter        →   capability: open_mr
Jenkins API      →   ci-adapter         →   capability: run_tests
Internal API     →   http-adapter       →   capability: call_payments
LangGraph app    →   langgraph-adapter  →   capability: research_topic
```

Adapters are thin. They translate between Keel contracts and
the external system's native protocol. They contain no business logic.

---

## 8. Communication Architecture

### Sync Path (gRPC)
- Task execution requests
- Capability invocations
- State queries
- Config/manifest applies

### Async Path (NATS JetStream + AsyncAPI)
- Workflow state change events
- Task lifecycle events
- Agent heartbeats
- System signals (human-in-the-loop)
- CI/CD event integrations

### AsyncAPI Spec
Every async event is documented in `spec/asyncapi/`. The AsyncAPI spec is:
- The contract for all async communication
- Validated in CI (asyncapi-cli lint)
- Generated into Go event types (analogous to proto → Go stubs)

---

## 9. Runtime Abstraction

Kubernetes is optional. Keel supports:

| Runtime | Use Case |
|---------|---------|
| Local (Docker Compose) | Development, testing |
| Kubernetes | Production, scale |
| Cloud APIs (ECS, Cloud Run) | Serverless deployment |

The `api-gateway` exposes `keel apply` which accepts YAML manifests
regardless of the underlying runtime. Local dev and production accept
identical YAML — the compiler and runtime layer handle the difference.

---

## 10. Data Flows

### Workflow Execution Flow

```
1. User: keel apply workflow.yaml
2. API Gateway: validate auth → forward to Workflow Compiler
3. Workflow Compiler: parse YAML → validate schema → compile to IR → select engine
4. Engine Adapter: translate IR → Temporal workflow → submit to Temporal
5. Temporal: executes workflow, calls capabilities via Task Broker
6. Task Broker: route capability call → dispatch to registered adapter
7. Adapter: execute (LLM call, API call, git op) → stream results
8. Event Bus: workflow emits state change events
9. Memory Service: adapters store/retrieve context
10. Observability: all steps traced, metered, logged
```

### Event Flow

```
External event (GitHub push) → git-adapter → Event Bus
Event Bus → Workflow Compiler (matches event to workflow trigger)
Workflow Compiler → signals running workflow via Engine Adapter
Engine Adapter → Temporal.Signal("push")
Temporal → transitions workflow state: fix → review
```

---

## 11. Language Interoperability

### The Protocol Is the Contract

Every integration point in Keel is defined in `protos/keel/v1/`. This is not
an implementation detail — it is a deliberate architectural guarantee. The proto
contract is the only thing that two systems need to agree on to work together.
Neither system needs to know what language, framework, or runtime the other uses.

This guarantee scales to every layer of the architecture:

- The Go workflow-compiler sends a compiled IR to the Go engine-adapter using
  a proto message. A future Rust engine-adapter would receive the identical message.
- The Go task-broker dispatches an `ExecuteCapabilityRequest` to whatever agent
  registered that capability. The agent may be Python, Go, TypeScript, or Java —
  the broker sends the same proto message regardless.
- A TypeScript web client sends `ApplyWorkflowRequest` to the Go API Gateway.
  The gateway processes a proto message. Language is invisible.

### Language Roles in the Platform

Go and Python are not equal in this architecture — they play different roles:

**Go owns the platform** (Layers 1 and 2). The workflow compiler, engine adapters,
task broker, agent registry, memory service, event bus, and API gateway are all Go.
These components handle control-plane concerns: state management, routing, scheduling,
contract validation. Go's performance characteristics, concurrency model, and type
system make it the right choice for this layer.

**Python owns the execution** (Layer 3). The AI/ML ecosystem — LangGraph, AutoGen,
CrewAI, Transformers, and every major AI framework — is Python-native. Python agents
and adapters are where intelligence is applied. The SDK provides ergonomic access to
the platform from this layer.

**Any language can participate in Layer 3**. The proto contract does not care about
Python. A Go adapter, a Java adapter, a Rust high-performance inference engine, and a
TypeScript API adapter are all equal participants in Layer 3. They implement the same
`AgentService` contract and receive the same task dispatch.

### The Import Hierarchy

| Component | How it consumes the proto contract |
|-----------|-----------------------------------|
| Go platform services (internal) | Import from `gen/go/keel/v1/` via `go.work` workspace |
| External Go consumers | Import `github.com/keel-io/keel/gen/go/keel/v1` via `go.mod` |
| Python SDK agents | `keel-sdk` wraps `protos/generated/python/` — proto is abstracted |
| Python raw-stub callers | Import directly from `protos/generated/python/` |
| Other languages | Run `buf generate` against `protos/keel/v1/` source |
| Future BSR consumers | Import from the Buf Schema Registry (planned for M1) |

The generated stubs in `gen/go/` and `protos/generated/python/` are committed to
the repository. They are regenerated on every proto change by `make generate-protos`.
They are never edited manually. They change when and only when the proto source changes.

### Why No Go SDK

A question that arises naturally: if Python has a SDK, why not Go?

Go platform services call each other using the generated stubs directly. The gRPC
interface, the generated client structs, and the Go type system together provide
everything a higher-level SDK would add. There is no registration boilerplate to
eliminate because Go gRPC clients are already minimal. There is no framework
integration to abstract because Go developers building capabilities implement the
`AgentService` interface directly — that is the idiomatic Go approach.

The Python SDK exists because the Python gRPC boilerplate for a server-role agent
(registration, heartbeat, streaming lifecycle, context injection) is meaningful
friction for developers whose primary skill is AI framework usage, not gRPC server
implementation. The SDK eliminates that friction. Go developers do not have the same
friction because implementing a gRPC server interface in Go is already straightforward
and idiomatic.

### Interoperability in Practice

A realistic end-to-end capability dispatch crosses languages multiple times:

1. A TypeScript CI dashboard submits a workflow via the API Gateway (TypeScript stubs
   calling Go service via gRPC).
2. The Go workflow-compiler compiles the YAML to IR and selects the Temporal engine.
3. The Go engine-adapter submits the IR to Temporal as a Go workflow.
4. Temporal executes the workflow and reaches a `summarize` capability action.
5. The Go engine-adapter calls the Go task-broker's `DispatchCapability` RPC.
6. The Go task-broker queries the Go agent-registry and finds a Python SDK agent
   registered with capability `summarize`.
7. The Go task-broker calls the Python SDK agent's `ExecuteCapability` RPC.
8. The Python SDK agent runs a LangGraph graph, reads from the Go memory service,
   and streams `TaskEvent` responses back to the broker.
9. The broker forwards results to the engine-adapter, which signals Temporal.
10. Temporal transitions the workflow state and emits an event to the Go event-bus.
11. The event-bus forwards the event to the TypeScript dashboard via a WebSocket
    bridge or gRPC stream.

At no point does any component know what language the others are written in.
The proto contracts are the only visible interfaces.

---

## 12. Milestones

See `ROADMAP.md` for the full timeline. Architecture aligns with:

| Milestone | Architecture Component |
|-----------|----------------------|
| M1 — Contracts | protos/ + spec/asyncapi/ |
| M2 — Workflow IR | services/workflow-compiler/ |
| M3 — Engine Adapters | services/engine-adapter/ + Temporal first |
| M4 — YAML System | spec/schemas/ + compiler → IR |
| M5 — Adapter Layer | agents/adapters/ (http, llm, git) |
| M6 — Runtime + CLI | `keel apply` + local runner |
| M7 — Observability | OTel traces across all layers |
