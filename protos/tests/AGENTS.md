# protos/tests/ — AGENTS.md

> BDD contract tests for all eight gRPC service boundaries.
> Written with [godog](https://github.com/cucumber/godog) using in-process
> `bufconn` transport — no network, no Docker, <100ms total.
>
> Full authoring guide (bufconn pattern, adding steps, two-file split):
> `docs/patterns/bdd-contract-testing.md`.
> Governing ADRs: ADR-016 (layered testing), ADR-017 (GOWORK=off).

---

## GOWORK=off — Always Required

**Every `go test` invocation in this directory must be prefixed with `GOWORK=off`.**

The repo root `go.work` lists service modules not yet on disk. Without `GOWORK=off`,
the toolchain fails with a confusing module-resolution error unrelated to the test code.

---

## Package Layout

```
protos/tests/
├── testserver/server.go            ← shared bufconn helper (used by all packages)
├── features/                       ← Gherkin feature files (one per service)
│   ├── agent_service.feature
│   ├── agent_registry_service.feature
│   ├── cloudevents_envelope.feature
│   ├── engine_adapter_service.feature
│   ├── event_bus_service.feature
│   ├── memory_service.feature
│   ├── task_broker_service.feature
│   └── workflow_compiler_service.feature
├── agent_service/steps_test.go
├── agent_registry_service/steps_test.go
├── cloudevents_envelope/steps_test.go
├── engine_adapter_service/
│   ├── lifecycle_steps_test.go     ← two-file split
│   └── signals_steps_test.go
├── event_bus_service/steps_test.go
├── memory_service/steps_test.go
├── task_broker_service/steps_test.go
└── workflow_compiler_service/steps_test.go
```

---

## Running Tests

```bash
# All contract tests
cd protos/tests
GOWORK=off go test ./... -v -timeout 120s

# One service package
GOWORK=off go test ./agent_registry_service/... -v -timeout 60s

# With race detector
GOWORK=off go test -race ./... -timeout 120s

# Via Makefile (inside Docker)
make test-bdd
```

---

## AI Anti-patterns

| Mistake | Correct approach |
|---------|-----------------|
| `go test ./...` without `GOWORK=off` | `GOWORK=off go test ./...` — every invocation (ADR-017) |
| Step definitions before `.feature` file committed | Commit `.feature` first, CI-green, then add steps |
| State in package-level variables | Keep state in the per-suite context struct; re-init in `ctx.Before` |
| Real business logic in the in-process stub | Return fixed or schema-valid responses only |
| Importing the real service's `internal/` into the test stub | The stub is a hand-written fake — no service imports |
| `go mod tidy` from repo root with workspace active | `cd protos/tests && GOWORK=off go mod tidy` |
