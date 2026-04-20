---
name: Feature Request
about: Propose a new capability or improvement to Keel
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

## Problem / Motivation

What problem are you trying to solve? Who has this problem and in what context?
Be specific — avoid "it would be nice if...". Describe the pain point.

---

## Proposed Capability

What should Keel be able to do that it cannot do today?

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

## Is This an Epic or a Story?

- [ ] **Story** — single, implementable issue (one PR or a small chain of ≤ 3 PRs)
- [ ] **Epic** — large feature spanning multiple issues and milestones
  _(If epic: describe the breakdown into child stories below)_

### Child Stories (if epic)

- [ ] Story 1: ...
- [ ] Story 2: ...
- [ ] Story 3: ...

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

## Additional Context

Related ADRs, RFCs, issues, external references, prior art in other projects.
