# services/agent-registry — AGENTS.md

> Go 1.22+. Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M3+ (not yet implemented).** BDD contract tests exist in `protos/tests/`.

---

## Purpose

The Agent Registry is the **source of truth for agent identity** in the mesh.

- Registers agents with their capabilities, endpoint, and metadata.
- Discovers agents by capability and status.
- Tracks liveness via a client-streaming heartbeat RPC.
- Transitions agents to `OFFLINE` after missed heartbeat windows.
- Publishes `AgentRegistered`, `AgentDeregistered`, `AgentStatusChanged` events.

Does NOT: assign tasks · store agent memory · authenticate external callers.

---

## Internal Layout

```
services/agent-registry/
├── cmd/agent-registry/main.go
├── internal/
│   ├── api/
│   │   └── handler.go          ← RegisterAgent, GetAgent, FindByCapability, Heartbeat
│   ├── domain/
│   │   ├── model.go            ← AgentID, Agent, Capability, AgentStatus
│   │   ├── service.go          ← AgentService (Register, FindByCapability, Deregister)
│   │   ├── repository.go       ← AgentRepository interface
│   │   └── errors.go           ← ErrAgentNotFound, ErrAgentAlreadyExists
│   └── infrastructure/
│       ├── postgres.go         ← PostgresAgentRepository
│       ├── nats_events.go      ← publish agent lifecycle events
│       └── heartbeat.go        ← background goroutine: expire stale agents
├── go.mod
└── Dockerfile
```

Config env prefix: `ZYNAX_REGISTRY_` · gRPC port: 50051

---

## Running Tests

```bash
cd services/agent-registry
GOWORK=off go test ./... -race -timeout 60s

# BDD contract tests
cd protos/tests
GOWORK=off go test ./agent_registry_service/... -v -timeout 60s
```
