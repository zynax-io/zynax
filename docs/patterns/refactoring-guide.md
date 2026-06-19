<!-- SPDX-License-Identifier: Apache-2.0 -->

# Refactoring Guide for AI Agents

> A decision tree for the four core refactoring operations, framed for AI-assisted work in
> this codebase. The goal is **consistent, auditable** refactoring: the same trigger always
> produces the same decision, and every decision is defensible in a Canvas Approach section.
>
> Refactoring does **not** require a Canvas upfront (that gate is `feat:`-only, ADR-019). But
> if a session changes **> 50 lines**, end it with `/lib:spdd-sync <canvas>` so the Canvas
> reflects the post-refactor state — see [SPDD Integration](#spdd-integration) below.

---

## Extract Function (Go / Python)

**Extract when:**
- function body > 30 lines (Go) / > 20 lines (Python), **or**
- a comment is needed to explain a block — the comment becomes the function name, **or**
- the same block is reused in 2+ places.

**Do not extract:**
- single-line operations,
- expressions that only make sense in their surrounding context,
- test helpers used exactly once (inlining keeps the test readable).

**Zynax example:** in `services/workflow-compiler/internal/domain/validator/`, the structural
and semantic passes are separate functions — each is a named domain concept reused by both
`CompileWorkflow` and `ValidateManifest`.

---

## Inline Function

**Inline when:**
- the function body is shorter than its call site, **or**
- the function is called from exactly one place and extraction adds no clarity.

**Do not inline:**
- functions called from tests — test readability outweighs production brevity,
- domain functions named after a business concept — the name is the documentation.

**Zynax example:** a one-line `isTerminal(state)` helper named after a domain concept stays a
function even though its body is a single comparison.

---

## Move Function / Method

**Move when:**
- a function uses more data from another module than from its own, **or**
- it is called from only one other module.

**Do not move across a layer boundary.** `domain ← api` or `domain ← infrastructure` violates
the hexagonal dependency rule — domain code never imports api/infrastructure (root
[AGENTS.md §The Three-Layer Separation](../../AGENTS.md#the-three-layer-separation-non-negotiable)).
Cross-service moves go over gRPC, never a shared package (ADR-001, ADR-008).

**Zynax example:** serialization helpers that read only `WorkflowIR` proto fields belong in
`domain/serializer/`, not in the gRPC `api/handler` that calls them.

---

## Replace Conditional with Polymorphism

**Replace when:**
- the same `switch` / `if-else` on a type appears in 3+ places, **or**
- adding a new type forces edits to multiple switch statements.

**Do not replace:**
- simple 2-branch conditions,
- conditions on error types — use `errors.Is` / `errors.As`, not a type switch.

**Zynax example:** engine selection sits behind the `WorkflowEngine` interface (ADR-015), not a
`switch engineName` repeated at every dispatch site — a new engine is a new implementation, not
a new case.

---

## SPDD Integration

Refactoring is exempt from the up-front Canvas gate, but the Canvas must still reflect reality:

- A session that changes **> 50 lines** ends with `/lib:spdd-sync <canvas-path>`, which updates
  the Canvas **Structure** (S) and **Operations** (O) sections to the post-refactor layout.
- If the refactor changed *requirements* rather than just structure, run
  `/lib:spdd-prompt-update` first (Canvas before code — ADR-019), then `/lib:spdd-sync`.

The Canvas **Safeguards** section is the best source of which invariants a refactor must
preserve — read it before moving or extracting anything.

---

*See also: [go-service-patterns.md](go-service-patterns.md) · [spdd-guide.md](spdd-guide.md) ·
root [AGENTS.md](../../AGENTS.md) AI anti-patterns table.*
