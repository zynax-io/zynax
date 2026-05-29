<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — CI Adapter (CI Pipeline Trigger)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #382
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-08
**Status:** Implemented

---

## R — Requirements

- **Problem:** Workflow steps that need to trigger external CI pipelines — dispatching a GitHub Actions workflow run or querying its status — currently require each operator to write a bespoke gRPC adapter from scratch. There is no reusable, config-driven adapter that wraps the GitHub Actions REST API and surfaces these operations as Zynax capabilities. This gap blocks automation workflows that must gate on CI outcomes.

- **Missing capability:** A standalone `ci-adapter` Go service that wraps GitHub Actions pipeline operations — `trigger_workflow` and `get_run_status` — as gRPC-delivered Zynax capabilities. A `PollLoop` with exponential backoff streams live status as `TASK_EVENT_TYPE_PROGRESS` events. Jenkins is supported as a future config flag (stub only in this milestone).

- **Definition of done — observable outcomes:**
  - A workflow step calls `trigger_workflow`; a `workflow_dispatch` event is sent to the GitHub Actions API; the adapter polls for up to 10 s for the resulting run ID to appear; the task broker receives `TASK_EVENT_TYPE_COMPLETED` with the run ID and run URL in the payload.
  - A workflow step calls `get_run_status`; the `PollLoop` emits `TASK_EVENT_TYPE_PROGRESS` per cycle with the current run URL and status; `TASK_EVENT_TYPE_COMPLETED` is emitted when the run reaches a terminal GitHub state (`completed`, `failure`, `cancelled`, `skipped`, `timed_out`).
  - Exceeding `timeout_seconds` during either capability produces `TASK_EVENT_TYPE_FAILED` with `code = "TIMEOUT"`.
  - A 429 or 403 response from the GitHub Actions API produces `TASK_EVENT_TYPE_FAILED` with `CapabilityError.code = "RESOURCE_EXHAUSTED"` and a sanitised message (no raw API response, no token values).
  - `provider: jenkins` config returns `TASK_EVENT_TYPE_FAILED` with `code = "INTERNAL"` and "not implemented" message.
  - `GetCapabilitySchema` returns the JSON Schema declared in the `AgentDef` YAML for each capability.
  - `make test` green · `make lint` clean · `make security` clean.
  - BDD contract scenarios in `protos/tests/features/ci_adapter.feature` pass.

---

## E — Entities

### Existing entities consumed (no changes in #382)

- **`AgentService`** (`protos/zynax/v1/agent.proto`) — two-RPC contract implemented by the adapter: `ExecuteCapability` (server-streaming `TaskEvent`) and `GetCapabilitySchema`. Contract invariants: exactly one terminal event per stream; `task_id` echoed on every event; `timeout_seconds` honoured; no events after terminal.
- **`AgentRegistryService`** (`protos/zynax/v1/agent_registry.proto`) — `RegisterAgent` called at startup; `DeregisterAgent` called on graceful shutdown.
- **`AgentDef`** (proto message) — `agent_id`, `name`, `description`, `endpoint` (`host:port`), `capabilities[]`. Built from YAML config at startup and sent to the registry.
- **`CapabilityDef`** (proto message) — `name` (snake_case, 1–64 chars), `description`, `input_schema` (JSON Schema bytes), `output_schema` (JSON Schema bytes).
- **`ExecuteCapabilityRequest`** (proto message) — `request_id`, `capability_name`, `task_id`, `workflow_id`, `input_payload` (JSON bytes), `timeout_seconds`.
- **`TaskEvent`** (proto message) — `task_id`, `event_type` (PROGRESS / COMPLETED / FAILED), `payload`, `timestamp`, `error` (`CapabilityError`).
- **`CapabilityError`** (proto message) — `code`, `message`, `details`. Well-known codes: `"TIMEOUT"`, `"INVALID_INPUT"`, `"UPSTREAM_ERROR"`, `"RESOURCE_EXHAUSTED"`, `"INTERNAL"`.

### New entities (introduced by #382)

