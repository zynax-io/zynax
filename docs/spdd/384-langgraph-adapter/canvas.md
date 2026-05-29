<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — LangGraph Adapter (LangGraph Graph Capability Adapter)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #384
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-08
**Status:** Implemented

---

## R — Requirements

- **Problem:** There is no reusable way to invoke a LangGraph graph as a step in a Zynax workflow. Any operator who wants to run an agent graph must write a bespoke gRPC adapter from scratch — with async streaming, per-node event mapping, error handling, and a Dockerfile — even though the invocation pattern is always the same: compile the graph, stream node outputs as PROGRESS events, and deliver the final state as COMPLETED. This barrier means agentic graph steps cannot be added to workflows without significant per-project engineering work.

- **Missing capability:** A config-driven Python adapter that maps any LangGraph `StateGraph` to a named Zynax capability, compiles the graph at startup, streams per-node PROGRESS events during execution, and delivers the final graph state as COMPLETED — without any code changes to the calling workflow.

- **Definition of done — observable outcomes:**
  - `zynax apply agent-def.yaml` registers the langgraph-adapter; a workflow step calls the mapped capability name; the task broker receives a `TASK_EVENT_TYPE_COMPLETED` `TaskEvent` with the final graph state as `payload`.
  - Each graph node that fires during execution produces a `TASK_EVENT_TYPE_PROGRESS` event containing `node_name` and the `state_update` for that node.
  - If no node fires within 2 seconds, a ticker `TASK_EVENT_TYPE_PROGRESS` event is emitted to confirm liveness.
  - A request exceeding `timeout_seconds` emits `TASK_EVENT_TYPE_FAILED` with `CapabilityError.code = "TIMEOUT"`.
  - A graph that fails to import or compile at startup causes the adapter process to exit non-zero immediately (fail-fast).
  - A graph that raises an exception during execution emits `TASK_EVENT_TYPE_FAILED` with a mapped `CapabilityError` code and a sanitised message.
  - Invalid `input_payload` (missing required fields, wrong types) emits `TASK_EVENT_TYPE_FAILED` with `code = "INVALID_INPUT"`.
  - `GetCapabilitySchema` returns the JSON Schema declared in the `AgentDef` YAML for the registered capability.
  - `make test` green · `make lint` clean · `make security` clean.
  - BDD contract scenarios in `protos/tests/features/langgraph_adapter.feature` pass.

---

## E — Entities

### Existing entities consumed (no changes in #384)

- **`AgentService`** (`protos/zynax/v1/agent.proto`) — two-RPC contract implemented by the adapter: `ExecuteCapability` (server-streaming `TaskEvent`) and `GetCapabilitySchema`. Contract invariants: exactly one terminal event per stream; `task_id` echoed on every event; `timeout_seconds` honoured; no events after terminal.
- **`AgentRegistryService`** (`protos/zynax/v1/agent_registry.proto`) — `RegisterAgent` called at startup; `DeregisterAgent` called on graceful shutdown.
- **`AgentDef`** (proto message) — `agent_id`, `name`, `description`, `endpoint` (`host:port`), `capabilities[]`. Built from YAML config at startup and sent to the registry.
- **`CapabilityDef`** (proto message) — `name` (snake_case, 1–64 chars), `description`, `input_schema` (JSON Schema bytes), `output_schema` (JSON Schema bytes).
- **`ExecuteCapabilityRequest`** (proto message) — `request_id`, `capability_name`, `task_id`, `workflow_id`, `input_payload` (JSON bytes), `timeout_seconds`.
- **`TaskEvent`** (proto message) — `task_id`, `event_type` (PROGRESS / COMPLETED / FAILED), `payload`, `timestamp`, `error` (`CapabilityError`).
- **`CapabilityError`** (proto message) — `code`, `message`, `details`. Well-known codes: `"TIMEOUT"`, `"INVALID_INPUT"`, `"UPSTREAM_ERROR"`, `"RESOURCE_EXHAUSTED"`, `"INTERNAL"`.

### New entities (introduced by #384)

