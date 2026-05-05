<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M3 Temporal Execution Engine

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #214
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-05
**Status:** Aligned

---

## R — Requirements

- **Problem:** WorkflowIR exists (M2 complete) but has no execution path. A workflow compiled by `WorkflowCompilerService` cannot actually run — the `EngineAdapterService` gRPC server is a stub with no backing implementation.
- **Missing capability:** The system cannot execute state transitions, dispatch agent capabilities, receive capability results, or publish workflow lifecycle events. The three-layer model is incomplete end-to-end.
- **M3 adds:** A Temporal-backed engine adapter that receives WorkflowIR, drives a generic state machine interpreter, routes capability actions to TaskBroker, and advances state based on agent result events.
- **Definition of done — observable outcomes:**
  - `EngineAdapterService.Submit(WorkflowIR)` returns a valid `ExecutionID` and the workflow transitions from `initial_state` through one or more intermediate states to a terminal state.
  - Each state transition is driven by an agent capability result: the `_event` field in the completed payload matches a `TransitionIR.event_type` in the active state.
  - CEL guards in `TransitionIR.conditions` are evaluated against the accumulated `ctx` map; only the matching guarded transition advances.
  - CloudEvents (`zynax.workflow.state.entered`, `zynax.workflow.state.exited`, `zynax.workflow.completed`, `zynax.workflow.failed`) are published to NATS at each lifecycle point.
  - `EngineAdapterService.Cancel`, `GetWorkflowStatus`, and `WatchWorkflow` (server-streaming) are wired through `TemporalEngine` and return the correct gRPC responses.
  - `ZYNAX_ENGINE_ACTIVE_ENGINE=temporal` selects the Temporal backend with no code change required.
  - All five BDD scenarios in `engine_adapter.feature` pass (including the env-var bug fix in scenario 5).
  - `make test` green, `make lint` clean, `make security` clean.

---

## E — Entities

### Existing entities (extended by M3)

- **WorkflowIR** (`protos/zynax/v1/workflow_compiler.proto`) — compiled protobuf representation of a workflow; input to the engine adapter. Unchanged in M3 — no new proto fields.
- **EngineAdapterService** (`protos/zynax/v1/engine_adapter.proto`) — gRPC contract with 5 methods: `Submit`, `Signal`, `Cancel`, `GetWorkflowStatus`, `WatchWorkflow`. Contract defined in M1; M3 provides the Temporal-backed implementation.
- **TaskBrokerService** (`protos/zynax/v1/task_broker.proto`) — downstream gRPC service; M3 calls `DispatchTask` from inside a Temporal activity.

### New entities (introduced by M3)

- **WorkflowEngine** (`services/engine-adapter/internal/domain/engine.go`) — Go interface; the only abstraction between the gRPC layer and any concrete engine backend. Selected at startup via `ZYNAX_ENGINE_ACTIVE_ENGINE` env var. Defines: `Submit(ctx, ir) (ExecutionID, error)`, `Signal(ctx, id, event, payload) error`, `Cancel(ctx, id, reason) error`, `GetStatus(ctx, id) (StatusResponse, error)`, `WatchStatus(ctx, id, stream) error`.
- **TemporalEngine** (`services/engine-adapter/internal/infrastructure/temporal.go`) — concrete implementation of `WorkflowEngine`; wraps the Temporal Go SDK client. Translates `WorkflowEngine` calls into Temporal SDK calls. Selected when `ZYNAX_ENGINE_ACTIVE_ENGINE=temporal`.
- **IRInterpreterWorkflow** (`services/engine-adapter/internal/domain/interpreter.go`) — Temporal workflow function; receives `WorkflowIR` as deterministic input; drives the state machine loop: execute actions, match event, evaluate CEL guards on local `ctx`, advance `currentState`, publish lifecycle CloudEvents.
- **DispatchCapabilityActivity** (`services/engine-adapter/internal/domain/activity.go`) — Temporal activity function; called by `IRInterpreterWorkflow` for each action in the active state; opens a gRPC connection to `TaskBrokerService`, calls `DispatchTask`, reads the terminal `TaskEvent`, extracts `_event` from the JSON payload, returns `{event_type, payload}`.
- **ExecutionContext** (`services/engine-adapter/internal/domain/interpreter.go`) — in-memory struct maintaining `currentState` (state ID string) and `ctx` (string→string accumulator map) across Temporal replay steps. Not persisted externally — Temporal's event log is the source of truth.

