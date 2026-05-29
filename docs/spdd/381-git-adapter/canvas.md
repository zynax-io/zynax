<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — Git Adapter (GitHub/GitLab Operations)

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #381
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-08
**Status:** Implemented

---

## R — Requirements

- **Problem:** Workflow steps that need to interact with a Git platform — opening a pull request, requesting reviewers, or fetching a diff — currently require each operator to write a bespoke gRPC adapter service. There is no reusable, config-driven adapter that speaks the GitHub API and surfaces these operations as Zynax capabilities. The resulting duplication blocks adoption for the common case of repo-automation workflows.

- **Missing capability:** A standalone `git-adapter` Go service that wraps GitHub repository operations — `open_pr`, `request_review`, and `get_diff` — as gRPC-delivered Zynax capabilities, with PAT or GitHub App authentication sourced from environment variables declared in the `AgentDef` YAML. GitLab is supported as a future config flag (stub only in this milestone).

- **Definition of done — observable outcomes:**
  - A workflow step calls `open_pr`; a pull request appears in the configured repository with the correct title, body, and head branch; the task broker receives `TASK_EVENT_TYPE_COMPLETED` with the PR URL in the payload.
  - A workflow step calls `request_review`; the designated reviewers are added to the PR; at least one `TASK_EVENT_TYPE_PROGRESS` event is emitted during the poll-for-confirmation phase before the terminal event.
  - A workflow step calls `get_diff`; the unified diff is returned as `TASK_EVENT_TYPE_COMPLETED` payload; diffs exceeding 4 MB are truncated with `truncated: true` set in the payload.
  - A 429 or 403 response from the GitHub API produces `TASK_EVENT_TYPE_FAILED` with `CapabilityError.code = "RESOURCE_EXHAUSTED"` and a sanitised message (no raw API response, no token values).
  - Exceeding `timeout_seconds` produces `TASK_EVENT_TYPE_FAILED` with `code = "TIMEOUT"`.
  - `GetCapabilitySchema` returns the JSON Schema declared in the `AgentDef` YAML for each capability.
  - `make test` green · `make lint` clean · `make security` clean.
  - BDD contract scenarios in `protos/tests/features/git_adapter.feature` pass.

---

## E — Entities

### Existing entities consumed (no changes in #381)

- **`AgentService`** (`protos/zynax/v1/agent.proto`) — two-RPC contract implemented by the adapter: `ExecuteCapability` (server-streaming `TaskEvent`) and `GetCapabilitySchema`. Contract invariants: exactly one terminal event per stream; `task_id` echoed on every event; `timeout_seconds` honoured; no events after terminal.
- **`AgentRegistryService`** (`protos/zynax/v1/agent_registry.proto`) — `RegisterAgent` called at startup; `DeregisterAgent` called on graceful shutdown.
- **`AgentDef`** (proto message) — `agent_id`, `name`, `description`, `endpoint` (`host:port`), `capabilities[]`. Built from YAML config at startup and sent to the registry.
- **`CapabilityDef`** (proto message) — `name` (snake_case, 1–64 chars), `description`, `input_schema` (JSON Schema bytes), `output_schema` (JSON Schema bytes).
- **`ExecuteCapabilityRequest`** (proto message) — `request_id`, `capability_name`, `task_id`, `workflow_id`, `input_payload` (JSON bytes), `timeout_seconds`.
- **`TaskEvent`** (proto message) — `task_id`, `event_type` (PROGRESS / COMPLETED / FAILED), `payload`, `timestamp`, `error` (`CapabilityError`).
- **`CapabilityError`** (proto message) — `code`, `message`, `details`. Well-known codes: `"TIMEOUT"`, `"INVALID_INPUT"`, `"UPSTREAM_ERROR"`, `"RESOURCE_EXHAUSTED"`, `"INTERNAL"`.

### New entities (introduced by #381)

