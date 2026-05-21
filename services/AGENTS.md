# services/ — Engineering Contract

> All services in this directory are **Go** (ADR-009). Python patterns belong in `agents/AGENTS.md`.
> Inherits all rules from the root `AGENTS.md`.
> Code templates (bootstrap, domain, repo, API handler, config, Dockerfile): `docs/patterns/go-service-patterns.md`.

---

## Pre-Code Checklist

Before writing any service code, verify:

1. The `.feature` file is written and committed first (ADR-016).
2. The service has a `go.mod` with the correct module path.
3. Config uses `envconfig` — no config files read at runtime.
4. The `domain/` layer has zero imports from `api/` or `infrastructure/`.
5. `cmd/<service>/main.go` is wiring-only — no business logic.
6. Health probes implemented (`/healthz`, `/readyz`, `/startupz`).
7. OTel instrumentation initialized.
8. Prometheus metrics initialized.
9. `golangci-lint` passes with the repo-level config.
10. Import layering enforced (CI fails on violations).

---

## Directory Structure

Every service follows this layout exactly:

```
services/<service-name>/
├── cmd/<service-name>/main.go   ← wiring only
├── internal/
│   ├── domain/                  ← pure business logic; ZERO imports from api or infrastructure
│   │   ├── models.go            ← value objects, entities, domain errors
│   │   ├── ports.go             ← repository/service interfaces
│   │   └── service.go           ← domain service
│   ├── api/                     ← gRPC handlers; proto ↔ domain; error translation here
│   │   └── handler.go
│   └── infrastructure/          ← DB, cache, external clients; implements domain ports
│       └── repository.go
├── go.mod
└── go.sum
```

Layer rule (CI-enforced import analysis):

```
api → domain ← infrastructure
       ↑
  domain: ZERO imports from api or infrastructure
```

---

## Testing Commands

Always use `GOWORK=off` — the workspace root lists modules not yet on disk (ADR-017).

```bash
# All tests (from within the service directory)
cd services/<service-name>
GOWORK=off go test ./... -race -timeout 60s

# With coverage
GOWORK=off go test ./... -coverprofile=coverage.out -covermode=atomic
GOWORK=off go tool cover -func=coverage.out | grep total:

# Via Makefile (runs inside Docker — no local Go needed)
make test-unit-svc SVC=<service-name>
```

Coverage requirement: ≥ 90% on `internal/domain/` (pure logic, no I/O to mock).

### Integration test convention (`//go:build integration`)

Tests that require external services (NATS, Redis, Temporal, a real DB) must carry
a build tag on the **first line** of the file:

```go
//go:build integration

package mypackage_test
```

- `make test-unit` / `go test -tags="" ./...` — **excludes** integration files
- `make test-integration` / `go test -tags=integration ./...` — **includes** them
- CI `test-unit` job never passes `-tags=integration`; `test-integration` job always does

Use `testcontainers-go` to spin up real backing services inside the test.
Never connect to a shared or external service from within a build-tagged test.

---

## Key ADRs

| ADR | Governs |
|-----|---------|
| [ADR-001](../docs/adr/ADR-001-grpc-inter-service-protocol.md) | gRPC as the only inter-service protocol |
| [ADR-008](../docs/adr/ADR-008-no-shared-databases.md) | Each service owns its own schema |
| [ADR-009](../docs/adr/ADR-009-language-strategy.md) | Go for all platform services |
| [ADR-016](../docs/adr/ADR-016-layered-testing-strategy.md) | Testing pyramid |
| [ADR-017](../docs/adr/ADR-017-contract-test-isolation.md) | GOWORK=off for all `go` commands |

---

## AI Anti-patterns (Services Layer)

| Mistake | Correct approach |
|---------|-----------------|
| `go test ./...` without `GOWORK=off` | `GOWORK=off go test ./...` — every time (ADR-017) |
| Integration test without `//go:build integration` | Add the tag so `make test-unit` skips it automatically |
| Importing `internal/` from another service | Use gRPC stubs; never share internal packages |
| Business logic in `api/` | Move to `internal/domain/`; `api/` translates only |
| External packages in `internal/domain/` | Define an interface in domain; implement in `infrastructure/` |
| Integration tests reaching a real DB without `testcontainers` | Use `testcontainers-go` for real DB (ADR-016) |
| Returning raw `error` from a gRPC handler | `return nil, status.Errorf(codes.InvalidArgument, "…")` |
| Makefile target that calls `go` or `python` directly on host | Wrap in `$(TOOLS_RUN) sh -c "…"` |
| Using `distroless/static:nonroot` for a service with `CGO_ENABLED=1` | Only `CGO_ENABLED=0` (fully static) binaries are compatible; use `distroless/cc` or Alpine if CGO is required |
