# 003: bufconn for In-Process gRPC Contract Tests

**Date:** 2026-04-21  **Author:** M1 Engineering

## Context

The BDD contract tests in `protos/tests/` need to make real gRPC calls against
stub server implementations without running an actual network server. The tests
need to be:
- Fast (no network round-trips)
- Parallel-safe (no port conflicts between packages)
- Teardown-safe (no leaked OS sockets after test failures)
- Self-contained (no external processes or Docker in unit test CI)

## Decision

Use `google.golang.org/grpc/test/bufconn` — a Go in-memory net.Listener that
implements the `net.Listener` interface but routes traffic through a byte buffer
rather than a socket.

The shared `testserver.NewBufconnServer(t)` helper in `protos/tests/testserver/`
creates a `bufconn.Listen(1 << 20)` (1 MiB buffer), starts a `grpc.Server` in a
goroutine, registers `t.Cleanup` to call `GracefulStop()` and `Close()`, and
returns the server + a dial function that connects via `lis.DialContext`.

Test packages call `grpc.NewClient("passthrough:///bufconn", grpc.WithContextDialer(dialer))`.

## Alternatives considered

- **Real TCP port (`:0`):** OS assigns a random free port. Works for single tests
  but causes flaky failures when many test packages run in parallel and exhaust
  ephemeral port ranges. `bufconn` uses no OS resources.
- **Unix domain sockets:** Avoids port exhaustion but requires a temp file path,
  cleanup logic, and doesn't work on Windows without WSL. `bufconn` is pure Go.
- **`grpc.TestServer` (gRPC built-in test helper):** Does not exist as a stable
  API; `bufconn` is the documented approach in the gRPC-Go repo.
- **Docker-based real server:** Correct for integration tests but too heavy for
  unit-level contract tests (slow startup, external process dependency, breaks
  offline dev).

## Setup cost

Adding `bufconn` to a new test package:
1. Import `testserver` (already in `protos/tests/go.mod`)
2. Call `testserver.NewBufconnServer(t)` in `ctx.Before`
3. Register the service stub: `pb.RegisterXxxServer(srv, &stubImpl{})`
4. Dial: `grpc.NewClient("passthrough:///bufconn", grpc.WithContextDialer(dialer), ...)`

Total boilerplate: ~10 lines. See `protos/tests/AGENTS.md §Adding a New Service
Test Suite` for the full template.

## Consequences

- All 8 service test packages use the same pattern — consistent and copy-pasteable.
- Tests run in CI without any Docker daemon or network access required.
- `GracefulStop` blocks until all in-flight RPCs complete — no race conditions on
  teardown even in streaming tests.
- Buffer size (1 MiB) is sufficient for all current test payloads; increase if
  tests start sending large binary proto messages.