- **`AdapterConfig`** — top-level YAML struct parsed at startup. Fields: `agent_id`, `name`, `description`, `endpoint` (bind `host:port`), `registry_endpoint` (agent-registry `host:port`), `capabilities[]` (list of `CICapabilityConfig`). Never contains credential values — only env-var name references.
- **`CIConfig`** — per-adapter CI config: `provider` (`github-actions` or `jenkins-stub`); `token_env` (name of the env var holding the API token — never the value); `poll_interval_seconds` (initial poll interval, default 2); `max_poll_interval_seconds` (poll backoff ceiling, default 30); `trigger_poll_timeout_seconds` (max time to wait for a run ID to appear after dispatch, default 10).
- **`CICapabilityConfig`** — per-capability config embedded in `AdapterConfig.capabilities[]`: `name` (snake_case), `description`, `owner` (GitHub org or user — static config), `repo` (repository name — static config), `workflow_id` (GitHub Actions workflow file name or numeric ID — static config), `input_schema_json`, `output_schema_json`. Fields that vary per invocation (git ref, inputs map) come from `input_payload` only.
- **`CapabilityRouter`** — map of `capability_name → CICapabilityConfig` built from `AdapterConfig` at startup. Immutable after initialisation. Dispatches `ExecuteCapabilityRequest.capability_name` to the correct handler.
- **`CIHandler`** — executes one capability invocation via the GitHub Actions REST API. Two concrete operations: `triggerWorkflow` (dispatch `workflow_dispatch`, poll for run ID) and `getRunStatus` (delegate to `PollLoop`). Stateless; one instance shared across all requests. Reads `CIConfig` for auth and poll settings; reads `CICapabilityConfig` for target org/repo/workflow. Never sources auth tokens or run URLs from `input_payload`.
- **`PollLoop`** — exponential backoff polling loop for run status: initial interval from `CIConfig.poll_interval_seconds` (default 2 s), doubles each cycle, capped at `CIConfig.max_poll_interval_seconds` (default 30 s). Emits `TASK_EVENT_TYPE_PROGRESS` per cycle with the current run URL (from static config + run ID) and status. Respects `timeout_seconds` via `context.Context` deadline; emits `TASK_EVENT_TYPE_FAILED` with `"TIMEOUT"` if the deadline is exceeded before a terminal run state is reached.
- **`AgentServer`** — gRPC server struct implementing `AgentServiceServer`. Holds the `CapabilityRouter`; routes `ExecuteCapability` calls to `CIHandler`; serves `GetCapabilitySchema` from the router config.

### Entity relationships

```
Task Broker
    │ gRPC ExecuteCapabilityRequest
    ▼
AgentServer (AgentServiceServer)
    │
    ├── CapabilityRouter ──► CICapabilityConfig (one per declared capability)
    │                              │
    │                       CIHandler
    │                              │ GitHub Actions REST API (token from env var)
    │                              ▼
    │               trigger_workflow:
    │                   POST /repos/{owner}/{repo}/actions/workflows/{id}/dispatches
    │                   → poll (≤10s) for run ID → COMPLETED{run_id, run_url}
    │
    │               get_run_status:
    │                   PollLoop ──► GET /repos/{owner}/{repo}/actions/runs/{run_id}
    │                       │ per cycle: PROGRESS{run_url, status}
    │                       └─► terminal state → COMPLETED{conclusion}
    │                             or ctx deadline → FAILED TIMEOUT
    │
    │               429/403 → FAILED RESOURCE_EXHAUSTED
    │               jenkins provider → FAILED INTERNAL "not implemented"
    │
    └── stream TaskEvent{PROGRESS…, COMPLETED|FAILED}
            ▲ task_id echoed; timestamp populated on every event

At startup:
    AdapterConfig parsed from YAML (path from ADAPTER_CONFIG env var)
    CIConfig auth token resolved from named env var
    CapabilityRouter built (immutable)
    AgentServer.RegisterAgent(AgentDef) → AgentRegistryService

On graceful shutdown (SIGTERM/SIGINT):
    AgentServer.DeregisterAgent(agent_id) → AgentRegistryService
    grpcServer.GracefulStop()
```

---

## A — Approach

### What we WILL do

