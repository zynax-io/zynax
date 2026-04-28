# services/ — AGENTS.md

> Inherit all rules from the root `AGENTS.md`. This file adds service-specific
> implementation patterns that apply to **every** service in this directory.
>
> All services in this directory are Go (ADR-009). Python patterns belong in
> `agents/AGENTS.md`.

---

## Service Checklist (Before Writing Any Code)

When creating or modifying a service, verify:

1. The `.feature` file is written and committed first (ADR-016).
2. The service has a `go.mod` with the correct module path.
3. Config uses `envconfig` — no config files read at runtime.
4. The `domain/` layer has zero imports from `api/` or `infrastructure/`.
5. `cmd/<service>/main.go` is wiring-only — no business logic.
6. Health probes are implemented and registered.
7. OTel instrumentation is initialized.
8. Prometheus metrics are initialized.
9. `golangci-lint` passes with the repo-level `.golangci.yml`.
10. Import layering enforced (CI fails on violations).

---

## Directory Structure

Every service follows this layout:

```
services/<service-name>/
├── cmd/
│   └── <service-name>/
│       └── main.go          ← wiring only — create server, inject deps, start
├── internal/
│   ├── domain/              ← pure business logic; ZERO imports from api or infrastructure
│   │   ├── models.go        ← value objects, entities, domain errors
│   │   ├── ports.go         ← repository/service interfaces (Go interfaces, not impls)
│   │   └── service.go       ← domain service — only imports domain sub-packages
│   ├── api/                 ← gRPC handlers; maps proto ↔ domain; error translation here
│   │   └── handler.go
│   └── infrastructure/      ← DB, cache, external clients; implements domain ports
│       └── repository.go
├── go.mod
└── go.sum
```

Layer rule (enforced by import analysis in CI):
```
api → domain ← infrastructure
       ↑
  domain: ZERO imports from api or infrastructure
```

---

## go.mod Template

Every service uses this base. Module path follows `github.com/zynax-io/zynax/services/<name>`.

```go
module github.com/zynax-io/zynax/services/<service-name>

go 1.22

require (
    google.golang.org/grpc v1.63.0
    google.golang.org/protobuf v1.34.0
    github.com/zynax-io/zynax/protos/generated/go v0.0.0
    go.opentelemetry.io/otel v1.26.0
    go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.51.0
    github.com/prometheus/client_golang v1.19.0
    github.com/kelseyhightower/envconfig v1.4.0
    log/slog v0.0.0  // stdlib since Go 1.21
)

replace github.com/zynax-io/zynax/protos/generated/go => ../../protos/generated/go
```

---

## gRPC Server Bootstrap Pattern

Every service `cmd/<service>/main.go` follows this exact pattern. Do not deviate.

```go
// cmd/<service-name>/main.go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "net"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "google.golang.org/grpc"
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    "<module>/internal/api"
    "<module>/internal/domain"
    "<module>/internal/infrastructure"
    "<module>/internal/config"
    pb "<proto-module>/zynax/v1"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        slog.Error("config load failed", "err", err)
        os.Exit(1)
    }

    // Infrastructure
    repo, err := infrastructure.NewRepository(cfg)
    if err != nil {
        slog.Error("repository init failed", "err", err)
        os.Exit(1)
    }

    // Domain
    svc := domain.NewService(repo)

    // gRPC server
    grpcServer := grpc.NewServer(
        grpc.StatsHandler(otelgrpc.NewServerHandler()),
    )
    pb.Register<ServiceName>Server(grpcServer, api.NewHandler(svc))

    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
    if err != nil {
        slog.Error("listen failed", "err", err)
        os.Exit(1)
    }

    // Metrics server (separate port from gRPC)
    mux := http.NewServeMux()
    mux.Handle("/metrics", promhttp.Handler())
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    metricsSrv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.MetricsPort), Handler: mux}
    go metricsSrv.ListenAndServe()

    // Graceful shutdown on SIGTERM (Kubernetes sends this)
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
    defer stop()

    go func() {
        slog.Info("server started", "grpc_port", cfg.GRPCPort)
        if err := grpcServer.Serve(lis); err != nil {
            slog.Error("grpc serve error", "err", err)
        }
    }()

    <-ctx.Done()
    slog.Info("shutting down")
    grpcServer.GracefulStop()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    metricsSrv.Shutdown(shutdownCtx)
    slog.Info("shutdown complete")
}
```

