# REASONS Canvas — Workflow-level output capture, return & display

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/lib:spdd-security-review <path>` before committing.

**Issue:** #1529 (epic M7.U)
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-30
**Status:** Aligned

---

## R — Requirements

**Problem.** After a workflow COMPLETES, the user cannot get its result back. `zynax result <run>`
hard-errors `no result payload for run X` even on a successful run, because nothing captures a
**workflow-level** output. The step-to-step data-flow delivered by EPIC #1167 lives only inside the
run: the run-scoped `WorkflowDataContext` is discarded at the terminal state
([interpreter.go:72-76](../../../services/engine-adapter/internal/domain/interpreter.go#L72-L76)),
there is no `outputs:` declaration surface, no carrier field on `WorkflowRun`, and no gateway read
path ([ARCHITECTURE.md:540](../../../ARCHITECTURE.md#L540)).

**Definition of done.**
- A workflow can declare outputs on its terminal state; the engine resolves them from the run-scoped
  data context **before** it is discarded and returns them as the workflow result.
- `zynax result <run>` prints the declared outputs and exits 0; a COMPLETED run with no declared
  output exits 0 with a graceful note (never a hard error).
- `GET /api/v1/workflows/{id}/outputs` returns the outputs JSON (`{}` when none, 404 unknown).
- The same result is returned identically whether the run executed on Temporal or Argo.
- Outputs are size-bounded at capture and control-char/ANSI-sanitized on every render surface.
- An interim runbook documents the reliable workaround until `zynax result` is fixed.

---

## E — Entities

- **WorkflowDataContext** — existing run-scoped key/value store (`states.<stateID>.output.<key>`);
  gains no new persistence, only a resolve-at-terminal read before discard.
- **Terminal StateIR.outputs** — new declaration: a map of result name → `$.states.<state>.output.<key>`
  reference (reusing the ADR-029 grammar) or literal, valid only on terminal states.
- **WorkflowRun.outputs** — new carrier: the resolved result map returned by `GetWorkflowStatus`.
- **Terminal WorkflowEvent payload** — typed JSON `{"completion":…, "outputs":{…}}` over the existing
  opaque `bytes payload`; `outputs` namespaced so it does not collide with the task-broker
  `completion` shape parsed by `CompletionText`.
- **Outputs read path** — `GET /api/v1/workflows/{id}/outputs` (api-gateway) → `GetWorkflowStatus`.

```
author manifest (terminal outputs:)
   → workflow-compiler  → StateIR.outputs
   → engine-adapter     → resolve at terminal from WorkflowDataContext → WorkflowRun.outputs (Temporal result)
   → api-gateway        → GET /workflows/{id}/outputs
   → cli                → zynax result
