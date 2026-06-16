<!-- SPDX-License-Identifier: Apache-2.0 -->

# SPDD Security Review — EPIC W: Workflow Data-Flow (#1167)

**Reviewed file:** `docs/spdd/1167-workflow-data-flow/canvas.md`
**Companion ADR:** ADR-029 (Accepted)
**Date:** 2026-06-16
**Standard:** `.claude/commands/spdd-security-review.md` (5 checks) + EPIC-W feature-specific
data-flow injection analysis.

## Overall verdict: PASS-with-flags

The canvas is publication-safe and the data-flow design closes its injection surface by
construction. All flags raised were **non-blocking documentation-staleness** issues and have
been resolved in the Draft. No Tier 2 content, no prompt injection, no abstraction leak, no
authority-hierarchy violation. A human still owns the `Status: Aligned` flip per ADR-019.

## Standard checks

| # | Check | Result | Notes |
|---|-------|--------|-------|
| 1 | Tier 2 content scan | PASS | No hostnames/IPs/internal TLDs/credentials/replica counts. Env prefixes (`ZYNAX_WC_`, `ZYNAX_ENGINE_ADAPTER_`) are public naming conventions, not topology. |
| 2 | Prompt-injection scan | PASS | No instruction-to-AI phrasing; all content is human documentation. |
| 3 | Abstraction check | PASS | E-section entities are abstract type names (WorkflowIR, Action, WorkflowDataContext, DataReference); no environment inference. |
| 4 | Authority hierarchy | PASS | N-section reinforces AGENTS.md (DCO, GOWORK=off, buf-breaking); nothing contradicts AGENTS.md. |
| 5 | Completeness | PASS | All 7 REASONS sections present; Status=Draft (valid); Context Security checklist present. |

## Feature-specific injection analysis (output/input bindings)

EPIC W introduces output/input bindings. The original threat framing assumed CEL/Go-template
expression evaluation. **ADR-029 §1/§3 explicitly defer any expression, transform, template,
filter, or arithmetic language to M-dx.** The M7 surface is literal values + JSON-path
references that resolve *verbatim* or fail at compile time. This materially shrinks the attack
surface.

| Surface | Exposure in M7 | Mitigation (canvas + ADR-029) | Status |
|---------|----------------|-------------------------------|--------|
| Expression injection | None — no eval/template/transform language in M7 | ADR-029 §1/§3; canvas A "We will NOT add an expression/transform language"; S-safeguard "Never introduce an expression language in M7" | CLOSED by design |
| Data-context read/write scoping | A run-scoped K/V store written only by `output_bindings`, read only by `input_bindings` | ADR-029 §2; canvas S-safeguard "Never let one workflow read another's data context (strict run-scoping)" | Mitigated |
| Untrusted workflow YAML as injection vector | JSON-path refs come from user manifests | Compile-time reference-resolution: every input ref must resolve to a declared upstream output, else `COMPILATION_ERROR` with line number. Paths are looked up, never evaluated as code | Mitigated |
| New gRPC boundary (W.2) | New IR behaviour on WorkflowCompilerService | `.feature` committed before implementation (ADR-016); canvas N-section + W.2 acceptance require it; `/spdd-api-test` flagged | Mitigated |
| Cross-run / cross-namespace leakage | Shared mutable state risk | ADR-029 §2 strict run-scoping, no persistence beyond run; aligns with ADR-008 | Mitigated |

## Implementation-time security acceptance (for W.2–W.4 reviewers)

These are not blockers to alignment but must be verified when the feat PRs land:

- W.3: unresolved/dangling JSON-path input refs are a hard `COMPILATION_ERROR` (fail closed),
  not a silent empty value.
- W.3/W.4: no path-traversal-style escape out of `$.states.<state>.output.<key>` — reject any
  reference that does not match the documented rooted form.
- W.4: a type-mismatched or missing context key fails the run with a structured error, never a
  panic or silent default.
- W.2: the proto change is strictly additive (new field numbers); `buf breaking` stays green.

## Flags raised and resolved in this Draft

| Flag | Severity | Resolution |
|------|----------|------------|
| Context Security checklist marked `result: PENDING` | WARN (staleness) | Updated to PASS-with-flags pointing at this artifact. |
| O-section listed W.1/#1175 as an open story, but it is CLOSED and ADR-029 is Accepted | WARN (staleness) | Marked W.1 ✅ DONE; clarified #1176–#1179 are the remaining 1:1 implementation stories. |

## Reminder

Per ADR-019, this review does **not** set `Status: Aligned`. A human reviewer flips the status
after confirming the alignment checklist in the PR description.
