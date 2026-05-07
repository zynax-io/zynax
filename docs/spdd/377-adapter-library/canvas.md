<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M5 Adapter Library

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #377
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-07
**Status:** Draft

---

## R — Requirements

- **Problem:** M4 proved the Zynax control plane works end-to-end — `zynax apply workflow.yaml` compiles, routes, and dispatches to Temporal. But the capability side is empty: every capability must be a bespoke agent service that the operator writes from scratch. Running an HTTP call, invoking an LLM, opening a PR, or triggering a CI pipeline each requires a new gRPC service with proto stubs, YAML config, Docker image, and a streaming event loop. This is the gap that blocks adoption: the platform is ready but there are no reusable capabilities to dispatch.

- **Missing capability:** A library of ready-to-use adapter services that wrap common external systems — REST APIs, LLM providers, Git platforms, CI systems, and LangGraph applications — so that operators can turn any existing service into a Zynax capability by writing an `AgentDef` YAML, not a new service.

- **M5 delivers:** Five standalone adapter services covering the most common integration patterns:
  - `http-adapter` (Go) — wraps any REST API via config-only route mapping (#380)
  - `git-adapter` (Go) — GitHub/GitLab operations: `open_pr`, `request_review`, `get_diff` (#381)
  - `ci-adapter` (Go) — CI pipeline triggers: `trigger_workflow`, `get_run_status` (#382)
  - `llm-adapter` (Python) — LLM inference via OpenAI / Bedrock / Ollama (#383)
  - `langgraph-adapter` (Python) — any LangGraph graph as a named capability (#384)

- **Definition of done — observable outcomes:**
  - `zynax apply agent-def.yaml` registers an http-adapter instance; a workflow step calls the declared capability; the task broker receives a `TASK_EVENT_TYPE_COMPLETED` `TaskEvent` with the proxied HTTP response payload.
  - A workflow step calls `open_pr` via the git-adapter; a PR appears in the target Git repository with the correct title, body, and head branch.
  - A workflow step calls `trigger_workflow` via the ci-adapter; a GitHub Actions run is dispatched; its status streams as `TASK_EVENT_TYPE_PROGRESS` events until terminal state.
  - A workflow step calls `chat_completion` via the llm-adapter; token chunks arrive as `TASK_EVENT_TYPE_PROGRESS` events; the full response arrives as `TASK_EVENT_TYPE_COMPLETED`.
  - A workflow step calls the mapped capability via the langgraph-adapter; per-node progress events stream to the task broker; the final graph output arrives as `TASK_EVENT_TYPE_COMPLETED`.
  - All five adapters: `make test` green · `make lint` clean · `make security` clean.
  - BDD contract scenarios in `protos/tests/features/` pass for every adapter.

---

## E — Entities

### Existing entities consumed (contracts unchanged in M5)

- **`AgentService`** (`protos/zynax/v1/agent.proto`) — the two-RPC contract every adapter implements: `ExecuteCapability` (server-streaming `TaskEvent`) and `GetCapabilitySchema`. Contract invariants: exactly one terminal event per stream; `task_id` echoed on every event; `timeout_seconds` honoured; no events after terminal. No proto changes in M5.
- **`AgentRegistryService`** (`protos/zynax/v1/agent_registry.proto`) — adapters call `RegisterAgent` on startup with their `AgentDef`; `DeregisterAgent` on graceful shutdown. No proto changes in M5.
- **`AgentDef`** (proto message) — declares `agent_id`, `name`, `description`, `endpoint` (`host:port`), and a list of `CapabilityDef`. Adapters build this from their YAML config at startup.
- **`CapabilityDef`** (proto message) — `name` (snake_case, 1–64 chars), `description`, `input_schema` (JSON Schema bytes), `output_schema` (JSON Schema bytes). Declared in the AgentDef YAML; never hardcoded in source.
- **`ExecuteCapabilityRequest`** (proto message) — `request_id` (UUID v4), `capability_name`, `task_id`, `workflow_id`, `input_payload` (JSON bytes), `timeout_seconds`. Sent by the task broker to trigger a capability.
- **`TaskEvent`** (proto message) — `task_id`, `event_type` (PROGRESS / COMPLETED / FAILED), `payload` (JSON bytes), `timestamp`, `error` (`CapabilityError`). Adapters stream these back to the task broker.
- **`CapabilityError`** (proto message) — `code` (well-known string: `"TIMEOUT"`, `"INVALID_INPUT"`, `"UPSTREAM_ERROR"`, `"RESOURCE_EXHAUSTED"`, `"INTERNAL"`), `message` (human-readable, sanitised — no raw API responses, no stack traces, no credential values).

### New entities (introduced by M5)

- **`AdapterConfig`** — YAML struct parsed at startup from the `AgentDef` YAML file on disk. Common fields: `agent_id`, `name`, `endpoint`, `capabilities[]`. Adapter-specific subsections carry system-facing config (route maps, provider selection, auth env-var references). Never contains credential values — only env-var name references.
- **`CapabilityRouter`** — dispatches `ExecuteCapabilityRequest.capability_name` to the registered `CapabilityHandler`. One per adapter process; initialised from `AdapterConfig` at startup; immutable thereafter.
- **`CapabilityHandler`** (interface/protocol) — one implementation per declared capability. Translates a single `ExecuteCapabilityRequest` into the system-facing protocol call and yields `TaskEvent` values. Stateless between invocations.
- **`RouteConfig`** (http-adapter) — maps one capability name to an HTTP method, URL (static config only — no user-controlled fields), and optional static request headers. URL is never derived from `input_payload` (SSRF prevention).
- **`GitConfig`** (git-adapter) — auth mode (`pat` / `github-app`); token sourced from the env-var name declared in `AdapterConfig` at startup (never from `input_payload`). Target org and repo declared per capability. GitLab: config flag only (stub).
- **`CIConfig`** (ci-adapter) — provider flag (`github-actions` / `jenkins-stub`); token from env-var reference; org/repo/workflow-id per capability. Includes poll interval and max-poll configuration.
- **`PollLoop`** (ci-adapter) — exponential backoff polling (2 s → 4 s → 8 s → max 30 s) for run status; honours `timeout_seconds` via `context.Context` deadline; emits `TASK_EVENT_TYPE_PROGRESS` per cycle.
- **`ProviderConfig`** (llm-adapter) — provider (`openai` / `bedrock` / `ollama`); model name from env-var reference; Ollama base URL from config; max-tokens ceiling from config.
- **`ChatCompletionHandler`** (llm-adapter) — async coroutine; token chunks → `TASK_EVENT_TYPE_PROGRESS` events; terminal `TASK_EVENT_TYPE_COMPLETED` with full response payload. Provider routing: `openai.AsyncOpenAI` for OpenAI; `aiobotocore` for Bedrock; `httpx.AsyncClient` for Ollama REST.
- **`GraphMount`** (langgraph-adapter) — maps one capability name to a Python module path and graph entry node; imported and compiled at adapter startup, not per-request.
- **`GraphLoader`** (langgraph-adapter) — imports the graph module from `GraphMount.graph_module`, retrieves the `StateGraph` object, and calls `graph.compile()`. Fails fast at startup if any graph fails to load.
- **`LangGraphHandler`** (langgraph-adapter) — async coroutine; calls `compiled_graph.astream(input_state)`, yielding one `TASK_EVENT_TYPE_PROGRESS` per `(node_name, state_update)` tuple; terminal `TASK_EVENT_TYPE_COMPLETED` with final graph state as JSON.

### Entity relationships

```
Task Broker
    │ gRPC ExecuteCapabilityRequest (server-streaming)
    ▼
Adapter gRPC Server
    │
    ├── CapabilityRouter ──► CapabilityHandler (one per declared capability)
    │                                │
    │                                ▼
    │                       System-facing call
    │              (HTTP REST / GitHub API / CI API / LLM API / LangGraph graph)
    │
    └── stream TaskEvent{PROGRESS…, COMPLETED|FAILED}
            ▲ task_id echoed on every event; timestamp on every event

At startup:
    Adapter ──► AgentRegistryService.RegisterAgent(AgentDef{capabilities…})

On graceful shutdown:
    Adapter ──► AgentRegistryService.DeregisterAgent(agent_id)
```

---

## A — Approach

### What we WILL do

- Implement five standalone adapter services — each is an independently deployable gRPC server with its own module (`go.mod` or `pyproject.toml`), Dockerfile, and docker-compose entry.
- Use **Go** for `http/`, `git/`, `ci/` adapters — stateless HTTP proxy pattern with no ML library dependency. Single-binary deployment, smaller images, faster cold start.
- Use **Python 3.12** for `llm/` and `langgraph/` adapters — the ML ecosystem (openai, aiobotocore, langgraph) is Python-native. Forcing Go here gains nothing and loses the ecosystem.
- All adapters are **config-driven**: capabilities are declared in an `AgentDef` YAML file; the adapter reads this at startup and never hardcodes capability names or system endpoints in source.
- BDD `.feature` file committed to `protos/tests/features/` before any implementation code per adapter (ADR-016).
- Individual REASONS Canvas committed in `docs/spdd/` before implementation per adapter PR (ADR-019).
- Each adapter registers with `AgentRegistryService.RegisterAgent` on startup and calls `DeregisterAgent` on graceful shutdown.
- Go adapters: standalone `go.mod` per adapter module; added to `go.work` with `use` directive.
- Python adapters: standalone `pyproject.toml` with `uv`; two-stage Docker image.
- The http-adapter (step 1) establishes the Go scaffold; git/ci adapters reuse it structurally.
- The llm-adapter (step 4) establishes the Python scaffold; langgraph-adapter reuses it structurally.

### What we WILL NOT do

- Import `agents/sdk/` in any adapter (ADR-013 — adapter-first, no SDK required).
- Store execution state between `ExecuteCapabilityRequest` invocations (stateless per ADR-013).
- Accept user-controlled URLs in `input_payload` for the http-adapter (SSRF prevention — all routes are static config).
- Implement webhook ingestion (inbound events to Zynax — out of M5 scope).
- Deploy to Kubernetes — Docker Compose only in M5 (Kubernetes is M6).
- Use LangGraph as a Zynax workflow engine — the langgraph-adapter wraps LangGraph as a **capability**, not as an engine. LangGraph as an engine replacement for Temporal is M6+, requires a new ADR (ADR-015 governs this boundary).
- Write any adapter in two languages or mix languages within one adapter.
- Extend any proto contract in M5 — all adapter contracts are already finalised in `protos/zynax/v1/`.

### Governing ADRs

- **ADR-001** — gRPC-only for all Zynax platform calls. No REST callbacks from adapters to platform.
- **ADR-009** — Go for stateless proxies; Python only where ML ecosystem is the explicit justification.
- **ADR-013** — Adapter-first: no SDK required. SDK is never imported in an adapter.
- **ADR-015** — Pluggable workflow engines. LangGraph-adapter wraps LangGraph as a capability (not an engine); crossing to "LangGraph as engine" requires a new ADR.
- **ADR-016** — BDD `.feature` file committed and CI-green before any implementation code.
- **ADR-019** — REASONS Canvas committed before any implementation code for each `feat:` PR.

---

## S — Structure

### New paths (one per adapter issue)

```
agents/adapters/
├── http/                        ← feat(adapters/http) #380  — Go module
│   ├── go.mod                   (module github.com/zynax-io/zynax/agents/adapters/http)
│   ├── cmd/http-adapter/
│   │   └── main.go              gRPC server bootstrap + graceful shutdown
│   ├── internal/
│   │   ├── adapter.go           CapabilityRouter + HTTP proxy CapabilityHandler
│   │   └── config.go            AdapterConfig YAML parsing + RouteConfig
│   ├── Dockerfile               two-stage Alpine (golang:*-alpine → alpine:*)
│   └── agent-def.yaml.example   operator documentation
│
├── git/                         ← feat(adapters/git) #381  — Go module
│   └── (same layout as http/)   go-github client; GitConfig; open_pr/request_review/get_diff
│
├── ci/                          ← feat(adapters/ci) #382  — Go module
│   └── (same layout as http/)   GitHub Actions REST API; CIConfig; PollLoop; trigger_workflow/get_run_status
│
├── llm/                         ← feat(adapters/llm) #383  — Python module
│   ├── pyproject.toml           openai, aiobotocore, httpx as deps; uv
│   ├── src/llm_adapter/
│   │   ├── server.py            gRPC server; ExecuteCapability; GetCapabilitySchema
│   │   ├── router.py            CapabilityRouter
│   │   ├── providers/           openai.py · bedrock.py · ollama.py
│   │   └── config.py            AdapterConfig + ProviderConfig
│   ├── Dockerfile               two-stage Python image (python:3.12-slim)
│   └── agent-def.yaml.example
│
└── langgraph/                   ← feat(adapters/langgraph) #384  — Python module
    ├── pyproject.toml           langgraph as dep; uv
    ├── src/langgraph_adapter/
    │   ├── server.py            gRPC server
    │   ├── router.py            CapabilityRouter
    │   ├── graph_loader.py      GraphLoader — imports and compiles graphs at startup
    │   └── config.py            AdapterConfig + GraphMount
    ├── Dockerfile
    └── agent-def.yaml.example
```

### Extended paths

- `go.work` — each new Go adapter module added: `use agents/adapters/http`, `use agents/adapters/git`, `use agents/adapters/ci`
- `protos/tests/features/` — new BDD feature files (one per adapter, committed before implementation): `http_adapter.feature`, `git_adapter.feature`, `ci_adapter.feature`, `llm_adapter.feature`, `langgraph_adapter.feature`
- `infra/docker/docker-compose.yml` — each adapter service added to the local dev profile with its own service block and AgentDef YAML volume mount
- `docs/spdd/` — individual REASONS Canvas per adapter: `380-http-adapter/canvas.md`, `381-git-adapter/canvas.md`, `382-ci-adapter/canvas.md`, `383-llm-adapter/canvas.md`, `384-langgraph-adapter/canvas.md`

### Unchanged paths

- `protos/zynax/v1/` — no proto changes in M5. All adapter contracts are finalised.
- `services/` — platform services are unchanged by M5 adapter work.
- `agents/sdk/` — adapters do not import the SDK. SDK module development is tracked separately as a prerequisite for #376.

---

## O — Operations

Each step is a separate `feat:` PR with its own REASONS Canvas and BDD feature file committed before implementation.
Steps 1–3 (Go track) and steps 4–5 (Python track) are independent and can proceed in parallel.

1. **feat(adapters/http) #380** — Go module scaffold (`go.mod`, `main.go` with graceful shutdown, `internal/adapter.go` with `CapabilityRouter` + HTTP proxy `CapabilityHandler`, `internal/config.go` with `AdapterConfig`/`RouteConfig` YAML parsing); `ExecuteCapability` streaming loop (ticker PROGRESS for >2 s, COMPLETED/FAILED terminal); `input_payload` JSON Schema validation before execution (INVALID_INPUT on failure); `GetCapabilitySchema` returning schema from config; SSRF prevention (all URLs static config, never from payload); `DeregisterAgent` on shutdown; two-stage Alpine Dockerfile; docker-compose entry. BDD: `http_adapter.feature`. Establishes the Go adapter scaffold reused by steps 2 and 3.

2. **feat(adapters/git) #381** — Go module (same layout as http-adapter); `go-github` client with PAT/App auth from env-var ref (never from payload); `GitConfig`; capabilities: `open_pr` (create PR, validate branch exists before API call), `request_review` (request reviewers, poll for confirmation with PROGRESS), `get_diff` (fetch unified diff, truncate at 4 MB with `truncated: true` flag); rate-limit awareness (`RESOURCE_EXHAUSTED` on 429/403); GitLab config flag (stub only). BDD: `git_adapter.feature`.

3. **feat(adapters/ci) #382** — Go module; GitHub Actions REST API; `CIConfig`; capabilities: `trigger_workflow` (dispatch `workflow_dispatch`, poll up to 10 s for run ID to appear), `get_run_status` (`PollLoop` with exponential backoff 2 s→30 s, PROGRESS per cycle with run URL and status, TIMEOUT enforcement via ctx deadline); Jenkins config flag (stub only, returns `INTERNAL` with "not implemented" message). BDD: `ci_adapter.feature`.

4. **feat(adapters/llm) #383** — Python module; `AdapterConfig` + `ProviderConfig` parsed at startup; `chat_completion` capability; provider routing: `openai.AsyncOpenAI` for OpenAI, `aiobotocore` for Bedrock (required — boto3 sync is forbidden on the event loop), `httpx.AsyncClient` for Ollama REST; async token streaming → PROGRESS events; COMPLETED with full response; `asyncio.wait_for` for TIMEOUT enforcement; `pydantic.SecretStr` for key fields (never log, never include in CapabilityError); `bandit`+`pip-audit`+`mypy --strict` clean; `[[tool.mypy.overrides]] ignore_missing_imports = true` for untyped provider SDKs. BDD: `llm_adapter.feature`. Establishes the Python adapter scaffold reused by step 5.

5. **feat(adapters/langgraph) #384** — Python module; `GraphMount` config; `GraphLoader` imports and compiles LangGraph graphs at adapter startup (fail-fast if any graph fails to import); `LangGraphHandler` calls `compiled_graph.astream(input_state)` async; one PROGRESS event per `(node_name, state_update)` tuple; ticker PROGRESS if no node fires within 2 s; final graph state serialised with `json.dumps(..., default=str)` fallback; `asyncio.wait_for` for TIMEOUT; graph exceptions mapped to typed `CapabilityError` codes; ADR-015 scope enforced (LangGraph as capability, not engine — documented in Canvas and code comments). BDD: `langgraph_adapter.feature`.

---

## N — Norms

Pulled from root `AGENTS.md` §Hard Constraints, `agents/AGENTS.md` §Rules, `agents/adapters/AGENTS.md` §Rules, `docs/patterns/go-service-patterns.md`, and `docs/patterns/python-agent-guide.md`.

### Universal (all adapters)

- Commit hygiene: subject ≤ 72 chars, imperative mood, no period, no emojis. `Signed-off-by:` and `Assisted-by: Claude/claude-sonnet-4-6` required on every commit. Never `Co-Authored-By:` for AI.
- One PR per adapter issue. One logical commit per implementation step within that PR.
- BDD `.feature` file committed and CI-green **before** any implementation code (ADR-016).
- Individual REASONS Canvas at `docs/spdd/<issue>-<slug>/canvas.md` committed **before** implementation for each `feat:` adapter PR (ADR-019).
- Adapters are **stateless** — no adapter-local state that survives a process restart.
- Capability names: `snake_case`, 1–64 characters, matching the `AgentDef` YAML declaration exactly.
- At least one `TASK_EVENT_TYPE_PROGRESS` event emitted for tasks running >2 seconds (ticker-based if no natural progress checkpoint exists).
- Exactly one terminal event (`TASK_EVENT_TYPE_COMPLETED` or `TASK_EVENT_TYPE_FAILED`) per stream. No events after the terminal.
- `timeout_seconds` honoured: emit `TASK_EVENT_TYPE_FAILED` with code `"TIMEOUT"` on breach.
- `task_id` echoed in every `TaskEvent`. `timestamp` populated on every `TaskEvent`.
- `CapabilityError.message` is human-readable and sanitised — no raw API response bodies, no stack traces, no credential values.
- Health probe endpoint exposed (gRPC health protocol).
- Structured logs to stdout only (no file sinks). Never log credential values or `SecretStr` fields.
- `input_payload` validated against the capability's declared `input_schema` before execution; return `INVALID_INPUT` on validation failure.

### Go adapters (http, git, ci)

- `GOWORK=off` for all `go test`, `go build`, and `go mod` invocations in adapter directories (ADR-017).
- `CGO_ENABLED=0`, `-trimpath` on all production builds.
- Two-stage Alpine Dockerfile: `golang:*-alpine AS builder` → `alpine:*`. Final image runs as unprivileged `zynax` user.
- Go functions ≤ 30 lines. No `panic` in production. All errors wrapped with `%w`.
- Never import from another module's `internal/` package.
- `defer` to close HTTP response bodies, archive readers, and file handles.
- `context.Context` as first parameter on all functions crossing a process or I/O boundary.
- `govulncheck` and `gosec` clean in CI.

### Python adapters (llm, langgraph)

- Python 3.12. `uv` package manager. `pyproject.toml` as the single source of dependencies.
- Two-stage Python Dockerfile. Final image runs as unprivileged user.
- Platform calls via gRPC stubs only — never HTTP (root `AGENTS.md` §Hard Constraints).
- Never import `agents/sdk/` in an adapter. Adapters are self-contained.
- Python functions ≤ 20 lines. `mypy --strict` clean (with `ignore_missing_imports` overrides for untyped provider SDKs). `ruff` clean (including `ruff D` Google docstring convention).
- `bandit` + `pip-audit` clean in CI.
- Credential values (`SecretStr`) never logged, never included in `CapabilityError.message`.
- All I/O resources closed in `finally` blocks or `async with` context managers.
- `asyncio` throughout; no blocking calls on the event loop. Bedrock: `aiobotocore` (not `boto3`). Ollama: `httpx.AsyncClient`.
- `asyncio.wait_for` for timeout enforcement; catch `asyncio.TimeoutError` → emit FAILED with `"TIMEOUT"`.

---

## S — Safeguards

### Context Security (mandatory before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, or deployment specifics
- [x] No PII: no email addresses, no personal names in sensitive context
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards

- **Never** import `agents/sdk/` in an adapter — adapters implement `AgentService` directly via generated proto stubs (ADR-013).
- **Never** hardcode capability names in source — capabilities are always declared in the `AgentDef` YAML and read at startup.
- **Never** emit a `TaskEvent` after the terminal event (`TASK_EVENT_TYPE_COMPLETED` or `TASK_EVENT_TYPE_FAILED`).
- **Never** allow user-controlled URL fields in `input_payload` for the http-adapter — all HTTP routes are static config in `RouteConfig` (SSRF prevention; this is a hard security requirement, not a style choice).
- **Never** source credentials (PAT, API keys, tokens, model names) from `input_payload` — credentials are sourced from named environment variables declared in `AdapterConfig`; the value is never passed through user-controlled fields.
- **Never** log or include credential values in `TaskEvent` payloads, `CapabilityError.message`, or structured log fields. Python: use `pydantic.SecretStr` for credential fields.
- **Never** use Go `panic` in adapter code — return errors via gRPC status codes and `CapabilityError` (root `AGENTS.md` §Hard Constraints).
- **Never** discard `error` return values in Go adapters (`_ = f()` is forbidden).
- **Never** call Zynax platform services via HTTP in Python adapters — gRPC stubs only (root `AGENTS.md` §Hard Constraints).
- **Never** run `go test` or `go build` in adapter directories without `GOWORK=off` (ADR-017).
- **Never** store execution state across `ExecuteCapabilityRequest` invocations — adapters are stateless (ADR-013).
- **Never** extend proto contracts in M5 — all adapter contracts are finalised in `protos/zynax/v1/`.
- **Never** deploy adapters to Kubernetes in M5 — Docker Compose scope only; K8s is M6.
- **Never** use LangGraph as a Zynax workflow engine in M5 — the langgraph-adapter wraps LangGraph as a capability only (ADR-015; the engine boundary is a one-way door requiring a new ADR).
- **Never** call blocking I/O on the Python `asyncio` event loop — Bedrock via `aiobotocore`, Ollama via `httpx.AsyncClient`, all other I/O via `async with` or `await`.
- **Never** skip `input_payload` JSON Schema validation — validate before any system-facing call and return `INVALID_INPUT` on failure.