---

## Domain Service Pattern

```go
// internal/domain/service.go
package domain

import (
    "context"
    "fmt"
)

// AgentRepository is the port — implemented by the infrastructure layer.
type AgentRepository interface {
    Save(ctx context.Context, agent *Agent) error
    FindByID(ctx context.Context, id AgentID) (*Agent, error)
    FindByCapability(ctx context.Context, capability string) ([]*Agent, error)
    Delete(ctx context.Context, id AgentID) error
}

type Service struct {
    repo AgentRepository
}

func NewService(repo AgentRepository) *Service {
    return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, agent *Agent) error {
    if err := agent.Validate(); err != nil {
        return fmt.Errorf("validate agent: %w", err)
    }
    existing, err := s.repo.FindByID(ctx, agent.ID)
    if err != nil && !IsNotFound(err) {
        return fmt.Errorf("check existing agent %s: %w", agent.ID, err)
    }
    if existing != nil {
        return fmt.Errorf("register agent %s: %w", agent.ID, ErrAgentAlreadyExists)
    }
    if err := s.repo.Save(ctx, agent); err != nil {
        return fmt.Errorf("save agent %s: %w", agent.ID, err)
    }
    return nil
}
```

---

## Repository Pattern

```go
// internal/infrastructure/repository.go
package infrastructure

import (
    "context"
    "database/sql"
    "errors"
    "fmt"

    "<module>/internal/domain"
)

type PostgresRepository struct {
    db *sql.DB
}

func NewRepository(cfg *config.Config) (*PostgresRepository, error) {
    db, err := sql.Open("pgx", cfg.DatabaseURL)
    if err != nil {
        return nil, fmt.Errorf("open db: %w", err)
    }
    return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id domain.AgentID) (*domain.Agent, error) {
    row := r.db.QueryRowContext(ctx, `SELECT id, name, endpoint FROM agents WHERE id = $1`, string(id))
    var a domain.Agent
    if err := row.Scan(&a.ID, &a.Name, &a.Endpoint); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, domain.ErrAgentNotFound
        }
        return nil, fmt.Errorf("scan agent %s: %w", id, err)
    }
    return &a, nil
}
```

---

## API Handler Pattern (gRPC ↔ Domain translation)

Error translation from domain to gRPC status codes lives **only** in the api layer.

```go
// internal/api/handler.go
package api

import (
    "context"
    "errors"
    "fmt"

    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    "<module>/internal/domain"
    pb "<proto-module>/zynax/v1"
)

type Handler struct {
    pb.Unimplemented<ServiceName>Server
    svc *domain.Service
}

func NewHandler(svc *domain.Service) *Handler {
    return &Handler{svc: svc}
}

func (h *Handler) RegisterAgent(ctx context.Context, req *pb.RegisterAgentRequest) (*pb.RegisterAgentResponse, error) {
    agent, err := protoToAgent(req)
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
    }
    if err := h.svc.Register(ctx, agent); err != nil {
        return nil, mapError(err)
    }
    return &pb.RegisterAgentResponse{AgentId: string(agent.ID)}, nil
}

func mapError(err error) error {
    switch {
    case errors.Is(err, domain.ErrAgentNotFound):
        return status.Errorf(codes.NotFound, err.Error())
    case errors.Is(err, domain.ErrAgentAlreadyExists):
        return status.Errorf(codes.AlreadyExists, err.Error())
    default:
        return status.Errorf(codes.Internal, "internal error")
    }
}
```

---

## Config Pattern

```go
// internal/config/config.go
package config

import (
    "fmt"

    "github.com/kelseyhightower/envconfig"
)

type Config struct {
    GRPCPort    int    `envconfig:"GRPC_PORT"     default:"50051"`
    MetricsPort int    `envconfig:"METRICS_PORT"  default:"9090"`
    DatabaseURL string `envconfig:"DATABASE_URL"  required:"true"`
    LogLevel    string `envconfig:"LOG_LEVEL"     default:"info"`
}

func Load() (*Config, error) {
    var cfg Config
    if err := envconfig.Process("ZYNAX_<SVC>", &cfg); err != nil {
        return nil, fmt.Errorf("load config: %w", err)
    }
    return &cfg, nil
}
```

