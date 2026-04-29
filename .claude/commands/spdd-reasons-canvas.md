# /spdd-reasons-canvas

Generate a complete REASONS Canvas from prior /spdd-analysis output. Produces a canvas.md file ready for human alignment review.

## Instructions

Using the analysis context in the current session (or re-run /spdd-analysis if needed):

1. Fill all 7 REASONS sections using the template below
2. For **N — Norms**: pull directly from the relevant AGENTS.md files (Go services: `docs/patterns/go-service-patterns.md`; Python agents: `docs/patterns/python-agent-guide.md`; etc.)
3. For **S — Safeguards**: pull from ADRs, architecture invariants in root AGENTS.md, and layer constraints
4. Every entity in **E** and every step in **O** must be Tier 1 (public-safe) — no internal hostnames, IPs, or credentials
5. After generating, immediately run /spdd-security-review on the output

## Canvas Template

```markdown
# REASONS Canvas — <Feature Title>

**Issue:** #<number>
**Author:** <maintainer name>
**Date:** YYYY-MM-DD
**Status:** Draft

---

## R — Requirements
> Problem statement: what breaks or is missing without this feature?
> Definition of done: observable outcomes that confirm delivery.

## E — Entities
> Domain entities and their relationships.
> Use a list or ASCII diagram. Tier 1 only — abstract names, no deployment specifics.

## A — Approach
> Solution strategy. Explicitly state: what we WILL do and what we WON'T do.
> Reference relevant ADRs that govern the choice.

## S — Structure
> System placement. Which services, packages, files are touched?
> Which gRPC contracts are extended or added?

## O — Operations
> Ordered, concrete, testable implementation steps.
> Each step = one reviewable unit (one PR or one commit).
> 1. <step>
> 2. <step>

## N — Norms
> Cross-cutting standards that apply to this feature.
> Pull from: root AGENTS.md Hard Constraints, layer AGENTS.md, docs/patterns/*.
> - Commit hygiene: Signed-off-by + Assisted-by required
> - <layer-specific norms>

## S — Safeguards (second S)
> Non-negotiable constraints. Things that MUST NEVER happen in this feature.
> Pull from: ADRs, architecture invariants, AGENTS.md mandates.
>
> ### Context Security (mandatory before committing this Canvas)
> - [ ] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
> - [ ] No PII: no personal names in sensitive context, no email addresses
> - [ ] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
> - [ ] All entities in E are public-safe abstractions
> - [ ] /spdd-security-review passed on this file
>
> ### Feature Safeguards
> - Never <specific invariant from relevant ADR>
> - Never <specific constraint>
```

## Output

Write the canvas to `docs/spdd/<issue>-<slug>/canvas.md` and then automatically invoke /spdd-security-review on it.

## Input

$ARGUMENTS — GitHub issue number or feature description (analysis must have been run first in this session)
