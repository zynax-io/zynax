# services/api-gateway — AGENTS.md

> Go toolchain pinned in the workspace [`go.work`](../../go.work). Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M5 Complete** — HTTP REST layer, bearer-token auth (constant-time), ReadHeaderTimeout, X-Request-ID middleware, gRPC deadlines all implemented. Rate limiting deferred to M6 (#580).

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
│   │   ├── ports.go                 ← CompilerPort, EnginePort, EventBusPort interfaces
│   │   ├── apply.go                 ← ApplyService (kind-routing, dry-run, cancel)
│   │   ├── kindrouter.go            ← extracts kind/apiVersion from raw YAML bytes
│   │   └── errors.go                ← ErrNotFound, ErrEngineUnavailable, ErrAgentDefRetired
│   ├── api/
│   │   └── handler.go               ← HTTP mux, request/response JSON, error mapping
│   └── infrastructure/
│       └── clients.go               ← GatewayClients: compiler + engine gRPC, events client
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
```

There is no registry port. Push registration is retired (ADR-039): `kind: AgentDef`
applies answer 410 Gone (`ErrAgentDefRetired`) pointing at the Agent custom
resource; the gateway never dials `AgentRegistryService` (push client deleted in
M9.A step 1, #1697). Agent identity is the `zynax.io/v1alpha1` Agent CR.

---

## Namespace Flow (multi-namespace, EPIC #767)

The namespace travels as a value through the control plane — never as an HTTP
header between services (ADR-001). It enters as the `?namespace=` query param and
flows through three hops unchanged:

```
HTTP  POST /api/v1/apply?namespace=team-a
  │   handler.go: ApplyRequest.Namespace = r.URL.Query().Get("namespace")
  ▼
CompileWorkflowRequest.namespace            (CompilerPort.CompileWorkflow arg)
  │   workflow-compiler embeds it into WorkflowIR.namespace (proto field 3)
  ▼
WorkflowIR.namespace  →  CompileResult.Namespace
  │   apply.go: submit() forwards compiled.Namespace (the IR is authoritative)
  ▼
SubmitWorkflowRequest.namespace             (EnginePort.SubmitWorkflow arg)
```

- The compiled IR namespace is **authoritative** at the submit hop — `submit()`
  passes `compiled.Namespace`, not `req.Namespace`, so the engine routes against
  the namespace the compiler actually embedded.
- When `?namespace=` is **absent**, the gateway passes an empty string through
  unchanged; the workflow-compiler substitutes `"default"`. The gateway must
  never invent a namespace of its own (backwards compatible).
- End-to-end coverage: `internal/api/namespace_propagation_test.go` asserts the
  namespace at the compile hop and the submit hop in a single HTTP request.

## Running Tests

```bash
cd services/api-gateway
GOWORK=off go test ./... -race -timeout 60s
```

Coverage requirement: ≥ 90% on `internal/domain/`.
