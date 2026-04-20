# agents/ — AGENTS.md

> **Agents are contracts + capabilities. Everything else is a plugin.**
> See `docs/adr/ADR-010-pluggable-agent-runtime.md`.
> Read this file entirely before writing any agent code.

---

## Which Integration Path to Choose

Before writing any code, choose the right path. The decision depends on what you
already have, what language you are working in, and how deeply you want to
integrate with Zynax.

### The Three Paths

**Path 1 — Adapter (preferred for existing systems)**

You have an existing system — a LangGraph app, a REST API, a CI tool, an LLM
provider — and you want it to be reachable as a Zynax capability. You do not want
to change that system's code. Deploy an adapter alongside it. The adapter speaks
the `AgentService` gRPC contract to Zynax and translates to the existing system's
native protocol on the other side. See `agents/adapters/AGENTS.md`.

Choose this path when:
- The system already exists and works
- You want zero Zynax dependencies in the existing codebase
- The integration is translation work, not new agent logic
- You work in any language — adapters have no language restriction

**Path 2 — Python SDK (for new Zynax-native Python agents)**

You are building something new in Python specifically to run inside Zynax. You want
to write agent logic and have the platform handle registration, task routing,
heartbeat, context injection, and shutdown. Install `zynax-sdk`, implement the
`AgentRuntime` Protocol, and the SDK wires everything else. See `agents/sdk/AGENTS.md`.

Choose this path when:
- You are building new agent logic, not wrapping an existing system
- You are working in Python and want to use AI frameworks (LangGraph, AutoGen, CrewAI)
- You want the platform plumbing to disappear and focus on the intelligence layer

**Path 3 — Raw proto stubs (for non-Python languages and client-side callers)**

You work in Go, TypeScript, Java, Rust, or any other language with gRPC support.
Generate stubs from `protos/zynax/v1/`, implement the `AgentService` contract directly,
and connect. There is no SDK requirement and no Zynax-specific library to adopt —
the generated stubs and a gRPC channel are sufficient. See `protos/AGENTS.md §8`.

Choose this path when:
- You are not working in Python
- You are calling Zynax services from an existing codebase (client role)
- You want the minimum possible coupling to Zynax internals

### Decision Table

| Situation | Recommended path |
|-----------|-----------------|
| Wrapping an existing LangGraph app | Adapter (`langgraph-adapter`) |
| Wrapping any HTTP API | Adapter (`http-adapter`) |
| Building a new Python agent with LangGraph | Python SDK (`LangGraphRuntime`) |
| Building a new Python agent, any framework | Python SDK (`DirectRuntime` or custom) |
| Building an agent in Go | Raw stubs — implement `AgentService` directly |
| Building an agent in TypeScript / Java / Rust | Raw stubs — generate and implement |
| Calling Zynax from an existing service (any language) | Raw stubs — client role only |
| Connecting a CI system (Jenkins, GitHub Actions) | Adapter (`ci-adapter` pattern) |
| Connecting an LLM provider | Adapter (`llm-adapter`) |

### What All Three Paths Have in Common

Regardless of path, the integration contract is always the same proto definition
in `protos/zynax/v1/`. The adapter, the SDK agent, and the raw-stub agent are all
identical from the task-broker's perspective. They register the same capability
names, they receive the same `ExecuteCapabilityRequest`, and they return the same
stream of `TaskEvent` responses. The path is an implementation detail that Zynax
is entirely unaware of.

---

## Mental Model

```
platform (Go) ──▶ AgentContract (gRPC)   ← Fixed. Versioned.
                        │
                  Zynax SDK           ← Registration, heartbeat, routing,
                  (Python pkg)               observability, shutdown. YOU DO NOT WRITE THIS.
                        │ injects AgentContext
                  AgentRuntime            ← YOU implement this ONE method.
                  (Protocol)                 execute(task, context) → events
                        │
           ┌────────────┼────────────────┐
      LangGraph    AutoGen/CrewAI    Plain Python    Your Custom Framework
```

The SDK handles the platform. You handle the intelligence.
The runtime is a Plugin — swap it without touching anything else.

---

## Repository Layout

