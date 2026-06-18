# SPDD — Structured Prompt-Driven Development

> Methodology reference for contributors and AI assistants.
> Governed by ADR-019. Applies to all `feat:` PRs.
> Canvas artifacts live in `docs/spdd/`. Templates in `docs/spdd/CANVAS_TEMPLATE.md`.

> **You normally drive this pipeline through two verbs, not the steps directly.** `/plan` runs
> analysis → story → canvas → security-review and aligns the Canvas; `/deliver` generates from an
> Aligned Canvas one Operations step at a time. The `/lib:spdd-*` building blocks below are what
> those verbs call — invoke them directly only when you want fine-grained control. Full command
> map: [.claude/commands/README.md](../../.claude/commands/README.md).

---

## Why SPDD Exists

AI-assisted development introduces failure modes that code review alone cannot catch:
prompt drift (generated code diverges from original intent), invisible reasoning (design
choices exist only in chat history), and context leakage (internal details end up in
public artifacts). SPDD treats the prompt as a first-class engineering artifact.

**The core rule:** fix the prompt first, then fix the code.

---

## The Six-Step Workflow

Every `feat:` issue follows this sequence. Steps 1–3 produce the Canvas; steps 4–6
execute from it.

```
1. /lib:spdd-analysis   → research
2. /lib:spdd-story      → decompose
3. /lib:spdd-canvas → generate Canvas (Draft)
         ↓
   human alignment review
         ↓
   Canvas status: Aligned
         ↓
4. /lib:spdd-generate   → implement one step at a time
5. /lib:spdd-security-review → before every Canvas commit
6. /lib:spdd-sync       → after refactors; /lib:spdd-prompt-update for requirement changes
```

No code is written before the Canvas is `Aligned`. This is enforced by `/lib:spdd-generate`.

---

## Step 1 — `/lib:spdd-analysis <issue>`

Scans the codebase and produces a structured analysis:

- Existing concepts the feature will extend (services, domain types, gRPC contracts)
- New concepts the feature introduces
- ADR constraints that govern the approach
- System boundaries and gRPC contracts touched
- Risk table: breaking changes, performance, security
- Tier 2 flags: any sensitive context to move to `canvas.private.md`
- Recommended design direction (2–3 sentences)

Run this before touching the Canvas. The analysis output lives in the session; the
Canvas generation in step 3 draws from it.

---

## Step 2 — `/lib:spdd-story <issue>`

Breaks the feature into 2–5 INVEST-compliant user stories and creates one GitHub
issue per story as a child of the parent epic:

- **I**ndependent — deliverable separately
- **N**egotiable — details are flexible
- **V**aluable — observable value on its own
- **E**stimable — can be sized
- **S**mall — fits in one PR (≤ 400 lines, excluding generated code)
- **T**estable — clear, verifiable acceptance criteria

Each story maps to one Operations step in the Canvas. The recommended implementation
order from this command feeds directly into Canvas section O.

**GitHub issue creation (default behaviour):** after displaying the stories,
`/lib:spdd-story` opens one `gh issue` per story. Title format:
`feat(<scope>): <title> (#<parent>, step <N>)`. The Canvas O section must link
to these issue numbers once they exist — update the canvas after running this command.

---

## Step 3 — `/lib:spdd-canvas <issue>`

Generates `docs/spdd/<issue>-<slug>/canvas.md` from the analysis. The Canvas has seven
sections (the REASONS acronym is the checklist):

| Section | What goes in it |
|---------|----------------|
| **R** — Requirements | Problem statement + definition of done (3–6 bullets, specific and observable) |
| **E** — Entities | Domain entities and relationships — ASCII diagram or list; Tier 1 only |
| **A** — Approach | What we will / will NOT do; governing ADR citations for one-way doors |
| **S** — Structure | Which services, packages, files, and gRPC contracts are touched |
| **O** — Operations | Ordered steps — each step = one reviewable unit; steps are independently verifiable |
| **N** — Norms | Cross-cutting standards from AGENTS.md + layer contracts (commit hygiene, BDD, GOWORK=off) |
| **S** — Safeguards | Non-negotiable constraints from ADRs + context security checklist (must be ✅ before commit) |

After generating, the command automatically runs `/lib:spdd-security-review` on the output.

**Canvas status starts as `Draft`.** A human must review and change the status to
`Aligned` before any code is generated.

---

## Human Alignment Review

Before marking `Aligned`, the reviewer confirms:

1. **R** — Does the problem statement match what the issue actually asks for?
2. **A** — Is the approach consistent with existing ADRs? Are the "will NOT" items correct?
3. **O** — Is the Operations sequence feasible? Are steps independently testable?
4. **S (Safeguards)** — Has `/lib:spdd-security-review` passed? Are all checkboxes ticked?

Update the Canvas header: `**Status:** Aligned`. The Canvas is now the source of truth.

---

## Step 4 — `/lib:spdd-generate <path/to/canvas.md>`

Executes one Operations step at a time. For each step it:

1. Refuses to run from a `Draft` Canvas
2. Reads all files listed in the Canvas S — Structure section
3. Checks every Safeguard — halts and reports if any would be violated
4. Generates the code change for this step only
5. Checks the output against: layer boundaries, no panic in production, GOWORK=off,
   BDD `.feature` file present if a gRPC boundary is touched

