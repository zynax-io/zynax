<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M4 YAML System + CLI

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #314
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-05
**Status:** Aligned

---

## R — Requirements

- **Problem:** M3 delivers a working Temporal execution engine, but there is no user-facing entry point. A developer cannot run `zynax apply workflow.yaml` — the `api-gateway` service has only stub BDD scenarios and no Go implementation, and no CLI binary exists. The three-layer architecture is complete internally but inaccessible from outside the platform.
- **Missing capability:** (1) The `api-gateway` has no HTTP handlers, no kind-routing logic, and no gRPC client wiring to the compiler, engine-adapter, or registry. (2) There is no `zynax` CLI binary. (3) There is no local Docker Compose stack for end-to-end development without Kubernetes. (4) `kind: AgentDef` manifests cannot be submitted to the registry via the public API. (5) Manifest JSON schema validation, REASONS Canvas structural checking, and JSON Schema meta-validation all run as Python-only scripts in `tools/`, creating a Python runtime dependency in CI and preventing distribution as part of the `zynax` binary.
- **M4 adds:** A fully implemented `api-gateway` with a `/api/v1/apply` endpoint that routes YAML manifests by `kind`, a `zynax` CLI with `apply`, `get`, `delete`, `status`, `logs`, `validate manifest`, and `gitops watch` commands, a `zynax-ci` CI toolchain binary that replaces all Python validation scripts in `tools/` and `count-ai-context.sh`, and a local Docker Compose runner.
- **Definition of done — observable outcomes:**
  - `zynax apply workflow.yaml` compiles + submits a workflow and prints a `run_id`.
  - `zynax apply --dry-run workflow.yaml` returns compiler errors with line numbers and exits non-zero.
  - `zynax logs workflow <run_id>` streams lifecycle CloudEvents until the workflow reaches a terminal state.
  - `zynax get workflow <run_id>` and `zynax status workflow <run_id>` return current state and status.
  - `POST /api/v1/apply` with `kind: AgentDef` registers an agent and returns `agent_id`.
  - `make run-local` starts all services; `zynax apply spec/workflows/examples/code-review.yaml` succeeds end-to-end against the local stack.
  - `zynax gitops watch <dir>` re-applies changed YAML files automatically.
  - `zynax validate manifest <file>` validates a single YAML manifest against its JSON schema; exits non-zero with structured errors on failure.
  - `zynax-ci validate canvas <dir>` validates all REASONS Canvas files under a directory; checks seven sections, `**Issue:**`/`**Author:**`/`**Date:**`/`**Status:**` header fields, Status value one of (Draft/Aligned/Implemented/Synced), and security checklist marker presence; exits non-zero on any failure.
  - `zynax-ci validate schema <file>` validates a JSON Schema document for well-formed JSON and correct `$schema` field; exits non-zero if invalid.
  - `zynax-ci validate workflows <dir>`, `validate agent-defs <dir>`, `validate capabilities <dir>`, `validate policies <dir>` batch-validate YAML manifests of each kind; replace the corresponding Python tools.
  - `zynax-ci check ai-context` reports line counts for all AI context files (CLAUDE.md, all AGENTS.md, `docs/ai-assistant-setup.md`) against per-file thresholds; exits 0 (advisory). Replaces `tools/count-ai-context.sh`.
  - `make validate-spec` and `make validate-canvas` invoke `zynax-ci` instead of `python3 tools/*.py`; all Python validation scripts and `count-ai-context.sh` removed from `tools/`.
  - All BDD scenarios in `services/api-gateway/tests/features/` pass; `make test` green; `make lint` clean.

---

## E — Entities

### Existing entities (extended by M4)

