<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — HTTP Adapter (REST Capability Proxy)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #380
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-08
**Status:** Draft

---

## R — Requirements

- **Problem:** There is no reusable way to expose an existing REST API as a Zynax capability. Every operator who wants to call an HTTP endpoint must write a bespoke gRPC adapter service from scratch — with proto stubs, a streaming event loop, YAML config, and a Dockerfile — even when the target service is a plain REST API. This is the single largest barrier to onboarding existing infrastructure.

- **Missing capability:** A config-driven HTTP adapter that turns any REST API into a Zynax capability by declaring routes in an `AgentDef` YAML file. Zero code changes to the target service; zero new gRPC code by the operator.

- **Definition of done — observable outcomes:**
  - `zynax apply agent-def.yaml` registers the http-adapter; a workflow step calls the declared capability; the task broker receives a `TASK_EVENT_TYPE_COMPLETED` `TaskEvent` with the proxied HTTP response payload.
  - A 2xx HTTP response from the upstream produces `TASK_EVENT_TYPE_COMPLETED` with the response body as `payload`.
  - A 4xx/5xx HTTP response produces `TASK_EVENT_TYPE_FAILED` with `CapabilityError.code = "UPSTREAM_ERROR"` and a sanitised message (no raw response body, no credential values).
  - A request taking >2 s emits at least one `TASK_EVENT_TYPE_PROGRESS` event before the terminal event.
  - Exceeding `timeout_seconds` emits `TASK_EVENT_TYPE_FAILED` with `code = "TIMEOUT"`.
  - `GetCapabilitySchema` returns the JSON Schema declared in the `AgentDef` YAML for any registered capability.
  - `make test` green · `make lint` clean · `make security` clean.
  - BDD contract scenarios in `protos/tests/features/http_adapter.feature` pass.

---

## E — Entities

### Existing entities consumed (no changes in #380)

- **`AgentService`** (`protos/zynax/v1/agent.proto`) — two-RPC contract implemented by the adapter: `ExecuteCapability` (server-streaming `TaskEvent`) and `GetCapabilitySchema`. Contract invariants: exactly one terminal event per stream; `task_id` echoed on every event; `timeout_seconds` honoured; no events after terminal.
- **`AgentRegistryService`** (`protos/zynax/v1/agent_registry.proto`) — `RegisterAgent` called at startup; `DeregisterAgent` called on graceful shutdown.
- **`AgentDef`** (proto message) — `agent_id`, `name`, `description`, `endpoint` (`host:port`), `capabilities[]`. Built from YAML config at startup and sent to the registry.
- **`CapabilityDef`** (proto message) — `name` (snake_case, 1–64 chars), `description`, `input_schema` (JSON Schema bytes), `output_schema` (JSON Schema bytes).
- **`ExecuteCapabilityRequest`** (proto message) — `request_id`, `capability_name`, `task_id`, `workflow_id`, `input_payload` (JSON bytes), `timeout_seconds`.
- **`TaskEvent`** (proto message) — `task_id`, `event_type` (PROGRESS / COMPLETED / FAILED), `payload`, `timestamp`, `error` (`CapabilityError`).
- **`CapabilityError`** (proto message) — `code`, `message`, `details`. Well-known codes: `"TIMEOUT"`, `"INVALID_INPUT"`, `"UPSTREAM_ERROR"`, `"RESOURCE_EXHAUSTED"`, `"INTERNAL"`.

### New entities (introduced by #380)