After each step: review the output, commit it, then call `/lib:spdd-generate` again for
the next step. Never batch steps.

**Safeguards that cause an automatic halt:**
- Hardcoding an engine name (ADR-015)
- New gRPC method without a `.feature` file first (ADR-016)
- Import from another service's `internal/` (ADR-008)
- Tier 2 context embedded in code comments

---

## Step 5 — `/lib:spdd-security-review <path>`

Run before committing any Canvas or KB file. Checks four things:

| Check | What it catches |
|-------|----------------|
| Tier 2 scan | Real hostnames, private IPs, credentials, PII, unpublished strategy |
| Prompt injection | Override attempts, persona injection, priority-override phrasing |
| Abstraction check | Entities that reveal internal infrastructure topology |
| Authority hierarchy | Canvas content that would cause an AI to contradict AGENTS.md |

Result is `PASS` or `FAIL` with a findings table. A Canvas cannot be committed
until the review passes. Update the Safeguards checklist in the Canvas with the result.

---

## Step 6 — Keeping the Canvas in Sync

Two scenarios after a Canvas is `Aligned`:

**Requirements changed mid-sprint → `/lib:spdd-prompt-update <canvas.md>`**

```
1. Describe the requirement change
2. Command identifies which REASONS sections are invalidated
3. Proposes before/after diff for each affected section
4. Lists alignment decisions the human must confirm
5. Resets Canvas to Draft — must re-align before code generation resumes
```

Never update code first. The Canvas is always updated before the code.

**Refactor with no logic change → `/lib:spdd-sync <canvas.md>`**

```
1. Compares current implementation against Canvas O and S sections
2. Proposes updates for moved files, renamed types, split operations
3. Does NOT change R, A, or S (Safeguards) — those reflect intent, not mechanics
4. Updates Canvas status to Synced
```

---

## Context Security — Three Tiers

| Tier | Content | Where it goes |
|------|---------|--------------|
| **Tier 1** | Architecture patterns, domain entity names, ADR refs, public-safe abstractions | `canvas.md` — committed to public repo |
| **Tier 2** | Internal hostnames, private IPs, credentials, PII, deployment specifics | `canvas.private.md` — gitignored, never committed |
| **Tier 3** | Current branch state, test output, session context | Session-only — never persisted anywhere |

`canvas.private.md` is distributed out-of-band (private repo, encrypted file, or
pasted into session at start). See `docs/spdd/PRIVATE_CANVAS_TEMPLATE.md`.

---

## Canvas File Layout

```
docs/spdd/
├── README.md                          ← naming convention, lifecycle, canvas index
├── CANVAS_TEMPLATE.md                 ← copy this to start a new canvas
├── PRIVATE_CANVAS_TEMPLATE.md         ← Tier 2 companion (gitignored when named canvas.private.md)
└── <issue>-<slug>/
    ├── canvas.md                      ← public Canvas (Tier 1 only, always committed)
    └── canvas.private.md              ← Tier 2 companion (gitignored, NEVER committed)
```

Slug: issue title kebab-cased, first 4–6 words. Example: `214-temporal-execution`.

---

## Worked Example — M3 Temporal Execution (issue #214)

```bash
# 1. Research
/lib:spdd-analysis 214

# 2. Decompose into stories
/lib:spdd-story 214

# 3. Generate Canvas → writes docs/spdd/214-temporal-execution/canvas.md
/lib:spdd-canvas 214

# 4. Security check (runs automatically, but can be re-run)
/lib:spdd-security-review docs/spdd/214-temporal-execution/canvas.md

# 5. Human reviews Canvas, sets status: Aligned, commits it
git add docs/spdd/214-temporal-execution/canvas.md
git commit -S -s -m "docs: SPDD Canvas for M3 Temporal Execution — aligned (#214)"

# 6. Execute step by step
/lib:spdd-generate docs/spdd/214-temporal-execution/canvas.md
# → asks which Operations step; generates only that step
# Review, commit, repeat for each step

# 7. After a refactor
/lib:spdd-sync docs/spdd/214-temporal-execution/canvas.md

# 8. If requirements change
/lib:spdd-prompt-update docs/spdd/214-temporal-execution/canvas.md
# → update Canvas, re-align, then continue with /lib:spdd-generate
```

---

## Quick Reference — When to Use Each Command

| Situation | Command |
|-----------|---------|
| Starting a new `feat:` issue | `/lib:spdd-analysis` → `/lib:spdd-story` → `/lib:spdd-canvas` |
| Before committing any Canvas | `/lib:spdd-security-review` |
| Ready to write code | `/lib:spdd-generate` (Canvas must be Aligned) |
| Need a BDD `.feature` file for a new gRPC method | `/lib:spdd-api-test` |
| Requirements changed | `/lib:spdd-prompt-update` (Canvas first, code second) |
| Just refactored, no logic change | `/lib:spdd-sync` |
| Checking what changed between Canvas and code | `/lib:spdd-sync` (read-only diff mode) |

---

*See also:* `docs/adr/ADR-019-spdd-prompt-governance.md` · `docs/spdd/CANVAS_TEMPLATE.md` · `docs/spdd/README.md`