```
agents/
├── AGENTS.md                         ← This file
├── sdk/                              ← Zynax Python SDK (no AI framework deps in core)
│   ├── AGENTS.md
│   ├── pyproject.toml                ← Published as: zynax-sdk
│   └── src/zynax_sdk/
│       ├── runtime.py                ← AgentRuntime Protocol, Task, TaskEvent
│       ├── context.py                ← AgentContext (injected — never constructed by agent)
│       ├── capability.py             ← @capability decorator
│       ├── contract.py               ← gRPC service implementation (do not edit)
│       ├── server.py                 ← AgentServer: wires contract → sdk → runtime
│       ├── platform.py               ← MemoryClient, RegistryClient, BrokerClient
│       ├── observability.py          ← structlog + OTel + Prometheus bootstrap
│       └── runtimes/                 ← Optional adapters (installed as extras)
│           ├── direct.py             ← DirectRuntime: plain async generator
│           ├── langgraph.py          ← LangGraphRuntime (zynax-sdk[langgraph])
│           ├── autogen.py            ← AutoGenRuntime  (zynax-sdk[autogen])
│           └── crewai.py             ← CrewAIRuntime   (zynax-sdk[crewai])
└── examples/
    ├── calculator/                   ← DirectRuntime — no AI framework
    ├── summarizer/                   ← LangGraphRuntime
    ├── researcher/                   ← AutoGenRuntime
    └── custom-runtime/               ← How to implement AgentRuntime from scratch
```

---

## The Three Core Types (defined in sdk — never redefine elsewhere)

### AgentRuntime — the ONLY interface you implement

```python
# sdk/src/zynax_sdk/runtime.py

from typing import AsyncIterator, Protocol, runtime_checkable
from dataclasses import dataclass
from enum import Enum

class TaskEventType(str, Enum):
    PROGRESS = "progress"   # Intermediate update — streamed to caller
    RESULT   = "result"     # Final result — terminal, exactly once
    ERROR    = "error"      # Unrecoverable failure — SDK handles retry

@dataclass(frozen=True)
class TaskEvent:
    type:    TaskEventType
    payload: dict[str, str]
    message: str = ""

    @classmethod
    def progress(cls, message: str, **payload: str) -> "TaskEvent":
        return cls(type=TaskEventType.PROGRESS, payload=payload, message=message)

    @classmethod
    def result(cls, **payload: str) -> "TaskEvent":
        return cls(type=TaskEventType.RESULT, payload=payload)

    @classmethod
    def error(cls, reason: str) -> "TaskEvent":
        return cls(type=TaskEventType.ERROR, payload={"reason": reason})

@dataclass(frozen=True)
class Task:
    id:         str
    capability: str
    payload:    dict[str, str]
    metadata:   dict[str, str]

@runtime_checkable
class AgentRuntime(Protocol):
    """Implement this. The SDK handles everything else."""

    async def setup(self) -> None:
        """Called once on startup. Load models, warm caches."""
        ...

    async def execute(
        self,
        task: Task,
        context: "AgentContext",
    ) -> AsyncIterator[TaskEvent]:
        """Execute a task. Async generator.

        - Yield TaskEvent.progress() for intermediate updates.
        - Yield TaskEvent.result() as the final event (exactly once).
        - Raise on unrecoverable failure (SDK handles retries).
        - Use context.memory / context.broker for ALL platform I/O.
        """
        ...

    async def teardown(self) -> None:
        """Called on graceful shutdown."""
        ...
```

### AgentContext — injected by the SDK, never constructed by you

```python
# sdk/src/zynax_sdk/context.py

@dataclass(frozen=True)
class AgentContext:
    """Everything an agent needs. Injected by the SDK.

    Never construct this in production code.
    In tests: use FakeAgentContext() from zynax_sdk.testing.
    """
    agent_id:  str
    task_id:   str
    memory:    MemoryClient    # memory-service gRPC client
    registry:  RegistryClient  # agent-registry gRPC client
    broker:    BrokerClient    # task-broker gRPC client
    config:    Mapping[str, Any]
    logger:    BoundLogger     # pre-bound with agent_id + trace_id
    tracer:    Tracer
```

### @capability — declare what your agent can do

```python
from zynax_sdk.capability import capability

@capability(
    name="summarize",                           # Must match capability string in task-broker
    description="Summarise documents into a concise paragraph.",
    input_schema={
        "type": "object",
        "properties": {"documents": {"type": "array", "items": {"type": "string"}}},
        "required": ["documents"],
    },
    output_schema={
        "type": "object",
        "properties": {"summary": {"type": "string"}},
    },
)
class SummarizerRuntime: ...
```

---

## How to Build an Agent — Four Options

### Option A: Direct (plain Python, no AI framework)

```python
from zynax_sdk.runtime import AgentRuntime, Task, TaskEvent, AgentContext
from zynax_sdk.capability import capability
from typing import AsyncIterator

@capability(name="calculate", description="Evaluate a math expression.",
            input_schema={"type":"object","properties":{"expression":{"type":"string"}},"required":["expression"]},
            output_schema={"type":"object","properties":{"result":{"type":"number"}}})
class CalculatorRuntime:

    async def setup(self) -> None:
        pass

    async def execute(self, task: Task, context: AgentContext) -> AsyncIterator[TaskEvent]:
        try:
            result = eval(task.payload["expression"], {"__builtins__": {}})  # noqa: S307
        except Exception as exc:
            yield TaskEvent.error(reason=str(exc))
            return
        yield TaskEvent.result(result=str(result))

    async def teardown(self) -> None:
        pass
```

