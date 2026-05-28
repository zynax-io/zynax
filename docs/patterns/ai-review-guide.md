<!-- SPDX-License-Identifier: Apache-2.0 -->

# AI-Output Review Guide

This guide explains why each item in the "AI-Output Review" section of the PR template matters
and what failure modes to watch for when reviewing AI-assisted PRs.

The `ai-assisted` label triggers the checklist. If the label is absent, skip the checklist.

---

## Why AI output needs a separate review pass

Lint and tests verify mechanical correctness — types match, tests pass. They do not verify that
the code does the right thing in the right context. AI-generated code has characteristic failure
modes that automated gates miss:

- **Plausible but wrong logic**: the code compiles and the tests pass, but the business rule is
  subtly reversed or the edge case is silently skipped.
- **Scope creep**: the AI adds "helpful" refactoring or abstractions beyond the issue scope,
  making the diff harder to review and introducing unintended behaviour.
- **Optimistic error handling**: AI-generated code tends to swallow errors or return zero-values
  instead of propagating failure — silent data loss that tests do not catch.
- **Stale Canvas**: the Canvas was aligned before implementation; the AI diverged from the
  design during generation and did not update the Canvas to match.

---

## Checklist rationale

### Correctness

**Business logic matches the intent** — Read the linked issue, not the diff. Ask: does this code
solve the stated problem, or does it solve a simpler proxy? Common failure: the AI solves the
example in the issue description rather than the general case.

**Edge cases handled** — Look at the acceptance criteria in the issue. Each criterion is a test
scenario. Verify the code handles the boundary, not just the happy path.

**No silent data loss** — Grep for `_ =`, bare `catch`, empty `except`, and `if err != nil {
return }` without logging. Each is a candidate for silent discard. AI code frequently discards
errors at intermediate steps where a human would propagate or log them.

**gRPC status codes** — Verify `NOT_FOUND` is used for missing resources, `INVALID_ARGUMENT` for
bad input, `INTERNAL` only for server faults. AI models conflate these. Incorrect status codes
break client retry logic and monitoring alerts.

### Scope

**Files scoped by Canvas or issue** — Cross-check the diff file list against the Canvas
`## Structure` section (feat:) or the issue "What Changed" section. Extra files indicate scope
creep; missing files indicate an incomplete implementation.

**No drive-by refactoring** — AI models often rename variables, reorder functions, or extract
helpers while implementing a feature. These changes are hard to review and introduce regression
risk. Each undocumented change is a liability during rollback.

**No premature abstraction** — AI tends to generalise: "I'll make this configurable for the
future." Zynax's engineering policy is explicit: no abstraction without a current requirement.
See `AGENTS.md §Anti-patterns`.

### SPDD

**Canvas reflects final implementation** — The Canvas was written before code. Verify the
O-steps in the Canvas still match what was actually built. If the implementation diverged, the
Canvas must be updated before merge (use `/spdd-sync`).

**`/spdd-security-review` passed** — The Canvas security checklist in the Safeguards section
must be complete. If it is empty or shows `pending`, the review is not done.

**Canvas status** — `Draft` means the Canvas has not been human-reviewed. Only `Aligned` or
`Implemented` are acceptable states for a feat: PR to merge.

### Security

**No Tier 2 context** — Tier 2 content is anything that should not appear in a public
repository: internal hostnames, credentials, names of internal systems, or PII. AI models
sometimes pull context from the conversation into code comments or error messages.
See `docs/knowledge-base-policy.md` for definitions.

**Boundary validation** — AI-generated handlers often skip validation on the assumption that
upstream validated the input. Verify that every new gRPC handler or HTTP endpoint validates
its inputs independently.

**Suppression comments** — `//nolint:` and `# type: ignore` inserted by AI are high-risk:
they silence warnings that exist for a reason. Each must have a comment explaining why the
suppression is safe.

---

*References: Epic #173 (Pillar 9 — Code Correctness) · ADR-019 (SPDD/Canvas requirement) ·
`docs/patterns/spdd-guide.md` · `docs/knowledge-base-policy.md`*
