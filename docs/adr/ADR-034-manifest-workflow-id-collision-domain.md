# ADR-034: ManifestWorkflowID 64-bit collision domain and canonicalization stability

**Status:** Proposed  **Date:** 2026-06-15
**Related:** ADR-012 (Workflow IR), ADR-029 (data-flow) · GitHub #583

---

## Context

The workflow-compiler derives a `ManifestWorkflowID` used as the durable identity of a compiled
workflow. #583 raised that the id's collision domain (a 64-bit space) and the canonicalization that
feeds the hash are undocumented: it is unclear which manifest fields are canonicalized, in what order,
and what the practical collision probability is. Without a recorded decision, a future change to
canonicalization could silently change ids and break idempotent `ApplyWorkflow`.

## Decision

(To be finalized during the EPIC-Q canvas alignment.) Record:

1. The exact **canonicalization** of a manifest before hashing (field set, ordering, normalization).
2. The **collision domain** (64-bit) and the practical collision bound for expected workflow counts.
3. The **stability guarantee**: canonicalization is part of the contract — changing it is a breaking
   change requiring a new ADR and a migration note.

## Rationale

| Option | Assessment |
|--------|------------|
| Document + freeze canonicalization (chosen) | ✅ Makes idempotency contractual; prevents silent id drift |
| Widen to 128-bit id | ✗ Deferred — proto/contract change; not justified at current scale |
| Leave undocumented | ✗ Rejected — invites accidental breaking changes |

## Consequences

- **Positive:** `ApplyWorkflow` idempotency is contractually grounded; future contributors know the rules.
- **Negative / trade-off:** canonicalization is now frozen — intentional changes cost an ADR + migration.
- **Neutral / follow-up:** if scale ever warrants a wider id space, a successor ADR handles the migration.
