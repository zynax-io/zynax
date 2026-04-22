# 002: PR Size Limit — 900 Lines

**Date:** 2026-04-21  **Author:** M1 Engineering

## Context

Large PRs are harder to review, more likely to contain subtle errors, and harder
to revert cleanly. We needed a concrete numeric limit that CI could enforce and
that contributors could plan against. The question was: what number?

## Decision

**900 lines changed** (additions + deletions, excluding generated files and
lock files) is the PR size limit. PRs over 900 lines are blocked by CI.

**What counts toward the limit:**
- All hand-written source code (`.go`, `.py`, `.proto`, `.yaml`, `.ts`)
- Documentation files (`.md`)
- CI workflow files (`.yml`)

**What is excluded:**
- `protos/generated/**` (machine-generated stubs — excluded by `.gitattributes`)
- `go.sum`, `poetry.lock`, `uv.lock` (dependency lock files)
- Migrations files (bulk data, reviewed separately)

## History

The initial limit was 800. During M1, two PRs legitimately required 850–870 lines
to implement a single cohesive feature slice (a proto contract + its BDD scenarios).
After review, 900 was chosen as the ceiling: high enough to accommodate realistic
feature slices, low enough to remain reviewable in a single sitting (~30 minutes
at a reading pace of 30 lines/minute including context).

Numbers considered and rejected:
- **800:** Tripped on legitimate feature slices; caused unnecessary splits.
- **1000:** Too large; a 1000-line PR requires multiple context switches to review.
- **500:** Too restrictive for combined proto + BDD work (typically 300 + 200 lines).

## Consequences

- PRs that approach the limit should be decomposed first. The issue template and
  CONTRIBUTING.md both document this constraint.
- Exceptions (genuinely indivisible changes above 900 lines) require a maintainer
  approval comment on the PR before review begins. These are tracked in git history
  so the pattern can be evaluated over time.
- The `protos/generated/` exclusion means a proto change + stub regen stays within
  the limit even though the generated diff is large.
