<!-- SPDX-License-Identifier: Apache-2.0 -->
# ADR-042: Workflow-level output capture & return — terminal-state outputs surfaced as the workflow result

**Status:** Accepted  **Date:** 2026-06-30
**Related:** ADR-029 (workflow data-flow semantics & scoping — this ADR extends it to the workflow boundary), ADR-008 (no shared database — service isolation), ADR-012 (additive proto evolution / `buf breaking`), ADR-015 (pluggable workflow engines — capture lives in the engine-agnostic interpreter), ADR-016 (`.feature` before implementation), ADR-019 (REASONS Canvas before code). Implements EPIC #1529 (M7.U); canvas `docs/spdd/1529-workflow-output-capture/canvas.md`.

---

## Context

EPIC #1167 (ADR-029) delivered **step-to-step** data-flow: an action publishes named outputs into a
run-scoped `WorkflowDataContext` (`states.<stateID>.output.<key>`) and a later action consumes them via
`$.states.<state>.output.<key>` references. That data-flow lives entirely *inside* a run.

There is no **workflow-level** output. The consequences are user-visible and code-verified:

- `services/engine-adapter/internal/domain/interpreter.go` discards the run-scoped `WorkflowDataContext`
  at the terminal state (the terminal branch publishes `zynax.workflow.completed` and returns `nil`); the
  accumulated outputs are never read back out.
- There is no declaration surface: `spec/schemas/workflow.schema.json` states are
  `additionalProperties:false` with no `outputs`; neither `WorkflowIR` nor `StateIR` carries an `outputs`
  field; only `ActionIR.output_bindings` exists.
- There is no carrier: `engine_adapter.proto` `WorkflowRun` has no field for final outputs, and
  `WorkflowEvent.payload` is opaque `bytes`.
- There is no read path: `ARCHITECTURE.md` records "api-gateway has no workflow-outputs or decision-log
  read path" as an open platform gap (the #1103 platform-readiness gap #4).
- `cmd/zynax/cmd/result.go` only scrapes the last action's `completion` text and **hard-errors**
  `no result payload for run X` — even on a `WORKFLOW_STATUS_COMPLETED` run that simply declared no output
  (e.g. echo / hello-world).

The net effect: a user runs a workflow, it COMPLETES, and `zynax result <run>` returns empty or errors.
This breaks the M7 ("Usable Workflows") goal narrative, whose final beat is *seeing the result*.

These are one-way-door decisions (proto shape, durability source, value typing) that another engineer
would reverse without the rationale, so they are fixed here before any implementation (ADR-019).

---

## Decision

**We will capture a workflow's declared outputs at the terminal state, carry them as the workflow result,
expose them over a dedicated read path, and display them — additively, with no new persistence store.**

1. **Declaration on the terminal `StateIR`.** A terminal state may declare an `outputs:` map. Each value
   is either a literal or an ADR-029 `$.states.<state>.output.<key>` reference — **no new expression
   language**. Compiled into a new additive proto field `map<string,string> outputs = 5` on `StateIR`.
   *(Rejected: a top-level `WorkflowIR.outputs` — it divorces resolution from state scope and conflates
   the compiled program envelope with captured values.)*
2. **Capture before discard.** At the terminal state, the engine-agnostic interpreter resolves
   `StateIR.outputs` against the still-live run-scoped `WorkflowDataContext` (reusing the ADR-029
   `ResolveInputs` grammar and scope isolation) **before** the context is discarded.