- **`AdapterConfig`** — top-level YAML struct parsed at startup via `load_config()`. Fields: `agent_id`, `name`, `description`, `endpoint` (bind `host:port`), `registry_endpoint` (agent-registry `host:port`), `capabilities[]` (list of capability declarations), `graphs[]` (list of `GraphMount`). Never contains credential values.
- **`GraphMount`** — maps one capability name to a Python module import path and the attribute name of the `StateGraph` object within that module. Fields: `capability_name` (snake_case), `graph_module` (dotted Python module path, e.g. `my_package.my_graph`), `graph_attr` (attribute name of the `StateGraph` in the module, default `graph`). Compiled at startup, not per-request.
- **`GraphLoader`** — loads and compiles all `GraphMount` entries at adapter startup. `load_all(mounts: list[GraphMount]) -> dict[str, CompiledGraph]`: for each mount, `importlib.import_module(mount.graph_module)`, retrieves `getattr(module, mount.graph_attr)`, calls `.compile()`, stores the compiled graph keyed by `capability_name`. If any graph fails to import or compile, raises immediately — the adapter process must not start with a partially loaded graph set.
- **`LangGraphHandler`** — async coroutine implementing the graph capability. `execute(compiled_graph, input_state, timeout_seconds, stream)`: wraps `compiled_graph.astream(input_state)` in `asyncio.wait_for`; for each `(node_name, state_update)` tuple yielded by the stream, emits a `TASK_EVENT_TYPE_PROGRESS` event with `payload = json.dumps({"node": node_name, "update": state_update}, default=str)`; if no node fires within 2 s, emits a ticker PROGRESS event; when the stream is exhausted, serialises the final graph state with `json.dumps(final_state, default=str)` and emits `TASK_EVENT_TYPE_COMPLETED`. Stateless between invocations — no checkpoint state persisted.
- **`CapabilityRouter`** — map of `capability_name → (compiled_graph, LangGraphHandler)` built at startup from the output of `GraphLoader.load_all()`. Immutable after initialisation. Dispatches `ExecuteCapabilityRequest.capability_name` to the correct compiled graph and handler.

### Entity relationships

```
Task Broker
    │ gRPC ExecuteCapabilityRequest
    ▼
AgentServer (AgentServiceServer)
    │
    ├── CapabilityRouter ──► capability_name → (CompiledGraph, LangGraphHandler)
    │                                │
    │                         input_payload validated against JSON Schema
    │                                │
    │                         LangGraphHandler.execute(compiled_graph, input_state)
    │                                │
    │                         compiled_graph.astream(input_state)
    │                                │
    │              ┌─────────────────┴──────────────────────┐
    │              │ (node_name, state_update) × N          │ no node within 2s
    │              │                                         │
    │   PROGRESS{node, update}                      PROGRESS{ticker: true}
    │              │
    │   stream exhausted → final_state
    │              │
    │   COMPLETED(json.dumps(final_state, default=str))
    │
    └── stream TaskEvent{PROGRESS…, COMPLETED|FAILED}
            ▲ task_id echoed; timestamp on every event

At startup:
    AdapterConfig parsed from YAML (path from ADAPTER_CONFIG env var)
    GraphLoader.load_all(config.graphs) → fails fast if any graph import/compile fails
    CapabilityRouter built (immutable)
    AgentServer.RegisterAgent(AgentDef) → AgentRegistryService (retry up to 5 attempts)

On graceful shutdown (SIGTERM):
    AgentServer.DeregisterAgent(agent_id) → AgentRegistryService
    gRPC server stopped
```

---

## A — Approach

### What we WILL do

- Implement a standalone Python 3.12 module at `agents/adapters/langgraph/` with its own `pyproject.toml` managed by `uv`.
- Parse `AdapterConfig` from a YAML file at startup (path from `ADAPTER_CONFIG` env var); fail fast if the file is missing or invalid.
- Call `GraphLoader.load_all()` at startup; if any graph fails to import or compile, log the error and exit non-zero immediately — a partially loaded adapter must never serve requests.
- Build `CapabilityRouter` from the compiled graphs at startup; treat it as immutable thereafter.
- Implement `ExecuteCapability`: validate `capability_name` and `task_id`; validate `input_payload` against the declared JSON Schema (`INVALID_INPUT` on failure); wrap `compiled_graph.astream(input_state)` in `asyncio.wait_for`; emit per-node PROGRESS events; emit a ticker PROGRESS event if no node fires within 2 s; emit exactly one terminal event with the final graph state serialised via `json.dumps(..., default=str)`.
- Implement `GetCapabilitySchema`: return `input_schema` / `output_schema` from the `AdapterConfig` capability declaration; return `NOT_FOUND` for unknown capability names.
- Register with `AgentRegistryService.RegisterAgent` at startup (exponential-backoff retry, max 5 attempts); deregister on graceful shutdown.
- Two-stage Python Dockerfile: `python:3.12-slim AS builder` → `python:3.12-slim` runtime stage; final image runs as unprivileged user.
- Add `langgraph-adapter` service to `infra/docker/docker-compose.yml`.
- Expose gRPC health protocol endpoint.
- Provide `agent-def.yaml.example` as operator documentation.