- **`AdapterConfig`** — top-level YAML struct parsed at startup. Fields: `agent_id`, `name`, `description`, `endpoint` (bind `host:port`), `registry_endpoint` (agent-registry `host:port`), `capabilities[]` (list of `GitCapabilityConfig`). Never contains credential values — only env-var name references.
- **`GitConfig`** — per-adapter auth config: `auth_mode` (`pat` or `github-app`); `token_env` (name of the env var holding the PAT or App token — never the value); `provider` (`github` or `gitlab`). GitLab: config flag only (stub in this milestone).
- **`GitCapabilityConfig`** — per-capability config embedded in `AdapterConfig.capabilities[]`: `name` (snake_case), `description`, `owner` (GitHub org or user — static config), `repo` (repository name — static config), `input_schema_json`, `output_schema_json`. Fields that vary per invocation (branch names, PR title, reviewer list) come from `input_payload` only.
- **`CapabilityRouter`** — map of `capability_name → GitCapabilityConfig` built from `AdapterConfig` at startup. Immutable after initialisation. Dispatches `ExecuteCapabilityRequest.capability_name` to the correct handler.
- **`GitHandler`** — executes one capability invocation using the `go-github` client. Three concrete operations: `openPR`, `requestReview`, `getDiff`. Stateless; one instance shared across all requests. Reads `GitConfig` for auth; reads `GitCapabilityConfig` for target org/repo. Never sources auth tokens or target URLs from `input_payload`.
- **`ProgressTicker`** — goroutine that emits `TASK_EVENT_TYPE_PROGRESS` while a slow operation (GitHub API call or review-confirmation poll) is in flight. Stopped as soon as the handler returns a result or error.
- **`AgentServer`** — gRPC server struct implementing `AgentServiceServer`. Holds the `CapabilityRouter`; routes `ExecuteCapability` calls to `GitHandler`; serves `GetCapabilitySchema` from the router config.

### Entity relationships

```
Task Broker
    │ gRPC ExecuteCapabilityRequest
    ▼
AgentServer (AgentServiceServer)
    │
    ├── CapabilityRouter ──► GitCapabilityConfig (one per declared capability)
    │                              │
    │                       GitHandler
    │                              │ go-github client (PAT/App auth from env var)
    │                              ▼
    │                       GitHub API (REST v3)
    │                              │
    │               open_pr   → PR URL in COMPLETED payload
    │               request_review → PROGRESS per poll cycle → COMPLETED
    │               get_diff  → diff bytes (truncated at 4 MB) in COMPLETED payload
    │               429/403   → FAILED RESOURCE_EXHAUSTED
    │
    ├── ProgressTicker ──► PROGRESS event every 2s (goroutine, races API call)
    │
    └── stream TaskEvent{PROGRESS…, COMPLETED|FAILED}
            ▲ task_id echoed; timestamp populated on every event

At startup:
    AdapterConfig parsed from YAML (path from ADAPTER_CONFIG env var)
    GitConfig auth token resolved from named env var
    CapabilityRouter built (immutable)
    AgentServer.RegisterAgent(AgentDef) → AgentRegistryService

On graceful shutdown (SIGTERM/SIGINT):
    AgentServer.DeregisterAgent(agent_id) → AgentRegistryService
    grpcServer.GracefulStop()
```

---

## A — Approach

### What we WILL do

- Implement a standalone Go module at `agents/adapters/git/` with its own `go.mod` (module path `github.com/zynax-io/zynax/agents/adapters/git`).
- Use the `go-github` library (GitHub API v3) for all GitHub operations. Auth token resolved at startup from the env-var name declared in `AdapterConfig`; never from `input_payload`.
- Parse `AdapterConfig` (including `GitConfig`) from a YAML file at startup (path from `ADAPTER_CONFIG` env var); fail fast if the file is missing, invalid, or if the declared auth env var is unset.
- Build `CapabilityRouter` from `AdapterConfig` at startup; treat it as immutable thereafter.
- Implement `ExecuteCapability` for three capabilities:
  - `open_pr`: validate that the head branch exists before calling the API; create the PR; return the PR URL in `TASK_EVENT_TYPE_COMPLETED` payload.
  - `request_review`: add reviewers to an existing PR; poll the GitHub API for review-request confirmation with `TASK_EVENT_TYPE_PROGRESS` per poll cycle; emit `TASK_EVENT_TYPE_COMPLETED` once confirmed or after all poll attempts.
  - `get_diff`: fetch the unified diff for the specified PR or commit range; truncate at 4 MB and set `truncated: true` in the payload if the diff exceeds the limit; emit `TASK_EVENT_TYPE_COMPLETED`.
