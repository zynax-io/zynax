---
name: Feature Request
about: Propose a new capability or improvement to Zynax
labels: ["type: feature", "status: needs-triage"]
assignees: ''
---

<!--
Before filing a feature request:
  1. Search existing issues and Discussions — this may already be proposed.
  2. Check ROADMAP.md — it may already be planned.
  3. Check docs/adr/ — the approach may already be decided.
  4. For large architectural features, consider starting a GitHub Discussion
     to explore the idea before filing a formal issue.
-->

## The one question this story answers

> _One sentence, in the reader's words — the job-to-be-done. A Zynax user should understand
> the point without repository knowledge. (For a large multi-PR effort, use the **Epic** template.)_

---

## Story (INVEST)

> The SPDD story spine. Keep it INVEST: **I**ndependent · **N**egotiable · **V**aluable ·
> **E**stimable · **S**mall · **T**estable.

- **As a** <role>, **I want** <capability>, **so that** <outcome>.
- Size: **XS · S · M · L** (Small = one PR, ≤ 400 lines excluding generated code)
- Canvas Operations step (if the parent epic has a canvas): `docs/spdd/<epic>-<slug>/canvas.md` → step N
- Depends on: #… · Blocks: #…

---

## Why it matters (product · adoption)

- **For the Zynax user / adopter:** <the observable value this story delivers>
- **Adoption angle:** <which lever — individual users · enterprises · CNCF interest · community>
- **Cost of not doing it:** <what stays broken / who stays blocked>

---

## Problem / Motivation

What problem are you trying to solve? Who has this problem and in what context?
Be specific — avoid "it would be nice if...". Describe the pain point.

---

## Parent epic & links

> Keep the chain linked both ways so nothing is orphaned.

- Parent epic: #…
- Milestone:
- REASONS Canvas / Operations step: see **Story (INVEST)** above
- RFC / ADR: 
- Related docs / architecture:

---

## Proposed Capability

What should Zynax be able to do that it cannot do today?

Write the expected behaviour as Gherkin if possible — this will become the
`.feature` file when implementation begins:

```gherkin
Feature: <feature name>
  As a <role>
  I want to <capability>
  So that <benefit>

  Scenario: <happy path>
    Given ...
    When ...
    Then ...

  Scenario: <error / edge case>
    Given ...
    When ...
    Then ...
```

---

## Story or Epic?

- [ ] **Story** — single, implementable issue (one PR or a small chain of ≤ 3 PRs). Continue below.
- [ ] **Epic** — large, multi-PR, multi-story effort → **use the Epic template instead** (it captures
  child stories, the one-question, and adoption impact at the right altitude).

---

## Architecture Impact

- [ ] New service required
- [ ] Proto contract change (will require RFC)
- [ ] New adapter type
- [ ] Change to the three-layer separation (requires ADR)
- [ ] No architecture changes — implementation only
- [ ] Not sure

---

## Breaking Change?

- [ ] Yes — requires RFC + major version bump. Describe impact:
- [ ] No — fully backward-compatible

---

## Which Milestone Does This Belong To?

Reference `ROADMAP.md` milestones (M1–M8) or state "new / unscheduled":

---

## Acceptance Criteria

How will we know this is done? (The Gherkin above is a good start. Add any
additional observable criteria here.)

1.
2.

---

## Manual validation (for user-visible stories)

> A tester unfamiliar with Zynax should be able to execute this and confirm the value.
> Follow the human-validation standard/template.

- Commands to run / expected output:
- Troubleshooting & rollback:
- Feedback questions:

---

## Additional Context

Related ADRs, RFCs, issues, external references, prior art in other projects.
