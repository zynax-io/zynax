---
name: ADR Proposal
about: Propose an Architectural Decision Record for a significant design choice
labels: ["type: adr-proposal", "status: needs-triage", "status: needs-design"]
assignees: ''
---

<!--
An ADR captures a significant architectural decision: its context, the options
considered, the choice made, and the consequences. ADRs are permanent records —
they do not change after acceptance (though they can be superseded by later ADRs).

Use this template when:
  - You have a concrete architectural proposal ready to discuss
  - A significant decision needs to be recorded after discussion

For early-stage exploration of an architectural idea, use GitHub Discussions first.
-->

## Decision Title

A short noun phrase describing the decision, e.g.:
"Use NATS JetStream as the event bus" or "Require DCO sign-off instead of CLA"

---

## Context

What forces and constraints make this decision necessary right now?
What happens if we do NOT make a decision here?

---

## Decision

State the decision clearly and unambiguously:

> "We will ..."

---

## Options Considered

### Option A: [Title]
**Pros:**
-
**Cons:**
-

### Option B: [Title]
**Pros:**
-
**Cons:**
-

### Option C: [Title] (Chosen)
**Pros:**
-
**Cons:**
-

---

## Consequences

What becomes easier? What becomes harder? What new decisions does this force?
Any migration required from current state?

---

## Architecture Layer Impact

Which of the three layers does this affect?
- [ ] Layer 1 — Intent (YAML manifests, schemas)
- [ ] Layer 2 — Communication (gRPC contracts, AsyncAPI)
- [ ] Layer 3 — Execution (engine adapters, agent adapters)
- [ ] Cross-cutting

---

## Related

- Related ADR(s): `docs/adr/ADR-NNN-*.md`
- Related RFC(s): `docs/rfcs/RFC-NNN-*.md`
- Related issues: #

---

## Status

- [ ] Draft — open for discussion
- [ ] Proposed — ready for maintainer vote

_After acceptance, a maintainer will create the ADR file in `docs/adr/`._