- **`AdapterConfig`** — top-level YAML struct parsed at startup. Fields: `agent_id`, `name`, `description`, `endpoint` (bind `host:port`), `registry_endpoint` (agent-registry `host:port`), `capabilities[]` (list of `RouteConfig`). Never contains credential values — only env-var name references for auth headers.
- **`RouteConfig`** — per-capability config: `name` (snake_case), `method` (HTTP verb), `url` (static — never derived from `input_payload`), `headers` (static key-value map; values may reference env-var names), `timeout_seconds` (per-route override), `input_schema_json`, `output_schema_json`, `description`.
- **`CapabilityRouter`** — map of `capability_name → RouteConfig` built from `AdapterConfig` at startup. Immutable after initialisation. Dispatches `ExecuteCapabilityRequest.capability_name` to the correct `RouteConfig`.
- **`HTTPHandler`** — executes one capability invocation: builds an `http.Request` from `RouteConfig` + `input_payload`, calls the upstream, maps response codes to `TaskEvent` types. Stateless; one instance shared across all requests.
- **`ProgressTicker`** — goroutine racing the HTTP call; emits `TASK_EVENT_TYPE_PROGRESS` every 2 s while the upstream call is in flight. Stopped as soon as the HTTP response (or error) is received.
- **`AgentServer`** — gRPC server struct implementing `AgentServiceServer`. Holds the `CapabilityRouter`; routes `ExecuteCapability` calls to `HTTPHandler`; serves `GetCapabilitySchema` from the router config.

### Entity relationships

```
Task Broker
    │ gRPC ExecuteCapabilityRequest
    ▼
AgentServer (AgentServiceServer)
    │
    ├── CapabilityRouter ──► RouteConfig (one per declared capability)
    │                              │
    │                       HTTPHandler
    │                              │ http.NewRequestWithContext(ctx, method, url, body)
    │                              ▼
    │                       Upstream REST API
    │                              │ HTTP response
    │                       2xx → COMPLETED payload
    │                       4xx/5xx → FAILED UPSTREAM_ERROR
    │
    ├── ProgressTicker ──► PROGRESS event every 2s (goroutine, races HTTP call)
    │
    └── stream TaskEvent{PROGRESS…, COMPLETED|FAILED}
            ▲ task_id echoed; timestamp populated on every event

At startup:
    AdapterConfig parsed from YAML
    CapabilityRouter built (immutable)
    AgentServer.RegisterAgent(AgentDef) → AgentRegistryService

On graceful shutdown (SIGTERM/SIGINT):
    AgentServer.DeregisterAgent(agent_id) → AgentRegistryService
    grpcServer.GracefulStop()
```

---

## A — Approach

### What we WILL do

- Implement a standalone Go module at `agents/adapters/http/` with its own `go.mod` (module path `github.com/zynax-io/zynax/agents/adapters/http`).
- Parse `AdapterConfig` from a YAML file at startup (path from `ADAPTER_CONFIG` env var); fail fast if the file is missing or invalid.
- Build `CapabilityRouter` from `AdapterConfig` at startup; treat it as immutable thereafter.
- Implement `ExecuteCapability`: validate `capability_name` and `task_id` first; validate `input_payload` against the capability's `input_schema_json`; call the upstream via `http.NewRequestWithContext` with a context derived from `timeout_seconds`; race the HTTP call against a `ProgressTicker` goroutine; map response codes to `TaskEvent` types.
- Forward `input_payload` as the HTTP request body (for POST/PUT/PATCH) or as query parameters (for GET/DELETE), never as URL path components.
- Implement `GetCapabilitySchema`: return `input_schema_json` / `output_schema_json` from `RouteConfig`; return `NOT_FOUND` for unknown capabilities.
- Register with `AgentRegistryService.RegisterAgent` at startup (with exponential-backoff retry, max 5 attempts); deregister on graceful shutdown.
- Two-stage Alpine Dockerfile: `golang:1.26-alpine AS builder` → `alpine:latest`; final image runs as unprivileged `zynax` user.
- Add `use agents/adapters/http` to `go.work`.
- Expose gRPC health protocol endpoint.
- Add `http-adapter` service to `infra/docker/docker-compose.yml`.
- Provide `agent-def.yaml.example` as operator documentation.

### What we WILL NOT do

- Accept user-controlled URL fields in `input_payload` — all HTTP routes are static config in `RouteConfig` (SSRF prevention; hard security requirement).
- Source credentials from `input_payload` — API key headers are declared as static values or env-var references in `AdapterConfig`.
- Implement retry logic — retry is owned by the task broker.
- Implement response body caching.
- Handle authentication flows beyond static API-key headers in config (OAuth, mTLS are out of scope).
- Import `agents/sdk/` — the adapter implements `AgentService` directly via generated stubs (ADR-013).
- Store execution state between requests — stateless (ADR-013).