### Option B: LangGraph

```python
from zynax_sdk.capability import capability
from zynax_sdk.runtimes.langgraph import LangGraphRuntime
from zynax_sdk.context import AgentContext
from typing import TypedDict
from langgraph.graph import StateGraph, END

# ── State ──────────────────────────────────────────────────────────

class SummarizerState(TypedDict):
    documents: list[str]
    context:   list[str]
    summary:   str
    error:     str | None

# ── Nodes — pure functions, one responsibility each ───────────────

async def fetch_context_node(state: SummarizerState, context: AgentContext) -> dict:
    """I/O lives here — never mix with logic nodes."""
    results = await context.memory.search_similar(
        namespace_id=f"agent:{context.agent_id}",
        query=state["documents"][0][:200],
        top_k=3,
    )
    return {"context": [r.content for r in results]}

async def summarize_node(state: SummarizerState, context: AgentContext) -> dict:
    """Pure transform: documents + context → summary. Only LLM I/O here."""
    from langchain_openai import ChatOpenAI
    llm = ChatOpenAI(model=context.config["llm_model"])
    response = await llm.ainvoke(
        f"Context: {state['context']}\n\nSummarise: {state['documents']}"
    )
    return {"summary": response.content}

def route(state: SummarizerState) -> str:
    return "error" if state.get("error") else END

# ── Runtime ────────────────────────────────────────────────────────

@capability(name="summarize",
            description="Summarise one or more documents.",
            input_schema={"type":"object","properties":{"documents":{"type":"array","items":{"type":"string"}}},"required":["documents"]},
            output_schema={"type":"object","properties":{"summary":{"type":"string"}}})
class SummarizerRuntime(LangGraphRuntime):
    """LangGraphRuntime calls build_graph() in setup() and streams node outputs
    as TaskEvent.progress(). You only define the graph. Nothing else changes."""

    def build_graph(self) -> StateGraph:
        g = StateGraph(SummarizerState)
        g.add_node("fetch_context", fetch_context_node)
        g.add_node("summarize", summarize_node)
        g.set_entry_point("fetch_context")
        g.add_edge("fetch_context", "summarize")
        g.add_edge("summarize", END)
        return g.compile()
```

### Option C: AutoGen / CrewAI

```python
from zynax_sdk.runtimes.autogen import AutoGenRuntime  # or CrewAIRuntime
from zynax_sdk.capability import capability

@capability(name="research", description="Research a topic and produce a report.",
            input_schema={"type":"object","properties":{"topic":{"type":"string"}},"required":["topic"]},
            output_schema={"type":"object","properties":{"report":{"type":"string"}}})
class ResearcherRuntime(AutoGenRuntime):
    def build_agents(self, context: AgentContext):
        ...  # Define your AutoGen agents — the runtime handles execute()
```

### Option D: Custom (any framework, any pattern)

```python
class MyRuntime:
    """Protocol is structural — no inheritance needed."""
    async def setup(self) -> None: ...
    async def execute(self, task, context) -> AsyncIterator[TaskEvent]: ...
    async def teardown(self) -> None: ...

assert isinstance(MyRuntime(), AgentRuntime)  # Verified at test time
```

---

## Wiring (always identical regardless of runtime)

```python
# src/<agent>/main.py — wiring only

import asyncio
from zynax_sdk.server import AgentServer
from zynax_sdk.observability import configure
from <agent>.config import settings
from <agent>.runtime import MyRuntime   # ← swap runtime here only

async def main() -> None:
    configure(settings)
    server = AgentServer(runtime=MyRuntime(), settings=settings)
    await server.run()  # handles everything: registration, heartbeat, task routing, shutdown

asyncio.run(main())
```

---

## Configuration (standard across all agents)

```python
# src/<agent>/config.py
from pydantic import Field, SecretStr
from pydantic_settings import BaseSettings, SettingsConfigDict

class AgentSettings(BaseSettings):
    model_config = SettingsConfigDict(env_prefix="ZYNAX_AGENT_", frozen=True)

    agent_id:       str          = Field(description="Unique id in the mesh")
    display_name:   str          = Field(description="Human-readable name")
    registry_url:   str          = Field(default="agent-registry:50051")
    memory_url:     str          = Field(default="memory-service:50053")
    broker_url:     str          = Field(default="task-broker:50052")
    grpc_port:      int          = Field(default=50060, ge=1024, le=65535)
    llm_model:      str          = Field(default="gpt-4o-mini")
    openai_api_key: SecretStr | None = Field(default=None)
    log_level:      str          = Field(default="INFO")
    otel_endpoint:  str          = Field(default="http://otel-collector:4317")

settings = AgentSettings()  # fails fast on misconfiguration
```