```

---

## A — Approach

**We will:**
- Decide the output contract in **ADR-042** (placement, carrier, value typing, empty-output contract,
  output safety) before any code (O.2).
- Declare outputs on the **terminal StateIR** (additive `map<string,string> outputs = 5`), reusing the
  ADR-029 `$.states.<state>.output.<key>` reference grammar — **no new expression language**.
- Carry the resolved result as the **Temporal workflow result** surfaced on `WorkflowRun.outputs`
  (additive field 12) — **no new persistence store** (upholds ADR-029 §2/§3, ADR-008).
- Resolve outputs at the terminal state **before** the data context is discarded.
- Expose a **dedicated** read route `GET /workflows/{id}/outputs` (the contract the existing
  `automation/tests/platform_client.py` calls), and make `zynax result` read it.
- Ship the interim runbook + graceful-empty `zynax result` first, so users are unblocked on day 0.
- Enforce **output safety**: per-key + total size bounds at capture; C0/C1 control-char + ANSI-escape
  sanitization before any TTY/SSE render.

**We will NOT:**
- Re-implement step-to-step bindings (EPIC #1167, CLOSED) or log streaming (EPIC #468, CLOSED).
- Add a database for outputs, or persist beyond Temporal retention (deferred).
- Introduce rich nested output typing — values are `map<string,string>` (JSON strings the consumer
  parses); rich typing deferred per ADR-042.
- Close #1103 gaps #2 (guards) or #3 (capability providers) — only gap #4 (gateway outputs read path).

**Positioning fit (user-facing).** This is heavily user-facing (runbook, `zynax result` help, error
strings). All copy leads with the **engine-portability wedge**: "see your declared workflow result
from one command — the same whether the run executed on Temporal or Argo." It must NOT use the generic
"control plane for AI agents" framing. See [docs/product/positioning.md](../../product/positioning.md).

**Governing ADRs:** ADR-042 (workflow-level output capture & return — new), ADR-029 (data-flow
semantics & scoping), ADR-008 (no shared DB), ADR-012 (additive proto), ADR-016 (.feature first),
ADR-019 (canvas before code).

---

## S — Structure (first S)

```
docs/adr/ADR-042-workflow-level-output-capture.md   ← contract (O.2)
docs/runbooks/see-workflow-result.md                ← interim workaround (O.1)
protos/zynax/v1/workflow_compiler.proto             ← StateIR.outputs = 5 (O.5)
protos/zynax/v1/engine_adapter.proto                ← WorkflowRun.outputs = 12, event payload (O.5)
protos/tests/features/*.feature                     ← BDD contract (O.4)
spec/schemas/workflow.schema.json                   ← terminal outputs: schema (O.6)
services/workflow-compiler/internal/domain/manifest.go   ← parse + validate (O.6)
services/engine-adapter/internal/domain/interpreter.go   ← capture-before-discard (O.7)
services/engine-adapter/internal/domain/datacontext.go   ← size bounds at capture (O.7)
services/api-gateway/.../handler.go                 ← GET /workflows/{id}/outputs (O.8)
cmd/zynax/cmd/result.go + client/gateway.go         ← read + print outputs (O.3, O.9)
spec/workflows/examples/{hello-world,code-review}.yaml   ← declare outputs (O.10)
automation/tests/test_platform_readiness.py         ← gated e2e, #1103 gap #4 (O.11)
```

Config env prefix: `ZYNAX_<SERVICE>_` · Engine-agnostic interpreter (Temporal / Argo).

---

## O — Operations

> Each step = one reviewable PR, mapped 1:1 to a story issue.

1. **O.1 (#1530, docs)** — Commit `docs/runbooks/see-workflow-result.md` documenting `zynax logs --follow`,
   re-submit-and-stream, json replay, and `zynax status`. Verified: runbook commands work against the CLI.
2. **O.2 (#1531, ADR)** — Author ADR-042 + register in `docs/adr/INDEX.md`; decide placement / carrier /
   typing / empty-output contract / output-safety. Verified: ADR Accepted; gates all `feat:`.
3. **O.3 (#1532, fix·cli)** — `zynax result` exits 0 with a graceful note on COMPLETED-empty; hard error
   kept for FAILED/CANCELLED. Verified: unit tests for COMPLETED-empty / FAILED / completion-present.
4. **O.4 (#1533, test·protos)** — BDD `.feature` scenarios (engine_adapter + workflow_compiler), RED
   before impl. Verified: scenarios committed and red; `protos/tests` compiles.
5. **O.5 (#1534, feat·protos)** — Add `StateIR.outputs=5`, `WorkflowRun.outputs=12`, document terminal
   event payload JSON; regenerate stubs. Verified: `buf breaking` green; stubs committed.
6. **O.6 (#1535, feat·compiler)** — Schema + `manifest.go` parse/validate terminal `outputs:` →
   `StateIR.outputs`; dangling/non-terminal ref → COMPILATION_ERROR + line. Verified: `make validate-spec`;
   domain cov ≥ 90%; O.4 compiler scenario green.
7. **O.7 (#1536, feat·engine)** — Resolve `StateIR.outputs` at the terminal state before discard; return as
   Temporal result onto `WorkflowRun.outputs`; widen `EventPublisher.Publish` with one additive arg; enforce
   size bounds. Verified: O.4 engine scenario green; empty=success, unresolved=DataReferenceError; cov ≥ 90%; race.
8. **O.8 (#1537, feat·gateway)** — `GET /api/v1/workflows/{id}/outputs` ({} / 404); outputs on SSE terminal
   event. Verified: `handler_test` populated/empty/404; `platform_client.get_outputs()` resolves.
9. **O.9 (#1538, feat·cli)** — `zynax result` reads `/outputs`, prints declared outputs, falls back to
   `CompletionText`, sanitizes control chars; wedge-first help. Verified: CLI tests outputs/fallback/empty/FAILED.
10. **O.10 (#1539, feat·spec)** — Declare terminal `outputs:` on hello-world + code-review; update comments +
    runbook. Verified: examples compile/validate; existing 9 examples unchanged; `zynax result` prints output.
11. **O.11 (#1540, test·engine)** — Gated e2e (`ZYNAX_PLATFORM_E2E=1`) proving apply→COMPLETED→`/outputs`;
    reconcile #1103 gap #4; document gaps #2/#3 still gating; mark ARCHITECTURE.md gap #4 closed. Verified:
    e2e run twice end-to-end (runtime smoke, not CI-green alone).

---

## N — Norms

- Commit hygiene: every commit carries `Signed-off-by:` + `Assisted-by: Claude/<model>` (never `Co-Authored-By` for AI).
- BDD: `.feature` committed before any gRPC-boundary implementation (ADR-016) — O.4 precedes O.5–O.7.
- Proto: additive only (new field numbers, never renumber/remove); `buf breaking` is a CI gate (ADR-012).
- `GOWORK=off` for all `go` / `go test` in `services/*`, `cmd/zynax/`, `protos/tests/` (ADR-017).
- Unit coverage ≥ 90% on `internal/domain` (ADR-016 tiers).
- One commit per logical change; one PR per story; conventional commit types only.
- Outputs are **untrusted** input — treat as such on every render surface (CLI, SSE, logs, gateway JSON).
- Runtime smoke before claiming done: run the COMPLETED path twice (persistence/2nd-run discipline).

---

## S — Safeguards (second S)

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no non-public email addresses
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in E section are public-safe abstractions
- [x] `/lib:spdd-security-review` passed — result: PASS

### Feature Safeguards

- Never persist outputs to a shared database or any store beyond the Temporal run result (ADR-008, ADR-029 §2/§3).
- Never hardcode an engine name — output capture lives in the engine-agnostic interpreter (ADR-015).
- Never renumber/remove a proto field — outputs are strictly additive (`StateIR.outputs=5`, `WorkflowRun.outputs=12`) (ADR-012).
- Never render an unbounded or unsanitized output to a TTY/SSE — enforce size bounds at capture and strip C0/C1/ANSI before display.
- Never hard-error a COMPLETED run for having no declared output — empty is success (`{}`).
- Never resolve outputs with non-deterministic I/O inside the Temporal workflow function (replay-safety).
- Never import another service's `internal/` — cross-service via gRPC only (ADR-008).