- **`WorkflowCompilerService`** (`protos/zynax/v1/workflow_compiler.proto`) — `CompileWorkflow`, `ValidateManifest`, `GetCompiledWorkflow`. The `api-gateway` calls `CompileWorkflow` on every `kind: Workflow` apply. Contract unchanged in M4.
- **`EngineAdapterService`** (`protos/zynax/v1/engine_adapter.proto`) — `SubmitWorkflow`, `SignalWorkflow`, `CancelWorkflow`, `GetWorkflowStatus`, `WatchWorkflow`. The `api-gateway` calls `SubmitWorkflow` after a successful compile; `WatchWorkflow` backs the `zynax logs` command. Contract unchanged in M4.
- **`AgentRegistryService`** (`protos/zynax/v1/agent_registry.proto`) — `RegisterAgent`, `GetAgent`, `ListAgents`. The `api-gateway` calls `RegisterAgent` for `kind: AgentDef` apply. Contract unchanged in M4.
- **`api-gateway` service** (`services/api-gateway/`) — skeleton exists (BDD feature file, no Go implementation). M4 provides the full Go implementation.
- **YAML manifest examples** (`spec/workflows/examples/`) — `code-review.yaml`, `ci-pipeline.yaml`, `research-task.yaml`, `agent-def-example.yaml`. Valid inputs for `zynax apply` in M4 end-to-end tests.

### New entities (introduced by M4)

- **`ApplyService`** (`services/api-gateway/internal/domain/`) — domain service that orchestrates a single apply request: decode YAML `kind`, call the appropriate downstream gRPC service(s), map results to an `ApplyResult` value object.
- **`KindRouter`** (`services/api-gateway/internal/domain/`) — pure function mapping a YAML `kind:` string to the downstream gRPC call sequence. Allowlist: `{Workflow, AgentDef}`. Returns `ErrUnknownKind` for anything else.
- **`ApplyResult`** (`services/api-gateway/internal/domain/`) — value object carrying `run_id` (for `Workflow`) or `agent_id` (for `AgentDef`), compilation warnings, and structured errors.
- **`HTTPHandler`** (`services/api-gateway/internal/api/`) — HTTP handler for `POST /api/v1/apply` and `GET /api/v1/workflows/<id>` and `GET /api/v1/workflows/<id>/logs`. Translates HTTP ↔ domain types; applies `MaxBytesReader` (1 MB body limit).
- **`GatewayClients`** (`services/api-gateway/internal/infrastructure/`) — gRPC client wrappers for `WorkflowCompilerService`, `EngineAdapterService`, `AgentRegistryService`. Implements domain ports — `domain/` never imports gRPC packages directly.
- **`GatewayConfig`** (`services/api-gateway/cmd/api-gateway/`) — `envconfig` struct: compiler address, engine-adapter address, agent-registry address, gRPC port, HTTP port, log level.
- **`zynax` CLI binary** (`cmd/zynax/`) — standalone Go module. Cobra CLI with sub-commands: `apply`, `get workflow`, `delete workflow`, `status workflow`, `logs workflow`, `gitops watch`. Reads `ZYNAX_API_URL` from env (default: `http://localhost:8080`). `--insecure` flag for local dev.
- **`GitOpsWatcher`** (`cmd/zynax/gitops/`) — `zynax gitops watch <dir>` sub-command; uses `fsnotify` to watch a directory for YAML changes; tracks content hashes in `.zynax-watch.state`; calls `POST /api/v1/apply` on create/modify.
- **`DockerComposeStack`** (`infra/docker-compose/`) — `docker-compose.yml` starting all services + Temporal + NATS on the `70xx` port range. `make run-local` / `make stop-local` targets.
- **`ValidateCmd`** (`cmd/zynax/cmd/validate.go`) — `zynax validate` parent Cobra command with one sub-command: `manifest`. Exits 0 on success, 1 on any validation error. `--format json` flag for machine-readable output.
- **`ManifestValidator`** (`cmd/zynax/validate/manifest.go`) — reads the `kind:` field from a YAML file, loads the matching JSON Schema from `spec/schemas/<kind>.json`, validates using a Go JSON Schema library. Returns structured `ValidationError` values with file path and JSON Pointer path.
- **`zynax-ci binary`** (`cmd/zynax-ci/`) — new standalone Go module (separate from `zynax`). CI and developer toolchain binary; replaces all Python scripts in `tools/` and `count-ai-context.sh`. Released as multi-platform binaries alongside `zynax`.
- **`CICanvasValidator`** (`cmd/zynax-ci/validate/canvas.go`) — full Go port of `tools/validate_canvas.py`: checks seven REASONS sections, `**Issue:**`/`**Author:**`/`**Date:**`/`**Status:**` header fields, Status value validity (Draft/Aligned/Implemented/Synced), security checklist marker (`Context Security`) presence. Batch mode: all `canvas.md` files under a directory.
- **`CISchemaValidator`** (`cmd/zynax-ci/validate/schema.go`) — replaces `tools/validate_json_schemas.py`: validates JSON Schema documents for well-formed JSON and correct `$schema` field.
- **`BatchManifestValidators`** (`cmd/zynax-ci/validate/`) — replace `validate_workflows.py`, `validate_agent_defs.py`, `validate_policies.py`: batch-validate all YAML files of a given kind in a directory against the corresponding JSON Schema.
- **`CapabilityValidator`** (`cmd/zynax-ci/validate/capabilities.go`) — replaces `tools/validate_capabilities.py`: validates capability declarations in AgentDef YAMLs against `spec/schemas/capability.schema.json`.
- **`AIContextChecker`** (`cmd/zynax-ci/check/context.go`) — Go port of `tools/count-ai-context.sh`: counts lines in CLAUDE.md, all AGENTS.md files, and `docs/ai-assistant-setup.md` against per-file thresholds; always exits 0 (advisory output only).

