# Python Agent Guide — Reference

> Implementation patterns for Python agents and adapters.
> Rules and constraints live in `agents/AGENTS.md` and the root `AGENTS.md`.
> This file is reference material — consult it when building an agent.

---

## Option A: Direct Runtime (plain Python, no AI framework)

```python
from zynax_sdk.runtime import AgentRuntime, Task, TaskEvent, AgentContext
from zynax_sdk.capability import capability
from typing import AsyncIterator

@capability(
    name="calculate",
    description="Evaluate a math expression.",
    input_schema={"type":"object","properties":{"expression":{"type":"string"}},"required":["expression"]},
    output_schema={"type":"object","properties":{"result":{"type":"number"}}},
)
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

---

## Option B: LangGraph Runtime

```python
from zynax_sdk.capability import capability
from zynax_sdk.runtimes.langgraph import LangGraphRuntime
from zynax_sdk.context import AgentContext
from typing import TypedDict
from langgraph.graph import StateGraph, END

class SummarizerState(TypedDict):
    documents: list[str]
    context:   list[str]
    summary:   str
    error:     str | None

async def fetch_context_node(state: SummarizerState, context: AgentContext) -> dict:
    results = await context.memory.search_similar(
        namespace_id=f"agent:{context.agent_id}",
        query=state["documents"][0][:200],
        top_k=3,
    )
    return {"context": [r.content for r in results]}

async def summarize_node(state: SummarizerState, context: AgentContext) -> dict:
    from langchain_openai import ChatOpenAI
    llm = ChatOpenAI(model=context.config["llm_model"])
    response = await llm.ainvoke(
        f"Context: {state['context']}\n\nSummarise: {state['documents']}"
    )
    return {"summary": response.content}

@capability(
    name="summarize",
    description="Summarise one or more documents.",
    input_schema={"type":"object","properties":{"documents":{"type":"array","items":{"type":"string"}}},"required":["documents"]},
    output_schema={"type":"object","properties":{"summary":{"type":"string"}}},
)
class SummarizerRuntime(LangGraphRuntime):
    def build_graph(self) -> StateGraph:
        g = StateGraph(SummarizerState)
        g.add_node("fetch_context", fetch_context_node)
        g.add_node("summarize", summarize_node)
        g.set_entry_point("fetch_context")
        g.add_edge("fetch_context", "summarize")
        g.add_edge("summarize", END)
        return g.compile()
```

---

## Option C: AutoGen / CrewAI

```python
from zynax_sdk.runtimes.autogen import AutoGenRuntime  # or CrewAIRuntime
from zynax_sdk.capability import capability

@capability(
    name="research",
    description="Research a topic and produce a report.",
    input_schema={"type":"object","properties":{"topic":{"type":"string"}},"required":["topic"]},
    output_schema={"type":"object","properties":{"report":{"type":"string"}}},
)
class ResearcherRuntime(AutoGenRuntime):
    def build_agents(self, context: AgentContext):
        ...  # Define your AutoGen agents — the runtime handles execute()
```

---

## Option D: Custom (any framework)

```python
class MyRuntime:
    """Protocol is structural — no inheritance needed."""
    async def setup(self) -> None: ...
    async def execute(self, task, context) -> AsyncIterator[TaskEvent]: ...
    async def teardown(self) -> None: ...

assert isinstance(MyRuntime(), AgentRuntime)  # verified at test time
```

---

## Wiring (always identical regardless of runtime)

```python
# src/<agent>/main.py — wiring only
import asyncio
from zynax_sdk.server import AgentServer
from zynax_sdk.observability import configure
from <agent>.config import settings
from <agent>.runtime import MyRuntime

async def main() -> None:
    configure(settings)
    server = AgentServer(runtime=MyRuntime(), settings=settings)
    await server.run()

asyncio.run(main())
```

---

## Configuration

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

## Testing

```python
# tests/unit/test_runtime.py
import pytest
from zynax_sdk.runtime import Task, TaskEventType
from zynax_sdk.testing import FakeAgentContext
from <agent>.runtime import MyRuntime

@pytest.fixture
def ctx() -> FakeAgentContext:
    return FakeAgentContext(
        agent_id="test-01",
        task_id="task-01",
        config={"llm_model": "gpt-4o-mini"},
    )

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
    ctx.memory.set.assert_called_once()
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

## Language Interoperability

SDK agents and non-Python callers are transparent to each other.
When the task-broker dispatches a `summarize` task it does not know or care
whether the executor is a Python SDK agent, a Go raw-stub agent, or a
TypeScript adapter. The `ExecuteCapabilityRequest` and the `TaskEvent` stream
are identical in every case.

Calling platform services from agent code always uses gRPC:
- Python SDK: `AgentContext.memory`, `AgentContext.broker`, `AgentContext.registry`
- Other languages: generate stubs with `buf generate`, open a gRPC channel, call the RPC

See `docs/patterns/proto-interop.md` for multi-language consuming patterns.
