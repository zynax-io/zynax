# protos/tests/ — AGENTS.md

> This directory contains the BDD contract tests for all eight gRPC service
> boundaries. Tests are written with [godog](https://github.com/cucumber/godog)
> and exercise in-process gRPC stubs over a `bufconn` in-memory transport.
>
> See `protos/AGENTS.md §7` for the contract testing mandate and `CLAUDE.md §testing`
> for the CI enforcement rules. The architectural decision is in ADR-016 and
> ADR-017.

---

## GOWORK=off — Required for Every go test Invocation

**Always prefix `go test` commands in this directory with `GOWORK=off`.**

```bash
cd protos/tests
GOWORK=off go test ./... -v -timeout 60s
```

**Why this is required:**

The repository root `go.work` workspace lists seven service modules (e.g.
`services/workflow-compiler`, `services/engine-adapter`) that will be created in
M2–M4. During M1 (contracts-only) none of those directories exist on disk. When
the Go workspace is active, the toolchain tries to resolve every module listed in
`go.work` — even those not imported by any test — and fails:

```
go: cannot load module providing package github.com/zynax-io/zynax/services/...:
    directory does not exist
```

`GOWORK=off` disables workspace resolution for that single command. It is the
standard Go mechanism and is stable across Go versions. The fix is covered in
ADR-017. Omitting the flag causes a misleading resolution error unrelated to the
test code.

---

## Package Layout

```
protos/tests/
├── go.mod                         ← module github.com/zynax-io/zynax/protos/tests
├── go.sum
├── testserver/
│   └── server.go                  ← shared bufconn helper (used by all packages)
├── features/                      ← shared Gherkin feature files (read by godog at runtime)
│   ├── agent_service.feature
│   ├── agent_registry_service.feature
│   ├── cloudevents_envelope.feature
│   ├── engine_adapter_service.feature
│   ├── event_bus_service.feature
│   ├── memory_service.feature
│   ├── task_broker_service.feature
│   └── workflow_compiler_service.feature
├── agent_service/
│   └── steps_test.go              ← single-file suite
├── agent_registry_service/
│   └── steps_test.go
├── cloudevents_envelope/
│   └── steps_test.go
├── engine_adapter_service/
│   ├── lifecycle_steps_test.go    ← two-file split (see below)
│   └── signals_steps_test.go
├── event_bus_service/
│   └── steps_test.go
├── memory_service/
│   └── steps_test.go
├── task_broker_service/
│   └── steps_test.go
└── workflow_compiler_service/
    └── steps_test.go
```

---

## bufconn — In-Process gRPC (No Network Required)

All tests use `testserver.NewBufconnServer` to create an in-process gRPC server
on an in-memory `bufconn` listener. No ports are bound, no network calls are made.

```go
// testserver/server.go (shared helper)
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
func TestMain(m *testing.M) {
    // godog TestMain — feature file path is relative to the module root
    os.Exit(m.Run())
}

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

**Why bufconn instead of a real network server:**
- No port allocation conflicts in parallel CI runs
- No OS teardown races — `GracefulStop` is synchronous
- Tests run at memory speed, not network speed
- No firewall or container networking required

---

## Running Tests

### All contract tests

```bash
cd protos/tests
GOWORK=off go test ./... -v -timeout 60s
```

### One service package

```bash
cd protos/tests
GOWORK=off go test ./agent_service/... -v -timeout 60s
```

### A specific godog tag

Godog tags map to the `@tag` annotations on Gherkin scenarios. Pass them via
`go test -run` (matches the Go test function name) or via godog's `Tags` option
in the suite initializer:

```bash
# Run only scenarios tagged @lifecycle
cd protos/tests
GOWORK=off go test ./engine_adapter_service/... -v -run TestLifecycle -timeout 60s
```

The suite struct controls which tags are active. In a test file:

```go
func TestLifecycle(t *testing.T) {
    suite := godog.TestSuite{
        Name:                "engine_adapter_lifecycle",
        ScenarioInitializer: InitializeLifecycleScenario,
        Options: &godog.Options{
            Format:   "pretty",
            Paths:    []string{"../features/engine_adapter_service.feature"},
            Tags:     "@lifecycle",
            TestingT: t,
        },
    }
    if suite.Run() != 0 {
        t.Fatal("non-zero exit status")
    }
}
```

### Race detector

```bash
cd protos/tests
GOWORK=off go test -race ./... -timeout 60s
```

---

## Adding a Test Step to an Existing Service

1. Open the relevant `*_steps_test.go` file — it is in `package <service>_test`.
2. Add a step function matching the Gherkin step text:
   ```go
   func (s *mySuite) theResponseContainsField(field string) error {
       if s.lastResponse == nil {
           return fmt.Errorf("no response recorded")
       }
       // ... assertion
       return nil
   }
   ```
3. Register the step in `InitializeScenario`:
   ```go
   ctx.Step(`^the response contains field "([^"]*)"$`, suite.theResponseContainsField)
   ```
4. Add a Gherkin scenario to the corresponding `.feature` file in `features/`.
5. Run `GOWORK=off go test ./...<service>/... -v` to verify.

All files in a package directory share the same `package <service>_test` declaration
— they see each other's types directly.

---

## Adding a New Service Test Suite

Follow this sequence:

1. Create the directory `protos/tests/<service_name>/`.

2. Create `steps_test.go` with this structure:

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

3. Create `features/<service_name>.feature` with at least one scenario.

4. The package is picked up automatically by `GOWORK=off go test ./...` — no
   registration required.

---

## Two-File Split Pattern

Large service test suites can be split across multiple `_test.go` files within
the same package. `engine_adapter_service` uses this:

```
engine_adapter_service/
├── lifecycle_steps_test.go   — workflow lifecycle RPCs (Start, Cancel, Complete, Fail)
└── signals_steps_test.go     — WatchWorkflow signal delivery
```

Both files declare `package engine_adapter_service_test` and share the same
`engineSuite` struct type and `engineStub` server type. Each file owns a
separate `TestXxx` function with its own godog `TestSuite` and tag filter.

Use this pattern when a single feature file has more than ~150 lines of step
definitions, or when two groups of scenarios have entirely separate setup/state.

---

## go.mod Dependencies

The test module depends on:

| Package | Purpose |
|---------|---------|
| `github.com/cucumber/godog` | BDD test runner |
| `github.com/zynax-io/zynax/protos/generated/go` | Generated gRPC stubs (replace directive → `../generated/go`) |
| `google.golang.org/grpc` | gRPC runtime + `bufconn` + `credentials/insecure` |
| `google.golang.org/protobuf` | Proto message types and `timestamppb` |

The `replace` directive in `go.mod` points the generated stubs to the local
`protos/generated/go/` directory so tests always use the stubs that were compiled
from the current `.proto` files:

```
replace github.com/zynax-io/zynax/protos/generated/go => ../generated/go
```
