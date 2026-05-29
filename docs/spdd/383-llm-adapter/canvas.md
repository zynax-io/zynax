<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — LLM Adapter (LLM Provider Capability Adapter)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #383
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-08
**Status:** Implemented

---

## R — Requirements

- **Problem:** There is no reusable way to call an LLM provider from a Zynax workflow. Any operator who wants to invoke `openai`, `bedrock`, or `ollama` must write a bespoke gRPC adapter from scratch — with async streaming, provider routing, credential management, and timeout handling — even though the call pattern is identical across providers: send a prompt, stream token chunks, return a full response. This barrier means LLM steps cannot be added to workflows without significant per-project engineering work.

- **Missing capability:** A config-driven Python adapter that exposes `chat_completion` as a Zynax capability, routes requests to the correct LLM provider based on YAML config, streams per-token PROGRESS events, and delivers the full response as COMPLETED — without any code changes to the calling workflow.

- **Definition of done — observable outcomes:**
  - `zynax apply agent-def.yaml` registers the llm-adapter; a workflow step calls `chat_completion`; the task broker receives a `TASK_EVENT_TYPE_COMPLETED` `TaskEvent` with the full LLM response as `payload`.
  - Per-token chunks arrive as `TASK_EVENT_TYPE_PROGRESS` events on the broker stream before the terminal event.
  - Provider routing is config-only: changing the `provider` field in the YAML switches from `openai` to `bedrock` to `ollama` with no code change.
  - A request exceeding `timeout_seconds` emits `TASK_EVENT_TYPE_FAILED` with `CapabilityError.code = "TIMEOUT"`.
  - Invalid `input_payload` (missing required fields, wrong types) emits `TASK_EVENT_TYPE_FAILED` with `code = "INVALID_INPUT"`.
  - Provider API errors emit `TASK_EVENT_TYPE_FAILED` with `code = "UPSTREAM_ERROR"` and a sanitised message — no raw API response bodies, no credential values.
  - `GetCapabilitySchema` returns the JSON Schema declared in the `AgentDef` YAML for the `chat_completion` capability.
  - `make test` green · `make lint` clean · `make security` clean.
  - BDD contract scenarios in `protos/tests/features/llm_adapter.feature` pass.

---

## E — Entities

### Existing entities consumed (no changes in #383)

- **`AgentService`** (`protos/zynax/v1/agent.proto`) — two-RPC contract implemented by the adapter: `ExecuteCapability` (server-streaming `TaskEvent`) and `GetCapabilitySchema`. Contract invariants: exactly one terminal event per stream; `task_id` echoed on every event; `timeout_seconds` honoured; no events after terminal.
- **`AgentRegistryService`** (`protos/zynax/v1/agent_registry.proto`) — `RegisterAgent` called at startup; `DeregisterAgent` called on graceful shutdown.
- **`AgentDef`** (proto message) — `agent_id`, `name`, `description`, `endpoint` (`host:port`), `capabilities[]`. Built from YAML config at startup and sent to the registry.
- **`CapabilityDef`** (proto message) — `name` (snake_case, 1–64 chars), `description`, `input_schema` (JSON Schema bytes), `output_schema` (JSON Schema bytes).
- **`ExecuteCapabilityRequest`** (proto message) — `request_id`, `capability_name`, `task_id`, `workflow_id`, `input_payload` (JSON bytes), `timeout_seconds`.
- **`TaskEvent`** (proto message) — `task_id`, `event_type` (PROGRESS / COMPLETED / FAILED), `payload`, `timestamp`, `error` (`CapabilityError`).
- **`CapabilityError`** (proto message) — `code`, `message`, `details`. Well-known codes: `"TIMEOUT"`, `"INVALID_INPUT"`, `"UPSTREAM_ERROR"`, `"RESOURCE_EXHAUSTED"`, `"INTERNAL"`.

### New entities (introduced by #383)

