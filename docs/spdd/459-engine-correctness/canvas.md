<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M5.B Engine Correctness Hardening

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #459
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-18
**Status:** Implemented

**Child issues:** #475 (resolveTemplate determinism) · #476 (guard parser) · #477 (compiler error contract) · #478 (SSE WriteTimeout)

---

## R — Requirements

**Problem:** The 2026-05 external architectural review identifies four correctness bugs in the implemented services that are production-incident generators:

1. `interpreter.go:204-209` — `resolveTemplate` iterates a `map[string]string` with randomised Go iteration order inside `IRInterpreterWorkflow`. Temporal workflows must be deterministic; non-determinism causes a `nondeterminism panic` making the workflow unrecoverable on worker restart. (Review C3, R2)
2. `interpreter.go:178-198` — `evalGuard` supports only `==` / `!=` but is documented as "CEL guards". Unrecognised expressions return `true` silently (fail-open), causing workflows to advance past gates they should be blocked at. (Review C4, R3)
3. `workflow-compiler/internal/api/server.go:46-60` — `CompileWorkflow` returns only the first error in gRPC status metadata, discarding the `Errors []CompilationError` list the proto contract promises. (Review C5, R1)
4. `api-gateway/cmd/api-gateway/main.go:64` — `WriteTimeout: 30s` kills every SSE log stream at 30 seconds. `zynax logs` is non-functional for workflows longer than 30 s. (Review H4, R5)

**Definition of done:**
- `resolveTemplate` called 10 times with a 5-key context map produces byte-identical output every time.
- Unrecognised guard expressions fail-closed (return `false`) OR are evaluated correctly via `cel-go`.
- `CompileWorkflow` with 3 distinct validation errors returns all 3 in `response.Errors`.
- `zynax logs wf-<hex>` stays connected for >60 s without server-side disconnect.

---

## E — Entities

- **`IRInterpreterWorkflow`** — Temporal workflow registered in engine-adapter; must be deterministic by Temporal's contract.
- **`resolveTemplate(template string, ctx map[string]string) []byte`** — context-variable substitution function at `interpreter.go:204-209`; called inside the Temporal workflow boundary.
- **`evalGuard(expr string, ctx map[string]string) bool`** — guard expression evaluator at `interpreter.go:178-198`; determines whether a state transition fires.
- **`cel-go`** — `github.com/google/cel-go` — the official Google CEL implementation; the recommended replacement for the bespoke parser.
- **`CompileWorkflow` RPC** — `WorkflowCompilerService.CompileWorkflow` at `workflow-compiler/internal/api/server.go`; currently returns single-error status instead of full `Errors` list.
- **`CompilationError`** — proto message carrying `code`, `message`, `state_name`, `line`, `column`; should be returned as a repeated field in `CompileWorkflowResponse`.
- **SSE handler** — `api-gateway/internal/api/handler.go:117-150`; server-sent event stream for `zynax logs`.
- **`http.ResponseController`** — Go 1.20+ stdlib type that allows per-handler write deadline override.

---

## A — Approach

**What we WILL do:**
- Fix `resolveTemplate` and `mergePayload` with sorted-key iteration (XS, ~10 LOC).
- For the guard parser: integrate `cel-go` (Option A, recommended) OR rename to `evalSimpleEquality` + flip default to `return false` + update all docs (Option B). Decision documented in #476 before implementation.
- Fix `CompileWorkflow` error reporting to populate `response.Errors` and return `codes.OK` with errors in the response body (per proto contract).
- Fix SSE WriteTimeout via `http.NewResponseController(w).SetWriteDeadline(time.Time{})` in the streaming handler.

**What we WON'T do:**
- Implement full CEL in this EPIC if Option B is chosen — that is a separate `feat:` issue.
- Change any proto field numbers or remove any proto fields (backward-compat rule, ADR-001).
- Touch the task-broker or agent-registry (M5.C).

**ADR references:**
- ADR-001: gRPC as inter-service protocol — backward-compat ordinals must not change.
- ADR-015: Pluggable workflow engines — `IRInterpreterWorkflow` must remain deterministic.
- ADR-016: Layered testing — regression tests required for each fix.

---

## S — Structure

**Files touched:**
- `services/engine-adapter/internal/domain/interpreter.go` — fixes for B1 (determinism) and B2 (guard parser)
- `services/engine-adapter/internal/domain/interpreter_test.go` — regression tests
- `services/workflow-compiler/internal/api/server.go` — fix for B4 (error contract)
- `services/workflow-compiler/internal/api/server_test.go` — test for multi-error return
- `api-gateway/cmd/api-gateway/main.go` or `api-gateway/internal/api/handler.go` — fix for B3 (SSE timeout)
- `protos/tests/features/` — BDD stubs updated to reflect corrected compiler error semantics

**No proto field changes. No new services.**

---

## O — Operations

1. **[#475]** Fix `resolveTemplate` and `mergePayload` map-iteration: sort keys before iterating. Add regression test asserting byte-identical output across 10 runs.
2. **[#476]** Guard parser: decide and implement Option A (`cel-go`) or Option B (rename + fail-closed). Update all documentation references to "CEL guards" accordingly.
3. **[#477]** Fix `CompileWorkflow` error contract: return structured `CompilationError` list in `response.Errors`; return `codes.OK`. Update BDD stub.
4. **[#478]** Fix SSE WriteTimeout: apply `http.NewResponseController` write-deadline override in the streaming handler. Verify connection stays open >60 s.

---

## N — Norms

- `fix:` PR type for all four bugs.
- `GOWORK=off go test ./... -race` required in every service directory touched.
- ≥90% domain coverage maintained on `internal/domain/` after changes.
- All errors wrapped with `fmt.Errorf("... : %w", err)` — no bare error returns.
- No `_ = f()` error discards — all errors must be handled or explicitly logged.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed

### Feature Safeguards
- Never change Temporal workflow function signatures or registration names — that breaks replay determinism.
- Never change proto field numbers or remove fields (ADR-001).
- Never make the guard parser fail-open by default — fail-closed is the correct and safe default.
- The `CompileWorkflow` fix must not break the `ValidateManifest` RPC (separate code path, same file).