### Governing ADRs

- **ADR-001** — gRPC for all Zynax platform calls; no HTTP callbacks to the platform from the adapter.
- **ADR-005** — Apache 2.0 SPDX header on every source file.
- **ADR-006** — monorepo; module added to `go.work`.
- **ADR-009** — Go for stateless HTTP proxy adapters; Python only for ML-ecosystem adapters.
- **ADR-013** — Adapter-first; never import `agents/sdk/`.
- **ADR-016** — BDD `.feature` file committed and CI-green before any implementation code.
- **ADR-017** — `GOWORK=off` for all `go test` / `go build` / `go mod` in adapter directories.
- **ADR-019** — REASONS Canvas committed and Aligned before implementation.

---

## S — Structure

### New paths

```
agents/adapters/http/
├── go.mod                          module github.com/zynax-io/zynax/agents/adapters/http
├── cmd/http-adapter/
│   └── main.go                     gRPC server bootstrap; graceful shutdown; RegisterAgent/DeregisterAgent
├── internal/
│   ├── config/
│   │   └── config.go               AdapterConfig + RouteConfig; YAML parsing; validation
│   ├── adapter/
│   │   ├── server.go               AgentServer (AgentServiceServer impl); CapabilityRouter
│   │   ├── handler.go              HTTPHandler; ProgressTicker
│   │   └── server_test.go          unit tests (table-driven, t.Run)
│   └── registry/
│       └── client.go               AgentRegistryService gRPC client; RegisterAgent with retry; DeregisterAgent
├── Dockerfile                      two-stage Alpine
└── agent-def.yaml.example          operator documentation
```

### Extended paths

- **`go.work`** — add `use agents/adapters/http`
- **`protos/tests/features/http_adapter.feature`** — BDD contract file (committed before implementation)
- **`infra/docker/docker-compose.yml`** — `http-adapter` service block with config volume mount

### Unchanged paths

- `protos/zynax/v1/` — no proto changes in #380
- `services/` — platform services unchanged
- `agents/sdk/` — never imported by the adapter

---

## O — Operations

This issue is a single `feat:` PR. Implementation is broken into logical commits within that PR.

1. **BDD feature file** — commit `protos/tests/features/http_adapter.feature` with adapter-specific scenarios: SSRF prevention (URL never from payload), static-header forwarding, 2xx→COMPLETED, 4xx/5xx→FAILED with `UPSTREAM_ERROR`, PROGRESS ticker fires for slow upstreams, `timeout_seconds` respected, `GetCapabilitySchema` returns route config schema, unknown capability returns `NOT_FOUND`, empty `capability_name` returns `INVALID_ARGUMENT`. CI must be green before any implementation code is committed.

2. **Module scaffold** — `go.mod` (Go 1.26.3, module path, `replace` directive for generated stubs), `go.work` updated with `use agents/adapters/http`, `cmd/http-adapter/main.go` skeleton (compile-only, no logic yet).

3. **Config layer** — `internal/config/config.go`: `AdapterConfig` struct with YAML tags; `RouteConfig` struct; `Load(path string)` function validating required fields (`agent_id`, `endpoint`, `registry_endpoint`, at least one capability with non-empty `name`, `method`, `url`); unit tests.

4. **HTTP handler + router** — `internal/adapter/server.go`: `AgentServer` struct; `CapabilityRouter` (map built from `AdapterConfig`); `ExecuteCapability` implementation: validate inputs → lookup route → call `HTTPHandler` → stream events. `internal/adapter/handler.go`: `HTTPHandler.Execute(ctx, route, payload)`: build request with `http.NewRequestWithContext`; race HTTP call against `ProgressTicker`; map response → `TaskEvent`; sanitise error messages. `GetCapabilitySchema` from router. Unit tests with `httptest.Server`.

5. **Registry client** — `internal/registry/client.go`: `RegisterAgent` with exponential backoff (2 s base, ×2, max 5 attempts); `DeregisterAgent`; both use `context.Context` from caller.