- **`AdapterConfig`** — top-level YAML struct parsed at startup via `load_config()`. Fields: `agent_id`, `name`, `description`, `endpoint` (bind `host:port`), `registry_endpoint` (agent-registry `host:port`), `capabilities[]` (list of capability declarations), `provider` (`ProviderConfig`). Never contains credential values — only env-var name references for API keys and model identifiers.
- **`ProviderConfig`** — per-provider config nested inside `AdapterConfig`. Fields: `name` (`openai` / `bedrock` / `ollama`), `model` (model identifier string), `ollama_base_url` (for Ollama only, from config — not from `input_payload`), `api_key_env` (name of the env-var holding the API key; value resolved at startup, stored as `SecretStr`), `max_tokens` (ceiling enforced by the adapter before calling the provider), `region` (for Bedrock only — AWS region name from config).
- **`ChatCompletionHandler`** — async coroutine implementing the `chat_completion` capability. Validates `input_payload` against the declared JSON Schema; invokes the correct provider implementation; yields `TASK_EVENT_TYPE_PROGRESS` per token chunk; yields terminal `TASK_EVENT_TYPE_COMPLETED` with the full concatenated response as `payload`. One shared instance per adapter process; stateless between invocations.
- **`ProviderRouter`** — map of `provider_name → provider_implementation` built from `AdapterConfig` at startup. Immutable after initialisation. Selects the correct async provider coroutine for each `ExecuteCapabilityRequest`.
- **`OpenAIProvider`** (`providers/openai.py`) — calls `openai.AsyncOpenAI.chat.completions.create(stream=True)`; iterates async stream; yields token chunks as bytes for the handler to emit as PROGRESS events.
- **`BedrockProvider`** (`providers/bedrock.py`) — calls Bedrock converse streaming API via `aiobotocore`; never `boto3` (blocking calls on the event loop are forbidden); yields token chunks.
- **`OllamaProvider`** (`providers/ollama.py`) — calls Ollama REST `/api/chat` via `httpx.AsyncClient` with streaming; iterates chunked response; yields token chunks.

### Entity relationships

```
Task Broker
    │ gRPC ExecuteCapabilityRequest
    ▼
AgentServer (AgentServiceServer)
    │
    ├── CapabilityRouter ──► "chat_completion" → ChatCompletionHandler
    │                                │
    │                         input_payload validated against JSON Schema
    │                                │
    │                         ProviderRouter
    │                                │
    │              ┌─────────────────┼──────────────────┐
    │              ▼                 ▼                   ▼
    │       OpenAIProvider    BedrockProvider     OllamaProvider
    │       (AsyncOpenAI)     (aiobotocore)       (httpx.AsyncClient)
    │              │                 │                   │
    │              └─────────────────┴──────────────────►│
    │                         token chunks
    │                                │
    │              PROGRESS(token chunk) × N
    │                                │
    │              COMPLETED(full response payload)
    │
    └── stream TaskEvent{PROGRESS…, COMPLETED|FAILED}
            ▲ task_id echoed; timestamp on every event

At startup:
    AdapterConfig parsed from YAML (path from ADAPTER_CONFIG env var)
    API key env-var values resolved → stored as SecretStr; never logged
    ProviderRouter built (immutable)
    AgentServer.RegisterAgent(AgentDef) → AgentRegistryService (retry up to 5 attempts)

On graceful shutdown (SIGTERM):
    AgentServer.DeregisterAgent(agent_id) → AgentRegistryService
    gRPC server stopped
```

---

## A — Approach

### What we WILL do

- Implement a standalone Python 3.12 module at `agents/adapters/llm/` with its own `pyproject.toml` managed by `uv`.
- Parse `AdapterConfig` from a YAML file at startup (path from `ADAPTER_CONFIG` env var); fail fast if the file is missing, invalid, or if the declared provider is not `openai`, `bedrock`, or `ollama`.
- Resolve API key env-var values at startup; store them as `pydantic.SecretStr`; never pass the raw string to logs, error messages, or `CapabilityError.message`.
- Build `ProviderRouter` from `AdapterConfig` at startup; treat it as immutable thereafter.
- Implement `ExecuteCapability`: validate `capability_name` and `task_id`; validate `input_payload` against the declared JSON Schema (`INVALID_INPUT` on failure); wrap the provider call in `asyncio.wait_for` using `timeout_seconds` (`asyncio.TimeoutError` → `TASK_EVENT_TYPE_FAILED` with `code = "TIMEOUT"`); emit per-token PROGRESS events; emit exactly one terminal event.
- Implement `GetCapabilitySchema`: return `input_schema` / `output_schema` from the `AdapterConfig` capability declaration; return `NOT_FOUND` for unknown capability names.
- Register with `AgentRegistryService.RegisterAgent` at startup (exponential-backoff retry, max 5 attempts); deregister on graceful shutdown.
- Two-stage Python Dockerfile: `python:3.12-slim AS builder` with `uv pip install --no-cache` → `python:3.12-slim` runtime stage; final image runs as unprivileged user.
- Add `llm-adapter` service to `infra/docker/docker-compose.yml`.
- Expose gRPC health protocol endpoint.
- Provide `agent-def.yaml.example` as operator documentation.

