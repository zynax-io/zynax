# agents/sdk — AGENTS.md

> The Zynax Python SDK.
> Published as `zynax-sdk` on PyPI.
> Framework-agnostic core. AI runtimes are optional extras.

---

## Purpose

The SDK is the **platform adapter layer** for Python agents. It owns:
- Agent registration and heartbeat with `agent-registry`.
- Task reception and routing from `task-broker`.
- Streaming `TaskEvent` updates back to the broker.
- `AgentContext` construction and injection.
- Structured logging, OTel tracing, Prometheus metrics bootstrap.
- Graceful shutdown on `SIGTERM`.

**The SDK never implements task execution logic.**
That is the `AgentRuntime`'s job.

---

## SDK vs Raw Proto Stubs — When to Use Which

The Python SDK is not the only way to connect Python code to Zynax. Understanding
when to use it and when to go directly to the raw generated stubs prevents
over-engineering and under-engineering in equal measure.

### Use the SDK when you are in server role building something new

The SDK's value is entirely in the server-side boilerplate it eliminates for
a Python agent that receives and executes tasks. That boilerplate — registration,
heartbeat, channel management, context injection, streaming lifecycle, graceful
shutdown — is roughly 200–300 lines of gRPC plumbing that every agent needs
but none should write from scratch. The SDK writes it once. You write the logic.

If you are:
- Building a new Python agent that receives `ExecuteCapabilityRequest` RPCs
- Using an AI framework (LangGraph, AutoGen, CrewAI) for agent logic
- Wanting the platform lifecycle managed for you

Then the SDK is the right choice. The `AgentServer` class is the entry point.
`AgentRuntime` is the only interface you implement.

### Use raw stubs when you are in client role or in a non-Python language

The SDK is a server-side tool. It is not designed for, and adds no value to,
the case where your Python code is calling Zynax services rather than serving them.
If you want to submit a workflow, query the agent registry, or read from the memory
service from Python code that is not itself an agent, import the raw generated stubs
from `protos/generated/python/` and call the service directly.

The same logic applies for any non-Python language. The SDK does not have a Go,
TypeScript, Java, or Rust edition and is not planned to have one. Those languages
use their generated stubs directly. See `protos/AGENTS.md §8`.

### What the SDK adds over raw stubs — precisely

| Concern | Raw stubs | SDK |
|---------|-----------|-----|
| gRPC channel to task-broker | You manage | SDK manages |
| Agent registration on startup | You implement | `AgentServer` handles |
| Heartbeat to agent-registry | You implement | `AgentServer` handles |
| Task reception loop | You implement | `AgentServer` handles |
| `AgentContext` construction and injection | You construct | SDK injects |
| Streaming `TaskEvent` to broker | You wire | SDK wires |
| Graceful shutdown on SIGTERM | You implement | `AgentServer` handles |
| Structured logging bootstrap | You configure | `observability.py` configures |
| OTel trace propagation | You wire | SDK wires |
| Prometheus metrics bootstrap | You configure | `observability.py` configures |
| AI framework integration (LangGraph, etc.) | You write | Runtime adapters provided |

Everything in the right column is identical across all SDK agents regardless
of the AI framework they use. It is stable, tested platform code that changes
only when the proto contract changes.

### The SDK does not change the proto contract

Installing and using `zynax-sdk` does not give you access to different RPCs,
different message types, or different capabilities than using raw stubs. The SDK
is an implementation of the `AgentService` proto contract, not an extension of it.
An agent built with the SDK and an agent built on raw stubs are indistinguishable
from the task-broker's perspective. Both satisfy the same contract. Both are
interoperable with every other part of the Zynax platform.

This means you can migrate from raw stubs to the SDK or vice versa without any
change to the platform, the workflow definitions, or the capability routing
configuration. The migration is purely internal to the agent's codebase.

---

## SDK Package Layout

```
sdk/src/keel_sdk/
├── runtime.py          ← AgentRuntime Protocol, Task, TaskEvent (PUBLIC API)
├── context.py          ← AgentContext (PUBLIC API — injected, never constructed)
├── capability.py       ← @capability decorator (PUBLIC API)
├── testing.py          ← FakeAgentContext, helpers (PUBLIC — for agent tests)
├── server.py           ← AgentServer: the main SDK entry point (PUBLIC API)
├── contract.py         ← gRPC AgentService implementation (INTERNAL — do not expose)
├── platform.py         ← MemoryClient, RegistryClient, BrokerClient (INTERNAL)
├── observability.py    ← logging + tracing + metrics bootstrap (INTERNAL)
└── runtimes/           ← Optional adapters (INTERNAL — exposed via extras)
    ├── _base.py        ← Shared adapter utilities
    ├── direct.py       ← DirectRuntime
    ├── langgraph.py    ← LangGraphRuntime
    ├── autogen.py      ← AutoGenRuntime
    └── crewai.py       ← CrewAIRuntime
```

---

## SDK Public API Contract

