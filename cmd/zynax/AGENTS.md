# AGENTS.md — cmd/zynax/

The `zynax` CLI is a **standalone Go module** (`github.com/zynax-io/zynax/cmd/zynax`).
It communicates with the api-gateway over HTTP REST only — no gRPC, no shared domain types
from any service, no imports from `services/*/internal/`.

---

## Module Rules

- This module is **not** in `go.work`. Always use `GOWORK=off` for every `go` command here.
- Depends only on `github.com/spf13/cobra` and the standard library.
- The api-gateway base URL comes from `ZYNAX_API_URL` or `--api-url` flag; default `http://localhost:8080`.

## Layout

```
cmd/zynax/
  main.go           entry point — calls cmd.Execute()
  go.mod / go.sum   standalone module (GOWORK=off)
  cmd/              Cobra command definitions (one file per sub-command)
    root.go         root command, persistent flags, Version variable
    apply.go        zynax apply <file> [--dry-run] [--engine]
    get.go          zynax get workflow <run-id>
    delete.go       zynax delete workflow <run-id>
    status.go       zynax status workflow <run-id>   (exit 0 = terminal, 2 = running)
    logs.go         zynax logs <run-id> [--format text|json]
    validate.go      zynax validate <file> [--schema-dir] [--format]  (local: schema + data-flow)
  validate/
    manifest.go     JSON Schema validation (validate.Manifest) + combined pipeline (validate.File)
    dataflow.go     Workflow state-machine checks (initial_state + transition goto targets)
  client/
    gateway.go      HTTP client for all api-gateway endpoints
    gateway_test.go unit tests (no network — uses httptest.Server)
```

## Hard Constraints

- **No gRPC imports** — the CLI speaks HTTP only. Proto-generated types must never appear here.
- **No imports from `services/*/`** — cross-layer coupling (ADR-001).
- **No `os.Exit` outside `cmd.Execute()`** — return errors up to Cobra, which handles exit codes.
- **No `panic`** — return errors; never crash the CLI.
- **Exit codes:** 0 = success · 1 = error (Cobra default) · 2 = "still running" (status command only).
- **Version string** is injected at build time via ldflags: `-X ...cmd.Version=v0.3.0`.

## Build & Install

```bash
# Build (repo root context, GOWORK=off required):
cd cmd/zynax && GOWORK=off go build -o zynax .

# Install to ~/bin:
cd cmd/zynax && GOWORK=off go build -o ~/bin/zynax .

# Or use the Makefile target:
make install-cli    # builds and installs to ~/bin/zynax
```

## Testing

```bash
cd cmd/zynax && GOWORK=off go test ./... -race -timeout 60s
```

Tests in `client/gateway_test.go` spin up `httptest.Server` — no running api-gateway needed.
Integration testing requires the local stack (`make run-local`).

## Adding a New Sub-Command

1. Add `cmd/zynax/cmd/<name>.go` with a `var <name>Cmd = &cobra.Command{...}`.
2. Register it in `init()` of the same file: `rootCmd.AddCommand(<name>Cmd)`.
3. Add any new HTTP calls to `client/gateway.go`.
4. Add unit tests in `client/gateway_test.go` using `httptest.Server`.
5. Document the command in this file and in `docs/local-dev.md`.

No `.feature` file is required for CLI commands (they are HTTP clients, not gRPC boundary
owners). A `.feature` file IS required for any new gRPC method on the api-gateway side.