### Entity relationships

```
zynax CLI  ──── POST /api/v1/apply ───────────────────────────►  api-gateway (HTTP)
                GET  /api/v1/workflows/<id>                            │
                GET  /api/v1/workflows/<id>/logs                       │
                                                                       ▼
                                                               ApplyService
                                                                  │       │
                                                               KindRouter  │
                                                          ┌───────┘        └────────┐
                                                          ▼                          ▼
                                              WorkflowCompilerService     AgentRegistryService
                                                          │
                                                          ▼
                                              EngineAdapterService
                                                (SubmitWorkflow / WatchWorkflow)
```

---

## A — Approach

**We will:**
- Implement `api-gateway` following the domain/api/infrastructure separation mandated by `services/AGENTS.md`. `internal/domain/` contains `ApplyService` and `KindRouter` with zero gRPC imports (ADR-001, ADR-009).
- Read only the `kind:` field from YAML in the gateway — no full schema parsing in the gateway layer. YAML is passed verbatim as bytes to `WorkflowCompilerService.CompileWorkflow` (ADR-011).
- Return all compiler `CompilationError` structs (including `line_number`) in the HTTP response body as structured JSON, never swallowed (ADR-016 — error visibility).
- Implement the `zynax` CLI as a separate Go module (`cmd/zynax/go.mod`) so it can be released as a standalone binary without pulling in service internals (ADR-008 — no cross-service internal imports).
- Use `fsnotify` for the GitOps watcher — no polling loop; content-hash tracked in `.zynax-watch.state` for idempotent restarts.
- Maintain a strict tool boundary: **`zynax`** = operational platform CLI (apply/manage/observe platform resources; single-file pre-apply manifest validation); **`zynax-ci`** = CI and developer toolchain (batch spec validation, Canvas structural checking, schema well-formedness, AI context budget monitoring). If the validation target is a runtime manifest, it is `zynax`. If it is a CI or documentation artifact, it is `zynax-ci`.
- Replace all Python validation scripts in `tools/` and `count-ai-context.sh` with Go implementations in `cmd/zynax-ci/`. This removes the Python runtime dependency from all CI validation paths and makes the validators distributable as a standalone `zynax-ci` binary alongside `zynax`.
- Write BDD `.feature` file for `/api/v1/apply` scenarios **before** any Go implementation (ADR-016 — contracts before code).
- Provide `make run-local` Docker Compose stack as a first-class development target. All services use the same Docker images built by `make build`.
- Forward `engine_hint` from the CLI `--engine` flag to `SubmitWorkflowRequest.engine_hint` — never hardcode an engine name in the gateway or CLI (ADR-015).
- Fix `state/current-milestone.md` to reflect M3 complete, M4 active as the first commit of Step 1.

