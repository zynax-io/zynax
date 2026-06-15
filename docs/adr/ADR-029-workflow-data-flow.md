# ADR-029: Workflow data-flow semantics (output/input bindings)

**Status:** Proposed  **Date:** 2026-06-15
**Related:** ADR-012 (Workflow IR), ADR-014 (event-driven state machine), ADR-015 (pluggable engines)

---

## Context

Workflows are event-driven state machines (ADR-014) compiled to a WorkflowIR (ADR-012). Today the
compiler **rejects** `output:` on actions (`"output: which is not yet implemented; … upgrade to M7+"`),
so no state can pass data to a later state. Every real pipeline — research→summarize, build→test→deploy
— is inexpressible. M7's central goal ("make Zynax usable for real workflows") requires a data-flow
model, but an over-rich expression language would be a large, hard-to-reverse one-way door on the proto
contract. This ADR fixes the **minimal** model.

## Decision

We will add a **minimal, additive** data-flow model to the WorkflowIR:

1. **`output_bindings`** on an action — named outputs an action publishes into a workflow-scoped data context.
2. **`input_bindings`** on an action — inputs resolved from the data context by **literal value** or
   **JSON-path reference** (e.g. `$.states.search.output.results`).
3. A **workflow-run-scoped `WorkflowDataContext`** owned by the interpreter; written by outputs, read by inputs;
   it does **not** persist beyond the run.
4. Bindings are **literal or path references only** — **no expression/transform language** in M7.

Proto fields are additive; manifests without bindings compile unchanged. `buf breaking` must pass.

## Rationale

| Option | Assessment |
|--------|------------|
| Minimal literal/path bindings (chosen) | ✅ Smallest contract that unblocks real workflows; additive; reversible-ish |
| Full expression language (CEL/JSONata) | ✗ Deferred to M-dx — large surface, security review burden, premature |
| Implicit shared state between states | ✗ Rejected — violates explicit-scoping; collides with ADR-008 spirit (no shared mutable state) |

## Consequences

- **Positive:** real multi-step workflows become expressible; the long-standing "not implemented"
  rejection is lifted; trace/log correlation (EPIC C/O) has real data to follow.
- **Negative / trade-off:** path references are stringly-typed; validation must catch unresolved refs
  at compile time (clear `COMPILATION_ERROR` with line number).
- **Neutral / follow-up:** transforms/filters and typed schemas for the data context are deferred to
  M-dx; this ADR will be revisited if/when an expression language is added.
