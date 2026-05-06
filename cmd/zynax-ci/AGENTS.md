# AGENTS.md — cmd/zynax-ci/

The `zynax-ci` CLI is a **standalone Go module** (`github.com/zynax-io/zynax/cmd/zynax-ci`).
It is the CI and developer toolchain for Zynax — it replaces all Python validation scripts
in `tools/` and `count-ai-context.sh`. It has no dependency on the `zynax` operational CLI,
the api-gateway, or any platform services.

---

## Module Rules

- This module is **not** in `go.work`. Always use `GOWORK=off` for every `go` command here.
- Depends only on `github.com/spf13/cobra` and the standard library.
- No gRPC, no HTTP client, no platform service imports.

## Layout

```
cmd/zynax-ci/
  main.go                  entry point — calls cmd.Execute()
  go.mod / go.sum          standalone module (GOWORK=off)
  AGENTS.md                this file
  cmd/
    root.go                root command, Version variable
    validate.go            validate parent command
    validate_canvas.go     zynax-ci validate canvas <path>
    validate_schema.go     zynax-ci validate schema <file>         [step 9]
    validate_workflows.go  zynax-ci validate workflows <dir>       [step 9]
    validate_agent_defs.go zynax-ci validate agent-defs <dir>      [step 9]
    validate_capabilities.go                                        [step 9]
    validate_policies.go                                            [step 9]
    check_ai_context.go    zynax-ci check ai-context               [step 9]
  validate/
    canvas.go              Canvas validator (full port of tools/validate_canvas.py)
    canvas_test.go
    schema.go              [step 9]
    manifest.go            [step 9]
    capabilities.go        [step 9]
  check/
    context.go             [step 9]
```

## Hard Constraints

- **No platform service imports** — no gRPC, no api-gateway client, no cross-service types.
- **No `os.Exit` outside `cmd.Execute()`** — return errors up to Cobra.
- **No `panic`** — return errors; never crash.
- **GOWORK=off** for all `go` commands inside this directory.
- **Exit codes:** 0 = all valid · 1 = validation errors found.
- Warnings (e.g., Draft status) are printed but do not cause exit 1.

## Build & Install

```bash
# Build:
cd cmd/zynax-ci && GOWORK=off go build -o zynax-ci .

# Install to ~/bin:
cd cmd/zynax-ci && GOWORK=off go build -o ~/bin/zynax-ci .

# Or via Makefile (step 10):
make install-ci-tools
```

## Testing

```bash
cd cmd/zynax-ci && GOWORK=off go test ./... -race -timeout 60s
```

## Adding a New Sub-Command

1. Add `cmd/zynax-ci/cmd/<name>.go` with a `var <name>Cmd = &cobra.Command{...}`.
2. Register it in `init()` under the correct parent: `validateCmd.AddCommand(...)` or `rootCmd.AddCommand(...)`.
3. Add any new validation logic to `validate/<name>.go` or `check/<name>.go`.
4. Add tests alongside the logic file.