**We will NOT:**
- Deploy agent containers. `kind: AgentDef apply` calls `RegisterAgent` only — container scheduling is M6 scope (Kubernetes + Helm).
- Add new proto fields to any existing `.proto` file. All M4 capabilities are achievable with existing contracts.
- Implement `kind: Policy` routing. The allowlist is `{Workflow, AgentDef}` for M4.
- Add TLS between services in the local Docker Compose stack. `--insecure` in the CLI and plain-text inter-service gRPC is acceptable for local dev. TLS is M6.
- Implement `zynax delete` as a hard-delete. It calls `CancelWorkflow` — Temporal owns the execution record.
- Implement multi-tenant auth beyond the token check already covered by the existing BDD scenarios. Auth hardening is M6.
- Implement `kind: Policy` apply or `kind: RoutingRule` apply (ADR-011 §Manifest Kinds are defined but M4 scope is Workflow + AgentDef only).
- Add canvas or schema validation sub-commands to the `zynax` CLI. Canvas and schema validation are exclusively in `zynax-ci`. The `zynax validate` command exposes only `manifest` (single-file pre-apply check).
- Re-implement a full JSON Schema or AsyncAPI specification validator from scratch — `zynax-ci validate schema` and `zynax validate manifest` delegate to established Go JSON Schema libraries; we only add the CLI wrappers and `kind:`-to-schema resolution logic.

**Governing ADRs:** ADR-001 (gRPC inter-service), ADR-008 (no shared databases / no cross-service imports), ADR-009 (Go for services), ADR-011 (declarative YAML control plane), ADR-012 (WorkflowIR as engine-agnostic IR), ADR-013 (adapter-first, no SDK required), ADR-014 (event-driven state machine), ADR-015 (pluggable engines), ADR-016 (layered testing — .feature before code), ADR-017 (GOWORK=off), ADR-019 (SPDD — Canvas before code)

---

## S — Structure

Files touched or created by M4:

```
services/api-gateway/
├── cmd/api-gateway/
│   └── main.go                          ← wiring: gRPC clients + HTTP server
├── internal/
│   ├── domain/
│   │   ├── apply.go                     ← ApplyService + ApplyResult
│   │   ├── kindrouter.go                ← KindRouter + ErrUnknownKind
│   │   └── ports.go                     ← CompilerPort, EnginePort, RegistryPort interfaces
│   ├── api/
│   │   └── handler.go                   ← HTTPHandler: POST /apply, GET /workflows/<id>, GET /workflows/<id>/logs
│   └── infrastructure/
│       └── clients.go                   ← GatewayClients (implements domain ports)
├── tests/
│   └── features/
│       └── api_gateway.feature          ← extend with /apply scenarios (BDD before code)
├── go.mod
└── go.sum

cmd/zynax/                               ← standalone Go module (operational CLI)
├── main.go
├── cmd/
│   ├── apply.go                         ← zynax apply
│   ├── get.go                           ← zynax get workflow <id>
│   ├── delete.go                        ← zynax delete workflow <id>
│   ├── status.go                        ← zynax status workflow <id>
│   ├── logs.go                          ← zynax logs workflow <id>
│   ├── validate.go                      ← zynax validate manifest (single-file only)
│   └── gitops.go                        ← zynax gitops watch <dir>
├── client/
│   └── gateway.go                       ← HTTP client for api-gateway REST
├── gitops/
│   └── watcher.go                       ← GitOps file watcher (fsnotify)
├── validate/
│   └── manifest.go                      ← JSON schema validation against spec/schemas/
├── go.mod
└── go.sum

cmd/zynax-ci/                            ← new standalone Go module (CI toolchain)
├── main.go
├── cmd/
│   ├── root.go                          ← zynax-ci root command
│   ├── validate.go                      ← zynax-ci validate (parent command)
│   ├── validate_canvas.go               ← zynax-ci validate canvas <dir>
│   ├── validate_schema.go               ← zynax-ci validate schema <file>
│   ├── validate_workflows.go            ← zynax-ci validate workflows <dir>
│   ├── validate_agent_defs.go           ← zynax-ci validate agent-defs <dir>
│   ├── validate_capabilities.go         ← zynax-ci validate capabilities <dir>
│   ├── validate_policies.go             ← zynax-ci validate policies <dir>
│   └── check_ai_context.go              ← zynax-ci check ai-context
├── validate/
│   ├── canvas.go                        ← Canvas validator (full port of validate_canvas.py)
│   ├── schema.go                        ← JSON Schema well-formedness validator
│   ├── manifest.go                      ← shared single-file manifest validation logic
│   └── capabilities.go                  ← capability declaration validator
├── check/
│   └── context.go                       ← AI context line counter (port of count-ai-context.sh)
├── AGENTS.md                            ← CI tool norms (mirrors cmd/zynax/AGENTS.md rules)
├── go.mod
└── go.sum

tools/                                   ← Python scripts + count-ai-context.sh deleted in step 11
                                           (replaced by zynax-ci)

infra/docker-compose/
├── docker-compose.yml                   ← all services + Temporal + NATS (70xx ports)
└── README.md                            ← port map + startup notes

state/current-milestone.md               ← update: M3 complete → M4 active

docs/local-dev.md                        ← one-page Docker-first quickstart

.github/workflows/zynax-ci-release.yml  ← multi-platform zynax-ci binary release (mirrors cli-release.yml)
```

