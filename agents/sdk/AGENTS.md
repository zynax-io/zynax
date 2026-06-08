<!-- SPDX-License-Identifier: Apache-2.0 -->

# agents/sdk — AGENTS.md

> The Zynax Python SDK (`zynax-sdk`). Minimal `Agent` base class for capability providers.
> Inherits all rules from root `AGENTS.md` and `agents/AGENTS.md`.

---

## Purpose

The SDK provides the **`Agent` base class** — an abstract gRPC servicer that handles
capability routing and `TaskEvent` streaming, so adapter authors focus on business
logic rather than gRPC plumbing.

What the SDK handles:
- Routing incoming `ExecuteCapability` requests to the matching `@capability` handler.
- Streaming `TaskEvent` responses (`PROGRESS`, `COMPLETED`, `FAILED`) back to the caller.
- Input validation (`capability_name`, `task_id`, `input_payload` JSON check).
- `GetCapabilitySchema` stub — returns metadata for a registered capability.

What the SDK does **not** handle (M6+):
- Agent registration and heartbeat with `agent-registry`.
- Prometheus metrics, OTel tracing, structured logging bootstrap.
- Graceful shutdown on `SIGTERM`.
- Health probes, Dockerfile, or docker-compose wiring.

---

## Module Structure

```
agents/sdk/src/zynax_sdk/
├── agent.py       ← Agent base class, @capability decorator, report_* helpers
└── __init__.py    ← Exports: Agent, capability, __version__
```

---

## Quickstart

```python
from zynax_sdk import Agent, capability

class Summarizer(Agent):
    @capability("summarize")
    async def summarize(self, request, context):
        # request.task_id, request.capability_name, request.input_payload
        yield self.report_progress(request.task_id, {"step": 1, "status": "processing"})
        yield self.report_completed(request.task_id, {"summary": "done"})
```

Then wire `Summarizer` into a `grpc.server`:

```python
import grpc
from concurrent import futures
from zynax.v1 import agent_pb2_grpc

server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
agent_pb2_grpc.add_AgentServiceServicer_to_server(Summarizer(), server)
server.add_insecure_port("[::]:50051")
server.start()
```

---

## API Reference

### `@capability(name: str)`

Decorator. Registers an `async def` method as a named capability handler.
The decorated method must be an async generator yielding `TaskEvent` objects:

```python
@capability("my_cap")
async def my_cap(self, request, context):
    yield self.report_progress(request.task_id, {...})
    yield self.report_completed(request.task_id, {...})
```

### `Agent.report_progress(task_id, payload) -> TaskEvent`

Creates a `TASK_EVENT_TYPE_PROGRESS` event. `payload` is a `dict[str, Any]` serialised to JSON bytes.

### `Agent.report_completed(task_id, payload) -> TaskEvent`

Creates a `TASK_EVENT_TYPE_COMPLETED` terminal event.

### `Agent.report_failed(task_id, code, message) -> TaskEvent`

Creates a `TASK_EVENT_TYPE_FAILED` terminal event with a structured `CapabilityError`.

### `Agent.ExecuteCapability(request, context) -> Generator[TaskEvent]`

gRPC handler (called by the gRPC framework). Routes the request to the registered handler.
Aborts with `INVALID_ARGUMENT` if `capability_name` or `task_id` is empty, or if
`input_payload` is not valid JSON. Yields `report_failed` if no handler is registered.

### `Agent.GetCapabilitySchema(request, context) -> GetCapabilitySchemaResponse`

gRPC handler. Returns basic schema metadata for a registered capability. Aborts with
`NOT_FOUND` if the capability is not registered.

---

## SDK vs Raw Stubs

Use the SDK when building a new Python agent (server role — receives and executes tasks).
Use raw stubs (`protos/generated/python/`) when calling Zynax services as a client
or when you want full control over the gRPC lifecycle.

---

## PyPI Trusted Publisher Setup

This section documents the **one-time manual steps** required to enable the
OIDC Trusted Publisher relationship between GitHub Actions and PyPI / TestPyPI.
No API keys or tokens are stored in GitHub Secrets — authentication uses OIDC.

### TestPyPI (dry-run, for PRs)

1. Log in to [test.pypi.org](https://test.pypi.org) as the `zynax-io` organisation owner.
2. Navigate to **Your projects → zynax-sdk → Publishing → Add a new publisher**.
3. Fill in the Trusted Publisher form:
   - **Owner:** `zynax-io`
   - **Repository:** `zynax`
   - **Workflow filename:** `tools-publish.yml`
   - **Environment name:** `testpypi`
4. Click **Add**.
5. In GitHub, create a **repository environment** named `testpypi` at
   `https://github.com/zynax-io/zynax/settings/environments`.
   No extra secrets are needed — OIDC mints a short-lived token automatically.

### PyPI (real publish, for version tags)

> Handled by `sdk-publish.yml` (issue #806 — F.2, not yet created).

1. Log in to [pypi.org](https://pypi.org) as the `zynax-io` organisation owner.
2. Navigate to **Your projects → zynax-sdk → Publishing → Add a new publisher**.
3. Fill in the Trusted Publisher form:
   - **Owner:** `zynax-io`
   - **Repository:** `zynax`
   - **Workflow filename:** `sdk-publish.yml`
   - **Environment name:** `pypi`
4. Click **Add**.
5. Create a **repository environment** named `pypi` in GitHub (add branch
   protection rule: only allow `v*` tags to deploy to this environment).

### Verification

After the Trusted Publisher is registered, open a PR that touches `agents/sdk/`.
The `SDK TestPyPI Dry-Run` workflow should pass its publish step. Check
`test.pypi.org/project/zynax-sdk/` to confirm the package was uploaded.

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
