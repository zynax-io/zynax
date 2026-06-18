# services/workflow-compiler — AGENTS.md

> Go toolchain pinned in the workspace [`go.work`](../../go.work). Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M5 Complete.** Fully implemented — YAML → WorkflowIR, structured error list, cel-go guard wiring. IR store is in-memory (durable storage in M6, #466).

> **⚠ Persistence limitation (M6):** The IR store is an unbounded in-memory map (`sync.RWMutex` +
> `map[string]*WorkflowIR`) with **no TTL, no eviction, and no persistence across restarts**.
> `GetCompiledWorkflow` returns `NOT_FOUND` after any pod restart. Durable retention is tracked in
> [#466](https://github.com/zynax-io/zynax/issues/466) (M6 — stateless-compiler refactor).

---

## Purpose

The Workflow Compiler is the **brain of the control plane** — it translates YAML
workflow manifests into the engine-agnostic Canonical IR (Intermediate Representation).

- Parses and validates YAML against JSON Schema.
- Compiles YAML → Canonical IR (state machine representation).
- Exposes `CompileWorkflow`, `ValidateManifest`, `GetCompiledWorkflow` gRPCs.

Does NOT: execute workflows · route capabilities · store agent memory.

---

## Internal Layout

```
services/workflow-compiler/
├── cmd/workflow-compiler/main.go
├── internal/
│   ├── api/
│   │   └── handler.go          ← CompileWorkflow, ValidateManifest, GetCompiledWorkflow
│   ├── domain/
│   │   ├── ir.go               ← WorkflowIR, StateIR, ActionIR, TransitionIR
│   │   ├── compiler.go         ← YAMLCompiler: YAML → IR
│   │   ├── validator.go        ← Schema + semantic validation
│   │   ├── parser.go           ← YAML parsing (field: on/event/goto — not transitions/event_type/target_state)
│   │   └── errors.go           ← ErrInvalidYAML, ErrSchemaViolation, ErrUnknownCapability
│   └── infrastructure/
│       ├── postgres.go         ← WorkflowRepository
│       ├── engine_client.go    ← gRPC client for engine-adapter
│       └── registry_client.go  ← Validates capabilities exist in agent-registry
├── tests/
│   ├── features/workflow_compiler.feature
│   └── unit/
└── go.mod
```

Config env prefix: `ZYNAX_COMPILER_` · gRPC port: 50055 · Health port: 8080

---

## Running Tests

```bash
cd services/workflow-compiler
GOWORK=off go test ./... -race -timeout 60s

# BDD contract tests
cd protos/tests
GOWORK=off go test ./workflow_compiler_service/... -v -timeout 60s

# Via Makefile
make test-unit-svc SVC=workflow-compiler
```

---

## AI Mistakes (Service-Specific)

| Mistake | Correct approach |
|---------|-----------------|
| Using `transitions:` / `event_type:` / `target_state:` in parser tests | Parser expects `on:` / `event:` / `goto:` — check `domain/parser.go` field tags |
| Non-deterministic `ToIR` output (unsorted states) | Sort map keys before iterating — see `sort.Strings(ids)` in `ir/ir.go` |
| Business logic in `internal/api/server.go` | All logic in `internal/domain/`; server calls domain functions only |