### What we WILL NOT do

- Accept provider selection or API keys from `input_payload` — all provider config is static YAML or env-var references declared at startup.
- Call blocking I/O on the event loop — Bedrock via `aiobotocore`, Ollama via `httpx.AsyncClient`, all other I/O via `async with` or `await`.
- Implement retry logic — retry is owned by the task broker.
- Support providers beyond `openai`, `bedrock`, and `ollama` in this issue.
- Import `agents/sdk/` — the adapter implements `AgentService` directly via generated proto stubs (ADR-013).
- Store execution state between invocations — stateless (ADR-013).
- Extend any proto contract — all adapter contracts are finalised in `protos/zynax/v1/`.
- Include raw LLM API response bodies, stack traces, or credential values in any log line or `CapabilityError.message`.
- Implement authentication flows beyond API-key env-var references (OAuth, mTLS are out of scope).

### Governing ADRs

- **ADR-001** — gRPC for all Zynax platform calls; no HTTP callbacks to the platform from the adapter.
- **ADR-005** — Apache 2.0 SPDX header on every source file.
- **ADR-009** — Python only for ML-ecosystem adapters; Go for stateless HTTP proxy adapters.
- **ADR-013** — Adapter-first; never import `agents/sdk/`.
- **ADR-016** — BDD `.feature` file committed and CI-green before any implementation code.
- **ADR-017** — `GOWORK=off` for Go; Python adapters use `uv` and `pyproject.toml` exclusively — no `pip install` outside the virtualenv managed by `uv`.
- **ADR-019** — REASONS Canvas committed and Aligned before implementation.

---

## S — Structure

### New paths

```
agents/adapters/llm/
├── pyproject.toml                  Python 3.12; deps: openai, aiobotocore, httpx, grpcio,
│                                   grpcio-tools, pydantic, pyyaml; dev: mypy, ruff, bandit,
│                                   pip-audit, pytest, pytest-asyncio
├── src/llm_adapter/
│   ├── __init__.py                 package marker; version string
│   ├── __main__.py                 entry point: load config → router → registry → gRPC server
│   ├── server.py                   AgentServiceServicer impl; ExecuteCapability; GetCapabilitySchema
│   ├── router.py                   CapabilityRouter; dispatches capability_name → handler
│   ├── config.py                   AdapterConfig + ProviderConfig (pydantic); load_config()
│   └── providers/
│       ├── __init__.py             ProviderRouter factory
│       ├── openai.py               OpenAIProvider: AsyncOpenAI streaming
│       ├── bedrock.py              BedrockProvider: aiobotocore converse streaming
│       └── ollama.py               OllamaProvider: httpx.AsyncClient streaming
├── tests/
│   ├── test_config.py              unit tests: valid config, missing fields, bad provider
│   ├── test_router.py              unit tests: capability routing, unknown capability
│   └── test_providers.py           unit tests: mocked provider responses, timeout, UPSTREAM_ERROR
├── Dockerfile                      two-stage python:3.12-slim; unprivileged user
└── agent-def.yaml.example          operator documentation
```

### Extended paths

- **`protos/tests/features/llm_adapter.feature`** — BDD contract file (committed before implementation)
- **`infra/docker/docker-compose.yml`** — `llm-adapter` service block with config volume mount and env-var references for API keys

### Unchanged paths

- `protos/zynax/v1/` — no proto changes in #383
- `services/` — platform services unchanged
- `agents/sdk/` — never imported by the adapter
- `go.work` — not updated; Python adapters do not participate in the Go workspace

---

## O — Operations

