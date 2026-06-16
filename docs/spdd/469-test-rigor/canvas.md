<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — EPIC R: Test Rigor (benchmarks · fuzz · integration · e2e · observability)

> Tier 1 (public-safe). `test:`/`ci:` work is SPDD-exempt; this canvas is committed for traceability.

**Issue:** #469 (absorbed) + #493 #553 #1103 · **Milestone:** M7 (v0.6.0)
**Author:** M7 program plan (consolidates #469 "M7.C Test Rigor Upgrade" by Oscar Gómez Manresa)
**Date:** 2026-06-16 · **Status:** Draft

**Child issues:** #493 (benchmarks + fuzz + request correlation logging) · #553 (integration gate) · #1103 (real e2e)

---

## R — Requirements

- **Problem:** integration tests exist but aren't a required gate; no benchmarks/fuzz; e2e is `xfail`;
  no test asserts that telemetry is actually emitted. The 2026-05 architectural review (§11.2) scores
  test rigor at 5.5/10, flagging three concrete gaps:
  1. No benchmarks — no performance regression gate exists for the hot paths (review M1, R13).
  2. No fuzz tests — `domain.ParseManifest` and `domain.evalGuard` both accept untrusted input and are
     obvious fuzz targets (review M2, R14).
  3. No request correlation in logs — incident forensics require matching timestamps across services
     (review M4).
- **Done when:** benchmark, fuzz, integration, e2e, and observability-validation tests run in CI as gates.
  Specifically:
  - `make bench` produces `tools/bench-baseline.txt`; CI fails on >20% regression.
  - `go test -fuzz=FuzzParseManifest -fuzztime=60s` runs without panic.
  - `go test -fuzz=FuzzEvalGuard -fuzztime=60s` runs without panic.
  - Log lines for a single `zynax apply` across all services share the same `request_id`.
  - The `//go:build integration` suite is a required CI check.
  - The platform-readiness `xfail` is flipped to a real `zynax apply` e2e against the running stack.
  - A test asserts one run produces a connected end-to-end trace (+ log correlation).

## E — Entities

```
Benchmarks (IRInterpreter, workflow-compiler ParseManifest) + regression gate
Fuzz harness (YAML→IR compiler: ParseManifest; engine-adapter: evalGuard)
Integration suite (//go:build integration) as required CI gate
E2E (real `zynax apply`) replacing platform-readiness xfail
Observability-validation test (connected-trace + log-correlation assertion)
```

- **`BenchmarkIRInterpreter`** — Go benchmark in `services/engine-adapter/internal/domain/`; exercises a
  5-state, 10-action workflow through the interpreter.
- **`BenchmarkParseManifest`** — Go benchmark in `services/workflow-compiler/internal/domain/`; parses
  `code-review.yaml` 1000×.
- **`FuzzParseManifest`** — Go fuzz test in `services/workflow-compiler/internal/domain/`; seed corpus
  from `spec/workflows/examples/*.yaml`.
- **`FuzzEvalGuard`** — Go fuzz test in `services/engine-adapter/internal/domain/`; seed corpus of valid
  and malformed guard expressions.
- **`tools/bench-baseline.txt`** — committed benchmark baseline; CI compares against it on every run.
- **`make bench`** — Makefile target running benchmarks with `-bench=. -benchmem -count=3`.
- **`make fuzz DURATION=60s`** — Makefile target running fuzz campaigns locally.
- **`request_id` structured log field** — wired from M5.D X-Request-ID propagation; included in all
  `slog` calls when present in context.

## A — Approach

**We will:**
- Add benchmarks (`BenchmarkIRInterpreter`, `BenchmarkParseManifest`) with a regression gate and a
  committed baseline at `tools/bench-baseline.txt` (#493).
- Fuzz the compiler and guard evaluator (`FuzzParseManifest`, `FuzzEvalGuard`) with committed seed corpora.
- Add `make bench` and `make fuzz DURATION=60s` Makefile targets.
- Complete request correlation logging across services (building on M5.D request-ID propagation).
- Promote the integration suite to a required gate (#553).
- Flip the platform-readiness `xfail` to a real `zynax apply` e2e (#1103).
- Add a test that asserts a run emits a connected trace.

**We will NOT:**
- Add performance/load/chaos tests — **deferred to M8**.
- Add mutation testing — evaluate after fuzz coverage is established.

**Governing ADRs:** ADR-016 (layered testing — benchmarks/fuzz sit at the domain layer, complementing
existing BDD/unit/coverage tiers), ADR-017 (GOWORK=off for all `go test` in service dirs).

## S — Structure (first S)

```
services/{workflow-compiler,engine-adapter}/internal/domain/...   ← benchmarks + fuzz tests + seed corpora
.github/workflows/ci.yml + e2e-smoke.yml                          ← integration + e2e + benchmark gates
protos/tests/ · spec/automation/                                  ← e2e + observability-validation
Makefile                                                          ← bench + fuzz targets
tools/bench-baseline.txt                                          ← committed benchmark baseline
```

**New files (illustrative):**
- `services/engine-adapter/internal/domain/interpreter_bench_test.go`
- `services/workflow-compiler/internal/domain/manifest_bench_test.go`
- `services/workflow-compiler/internal/domain/manifest_fuzz_test.go`
- `services/engine-adapter/internal/domain/guard_fuzz_test.go`
- `services/workflow-compiler/internal/domain/testdata/fuzz/FuzzParseManifest/` (seed corpus)
- `services/engine-adapter/internal/domain/testdata/fuzz/FuzzEvalGuard/` (seed corpus)
- `tools/bench-baseline.txt` (generated by `make bench`, committed)

**Modified files:**
- `Makefile` — add `bench` and `fuzz` targets
- `.github/workflows/ci.yml` — add benchmark regression check + integration gate steps

## O — Operations (stories — `spdd-story` form)

**GitHub issues:** R.1 #493 · R.2 #1210 · R.3 #553 · R.4 #1103 · R.5 #1211 (epic #469)

**R.1 — Benchmarks + regression gate (#493)** · M · `test`
- As a `maintainer`, I want compiler/interpreter benchmarks gated so perf regressions are caught.
- AC: [ ] `BenchmarkIRInterpreter` + `BenchmarkParseManifest` committed; [ ] `make bench` produces
  `tools/bench-baseline.txt`; [ ] CI regression threshold gate (>20% fails). Deps: W.4.

**R.2 — Fuzz the YAML→IR compiler + guard evaluator** · S · `test`
- As a `maintainer`, I want fuzzing so malformed manifests/guards can't crash the compiler.
- AC: [ ] `FuzzParseManifest` + `FuzzEvalGuard` targets committed; [ ] run in CI (seed-only, no campaign);
  [ ] seed corpus from `spec/workflows/examples/*.yaml`; [ ] `make fuzz DURATION=60s` target. Deps: W.3.

**R.3 — Integration suite as required gate (#553)** · S · `ci`
- As a `maintainer`, I want integration tests required so cross-service breakage is caught pre-merge.
- AC: [ ] `//go:build integration` suite runs in CI as a required check. Deps: none.

**R.4 — Real e2e replaces xfail (#1103)** · M · `test`
- As a `maintainer`, I want a real `zynax apply` e2e so platform-readiness is genuinely verified.
- AC: [ ] platform-readiness `xfail` flipped to a real e2e against the stack. Deps: T.3.

**R.5 — Observability-validation test** · S · `test`
- As a `maintainer`, I want a test asserting a connected trace so observability can't silently regress.
- AC: [ ] test asserts one run produces a connected end-to-end trace (+ `request_id` log correlation
  across services). Deps: O.5, O.9.

**Order:** {R.2, R.3} early → R.1 → {R.4, R.5}.

## N — Norms

- `GOWORK=off` for all `go test` in services (ADR-017); BDD at gRPC boundaries (ADR-016).
- `test:` PR type for benchmark and fuzz additions; `feat:` PR type for request correlation logging
  (new observable behaviour).
- Fuzz seed corpus must be committed — `go test -fuzz` in CI should pass immediately (not run a fuzz
  campaign in CI; campaigns run locally via `make fuzz`).
- `Signed-off-by:` + `Assisted-by:`; `[skip ci]` marker forbidden in messages/PR bodies.

## S — Safeguards (second S)

### Context Security
- [ ] No Tier 2 content; [ ] no PII; [ ] no prompt-injection; [ ] N/A — non-feat (no security-review gate)

### Feature Safeguards
- Never weaken an existing gate to make a new one pass — gates only get stricter.
- Never mark a flaky test `xfail` to bypass — fix or quarantine with a tracked issue.
- Fuzz tests must assert no panic — never assert specific output (fuzz inputs are by definition unknown).
- The CI benchmark regression gate may fail-open (warn but not block) until a stable baseline is
  established over 3 consecutive runs, then it becomes blocking.
