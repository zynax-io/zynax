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

2. **Merge strategy: squash-merge** (`gh pr merge --squash`). The repo has
   `required_signatures` enabled; GitHub's rebase-merge cannot auto-sign the
   replayed commits, so `--rebase` is rejected. Squash-merge produces a single
   signed commit on `main` — it satisfies `required_linear_history` (no merge
   commit) and keeps a clean, signed history. Never use the "Create a merge commit"
   button — that violates `required_linear_history`.

3. **Rebase branch onto current `main` immediately before merge.** Never merge
   a branch that has diverged from `main`. The sequence is always:
   ```bash
   git fetch origin main
   git rebase origin/main          # resolve any conflicts
   git push --force-with-lease
   gh pr checks <PR> --watch
   gh pr merge <PR> --squash
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

### Settings applied (2026-06-03)

- ✅ **"Automatically delete head branches"** — enabled via API (`delete_branch_on_merge: true`)

### Known limitation — push restrictions require GitHub Team

**"Block direct pushes"** (branch protection `restrictions`) is a **GitHub Team /
Enterprise feature** and is not available on the `zynax-io` free-plan organisation.
Attempting to enable it via the API returns HTTP 404.

**Residual risk:** a contributor with write access can still `git push origin main`
directly, bypassing the PR + CI gate, as long as the commit is locally signed
(satisfying `required_signatures`). The 10 required status checks only gate PR
merges, not direct pushes.

**Mitigations in place without paid push restrictions:**

| Control | Effect |
|---------|--------|
| `required_signatures: true` | Direct push must be locally SSH-signed — accidental pushes are blocked |
| `enforce_admins: true` | All protection rules apply to the owner too |
| `required_linear_history: true` | Force-push of a merge commit is rejected |
| Process discipline (this ADR + AGENTS.md) | Explicit policy makes the bypass intentional, not accidental |
| `/resume-m6` Branch discipline section | Session-level guardrail for the AI-assisted workflow |

**Upgrade path:** Upgrading `zynax-io` to GitHub Team unlocks the `restrictions`
field. At that point, set `restrictions: {users: [], teams: [], apps: []}` to
fully block direct pushes for all actors. No ADR amendment needed — this ADR
already records the intent.

---

## Alternatives considered

| Option | Rejected because |
|--------|-----------------|
| Keep `restrictions: null`, rely on convention | Proved insufficient — the /resume-m6 "no PR" path was an explicit instruction, not an accidental omission |
| Upgrade to GitHub Team immediately to enable push restrictions | Cost vs. benefit: residual risk is low given required_signatures + process controls; revisit when team grows |
| Merge-queue (issue #544) | Adds significant tooling complexity for a solo-maintainer repo; deferred to a future ADR when team size warrants it |
| Rebase-and-merge as default | `required_signatures` prevents GitHub from auto-signing replayed commits; `gh pr merge --rebase` is rejected by the API |
