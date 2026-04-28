# BDD Contract Testing Guide

> How to write, run, and extend BDD contract tests for Zynax gRPC services.
> Tactical patterns for `protos/tests/` — the structural rules live in
> `protos/tests/AGENTS.md` and `protos/AGENTS.md`.
>
> Governing ADRs: ADR-016 (layered testing), ADR-017 (GOWORK=off).

---

## bufconn — In-Process gRPC

All tests use `testserver.NewBufconnServer` for an in-memory gRPC server.
No ports are bound, no network calls made. Tests run at memory speed.

```go
// testserver/server.go
func NewBufconnServer(t *testing.T) (*grpc.Server, func(context.Context, string) (net.Conn, error)) {
    lis := bufconn.Listen(1 << 20)
    srv := grpc.NewServer()
    t.Cleanup(func() { srv.GracefulStop(); lis.Close() })
    go func() { _ = srv.Serve(lis) }()
    dialer := func(ctx context.Context, _ string) (net.Conn, error) {
        return lis.DialContext(ctx)
    }
    return srv, dialer
}
```

Usage in a test file:

```go
func InitializeScenario(ctx *godog.ScenarioContext) {
    suite := &agentSuite{}
    ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
        t := &testing.T{}
        srv, dialer := testserver.NewBufconnServer(t)
        pb.RegisterAgentServiceServer(srv, &agentStub{})
        conn, _ := grpc.NewClient("passthrough:///bufconn",
            grpc.WithContextDialer(dialer),
            grpc.WithTransportCredentials(insecure.NewCredentials()))
        suite.client = pb.NewAgentServiceClient(conn)
        return ctx, nil
    })
    ctx.Step(`^...`, suite.someStep)
}
```

---

## Adding a Step to an Existing Service

1. Open the relevant `*_steps_test.go` file — it is in `package <service>_test`.
2. Add a step function:
   ```go
   func (s *mySuite) theResponseContainsField(field string) error {
       if s.lastResponse == nil {
           return fmt.Errorf("no response recorded")
       }
       // assertion
       return nil
   }
   ```
3. Register in `InitializeScenario`:
   ```go
   ctx.Step(`^the response contains field "([^"]*)"$`, suite.theResponseContainsField)
   ```
4. Add a Gherkin scenario to `features/<service>.feature`.
5. Verify: `GOWORK=off go test ./<service>/... -v`

---

## Adding a New Service Suite

1. Create `protos/tests/<service_name>/steps_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0
package <service_name>_test

import (
    "context"
    "os"
    "testing"

    "github.com/cucumber/godog"
    pb "github.com/zynax-io/zynax/protos/generated/go/zynax/v1"
    "github.com/zynax-io/zynax/protos/tests/testserver"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

type <service>Suite struct {
    client pb.<ServiceName>Client
}

type <service>Stub struct {
    pb.Unimplemented<ServiceName>Server
}

func TestMain(m *testing.M) {
    os.Exit(m.Run())
}

func Test<ServiceName>(t *testing.T) {
    suite := godog.TestSuite{
        Name:                "<service_name>",
        ScenarioInitializer: InitializeScenario,
        Options: &godog.Options{
            Format:   "pretty",
            Paths:    []string{"../features/<service_name>.feature"},
            TestingT: t,
        },
    }
    if suite.Run() != 0 {
        t.Fatal("non-zero exit status")
    }
}

func InitializeScenario(ctx *godog.ScenarioContext) {
    suite := &<service>Suite{}
    ctx.Before(func(goCtx context.Context, sc *godog.Scenario) (context.Context, error) {
        t := &testing.T{}
        srv, dialer := testserver.NewBufconnServer(t)
        pb.Register<ServiceName>Server(srv, &<service>Stub{})
        conn, _ := grpc.NewClient("passthrough:///bufconn",
            grpc.WithContextDialer(dialer),
            grpc.WithTransportCredentials(insecure.NewCredentials()))
        suite.client = pb.New<ServiceName>Client(conn)
        return goCtx, nil
    })
    // Register step definitions here
}
```

2. Create `protos/tests/features/<service_name>.feature` with at least one scenario.
3. The package is auto-discovered by `GOWORK=off go test ./...`.

---

## Two-File Split Pattern

Use when a single service has more than ~150 lines of step definitions, or when
two groups of scenarios have entirely separate setup/state.

```
engine_adapter_service/
├── lifecycle_steps_test.go   ← workflow lifecycle RPCs (Start, Cancel, Complete, Fail)
└── signals_steps_test.go     ← WatchWorkflow signal delivery
```

Both files declare `package engine_adapter_service_test` and share the same
`engineSuite` struct. Each file owns a separate `TestXxx` function with its own
godog `TestSuite` and tag filter.

Running one file:
```bash
cd protos/tests
GOWORK=off go test ./engine_adapter_service/... -v -run TestLifecycle -timeout 60s
```

---

## Running Specific Godog Tags

```go
// In a test file
func TestLifecycle(t *testing.T) {
    suite := godog.TestSuite{
        Options: &godog.Options{
            Tags: "@lifecycle",
        },
    }
}
```

---

## go.mod Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/cucumber/godog` | BDD test runner |
| `github.com/zynax-io/zynax/protos/generated/go` | Generated gRPC stubs (replace → `../generated/go`) |
| `google.golang.org/grpc` | gRPC runtime + `bufconn` + `credentials/insecure` |
| `google.golang.org/protobuf` | Proto message types |

The `replace` directive always points stubs to `../generated/go` — tests always use
stubs compiled from the current `.proto` files, never a cached version.

---

## Key Anti-patterns

| Mistake | Correct approach |
|---------|-----------------|
| Step definitions before `.feature` file committed | Commit `.feature` first, get CI-green, then add steps |
| State in package-level variables | Keep all state in the per-suite context struct; re-initialise in `ctx.Before` |
| Real business logic in the in-process stub | Return fixed or schema-valid responses — test the contract shape, not the implementation |
| Importing the real service's `internal/` into the test stub | The stub is a hand-written fake satisfying the proto interface — no service imports |
| `go mod tidy` from repo root with workspace active | `cd protos/tests && GOWORK=off go mod tidy` |
