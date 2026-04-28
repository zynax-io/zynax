# Go Service Patterns — Reference

> Canonical templates for all Go platform services.
> Rules and constraints live in `services/AGENTS.md` and the root `AGENTS.md`.
> This file is reference material — consult it when implementing a service.

---

## go.mod Template

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
)

replace github.com/zynax-io/zynax/protos/generated/go => ../../protos/generated/go
```

---

## gRPC Server Bootstrap (cmd/<service>/main.go)

```go
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

    repo, err := infrastructure.NewRepository(cfg)
    if err != nil {
        slog.Error("repository init failed", "err", err)
        os.Exit(1)
    }

    svc := domain.NewService(repo)

    grpcServer := grpc.NewServer(
        grpc.StatsHandler(otelgrpc.NewServerHandler()),
    )
    pb.Register<ServiceName>Server(grpcServer, api.NewHandler(svc))

    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
    if err != nil {
        slog.Error("listen failed", "err", err)
        os.Exit(1)
    }

    mux := http.NewServeMux()
    mux.Handle("/metrics", promhttp.Handler())
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    metricsSrv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.MetricsPort), Handler: mux}
    go metricsSrv.ListenAndServe() //nolint:errcheck

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
    metricsSrv.Shutdown(shutdownCtx) //nolint:errcheck
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

// AgentRepository is the port — implemented by infrastructure.
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

## API Handler Pattern (gRPC ↔ domain)

Error translation from domain to gRPC status codes lives **only** in the `api` layer.

```go
// internal/api/handler.go
package api

import (
    "context"
    "errors"

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
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /service ./cmd/<service-name>

FROM gcr.io/distroless/static:nonroot AS runtime
COPY --from=builder /service /service
EXPOSE 50051 9090
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/service", "-healthcheck"]
ENTRYPOINT ["/service"]
```

---

## Health Probes

Every service exposes health on the metrics HTTP server:

| Path | Response | Meaning |
|------|----------|---------|
| `/healthz` | `200 OK` | Process alive (liveness probe) |
| `/readyz` | `200 OK` / `503` | Ready to serve traffic (readiness probe) |

`/readyz` MUST return `503` until the gRPC server is fully started and all dependency
connections (DB, NATS) are verified.

---

## Logging

Use `slog` with structured, contextual fields. Never log credentials or auth tokens.

```go
slog.InfoContext(ctx, "workflow compiled",
    "workflow_id", id,
    "target_engine", engine,
    "states", len(states),
    "duration_ms", time.Since(start).Milliseconds())
```
