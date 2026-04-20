<!-- SPDX-License-Identifier: Apache-2.0 -->

# Git Workflow — Zynax

> This document is the definitive reference for how Zynax manages its git history.
> Clean history is a first-class project value: it is documentation, audit trail,
> and bisection tool all at once.

---

## Table of Contents

1. [Philosophy](#1-philosophy)
2. [Branch Strategy](#2-branch-strategy)
3. [Commit Atomicity](#3-commit-atomicity)
4. [Commit Message Anatomy](#4-commit-message-anatomy)
5. [Cleaning History Before Review](#5-cleaning-history-before-review)
6. [PR Merge Strategy](#6-pr-merge-strategy)
7. [Stacked PRs](#7-stacked-prs)
8. [Release Branches](#8-release-branches)
9. [Hotfixes](#9-hotfixes)
10. [Tagging & Versioning](#10-tagging--versioning)
11. [Rebasing Rules](#11-rebasing-rules)
12. [Common Mistakes](#12-common-mistakes)

---

## 1. Philosophy

Git history is the story of WHY the codebase is the way it is. A future contributor
will run `git log`, `git show`, and `git blame`. What they find should read like
precise technical prose — each change self-contained, well-motivated, and traceable
to a decision.

Three properties we optimise for:

| Property | What it enables |
|----------|----------------|
| **Atomicity** | Each commit can be cherry-picked, reverted, or bisected independently |
| **Motivation** | Every commit body answers "why was this necessary?" |
| **Traceability** | Every commit links to an issue or ADR |

These are not aesthetic preferences. They are operational requirements:
- Bisection on a large codebase requires atomic commits.
- Release notes are generated from commit messages.
- Security audits trace changes to decisions.

---

## 2. Branch Strategy

Zynax uses **trunk-based development**: all work happens in short-lived branches
off `main`. Long-lived feature branches are prohibited.

### Branch Name Format

```
<type>/ISSUE-<number>-<short-description>
```

| Component | Rule |
|-----------|------|
| `type` | One of: `feat`, `fix`, `docs`, `test`, `refactor`, `ci`, `chore`, `release` |
| `ISSUE-<number>` | Required for `feat` and `fix`. Omit for `docs`, `ci`, `chore`. |
| `short-description` | ≤ 5 words, lowercase, hyphen-separated |

**Valid branch names:**
```
feat/ISSUE-123-capability-based-discovery
fix/ISSUE-456-retry-storm-backoff
docs/ISSUE-789-adapter-pattern-guide
test/ISSUE-101-add-bdd-for-memory-ttl
refactor/ISSUE-202-extract-routing-logic
ci/update-golangci-lint-version
chore/remove-deprecated-agent-sdk-shims
release/v0.2.0
```

**Invalid:**
```
my-feature              ← no type prefix
feat/new-stuff          ← no issue number, no description
ISSUE-123               ← no type, description required
feature/ISSUE-123       ← "feature" not a valid type
```

### Branch Lifetime

| Branch type | Max lifetime |
|-------------|-------------|
| `feat/*`, `fix/*` | 7 days active work; merge or convert to Draft |
| `docs/*`, `ci/*`, `chore/*` | 3 days |
| `release/*` | Until release is tagged, then deleted |
| Stale branches | Deleted after 30 days of no activity |

Short lifetimes prevent merge conflicts and force decomposition of large features.

### Protected Branches

| Branch | Protection |
|--------|-----------|
| `main` | No direct push. PRs required. CI must pass. 1–2 approvals required. |
| `release/*` | No direct push except by release manager. Tags only from release branches. |

---

## 3. Commit Atomicity

### The Rule

One commit = one logical change that:
1. Leaves the codebase in a **compilable, working state**
2. Changes **one thing** (the problem, not the solution and the refactor and the fix)
3. Can be **reverted independently** without breaking other commits
4. Delivers **observable value** — something a test or a reviewer can verify

Point 4 is the principle behind PR decomposition. Do not split a PR to hit a
line-count target if every resulting sub-PR would have no independently observable
behaviour. A small PR with no functional output is worse than a larger PR that
delivers a complete, testable capability.

### What "One Logical Change" Means

**Atomic (good):**
```
feat(agent-registry): add capability index data structure

feat(agent-registry): wire capability index into registration handler

feat(agent-registry): expose QueryByCapability RPC method

test(agent-registry): BDD scenarios for capability query
```
Four commits, four concerns. Each can be understood, reviewed, and reverted alone.

**Non-atomic (bad):**
```
add capability discovery feature and fix the existing tests that were
broken and also update the proto and add some docs
```
Everything tangled. Cannot bisect. Cannot revert the proto change without reverting
everything. Reviewer must understand all concerns simultaneously.

### The "Fixup Commit" Pattern

During development you will make fixup commits — that's fine. Clean them up
before the PR is ready for review (see §5).

```bash
# While working — messy is OK
git commit -m "wip: trying the index approach"
git commit -m "fix: oops wrong field name"
git commit -m "add test for empty result"

# Before opening PR — clean it up
git rebase -i main
```

### Commit Granularity Guide

| Change | Separate commit? |
|--------|-----------------|
| Proto definition change | Yes — always separate from implementation |
| Generated code (`*.pb.go`) | Same commit as the proto it derives from |
| Domain model change | Yes |
| Infrastructure change that uses the domain | Yes (separate from domain) |
| Test for a specific behaviour | Same commit as the code that makes it pass, OR a preceding commit that defines the BDD scenario |
| BDD `.feature` file | Separate, committed BEFORE implementation |
| Docs update for a feature | Separate commit or same as the feature commit in small PRs |
| Linting / formatting fixes | Separate from functional changes |
| Dependency version bump | Separate commit |

---

## 4. Commit Message Anatomy

### Format

```
<type>(<scope>): <short description>       ← subject line ≤ 72 chars
<blank line>
<body>                                      ← 72 chars per line, explain WHY
<blank line>
<footer>                                    ← Closes, BREAKING CHANGE, Signed-off-by
```

### Subject Line Rules

- Imperative mood: "Add", "Fix", "Remove" — not "added", "fixes", "removed"
- Capitalize the first word of the description (after the colon)
- ≤ 72 characters total for the full header (enforced by `commitlint`)
- No period at the end
- No `@mentions` anywhere in commit messages — GitHub references belong in the footer only
- No emojis
- `scope` matches the service or layer affected

### Body Rules

The body is **mandatory for `feat` and `fix`**. Optional for `docs`, `ci`, `chore`.

The body must answer: **Why was this change necessary? What problem does it solve?
What alternatives were considered?**

It must NOT describe WHAT the code does — the diff shows that. It must describe
the reasoning that is not visible in the code.

**Good body:**
```
The task-broker was calling FindAgents on the registry synchronously in the
hot path for every task submission. With 10k tasks/sec and 50ms registry
P99, this was adding 500ms median latency to task assignment.

Moving to a local capability cache (refreshed every 5s via watch stream)
reduces registry calls by 99% and brings task assignment latency back under
5ms P99. Cache invalidation is safe: agents that deregister mid-stream get
a "no agents available" error which triggers retry.

Alternative considered: denormalising capabilities into the task queue.
Rejected because it would couple task-broker domain to agent-registry schema.
```

**Bad body:**
```
Fixed the performance issue with the registry.
```

### Footer Rules

Required footers:
- `Closes #<number>` or `Fixes #<number>` — links to the GitHub issue
- `Signed-off-by: Full Name <email>` — DCO sign-off (always required)

Optional footers:
- `BREAKING CHANGE: <description>` — for breaking changes (also add `!` to type)
- `Co-Authored-By: Name <email>` — for human pair authors only
- `Assisted-by: ToolName/model-id` — for AI tool attribution (not `Co-Authored-By:`)
- `Refs #<number>` — related issues that are not closed by this commit

### Commit Signature Verification

Every commit must be **GPG-signed**. This is enforced at the branch protection
level — unsigned commits are rejected at push.

Configure once:
```bash
# Set your signing key
git config --global user.signingkey <YOUR_GPG_KEY_ID>

# Sign every commit automatically (no -S flag needed)
git config --global commit.gpgsign true

# Verify a commit is signed
git log --show-signature -1
```

The DCO `Signed-off-by:` and GPG signature are both required and independent:
- `commit.gpgsign true` — cryptographic proof of authorship
- `git commit -s` — legal DCO certification

A properly formed commit footer looks like:
```
Closes #123
Signed-off-by: Jane Doe <jane@example.com>
Assisted-by: Claude Code/claude-sonnet-4-6
```
`Co-Authored-By:` is for human pair authors only. `Signed-off-by:` certifies the DCO.
Neither tag may contain an AI tool.
The GPG signature is stored in the git object, not in the commit message body.

### Complete Example

```
fix(task-broker): prevent retry storm when all agents unavailable

When the last agent for a capability deregisters, the task-broker was
retrying assignment immediately in a tight loop. Under load, this caused
CPU to spike to 100% and starved all other goroutines.

The fix adds exponential backoff with full jitter (1s base, 30s cap) for
the "no agents available" case. The jitter prevents thundering herd when
agents come back online simultaneously.

Tested with a 10k task load while cycling all agents: previously caused
OOM in ~30s; with fix, CPU stays under 15% and tasks drain normally.

Closes #456
Signed-off-by: Jane Doe <jane@example.com>
```

---

## 5. Cleaning History Before Review

Before converting a Draft PR to "Ready for Review", your branch history must be
clean. This means:
- No "WIP" commits
- No "fix typo" commits
- No "address review comments" commits
- No merge commits from `main` (use rebase instead)
- Every commit message follows the format in §4

### Interactive Rebase Workflow

```bash
# See what you have
git log --oneline main..HEAD

# Interactive rebase — opens your editor
git rebase -i main
```

In the editor, you will see:
```
pick abc1234 feat(agent-registry): initial capability index structure
pick def5678 wip: trying different approach
pick ghi9012 fix: wrong field name
pick jkl3456 add test
```

Rewrite it as:
```
pick abc1234 feat(agent-registry): add capability index data structure
squash def5678 wip: trying different approach
squash ghi9012 fix: wrong field name
pick jkl3456 test(agent-registry): BDD scenarios for capability query
```

The squash merges commits together; you then rewrite the combined message.

### Rebase vs Merge for Staying Current

**Always rebase, never merge from `main` into your branch:**

```bash
# Wrong: creates a merge commit in your branch history
git merge main

# Right: replays your commits on top of current main
git fetch origin
git rebase origin/main
```

Why: merge commits in feature branches pollute the squash commit history and
make interactive rebase harder.

### Force-Push Rules

After rebasing, you must force-push your branch:
```bash
git push --force-with-lease origin feat/ISSUE-123-capability-discovery
```

`--force-with-lease` is safer than `--force`: it fails if someone else pushed
to the same branch since your last fetch, preventing accidental overwrites.

**Never force-push to `main` or `release/*`.**

### During Review — Do Not Force-Push

While a PR is under active review, do not amend commits or rebase. Push fixup
commits instead so reviewers can see what changed between review rounds:

```bash
git commit -s --fixup HEAD~1   # or --fixup <sha> to target a specific commit
git push                        # no force-push needed for fixups
```

Once all approvals are in and blocking comments resolved, squash and push:

```bash
git fetch origin
git rebase -i origin/main --autosquash   # collapses fixups automatically
git push --force-with-lease
```

---

## 6. PR Merge Strategy

### Squash-and-Merge (Default)

Feature branches are squash-merged into `main`. The squash commit:
- Title = PR title (must be a valid Conventional Commit message)
- Body = PR description's "Why" section
- Footer = `Closes #<issue>`, `Signed-off-by`, `Assisted-by` (if AI tools were used)

Result: `main` history has exactly **one commit per PR**. This is clean,
bisectable, and maps directly to `CHANGELOG.md` entries.

### Rebase-Merge (PR Chains Only)

When a PR is part of a stacked chain and each commit represents a meaningful
independent change, use rebase-merge to preserve the individual commits.

Declare this in the PR description:
```
Merge strategy: rebase-merge (stacked PR, commits are independent)
```

### Never Merge Commits on Main

Merge commits (the three-parent kind) are never used on `main`. They make
`git bisect` and `git log --oneline` harder to read.

---

## 7. Stacked PRs

Stacked PRs (also called "PR chains") are the mechanism for shipping a large
feature as a sequence of small, reviewable, independently-mergeable changes.

### Setup

```bash
# Base branch
git checkout main
git checkout -b feat/ISSUE-123-protos

# ... implement proto changes ...
git commit -s -m "feat(protos): add capability fields to AgentSpec"

# Stack next branch on top of first
git checkout -b feat/ISSUE-123-registry
# ... implement registry changes ...
git commit -s -m "feat(agent-registry): index capabilities on registration"
```

### Opening Stacked PRs

Open PRs in order:
- PR #201: `feat/ISSUE-123-protos` → target `main`
- PR #202: `feat/ISSUE-123-registry` → target `feat/ISSUE-123-protos`

In PR #202's description, add:
```
Stacked on #201. Base will change to `main` once #201 merges.
```

### After Base PR Merges

When PR #201 merges to `main`:
```bash
git checkout feat/ISSUE-123-registry
git rebase origin/main   # Move base from merged branch to main
git push --force-with-lease origin feat/ISSUE-123-registry
```

Then update PR #202's base branch to `main` on GitHub.

---

## 8. Release Branches

Releases are cut by maintainers following the process in `GOVERNANCE.md`.

```bash
# Cut a release branch from main
git checkout main
git pull
git checkout -b release/v0.2.0

# Final pre-release fixes (cherry-pick only, no new features)
git cherry-pick <commit-hash>

# Tag the release
git tag -s v0.2.0 -m "v0.2.0: Temporal execution (M3)"
git push origin release/v0.2.0
git push origin v0.2.0
```

Release branches are never merged back to `main`. Post-release fixes go into
`main` first and are cherry-picked to the release branch.

---

## 9. Hotfixes

A hotfix is a critical fix for an issue in a tagged release.

```bash
git checkout v0.2.0              # Start from the tag
git checkout -b fix/ISSUE-789-critical-auth-bypass

# ... fix the issue ...
git commit -s -m "fix(api-gateway): prevent unauthenticated capability invocation

BREAKING CHANGE: none. This fixes a privilege escalation in the capability
dispatch path. The check was skipped when the request carried an empty
principal claim rather than a missing one.

CVE: pending GHSA assignment
Closes #789
Signed-off-by: Jane Doe <jane@example.com>"

# Open PR against the release branch AND against main
```

Hotfix PRs are opened against both the release branch and `main`. The fix
must be in `main` before it is cherry-picked to the release branch.

---

## 10. Tagging & Versioning

Zynax follows [Semantic Versioning 2.0](https://semver.org):

```
v<MAJOR>.<MINOR>.<PATCH>

MAJOR: breaking changes in public APIs or proto contracts
MINOR: new capabilities, backward-compatible
PATCH: bug fixes, backward-compatible
```

Tags are:
- GPG-signed by the release manager: `git tag -s v0.2.0 -m "..."`
- Annotated with the milestone summary
- Generated from `CHANGELOG.md` entries (which come from Conventional Commits)

Pre-release tags: `v0.2.0-alpha.1`, `v0.2.0-rc.1`

---

## 11. Rebasing Rules

| Action | Allowed? | Notes |
|--------|----------|-------|
| `rebase -i` on your own branch | Yes | Clean history before review |
| `rebase origin/main` on your branch | Yes | Stay current with main |
| Force-push your own branch | Yes | Use `--force-with-lease` |
| Force-push `main` | Never | Hard protected |
| Force-push `release/*` | Never | Hard protected |
| Rebase a branch that others are working on | No | Coordinate first |
| Amend a commit that is already in a PR | Only if PR has no approvals | Notify reviewers |

---

## 12. Common Mistakes

| Mistake | Why it matters | Fix |
|---------|---------------|-----|
| Merge commit from `main` in feature branch | Pollutes history, harder to squash | `git rebase origin/main` instead |
| "WIP" commit in final PR | Not a meaningful history entry | `git rebase -i main` and squash |
| Empty commit body on `feat`/`fix` | Loses the "why" forever | Edit with `git rebase -i` before push |
| Missing `Closes #` footer | Issue stays open, no traceability | Add to commit footer or PR description |
| Missing `Signed-off-by` | DCO bot blocks merge | `git commit -s` always |
| Branch off stale local `main` | Work based on old code | `git fetch && git checkout -b feat/... origin/main` |
| Committing generated code separately from proto | Out-of-sync generated files | Same commit as the `.proto` that caused the regeneration |
| `git push --force` without `--lease` | Can overwrite collaborator's push | Always `--force-with-lease` |
