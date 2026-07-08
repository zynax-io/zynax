<!-- SPDX-License-Identifier: Apache-2.0 -->
# ADR-047: GitHub merge queue (squash) replaces the strict up-to-date requirement

**Status:** Accepted  **Date:** 2026-07-08
**Related:** ADR-023 (**supersedes its merge-queue deferral only** — the Alternatives table deferred merge queue as "significant tooling complexity for a solo-maintainer repo"; that premise is obsolete with parallel agent sessions plus external fork contributors. Everything else in ADR-023 stands: squash-only, `required_signatures`, `required_linear_history`, no direct pushes except the ADR-027 bot exception), ADR-027 (**upheld** — the digest-sync direct push keeps near-atomicity; it gains a bounded queue-aware debounce, never daily batching), ADR-019 (SPDD — this epic's stories are `ci:`/`docs:` and canvas'd by explicit maintainer request: `docs/spdd/1680-merge-queue/canvas.md`), ADR-016 (no gRPC boundary → no `.feature` gate). Resolves EPIC #1680 (M8.I); case study fork PR #1668.

---

## Context

- **The BEHIND race.** The `main` ruleset requires PR branches to be up to date
  (`strict_required_status_checks_policy: true`) before its 12 required checks count. `main`
  receives 15–20 commits/day in bursts (parallel agent sessions; digest-sync direct pushes are
  ~20% of churn). Every merge flips every open PR to `BEHIND` and restarts its CI.
- **Structural unfairness to forks.** Agent sessions survive by racing: rebase onto
  `origin/main` + `git push --force-with-lease` + armed `gh pr merge --squash --auto`
  (codified in the deliver commands and `docs/ai-learnings/ci-release.md`). Fork contributors
  have no equivalent: GitHub's update-branch API produces a merge commit without
  `Signed-off-by` (fails the DCO gate), and Actions tokens cannot push to fork branches. Fork
  PR #1668 flipped `BEHIND` repeatedly while agent PRs out-rebased it — the contributor
  experience M8 (CNCF Sandbox) is supposed to deliver breaks exactly here.
