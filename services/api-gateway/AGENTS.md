# services/api-gateway — AGENTS.md

> Go 1.26.3+. Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M4 in progress** — HTTP REST layer implemented; auth/rate-limit deferred to M5.

---

## Purpose

The API Gateway is the **single external entry point** to the Zynax platform.
It accepts HTTP requests, routes by manifest `kind`, and delegates to internal domain services.

- `POST /api/v1/apply` — compile + submit a `Workflow`, or register an `AgentDef`
- `GET /api/v1/workflows/{id}` — fetch workflow run status
- `DELETE /api/v1/workflows/{id}` — cancel a running workflow
- `?dry_run=true` — validate without submitting; returns compile errors

Does NOT: implement business logic · store data · call backing services except via port interfaces.

---

## Actual Layout

```
services/api-gateway/
├── cmd/api-gateway/main.go          ← wiring only
├── internal/
│   ├── domain/
│   │   ├── ports.go                 ← CompilerPort, EnginePort, RegistryPort interfaces
│   │   ├── apply.go                 ← ApplyService (kind-routing, dry-run, cancel)
│   │   ├── kindrouter.go            ← extracts kind/apiVersion from raw YAML bytes
│   │   └── errors.go                ← ErrNotFound, ErrEngineUnavailable, ErrAgentAlreadyExists
│   ├── api/
│   │   └── handler.go               ← HTTP mux, request/response JSON, error mapping
│   └── infrastructure/
│       └── clients.go               ← GatewayClients: all three ports via gRPC
├── tests/features/api_gateway.feature
├── go.mod
└── go.sum
```

Config env prefix: `ZYNAX_GW_` · HTTP port: 8080

---

## Port Interfaces (domain/ports.go)

```go
// CompilerPort → WorkflowCompilerService gRPC
CompileWorkflow(ctx, manifestYAML []byte, namespace string, dryRun bool) (CompileResult, error)

// EnginePort → EngineAdapterService gRPC
SubmitWorkflow(ctx, irBytes []byte, engineHint string) (runID string, error)
GetWorkflowStatus(ctx, runID string) (WorkflowRunSummary, error)
CancelWorkflow(ctx, runID string) error

// RegistryPort → AgentRegistryService gRPC
RegisterAgent(ctx, manifestYAML []byte, namespace string) (AgentRegistration, error)
```

---

## Running Tests

```bash
cd services/api-gateway
GOWORK=off go test ./... -race -timeout 60s
```

Coverage requirement: ≥ 90% on `internal/domain/`.
