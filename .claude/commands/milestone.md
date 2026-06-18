---
description: Release-lifecycle command (the only milestone-aware one). `open` scaffolds the next milestone; `close` verifies done, tags a signed version, publishes a release, and rotates state/milestone.yaml. No-arg prints current milestone status (read-only). Everything else in this command set is milestone-agnostic and uses an optional --milestone filter instead.
argument-hint: "[open <name> <version> \"<title>\" | close [--dry-run]]   default: print status"
---

# /milestone — open · close · status

Milestones are **optional** in this command set — `/plan` and `/deliver` work repo-wide and treat
`--milestone` as a filter. This single command is the one place milestones are first-class: it owns
the **release lifecycle** and is (with nothing else) the sole writer of `state/milestone.yaml`.

## Parse `$ARGUMENTS`

- **no argument** → print the current milestone status: read `state/milestone.yaml` + `state/current-milestone.md`
  and the live GitHub milestone counts. Read-only.
- **`open <name> <version> "<title>"`** → `/lib:milestone-open` — create the GitHub milestone, scaffold
  the planning doc skeleton, and fill the `active` block of `state/milestone.yaml`.
- **`close [--dry-run]`** → `/lib:milestone-close` — verify every EPIC is done, cut a **signed** version
  tag, publish the GitHub Release, and rotate the active milestone into `history` in `state/milestone.yaml`.

## Process note (changed)
`close` no longer auto-runs the repo truth-pass. Before closing, it will **suggest** running
`/reconcile` to reconcile status surfaces and prune branches — but reconciliation is now an
on-demand action, not an automatic step in the lifecycle.

## Guardrails
- Only `/milestone open` and `/milestone close` write `state/milestone.yaml` — never edit it by hand.
- `close` requires an active milestone whose exit criteria are met; `--dry-run` reports readiness
  without tagging or rotating.
- Tagging uses signed tags; never disable signing.

See `.claude/commands/README.md`.
