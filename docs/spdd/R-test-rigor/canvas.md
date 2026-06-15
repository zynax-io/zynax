# REASONS Canvas — EPIC R: Test Rigor (benchmarks · fuzz · integration · e2e · observability)

> Tier 1 (public-safe). `test:`/`ci:` work is SPDD-exempt; this canvas is committed for traceability.

**Issue:** #469 (absorbed) + #493 #553 #1103 · **Milestone:** M7 (v0.6.0)
**Author:** M7 program plan · **Date:** 2026-06-15 · **Status:** Draft

---

## R — Requirements
- **Problem:** integration tests exist but aren't a required gate; no benchmarks/fuzz; e2e is `xfail`;
  no test asserts that telemetry is actually emitted.
- **Done when:** benchmark, fuzz, integration, e2e, and observability-validation tests run in CI as gates.

## E — Entities
```
Benchmarks (IRInterpreter, workflow-compiler) + regression gate
Fuzz harness (YAML→IR compiler)
Integration suite (//go:build integration) as required CI gate
E2E (real `zynax apply`) replacing platform-readiness xfail
Observability-validation test (connected-trace assertion)
```

## A — Approach
**We will:** add benchmarks with a regression gate (#493); fuzz the compiler; promote the integration
suite to a required gate (#553); flip the platform-readiness `xfail` to a real e2e (#1103); add a test
that asserts a run emits a connected trace.
**We will NOT:** add performance/load/chaos tests — **deferred to M8**.
**Governing ADRs:** ADR-016 (layered testing), ADR-017 (GOWORK=off).

## S — Structure (first S)
```
services/{workflow-compiler,engine-adapter}/...   ← benchmarks + fuzz
.github/workflows/ci.yml + e2e-smoke.yml          ← integration + e2e + benchmark gates
protos/tests/ · spec/automation/                  ← e2e + observability-validation
```

## O — Operations (stories — `spdd-story` form)
**R.1 — Benchmarks + regression gate (#493)** · M · `test`
- As a `maintainer`, I want compiler/interpreter benchmarks gated so perf regressions are caught.
- AC: [ ] benchmarks committed; [ ] regression threshold gate in CI. Deps: W.4.

**R.2 — Fuzz the YAML→IR compiler** · S · `test`
- As a `maintainer`, I want fuzzing so malformed manifests can't crash the compiler.
- AC: [ ] fuzz target committed; [ ] runs in CI; [ ] seed corpus from examples. Deps: W.3.

**R.3 — Integration suite as required gate (#553)** · S · `ci`
- As a `maintainer`, I want integration tests required so cross-service breakage is caught pre-merge.
- AC: [ ] `//go:build integration` suite runs in CI as a required check. Deps: none.

**R.4 — Real e2e replaces xfail (#1103)** · M · `test`
- As a `maintainer`, I want a real `zynax apply` e2e so platform-readiness is genuinely verified.
- AC: [ ] platform-readiness `xfail` flipped to a real e2e against the stack. Deps: T.3.

**R.5 — Observability-validation test** · S · `test`
- As a `maintainer`, I want a test asserting a connected trace so observability can't silently regress.
- AC: [ ] test asserts one run produces a connected end-to-end trace (+log correlation). Deps: O.5, O.9.

**Order:** {R.2, R.3} early → R.1 → {R.4, R.5}.

## N — Norms
- `GOWORK=off` for all `go test` in services (ADR-017); BDD at gRPC boundaries (ADR-016).
- `Signed-off-by:` + `Assisted-by:`; `[skip ci]` marker forbidden in messages/PR bodies.

## S — Safeguards (second S)
### Context Security
- [ ] No Tier 2 content; [ ] no PII; [ ] no prompt-injection; [ ] N/A — non-feat (no security-review gate)

### Feature Safeguards
- Never weaken an existing gate to make a new one pass — gates only get stricter.
- Never mark a flaky test `xfail` to bypass — fix or quarantine with a tracked issue.
