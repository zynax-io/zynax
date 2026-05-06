# AGENTS.md — cmd/zynax-ci/

The `zynax-ci` CLI is a **standalone Go module** (`github.com/zynax-io/zynax/cmd/zynax-ci`).
It is the CI and developer toolchain for Zynax. It has no dependency on the `zynax` operational
CLI, the api-gateway, or any platform services.

Pre-built binaries are published to GitHub Releases on `v*.*.*` tags (see
`.github/workflows/zynax-ci-release.yml`). The binary is also shipped inside the
`ghcr.io/zynax-io/zynax/tools` Docker image, rebuilt on every main merge that
changes `cmd/zynax-ci/**` or `infra/docker/Dockerfile.tools`.

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
    validate_schema.go     zynax-ci validate schema <file>
    validate_workflows.go  zynax-ci validate workflows <dir>
    validate_agent_defs.go zynax-ci validate agent-defs <dir>
    validate_capabilities.go  zynax-ci validate capabilities <dir>
    validate_policies.go   zynax-ci validate policies <dir>
    check_ai_context.go    zynax-ci check ai-context
  validate/
    canvas.go              Canvas validator (seven REASONS sections, header fields, Status)
    canvas_test.go
    schema.go              JSON Schema well-formedness validator
    manifest.go            Batch YAML manifest validator (Workflow/AgentDef/Policy)
    capabilities.go        Capability declaration validator
    helpers.go             Shared ValidationError helpers
  check/
    context.go             AI context line-count reporter (advisory, always exits 0)
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