---

## Testing (framework-independent, always the same pattern)

```python
# tests/unit/test_runtime.py
import pytest
from zynax_sdk.runtime import Task, TaskEventType
from zynax_sdk.testing import FakeAgentContext   # provided by SDK
from <agent>.runtime import MyRuntime

@pytest.fixture
def ctx() -> FakeAgentContext:
    return FakeAgentContext(agent_id="test-01", task_id="task-01",
                            config={"llm_model": "gpt-4o-mini"})

@pytest.mark.asyncio
async def test_execute_emits_result_as_final_event(ctx) -> None:
    runtime = MyRuntime()
    await runtime.setup()
    task = Task(id="t1", capability="summarize",
                payload={"documents": ["hello world"]}, metadata={})
    events = [e async for e in runtime.execute(task, ctx)]
    assert events[-1].type == TaskEventType.RESULT

@pytest.mark.asyncio
async def test_execute_stores_result_in_memory(ctx) -> None:
    runtime = MyRuntime()
    await runtime.setup()
    task = Task(id="t2", capability="summarize",
                payload={"documents": ["doc"]}, metadata={})
    async for _ in runtime.execute(task, ctx): pass
    ctx.memory.set.assert_called_once()  # FakeAgentContext uses AsyncMock
```

---

## BDD Feature File Template

```gherkin
# tests/features/<agent>.feature
Feature: <Agent Name>
  As a task orchestrator
  I want to submit <capability> tasks to this agent
  So that I receive <output> without coupling to the runtime framework

  Background:
    Given the agent is running with a fake AgentContext

  Scenario: Execute task returns result event as final event
    Given a task with capability "<cap>" and valid payload
    When the agent executes the task
    Then the last event type is RESULT
    And no ERROR event is emitted

  Scenario: Execute task emits progress events before result
    Given a task requiring multiple steps
    When the agent executes the task
    Then at least one PROGRESS event is emitted before RESULT

  Scenario: Execute stores result in agent memory
    Given a task with valid payload
    When the agent executes the task
    Then context.memory.set was called with the result

  Scenario: Execute fails gracefully on invalid payload
    Given a task with missing required payload field
    When the agent executes the task
    Then the last event type is ERROR
    And the error message is descriptive
```

---

## Language Interoperability Notes

### SDK agents and non-Python callers are invisible to each other

When the task-broker dispatches a `summarize` task, it does not know or care
whether the agent receiving it is a Python SDK agent, a Go raw-stub agent, or
a TypeScript adapter. The `ExecuteCapabilityRequest` message is identical in
every case. The stream of `TaskEvent` responses is identical in every case.

A workflow written in YAML, compiled to IR by the Go workflow-compiler, executed
by Temporal via the Go engine-adapter, and dispatching to a Python SDK agent is
a fully cross-language execution path — by design, not by accident.

### Calling platform services from agent code

Agent code sometimes needs to call back into Zynax platform services during
execution — reading from the memory service, querying the agent registry, or
submitting a subtask to the broker. The mechanism is always the same: generate
or import stubs for the target service's proto, open a gRPC channel using the
service address from environment config, and call the RPC.

The Python SDK provides pre-wired clients for memory, registry, and broker via
`AgentContext`. In other languages, you wire these clients yourself from the
generated stubs. The contracts are identical. The difference is convenience, not
capability.

### Adding a non-Python language as Tier 1

If a specific language becomes a priority for official Tier 1 support, the
process is: open an RFC proposing the language addition, commit the generated
stubs to a `gen/<language>/` directory, add stub generation to `make generate-protos`,
and add stub validation to CI. The proto source does not change — only the
generation output and the CI pipeline.

---

## Rules

| Rule | Reason |
|------|--------|
| Never instantiate platform clients in a Runtime | Testability: context injects them |
| Always use `FakeAgentContext` in tests | No platform running in unit tests |
| Nodes (LangGraph) are pure: no mixed I/O + logic | Single responsibility |
| `AgentRuntime` is a Protocol — never subclass it | Structural subtyping, no coupling |
| `main.py` is wiring only | Clean architecture |
| `.feature` file before implementation | BDD-first |
| LLM model always from `context.config` | 12-Factor, easy model upgrades |
| Never log `SecretStr` fields | Security |
