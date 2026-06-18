---
description: "Documentation for the Zynax Claude command set — not a runnable command."
user-invocable: false
disable-model-invocation: true
---
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax Claude commands — how to drive this repo

**The one question this answers:** *"What do I run to get from an idea to merged, aligned work in
this repo — without memorizing 20 commands?"*

You reach for **five verbs**. Everything else is a building block they call for you.

```
                 idea / prompt / #issue / #epic
                              │
                          /plan ──────────────►  aligned REASONS Canvas + linked issues (orchestrate-ready)
                              │
                         /deliver ───────────►  claim → implement → PR → CI → squash-merge → post-merge verify
                              │
                          /learn  ────────────►  fold session learnings into the expert guides
                              
   anytime:  /review   (architecture · security · product · market-fit)
   on break: /reconcile (recover from a failed run + clean the repo)
   release:  /milestone (open · close — the only milestone-aware command)
```

Everything is **milestone-agnostic**: commands work from repository state (issues, epics, canvases,
labels). Milestone is an **optional `--milestone M` filter**, not the organizing principle.
Everything is **safe by default**: with no arguments (and without `--execute`) a command *plans and
proposes* — it doesn't mutate. Add `--execute` (or say "go") to apply.

---

## The five main commands

| Command | Run it to… | No-arg default | Common args |
|---------|-----------|----------------|-------------|
| **`/plan`** | turn an idea/prompt/issue/epic into an **Aligned canvas + linked issues** (SPDD pipeline, then align + link) | infer unfiled/ready work and propose canvases (PLAN only) | `"prompt"` · `#issue` · `#epic` · `--milestone M` · `--execute` |
| **`/deliver`** | **implement ready work** end-to-end → PR → CI → squash-merge → post-merge verify | pick the next ready work repo-wide and deliver a safe batch | `#issue` · `#epic` · `<canvas>` · `--milestone M` · `--batch N` |
| **`/review`** | produce dated **review docs** (architecture/security/product/market) | run all four in parallel | `--only <kind>` · `--pr` · `--split` · `--since <path>` |
| **`/reconcile`** | **recover from a failed run** + **clean the repo** (truth-pass surfaces, triage `[AUTO]` issues, prune branches) | PLAN a truth-pass + report recovery items | `--execute` · `--include-stale` · `--milestone M` |
| **`/learn`** | fold accumulated **session learnings** into the expert guides | synthesize → write PENDING proposals to `APPLY_LOG.md` | `--domain D` · `--apply` |

### Plus one lifecycle command
**`/milestone open <name> <version> "<title>"` · `/milestone close [--dry-run]`** — the only
milestone-aware command and the sole writer of `state/milestone.yaml`. No-arg prints current status.
Use it only when cutting/closing a release; day-to-day work doesn't need it.

---

## The decision tree (same as before — fewer doors)

1. **Have an idea / a prompt / a filed issue?** → `/plan`. It runs `analysis → story → canvas →
   security-review`, then **aligns** the canvas and **links** issues↔canvas (Operations-step refs,
   `status: ready`, `spdd: canvas`). `feat:` work must reach an **Aligned** canvas before code (ADR-019).
2. **Have aligned, ready work?** → `/deliver`. It claims (deterministic `<type>/<N>` branch),
   dispatches the right `experts/*` persona, generates per Operations step, runs gates, opens a
   signed PR, watches CI, squash-merges, and verifies post-merge.
3. **Finished a batch?** → `/learn` to capture what worked into the expert guides.
4. **Something broke mid-flight, or the repo drifted?** → `/reconcile`.
5. **Want a fresh read of where the project stands?** → `/review`.

---

## Folder structure

```
.claude/commands/
  README.md          ← you are here
  plan.md  deliver.md  review.md  reconcile.md  learn.md   ← the 5 verbs (top-level, safe defaults)
  milestone.md        ← release lifecycle (open|close|status)
  lib/                ← building blocks the main commands invoke (power-users: /lib:<name>)
  experts/            ← domain personas /deliver dispatches (not run directly)
```

### `lib/` — building blocks (you rarely call these directly)
Invoked by the main commands; available as `/lib:<name>` if you want fine-grained control.

| Group | Files | Called by |
|-------|-------|-----------|
| SPDD pipeline | `spdd-analysis` `spdd-story` `spdd-canvas` `spdd-security-review` `spdd-generate` `spdd-prompt-update` `spdd-sync` `spdd-api-test` | `/plan`, `/deliver` |
| Delivery machinery | `deliver-one` `deliver-batch` `deliver-resume` `sequence` | `/deliver` |
| Roadmap inference | `plan-infer` | `/plan` (no-arg) |
| Review docs | `architecture-review` `security-review-doc` `product-review` `market-fit-review` | `/review` |
| Milestone lifecycle | `milestone-open` `milestone-close` | `/milestone` |

### `experts/` — domain personas
`bdd-contract · go-services · python-adapters · ci-release · infra-helm · git-ops · spdd-canvas ·
post-merge`. `/deliver` routes each issue to the matching persona (by title scope / file area). They
are **updated by `/learn`** as session learnings accumulate; you don't invoke them directly.

---

## Worked example — from a prompt to merged

```
/plan "let a user try Zynax with no clone"        # → drafts epic + stories, canvas, security-review
/plan "...same..." --execute                       # → files issues, commits Aligned canvas, links them
/deliver #<epic>                                    # → implements ready O-steps → PRs → CI → merge → verify
/learn                                              # → proposes expert-guide learnings (you approve)
# if a run stalls:  /reconcile --execute
```

---

## Conventions (enforced)

- **Canvas-first for `feat:`** — a REASONS Canvas committed and **Aligned** before any implementation
  code (ADR-019). Prompt-first: requirements change → update the canvas (`/lib:spdd-prompt-update`)
  → then code, never the reverse. Guide: [docs/patterns/spdd-guide.md](../../docs/patterns/spdd-guide.md) ·
  template: [docs/spdd/CANVAS_TEMPLATE.md](../../docs/spdd/CANVAS_TEMPLATE.md).
- **Commits/PRs** — conventional title (`feat|fix|refactor|docs|test|ci|chore`), DCO `Signed-off-by`,
  `Assisted-by: Claude/<model>` (never `Co-Authored-By` for AI), squash-merge only.
- **Canvas safety** — Tier 1 (public) only; sensitive context → gitignored `canvas.private.md`. Never
  reference a filename containing a dotted `local`/`internal`/`corp` label (gitleaks `internal-hostname`
  BLOCKs the commit — rename the artifact).
- **Editing these commands** — `.claude/` is gitignored with a `!.claude/commands/` negation, so a new
  command file needs `git add -f`. Command files are PR-size-exempt.
- **Safety** — main commands PLAN by default; mutations gated by `--execute` (or an explicit "go").

## Authoring metadata (frontmatter)
`description` (menu summary) · `argument-hint` (usage) · optionally `model`, `allowed-tools`. A `lib/`
file is namespaced `/lib:<name>` and stays model-invocable so the main commands can call it.