- Implement a standalone Go module at `agents/adapters/ci/` with its own `go.mod` (module path `github.com/zynax-io/zynax/agents/adapters/ci`).
- Use the GitHub Actions REST API directly via `net/http` (no third-party GitHub client required for the two endpoints needed). Auth token resolved at startup from the env-var name declared in `AdapterConfig`; never from `input_payload`.
- Parse `AdapterConfig` (including `CIConfig`) from a YAML file at startup (path from `ADAPTER_CONFIG` env var); fail fast if the file is missing, invalid, or if the declared auth env var is unset.
- Build `CapabilityRouter` from `AdapterConfig` at startup; treat it as immutable thereafter.
- Implement `ExecuteCapability` for two capabilities:
  - `trigger_workflow`: dispatch a `workflow_dispatch` event to GitHub Actions; poll the runs list endpoint for up to `CIConfig.trigger_poll_timeout_seconds` (default 10 s) for the new run ID to appear; emit `TASK_EVENT_TYPE_COMPLETED` with run ID and run URL once found, or `TASK_EVENT_TYPE_FAILED` with `"TIMEOUT"` if the run ID does not appear in time.
  - `get_run_status`: start `PollLoop`; emit `TASK_EVENT_TYPE_PROGRESS` per poll cycle with the run URL (constructed from static config fields + run ID from `input_payload`) and current status; emit `TASK_EVENT_TYPE_COMPLETED` with conclusion when GitHub reports a terminal state; emit `TASK_EVENT_TYPE_FAILED` with `"TIMEOUT"` if the context deadline is exceeded.
- Map GitHub API HTTP 429 and 403 responses to `CapabilityError.code = "RESOURCE_EXHAUSTED"`.
- Sanitise all `CapabilityError.message` values: no raw GitHub API response bodies, no token values, no stack traces. Truncate at 512 chars.
- Validate `input_payload` against the capability's `input_schema_json` before any API call; return `INVALID_INPUT` on validation failure.
- Implement `GetCapabilitySchema`: return schemas from `CICapabilityConfig`; return `NOT_FOUND` for unknown capabilities.
- Register with `AgentRegistryService.RegisterAgent` at startup (exponential-backoff retry, max 5 attempts); deregister on graceful shutdown.
- Add `use agents/adapters/ci` to `go.work`.
- Two-stage Alpine Dockerfile: `golang:1.26-alpine AS builder` → `alpine:latest`; final image runs as unprivileged `zynax` user.
- Expose gRPC health protocol endpoint.
- Add `ci-adapter` service to `infra/docker/docker-compose.yml`.
- Provide `agent-def.yaml.example` as operator documentation.

### What we WILL NOT do

- Source API token from `input_payload` — auth tokens are resolved from named env vars at startup only.
- Construct run URLs from `input_payload` fields — run URLs are built from static config (`owner`, `repo`) plus the run ID obtained from the GitHub API response; no `input_payload` string is interpolated into a URL.
- Include raw GitHub API response bodies in `CapabilityError.message` — sanitise before emitting.
- Implement Jenkins operations — Jenkins is a config flag stub only in this milestone; `provider: jenkins-stub` returns `INTERNAL` with "not implemented" message.
- Implement webhook ingestion (inbound CI events) — out of M5 scope.
- Implement retry logic — retry is owned by the task broker.
- Import `agents/sdk/` — the adapter implements `AgentService` directly via generated stubs (ADR-013).
- Store execution state between requests — stateless (ADR-013).
- Accept user-controlled `owner`, `repo`, or `workflow_id` from `input_payload` — these are always static config in `CICapabilityConfig`.

### Governing ADRs

- **ADR-001** — gRPC for all Zynax platform calls; no HTTP callbacks to the platform from the adapter.
- **ADR-005** — Apache 2.0 SPDX header on every source file.
- **ADR-006** — monorepo; module added to `go.work`.
- **ADR-009** — Go for stateless adapters; Python only for ML-ecosystem adapters.
- **ADR-013** — Adapter-first; never import `agents/sdk/`.
- **ADR-016** — BDD `.feature` file committed and CI-green before any implementation code.
- **ADR-017** — `GOWORK=off` for all `go test` / `go build` / `go mod` in adapter directories.
- **ADR-019** — REASONS Canvas committed and Aligned before implementation.

