# 004: GOWORK=off for Contract Test Isolation

**Date:** 2026-04-21  **Author:** M1 Engineering

## Summary

This decision is fully documented in **ADR-017** (`docs/adr/ADR-017-contract-test-isolation.md`).

The short version: `go.work` lists M2–M4 service modules that do not exist on disk
during M1. Running `go test ./...` in `protos/tests/` without disabling the workspace
causes the Go toolchain to fail resolving those non-existent modules. `GOWORK=off`
disables workspace resolution for the duration of the command without affecting any
other tooling.

## Quick reference

```bash
# Always run contract tests like this:
cd protos/tests
GOWORK=off go test ./... -v -timeout 60s
```

See ADR-017 for the full rationale, alternatives analysis, and consequences.
See `protos/tests/AGENTS.md` for the operational guide.