- **The M5 attempt and why it failed.** #544/PR #587 enabled a merge queue and added
  `merge_group` triggers; #588 patched concurrency; #589 removed everything within the hour.
  Post-mortem (git archaeology, 2026-07-07): (a) concurrency groups keyed by SHA
  cross-cancelled queue runs, and (b) several required contexts had no queue-context
  reporter, so entries stalled on "Expected". Both causes are gone: concurrency groups are
  ref-keyed (`gh-readonly-queue/main/*` refs are unique per group), the `if: pull_request`
  guards from #587 survive in `pr-checks.yml`/`pr-size.yml`, and the e2e required-check shim
  (#1092, `e2e-smoke-skip.yml`) now exists.
- **Not deciding** keeps the maintainer hand-rebasing every fork PR, keeps agents burning CI
  on rebase storms, and leaves merge order decided by who pushes fastest.

---

## Decision

**We will enable the GitHub merge queue on `main` with the squash merge method, set
`strict_required_status_checks_policy: false`, and require every required status-check
context to have a `merge_group` reporter. The queue validates each PR merged with current
main (plus queued predecessors) on `gh-readonly-queue/main/*` refs; contributors and agents
arm `gh pr merge --squash --auto` and never rebase for freshness. Rollback is one API call:
remove the queue rule, keep strict false.**

1. **Reporter mandate.** A required context with no `merge_group` reporter stalls the queue —
   the M5 failure mode. Every one of the 12 contexts reports via one of three shapes:

   | Required context | Workflow | merge_group reporter |
   |---|---|---|
   | dco | `ci.yml` | real run — non-PR events fast-exit 0 (verified on the PR leg) |
   | lint-proto · lint-go · lint-python | `ci.yml` | real run (lane-gated by `changes`) |
   | test-unit · test-integration | `ci.yml` | real run (lane-gated by `changes`) |
   | security | `ci.yml` | real run |
   | GitHub Actions workflow lint | `pr-checks.yml` | guard-skip (`if: pull_request` — a skipped job satisfies a required check) |
   | Conventional Commit title | `pr-checks.yml` | guard-skip (PR-metadata check; already validated on the PR leg) |
   | Secret scan (gitleaks) | `pr-checks.yml` | real run (non-PR fallback range) |
   | PR size label | `pr-size.yml` | guard-skip |
   | e2e smoke (temporal) | `e2e-smoke-skip.yml` | shim — always satisfies on queue refs (path filters do not apply to `merge_group`) |

2. **Queue-leg cost control.** `ci.yml`'s `changes` job diffs
   `github.event.merge_group.base_sha..head_sha`; without this its unknown-event fail-safe
   enables all lanes and every queue entry runs the full pipeline (~15–25 min instead of ~5).
3. **Direct pushes yield to the queue.** The ADR-027 digest-sync push waits (bounded, then
   pushes anyway) while the queue is non-empty — a direct push to `main` invalidates every
   in-flight queue group.
4. **Cutover is order-sensitive.** The ruleset gains the `merge_queue` rule only after the
   `merge_group` reporter legs are on `main`, and immediately after the guidance surfaces
   (CONTRIBUTING, deliver commands) flip to the queue flow. Runbook (ruleset `17547241`):
   `gh api -X PUT repos/zynax-io/zynax/rulesets/17547241` with the existing rules plus a
   `merge_queue` rule (`merge_method: SQUASH`) and `strict_required_status_checks_policy`
   set `false`, followed by a canary: 2–3 low-stakes in-repo PRs plus one fork-PR dry run
   through the queue (evidence on #1685).
5. **Rollback.** Remove the `merge_queue` rule; **keep strict false**. Merging then degrades
   to plain armed auto-merge (green merges regardless of staleness) — the Option B posture
   below, still strictly better than the pre-ADR state. Restoring strict true is a separate,
   deliberate decision, not part of rollback.

---

## Rationale

| Option | Assessment |
|--------|------------|
| Merge queue (squash) + strict false | ✅ Chosen — removes the rebase treadmill entirely; FIFO order is structurally fair to forks (queue refs build in base-repo context, no fork-token limits); CI validates the actual merged result |
| Drop strict only, no queue | ✗ Rejected as the end state — green-but-stale PRs merge unvalidated against current main; semantic conflicts land and are caught post-merge. Retained as the **rollback posture** |
| Auto-update-branch bot (maintainer PAT) | ✗ Rejected — automates the treadmill instead of removing it (every main push re-runs CI on every open PR); PAT custody risk on a public repo; the race remains |
| Convention-based agent yielding to fork PRs | ✗ Rejected — unenforceable across parallel sessions, deadlock-prone, and merging third-party code with 0 required approvals is a governance question; queue ordering IS the fairness mechanism |
| Keep the status quo | ✗ Rejected — fork PRs structurally cannot win the race (#1668); maintainer becomes the rebase bot |

---

## Consequences

- **Positive:** nobody rebases to merge — arm `gh pr merge --squash --auto` and walk away;
  fork PRs merge on their turn unattended; deliver commands lose the rebase/force-push
  preamble (simpler, fewer failure modes); required checks run against the true merged
  result; fewer wasted CI re-runs from rebase storms.
- **Negative / trade-off:** checks run twice per PR (PR leg + queue leg — mitigated by the
  lane diff); **e2e validates the PR leg only** (the shim satisfies the queue leg; a
  45-minute kind matrix per merge is not viable) — post-merge verification remains the e2e
  backstop; a red queue entry restarts the entries behind it, adding latency during bursts.
- **Neutral / follow-up:** `DIRTY` (real textual conflicts) still requires a manual
  `git rebase --signoff` — the queue does not resolve conflicts; the update-branch API
  remains forbidden (DCO); revisit queue batching settings and real-e2e-on-queue if e2e
  wall time drops; guidance surfaces keep a pre-cutover/rollback fallback line until the
  canary has passed.