- Map GitHub API HTTP 429 and 403 responses to `CapabilityError.code = "RESOURCE_EXHAUSTED"`.
- Sanitise all `CapabilityError.message` values: no raw GitHub API response bodies, no token values, no stack traces. Truncate at 512 chars.
- Validate `input_payload` against the capability's `input_schema_json` before any API call; return `INVALID_INPUT` on validation failure.
- Implement `GetCapabilitySchema`: return schemas from `GitCapabilityConfig`; return `NOT_FOUND` for unknown capabilities.
- Register with `AgentRegistryService.RegisterAgent` at startup (exponential-backoff retry, max 5 attempts); deregister on graceful shutdown.
- Add `use agents/adapters/git` to `go.work`.
- Two-stage Alpine Dockerfile: `golang:1.26-alpine AS builder` → `alpine:latest`; final image runs as unprivileged `zynax` user.
- Expose gRPC health protocol endpoint.
- Add `git-adapter` service to `infra/docker/docker-compose.yml`.
- Provide `agent-def.yaml.example` as operator documentation.

### What we WILL NOT do

- Source PAT or App credentials from `input_payload` — auth tokens are resolved from named env vars at startup only.
- Include raw GitHub API response bodies in `CapabilityError.message` — sanitise before emitting.
- Implement GitLab operations — GitLab is a config flag stub only in this milestone; returns `INTERNAL` with "not implemented" message if `provider: gitlab` is set.
- Implement webhook ingestion (inbound events) — out of M5 scope.
- Implement retry logic — retry is owned by the task broker.
- Handle OAuth flows — PAT and GitHub App auth only.
- Import `agents/sdk/` — the adapter implements `AgentService` directly via generated stubs (ADR-013).
- Store execution state between requests — stateless (ADR-013).
- Accept user-controlled `owner` or `repo` fields from `input_payload` — these are always static config in `GitCapabilityConfig`.

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
agents/adapters/git/
├── go.mod                          module github.com/zynax-io/zynax/agents/adapters/git
├── cmd/git-adapter/
│   └── main.go                     gRPC server bootstrap; graceful shutdown; RegisterAgent/DeregisterAgent
├── internal/
│   ├── config/
│   │   └── config.go               AdapterConfig + GitConfig + GitCapabilityConfig; YAML parsing; validation
│   ├── adapter/
│   │   ├── server.go               AgentServer (AgentServiceServer impl); CapabilityRouter
│   │   ├── handler.go              GitHandler (openPR, requestReview, getDiff); ProgressTicker
│   │   └── server_test.go          unit tests (table-driven, t.Run, mock HTTP server)
│   └── registry/
│       └── client.go               AgentRegistryService gRPC client; RegisterAgent with retry; DeregisterAgent
├── Dockerfile                      two-stage Alpine
└── agent-def.yaml.example          operator documentation
```

### Extended paths

- **`go.work`** — add `use agents/adapters/git`
- **`protos/tests/features/git_adapter.feature`** — BDD contract file (committed before implementation)
- **`infra/docker/docker-compose.yml`** — `git-adapter` service block with config volume mount

### Unchanged paths

- `protos/zynax/v1/` — no proto changes in #381
- `services/` — platform services unchanged
- `agents/sdk/` — never imported by the adapter
- `agents/adapters/http/` — scaffold reference only; no modifications

---

## O — Operations

This issue (#381) is a single `feat:` PR. Implementation is broken into logical commits within that PR, following the same pattern established by the http-adapter (#380). Child issues track each step.

1. **BDD feature file** (#399) — commit `protos/tests/features/git_adapter.feature` with adapter-specific scenarios: `open_pr` creates a PR and returns the URL in COMPLETED payload; `request_review` emits PROGRESS per poll cycle and COMPLETED on confirmation; `get_diff` returns diff bytes and sets `truncated: true` when diff exceeds 4 MB; 429/403 from GitHub API produces FAILED with `RESOURCE_EXHAUSTED`; `timeout_seconds` breach produces FAILED with `TIMEOUT`; unknown capability returns `NOT_FOUND`; `gitlab` provider flag returns FAILED with `INTERNAL` and "not implemented"; credentials never appear in `CapabilityError.message`. CI must be green before any implementation code is committed (ADR-016).

2. **Module scaffold + config layer** (#400) — `go.mod` (Go 1.26.3, module path, `replace` directive for generated stubs, `go-github` dependency), `go.work` updated with `use agents/adapters/git`, `cmd/git-adapter/main.go` skeleton (compile-only); `internal/config/config.go`: `AdapterConfig`, `GitConfig`, and `GitCapabilityConfig` structs with YAML tags; `Load(path string)` validating required fields and auth env-var presence; unit tests.

3. **Capability handler** (#401) — `internal/adapter/server.go`: `AgentServer` struct; `CapabilityRouter`; `ExecuteCapability`. `internal/adapter/handler.go`: `GitHandler` with `openPR` (branch existence check), `requestReview` (poll PROGRESS), `getDiff` (4 MB `io.LimitReader`, `truncated: true`); `ProgressTicker`; 429/403 → `RESOURCE_EXHAUSTED`; `GetCapabilitySchema`. Unit tests with mock `httptest.Server`.

4. **Registry client + bootstrap** (#402) — `internal/registry/client.go`: `RegisterAgent` (2 s backoff, ×2, max 5 attempts); `DeregisterAgent`; `cmd/git-adapter/main.go` fully wired: config → auth token → router → registry → gRPC server + health → SIGTERM → deregister + stop.

5. **Dockerfile + docker-compose** (#403) — two-stage Alpine Dockerfile (`CGO_ENABLED=0 -trimpath`; `USER zynax`); `infra/docker/docker-compose.yml` `git-adapter` service block; `agent-def.yaml.example`.

---

## N — Norms

Pulled from root `AGENTS.md` §Hard Constraints, `agents/adapters/AGENTS.md` §Rules, and `docs/patterns/go-service-patterns.md`.

- Commit hygiene: subject ≤ 72 chars, imperative mood, no period, no emojis. `Signed-off-by:` and `Assisted-by: Claude/claude-sonnet-4-6` on every commit. Never `Co-Authored-By:` for AI.
- One PR for this issue. BDD feature file in its own first commit, CI-green before any implementation (ADR-016).
- REASONS Canvas committed and Aligned before any implementation code (ADR-019).
- SPDX header `// SPDX-License-Identifier: Apache-2.0` on every `.go` source file.
- `GOWORK=off` for all `go test`, `go build`, and `go mod` in `agents/adapters/git/` (ADR-017).
- `CGO_ENABLED=0`, `-trimpath` on all production builds.
- Go functions ≤ 30 lines. No `panic` in production code. All errors wrapped with `%w`.
- Never discard `error` return values (`_ = f()` is forbidden).
- `context.Context` as first parameter on all functions crossing a process or I/O boundary.
- `defer` to close HTTP response bodies, file handles, gRPC connections.
- Structured logs to stdout only (`log/slog`). Never log credential values, auth tokens, raw API responses, or full `input_payload`.
- `input_payload` validated against `input_schema_json` before any API call; return `INVALID_INPUT` on validation failure.
- At least one `TASK_EVENT_TYPE_PROGRESS` event for operations running >2 s (ticker-based for `open_pr`/`get_diff`; poll-cycle events for `request_review`).
- Exactly one terminal event (`TASK_EVENT_TYPE_COMPLETED` or `TASK_EVENT_TYPE_FAILED`) per stream. No events after terminal.
- `task_id` echoed on every `TaskEvent`. `timestamp` populated on every `TaskEvent`.
- `CapabilityError.message` sanitised: no raw GitHub API response bodies, no token values, no stack traces. Truncated at 512 chars.
- GitHub API 429/403 → `CapabilityError.code = "RESOURCE_EXHAUSTED"`.
- Diff response body read with `io.LimitReader` at 4 MB + 1 byte to detect truncation; set `truncated: true` in payload if limit is reached.
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

- **Never** source PAT or GitHub App credentials from `input_payload` — auth tokens are resolved from the env-var name declared in `AdapterConfig` at startup only; the token value is never passed through user-controlled fields.
- **Never** include raw GitHub API response bodies in `CapabilityError.message` — sanitise and truncate before emitting; no status codes, headers, or body text from the API response.
- **Never** accept user-controlled `owner` or `repo` fields from `input_payload` — target org and repository are always static config in `GitCapabilityConfig`.
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
- **Never** read the GitHub API diff response body without `io.LimitReader` — cap at 4 MB + 1 byte to detect truncation and prevent memory exhaustion.
- **Never** log auth token values, even partially — treat the resolved token as an opaque secret throughout the adapter lifecycle.
- **Never** implement GitLab operations in this milestone — `provider: gitlab` returns FAILED with `INTERNAL` and "not implemented" message; full GitLab support is deferred to a future issue.
