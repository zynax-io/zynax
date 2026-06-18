# services/agent-registry — AGENTS.md

> Go toolchain pinned in the workspace [`go.work`](../../go.work). Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M5 Complete.** gRPC service wired with in-memory round-robin store; compose-wired (#481); persistence deferred to M6 (#480 delivered).

---

## Purpose

The Agent Registry is the **source of truth for agent identity and capability lookup** in the mesh.

- Registers agents with their capabilities, endpoint, and metadata on startup.
- Discovers agents by capability name for task-broker dispatch routing.
- Deregisters agents gracefully; deregistered records are retained for audit.
- All state is in-memory (single replica, no persistence — M6+ adds Postgres).

Does NOT: assign tasks · store agent memory · authenticate external callers · publish events (M6+).

---

## Internal Layout

```
services/agent-registry/
├── cmd/agent-registry/
│   └── main.go             ← composition root (envconfig, gRPC server, graceful shutdown)
├── internal/
│   ├── api/
│   │   └── handler.go      ← 5 RPCs: RegisterAgent, DeregisterAgent, GetAgent, ListAgents, FindByCapability
│   ├── domain/
│   │   ├── model.go        ← Agent, Capability, AgentStatus, ListFilter, ListResult
│   │   ├── service.go      ← AgentRegistryService (Register, Deregister, GetByID, FindByCapability, List)
│   │   ├── repository.go   ← AgentRepository interface (port)
│   │   └── errors.go       ← ErrAgentNotFound, ErrAgentAlreadyExists, ErrInvalidArgument
│   └── infrastructure/
│       └── memory_repo.go  ← in-memory AgentRepository (map + sync.RWMutex + capability secondary index)
├── tests/
│   └── features/
│       └── agent_registry.feature  ← BDD spec (trimmed to proto scope in #526)
├── go.mod                  ← module github.com/zynax-io/zynax/services/agent-registry
├── go.sum
└── Dockerfile              ← multi-stage: golang:1.26.3-alpine builder → alpine:3.20 runtime
```

---

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `ZYNAX_REGISTRY_GRPC_PORT` | `50052` | gRPC listener port |
| `ZYNAX_REGISTRY_LOG_LEVEL` | `info` | Log level: debug, info, warn, error |

Config prefix: `ZYNAX_REGISTRY_` (via `kelseyhightower/envconfig`).

---

## gRPC RPCs

| RPC | Request | Response | Notes |
|-----|---------|----------|-------|
| `RegisterAgent` | `RegisterAgentRequest` | `RegisterAgentResponse` | Returns ALREADY_EXISTS if active agent exists |
| `DeregisterAgent` | `DeregisterAgentRequest` | `DeregisterAgentResponse` | Returns NOT_FOUND if unknown |
| `GetAgent` | `GetAgentRequest` | `AgentDef` | Returns deregistered agents (audit) |
| `ListAgents` | `ListAgentsRequest` | `ListAgentsResponse` | Label selector + pagination; excludes deregistered by default |
| `FindByCapability` | `FindByCapabilityRequest` | `FindByCapabilityResponse` | Hot path for task-broker dispatch; secondary index O(1) |

Proto source: `protos/zynax/v1/agent_registry.proto`

---

## In-Memory Store Invariants

- **Primary store**: `map[string]domain.Agent` keyed by `agent_id`.
- **Secondary index**: `map[string]map[string]struct{}` (capability name → set of registered agent IDs). Updated atomically with `Save`/`Delete` under write lock.
- `FindByCapability` returns only `AGENT_STATUS_REGISTERED` agents (enforced by the secondary index).
- `GetAgent` returns agents of any status (for audit purposes).
- `ListAgents` excludes deregistered agents unless `include_deregistered` is set.
- State is lost on restart — M6 will add Postgres persistence.

---

## Running Tests

```bash
cd services/agent-registry
GOWORK=off go test ./... -race -timeout 60s

# BDD contract tests (proto-level, separate module)
cd protos/tests
GOWORK=off go test ./agent_registry_service/... -v -timeout 60s
```

## Known Limitations (M5)

- No persistence — in-memory only; single replica.
- No heartbeat / liveness tracking — M6+ adds `Heartbeat` streaming RPC.
- No authentication middleware — M6+.
- No event publishing (`AgentRegistered`, `AgentDeregistered`) — M6+.

See `docs/spdd/480-agent-registry/canvas.md` for the full REASONS Canvas.