---

## S — Structure

### New paths

```
agents/adapters/ci/
├── go.mod                          module github.com/zynax-io/zynax/agents/adapters/ci
├── cmd/ci-adapter/
│   └── main.go                     gRPC server bootstrap; graceful shutdown; RegisterAgent/DeregisterAgent
├── internal/
│   ├── config/
│   │   └── config.go               AdapterConfig + CIConfig + CICapabilityConfig; YAML parsing; validation
│   ├── adapter/
│   │   ├── server.go               AgentServer (AgentServiceServer impl); CapabilityRouter
│   │   ├── handler.go              CIHandler (triggerWorkflow, getRunStatus); PollLoop
│   │   └── server_test.go          unit tests (table-driven, t.Run, mock HTTP server)
│   └── registry/
│       └── client.go               AgentRegistryService gRPC client; RegisterAgent with retry; DeregisterAgent
├── Dockerfile                      two-stage Alpine
└── agent-def.yaml.example          operator documentation
```

### Extended paths

- **`go.work`** — add `use agents/adapters/ci`
- **`protos/tests/features/ci_adapter.feature`** — BDD contract file (committed before implementation)
- **`infra/docker/docker-compose.yml`** — `ci-adapter` service block with config volume mount

### Unchanged paths

- `protos/zynax/v1/` — no proto changes in #382
- `services/` — platform services unchanged
- `agents/sdk/` — never imported by the adapter
- `agents/adapters/http/` — scaffold reference only; no modifications
- `agents/adapters/git/` — independent; no shared code or hard dependency

---

## O — Operations