### Entity relationships

```
EngineAdapterService (gRPC handler)
  └── WorkflowEngine (interface)
        └── TemporalEngine (infrastructure)
              ├── IRInterpreterWorkflow (Temporal workflow)
              │     ├── ExecutionContext  (local state)
              │     └── DispatchCapabilityActivity (Temporal activity)
              │           └── TaskBrokerService (downstream gRPC)
              └── Temporal Go SDK client
```

---

## A — Approach

**We will:**
- Implement `WorkflowEngine` as a Go interface in `internal/domain/` so the gRPC layer never imports engine-specific code directly (ADR-015).
- Implement `TemporalEngine` as the first and only concrete backend for M3 (ADR-015 — pluggable engines).
- Use the **IR Interpreter pattern**: one generic `IRInterpreterWorkflow` function receives the full WorkflowIR as workflow input; no code generation, no per-workflow Temporal workflow types (execution-architecture.md §3.1).
- Use the `_event` field convention for capability result routing: the agent's terminal `TaskEvent.payload` JSON must contain `"_event": "<event_type>"` matching a `TransitionIR.event_type`. Default to `<capability_name>.completed` / `<capability_name>.failed` when `_event` is absent.
- Evaluate CEL guards inside `IRInterpreterWorkflow` using only the local `ctx` map — no external service calls during guard evaluation (Temporal determinism constraint).
- Fix the BDD scenario 5 env var bug (`KEEL_ENGINE_ACTIVE_ENGINE` → `ZYNAX_ENGINE_ACTIVE_ENGINE`) as the first commit, before any implementation (ADR-016).
- Publish lifecycle CloudEvents (`zynax.workflow.state.entered`, `zynax.workflow.state.exited`, `zynax.workflow.completed`, `zynax.workflow.failed`) from within `IRInterpreterWorkflow` via the EventBus gRPC stub (ADR-014).

**We will NOT:**
- Add `event_expression` to `ActionIR` proto in M3. This design decision is deferred to M3.1 to avoid a breaking proto change mid-sprint. The `_event` convention covers the M3 success criterion. (execution-architecture.md §3.2 — Open design decision)
- Implement the LangGraph or Argo engine adapters. M3 scope is Temporal only.
- Add WorkflowIR persistence (database or cache). The Temporal event history is the state store for M3. Persistence is a future milestone concern.
- Implement the `MemoryService` integration from within the engine adapter. Memory calls are agent-side; the adapter passes `workflow_id` through the IR so agents can scope their own memory calls.
- Implement multi-engine routing (selecting engine per workflow). The active engine is process-wide via env var in M3.
- Implement the `event_expression` CEL field in `ActionIR` (deferred to M3.1).

**Governing ADRs:** ADR-001 (gRPC inter-service), ADR-008 (no shared databases), ADR-009 (Go for services), ADR-012 (WorkflowIR as engine-agnostic IR), ADR-014 (event-driven state machine model), ADR-015 (pluggable workflow engines), ADR-016 (layered testing — .feature before code), ADR-017 (GOWORK=off), ADR-019 (SPDD — Canvas before code)

---

## S — Structure

Files touched or created by M3:

