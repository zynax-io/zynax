<!-- SPDX-License-Identifier: Apache-2.0 -->

# Go Best Practices — Zynax Platform Services

> Scope: `services/*/`, `cmd/zynax/`, `cmd/zynax-ci/`  
> Enforcement: `golangci-lint` (`.golangci.yml`), `make lint test`

---

## Service Layout

Every service follows the hexagonal structure:
```
services/<service>/
  internal/
    api/           ← gRPC handler (delegates to domain; no business logic)
    domain/        ← business logic (ZERO proto/gRPC/Temporal imports)
    infrastructure/ ← concrete adapters (DB, gRPC clients, Temporal SDK)
  cmd/<service>/   ← main.go: wire dependencies and start server
```

**Rule:** `internal/domain/` must have zero imports from `google.golang.org/grpc`,
`go.temporal.io/`, or any `proto` package. The `layer-boundaries` CI gate enforces this.

---

## Context Propagation

```go
// ✅ Always thread context through the call stack
func (s *Service) Dispatch(ctx context.Context, req *Request) (*Response, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    return s.repo.FindByCapability(ctx, req.Capability)
}

// ❌ Never use context.Background() for request-scoped work
go func() {
    s.executeAsync(context.Background(), task) // loses request-ID, tracing, cancellation
}()
```

For long-running goroutines spawned from a request context, derive a detached context with
a fresh cancel — do not carry the HTTP/gRPC deadline, but DO carry the request-ID via
`metadata.NewOutgoingContext` or similar. See #570.

---

## Error Handling

```go
// ✅ Wrap errors with %w so callers can inspect
if err := s.store.Save(ctx, ir); err != nil {
    return nil, fmt.Errorf("save workflow IR %s: %w", ir.WorkflowId, err)
}

// ✅ Use typed sentinel errors at domain boundaries
var ErrNotFound = errors.New("not found")

// ❌ Never discard errors
_ = conn.Close()          // ❌
defer conn.Close()         // ✅ (log if error matters)
```

Map domain errors to gRPC status codes in the `api/` layer only — never in `domain/`.

---

## gRPC Service Implementation

```go
// ✅ Return codes, not panics
func (s *Server) GetWorkflowStatus(ctx context.Context, req *pb.GetWorkflowStatusRequest) (*pb.GetWorkflowStatusResponse, error) {
    if req.WorkflowId == "" {
        return nil, status.Error(codes.InvalidArgument, "workflow_id required")
    }
    state, err := s.engine.GetWorkflowStatus(ctx, domain.ExecutionID(req.WorkflowId))
    if errors.Is(err, domain.ErrNotFound) {
        return nil, status.Error(codes.NotFound, "workflow not found")
    }
    if err != nil {
        return nil, status.Errorf(codes.Internal, "get status: %v", err)
    }
    return mapToProto(state), nil
}
```

---

## Constant-Time Secret Comparison

```go
// ✅ Use crypto/subtle for all bearer-token comparisons
import "crypto/subtle"

want := []byte("Bearer " + key)
got  := []byte(r.Header.Get("Authorization"))
if subtle.ConstantTimeCompare(want, got) != 1 {
    writeError(w, http.StatusUnauthorized, "unauthorized", "UNAUTHORIZED")
    return
}
```

See #567. Never use `==` or `!=` for secret comparison.

---

## HTTP Server Hardening

```go
// ✅ Always set ReadHeaderTimeout to prevent Slowloris attacks
srv := &http.Server{
    Addr:              addr,
    Handler:           mux,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       30 * time.Second,
    WriteTimeout:      60 * time.Second,
    IdleTimeout:       120 * time.Second,
}
```

See #568. Without `ReadHeaderTimeout`, the server is vulnerable to Slowloris.

---

## Logging

```go
// ✅ Use log/slog with structured fields
slog.InfoContext(ctx, "workflow submitted",
    slog.String("workflow_id", id),
    slog.String("engine",      engine),
)

// ❌ No fmt.Println, no log.Printf in production code
// ❌ Never log secrets, tokens, or PII
```

---

## Table-Driven Tests + Race Detector

```go
func TestDispatch(t *testing.T) {
    tests := []struct {
        name       string
        capability string
        wantErr    bool
    }{
        {"found", "summarize", false},
        {"not found", "nonexistent", true},
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            // ...
        })
    }
}
```

Always run `GOWORK=off go test ./... -race -timeout 60s` in service directories.
The `-race` flag is required; CI enforces it.

---

## gRPC Client Deadlines

```go
// ✅ Always set a deadline on outgoing gRPC calls
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
resp, err := client.CompileWorkflow(ctx, req)
```

No `context.Background()` without a timeout on any outgoing gRPC call.

---

## Shared Interceptors

gRPC interceptor boilerplate (logging, tracing, recovery) is copy-pasted across services today.
**When extracting a shared helper:**
- Put it in `pkg/grpcmw/` (internal to this module, not exported)
- ADR-008 "no shared DB" does not prohibit shared utility packages — the rule is about
  data stores and domain types, not helpers

Tracked by Phase 5 standards / review §5.2.

---

## Generated Code

Generated code lives in `gen/go/zynax/v1/` and `protos/generated/python/`.
- **Never edit generated files by hand.** Run `make generate-protos`.
- Generated files are committed to the repository.
- CI verifies freshness via the `stubs-freshness` gate.

---

## Key linter rules (`.golangci.yml`)

| Linter | Rule |
|---|---|
| `errcheck` | Every error must be checked |
| `govet` | All vet checks |
| `staticcheck` | SA + ST checks |
| `gosec` | Security-focused checks (hardcoded secrets, weak crypto) |
| `goconst` | Repeated strings ≥ 3 occurrences must be constants |
| `funlen` | Functions ≤ 60 lines (suppress with `//nolint:funlen // reason`) |
| `goconst` | Suppressed in test files via `ignore-tests: true` |
