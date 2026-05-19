<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M5.C Capability Dispatch End-to-End

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #460
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-18
**Status:** Aligned

**Child issues:** #479 (task-broker MVP — code merged; cleanup: #530 #531 #532) · #480 (agent-registry MVP — steps: #526 #527 #528) · #481 (compose wiring)

---

## R — Requirements

**Problem:** The 2026-05 architectural review (§3.2, §5.1) identifies that `DispatchCapabilityActivity` in engine-adapter dials `TaskBrokerService` at startup. There is no task-broker server in the repository. Every `zynax apply` fails immediately at capability dispatch with connection refused. The advertised end-to-end path (`make run-local → zynax apply → zynax logs`) does not work.

**Definition of done:**
- `make run-local` starts 5 services without error: api-gateway, workflow-compiler, engine-adapter, task-broker, agent-registry.
- `zynax apply spec/workflows/examples/code-review.yaml` reaches capability dispatch, the http-adapter executes the capability, and a `TASK_EVENT_TYPE_COMPLETED` event is received by the engine-adapter.
- `zynax status wf-<hex>` reports a terminal state.
- The README "Try it with Docker" section is accurate.

---

## E — Entities

### Existing entities (contracts unchanged)
- **`TaskBrokerService`** (`protos/zynax/v1/task_broker.proto`) — existing proto contract with `DispatchTask`, `AcknowledgeTask`, `GetTask`, `ListTasks`, `CancelTask`. The implementation must honour this contract exactly.
- **`AgentRegistryService`** (`protos/zynax/v1/agent_registry.proto`) — existing proto contract with `RegisterAgent`, `DeregisterAgent`, `GetAgent`, `ListAgents`, `FindByCapability`.
- **`AgentDef`** / **`CapabilityDef`** — proto messages describing an agent and its capabilities.
- **`TaskEvent`** — proto message streaming PROGRESS / COMPLETED / FAILED events from agent to broker.
- **`DispatchCapabilityActivity`** — Temporal activity in engine-adapter that submits a task to the broker and waits for completion.
- **`http-adapter`** — existing Go adapter (`agents/adapters/http/`) that implements `AgentService`; must register with the agent-registry on startup.

### New entities
- **`services/task-broker/`** — new Go service implementing `TaskBrokerService`. In-memory. Single replica. No persistence.
- **`services/agent-registry/`** — new Go service implementing `AgentRegistryService`. In-memory. No persistence.
- **In-memory task store** — `map[string]*Task` with `sync.RWMutex` inside task-broker.
- **In-memory agent store** — `map[string]*AgentDef` keyed by agent ID inside agent-registry; secondary index by capability name.

---

## A — Approach

**What we WILL do:**
- Implement task-broker as a standalone Go service following the hexagonal layout (domain / api / infrastructure).
- Implement agent-registry as a standalone Go service following the same layout.
- Use in-memory stores — no persistence, no shared database (ADR-008).
- Wire both services into `infra/docker-compose/docker-compose.yml`.
- HTTP adapter calls `RegisterAgent` on startup and `DeregisterAgent` on graceful shutdown.
- Commit BDD `.feature` files for both services before any implementation code (ADR-016).

**What we WON'T do:**
- Add persistence or replication (that is M6+).
- Add retries or circuit breakers (M6+).
- Implement memory-service or event-bus (separate epics).
- Change any existing proto field numbers (ADR-001).

**ADR references:**
- ADR-001: gRPC inter-service protocol — task-broker and agent-registry must implement their proto contracts exactly.
- ADR-008: No shared databases — in-memory stores are the correct approach for MVP.
- ADR-016: Layered testing — BDD `.feature` file before any implementation code.
- ADR-009: Language strategy — both services are Go.

---

## S — Structure

**New directories:**
- `services/task-broker/cmd/task-broker/main.go`
- `services/task-broker/internal/domain/`
- `services/task-broker/internal/api/`
- `services/agent-registry/cmd/agent-registry/main.go`
- `services/agent-registry/internal/domain/`
- `services/agent-registry/internal/api/`

**Modified files:**
- `protos/tests/features/task_broker_service.feature` — BDD scenarios (committed first)
- `protos/tests/features/agent_registry_service.feature` — update existing stubs to test real server
- `agents/adapters/http/` — add `RegisterAgent` call on startup
- `infra/docker-compose/docker-compose.yml` — add task-broker and agent-registry services
- `go.work` — add task-broker and agent-registry modules

---

## O — Operations

1. **[#479]** ✅ Implement `services/task-broker/` — 5 RPCs (`DispatchTask`, `AcknowledgeTask`, `GetTask`, `ListTasks`, `CancelTask`). In-memory round-robin assignment by capability name. Domain coverage 92.7%. Child quality issues: #530 (AGENTS.md), #531 (BDD align), #532 (handler tests).
2. **[#480]** Implement `services/agent-registry/` — BDD feature file trim (#526) first, then domain (#527) then gRPC wiring (#528). `RegisterAgent`, `DeregisterAgent`, `GetAgent`, `ListAgents`, `FindByCapability`. ≥90% domain coverage.
3. **[#481]** Wire both services into docker-compose; fix `ZYNAX_GW_REGISTRY_ADDR: "localhost:50052"` → `agent-registry:50051`; verify end-to-end path; update README. Depends on #528.

---

## N — Norms

- `feat:` PR type for task-broker and agent-registry implementation.
- `chore:` PR type for compose wiring.
- BDD `.feature` file committed in a separate PR or commit before any implementation code (ADR-016).
- `GOWORK=off go test ./... -race` in both new service directories.
- ≥90% domain coverage for `internal/domain/` in both services.
- Hexagonal layout: `cmd/` (composition root) / `internal/domain/` (pure) / `internal/api/` (gRPC handlers) / `internal/infrastructure/` (if any external deps).
- No `panic` in production paths.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed

### Feature Safeguards
- Never share a database or in-memory store across services — each service owns its own state (ADR-008).
- Never import domain types from one service into another — cross-service data flows through gRPC only.
- BDD `.feature` file must be committed and CI-green before any implementation code (ADR-016).
- Task assignment must use capability name routing — never hard-coded agent IDs (ADR-013, capability routing model).