6. **Bootstrap** — `cmd/http-adapter/main.go`: load config; build `CapabilityRouter`; dial registry; `RegisterAgent`; start gRPC server (health protocol); `signal.NotifyContext` for SIGTERM/SIGINT; `DeregisterAgent` + `GracefulStop` on shutdown.

7. **Dockerfile + docker-compose** — two-stage Alpine Dockerfile; `CGO_ENABLED=0 -trimpath`; `USER zynax`; `infra/docker/docker-compose.yml` `http-adapter` service block; `agent-def.yaml.example`.

---

## N — Norms

Pulled from root `AGENTS.md` §Hard Constraints, `agents/adapters/AGENTS.md` §Rules, and `docs/patterns/go-service-patterns.md`.

- Commit hygiene: subject ≤ 72 chars, imperative mood, no period, no emojis. `Signed-off-by:` and `Assisted-by: Claude/claude-sonnet-4-6` on every commit. Never `Co-Authored-By:` for AI.
- One PR for this issue. BDD feature file in its own first commit.
- SPDX header `// SPDX-License-Identifier: Apache-2.0` on every `.go` source file.
- `GOWORK=off` for all `go test`, `go build`, and `go mod` in `agents/adapters/http/` (ADR-017).
- `CGO_ENABLED=0`, `-trimpath` on all production builds.
- Go functions ≤ 30 lines. No `panic` in production code. All errors wrapped with `%w`.
- Never discard `error` return values (`_ = f()` is forbidden).
- `context.Context` as first parameter on all functions crossing a process or I/O boundary.
- `defer` to close HTTP response bodies, file handles, gRPC connections.
- Structured logs to stdout only (`log/slog`). Never log credential values, raw API responses, or full `input_payload`.
- `input_payload` validated against `input_schema_json` before any HTTP call; return `INVALID_INPUT` on validation failure.
- At least one `TASK_EVENT_TYPE_PROGRESS` event for requests running >2 s.
- Exactly one terminal event (`TASK_EVENT_TYPE_COMPLETED` or `TASK_EVENT_TYPE_FAILED`) per stream. No events after terminal.
- `task_id` echoed on every `TaskEvent`. `timestamp` populated on every `TaskEvent`.
- `CapabilityError.message` sanitised: no raw API response bodies, no stack traces, no credential values. Truncated at 512 chars.
- `govulncheck` and `gosec` clean in CI.
- Two-stage Alpine Dockerfile; final image runs as unprivileged `zynax` user.
- Never import another module's `internal/` package.

---

## S — Safeguards

### Context Security (mandatory before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no email addresses, no personal names in sensitive context
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards

- **Never** accept user-controlled URL fields in `input_payload` — all HTTP routes are static config in `RouteConfig.url` (SSRF prevention; hard security requirement, not a style choice).
- **Never** source credentials from `input_payload` — API key headers are static config or env-var references in `AdapterConfig`; the value is never passed through user-controlled fields.
- **Never** log or include credential values or raw API response bodies in `TaskEvent` payloads, `CapabilityError.message`, or structured log fields.
- **Never** emit a `TaskEvent` after the terminal event (`TASK_EVENT_TYPE_COMPLETED` or `TASK_EVENT_TYPE_FAILED`).
- **Never** import `agents/sdk/` — the adapter implements `AgentService` directly via generated proto stubs (ADR-013).
- **Never** store execution state across `ExecuteCapabilityRequest` invocations — the adapter is stateless (ADR-013).
- **Never** call Zynax platform services via HTTP — gRPC stubs only (ADR-001).
- **Never** run `go test` or `go build` without `GOWORK=off` in this module (ADR-017).
- **Never** use `panic` in production code — return errors via gRPC status codes and `CapabilityError`.
- **Never** discard `error` return values.
- **Never** extend proto contracts in this issue — all adapter contracts are finalised in `protos/zynax/v1/`.
- **Never** commit implementation code before the BDD `.feature` file is committed and CI-green (ADR-016).
- **Never** commit implementation code before this Canvas is Aligned (ADR-019).
- **Never** read the HTTP response body without `io.LimitReader` — cap at a safe maximum (e.g. 10 MB) to prevent memory exhaustion from large upstream responses.
