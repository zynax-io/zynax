<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-029 — Workflow Data-Flow Semantics (output/input bindings)

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-16 |
| **Deciders** | Oscar Gómez Manresa |
| **Scope** | `WorkflowIR` proto (`protos/zynax/v1/workflow_compiler.proto`), `services/workflow-compiler/`, `services/engine-adapter/` — M7 EPIC W (#1167) |
| **Related** | ADR-012 (Workflow IR), ADR-014 (event-driven state machine), ADR-015 (pluggable engines), ADR-008 (no shared databases / no shared mutable state) |

---

## Context

Workflows are event-driven state machines (ADR-014) compiled to an engine-agnostic
WorkflowIR (ADR-012). Today the compiler **rejects** `output:` on actions —
`"output: which is not yet implemented; … upgrade to M7+"` — so no state can pass
data to a later state. Every real pipeline (research → summarize, build → test →
deploy) is therefore inexpressible.

M7's central goal is to make Zynax usable for real workflows, which requires a
data-flow model. The risk is over-reach: an expression/transform language baked
into the proto contract would be a large, security-sensitive, hard-to-reverse
one-way door. Because this decision shapes the proto contract that EPIC W steps
W.2–W.5 build on, it must be fixed **before** any implementation lands (this
story gates W.2). This ADR records the **minimal** binding model so the contract
is stable first.

## Decision

We adopt a **minimal, additive** data-flow model on the WorkflowIR.

### 1. Binding syntax — literal values and JSON-path references only

An action declares two binding maps:

- **`output_bindings`** — named outputs an action *publishes* into the
  workflow-scoped data context. Each entry maps a context key to a source path
  within the action's result (e.g. `results` ← the action output payload).
- **`input_bindings`** — inputs an action *consumes*. Each entry resolves to a
  value by exactly one of two forms:
  - a **literal** value (string/number/bool), or
  - a **JSON-path reference** into the data context, written as a dotted path
    rooted at `$.states.<state>.output.<key>` —
    e.g. `$.states.search.output.results`.

There is **no expression, transform, filter, templating, or arithmetic
syntax** in M7. A reference either resolves to a stored value verbatim or it is
a compile-time error. This is the entire surface.

### 2. Scoping model — one run-scoped data context, explicitly read/written

- A single **`WorkflowDataContext`** exists per workflow **run**. It is a
  key/value store owned by the interpreter (`IRInterpreterWorkflow` in
  engine-adapter, behind the `WorkflowEngine` interface per ADR-015).
- The context is **written only** by `output_bindings` and **read only** by
  `input_bindings` — there is no implicit/global mutable state and no ambient
  read of one state's locals by another.
- The context is **strictly run-scoped**: it does not leak across workflow runs,
  workflows, or namespaces, and it does **not persist beyond the run** (no
  durable data store in M7). This keeps the model aligned with ADR-008's "no
  shared mutable state" spirit.
- **Compile-time validation:** every `input_bindings` JSON-path reference must
  resolve to a key declared by an upstream state's `output_bindings`. An
  unresolved reference is a `COMPILATION_ERROR` carrying the manifest line
  number — failure is loud and early, not at run time.

### 3. Non-goals (explicit scope guard for M7)

- **No expression / transform language** (CEL, JSONata, math, string templating,
  filters) — deferred to a later milestone (M-dx).
- **No cross-workflow or cross-namespace data sharing.**
- **No persistence** of the data context beyond a single run.
- **No typed schema** for context values in M7 — references are stringly-typed
  paths; a typed data-context schema is future work.

### 4. `buf breaking` implications — additive only

The new IR fields (`output_bindings`, `input_bindings`, and the data-context
representation) are added as **new fields with new field numbers** on existing
messages. They are purely **additive**:

- Manifests without bindings compile unchanged (the maps are simply empty).
- No existing field is renamed, renumbered, retyped, or removed.
- Therefore `buf breaking` (a CI gate per ADR-012) stays **green**; the proto
  contract remains backward-compatible. W.2 must keep the change additive — any
  non-additive edit is blocked.

## Rationale

| Option | Assessment |
|--------|------------|
| **A — Minimal literal + JSON-path bindings (chosen)** | ✅ Smallest contract that unblocks real multi-step workflows; purely additive to the proto, so `buf breaking` stays green; references validated at compile time; reversible-enough — an expression language can be layered on later as a strict superset. |
| **B — Full expression language (CEL / JSONata) now** | ✗ Rejected for M7 — large API and security-review surface, premature before real usage data; can be added later without breaking A. Deferred to M-dx. |
| **C — Implicit shared state between states** | ✗ Rejected — violates explicit read/write scoping and collides with ADR-008's no-shared-mutable-state spirit; makes data-flow untraceable for EPIC C/O correlation. |

## Consequences

### Positive

- Real multi-step workflows (research → summarize, build → test → deploy) become
  expressible; the long-standing `output:` rejection is lifted in W.3.
- The proto change is additive, so no consumer breaks and `buf breaking` passes.
- Trace/log correlation (EPIC C/O) gains concrete data to follow through a run.
- The minimal surface keeps the security-review and validation burden small.

### Negative / trade-offs

- JSON-path references are **stringly-typed**; correctness depends on compile-time
  reference-resolution validation rather than the type system.
- Without transforms, some shaping that an expression language would do inline
  must instead be done inside an agent/capability or deferred to M-dx.

### Neutral / follow-up required

| Action | Tracking |
|--------|---------|
| Add `output_bindings`/`input_bindings` IR fields + `.feature` (additive; `buf breaking` green) | EPIC W step W.2 (#1176) |
| Compile + validate bindings; lift the `output:` rejection | EPIC W step W.3 (#1177) |
| Thread run-scoped `WorkflowDataContext` through the interpreter | EPIC W step W.4 (#1178) |
| Prove real workflows run end-to-end | EPIC W step W.5 (#1179) |
| Revisit if/when an expression/transform language is added | M-dx (future) |