---

## Dockerfile Template

```dockerfile
# syntax=docker/dockerfile:1.6
# ─────────────────────────────────────────────────
# Stage 1: builder
# ─────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /service ./cmd/<service-name>

# ─────────────────────────────────────────────────
# Stage 2: runtime
# ─────────────────────────────────────────────────
FROM gcr.io/distroless/static:nonroot AS runtime

COPY --from=builder /service /service

# gRPC, metrics, health (metrics/health share one port via HTTP mux)
EXPOSE 50051 9090

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/service", "-healthcheck"]

ENTRYPOINT ["/service"]
```

---

## Testing

Run service tests (add `-v` for verbose output):

```bash
# All tests for one service
cd services/<service-name>
go test ./... -timeout 60s

# With race detector
go test -race ./... -timeout 60s

# Single package
go test ./internal/domain/... -v

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Lint:

```bash
# From repo root (uses tools Docker image)
make lint

# Or directly (requires golangci-lint installed)
golangci-lint run ./services/<service-name>/...
```

Coverage requirement: ≥ 90% on `internal/domain/` (pure logic, no I/O to mock).
Integration tests hitting real databases use `testcontainers-go`.

**Always use `GOWORK=off`** — the workspace root lists modules not yet on disk:

```bash
cd services/<service-name>
GOWORK=off go test ./... -race -timeout 60s
GOWORK=off go test ./internal/domain/... -v -coverprofile=coverage.out
```

**Governing ADRs:**

| ADR | Governs |
|-----|---------|
| [ADR-001](../docs/adr/ADR-001-grpc-inter-service-protocol.md) | gRPC as the only inter-service protocol |
| [ADR-008](../docs/adr/ADR-008-no-shared-databases.md) | Each service owns its own schema; no shared tables |
| [ADR-009](../docs/adr/ADR-009-language-strategy.md) | Go for all platform services |
| [ADR-016](../docs/adr/ADR-016-layered-testing-strategy.md) | Testing pyramid: BDD at boundaries, unit in domain |
| [ADR-017](../docs/adr/ADR-017-contract-test-isolation.md) | GOWORK=off for all `go test` inside service directories |

---

## Common AI Mistakes

| Mistake | Why it fails | Correct approach |
|---------|-------------|-----------------|
| `go test ./...` without `GOWORK=off` | `go.work` resolves non-existent modules and breaks | `GOWORK=off go test ./...` — every time, no exceptions (ADR-017) |
| Importing `internal/` from another service | Go toolchain blocks cross-module `internal/` imports | Use gRPC stubs; never share internal packages across service boundaries |
| Putting business logic in `api/` | Violates layer separation; `api/` is a translation layer only | Move logic to `internal/domain/`; `api/` only marshals/unmarshals and calls domain |
| Calling external packages in `internal/domain/` | Domain must be pure Go with zero I/O | Define an interface in domain; implement it in `internal/infrastructure/` |
| Writing `go test` integration tests that reach a real DB without `testcontainers` | Flaky in CI; hard to reproduce locally | Use `testcontainers-go` to spin up real dependencies per test run (ADR-016) |
| Returning a raw `error` from a gRPC handler instead of `status.Errorf` | Client receives `Unknown` status — not actionable | Always `return nil, status.Errorf(codes.InvalidArgument, "…")` |
| Adding a new Makefile target that directly calls `go` or `python` commands | Breaks Docker-only workflow; contributors without the tool will fail | Wrap in `$(TOOLS_RUN) sh -c "…"` so it runs inside the zynax-tools Docker image |

---

## Health Probes

Every service exposes health on the metrics HTTP server:

| Path | Response | Meaning |
|------|----------|---------|
| `/healthz` | `200 OK` | Process is alive (liveness probe) |
| `/readyz` | `200 OK` / `503` | Ready to serve traffic (readiness probe) |

`/readyz` MUST return `503` until the gRPC server is fully started and all dependency
connections (DB, NATS) are verified. Kubernetes uses readiness to gate traffic — a
service that returns `200` before it's ready causes request failures on rollout.