3. **Carrier = the workflow result; no new persistence.** Resolved outputs travel as the **Temporal
   workflow result** (durable in Temporal's own event log, queryable via `GetWorkflow().Get`) and are
   surfaced on a new additive field `map<string,string> outputs = 12` on `WorkflowRun`, returned by
   `GetWorkflowStatus`. The terminal `WorkflowEvent` reuses its existing opaque `bytes payload = 7`
   carrying a typed JSON shape `{"completion": …, "outputs": { … }}`, with `outputs` **namespaced** so it
   does not collide with the task-broker `completion`/`result_payload` shape parsed by `CompletionText`.
   **No database and no store beyond the Temporal run result** — ADR-029 §2/§3 (run-scoped, no durable
   data store) and ADR-008 (no shared DB) are upheld.
4. **Value typing = `map<string,string>`.** Values may be JSON strings the consumer parses; richer nested
   typing is **deferred**. *(Rejected for now: `google.protobuf.Struct` — larger contract, diverges from
   ADR-029's stringly-typed scalar stance.)*
5. **Empty outputs is success.** A COMPLETED run that declared no outputs returns `{}` and exits 0 — never
   a hard error. The current `zynax result` hard-error on empty is the user-facing bug being removed.
6. **Output safety — outputs are untrusted.** Workflow/capability outputs are attacker-influenced and are
   rendered to terminal, SSE, logs, and gateway JSON. Therefore: **per-key and total size bounds are
   enforced at capture** (a typed error on overflow, never a silent truncate), and **C0/C1 control
   characters and ANSI escape sequences are stripped before any TTY/SSE render**. Outputs are treated as
   untrusted on every render surface.
7. **Dedicated read path.** `GET /api/v1/workflows/{id}/outputs` returns the outputs JSON (`{}` when none,
   `404` for unknown id) — the contract `automation/tests/platform_client.py` already calls. This closes
   #1103 platform gap #4. Gaps #2 (CEL-vs-Go-template guards) and #3 (capability providers) remain out of
   scope and keep the strict platform-readiness e2e gated.

All three proto changes are **new field numbers** (verified next-free: `StateIR.outputs = 5`,
`WorkflowRun.outputs = 12`); nothing is renumbered, removed, or retyped, so `buf breaking` stays green
(ADR-012). Manifests/IRs/runs without `outputs` behave exactly as today (empty map).

---

## Rationale

| Option | Assessment |
|--------|------------|
| **Terminal-`StateIR` declaration + Temporal-result carrier + `map<string,string>` + dedicated `/outputs`** | ✅ **Chosen** — reuses the ADR-029 grammar and scope; additive/backward-compatible; no new infrastructure (Temporal log is the durable record); matches the existing `platform_client.get_outputs()` contract; closes #1103 gap #4. |
| Declare outputs on top-level `WorkflowIR` | ✗ Rejected — needs a second resolution path divorced from state scope; conflates compiled-program envelope with captured result. |
| Persist outputs in Postgres (`WorkflowRun` row) | ✗ Rejected for now — adds storage and an M6-repo write path; violates the no-new-store stance. **Deferred:** revisit only if outputs must outlive Temporal retention. |
| `google.protobuf.Struct` (rich nested typing) | ✗ Rejected for now — larger contract; diverges from ADR-029 stringly-typed scalars. Nested needs are met by JSON-string values the client parses; revisit later. |
| New typed field on `WorkflowEvent` | ✗ Rejected — the existing opaque `payload` carries a namespaced JSON shape with a smaller blast radius. |
| Keep status quo (`zynax result` scrapes last completion) | ✗ Rejected — leaves M7's goal ("see the result") unmet; hard-errors on successful runs. |

---

## Consequences

- **Positive:**
  - `zynax result <run>` returns a workflow's declared result, and `GET /workflows/{id}/outputs` exposes
    it over REST — the M7 "see the result" beat is delivered.
  - The result is captured in the **engine-agnostic interpreter**, so it is returned identically whether
    the run executed on Temporal or Argo (advances the engine-portability wedge).
  - Strictly additive: existing manifests, IRs, runs, and `buf breaking` are unaffected.
  - No new persistence, repo, or service — the Temporal event log is the durable carrier (ADR-008/029 held).
  - Closes #1103 platform-readiness gap #4 (gateway outputs read path).
- **Negative / trade-off:**
  - Durability is bounded by **Temporal retention** — a run queried after its history expires returns no
    outputs. **Mitigation / follow-up:** persist on the `WorkflowRun` row only if a longer window becomes a
    requirement (explicitly deferred).
  - Flat `map<string,string>` forces consumers to `json.loads` values that hold nested JSON; the gateway
    and `platform_client` must error-handle (not panic on) that parse.
  - Size bounds **reject** oversized outputs with a typed error rather than truncating — a workflow that
    emits very large outputs will fail loudly and must be redesigned to emit a reference/handle instead.
- **Neutral / follow-up required:**
  - `.feature` contract committed before implementation (ADR-016); proto fields then compiler, engine,
    gateway, CLI, examples, and a gated e2e — Operations O.4–O.11 of canvas #1529.
  - Runtime smoke before claiming done: exercise a COMPLETED workflow end-to-end (twice) and confirm
    `zynax result` is non-empty — CI-green alone is insufficient.
