# agents/sdk — AGENTS.md

> The Zynax Python SDK (`zynax-sdk`). Framework-agnostic core.
> Inherits all rules from root `AGENTS.md` and `agents/AGENTS.md`.
> Full implementation patterns: `docs/patterns/python-agent-guide.md`.

---

## Purpose

The SDK is the **platform adapter layer** for Python agents. It handles:
- Agent registration and heartbeat with `agent-registry`.
- Task reception and routing from `task-broker`.
- Streaming `TaskEvent` updates back to the broker.
- `AgentContext` construction and injection.
- Structured logging, OTel tracing, Prometheus metrics bootstrap.
- Graceful shutdown on `SIGTERM`.

**The SDK never implements task execution logic** — that is the `AgentRuntime`'s job.

---

## Module Structure

```
agents/sdk/src/zynax_sdk/
├── runtime.py         ← AgentRuntime Protocol, Task, TaskEvent (do not modify)
├── context.py         ← AgentContext (injected — never constructed by agent)
├── capability.py      ← @capability decorator
├── contract.py        ← gRPC service implementation (do not edit)
├── server.py          ← AgentServer: wires contract → sdk → runtime
├── platform.py        ← MemoryClient, RegistryClient, BrokerClient
├── observability.py   ← structlog + OTel + Prometheus bootstrap
└── runtimes/
    ├── direct.py      ← DirectRuntime: plain async generator
    ├── langgraph.py   ← LangGraphRuntime (extras: zynax-sdk[langgraph])
    ├── autogen.py     ← AutoGenRuntime  (extras: zynax-sdk[autogen])
    └── crewai.py      ← CrewAIRuntime   (extras: zynax-sdk[crewai])
```

---

## SDK vs Raw Stubs

Use the SDK when building a new Python agent (server role — receives and executes tasks).
Use raw stubs (`protos/generated/python/`) when calling Zynax services as a client
or when you want full control over the gRPC lifecycle.

---

## Testing

```bash
# Unit tests
cd agents/sdk
uv run pytest tests/ --cov=src --cov-fail-under=90 -v

# Via Makefile (inside Docker)
make test-unit-agents
```

Coverage requirement: ≥ 90% on `src/zynax_sdk/`.
