<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Execution Architecture

**Document type:** Architecture Reference  
**Status:** Accepted — governs M3 implementation decisions  
**Related:** `competitive-analysis-2026.md` · ADR-012 · ADR-014 · ADR-015

---

## 1. The Three-Boundary Model

Every unit of work crosses exactly three boundaries. Engine-agnosticism is
preserved at each one — nothing downstream knows about what's upstream.

```
┌─────────────────────────────────────────────┐
│  YAML Manifest (Layer 1)                    │
│  kind: Workflow  ·  states  ·  capabilities │
└───────────────────┬─────────────────────────┘
                    │  WorkflowCompilerService
                    │  parse → validate → compile
┌───────────────────▼─────────────────────────┐
│  WorkflowIR (Layer 2)                       │
│  states · actions · transitions · guards    │
│  Protobuf · versioned · engine-agnostic     │
└──────────┬──────────────┬───────────────────┘
           │              │
    ┌──────▼──────┐  ┌────▼────────┐  ...
    │  Temporal   │  │  LangGraph  │
    │  Adapter    │  │  Adapter    │
    └──────┬──────┘  └─────────────┘
           │  (Layer 3 — Execution Engine)
           │  TaskBroker.DispatchTask
    ┌──────▼──────────────────────────────────┐
    │  Agents (stateless capability executors) │
    │  AgentService.ExecuteCapability (gRPC)   │
    └─────────────────────────────────────────┘
```

Temporal does not know YAML exists. Agents do not know Temporal exists.
The engine adapter is the only component that knows both the IR and the engine SDK.

---

## 2. Layer 1 → 2: YAML to WorkflowIR

The `WorkflowCompilerService` transforms YAML in four sequential phases:

```
manifest_yaml
  → ParseManifest()         # YAML syntax, type coercion, line numbers
  → Build(manifest)         # WorkflowGraph: reachability, references
  → ValidatorPipeline.Run() # Structural + semantic invariants
  → ToIR(graph)             # WorkflowIR protobuf
```

**What the IR contains:**

| IR field | Content | Set by |
|----------|---------|--------|
| `workflow_id` | UUID v4 | Compiler at compile time |
| `initial_state` | Starting state ID | Compiler from manifest |
| `states[]` | `StateIR` — sorted by ID for determinism | Compiler |
| `states[].actions[]` | `ActionIR` — capability name, timeout, input template | Compiler |
| `states[].transitions[]` | `TransitionIR` — event_type, target_state, CEL guards | Compiler |
| `ir_version` | `"v1"` until breaking IR change ships | Compiler |
| `ir_payload` | Opaque serialized bytes (full IR) | Compiler |

**Compile-time guarantees** (catch errors before any execution begins):

- At least one terminal state reachable from `initial_state`
- No orphan states (BFS from initial state covers all)
- All `goto` targets resolve to known states
- All capability names are valid `snake_case`
- All event names follow `domain.resource.action` convention
- CEL guard expressions are syntactically valid
- No unguarded transition ambiguity within a state

---

## 3. Layer 2 → 3: How Engines Execute WorkflowIR

### 3.1 The IR Interpreter Pattern

The engine adapter does **not** code-generate one engine-specific workflow per
YAML file. It runs a **single generic interpreter** that receives the IR as
runtime data:

```
EngineAdapter.SubmitWorkflow(WorkflowIR)
  → Engine: start IRInterpreterWorkflow(ir, run_id)
```

The interpreter maintains:
- `currentState` — active state ID
- `ctx` — context map (key-value accumulator across states)

For each state it:

```
1. Publish: zynax.workflow.state.entered (CloudEvent → NATS)
2. For each action in currentState.actions:
   a. Resolve input template against ctx ({{ .ctx.key }}, {{ .trigger.field }})
   b. Call DispatchCapabilityActivity(capability, resolved_input, workflow_id)
   c. Activity: TaskBroker.DispatchTask → agent → result event + payload
3. Match result event_type against currentState.transitions
4. Apply matched transition's set{} mutations to ctx
5. Publish: zynax.workflow.state.exited (CloudEvent → NATS)
6. Advance currentState = matched_transition.target_state
7. Repeat until currentState is terminal
8. Publish: zynax.workflow.completed / zynax.workflow.failed
```

