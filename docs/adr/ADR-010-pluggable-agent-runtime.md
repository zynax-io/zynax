# ADR-010: Pluggable Agent Runtime Model

**Status:** Accepted  
**Date:** 2025-04-01

## Core Principle

> **Agents are contracts + capabilities. Everything else is a plugin.**

An agent is defined by:
- **Contract**: versioned gRPC interface every agent implements (ExecuteTask, GetCapabilities, HealthCheck).
- **Capabilities**: declared list of task types it handles (`["summarize", "search"]`).

HOW the agent executes — LangGraph, AutoGen, CrewAI, raw Python, external API — is
irrelevant to the platform. It is encapsulated behind the `AgentRuntime` Protocol.

## Three-Layer Model

```
Layer 1 — AgentContract (gRPC)         Fixed. Versioned. Platform-facing.
Layer 2 — Zynax SDK (Python pkg)   Registration, heartbeat, task routing, observability.
Layer 3 — AgentRuntime (Plugin)        execute(task, context) → events. Swappable.
```

## Decision

The `AgentRuntime` Protocol has one method:

```python
class AgentRuntime(Protocol):
    async def setup(self) -> None: ...
    async def execute(self, task: Task, context: AgentContext) -> AsyncIterator[TaskEvent]: ...
    async def teardown(self) -> None: ...
```

Runtime adapters ship as optional extras:
  zynax-sdk                  # core only, zero AI framework deps
  zynax-sdk[langgraph]       # + LangGraphRuntime
  zynax-sdk[autogen]         # + AutoGenRuntime
  zynax-sdk[crewai]          # + CrewAIRuntime

## Consequences

+ Switching AI frameworks = swap one class, nothing else changes.
+ Agents are testable: inject fake AgentContext, call execute().
+ Platform permanently decoupled from AI framework evolution.
+ A Python function is a first-class agent runtime.
- Contributors must understand the runtime vs SDK distinction.

## Rules

- Never call platform services directly in a Runtime impl — use injected AgentContext.
- Never hardcode an AI framework in the SDK core.
- AgentRuntime is a Protocol (structural subtyping) — never a base class.
