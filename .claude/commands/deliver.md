---
description: Deliver ready work end-to-end — claim → implement (expert dispatch + SPDD generate) → local gates → signed PR → CI → squash auto-merge → post-merge verify → learnings. No-arg picks the next ready work repo-wide; pass #issue for one story, #epic to resume a cluster, --batch N for parallel delivery. Milestone-agnostic; --milestone is an optional filter. Preserves worktree isolation + deterministic claim-key idempotency.
argument-hint: "[#issue | #epic | <canvas-path> | (none = next ready)] [--milestone M] [--batch N]"
---

# /deliver — ready work → merged + verified

One front door for implementation. It routes to the battle-tested delivery building blocks under
`lib/` (worktree isolation, deterministic `<type>/<N>` claim key, parallel batches, post-merge
verification) — you no longer choose between orchestrate / issue-deliver / resume by hand.

> **Milestone-agnostic.** With no `--milestone`, scope is **repo-wide ready work**: open issues
> labelled `status: ready`, with an `Aligned` canvas if they are `feat:`. `--milestone M` filters to
> that milestone. Nothing requires a milestone to exist.

## Parse `$ARGUMENTS` and route

- **`#<issue>`** (single story) → `/lib:deliver-one <issue>` — claim → (canvas if `feat:` and missing
  → `/plan #<issue>` first) → expert dispatch → `/lib:spdd-generate` per O-step → gates → PR → CI →
  squash auto-merge → `experts:post-merge` verify.
- **`#<epic>`** (resume a cluster) → `/lib:deliver-resume <epic>` — deliver the epic's ready O-steps,
  open PRs in parallel, enable auto-merge, then stop for review.
- **no argument, or `--batch N`** → `/lib:deliver-batch` — claim up to N (default 3) ready issues
  repo-wide (or within `--milestone`), route each to the right `experts/*` persona in parallel,
  run the merge pass, collect results, append learnings.
- **`<canvas-path>`** → deliver the next unimplemented Operations step of that canvas via
  `/lib:spdd-generate`.

The orchestration scope (which issues are "ready") is **provided by this dispatcher** to the lib
procedure: repo-wide ready work by default, or the `--milestone` filter. Where a lib procedure says
"active milestone", read "the scope this command passed in".

## Guardrails (preserved from the underlying machinery)
- **Idempotent & parallel-safe:** the `<type>/<N>` branch is the sole claim mutex; a lost race is a
  no-op. Worktrees are per-run and private — the user's checkout is never mutated.
- **`feat:` is canvas-gated:** no `Aligned` canvas → route to `/plan` first; never implement from an
  unaligned canvas (ADR-019).
- **Commit/PR hygiene:** conventional title, DCO `Signed-off-by`, `Assisted-by: Claude/<model>`,
  squash auto-merge (rebase blocked by `required_signatures`).
- **Stops on blockers** (CI red it can't fix, context budget) and reports how to resume — recover
  with `/reconcile`.
- **Runtime evidence, not config evidence:** for changes touching `infra/docker-compose/**`,
  `infra/helm/**`, a Makefile `demo`/`run-local`/compose target, `services/*/cmd/**`,
  `agents/adapters/**`, or `cmd/zynax*`, "done" requires actually booting the documented path and
  observing the user-facing outcome — `docker compose config`, a build, or CI-green are **not**
  runtime evidence. Re-run stateful paths **twice** on the same volumes (persistence bugs surface on
  run #2). Claim "works end-to-end" only for what was executed; label config/build-only checks as such.

## Output
Per issue: claim status, PR link, merge state, post-merge verification, and learnings recorded.
At the end: what merged, what's blocked, and the suggested next `/deliver` or `/reconcile`.

See `.claude/commands/README.md` for the full decision tree.
