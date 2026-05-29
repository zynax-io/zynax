<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — Task Broker MVP (in-memory)

> **All content in this Canvas is Tier 1 (public-safe).**
> **Retroactive canvas** — implementation PRs #520, #522, #523 already merged.
> O-steps 1–5 are complete; O-steps 6–8 are the remaining open work.

**Issue:** #479 (Epic)
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-19
**Status:** Implemented

**Parent epic:** [#460 M5.C Capability Dispatch End-to-End](https://github.com/zynax-io/zynax/issues/460)
**Continues:** [#460 canvas](../460-capability-dispatch/canvas.md) O1

**Completed PRs:** #18 (proto) · #59 (proto BDD) · #520 (domain) · #522 (gRPC wiring) · #523 (fix)
**Open child issues:** #530 (AGENTS.md) · #531 (service BDD) · #532 (api tests) · #481 (compose)

---

## R — Requirements

**Problem:** `DispatchCapabilityActivity` in engine-adapter opens a gRPC channel to
`TaskBrokerService` on startup. Before PR #520/#522 landed, there was no task-broker
server — every `zynax apply` failed immediately at capability dispatch with
connection-refused. The service is now running, but three internal quality gaps remain:
`AGENTS.md` describes a different service than what was built, the service-level BDD
feature file references phantom proto concepts with no step definitions, and the gRPC
handler layer has zero test coverage.

**Definition of done (full epic):**
- `TaskBrokerService` implements all 5 proto RPCs: `DispatchTask`, `AcknowledgeTask`,
  `GetTask`, `ListTasks`, `CancelTask`. ✅ **Done.**
- In-memory task store; no persistence; single replica. ✅ **Done.**
- Proto BDD contract suite (28 scenarios in `protos/tests/`) all passing. ✅ **Done.**
- Domain coverage ≥ 90% on `internal/domain/`. ✅ **Done** (92.7%).
- `services/task-broker/AGENTS.md` accurately describes the actual implementation.
- Service-level BDD feature file aligned to proto scope with godog step definitions.
- `internal/api/handler_test.go` covering all 5 handlers and `grpcErr` mapping.
- After #481 lands: task-broker running in `make run-local` alongside the other services.

---

## E — Entities

### Implemented entities (all in `services/task-broker/`)

- **`TaskBrokerService`** (`protos/zynax/v1/task_broker.proto`): 5-RPC gRPC contract.
  `TaskStatus` enum: `PENDING = 1`, `DISPATCHED = 2`, `RETRYING = 3`, `COMPLETED = 4`,
  `FAILED = 5`, `CANCELLED = 6`. `WorkflowTask` is the canonical task record.
- **`TaskService`** (`internal/domain/service.go`): pure domain service. `DispatchTask`
  validates the request, saves a `PENDING` task, launches an async goroutine to execute.
  `AcknowledgeTask` applies retry logic (FAILED → RETRYING when retry_count < max_retries).
  Round-robin agent selection via `atomic.Uint64` index.
- **`Task` / `TaskError` / `AgentInfo`** (`internal/domain/model.go`): canonical domain
  types. `TaskStatus.IsTerminal()` guards state-machine transitions.
- **`TaskRepository` interface** (`internal/domain/ports.go`): `Save`, `GetByID`,
  `Update`, `List`.
- **`AgentFinder` interface** (`internal/domain/ports.go`): `FindByCapability` —
  resolves capability name → slice of `AgentInfo`.
- **`CapabilityExecutor` interface** (`internal/domain/ports.go`): `Execute` — invokes
  a capability on a specific agent, returns result payload or `TaskError`.
- **`memoryRepo`** (`internal/infrastructure/memory_repo.go`): `AgentRepository` backed
  by `map[string]*Task` + `sync.RWMutex`. Cursor-based pagination.
- **`agentExecutor`** (`internal/infrastructure/agent_executor.go`): calls
  `AgentService.ExecuteCapability` on the resolved agent endpoint.
- **`registryClient`** (`internal/infrastructure/registry_client.go`): dials
  `AgentRegistryService.FindByCapability` to resolve capability → agents.
- **`Handler`** (`internal/api/handler.go`): all 5 RPCs. `grpcErr()` maps domain
  sentinels (`ErrTaskNotFound`, `ErrNoEligibleAgent`, `ErrTaskTerminal`,
  `ErrInvalidArgument`) to gRPC status codes.
- **Composition root** (`cmd/task-broker/main.go`): wires repo + executor + finder +
  gRPC server. Config via `envconfig`, prefix `ZYNAX_BROKER_`, gRPC port `50053`.

### Remaining entities (open work)

- **`internal/api/handler_test.go`**: unit tests for all 5 handlers and `grpcErr`
  mapping. Currently the api layer has zero test coverage.
- **Aligned `tests/features/task_broker.feature`**: trim `WatchTask` scenario and
  `ASSIGNED`/`RUNNING`/`SUCCEEDED` status references; add godog step definitions
  targeting the real task-broker server (mirrors `protos/tests/task_broker_service/`
  pattern).

---

## A — Approach

**What was done (PRs #520, #522, #523):**
- Implemented all 5 proto RPCs with in-memory storage (no persistence, ADR-008).
- Hexagonal layout: domain has zero imports from api or infrastructure.
- Async dispatch: `DispatchTask` returns immediately; a goroutine drives the task
  lifecycle to completion by calling the agent and then `applyAcknowledgement`.
- Agent selection: simple round-robin across agents returned by `FindByCapability`.
- `newTaskID()` uses `crypto/rand` with an explicit panic on read failure — justified by
  comment: a `crypto/rand` failure is unrecoverable; the broker cannot safely assign IDs.
  This is the only exception to the no-panic policy.
- Proto BDD contract suite tests against a stub server to verify contract shape.

**What will be done (open O-steps):**
- Update `AGENTS.md` to accurately reflect the implemented layout, correct RPCs,
  correct status names, in-memory-only infrastructure, and actual gRPC port.
- Align service-level BDD feature file: remove phantom proto concepts; add godog
  step definitions that exercise the real service binary via bufconn.
- Add `handler_test.go` to cover all 5 gRPC handlers and the `grpcErr` error-code
  mapping using a bufconn test server (no mock — real domain service behind it).

**What will NOT be done in this epic:**
- Persistence or replication (M6+).
- Priority-based task assignment (not in proto; deferred to M6).
- `WatchTask` streaming RPC (not in current proto; deferred to M6).
- Timeout watchdog / exponential backoff retries (M6+).
- NATS event publishing (M6+, event-bus epic).
- Any proto changes (ADR-001).

**ADR references:**
- ADR-001: gRPC inter-service protocol — implement proto contract exactly; no new RPCs.
- ADR-008: No shared databases — in-memory store is correct for MVP.
- ADR-009: Language strategy — Go service.
- ADR-016: Layered testing — BDD contract tests in `protos/tests/`; domain ≥ 90%;
  service BDD with step definitions.
- ADR-017: Contract test isolation — `GOWORK=off go test ./...` in service directory.
- ADR-019: SPDD — this canvas is retroactive but required before close-out.

---

## S — Structure

```
services/task-broker/
├── AGENTS.md                               ← stale → update (O6)
├── cmd/task-broker/main.go                 ← ✅ done
├── Dockerfile                              ← ✅ done
├── go.mod / go.sum                         ← ✅ done
├── internal/
│   ├── api/
│   │   ├── handler.go                      ← ✅ done (5 RPCs + grpcErr)
│   │   └── handler_test.go                 ← missing → add (O8)
│   ├── domain/
│   │   ├── model.go                        ← ✅ done
│   │   ├── ports.go                        ← ✅ done
│   │   ├── service.go                      ← ✅ done
│   │   ├── service_test.go                 ← ✅ done (92.7% coverage)
│   │   └── errors.go                       ← ✅ done
│   └── infrastructure/
│       ├── memory_repo.go                  ← ✅ done
│       ├── agent_executor.go               ← ✅ done
│       └── registry_client.go              ← ✅ done
└── tests/
    └── features/
        └── task_broker.feature             ← stale → align + add steps (O7)
```

**Proto contracts (unchanged):**
- `protos/zynax/v1/task_broker.proto` — no changes planned.
- `protos/tests/task_broker_service/steps_test.go` — 28 scenarios, all passing. No changes.

**Deferred to #481:**
- `infra/docker-compose/docker-compose.yml` — add task-broker service.

---

## O — Operations

### Completed ✅

1. **[PR #18 / Issue #4]** Define `TaskBrokerService` proto — `DispatchTask`,
   `AcknowledgeTask`, `GetTask`, `ListTasks`, `CancelTask`. `TaskStatus` enum. Merged.

2. **[PR #59]** Proto BDD contract tests — 28 godog scenarios in
   `protos/tests/task_broker_service/`. Stub-server based. All passing. Merged.

3. **[PR #520]** `feat(task-broker)`: Domain layer — `Task` model, `TaskRepository` /
   `AgentFinder` / `CapabilityExecutor` ports, `TaskService`, domain errors, unit tests.
   Domain coverage 92.7%. Merged.

4. **[PR #522]** `feat(task-broker)`: gRPC wiring — `Handler` (5 RPCs + grpcErr),
   `memoryRepo`, `agentExecutor`, `registryClient`, composition root `main.go`,
   `Dockerfile`, `go.mod`. Merged.

5. **[PR #523]** `fix(task-broker)`: Replace panic in `newRequestID` — add inline
   justification comment for `crypto/rand` failure case. Merged.

### Open

6. **[#530]** `docs(task-broker)`: Rewrite `services/task-broker/AGENTS.md` to
   reflect the actual implementation — correct RPC names, `TaskStatus` values,
   in-memory-only infrastructure layout, correct gRPC port (50053), status "M5
   Implemented". Remove references to `postgres.go`, `redis_lock.go`, `nats_events.go`,
   `watchdog.go`, `WatchTask`, `SubmitTask`, `ASSIGNED`/`RUNNING`/`SUCCEEDED`/`TIMED_OUT`.

7. **[#531]** `test(task-broker)`: Align `services/task-broker/tests/features/
   task_broker.feature` to proto scope — remove `WatchTask` scenario, replace
   `ASSIGNED`/`RUNNING`/`SUCCEEDED` with `DISPATCHED`/`COMPLETED`, remove priority
   scenario. Add `tests/steps_test.go` with godog step definitions exercising the real
   service via bufconn (mirrors `protos/tests/task_broker_service/` pattern).

8. **[#532]** `test(task-broker)`: Add `internal/api/handler_test.go` — unit tests
   for all 5 gRPC handlers using a bufconn server backed by real `TaskService` + fake
   repo. Cover `grpcErr` mapping: `ErrTaskNotFound` → `codes.NotFound`,
   `ErrNoEligibleAgent` → `codes.NotFound`, `ErrTaskTerminal` → `codes.FailedPrecondition`,
   `ErrInvalidArgument` → `codes.InvalidArgument`. Bring total service coverage above 80%.

9. **[#481]** `chore(infra)`: Docker-compose wiring — add task-broker and agent-registry
   to compose; fix `ZYNAX_GW_REGISTRY_ADDR`; verify end-to-end path; update README.
   (Existing issue — depends on #528 agent-registry wiring landing first.)

---

## N — Norms

- `docs:` PR type for O6; `test:` for O7–O8; `chore:` for O9 (#481).
- `GOWORK=off go test ./... -race` in `services/task-broker/`.
- Domain coverage ≥ 90% on `internal/domain/` — gate already passing.
- Api-layer coverage target: bring combined service coverage above 80% after O8.
- Hexagonal layout enforced: `internal/domain/` has zero imports from `api/` or
  `infrastructure/`.
- No `panic` in production paths except the explicitly justified `newTaskID()` case
  (crypto/rand failure — documented by inline comment per PR #523).
- No `_ = err` without inline justification.
- No `insecure.NewCredentials()` outside bufconn/test helpers.
- Config via `envconfig`; env prefix `ZYNAX_BROKER_`; gRPC port `50053`.
- Every commit carries the required trailers per CONTRIBUTING.md §Commit Hygiene.
- PR size ≤ 400 LOC for each remaining step.

---

## S — Safeguards

### Context Security

- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards

- Never share the in-memory task store across services — `services/task-broker/` owns its
  state exclusively (ADR-008).
- Never import domain types from `services/task-broker/` into another service — cross-
  service data flows through gRPC only (ADR-001).
- Never modify existing proto field numbers or remove enum values in `task_broker.proto`
  (ADR-001 §backward-compat).
- Never add business logic to `internal/api/handler.go` — domain logic lives in
  `internal/domain/` only; the handler translates proto ↔ domain.
- Never skip the `GOWORK=off` prefix when running `go test` in this service (ADR-017).
- Never add persistence, WatchTask streaming, or priority assignment in this epic —
  those are M6+ scope; open follow-up issues if needed.
