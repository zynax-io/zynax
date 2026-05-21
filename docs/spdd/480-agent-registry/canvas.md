<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — Agent Registry MVP (in-memory)

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #480 (Epic)
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-19
**Status:** Aligned

**Child issues:** #526 (BDD trim) · #527 (domain layer) · #528 (gRPC wiring + cmd)
**Continues:** #481 (compose wiring, existing)

---

## R — Requirements

**Problem:** `DispatchCapabilityActivity` in engine-adapter queries `AgentRegistryService`
at capability-dispatch time to resolve a capability name to an agent endpoint. The
service directory (`services/agent-registry/`) exists with a committed BDD feature file
and `AGENTS.md` but has zero implementation. `make run-local` fails at every capability
routing call with connection-refused. The advertised end-to-end path
(`zynax apply → capability dispatch → TASK_EVENT_TYPE_COMPLETED`) does not work.

A secondary problem: the committed BDD feature file tests proto concepts that do not
exist in the current contract — `request_id` idempotency key, `Heartbeat` streaming RPC,
`AGENT_STATUS_ACTIVE`/`AGENT_STATUS_OFFLINE`, and `WatchAgentEvents`. These must be
trimmed to the current proto before implementation begins so BDD CI does not describe
a phantom contract.

**Definition of done:**
- `AgentRegistryService` implements all 5 proto RPCs: `RegisterAgent`, `DeregisterAgent`,
  `GetAgent`, `ListAgents`, `FindByCapability`.
- In-memory store; no persistence; single replica.
- BDD feature file aligned with current proto contract; CI green.
- `go.work` updated; `AGENTS.md` reflects in-memory layout.
- After #481 lands: `make run-local` starts 5 services including agent-registry; http-adapter
  registers on startup (already implemented in `agents/adapters/http/`).

---

## E — Entities

### Existing entities (unchanged)

- **`AgentRegistryService`** (`protos/zynax/v1/agent_registry.proto`): 5-RPC gRPC
  contract. `RegisterAgent`, `DeregisterAgent`, `GetAgent`, `ListAgents`,
  `FindByCapability`. Generated stubs live in `protos/generated/go/zynax/v1/`.
- **`AgentDef`** / **`CapabilityDef`** (proto messages): canonical agent record and
  capability descriptor. `RegisterAgentRequest` carries a single `AgentDef agent = 1`
  field — `agent_id` is the idempotency key (proto contract invariant 1).
- **`AgentStatus`** enum: `AGENT_STATUS_REGISTERED = 1`, `AGENT_STATUS_DEREGISTERED = 2`.
- **`DispatchCapabilityActivity`**: Temporal activity in engine-adapter; calls
  `FindByCapability` at dispatch time to resolve agent endpoint.
- **`agents/adapters/http/internal/registry/client.go`**: Already calls `RegisterAgent`
  (exponential backoff) and `DeregisterAgent` from `main.go`. No changes needed.

### New entities

- **`AgentRegistryService` domain service** (`services/agent-registry/internal/domain/service.go`):
  pure business logic — `Register`, `Deregister`, `GetByID`, `FindByCapability`, `List`.
- **`AgentRepository` interface** (`internal/domain/repository.go`): persistence port.
- **`Agent` domain model** (`internal/domain/model.go`): `AgentID`, capabilities slice,
  status, timestamps. Maps from/to `AgentDef` proto at the API boundary only.
- **Domain errors** (`internal/domain/errors.go`): `ErrAgentNotFound`,
  `ErrAgentAlreadyExists`.
- **`memoryRepo`** (`internal/infrastructure/memory_repo.go`): `AgentRepository` backed
  by `map[AgentID]*Agent` + `sync.RWMutex`. Secondary index: `map[string][]AgentID`
  (capability name → agent IDs) for O(1) `FindByCapability` lookups.
- **gRPC handler** (`internal/api/handler.go`): translates proto ↔ domain; maps gRPC
  error codes.
- **Composition root** (`cmd/agent-registry/main.go`): wiring only; `envconfig`-based
  config; gRPC server + health probe.

---

## A — Approach

**What we WILL do:**
- Implement the 5 `AgentRegistryService` proto RPCs against the current contract exactly.
- Use `services/task-broker/` as the structural template (identical hexagonal layout,
  `envconfig` config, `sync.RWMutex` in-memory store).
- Trim the BDD feature file in a dedicated `test:` PR first (step 1), removing the four
  scenarios that test proto concepts absent from the current contract: `request_id`
  idempotency, `Heartbeat` RPC, `ACTIVE`/`OFFLINE` status, `WatchAgentEvents`.