Config env prefix: `ZYNAX_API_GATEWAY_` · gRPC port: `ZYNAX_API_GATEWAY_GRPC_PORT` · HTTP port: `ZYNAX_API_GATEWAY_HTTP_PORT`

CLI env: `ZYNAX_API_URL` (default: `http://localhost:8080`)

gRPC contracts: none modified — all existing contracts are sufficient for M4.

---

## O — Operations

Each step is a single PR, independently verifiable. Steps must be executed in order (gateway must exist before CLI can point at it). Each step has a tracking issue.

1. **BDD contract + api-gateway skeleton + `/apply` for `kind: Workflow`** ([#315](https://github.com/zynax-io/zynax/issues/315)) — Update `state/current-milestone.md` (M3 complete, M4 active). Add BDD scenarios for `/api/v1/apply` to `api_gateway.feature`: happy path, YAML parse error with line number, compiler validation error, engine adapter unavailable. Implement `ApplyService`, `KindRouter` (Workflow only), `HTTPHandler`, `GatewayClients`, `main.go` wiring. `POST /api/v1/apply` returns `202` with `run_id`; `GET /api/v1/workflows/<id>` returns `WorkflowRun`. Verify: all BDD scenarios pass; `GOWORK=off go test ./... -race` green; ≥ 90% coverage on `internal/domain/`.

2. **api-gateway `kind: AgentDef` routing** ([#316](https://github.com/zynax-io/zynax/issues/316)) — Extend `KindRouter` to route `AgentDef` → `AgentRegistryService.RegisterAgent`. Add BDD scenarios: valid AgentDef → 201 with `agent_id`; duplicate → 409; unknown `kind:` → 400. Add `RegistryPort` interface + `GatewayClients` implementation for `AgentRegistryService`. Allowlist enforced — no open-ended kind routing. Verify: new BDD scenarios pass; `make test` green.

3. **`zynax` CLI — `apply`, `get`, `delete`, `status` commands** ([#317](https://github.com/zynax-io/zynax/issues/317)) — New Go module `cmd/zynax/`. Cobra CLI with `apply` (calls `POST /api/v1/apply`, prints `run_id`), `apply --dry-run` (prints compiler warnings, exits non-zero on errors), `get workflow <id>` (prints state + status table), `delete workflow <id>` (calls `CancelWorkflow` via gateway), `status workflow <id>` (exits 2 if running, 0 if terminal). `ZYNAX_API_URL` env var; `--insecure` flag. Verify: `GOWORK=off go test ./...` green; `zynax apply spec/workflows/examples/code-review.yaml` succeeds against a running gateway.

4. **`zynax logs` — streaming `WatchWorkflow`** ([#318](https://github.com/zynax-io/zynax/issues/318)) — Add `logs workflow <id>` sub-command to the CLI module. Add `GET /api/v1/workflows/<id>/logs` SSE endpoint to `api-gateway` that proxies `WatchWorkflow`. Default output: human-readable; `--format json` for machine-readable. Stream closes on terminal event; `Ctrl+C` exits 0. Verify: unit tests for event formatting; end-to-end smoke test against local stack.

5. **Local Docker Compose runner** ([#319](https://github.com/zynax-io/zynax/issues/319)) — `infra/docker-compose/docker-compose.yml` for all services + Temporal + NATS on 70xx ports. `make run-local` / `make stop-local` targets. `docs/local-dev.md` quickstart (≤ 100 lines). Verify: `make run-local` starts all services; all `/healthz` probes pass; `zynax apply spec/workflows/examples/code-review.yaml` succeeds end-to-end.

6. **`zynax gitops watch`** ([#320](https://github.com/zynax-io/zynax/issues/320)) — Add `gitops watch <dir>` sub-command using `fsnotify`. Content-hash tracking in `.zynax-watch.state`. Re-applies on create/modify; skips unchanged files on restart. `Ctrl+C` exits 0. Verify: unit tests for hash-tracking; integration smoke: modify a YAML file → confirm re-apply triggered.

7. **`zynax validate manifest` — local JSON schema validation** ([#332](https://github.com/zynax-io/zynax/issues/332)) — Add `cmd/zynax/cmd/validate.go` (parent Cobra command, `manifest` sub-command only) and `cmd/zynax/validate/manifest.go`. Reads the `kind:` field from the target YAML, resolves `spec/schemas/<kind>.json`, validates using a Go JSON Schema library. Structured output: one error per line with file path and JSON Pointer path; `--format json` for machine-readable. Exits 0 on valid, 1 on errors. Verify: `GOWORK=off go test ./... -race` green; `zynax validate manifest spec/workflows/examples/code-review.yaml` exits 0.

8. **`zynax-ci` module scaffold + `validate canvas`** ([#333](https://github.com/zynax-io/zynax/issues/333)) — Create `cmd/zynax-ci/` as a new standalone Go module (`go.mod`, `main.go`, `AGENTS.md`). Cobra CLI with `validate` parent and `validate canvas <dir>` sub-command. Full Go port of `tools/validate_canvas.py`: seven REASONS sections, `**Issue:**`/`**Author:**`/`**Date:**`/`**Status:**` header fields, Status one of (Draft/Aligned/Implemented/Synced), `Context Security` checklist marker present. Batch mode: all `canvas.md` files under a directory tree. Exits 0 if all pass, 1 on any failure. Update the `SPDD Canvas freshness` CI gate in `pr-checks.yml` to invoke `zynax-ci validate canvas` instead of `python tools/validate_canvas.py`. Verify: `GOWORK=off go test ./... -race` green; `zynax-ci validate canvas docs/spdd/` exits 0.

9. **`zynax-ci` remaining validators + `check ai-context`** ([#334](https://github.com/zynax-io/zynax/issues/334)) — Add to `cmd/zynax-ci/`: `validate schema <file>` (replaces `validate_json_schemas.py`; checks well-formed JSON and `$schema` field), `validate workflows <dir>` (replaces `validate_workflows.py`), `validate agent-defs <dir>` (replaces `validate_agent_defs.py`), `validate capabilities <dir>` (replaces `validate_capabilities.py`), `validate policies <dir>` (replaces `validate_policies.py`), `check ai-context` (Go port of `count-ai-context.sh`; per-file line thresholds for CLAUDE.md/AGENTS.md/ai-assistant-setup.md; always exits 0). Verify: `GOWORK=off go test ./... -race` green; each validator produces correct pass/fail output against fixture inputs.

10. **CI: update Makefile and workflows to use `zynax-ci`** ([#335](https://github.com/zynax-io/zynax/issues/335)) — Replace all `python3 tools/validate_*.py` invocations in `Makefile` targets (`validate-spec`, `validate-canvas`, `validate-capability-schemas`, etc.) with the corresponding `zynax-ci validate` commands. Update `Dockerfile.tools` (`make build-tools`) to build and embed the `zynax-ci` binary, removing the Python layer. Add `make install-ci-tools` Makefile target (builds `zynax-ci` to `~/bin/zynax-ci`). Add `.github/workflows/zynax-ci-release.yml` for multi-platform binary release on `v*.*.*` tags. Verify: `make validate-spec` and `make validate-canvas` pass with no Python invocations; CI green end-to-end; `Dockerfile.tools` builds without Python.

11. **Remove Python validation scripts from `tools/`** ([#336](https://github.com/zynax-io/zynax/issues/336)) — Delete `tools/validate_canvas.py`, `tools/validate_workflows.py`, `tools/validate_agent_defs.py`, `tools/validate_capabilities.py`, `tools/validate_policies.py`, `tools/validate_json_schemas.py`, and `tools/count-ai-context.sh`. Update `tools/README.md` to document that validation is now handled by `zynax-ci`. Verify: `tools/` contains no Python validation scripts and no shell validation scripts; `make validate-spec`, `make validate-canvas`, and `make lint` still green after deletion.

---

## N — Norms

Cross-cutting standards pulled from root `AGENTS.md`, `services/AGENTS.md`, and `docs/patterns/go-service-patterns.md`:

- **Commit hygiene:** Both `Signed-off-by` and `Assisted-by: Claude/claude-sonnet-4-6` trailers required on every commit (per AGENTS.md §Hard Constraints). Never `Co-Authored-By:` for AI. No emojis in commit messages.
- **BDD-first:** `.feature` file committed and CI-green before any implementation code (ADR-016). `/apply` BDD scenarios are the first commit of Step 1.
- **GOWORK=off:** All `go test` and `go build` invocations inside `services/api-gateway/` and `cmd/zynax/` must use `GOWORK=off` (ADR-017).
- **Interface boundary:** `internal/domain/` defines `CompilerPort`, `EnginePort`, `RegistryPort` interfaces. `internal/infrastructure/` implements them. `internal/domain/` imports nothing from `internal/api/` or `internal/infrastructure/` (services/AGENTS.md layer rule).
- **YAML parsing in gateway is minimal:** The gateway reads only the top-level `kind:` field to route — it does not validate or interpret the full manifest. The raw YAML bytes are forwarded verbatim to `WorkflowCompilerService` (ADR-011).
- **Go functions ≤ 30 lines** (root AGENTS.md §Clean Code). HTTP handler and `ApplyService` orchestration logic must be split into named helpers.
- **No `panic` in production.** All unrecoverable errors become HTTP `500` or `503` responses with structured JSON bodies.
- **All errors wrapped:** use `fmt.Errorf("api-gateway: %w", err)` pattern throughout.
- **Body size limit:** `HTTPHandler` wraps every request body with `http.MaxBytesReader` (1 MB). Large manifests rejected with `413 Request Entity Too Large`.
- **Coverage gate:** ≥ 90% on `internal/domain/` (pure logic, no I/O).
- **Config via env vars:** `envconfig` struct in `cmd/api-gateway/main.go`; no config files at runtime (Twelve-Factor).
- **Health probes:** `/healthz`, `/readyz`, `/startupz` implemented before Step 1 is merged.
- **CLI exit codes:** `0` = success or terminal state reached; `1` = error; `2` = workflow still running (for `status` command scripting).

---

## S — Safeguards

### Context Security (mandatory before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards

- **Never** parse or interpret the full YAML manifest body in `api-gateway`. The gateway reads only `kind:` for routing; all parsing and validation is delegated to `WorkflowCompilerService`. This is ADR-011 — YAML is the compiler's concern, not the gateway's.
- **Never** call `WorkflowCompilerService`, `EngineAdapterService`, or `AgentRegistryService` via HTTP. All downstream calls must use gRPC stubs generated by `make generate-protos` (ADR-001).
- **Never** import domain types across services. `ApplyResult`, `KindRouter`, etc. are internal to `services/api-gateway/internal/`. The CLI communicates with the gateway via HTTP REST only (ADR-008).
- **Never** hardcode an engine name (`"temporal"`) in `api-gateway` or the CLI. The `--engine` flag passes `engine_hint` to `SubmitWorkflowRequest`; the engine adapter selects the backend (ADR-015).
- **Never** add new proto fields or new gRPC methods in M4. All M4 use cases are served by existing contracts. A proto change requires a separate proto review PR.
- **Never** open a `feat:` PR without first verifying this Canvas is status `Aligned` (ADR-019). The human reviewer must change `Draft` → `Aligned` before any Step 2–11 code lands.
- **Never** deploy agent containers from `kind: AgentDef apply`. Registration in the agent registry is the only M4 action — container scheduling is M6 (Kubernetes + Helm).
- **Never** merge a PR where `make lint` or `make test` is red. All BDD scenarios in `api_gateway.feature` must pass before Step 1 is declared done.
- **Never** route an unknown `kind:` value to any downstream service. The `KindRouter` allowlist (`{Workflow, AgentDef}`) must return `ErrUnknownKind` for any other value; the HTTP handler maps this to `400 Bad Request`.
