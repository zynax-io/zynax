<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — Guard Evaluator: cel-go Integration

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #476 (Epic)
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-19
**Status:** Implemented

**Parent epic:** [#459 M5.B Engine Correctness Hardening](https://github.com/zynax-io/zynax/issues/459)
**Track:** M5.B

**Child issues:** #538 (cel-go integration) · #539 (test suite + fuzz seed) · #540 (docs update)

---

## R — Requirements

**Problem:** `evalGuard` in `services/engine-adapter/internal/domain/interpreter.go:168–187` is a 24-line string parser that handles only `==` and `!=`. Any guard expression it does not recognise (e.g. `count < 2`, `ctx.status in ["a","b"]`, a bare variable reference) silently returns `true` (fail-open). This means:

1. A workflow with a malformed or unsupported guard silently advances past a gate it should be blocked at — a silent correctness failure.
2. Documentation and comments describe the feature as "CEL guards" and "CEL-like guard", misleading operators into writing full CEL expressions that are never evaluated.
3. The `IRInterpreterWorkflow` Temporal workflow is rated a production-incident generator (review C4, R3) because silent fail-open can drive a workflow into a terminal state it should never reach.

**Definition of done:**
- Any guard expression not parseable by the evaluator fails-closed (returns `false`) instead of fail-open.
- The evaluator correctly handles at minimum: `ctx.<key> == "value"`, `ctx.<key> != "value"`, and the full `cel-go` expression set (if Option A chosen).
- All existing `==` / `!=` guards in `spec/workflows/examples/*.yaml` pass without change.
- Documentation and code comments contain no claims of "CEL" semantics unless `cel-go` is integrated.
- A fuzz test seed file is committed (execution deferred to M7.C #469, but seed is committed now).
- `GOWORK=off go test ./... -race` passes in `services/engine-adapter/`.

---

## E — Entities

- **`evalGuard(expr string, ctx map[string]string) bool`** — current function at `interpreter.go:168–187`; 24-line string parser; fail-open on unknown expressions.
- **`resolveOperand(expr string, ctx map[string]string) string`** — operand resolver called by `evalGuard`; handles `ctx.<key>` references.
- **`IRInterpreterWorkflow`** — Temporal workflow that calls `evalGuard` at `interpreter.go:157`; must remain deterministic (ADR-015, Temporal contract).
- **`github.com/google/cel-go`** — Go CEL implementation; evaluates Common Expression Language expressions. Deterministic — safe inside Temporal workflow boundary.
- **`cel.Program`** — compiled CEL expression; computed once at workflow invocation time (not per-step) to remain deterministic.
- **`interpreter_test.go`** — existing test file in `services/engine-adapter/internal/domain/`; extended in O2.
- **Fuzz seed** — `services/engine-adapter/internal/domain/testdata/fuzz/FuzzEvalGuard/` — empty input corpus committed as the seed.

---

## A — Approach

**What we WILL do (Option A — cel-go integration, recommended):**
- Replace the 24-line `evalGuard` string parser with a `cel-go`-based evaluator (~80 LOC including tests).
- The `cel.Environment` is created once at process startup (or lazily on first call). `cel.Program` is compiled per unique expression string and cached in a `sync.Map` to avoid recompilation on repeated evaluations.
- Fail-closed on `cel.Compile` or `cel.Program.Eval` error (return `false`, log the error at WARN level).
- The `ctx` map is passed to CEL as a `map<string, string>` variable named `ctx`; existing `ctx.<key>` guard expressions work unchanged.
- Remove all "CEL-like" / "CEL" comments and replace with "cel-go (github.com/google/cel-go)".
- Commit fuzz seed under `testdata/fuzz/FuzzEvalGuard/`.

**What we WON'T do:**
- Run full fuzz testing in CI (deferred to M7.C #469 — fuzz testing is expensive and belongs in a dedicated test-rigor pass).
- Change the `evalGuard` function signature (callers remain unchanged).
- Add persistence, event publishing, or any infrastructure changes.
- Touch any proto files (ADR-001).

**Why not Option B (rename + fail-closed)?** Renaming `evalGuard` → `evalSimpleEquality` and flipping the default to `return false` fixes the immediate correctness bug but leaves operators with no path to richer conditions. Documentation would need to say "simple equality only" — which is a regression from the advertised "CEL guards" and blocks M6 use cases. `cel-go` adds ~80 LOC and zero new dependencies beyond the cel-go module; the correctness and expressiveness gain justifies the choice.

**ADR references:**
- ADR-001: No proto changes in this epic.
- ADR-015: Pluggable workflow engines — `IRInterpreterWorkflow` must remain Temporal-deterministic; `cel.Program.Eval` is pure-function and deterministic.
- ADR-016: Layered testing — domain coverage target ≥ 90% on `internal/domain/` post-fix.
- ADR-017: `GOWORK=off go test ./... -race` in `services/engine-adapter/`.

---

## S — Structure

```
services/engine-adapter/
└── internal/
    └── domain/
        ├── interpreter.go              ← replace evalGuard body (O1)
        ├── interpreter_test.go         ← extend with cel-go scenarios (O2)
        └── testdata/
            └── fuzz/
                └── FuzzEvalGuard/      ← empty seed corpus (O2)

go.mod (services/engine-adapter/)      ← add github.com/google/cel-go (O1)
go.sum                                 ← updated (O1)
go.work.sum                            ← updated via go work sync (O1)
```

**Modified docs (O3):**
- `services/engine-adapter/AGENTS.md` — remove "CEL-like" language; state "cel-go full CEL".
- Any `.yaml` example files that carry guard comments referencing "CEL" (if any).

---

## O — Operations

1. **[#538]** `fix(engine-adapter)`: Integrate `cel-go` as the guard evaluator — replace `evalGuard` body; add `cel.Environment` + `sync.Map` expression cache; fail-closed on error; add `github.com/google/cel-go` to `go.mod`; run `go work sync`.

2. **[#539]** `test(engine-adapter)`: Guard evaluator test suite + fuzz seed — extend `interpreter_test.go` with fail-closed scenarios, CEL boolean expressions, type-mismatch cases, and error propagation. Commit fuzz corpus seed directory. Domain coverage ≥ 90%.

3. **[#540]** `docs(engine-adapter)`: Remove CEL misrepresentation — update `services/engine-adapter/AGENTS.md` and any inline comments that claim "CEL-like" semantics; replace with accurate "cel-go (github.com/google/cel-go)" references.

---

## N — Norms

- `fix:` for O1; `test:` for O2; `docs:` for O3.
- `GOWORK=off go test ./... -race` in `services/engine-adapter/` after every step.
- Domain coverage target ≥ 90% on `internal/domain/`.
- No `_ = err` suppression — cel-go errors logged at WARN and returned as fail-closed `false`.
- `cel.Program.Eval` must be called inside a function that has no side effects — required for Temporal determinism.
- Every commit carries the required trailers per CONTRIBUTING.md §Commit Hygiene.
- PR size ≤ 400 LOC per step.

---

## S — Safeguards

### Context Security

- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards

- Never use `cel.Program.Eval` with user-controlled data that bypasses the `ctx map<string, string>` variable binding — all runtime data enters CEL only through the typed `ctx` variable (prevents expression injection).
- Never evaluate guard expressions in a goroutine with a separate execution context — Temporal determinism requires evaluation inside the workflow boundary, in the main goroutine.
- Never skip the fuzz seed commit (O2) — the seed is low-cost; deferring it to M7.C is acceptable but the empty corpus must be committed to mark the boundary.
- Never modify `evalGuard` signature — callers in `interpreter.go:157` and tests are stable references.
