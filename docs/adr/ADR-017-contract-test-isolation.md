# ADR-017: Contract Test Isolation — GOWORK=off for protos/tests

**Status:** Accepted  **Date:** 2026-04-21
**Related:** ADR-016 (Layered Testing Strategy)

---

## Context

`protos/tests/` contains the Godog BDD contract tests for all eight gRPC service
boundaries. The module at `protos/tests/go.mod` declares:

```
module github.com/zynax-io/zynax/protos/tests
```

The repository root `go.work` workspace file lists seven service modules that will
be created in M2–M4 (e.g. `services/workflow-compiler`, `services/engine-adapter`).
None of these modules exist on disk during M1, which is contracts-only.

When `go test ./...` runs inside `protos/tests/` with the workspace active, the Go
toolchain attempts to resolve every module listed in `go.work`. Resolution fails for
the non-existent service modules, breaking `go test` even though those modules are
not imported by any test package.

This failure manifests as:

```
go: cannot load module providing package github.com/zynax-io/zynax/services/...:
    directory does not exist
```

The bug was discovered and fixed in issue #67 (PR #68) after CI tests began failing
on PRs that did not touch the service directories at all.

## Decision

All `go test` invocations targeting `protos/tests/` MUST be prefixed with
`GOWORK=off` to disable the workspace for the duration of that command.

```bash
cd protos/tests
GOWORK=off go test ./... -v -timeout 60s
```

This applies to:
- Local development (`make test`)
- CI (`test-unit` job in `.github/workflows/ci.yml`)
- Any manual `go test` command documented in AGENTS.md files

## Rationale

| Option | Assessment |
|--------|------------|
| `GOWORK=off` per test run | ✅ Minimal scope — disables workspace only for the affected command; all other workspace tooling unaffected |
| Remove service modules from `go.work` | ✗ Defeats the purpose — workspace exists to prepare IDE and tooling for future modules |
| Create empty stub modules for M2–M4 services now | ✗ Adds maintenance burden and misleads contributors into thinking services exist |
| Move contract tests outside the workspace | ✗ Severs the logical connection between `protos/` and `protos/tests/`; breaks `buf` integration |
| `go.work.sum` exclude directives | ✗ No such mechanism exists in the Go workspace spec |

`GOWORK=off` is the standard Go mechanism for opting a single invocation out of the
workspace. It is stable across Go versions and requires zero structural changes.

## Consequences

- **`make test`** and all CI steps that run contract tests must pass `GOWORK=off`.
  Omitting it will produce a confusing resolution error unrelated to the test being run.
- **`protos/tests/AGENTS.md`** must document this requirement so contributors do not
  cargo-cult `go test ./...` without the flag.
- **`CLAUDE.md`** §Testing approach documents the flag in the critical note block so
  AI assistants generating test commands include it by default.
- As M2–M4 service modules are created and added to the workspace for real, the flag
  remains correct and harmless — `GOWORK=off` on a fully populated workspace is a
  no-op in terms of correctness.
- When all service modules listed in `go.work` exist on disk, this ADR may be
  revisited to evaluate dropping the flag (optional cleanup, not required for
  correctness).
