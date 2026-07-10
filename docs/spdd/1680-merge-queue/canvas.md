# REASONS Canvas — GitHub merge queue: automated, fair PR merging for agents and forks (M8.I)

> **All content in this Canvas is Tier 1 (public-safe).**
> Run `/lib:spdd-security-review <path>` before committing.

**Issue:** #1680
**Author:** Oscar Gómez Manresa
**Date:** 2026-07-07
**Status:** Implemented (2026-07-10 — epic #1680 closed; fork-canary evidence on #1685)

> Story issues: step 1 → #1681 · step 2 → #1682 · step 3 → #1683 · step 4 → #1684 ·
> step 5 → #1685

---

## R — Requirements

- **Problem.** The `main` ruleset requires every PR to be up to date with main
  (`strict_required_status_checks_policy: true`, 12 required checks). main moves 15–20
  commits/day in bursts (parallel agent sessions + digest-sync direct pushes), so every merge
  flips every open PR to `BEHIND`. Agent sessions survive via a rebase + `--force-with-lease` +
  armed `gh pr merge --squash --auto` race baked into the deliver commands; fork contributors
  have no equivalent (GitHub's update-branch merge commit carries no `Signed-off-by` → the DCO
  gate fails; Actions tokens cannot push to fork branches) and structurally lose the race —
  case study: fork PR #1668. A merge queue was enabled in M5 (#544/PR #587) and removed within
  the hour (#589): sha-keyed concurrency groups cross-cancelled queue runs and several required
  contexts had no queue-context reporter. Both causes are fixed today (ref-keyed groups; the
  `if: pull_request` guards from #587 survive; the e2e skip shim #1092 exists).
- **Done — merging is automated, fair, and validated on the merged result:**
  - GitHub merge queue (squash method) active on `main`; `strict_required_status_checks_policy`
    is `false`. Rollback is one API call: remove the queue rule, keep strict false.
  - All 12 required contexts report on `merge_group` refs (real run, guard-skip, or shim) — no
    queue entry ever stalls on "Expected".
  - Queue-leg CI wall time for a small PR stays ~5 min: the CI lane-detection (`changes`) job
    diffs `merge_group.base_sha..head_sha` instead of tripping its all-lanes fail-safe.
  - Digest-sync direct pushes defer briefly while the queue is busy (bounded wait, then push),
    preserving ADR-027 near-atomicity.
  - A PR that is green but `BEHIND` merges with zero manual rebase after arming
    `gh pr merge --squash --auto`; a maintainer-armed fork PR merges unattended.
  - ADR-047 records the decision, the 12-contexts→reporter mapping, the M5 archaeology, the
    residual risks, the cutover runbook and the rollback line; every merge-guidance surface
    (CONTRIBUTING, deliver commands, ai-learnings) describes the queue flow with a
    pre-cutover/rollback fallback.
  - Canary evidence (2–3 in-repo PRs + 1 fork-PR dry run) posted on #1685 before the epic closes.

## E — Entities

- **Ruleset `main`** — branch protection: 12 required contexts, squash-only, required
  signatures, linear history; gains a `merge_queue` rule (SQUASH), drops strict up-to-date.
- **Merge queue** — GitHub-managed FIFO; builds `gh-readonly-queue/main/*` refs = PR + current
  main (+ queued predecessors); emits `merge_group` events; ejects red entries and rebuilds.
- **Required-context reporters** — per check, one of: *real run* (lints, tests, security,
  gitleaks), *guard-skip* (`if: pull_request` job skip satisfies the check: PR size label,
  Conventional Commit title, actionlint, dco fast-exit), *shim* (e2e smoke satisfier).
- **Lane detection (`changes` job)** — computes which CI lanes run from the event's diff; needs
  a `merge_group` branch (base_sha..head_sha) so queue refs don't trigger the all-lanes
  fail-safe.
- **Digest-sync push** — direct-to-main commit (skip-ci marker) from the release workflow after
  image promotion; a direct push invalidates in-flight queue groups → gains a queue-aware
  debounce.
- **Merge-guidance surfaces** — CONTRIBUTING §merge flow; deliver command steps (ready gate,
  merge step, error table); the ai-learnings rebase-then-arm entry. All must describe one flow.

```
PR (fork or agent) --green--> armed auto-merge --> merge queue (FIFO)
                                                     |  builds gh-readonly-queue/main/<n>
                                                     v
                                       merge_group event -> 12 reporters
                                                     |  all green
                                                     v
                                     GitHub-signed squash commit on main
         release workflow --digest push (debounced while queue busy)--> main
```

## A — Approach

- **WILL:** land inert `merge_group:` reporter legs + the lane-diff fix first (step 2); debounce
  the digest push (step 3); update every guidance surface with a fallback line (step 4); flip
  the ruleset last and canary it (step 5). ADR-047 (step 1) gates all implementation — the
  decision, mapping, runbook and rollback live there. Supersedes only the merge-queue deferral
  in ADR-023; its squash-only/signature/linear-history policy stands.
- **WON'T:** no auto-update-branch bot (PAT custody risk; automates the rebase treadmill
  instead of removing it); no convention-based agent yielding (unenforceable, deadlock-prone —
  queue ordering IS the fairness mechanism); no daily digest batching (breaks ADR-027
  near-atomicity — bounded debounce instead); no real e2e on queue refs (a 45-min kind matrix
  per merge is not viable — the shim satisfies the context; post-merge verify is the backstop);
  no changes to protos, services, or runtime code.
- Governing ADRs: ADR-023 (merge strategy — partially superseded), ADR-027 (digest atomicity),
  ADR-019 (SPDD scope; stories are ci:/docs: — canvas by explicit maintainer request),
  ADR-016 (no gRPC boundary → no `.feature` file).

## S — Structure

- `.github/workflows/ci.yml` — `on:` gains `merge_group:`; `changes` job gains a merge_group
  diff branch (7 required contexts live here; dco already fast-exits on non-PR events).
- `.github/workflows/pr-checks.yml` — `on:` gains `merge_group:` (actionlint + Conventional
  Commit title guard-skip; gitleaks runs its non-PR fallback range).
- `.github/workflows/pr-size.yml` — `on:` gains `merge_group:` (guard-skip).
- `.github/workflows/e2e-smoke-skip.yml` — `on:` gains a bare `merge_group:` (path filters do
  not apply to merge_group; the shim satisfies "e2e smoke (temporal)" on every queue ref).
- `.github/workflows/release.yml` — digest-sync step gains the queue-aware bounded wait.
- `docs/adr/ADR-047-github-merge-queue.md` (+ `docs/adr/INDEX.md` row) — new.
- `CONTRIBUTING.md`, `.claude/commands/lib/deliver-one.md`, `deliver-batch.md`,
  `deliver-resume.md`, `docs/ai-learnings/ci-release.md` — guidance updates.
- Repo ruleset (settings, not files): `merge_queue` rule + strict=false — via the ADR-047
  runbook, after all file changes are on main.
- No gRPC contracts touched.

## O — Operations

> Each step = one PR (step 5 is operational). Order is load-bearing:
> #1681 → (#1682 ∥ #1683) → #1684 → #1685, with the step-5 ruleset flip executed immediately
> after #1684 merges so no live surface gives stale guidance.

1. **ADR-047** (#1681, `docs:`) — decision record superseding ADR-023's merge-queue deferral:
   context, options, 12-contexts→reporter mapping, M5 removal archaeology, residual risks
   (e2e validates the PR leg only; burst latency), cutover runbook, rollback line. INDEX row.
2. **merge_group reporter legs + lane diff** (#1682, `ci:`) — add `merge_group:` to the four
   workflow files; teach `changes` to diff `merge_group.base_sha..head_sha`; keep concurrency
   groups ref-keyed. Inert until the ruleset flips; PR-leg behaviour unchanged.
3. **Digest-push debounce** (#1683, `ci:`) — bounded wait (suggest 10 min) on merge-queue
   emptiness before the digest-sync push; log the outcome; byte-identical behaviour while no
   queue rule exists.
4. **Guidance retirement** (#1684, `docs:`) — CONTRIBUTING + deliver commands + ai-learnings:
   arm-early flow, BEHIND is cosmetic, DIRTY still means a real `--signoff` rebase, one
   re-queue retry on ejection, and a pre-cutover/rollback fallback line on every surface.
5. **Cutover + canary** (#1685, `ci:`, operational) — execute the ADR-047 runbook (queue rule
   SQUASH + strict false), then canary: 2–3 in-repo PRs + 1 fork-PR dry run through the queue;
   digest push during the window must not thrash; evidence on the issue; tick the epic boxes.

## N — Norms

- Commit hygiene: DCO `Signed-off-by` + `Assisted-by: Claude/<model>` (never `Co-Authored-By`
  for AI); SSH-signed commits; conventional types `docs:`/`ci:` per story; one PR per issue,
  one commit per logical change; PR title `<type>: <subject>` total ≤ 72 chars.
- Workflow edits: image references in banner-marked regions come from `images/images.yaml`
  (`make sync-images`); `make lint` includes actionlint via the tools image; keep `.github/
  workflows/` consistent with the fork-PR `workflow_run` poster pattern (read-only PR tokens).
- Docs: repo-relative paths only; no literal skip-ci token in commit messages or PR bodies
  (write "skip-ci marker"); no literal e-mail addresses in committed text.
- Canvas discipline (ADR-019): this canvas is committed before any implementation PR;
  requirements change → `/lib:spdd-prompt-update` first, never patch code first.

## S — Safeguards (second S)

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /lib:spdd-security-review passed on this file

### Feature Safeguards
- Never flip the ruleset (step 5) before the `merge_group` reporter legs (#1682) are on `main`
  — queue entries would stall on "Expected"; this is the M5 failure mode.
- Never re-key workflow concurrency groups by SHA for queue events — `gh-readonly-queue/*`
  refs must never cross-cancel (the other M5 killer).
- Never weaken the PR-leg gates: DCO, gitleaks, conventional-commit and size checks keep full
  enforcement on `pull_request`; queue-leg skips satisfy already-validated PR state only.
- Never batch digest syncs beyond a bounded debounce — ADR-027 near-atomicity stands; on
  timeout, push anyway.
- Never remove the strict-mode fallback guidance from CONTRIBUTING/deliver commands until the
  queue has survived its canary — rollback (queue off, strict stays false) must leave every
  surface accurate.
- Never use GitHub's update-branch API on PRs — its merge commit lacks `Signed-off-by` and
  fails the DCO gate (standing rule, unchanged by this epic).
