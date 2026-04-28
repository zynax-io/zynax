# services/engine-adapter — AGENTS.md

> Go 1.22+. Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M3 target.** Implementation begins in M3 (Temporal first).

---

## Purpose

The Engine Adapter is the **execution bridge** between Zynax IR and concrete
workflow engines. One Go interface (`WorkflowEngine`), multiple backends
selectable at deploy time.

- Implements `WorkflowEngine` for Temporal (M3), LangGraph (M5), Argo (M6).
- Translates Canonical IR → engine-native format.
- Translates engine-native events → Zynax `WorkflowEvent`.
- Routes capability invocations to task-broker.
- Streams execution state changes via gRPC server-streaming.

Does NOT: compile YAML (workflow-compiler) · route capabilities (task-broker) · decide which engine to use (workflow-compiler).

---

## Internal Layout

```
services/engine-adapter/
├── cmd/engine-adapter/main.go
├── internal/
│   ├── api/
│   │   └── handler.go          ← Submit, Signal, Query, Cancel, Watch
│   ├── domain/
│   │   ├── engine.go           ← WorkflowEngine interface (the core port)
│   │   ├── model.go            ← ExecutionID, ExecutionState, WorkflowEvent
│   │   └── errors.go           ← ErrEngineUnavailable, ErrExecutionNotFound
│   └── infrastructure/
│       └── adapters/
│           ├── temporal.go     ← TemporalEngine (M3)
│           ├── langgraph.go    ← LangGraphEngine (M5)
│           └── argo.go         ← ArgoEngine (M6)
├── go.mod
└── Dockerfile
```

Config env prefix: `ZYNAX_ENGINE_` · gRPC port: 50056

---

## Critical Rule

**Never hardcode engine names** in business logic. The string `"temporal"` lives
only in config. All dispatch goes through the `WorkflowEngine` interface (ADR-015).

---

## Running Tests

```bash
cd services/engine-adapter
GOWORK=off go test ./... -race -timeout 60s

# BDD contract tests
cd protos/tests
GOWORK=off go test ./engine_adapter_service/... -v -timeout 60s
```