These are the stable public symbols. Changing them is a breaking change requiring ADR + semver major bump.

```python
# All public symbols exported from keel_sdk.__init__
from keel_sdk import (
    AgentRuntime,       # Protocol
    AgentContext,       # dataclass (injected)
    Task,               # dataclass
    TaskEvent,          # dataclass
    TaskEventType,      # Enum
    capability,         # decorator
    AgentServer,        # main entry point
    AgentSettings,      # base pydantic-settings class
    FakeAgentContext,   # for testing
)
```

---

## AgentServer — How It Wires Everything

```python
# sdk/src/keel_sdk/server.py

class AgentServer:
    """The main SDK entry point.

    Handles:
    - Validating that runtime satisfies AgentRuntime Protocol.
    - Reading capabilities from runtime._capabilities (set by @capability).
    - Registering the agent with agent-registry on startup.
    - Starting the heartbeat background task.
    - Starting the gRPC server for receiving tasks from task-broker.
    - Starting health + metrics HTTP servers.
    - Injecting AgentContext before calling runtime.execute().
    - Streaming TaskEvent updates back to task-broker.
    - Graceful shutdown on SIGTERM.

    Usage:
        server = AgentServer(runtime=MyRuntime(), settings=settings)
        await server.run()  # blocks until SIGTERM
    """

    def __init__(self, runtime: AgentRuntime, settings: AgentSettings) -> None:
        if not isinstance(runtime, AgentRuntime):
            raise TypeError(
                f"{type(runtime).__name__} does not satisfy AgentRuntime Protocol. "
                "Implement setup(), execute(), and teardown()."
            )
        self._runtime = runtime
        self._settings = settings
```

---

## FakeAgentContext — for agent unit tests

```python
# sdk/src/keel_sdk/testing.py

from unittest.mock import AsyncMock
import structlog
from opentelemetry import trace
from keel_sdk.context import AgentContext

def FakeAgentContext(
    agent_id: str = "test-agent-01",
    task_id: str = "test-task-01",
    config: dict | None = None,
) -> AgentContext:
    """Create a fully fake AgentContext for unit tests.

    All platform clients are AsyncMock — no real services needed.
    Call assertions on ctx.memory.set, ctx.broker.submit_task, etc.

    Example:
        ctx = FakeAgentContext(config={"llm_model": "gpt-4o-mini"})
        async for event in runtime.execute(task, ctx): ...
        ctx.memory.set.assert_called_once()
    """
    return AgentContext(
        agent_id=agent_id,
        task_id=task_id,
        memory=AsyncMock(),
        registry=AsyncMock(),
        broker=AsyncMock(),
        config=config or {},
        logger=structlog.get_logger().bind(agent_id=agent_id),
        tracer=trace.get_tracer("test"),
    )
```

---

## Runtime Adapter Pattern

Each built-in runtime adapter follows this pattern:

```python
# sdk/src/keel_sdk/runtimes/langgraph.py

class LangGraphRuntime:
    """Base class for LangGraph-backed runtimes.

    Subclass this and implement build_graph().
    The execute() method is provided — it runs the graph and
    maps LangGraph node outputs to TaskEvent.progress() events.

    This is a convenience class, not a requirement.
    Agents can implement AgentRuntime directly if they prefer.
    """

    def build_graph(self) -> "StateGraph":
        raise NotImplementedError("Subclass must implement build_graph()")

    async def setup(self) -> None:
        self._graph = self.build_graph()

    async def execute(
        self,
        task: Task,
        context: AgentContext,
    ) -> AsyncIterator[TaskEvent]:
        state = self._build_initial_state(task, context)
        async for event in self._graph.astream(state, config={"callbacks": [...]}):
            node_name, node_output = next(iter(event.items()))
            yield TaskEvent.progress(message=f"completed: {node_name}", **node_output)

        final_state = await self._graph.aget_state(...)
        yield TaskEvent.result(**self._extract_result(final_state))

    async def teardown(self) -> None:
        pass
```

---

## pyproject.toml (SDK)

```toml
[project]
name = "zynax-sdk"
version = "0.1.0"
requires-python = ">=3.12"
description = "Zynax Python SDK — framework-agnostic agent runtime adapter"
license = { text = "Apache-2.0" }

# Core: zero AI framework deps
dependencies = [
    "grpcio>=1.64.0",
    "pydantic>=2.7.0",
    "pydantic-settings>=2.3.0",
    "structlog>=24.2.0",
    "prometheus-client>=0.20.0",
    "opentelemetry-api>=1.25.0",
    "opentelemetry-sdk>=1.25.0",
    "opentelemetry-instrumentation-grpc>=0.46b0",
]

# Optional extras — AI frameworks
[project.optional-dependencies]
langgraph = ["langgraph>=0.1.0", "langchain-openai>=0.1.0"]
autogen   = ["pyautogen>=0.2.0"]
crewai    = ["crewai>=0.28.0"]
all       = ["zynax-sdk[langgraph,autogen,crewai]"]
```