This issue (#382) is a single `feat:` PR. Can proceed in parallel with #381 (git-adapter) — no shared code or hard dependency. Child issues track each step.

1. ✅ **BDD feature file** (#404) — commit `protos/tests/features/ci_adapter.feature` with adapter-specific scenarios: `trigger_workflow` dispatches `workflow_dispatch` → COMPLETED with run ID + run URL; trigger TIMEOUT when run ID doesn't appear within `trigger_poll_timeout_seconds`; `get_run_status` PROGRESS per poll cycle → COMPLETED on terminal state; `timeout_seconds` → TIMEOUT; 429/403 → RESOURCE_EXHAUSTED; `provider: jenkins-stub` → INTERNAL "not implemented"; run URL never from `input_payload`. CI must be green before any implementation (ADR-016).

2. ✅ **Module scaffold + config layer** (#405) — `go.mod`, `go.work` updated; `internal/config/config.go`: `AdapterConfig`, `CIConfig` (provider, token_env, poll intervals, trigger timeout), `CICapabilityConfig` (owner, repo, workflow_id); `Load()` validates required fields and applies defaults; `ResolveToken()` reads token from env var; 15 unit tests, 95.2% coverage.

3. **CIHandler + PollLoop** (#406) — `internal/adapter/server.go`: `AgentServer`, `CapabilityRouter`, `ExecuteCapability`, `GetCapabilitySchema`; `internal/adapter/handler.go`: `CIHandler` dispatching `triggerWorkflow` (dispatch + ≤10 s run-ID poll) and `getRunStatus` (`PollLoop`: 2 s→30 s backoff, PROGRESS per cycle, ctx deadline → TIMEOUT); 429/403 → RESOURCE_EXHAUSTED; Jenkins stub → INTERNAL; unit tests with `httptest.Server`.

4. **Registry client + bootstrap** (#407) — `internal/registry/client.go`: `RegisterAgent` (2 s backoff, ×2, max 5 attempts); `DeregisterAgent`; `cmd/ci-adapter/main.go` fully wired: config → auth token → router → registry → gRPC + health → SIGTERM → deregister + stop.

5. **Dockerfile + docker-compose** (#408) — two-stage Alpine Dockerfile (`CGO_ENABLED=0 -trimpath`; `USER zynax`); `ci-adapter` service block in docker-compose; `agent-def.yaml.example` documenting poll parameters.

---

## N — Norms

Pulled from root `AGENTS.md` §Hard Constraints, `agents/adapters/AGENTS.md` §Rules, and `docs/patterns/go-service-patterns.md`.

- Commit hygiene: subject ≤ 72 chars, imperative mood, no period, no emojis. `Signed-off-by:` and `Assisted-by: Claude/claude-sonnet-4-6` on every commit. Never `Co-Authored-By:` for AI.
- One PR for this issue. BDD feature file in its own first commit, CI-green before any implementation (ADR-016).
- REASONS Canvas committed and Aligned before any implementation code (ADR-019).
- SPDX header `// SPDX-License-Identifier: Apache-2.0` on every `.go` source file.
- `GOWORK=off` for all `go test`, `go build`, and `go mod` in `agents/adapters/ci/` (ADR-017).
- `CGO_ENABLED=0`, `-trimpath` on all production builds.
- Go functions ≤ 30 lines. No `panic` in production code. All errors wrapped with `%w`.
- Never discard `error` return values (`_ = f()` is forbidden).
- `context.Context` as first parameter on all functions crossing a process or I/O boundary. `PollLoop` derives its per-iteration context from the caller's context so that `timeout_seconds` is enforced without a separate timer.
- `defer` to close HTTP response bodies, file handles, gRPC connections.
- Structured logs to stdout only (`log/slog`). Never log credential values, auth tokens, raw API responses, or full `input_payload`.
- `input_payload` validated against `input_schema_json` before any API call; return `INVALID_INPUT` on validation failure.
- `PollLoop` initial interval: 2 s. Backoff multiplier: ×2 per cycle. Ceiling: 30 s. Emits `TASK_EVENT_TYPE_PROGRESS` per cycle.
- `trigger_workflow` polls for run ID for at most `CIConfig.trigger_poll_timeout_seconds` (default 10 s); separate from the overall `timeout_seconds` deadline.
- Exactly one terminal event (`TASK_EVENT_TYPE_COMPLETED` or `TASK_EVENT_TYPE_FAILED`) per stream. No events after terminal.
- `task_id` echoed on every `TaskEvent`. `timestamp` populated on every `TaskEvent`.
- `CapabilityError.message` sanitised: no raw GitHub API response bodies, no token values, no stack traces. Truncated at 512 chars.
- GitHub API 429/403 → `CapabilityError.code = "RESOURCE_EXHAUSTED"`.
- API response bodies read with `io.LimitReader` (e.g. 10 MB cap) to prevent memory exhaustion from large responses.
- `govulncheck` and `gosec` clean in CI.
- Two-stage Alpine Dockerfile; final image runs as unprivileged `zynax` user.
- Never import another module's `internal/` package.
- Adapters are stateless — no adapter-local state surviving a process restart (ADR-013).

---

## S — Safeguards

### Context Security (mandatory before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, or deployment specifics
- [x] No PII: no email addresses, no personal names in sensitive context
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] `/spdd-security-review` passed — result: PASS

### Feature Safeguards

- **Never** source API token from `input_payload` — auth tokens are resolved from the env-var name declared in `AdapterConfig` at startup only; the token value is never passed through user-controlled fields.
- **Never** construct run URLs from `input_payload` fields — run URLs are built from static config fields (`owner`, `repo`) plus the run ID obtained from the GitHub API response; no `input_payload` string is interpolated into any URL component.
- **Never** include raw GitHub API response bodies in `CapabilityError.message` — sanitise and truncate before emitting; no status codes, headers, or body text from the API response.
- **Never** accept user-controlled `owner`, `repo`, or `workflow_id` from `input_payload` — these are always static config in `CICapabilityConfig`.
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
- **Never** read GitHub API response bodies without `io.LimitReader` — cap at a safe maximum to prevent memory exhaustion from unexpected large responses.
- **Never** log auth token values, even partially — treat the resolved token as an opaque secret throughout the adapter lifecycle.
- **Never** implement Jenkins operations in this milestone — `provider: jenkins-stub` returns FAILED with `INTERNAL` and "not implemented" message; full Jenkins support is deferred to a future issue.
- **Never** block the `PollLoop` goroutine beyond the context deadline — all sleep intervals must use `time.After` or `time.NewTimer` selected against `ctx.Done()` so that SIGTERM causes clean shutdown without hanging poll cycles.
