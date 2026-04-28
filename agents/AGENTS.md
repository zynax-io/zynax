# agents/ — Engineering Contract

> Agents are contracts + capabilities. Everything else is an adapter.
> Inherits all rules from the root `AGENTS.md`.
> Full code examples: `docs/patterns/python-agent-guide.md`.
>
> ADRs: ADR-010 (pluggable runtime), ADR-013 (adapter-first, no SDK required).

---

## Which Integration Path to Choose

| Situation | Path |
|-----------|------|
| Wrapping an existing system (LangGraph app, REST API, CI, LLM provider) | **Adapter** — `agents/adapters/AGENTS.md` |
| Building a new Python agent with LangGraph / AutoGen / CrewAI | **Python SDK** — `agents/sdk/AGENTS.md` |
| Plain Python agent, no AI framework | **Python SDK** (`DirectRuntime`) |
| Building an agent in Go, TypeScript, Java, Rust | **Raw stubs** — `protos/AGENTS.md §8` |
| Calling Zynax from an existing service (any language) | **Raw stubs** — client role only |

All three paths are identical from the task-broker's perspective. The integration
contract is always `protos/zynax/v1/`. See `docs/patterns/proto-interop.md`.

---

## Mental Model

```
platform (Go) ──▶ AgentService (gRPC)   ← Fixed. Versioned.
                        │
                  Zynax SDK           ← Registration, heartbeat, routing,
                  (Python pkg)               observability, shutdown.
                        │ injects AgentContext
                  AgentRuntime            ← YOU implement this one method.
                  (Protocol)
                        │
           ┌────────────┼────────────────┐
      LangGraph    AutoGen/CrewAI    DirectRuntime    Custom
```

The SDK handles the platform. You handle the intelligence.

---

## Core Contract Types (defined in sdk — never redefine)

```python
class AgentRuntime(Protocol):
    async def setup(self) -> None: ...
    async def execute(self, task: Task, context: AgentContext) -> AsyncIterator[TaskEvent]: ...
    async def teardown(self) -> None: ...

class Task:
    id: str
    capability: str
    payload: dict[str, str]
    metadata: dict[str, str]

class TaskEvent:
    type: TaskEventType   # PROGRESS | RESULT | ERROR
    payload: dict[str, str]
    message: str
```

`AgentContext` is injected by the SDK. **Never construct it in production code.**
In tests: use `FakeAgentContext` from `zynax_sdk.testing`.

---

## Repository Layout

```
agents/
├── sdk/                     ← zynax-sdk (optional — no SDK required)
│   └── src/zynax_sdk/
│       ├── runtime.py       ← AgentRuntime Protocol, Task, TaskEvent
│       ├── context.py       ← AgentContext (injected)
│       ├── capability.py    ← @capability decorator
│       ├── server.py        ← AgentServer: wires contract → sdk → runtime
│       └── runtimes/        ← LangGraphRuntime, AutoGenRuntime, etc. (extras)
├── adapters/                ← Adapter implementations (no SDK required)
│   ├── http/
│   ├── llm/
│   ├── git/
│   └── langgraph/
└── examples/
    ├── calculator/          ← DirectRuntime example
    ├── summarizer/          ← LangGraphRuntime example
    └── researcher/          ← AutoGenRuntime example
```

---

## Rules

| Rule | Reason |
|------|--------|
| Never instantiate platform clients in a Runtime | Testability: context injects them |
| Always use `FakeAgentContext` in tests | No platform running in unit tests |
| `AgentRuntime` is a Protocol — never subclass it | Structural subtyping, no coupling |
| `main.py` is wiring only | Clean architecture |
| `.feature` file before implementation | BDD-first (ADR-016) |
| LLM model always from `context.config` | 12-Factor, easy model upgrades |
| Never log `SecretStr` fields | Security |
| Nodes (LangGraph) are pure: no mixed I/O + logic | Single responsibility |