- Implement domain layer (step 2) and gRPC wiring (step 3) as separate PRs,
  mirroring the task-broker delivery pattern (#520, #522).
- Add `./services/agent-registry` to `go.work` and update stale `AGENTS.md` in step 3.

**What we WON'T do:**
- Add persistence or replication (M6+).
- Extend the proto with `Heartbeat` RPC or `AGENT_STATUS_OFFLINE` (M6+; deferred via
  follow-up issue).
- Change any existing proto field numbers or method signatures (ADR-001).
- Add retries or circuit breakers inside the service (M6+).
- Implement `WatchAgentEvents` event publishing (M6+, event-bus epic).

**ADR references:**
- ADR-001: gRPC inter-service protocol — implement proto contract exactly.
- ADR-008: No shared databases — in-memory is correct for MVP.
- ADR-009: Language strategy — Go service.
- ADR-013: Adapter-first — `FindByCapability` is the hot-path; secondary index required.
- ADR-016: Layered testing — BDD file before implementation code; domain coverage ≥ 90%.
- ADR-017: Contract test isolation — `GOWORK=off go test ./...` in service directory.
- ADR-019: SPDD — this Canvas precedes all implementation PRs.

---

## S — Structure

**New files:**
```
services/agent-registry/
├── cmd/agent-registry/main.go          ← composition root (step 3)
├── internal/
│   ├── domain/
│   │   ├── model.go                    ← Agent, Capability, AgentStatus (step 2)
│   │   ├── service.go                  ← AgentRegistryService domain service (step 2)
│   │   ├── repository.go               ← AgentRepository interface (step 2)
│   │   └── errors.go                   ← ErrAgentNotFound, ErrAgentAlreadyExists (step 2)
│   ├── api/
│   │   └── handler.go                  ← gRPC handler (step 3)
│   └── infrastructure/
│       └── memory_repo.go              ← in-memory AgentRepository (step 3)
├── go.mod                              ← new module (step 3)
├── go.sum
└── Dockerfile                          ← service image (step 3)
```

**Modified files:**
- `services/agent-registry/tests/features/agent_registry.feature` — trim to proto scope (step 1)
- `services/agent-registry/AGENTS.md` — update to in-memory layout (step 3)
- `go.work` — add `./services/agent-registry` (step 3)

**Deferred to #481:**
- `infra/docker-compose/docker-compose.yml` — add agent-registry service

---

## O — Operations

1. **[#526]** `test(agent-registry)`: Trim BDD feature file to current proto scope.
   Remove Heartbeat Feature block (3 scenarios), `request_id` idempotency scenarios,
   `ACTIVE`/`OFFLINE` status references, and `WatchAgentEvents` mention. Commit alone;
   CI must be green before step 2 begins.

2. ✅ **[#527]** `feat(agent-registry)`: Domain layer — `Agent` model, `AgentRepository`
   port, `AgentRegistryService` domain service, `ErrAgentNotFound` / `ErrAgentAlreadyExists`
   errors, domain unit tests. ≥ 90% domain coverage. No imports from `api/` or
   `infrastructure/`.

3. **[#528]** `feat(agent-registry)`: gRPC handler, in-memory repository, composition
   root (`main.go`), `Dockerfile`, `go.mod`. Add `./services/agent-registry` to `go.work`.
   Update `AGENTS.md` to reflect in-memory layout. All CI checks green.

4. **[#481]** `chore(infra)`: Wire agent-registry and task-broker into docker-compose;
   verify end-to-end path. (Existing issue.)

---

## N — Norms

- `feat:` PR type for steps 2–3; `test:` for step 1; `chore:` for step 4 (#481).
- BDD feature file trimmed and CI-green before any implementation code (ADR-016).
- `GOWORK=off go test ./... -race` in `services/agent-registry/`.
- Domain coverage ≥ 90% on `internal/domain/`.
- Hexagonal layout enforced: `domain/` has zero imports from `api/` or `infrastructure/`.
- No `panic` in production paths.
- No `_ = err` without inline justification.
- No `insecure.NewCredentials()` outside bufconn/test helpers.
- Config via `envconfig`; env prefix `ZYNAX_REGISTRY_`; gRPC port `50051`.
- Every commit carries the required trailers per CONTRIBUTING.md §Commit Hygiene.
- PR size ≤ 400 LOC per step (domain layer and gRPC wiring are separate PRs for this reason).

---

## S — Safeguards

### Context Security

- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards

- Never share the in-memory store across services — `services/agent-registry/` owns its state exclusively (ADR-008).
- Never import domain types from `services/agent-registry/` into another service — cross-service data flows through gRPC only (ADR-001).
- Never modify existing proto field numbers or remove enum values in `agent_registry.proto` (ADR-001 §backward-compat).
- Never add business logic to `cmd/agent-registry/main.go` or `internal/api/handler.go` — domain logic lives in `internal/domain/` only.
- Never skip BDD trim (step 1) before implementing (steps 2–3): CI would test a phantom contract.
- Never extend the proto in this epic — heartbeat/OFFLINE/WatchAgentEvents are M6+ scope; open a follow-up issue.