```
services/engine-adapter/
├── cmd/engine-adapter/
│   └── main.go                          ← wire Temporal worker + gRPC server (wiring only)
├── internal/
│   ├── domain/
│   │   ├── engine.go                    ← WorkflowEngine interface + domain types
│   │   ├── interpreter.go               ← IRInterpreterWorkflow + ExecutionContext
│   │   └── activity.go                  ← DispatchCapabilityActivity
│   ├── api/
│   │   └── handler.go                   ← EngineAdapterService gRPC handler → WorkflowEngine
│   └── infrastructure/
│       └── temporal.go                  ← TemporalEngine (implements WorkflowEngine)
├── tests/
│   └── features/
│       └── engine_adapter.feature       ← fix env var bug; add M3 BDD scenarios
├── go.mod
└── go.sum

protos/zynax/v1/
└── engine_adapter.proto                 ← read-only in M3; no new fields needed
```

Config env prefix: `ZYNAX_ENGINE_ADAPTER_` · Active engine: `ZYNAX_ENGINE_ACTIVE_ENGINE` (default: `temporal`)

gRPC contracts extended: none — all 5 `EngineAdapterService` methods exist in proto; M3 provides the backing implementation.

---

## O — Operations

Each step is a single PR, independently verifiable. Steps must be executed in order (each depends on the previous domain layer). Each step has a tracking issue.

