# services/engine-adapter — AGENTS.md

> Go 1.25+. Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M3 Complete.** Temporal backend fully implemented; LangGraph/Argo deferred to M5/M6.

---

## Purpose

The Engine Adapter is the **execution bridge** between Zynax IR and concrete
workflow engines. One Go interface (`WorkflowEngine`), multiple backends
selectable at deploy time.

- Implements `WorkflowEngine` interface; `TemporalEngine` is the M3 concrete backend.
- `IRInterpreterWorkflow` (Temporal workflow) drives the state machine: executes actions,
  matches `_event` from agent results to `TransitionIR`, evaluates CEL guards, advances state.
- `DispatchCapabilityActivity` (Temporal activity) calls `TaskBrokerService.DispatchTask` gRPC.
- Publishes `zynax.workflow.state.entered/exited/completed/failed` CloudEvents.
- Streams execution state via `WatchWorkflow` gRPC server-streaming.
- Active engine selected via `ZYNAX_ENGINE_ACTIVE_ENGINE` env var (default: `temporal`).

Does NOT: compile YAML (workflow-compiler) · route capabilities (task-broker) · decide which engine to use (workflow-compiler).

---

## Internal Layout

```
services/engine-adapter/
├── cmd/engine-adapter/main.go      ← wire Temporal worker + gRPC server
├── internal/
│   ├── api/
│   │   └── handler.go              ← Submit, Signal, Cancel, GetWorkflowStatus, WatchWorkflow
│   ├── domain/
│   │   ├── engine.go               ← WorkflowEngine interface (the core port)
│   │   ├── interpreter.go          ← IRInterpreterWorkflow + ExecutionContext
│   │   └── activity.go             ← DispatchCapabilityActivity
│   └── infrastructure/
│       └── temporal.go             ← TemporalEngine implements WorkflowEngine
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
