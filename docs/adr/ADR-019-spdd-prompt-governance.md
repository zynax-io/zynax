# ADR-019: Structured Prompt-Driven Development (SPDD)

**Status:** Accepted  **Date:** 2026-04-30
**Related:** ADR-016 (Layered Testing), ADR-018 (AI KB Authorization Model)

---

## Context

Zynax is built with significant AI assistance. This creates a class of failure modes
that traditional code review does not catch:

1. **Prompt drift.** AI-generated code satisfies the immediate request but diverges
   from the original intent as requirements are clarified mid-implementation.
2. **Invisible reasoning.** The rationale for design choices made during an AI session
   exists only in chat history — not in the codebase or review record.
3. **Compounding errors.** A flawed assumption in an AI prompt propagates across all
   generated code before anyone notices.
4. **Context leakage.** AI sessions can embed internal hostnames, credentials, or
   unpublished strategy into generated artifacts (comments, Canvas files, commit
   messages) that end up in the public repo.

The root cause in all four cases is the same: **the prompt is an unmanaged artifact.**
Requirements change, but no one updates the prompt. The Canvas is reviewed, but no
one checks for leaked context. Code is regenerated, but the rationale is lost.

---

## Decision

Adopt **Structured Prompt-Driven Development (SPDD)** as the mandatory methodology
for all `feat:` pull requests. SPDD treats the prompt as a first-class engineering
artifact with the same governance as code.

### Core rule: Canvas before code

Every `feat:` PR must include a REASONS Canvas committed to
`docs/spdd/<issue>-<slug>/canvas.md` **before any implementation code is written**.
The Canvas is the prompt. Code is generated from it. If requirements change,
the Canvas is updated first — then code is regenerated or patched.

### The REASONS Canvas structure

A Canvas has seven sections. The acronym is the checklist:

| Letter | Section | Purpose |
|--------|---------|---------|
| R | Requirements | Problem statement and definition of done |
| E | Entities | Domain entities and their relationships |
| A | Approach | What we will and will not do; governing ADRs |
| S | Structure | Which services, files, and gRPC contracts are touched |
| O | Operations | Ordered, testable implementation steps |
| N | Norms | Cross-cutting standards from AGENTS.md and layer contracts |
| S | Safeguards | Non-negotiable constraints; context security checklist |

### Canvas lifecycle

```
Draft  →  Aligned  →  Synced
  │           │
  │     (human signs off)
  │
  └── Prompt-update → Draft → Aligned (if requirements change)
```

A Canvas must reach `Aligned` status (human sign-off) before `/spdd-generate`
will execute any Operations step. If requirements change after alignment,
`/spdd-prompt-update` resets the Canvas to `Draft` — the human must re-align
before code generation resumes.

### Context security tiers

Every Canvas and KB file is classified before it is committed:

| Tier | Content | Handling |
|------|---------|---------|
| Tier 1 | Architecture patterns, domain entity names, ADR references | Canvas (`canvas.md`) — public repo |
| Tier 2 | Internal hostnames, credentials, deployment specifics, PII | Private companion (`canvas.private.md`, gitignored) |
| Tier 3 | Branch state, test output, session context | Session-only — never persisted |

`/spdd-security-review` must pass before any Canvas is committed.

### Slash commands

| Command | When to use |
|---------|------------|
| `/spdd-analysis` | Research phase — scan codebase, surface ADRs, classify context |
| `/spdd-story` | Decompose a feature into INVEST-compliant user stories |
| `/spdd-reasons-canvas` | Generate Canvas from analysis output |
| `/spdd-security-review` | Check Canvas for Tier 2 leaks and prompt injection before commit |
| `/spdd-generate` | Execute one Operations step from an Aligned Canvas |
| `/spdd-prompt-update` | Update Canvas when requirements change (before any code change) |
| `/spdd-sync` | Sync Canvas to implementation after a refactor |
| `/spdd-api-test` | Generate BDD `.feature` file for a new gRPC boundary |

### Scope

SPDD applies to all `feat:` PRs. It does NOT apply to:
- `fix:`, `refactor:`, `docs:`, `ci:`, `chore:`, `test:` PRs.
- Hotfixes where a Canvas would delay a critical fix — document the exception in the PR.

---

## Rationale

| Option | Assessment |
|--------|------------|
| SPDD — Canvas as managed prompt artifact | ✅ Chosen — audit trail, drift prevention, context security, AI-safe review |
| Unstructured AI assistance (no Canvas) | ✗ Rejected — no audit trail; prompt drift identified as root cause of most AI-related review failures |
| Canvas optional / best-effort | ✗ Rejected — without enforcement, adoption is inconsistent; the value of SPDD is in the gate, not the template |
| Full human authorship, no AI | ✗ Rejected — counter to project velocity goals; AI assistance is productive when governed |

---

## Consequences

- **Positive:** Every `feat:` PR has an auditable reasoning trail. Requirements changes
  are visible in Canvas diffs. Context security is enforced before public commit.
  AI-generated code is always anchored to a human-aligned intent document.
- **Negative / trade-off:** `feat:` PRs require a Canvas step before implementation,
  adding one review round for new features. This is the intended cost — it replaces
  implicit prompt negotiation with explicit alignment.
- **Neutral / follow-up required:**
  - ADR-019 is listed in `docs/adr/INDEX.md`.
  - `AGENTS.md` Hard Constraints section updated with Canvas mandate (issue #208).
  - `CLAUDE.md` updated with Canvas before code rule (issue #208).
  - Canvas template committed to `docs/spdd/` (issue #207).
  - SPDD guide committed to `docs/patterns/spdd-guide.md` (issue #207).
  - PR template updated with Canvas link field (issue #209).
  - Canvas validator added (`make validate-canvas`, issue #211).
  - CI freshness gate added for `feat:` PRs (issue #212).