### What we WILL NOT do

- Use LangGraph as a Zynax workflow engine — the adapter wraps LangGraph as a **capability** only. LangGraph as an engine replacement for Temporal is out of scope for M5 and requires a new ADR (ADR-015 governs this boundary).
- Persist graph checkpoint state between invocations — each `ExecuteCapabilityRequest` is an independent, stateless execution.
- Accept graph module paths or attribute names from `input_payload` — all graph config is static YAML declared at startup.
- Implement LangGraph checkpoint backends (`SqliteSaver`, `MemorySaver`, etc.) — checkpointing is out of scope.
- Import `agents/sdk/` — the adapter implements `AgentService` directly via generated proto stubs (ADR-013).
- Store execution state between invocations — stateless (ADR-013).
- Extend any proto contract — all adapter contracts are finalised in `protos/zynax/v1/`.
- Call blocking I/O on the event loop — all I/O via `async with` or `await`.
- Hard-depend on the llm-adapter module — the scaffold pattern is referenced structurally but there is no Python import dependency between the two adapters.

### Governing ADRs

- **ADR-001** — gRPC for all Zynax platform calls; no HTTP callbacks to the platform from the adapter.
- **ADR-005** — Apache 2.0 SPDX header on every source file.
- **ADR-009** — Python only for ML-ecosystem adapters.
- **ADR-013** — Adapter-first; never import `agents/sdk/`.
- **ADR-015** — Pluggable workflow engines. LangGraph-adapter wraps LangGraph as a capability (not an engine); the engine boundary is a one-way door that requires a new ADR before it can be crossed.
- **ADR-016** — BDD `.feature` file committed and CI-green before any implementation code.
- **ADR-019** — REASONS Canvas committed and Aligned before implementation.

---

## S — Structure

### New paths

```
agents/adapters/langgraph/
├── pyproject.toml                  Python 3.12; deps: langgraph, grpcio, grpcio-tools,
│                                   pydantic, pyyaml; dev: mypy, ruff, bandit, pip-audit,
│                                   pytest, pytest-asyncio
├── src/langgraph_adapter/
│   ├── __init__.py                 package marker; version string
│   ├── __main__.py                 entry point: load config → loader → router → registry → gRPC server
│   ├── server.py                   AgentServiceServicer impl; ExecuteCapability; GetCapabilitySchema
│   ├── router.py                   CapabilityRouter; dispatches capability_name → (graph, handler)
│   ├── graph_loader.py             GraphLoader.load_all(); fail-fast on import/compile error
│   └── config.py                   AdapterConfig + GraphMount (pydantic); load_config()
├── tests/
│   ├── test_config.py              unit tests: valid config, missing fields, unknown graph_module
│   ├── test_graph_loader.py        unit tests: successful load, import failure, compile failure
│   ├── test_router.py              unit tests: capability routing, unknown capability
│   └── test_handler.py             unit tests: per-node PROGRESS, ticker PROGRESS, timeout,
│                                   COMPLETED serialisation, graph exception → INTERNAL
├── Dockerfile                      two-stage python:3.12-slim; unprivileged user
└── agent-def.yaml.example          operator documentation
```

### Extended paths

- **`protos/tests/features/langgraph_adapter.feature`** — BDD contract file (committed before implementation)
- **`infra/docker/docker-compose.yml`** — `langgraph-adapter` service block with config volume mount and graph module path visible to the container

### Unchanged paths

- `protos/zynax/v1/` — no proto changes in #384
- `services/` — platform services unchanged
- `agents/sdk/` — never imported by the adapter
- `agents/adapters/llm/` — scaffold is referenced for structural consistency only; no Python import dependency
- `go.work` — not updated; Python adapters do not participate in the Go workspace

---

## O — Operations

