# ADR-034: ManifestWorkflowID 64-bit collision domain and canonicalization stability

**Status:** Proposed  **Date:** 2026-06-15
**Related:** ADR-012 (Workflow IR), ADR-029 (data-flow) · GitHub #583

---

## Context

The workflow-compiler stamps every compiled `WorkflowIR` with a `ManifestWorkflowID` (the
`workflow_id` envelope field) that becomes the durable identity of a compiled workflow. #583
raised that the id's collision domain (a 64-bit space) and its canonical form were never recorded:
it was unclear how wide the id is, how it is encoded, and what the practical collision probability
is at expected workflow counts. Without a recorded decision a future change to the derivation could
silently change ids and break consumers that key on `workflow_id`.

This ADR documents the **current** scheme exactly as implemented in
`services/workflow-compiler/internal/api/server.go` (`generateWorkflowID`). It does **not** change
the algorithm — it freezes the present behaviour as a contract.

### Current scheme (as implemented)

```go
func generateWorkflowID() string {
    randBytes := make([]byte, 8)            // 8 bytes = 64 bits
    _, _ = rand.Read(randBytes)             // crypto/rand
    return fmt.Sprintf("wf-%s", hex.EncodeToString(randBytes))
}
```

- **Source of entropy:** 8 bytes (64 bits) drawn from `crypto/rand`. The id is a freshly generated
  random value per `CompileWorkflow` call — it is **not** a hash of the manifest, and identical
  manifests therefore produce distinct ids.
- **Collision domain:** the 64-bit value space, i.e. 2^64 ≈ 1.8 × 10^19 distinct ids.
- **Canonical string form:** the literal prefix `wf-` followed by exactly 16 lowercase hexadecimal
  characters (`hex.EncodeToString` is fixed-width, zero-padded, lowercase). Total length 19 chars.
  This string form — prefix, width, lowercase hex alphabet — **is** the canonicalization that
  consumers may rely on for parsing, matching, and storage.

## Decision

1. **Collision domain is 64 bits.** The id occupies the full 2^64 space drawn from a CSPRNG. By the
   birthday bound, the expected number of ids before a ~50% chance of any collision is
   ≈ 1.2 × √(2^64) ≈ 5.1 × 10^9. For a corpus of N compiled workflows the collision probability is
   approximately N² / 2^65; at N = 1 × 10^6 workflows this is ≈ 2.7 × 10^-8 — negligible at the
   scales Zynax targets. Widening the id is therefore deferred (see Rationale).

2. **Canonical form is frozen.** A `ManifestWorkflowID` is canonically `wf-` + 16 lowercase hex
   characters. The prefix, the 16-char width, and the lowercase-hex alphabet are part of the
   contract: consumers MAY parse and validate against this shape.

3. **Stability guarantee.** The width (64 bits / 16 hex chars), the `wf-` prefix, and the
   lowercase-hex encoding are a backward-compatibility contract. Changing any of them — narrowing or
   widening the entropy, switching to a manifest-derived hash, or altering the encoding — is a
   **breaking change** that requires a successor ADR and a migration note for stored ids.

## Rationale

| Option | Assessment |
|--------|------------|
| Document + freeze the current 64-bit random id (chosen) | ✅ Records reality; makes the id shape a parseable contract; prevents silent drift |
| Switch to a manifest-derived hash | ✗ Different semantics (would make ids content-addressable / idempotent); a behaviour change, out of scope for #583 |
| Widen to 128-bit id | ✗ Deferred — proto/contract change with no collision pressure at current scale |
| Leave undocumented | ✗ Rejected — invites accidental breaking changes to width or encoding |

## Consequences

- **Positive:** the id's width, encoding, and collision bound are now contractual; consumers can
  parse `workflow_id` against a fixed shape and reason about collision risk.
- **Negative / trade-off:** the encoding is now frozen — any intentional change costs a successor ADR
  plus a migration note.
- **Neutral / follow-up:** ids are random, not content-addressable; if a future requirement needs
  idempotent (manifest-derived) ids, a successor ADR introduces that scheme and handles migration. If
  scale ever warrants a wider id space, the same successor-ADR path applies.
