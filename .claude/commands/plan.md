---
description: From an idea, prompt, issue, or epic to an aligned REASONS Canvas with linked issues — runs the SPDD pipeline (analysis → story → canvas → security-review), aligns the canvas, and links issues↔canvas so the cluster is orchestrate-ready. No-arg infers unfiled/ready work from the repo. Milestone-agnostic; --milestone is an optional filter. PLAN by default; mutations gated by --execute.
argument-hint: "[\"prompt\" | #issue | #epic] [--milestone M] [--execute]   default: infer + PLAN, no mutations"
---

# /plan — idea → aligned canvas + linked issues

Turn *what you need* into a committed, security-reviewed, **Aligned** REASONS Canvas with linked
GitHub issues, ready for `/deliver`. This is the single front door for the SPDD pipeline — it
composes the `lib:spdd-*` building blocks so you don't run them by hand.

> **Safe by default.** With no `--execute`, this only researches, drafts, and proposes — it prints
> the plan (issues it would create, the canvas it would write) and **mutates nothing**. Re-run with
> `--execute` (or say "go") to file issues, commit the canvas, and link everything.

## Parse `$ARGUMENTS`

- **free-text prompt** (e.g. `/plan "let users try Zynax with no clone"`) → a NEW idea. Draft an epic
  from it (use the Epic issue template framing: *the one question this answers* + product/adoption
  impact), then run the pipeline below.
- **`#<number>`** → an existing issue/epic. Run the pipeline for it.
- **no argument** → **infer** unfiled/ready work from the repo (delegate to `/lib:plan-infer`) and
  propose canvases for the highest-value gaps. PLAN only unless `--execute`.
- **`--milestone M`** → optional filter/assignment (otherwise milestone-agnostic; work from repo state).
- **`--execute`** → perform the mutations (file issues, commit canvas, link, label).

## Pipeline (the same decision tree, fewer commands)

For a prompt/issue/epic target, run the building blocks in order and stop on any blocker:

1. `/lib:spdd-analysis <target>` — research: codebase scan, ADRs, risk table, Tier 2 flags.
2. `/lib:spdd-story <target>` — decompose into INVEST stories. **Idempotency guard:** if the epic
   already has child issues, reconcile/validate them — never create duplicates.
3. `/lib:spdd-canvas <target>` — generate `docs/spdd/<issue>-<slug>/canvas.md` (Status: Draft),
   using `docs/spdd/CANVAS_TEMPLATE.md`. Keep it Tier 1; move sensitive context to `canvas.private.md`.
4. `/lib:spdd-security-review <canvas>` — must return **PASS** before the canvas is committed
   (no Tier 2 content, no injection, no abstraction leaks). Avoid `.local`/`.internal`/`.corp` in
   any referenced filename (gitleaks `internal-hostname` BLOCKs the commit — rename the artifact).
5. **Align + link** (with `--execute`): set the canvas `Status: Aligned`; map every Operations step
   1:1 to a story issue; back-link the canvas path + step into each issue; label stories
   `status: ready`; put `spdd: canvas` on the epic; assign `--milestone M` if given. Leave the
   cluster orchestrate-ready for `/deliver`.

If requirements change later, run `/lib:spdd-prompt-update <canvas>` (prompt-first rule: update the
Canvas, which resets it to Draft and re-runs the security review — never patch code first).

## Output

- **PLAN mode (default):** the proposed epic/stories, the canvas outline, the link/label plan — and
  an explicit "re-run with `--execute` to apply" line.
- **`--execute`:** issue numbers created/reconciled, the committed canvas path, the links applied,
  and the exact `/deliver` command to run next.

## Norms
Conventional commits (feat/fix/refactor/docs/test/ci/chore) · DCO `Signed-off-by` + `Assisted-by:
Claude/<model>` (never `Co-Authored-By` for AI) · canvas committed before any implementation code
(ADR-019) · new `.claude/commands` files need `git add -f`. Full guide: `.claude/commands/README.md`.