This issue (#384) is a single `feat:` PR. Implementation is broken into logical commits within that PR.

1. **BDD feature file** (#414) — commit `protos/tests/features/langgraph_adapter.feature` with adapter-specific scenarios: mapped capability streams per-node PROGRESS then COMPLETED with final graph state; ticker PROGRESS emitted when no node fires within 2 s; `timeout_seconds` exceeded emits FAILED with `"TIMEOUT"`; invalid `input_payload` emits FAILED with `"INVALID_INPUT"`; graph exception during execution emits FAILED with mapped code and sanitised message; unknown capability name returns `NOT_FOUND`; `GetCapabilitySchema` returns declared schema; graph state serialised with `json.dumps(..., default=str)` fallback (non-JSON-serialisable values become strings); adapter fails to start if any graph fails to import or compile. CI must be green before any implementation code is committed.

2. **Module scaffold** (#415) — `pyproject.toml` (Python 3.12, all runtime and dev dependencies, `mypy --strict` config with `ignore_missing_imports` override for the `langgraph` package, `ruff` config with Google docstring convention); `src/langgraph_adapter/__init__.py`; `server.py` skeleton (class with `ExecuteCapability` and `GetCapabilitySchema` method stubs, no logic); `__main__.py` stub.

3. **Config layer** (#415) — `src/langgraph_adapter/config.py`: `GraphMount` pydantic model (fields: `capability_name`, `graph_module`, `graph_attr` with default `"graph"`); `AdapterConfig` pydantic model; `load_config(path: str) -> AdapterConfig` reading from YAML file at `path`, validating that each `GraphMount.capability_name` is unique and in snake_case, failing fast on missing required fields. Unit tests: valid config round-trip, duplicate capability names, missing `agent_id`, invalid `capability_name` format.

4. **Graph loader + handler** (#416) — `src/langgraph_adapter/graph_loader.py`: `GraphLoader.load_all(mounts: list[GraphMount]) -> dict[str, CompiledGraph]`: iterates mounts; calls `importlib.import_module(mount.graph_module)`, retrieves `getattr(module, mount.graph_attr)`, calls `.compile()`; raises `RuntimeError` immediately on any failure (caller must not catch and continue). `src/langgraph_adapter/` handler logic in `server.py` or a dedicated `handler.py`: `LangGraphHandler.execute()` async method: `asyncio.wait_for(compiled_graph.astream(input_state), timeout_seconds)`, iterates `(node_name, state_update)` tuples, emits PROGRESS per node, uses `asyncio.wait_for` on each iteration step with 2 s ticker fallback, collects final state, emits COMPLETED with `json.dumps(final_state, default=str)`, maps graph exceptions to `CapabilityError` codes (`ValueError` → `"INVALID_INPUT"`, others → `"INTERNAL"`). Unit tests: dummy `StateGraph` with two nodes; verify PROGRESS count matches node count; ticker fires when node delay > 2 s (mocked); `asyncio.TimeoutError` → FAILED `"TIMEOUT"`; `ValueError` from graph → FAILED `"INVALID_INPUT"`.

5. **Registry client** (#417) — `src/langgraph_adapter/registry/client.py`: `register_agent(config: AdapterConfig, stub) -> None` with exponential-backoff retry (2 s base, ×2, max 5 attempts); `deregister_agent(agent_id: str, stub) -> None`; both use `asyncio` and accept a gRPC stub argument for testability. (Same pattern as llm-adapter step 5.)

6. **Bootstrap** (#417) — `src/langgraph_adapter/__main__.py`: `async def main()`: load config from `ADAPTER_CONFIG` env var path; `GraphLoader.load_all(config.graphs)` — exit non-zero on failure; build `CapabilityRouter` from loaded graphs; dial registry; `register_agent`; start gRPC server (health protocol); install `SIGTERM` handler via `asyncio` loop; on signal: `deregister_agent` then stop gRPC server. All I/O in `async with` or `try/finally`.

7. **Dockerfile + docker-compose** (#418) — two-stage `python:3.12-slim` Dockerfile: builder stage uses `uv pip install --no-cache -r requirements.txt` into `/install`; runtime stage copies `/install`, `src/`, and any graph modules the operator mounts (via volume); final `CMD` runs `python -m langgraph_adapter`; image runs as unprivileged user. `infra/docker/docker-compose.yml` `langgraph-adapter` service block with config volume mount and a `volumes` entry for graph module paths. `agent-def.yaml.example` documenting a single-graph capability with `input_schema` and `output_schema` examples.

---

## N — Norms

Pulled from root `AGENTS.md` §Hard Constraints, `agents/adapters/AGENTS.md` §Rules, and `docs/patterns/python-agent-guide.md`.

- Commit hygiene: subject ≤ 72 chars, imperative mood, no period, no emojis. `Signed-off-by:` and `Assisted-by: Claude/claude-sonnet-4-6` on every commit. Never `Co-Authored-By:` for AI.
- One PR for this issue. BDD feature file in its own first commit.
- SPDX header `# SPDX-License-Identifier: Apache-2.0` on every `.py` source file.
- Python 3.12. `uv` package manager. `pyproject.toml` as the single source of dependencies, tool config, and scripts.
- `mypy --strict` clean. LangGraph SDK overrides use `[[tool.mypy.overrides]]` with `ignore_missing_imports = true` — never silence errors globally.
- `ruff` clean, including `ruff D` Google docstring convention on all public functions and classes.
- `bandit` + `pip-audit` clean in CI.
- `asyncio` throughout. No blocking calls on the event loop.
- `asyncio.wait_for` for timeout enforcement; catch `asyncio.TimeoutError` → emit `TASK_EVENT_TYPE_FAILED` with `code = "TIMEOUT"`.
- Functions ≤ 20 lines. If a function exceeds 20 lines it must be decomposed.
- All I/O resources (gRPC channels, async iterators) closed in `finally` blocks or `async with` context managers.
- Never import `agents/sdk/`. Adapter implements `AgentService` directly via generated proto stubs (ADR-013).
- Platform calls via gRPC stubs only — never HTTP to Zynax platform services (ADR-001).
- Never use LangGraph as a Zynax workflow engine — the adapter wraps LangGraph as a capability only (ADR-015). This boundary is a one-way door; crossing it requires a new ADR and is explicitly out of scope for M5. Code comments in `server.py` and `graph_loader.py` must reference ADR-015 at the relevant call site.
- Never persist graph checkpoint state between invocations — each `ExecuteCapabilityRequest` is an independent execution. Do not instantiate `SqliteSaver`, `MemorySaver`, or any LangGraph checkpoint backend.
- `input_payload` validated against declared `input_schema` before graph execution; return `INVALID_INPUT` on failure.
- At least one `TASK_EVENT_TYPE_PROGRESS` event emitted before the terminal event (per-node events satisfy this naturally; ticker fires if no node runs within 2 s).
- Exactly one terminal event per stream. No events after terminal.
- `task_id` echoed on every `TaskEvent`. `timestamp` populated on every `TaskEvent`.
- Graph state serialised with `json.dumps(final_state, default=str)` — the `default=str` fallback must always be present to handle non-JSON-serialisable LangGraph state values.
- `CapabilityError.message` sanitised: no raw graph state dumps, no stack traces. Truncated at 512 chars.
- Structured logs to stdout only (`logging` with JSON formatter or `structlog`). Never log full `input_payload` or graph state at INFO level.
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

- **Never** use LangGraph as a Zynax workflow engine — the adapter wraps LangGraph as a capability wrapper only (ADR-015). Using LangGraph as an engine replacement for Temporal is a one-way architectural door that requires a new ADR; it is explicitly out of scope for M5.
- **Never** persist graph checkpoint state between invocations — each `ExecuteCapabilityRequest` is an independent, stateless execution. Do not instantiate any LangGraph checkpoint backend (`SqliteSaver`, `MemorySaver`, or similar).
- **Never** accept graph module paths or graph attribute names from `input_payload` — all graph config is static YAML declared in `AdapterConfig` at startup.
- **Never** start the adapter with a partially loaded graph set — if `GraphLoader.load_all()` fails for any mount, the process must exit non-zero immediately.
- **Never** emit a `TaskEvent` after the terminal event (`TASK_EVENT_TYPE_COMPLETED` or `TASK_EVENT_TYPE_FAILED`).
- **Never** import `agents/sdk/` — the adapter implements `AgentService` directly via generated proto stubs (ADR-013).
- **Never** store execution state across `ExecuteCapabilityRequest` invocations — the adapter is stateless (ADR-013).
- **Never** call Zynax platform services via HTTP — gRPC stubs only (ADR-001).
- **Never** call blocking I/O on the `asyncio` event loop — all I/O via `async with` or `await`.
- **Never** skip `input_payload` JSON Schema validation — validate before graph execution and return `INVALID_INPUT` on failure.
- **Never** extend proto contracts in this issue — all adapter contracts are finalised in `protos/zynax/v1/`.
- **Never** commit implementation code before the BDD `.feature` file is committed and CI-green (ADR-016).
- **Never** commit implementation code before this Canvas is Aligned (ADR-019).
- **Never** suppress `mypy` errors globally — use `[[tool.mypy.overrides]]` scoped to the `langgraph` package.
- **Never** omit the `default=str` fallback in `json.dumps` calls on graph state — LangGraph state may contain non-serialisable Python objects.
- **Never** log full `input_payload` or raw graph state at INFO level — these may contain user data or large intermediate states.
- **Never** import from another adapter module (e.g. `llm_adapter`) — adapters are independently deployable and have no inter-adapter Python dependencies.