This issue (#383) is a single `feat:` PR. Implementation is broken into logical commits within that PR.

1. **BDD feature file** (#409) — commit `protos/tests/features/llm_adapter.feature` with adapter-specific scenarios: `chat_completion` with OpenAI provider streams PROGRESS then COMPLETED; Bedrock provider streams PROGRESS then COMPLETED; Ollama provider streams PROGRESS then COMPLETED; `timeout_seconds` exceeded emits FAILED with `"TIMEOUT"`; invalid `input_payload` emits FAILED with `"INVALID_INPUT"`; provider API error emits FAILED with `"UPSTREAM_ERROR"` (sanitised message, no credential values); unknown capability name emits `NOT_FOUND`; `GetCapabilitySchema` returns declared schema for `chat_completion`; credential values never appear in any TaskEvent payload or CapabilityError. CI must be green before any implementation code is committed.

2. **Module scaffold** (#410) — `pyproject.toml` (Python 3.12, all runtime and dev dependencies, `mypy --strict` config with `ignore_missing_imports` override for untyped provider SDKs, `ruff` config with Google docstring convention); `src/llm_adapter/__init__.py`; `server.py` skeleton (class with `ExecuteCapability` and `GetCapabilitySchema` method stubs, no logic); `__main__.py` stub.

3. **Config layer** (#410) — `src/llm_adapter/config.py`: `ProviderConfig` pydantic model (fields: `name`, `model`, `api_key_env`, `ollama_base_url`, `max_tokens`, `region`); `AdapterConfig` pydantic model; `load_config(path: str) -> AdapterConfig` reading from YAML file at `path`, resolving API key env-var at load time and storing as `SecretStr`, validating provider name is one of `openai | bedrock | ollama`, failing fast on missing required fields. Unit tests: valid config round-trip, missing `agent_id`, unknown provider name, missing `api_key_env` env-var, `SecretStr` not exposed in `repr`.

4. **Provider handlers** (#411) — `src/llm_adapter/providers/__init__.py` (factory returning the correct provider coroutine from `ProviderConfig`); `providers/openai.py` (`OpenAIProvider` async generator: `openai.AsyncOpenAI` streaming, yields `bytes` token chunks, propagates `OpenAIError` as `UPSTREAM_ERROR`); `providers/bedrock.py` (`BedrockProvider` async generator: `aiobotocore` session, converse-stream API, yields token chunks — boto3 import at any point is a test failure); `providers/ollama.py` (`OllamaProvider` async generator: `httpx.AsyncClient`, `/api/chat` with `stream=True`, yields token chunks, maps HTTP errors to `UPSTREAM_ERROR`); `router.py` (`CapabilityRouter`: dict built from `AdapterConfig`; `dispatch(capability_name) -> ChatCompletionHandler`; `get_schema(capability_name) -> tuple[bytes, bytes]`; raises `KeyError` for unknown names). Unit tests: each provider with a mocked async client; timeout via `asyncio.wait_for`; `UPSTREAM_ERROR` on simulated provider failure.

5. **Registry client** (#412) — `src/llm_adapter/registry/client.py`: `register_agent(config: AdapterConfig, stub) -> None` with exponential-backoff retry (2 s base, ×2, max 5 attempts); `deregister_agent(agent_id: str, stub) -> None`; both use `asyncio` and accept a gRPC stub argument for testability.

6. **Bootstrap** (#412) — `src/llm_adapter/__main__.py`: `async def main()`: load config from `ADAPTER_CONFIG` env var path; build `ProviderRouter`; build `CapabilityRouter`; dial registry; `register_agent`; start gRPC server (health protocol); install `SIGTERM` handler via `asyncio` loop; on signal: `deregister_agent` then stop gRPC server. All I/O in `async with` or `try/finally`.

7. **Dockerfile + docker-compose** (#413) — two-stage `python:3.12-slim` Dockerfile: builder stage uses `uv pip install --no-cache -r requirements.txt` into `/install`; runtime stage copies `/install` and `src/`; final `CMD` runs `python -m llm_adapter`; image runs as unprivileged user. `infra/docker/docker-compose.yml` `llm-adapter` service block with env-var references for API keys (values never in the Compose file). `agent-def.yaml.example` documenting `chat_completion` capability with `input_schema` and `output_schema` examples.

---

## N — Norms

Pulled from root `AGENTS.md` §Hard Constraints, `agents/adapters/AGENTS.md` §Rules, and `docs/patterns/python-agent-guide.md`.

- Commit hygiene: subject ≤ 72 chars, imperative mood, no period, no emojis. `Signed-off-by:` and `Assisted-by: Claude/claude-sonnet-4-6` on every commit. Never `Co-Authored-By:` for AI.
- One PR for this issue. BDD feature file in its own first commit.
- SPDX header `# SPDX-License-Identifier: Apache-2.0` on every `.py` source file.
- Python 3.12. `uv` package manager. `pyproject.toml` as the single source of dependencies, tool config, and scripts.
- `mypy --strict` clean. Provider SDK overrides use `[[tool.mypy.overrides]]` with `ignore_missing_imports = true` — never silence errors globally.
- `ruff` clean, including `ruff D` Google docstring convention on all public functions and classes.
- `bandit` + `pip-audit` clean in CI.
- `asyncio` throughout. No blocking calls on the event loop — Bedrock via `aiobotocore`, Ollama via `httpx.AsyncClient`.
- `asyncio.wait_for` for timeout enforcement; catch `asyncio.TimeoutError` → emit `TASK_EVENT_TYPE_FAILED` with `code = "TIMEOUT"`.
- `pydantic.SecretStr` for API key fields. Never log `SecretStr` values. Never include them in `CapabilityError.message`. Never pass them as plain `str` to provider SDKs; use `.get_secret_value()` only inside the provider call, never stored in a local variable that outlives the call scope.
- Functions ≤ 20 lines. If a function exceeds 20 lines it must be decomposed.
- All I/O resources (gRPC channels, HTTP clients, async iterators) closed in `finally` blocks or `async with` context managers.
- Never import `agents/sdk/`. Adapter implements `AgentService` directly via generated proto stubs (ADR-013).
- Platform calls via gRPC stubs only — never HTTP to Zynax platform services (ADR-001).
- `input_payload` validated against declared `input_schema` before any provider call; return `INVALID_INPUT` on failure.
- At least one `TASK_EVENT_TYPE_PROGRESS` event emitted before the terminal event (per-token events satisfy this naturally; if a provider yields no chunks before terminal, emit one synthetic PROGRESS event with an empty chunk).
- Exactly one terminal event per stream. No events after terminal.
- `task_id` echoed on every `TaskEvent`. `timestamp` populated on every `TaskEvent`.
- `CapabilityError.message` sanitised: no raw LLM API response bodies, no stack traces, no credential values. Truncated at 512 chars.
- Structured logs to stdout only (`logging` with JSON formatter or `structlog`). Never log credential values, raw provider responses, or full `input_payload`.
- Two-stage `python:3.12-slim` Dockerfile. Final image runs as unprivileged user.

---

## S — Safeguards

### Context Security (mandatory before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no email addresses, no personal names in sensitive context
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards

- **Never** accept provider selection, API keys, or model identifiers from `input_payload` — all provider config is static YAML or env-var references declared in `AdapterConfig` at startup.
- **Never** call blocking I/O on the `asyncio` event loop — Bedrock must use `aiobotocore` (not `boto3`); Ollama must use `httpx.AsyncClient`; all other I/O via `async with` or `await`.
- **Never** log or include `SecretStr` values, raw LLM API response bodies, or stack traces in `TaskEvent` payloads, `CapabilityError.message`, or structured log fields.
- **Never** emit a `TaskEvent` after the terminal event (`TASK_EVENT_TYPE_COMPLETED` or `TASK_EVENT_TYPE_FAILED`).
- **Never** import `agents/sdk/` — the adapter implements `AgentService` directly via generated proto stubs (ADR-013).
- **Never** store execution state across `ExecuteCapabilityRequest` invocations — the adapter is stateless (ADR-013).
- **Never** call Zynax platform services via HTTP — gRPC stubs only (ADR-001).
- **Never** skip `input_payload` JSON Schema validation — validate before any provider call and return `INVALID_INPUT` on failure.
- **Never** extend proto contracts in this issue — all adapter contracts are finalised in `protos/zynax/v1/`.
- **Never** commit implementation code before the BDD `.feature` file is committed and CI-green (ADR-016).
- **Never** commit implementation code before this Canvas is Aligned (ADR-019).
- **Never** suppress `mypy` errors globally — use `[[tool.mypy.overrides]]` scoped to specific untyped provider packages.
- **Never** call `.get_secret_value()` outside the provider implementation and store the result in a variable that outlives the provider call.
- **Never** truncate the async stream without emitting a terminal event — every stream must end with exactly one COMPLETED or FAILED event.