### 3.2 Capability Result → Transition Routing

The `TransitionIR.event_type` is matched against the agent's result. The agent's
terminal `TaskEvent.payload` (JSON) **must include** an `_event` field:

```json
{"_event": "review.approved", "decision": "approved", "feedback": "..."}
```

The engine adapter:
1. Reads `_event` from the agent's COMPLETED payload
2. Matches against `transition.event_type` in the current state
3. Evaluates optional CEL `conditions` (guards) against `ctx`
4. If no `_event` field: defaults to `<capability_name>.completed` or
   `<capability_name>.failed` (safe default for linear single-transition states)

**Open design decision for M3:** add `event_expression` to `ActionIR` — a CEL
expression evaluated against the agent output to derive the event type
(e.g., `event_expression: "result.status"` where agent returns
`{"status": "review.approved"}`). More explicit than the `_event` convention;
backward-compatible proto addition. Decide before M3 implementation begins.

### 3.3 External Events and Human-in-the-Loop

For `human_in_the_loop` states, the interpreter **blocks** waiting for an
external signal instead of dispatching an activity:

```
State type: human_in_the_loop
  → Interpreter: wait for signal channel / external event

External system (GitHub webhook, Slack button, CLI)
  → API Gateway
  → EngineAdapter.SignalWorkflow(run_id, event_type, payload)
  → Engine: inject signal into running workflow instance
  → Interpreter: receives signal, evaluates transitions, advances state
```

Engine-specific signal mechanisms:

| Engine | Signal mechanism | Interpreter receives |
|--------|-----------------|---------------------|
| Temporal | `client.SignalWorkflow()` + `workflow.GetSignalChannel()` | Signal value |
| LangGraph | `HumanMessage` interrupt | Message content |
| Argo | `argo resume` / suspend step | Resume trigger |

The `EngineAdapterService.SignalWorkflow` gRPC call hides all of this. Callers
always send the same request regardless of which engine is active.

---

## 4. Layer 3 → Capabilities: The Dispatch Chain

When the interpreter executes an action, `DispatchCapabilityActivity` performs:

```
1. TaskBroker.DispatchTask(capability_name, workflow_id, input_json, timeout)
   → Broker assigns task_id (caller must not supply)
   → Broker: AgentRegistry.FindByCapability(capability_name)
   → Registry returns: []{agent_id, endpoint, status: REGISTERED}
2. Broker selects agent (assignment strategy: round-robin or least-loaded)
3. Broker opens gRPC stream: Agent.ExecuteCapability(request_id, capability_name,
                                                      task_id, workflow_id,
                                                      input_payload, timeout)
4. Agent streams TaskEvents: PROGRESS... then COMPLETED or FAILED (exactly one terminal)
5. On COMPLETED: Broker transitions task to COMPLETED, stores result_payload
6. On FAILED with retry_count < max_retries: Broker transitions to RETRYING,
   re-dispatches (exponential backoff + jitter)
7. On FAILED with retries exhausted: Broker transitions to FAILED
8. Activity returns: {event_type: from result payload, payload: result_payload}
```

**Invariants the activity relies on:**
- Task IDs are broker-assigned; agents never see a pre-specified ID
- Agent result streams end with exactly one terminal event
- Agent must honour `timeout_seconds`; emit FAILED with code TIMEOUT if exceeded
- Agents are stateless — all cross-invocation context is in `MemoryService`

---

## 5. Memory Integration

### 5.1 Model

Memory is the only shared state within a workflow. All agents are stateless
twelve-factor services. Cross-agent, cross-state context flows exclusively
through `MemoryService`, scoped to `workflow_id`.

```
State: search
  → Agent: search_web(topic)
    context.memory.set("raw_results", results)     # MemoryService.Set(workflow_id, ...)
    return {_event: "search.completed"}

State: summarize
  → Agent: summarize(...)
    raw = context.memory.get("raw_results")        # MemoryService.Get(workflow_id, ...)
    context.memory.store_vector(embedding, text)   # MemoryService.StoreVector(...)
    return {_event: "summarize.completed", summary: "..."}

State: human_review
  → Agent: notify_human(...)
    similar = context.memory.query_vector(emb, 5)  # MemoryService.QueryVector(...)
```

