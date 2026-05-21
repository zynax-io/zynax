<!-- SPDX-License-Identifier: Apache-2.0 -->

# Python Best Practices — Zynax Adapters and SDK

> Scope: `agents/sdk/`, `agents/adapters/`  
> Enforcement: `ruff` (lint+format), `mypy --strict`, `bandit` (SAST), `pip-audit`
> Run via `make lint security` (all inside Docker)

---

## Project Setup

All Python projects use `uv` (ADR-003). Each adapter has its own `pyproject.toml` and
`uv.lock`.

```toml
# pyproject.toml
[project]
name = "zynax-llm-adapter"
requires-python = ">=3.12"
dependencies = [
    "grpcio>=1.63",
    "grpcio-tools>=1.63",
    "pydantic-settings>=2.0",
    "openai>=1.0",          # example: llm-adapter
]

[tool.ruff.lint]
select = ["E", "F", "I", "S", "B", "ANN"]

[tool.mypy]
strict = true
```

---

## Type Hints (mypy --strict)

```python
# ✅ Full type hints on all public functions
from typing import AsyncIterator
import grpc

async def execute(
    self,
    request: TaskRequest,
    context: grpc.aio.ServicerContext,
) -> AsyncIterator[TaskEvent]:
    ...

# ❌ Bare Any or missing annotations will fail mypy --strict
```

---

## Agent Base Class (zynax-sdk)

Use the `Agent` base class from `agents/sdk/` for all Python adapters:

```python
from zynax_sdk import Agent, capability

class LLMAdapter(Agent):
    @capability("chat_completion")
    async def chat(self, request: CapabilityRequest) -> AsyncIterator[TaskEvent]:
        # Implementation
        yield TaskEvent(type=TaskEventType.PROGRESS, ...)
        yield TaskEvent(type=TaskEventType.COMPLETED, result=result)
```

The `@capability` decorator registers the handler with the gRPC `AgentService`.
Use `async` generators for streaming responses — `PROGRESS` events during generation,
`COMPLETED` on finish.

---

## Configuration (Pydantic Settings — ADR-007)

```python
from pydantic_settings import BaseSettings

class LLMConfig(BaseSettings):
    api_key: str                    # required — no default, fails fast if missing
    model: str = "gpt-4o"           # overridable via LLM_MODEL env var
    registry_addr: str = "agent-registry:50057"

    model_config = {"env_prefix": "LLM_"}

# Usage: config = LLMConfig()  # reads from environment
```

**Rules:**
- Never hardcode API keys, model names, or service addresses
- All config from environment variables (12-Factor)
- Use `env_prefix` to namespace vars per adapter
- Fail fast at startup if required vars are missing

---

## gRPC Patterns

```python
# ✅ Always use async gRPC (grpc.aio)
import grpc.aio

async def run() -> None:
    server = grpc.aio.server()
    add_AgentServiceServicer_to_server(adapter, server)
    server.add_insecure_port(f"[::]:{config.grpc_port}")
    await server.start()
    await server.wait_for_termination()

# ✅ Use context managers for gRPC channels
async with grpc.aio.insecure_channel(registry_addr) as channel:
    stub = AgentRegistryStub(channel)
    await stub.RegisterAgent(request)
```

Never call platform services via HTTP — always use gRPC stubs from
`protos/generated/python/`. See ADR-001.

---

## Security (bandit)

```python
# ✅ Never log secrets
logger.info("registered adapter", extra={"capability": cap_name})
# ❌ Never: logger.info(f"API key: {config.api_key}")

# ✅ Use subprocess with explicit args list (not shell=True)
result = subprocess.run(["git", "clone", url], capture_output=True)

# ✅ Validate URLs before making requests (SSRF prevention)
from urllib.parse import urlparse
parsed = urlparse(url)
if parsed.scheme not in ("https",):
    raise ValueError(f"unsafe URL scheme: {parsed.scheme}")
```

Run `bandit -r agents/` (via `make security`) to catch SAST issues.

---

## I/O Resource Management

```python
# ✅ Use context managers or try/finally
async with aiofiles.open(path) as f:
    content = await f.read()

# ✅ Clean up gRPC channels
try:
    async with grpc.aio.insecure_channel(addr) as channel:
        ...
finally:
    # channel closed automatically
    pass
```

---

## Testing

```python
# ✅ Use pytest + pytest-asyncio for async tests
import pytest

@pytest.mark.asyncio
async def test_chat_capability() -> None:
    adapter = LLMAdapter(config=MockConfig())
    events = [e async for e in adapter.chat(mock_request)]
    assert events[-1].type == TaskEventType.COMPLETED

# ✅ Mock external API calls, not gRPC stubs
@pytest.fixture
def mock_openai(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr("openai.AsyncOpenAI", MockOpenAI)
```

Coverage target: ≥ 85% for adapters (same spirit as Go's 90% domain target).

---

## Key tool versions (pinned in ci-runner image)

| Tool | Purpose | Pin |
|---|---|---|
| `ruff` | Lint + format | via `pyproject.toml` |
| `mypy` | Static type checking | via `pyproject.toml` |
| `bandit` | SAST security scanner | via CI |
| `pip-audit` | Dependency CVE check | via CI |
| `pytest` + `pytest-asyncio` | Test runner | via `pyproject.toml` |
| `uv` | Package manager (ADR-003) | Pinned in `Dockerfile.ci-runner` |
