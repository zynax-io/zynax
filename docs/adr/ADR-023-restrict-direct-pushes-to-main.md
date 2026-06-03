<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-023 — Restrict Direct Pushes to `main`; Rebase-Merge Only

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-03 |
| **Deciders** | Oscar Gómez Manresa |
| **Scope** | Branch protection, merge strategy, branch lifecycle |

---

## Context

`main` has `required_linear_history: true` and `enforce_admins: true` enforced
via branch protection, preventing merge commits for all contributors including
the owner. However, `restrictions: null` (no push restriction) means anyone with
write access can bypass the PR + CI gate by pushing directly to `main`. This gap
was exploited by the `/resume-m6` Step 1.3 instruction ("small docs: commit,
no PR"), which pushed milestone reconciliation edits straight to `main` without
running CI or opening a review surface.

The direct-push path is a one-way door risk: once a bad commit lands on `main`
without CI, rolling it back requires either a revert PR (which must pass CI) or
a force-push (which destroys history). The `enforce_admins: true` rule was
chosen precisely to close this gap for all actors — a push-restriction closes
the remaining bypass.

### Failure mode this ADR prevents (PR #775)

Three commits were bundled (docs + chore + feature) onto a single branch that
drifted behind `main`. The branch was closed without merging; commits stranded.
Root cause: the `/resume-m6` command encouraged the "no PR" direct-push for doc
changes, creating a pattern where not all changes to `main` went through CI.

---

## Decision

1. **Enable "Block direct pushes" / "Restrict pushes that create matching
   branches"** on the `main` branch protection rule. This prevents any actor —
   including repo admins — from pushing directly to `main`. All changes must go
   through a PR.

2. **Merge strategy: rebase-merge only** (`gh pr merge --rebase`). Squash-merge
   is permitted only when a branch has multiple intermediate commits that are
   noise (e.g. fixup commits during review); in that case the author interactively
   squashes locally before opening the PR, so the PR itself contains one clean
   commit that can be rebase-merged. The "squash-and-merge" button in the GitHub
   UI must not be used — it obscures authorship and discards the original commit
   SHA.

3. **Rebase branch onto current `main` immediately before merge.** Never merge
   a branch that has diverged from `main`. The sequence is always:
   ```bash
   git fetch origin main
   git rebase origin/main          # resolve any conflicts
   git push --force-with-lease
   gh pr checks <PR> --watch
   gh pr merge <PR> --rebase
   ```

4. **Delete the remote branch immediately after merge:**
   ```bash
   git push origin --delete <branch>
   ```
   No merged or closed branch should remain on the remote. GitHub's
   "Automatically delete head branches" repository setting should be enabled to
   enforce this for merge-button operations.

5. **Never reopen a closed PR or stale branch.** If commits from a closed branch
   are still wanted, cherry-pick or rebase them onto a **fresh branch off current
   `main`**, open a new PR, run CI, then rebase-merge. This ensures CI always
   runs against current HEAD before any commit touches `main`.

---

## Consequences

### Positive

- Every commit on `main` passed CI against the HEAD it was rebased onto.
- No actor can sneak a "quick fix" to `main` without review surface and CI.
- Branch list stays clean (no abandoned branches cluttering `git branch -r`).
- The PR #775 failure mode (stale branch + no-CI doc push) cannot recur.

### Negative / trade-offs

- Slightly more friction for the solo-maintainer phase: every change, including
  a one-line doc fix, requires a branch + PR + CI wait (~2 min). This is
  acceptable given the CI suite is fast and the correctness guarantee is worth it.
- Force-pushes during active review (after rebase) require `--force-with-lease`,
  which is already the repo convention (`CONTRIBUTING.md §8`).

### Settings change required (manual, not automated)

In GitHub → Settings → Branches → `main` rule:
- Enable **"Block direct pushes"** (or equivalent "Restrict pushes" option)
- Enable **"Automatically delete head branches"** (repository-level setting)

These are not applied by this ADR commit — they are a manual settings action
by the repo owner after this ADR is merged.

---

## Alternatives considered

| Option | Rejected because |
|--------|-----------------|
| Keep `restrictions: null`, rely on convention | Proved insufficient — the /resume-m6 "no PR" path was an explicit instruction, not an accidental omission |
| Merge-queue (issue #544) | Adds significant tooling complexity for a solo-maintainer repo; deferred to a future ADR when team size warrants it |
| Squash-and-merge as default | Discards original commit SHAs, making `git bisect` and `git blame` less useful; rebase-merge is strictly better for the repo's single-author style |