### 5.2 Storage Planes

| Plane | Backend | Use case |
|-------|---------|---------|
| **KV** | Redis | Fast ephemeral state, agent scratch-pads, session data, TTL-based expiry |
| **Vector** | PostgreSQL + pgvector | Semantic search, RAG retrieval, long-term workflow memory |

Backends are invisible to agents and engines. Swapping Redis → another KV store
or pgvector → another vector DB requires no agent or engine changes — only
`MemoryService` implementation changes.

### 5.3 Isolation Guarantees

- `workflow_id` is required on every `MemoryService` call (enforced by proto contract)
- KV and vector namespaces are isolated per `workflow_id`
- Agents cannot read another workflow's namespace
- `DeleteNamespace` is called when a workflow reaches a terminal state (cleanup)

---

## 6. Responsibility Matrix

| Component | Owns | Does not own |
|-----------|------|-------------|
| **WorkflowCompiler** | YAML parsing, IR generation, compile-time validation | Execution, routing, memory |
| **EngineAdapter (Temporal)** | IR → Temporal translation, signal delivery, status streaming | YAML, agent implementations, memory backends |
| **EngineAdapter (LangGraph)** | IR → LangGraph graph, interrupt handling | Temporal, YAML, agent implementations |
| **TaskBroker** | Capability routing, retry policy, assignment strategy | Which engine runs, what the workflow does |
| **AgentRegistry** | Agent endpoint discovery, capability schema storage | Task dispatch, engine state |
| **Agent** | One capability, using `AgentContext` API | State machines, other agents, engine state |
| **MemoryService** | Workflow-scoped KV + vector storage | What capabilities produce, what engines run |
| **EventBus** | CloudEvent delivery, topic pattern routing | Workflow logic, engine internals |

---

## 7. Engine-Agnosticism Invariants

Three mechanisms make engine-agnosticism real rather than aspirational:

**1. WorkflowIR is the lingua franca.** All adapters receive identical input.
No engine primitives appear above Layer 2. The IR is formally typed (protobuf),
versioned (`ir_version`), and compiled from the same source regardless of
which engine will execute it.

**2. Capability dispatch is fully mediated.** Engines never call agents directly.
All calls go through `TaskBroker.DispatchTask`. The engine only knows
`{capability_name, input_json, workflow_id, timeout}`. Retry logic, agent
selection, and load balancing are broker concerns — not engine concerns.

**3. All lifecycle events are CloudEvents on NATS.** Engine adapters translate
engine-native events into CloudEvents and publish them. External subscribers
(monitoring, audit, data pipelines) never need engine-specific clients.

---

## 8. M3 Implementation Sequence (Temporal)

Build in this order so each piece is independently testable before the next:

| Step | What | Dependency |
|------|------|-----------|
| 1 | `DispatchCapabilityActivity` — Temporal Activity calling task-broker | None |
| 2 | `IRInterpreterWorkflow` — generic state machine interpreter | Step 1 |
| 3 | Decide `_event` convention vs `event_expression` proto field | Step 2 design |
| 4 | `TemporalEngine` — implements `WorkflowEngine` interface | Step 2 |
| 5 | Engine adapter gRPC server — wires `TemporalEngine` behind contract | Step 4 |
| 6 | BDD `.feature` file for `EngineAdapterService` — commit before step 5 | None (write first) |
| 7 | End-to-end: YAML → compiler → IR → adapter → Temporal → agent → terminal | Steps 1-5 |

**Success criterion for M3:** a workflow compiled from YAML reaches its terminal
state via Temporal and produces the same output as an equivalent in-memory runner.
This proves the three-layer model end-to-end.

**Critical risk:** CEL guard evaluation must happen inside the adapter (in Go),
not inside Temporal's workflow code. Temporal's determinism constraint prohibits
non-deterministic operations; CEL evaluation is deterministic and safe, but
external lookups (e.g., context fetches from MemoryService during guard eval)
are not. Resolve before implementation begins.

---

*See also:* `competitive-analysis-2026.md` for positioning context ·
`docs/patterns/go-service-patterns.md` for service templates ·
`docs/engineering/renovate-fix-sop.md` for dependency maintenance