1. **BDD contract + WorkflowEngine interface** ([#301](https://github.com/zynax-io/zynax/issues/301)) — Fix env var bug in `engine_adapter.feature` scenario 5 (`KEEL_ENGINE_ACTIVE_ENGINE` → `ZYNAX_ENGINE_ACTIVE_ENGINE`). Define `WorkflowEngine` Go interface in `internal/domain/engine.go` with method signatures matching the 5 `EngineAdapterService` gRPC methods. No implementation — interface only. Verify: `.feature` file lints; `go build ./...` succeeds with stub implementations.

2. **DispatchCapabilityActivity** ([#302](https://github.com/zynax-io/zynax/issues/302)) — Implement Temporal activity function in `internal/domain/activity.go`. It receives `(capability_name, input_json, workflow_id, timeout)`, creates a gRPC client to `TaskBrokerService`, calls `DispatchTask`, streams `TaskEvent`s, extracts `_event` from the terminal COMPLETED payload JSON, and returns `{event_type, payload}`. Unit tests cover: successful dispatch, FAILED terminal event, missing `_event` field (defaults to `<capability>.completed`), timeout. Verify: `GOWORK=off go test ./internal/domain/... -race` green.

3. **IRInterpreterWorkflow** ([#303](https://github.com/zynax-io/zynax/issues/303)) — Implement Temporal workflow function in `internal/domain/interpreter.go`. Drives the state machine loop using `ExecutionContext` (currentState + ctx map): resolve input template against ctx, execute `DispatchCapabilityActivity`, match returned `event_type` against `TransitionIR`, evaluate CEL guards on local ctx only, apply `set{}` mutations to ctx, advance state, publish lifecycle CloudEvents. Temporal determinism constraint: no external gRPC calls in the workflow function body except via registered activities. Verify: unit tests with Temporal test suite (replay tests for state advancement, guard branching, terminal detection).

4. **TemporalEngine** ([#304](https://github.com/zynax-io/zynax/issues/304)) — Implement `TemporalEngine` struct in `internal/infrastructure/temporal.go`. Implements `WorkflowEngine` interface. Wraps Temporal Go SDK `client.Client`. `Submit`: calls `client.ExecuteWorkflow(IRInterpreterWorkflow, ir)` and returns workflow ID as `ExecutionID`. `Signal`: calls `client.SignalWorkflow`. `Cancel`: calls `client.CancelWorkflow`. `GetStatus`: calls `client.DescribeWorkflowExecution` and maps Temporal status to `EngineAdapterService` status enum. `WatchStatus`: polls `DescribeWorkflowExecution` in a loop, writes to the server-streaming gRPC channel. Verify: unit tests with Temporal test suite mock client; `go vet` and `golangci-lint` clean.

5. **gRPC server wiring + end-to-end BDD** ([#305](https://github.com/zynax-io/zynax/issues/305)) — Wire `EngineAdapterService` gRPC handler in `internal/api/handler.go` to delegate all calls to the `WorkflowEngine` interface. Wire `cmd/engine-adapter/main.go`: start Temporal worker (registers `IRInterpreterWorkflow` and `DispatchCapabilityActivity`), start gRPC server with `EngineAdapterService`. BDD integration test: compile a YAML workflow fixture → Submit to engine adapter → assert terminal state reached → assert CloudEvents published. Verify: all 5 BDD scenarios pass; `make test` green; `make lint` clean.

---

## N — Norms

Cross-cutting standards pulled from root `AGENTS.md`, `services/AGENTS.md`, and `docs/patterns/go-service-patterns.md`:

- **Commit hygiene:** `Signed-off-by` (maintainer name + email per AGENTS.md §Hard Constraints) and `Assisted-by: Claude/claude-sonnet-4-6` on every commit. Never `Co-Authored-By:` for AI. No emojis in commit messages.
- **BDD-first:** `.feature` file committed and CI-green before any implementation code (ADR-016). Scenario 5 env var fix is the first commit of Step 1.
- **GOWORK=off:** All `go test` and `go build` invocations inside `services/engine-adapter/` must use `GOWORK=off` (ADR-017).
- **Interface boundary:** `internal/domain/` imports nothing from `internal/api/` or `internal/infrastructure/`. The `WorkflowEngine` interface lives in `domain/` — handlers and infrastructure reference it upward only.
- **Go functions ≤ 30 lines.** `IRInterpreterWorkflow` state machine loop body exceeds this — split into named helpers (`executeActions`, `resolveTransition`, `applyMutations`).
- **No `panic` in production.** All unrecoverable errors become `codes.Internal` gRPC status errors.
- **All errors wrapped:** use `fmt.Errorf("engine-adapter: %w", err)` pattern.
- **Observability:** structured log entry per state transition; OTel span per `DispatchCapabilityActivity` execution; Prometheus counter `engine_adapter_state_transitions_total{workflow_id, from_state, to_state}`.
- **Coverage gate:** ≥ 90% on `internal/domain/` (pure logic, no I/O).
- **Config via env vars:** `envconfig` struct in `cmd/engine-adapter/main.go`; no config files at runtime.
- **Health probes:** `/healthz`, `/readyz`, `/startupz` implemented before Step 5 is merged.

---

## S — Safeguards

### Context Security (mandatory before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards

- **Never** call `MemoryService`, `AgentRegistryService`, or any external gRPC service from inside `IRInterpreterWorkflow` directly. All external I/O must go through registered Temporal activities — this is Temporal's determinism constraint, not a style preference.
- **Never** evaluate CEL guards against data fetched from an external service at guard-eval time. Guards operate only on the accumulated local `ctx` map. External lookups belong in capability activities, not transition logic.
- **Never** import engine-specific packages (`go.temporal.io/sdk`) above `internal/infrastructure/`. The `internal/domain/` and `internal/api/` layers must compile without the Temporal SDK present (ADR-015).
- **Never** share domain types across services. `WorkflowIR`, `TaskEvent`, `ExecutionID` crossing a service boundary must be proto messages — no Go struct sharing (ADR-008, ADR-001).
- **Never** add `event_expression` to `ActionIR` proto in M3 scope. This change requires a proto review and is deferred to M3.1. Using the `_event` convention in M3 is intentional and documented.
- **Never** hardcode `"temporal"` as the engine name in business logic. Selection is via `ZYNAX_ENGINE_ACTIVE_ENGINE` env var and the `WorkflowEngine` interface. A future engine adapter must require zero changes to domain or api layers (ADR-015).
- **Never** open a `feat:` PR without first verifying this Canvas is status `Aligned` (ADR-019). The human reviewer must change `Draft` → `Aligned` before any Step 2–5 code lands.
- **Never** merge a PR where `make lint` or `make test` is red. All 5 BDD scenarios must pass before Step 5 is declared done.
