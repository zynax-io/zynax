# services/engine-adapter — AGENTS.md

> Go toolchain pinned in the workspace [`go.work`](../../go.work). Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M5 Complete.** Temporal backend fully implemented; cel-go guard evaluation live (#538); explicit RetryPolicy on `DispatchCapabilityActivity` (#569). LangGraph/Argo `WorkflowEngine` backends deferred to M6+.

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
- Active engine selected via `ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE` env var (default: `temporal`).

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

Config env prefix: `ZYNAX_ENGINE_ADAPTER_` · gRPC port: 50055

---

## Critical Rule

**Never hardcode engine names** in business logic. The string `"temporal"` lives
only in config. All dispatch goes through the `WorkflowEngine` interface (ADR-015).

---

## Guard expressions

Transition conditions use **full [CEL](https://cel.dev) syntax** evaluated by
[`github.com/google/cel-go`](https://github.com/google/cel-go). All CEL
operators, macros, and built-ins are available — not a restricted subset.

**Variable binding.** A single variable `ctx` of type `map<string,string>` is
available in every expression. Workflow context entries set by agent payloads are
accessible as `ctx.key` (CEL select syntax, equivalent to `ctx["key"]`).

**Fail-closed.** `evalGuard` returns `false` for any of:
- empty expression string
- compile error (invalid CEL syntax)
- runtime evaluation error
- non-bool result type

No exception is raised — the interpreter logs a `slog.Warn` and treats the
condition as unmet, so the transition is skipped.

**Performance.** `cel.Program` objects are compiled once and cached per unique
expression string in a `sync.Map`. Safe for Temporal workflow replays (programs
are deterministic pure functions with no side effects).

**Example expressions:**

```cel
ctx.status == "approved"
ctx.score >= "90"
ctx.env == "prod" && ctx.feature_flag == "enabled"
has(ctx.error_code)
```

---

## Running Tests

```bash
cd services/engine-adapter
GOWORK=off go test ./... -race -timeout 60s

# BDD contract tests
cd protos/tests
GOWORK=off go test ./engine_adapter_service/... -v -timeout 60s
```
