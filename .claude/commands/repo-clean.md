---
description: Truth-pass for the repo — reconcile every status surface (README/ROADMAP/ARCHITECTURE/CLAUDE/state/milestone-planning/SPDD canvases) to LIVE GitHub state, triage the pile of [AUTO] pipeline issues (digest-drift / smoke / size / security), close stale issues with traceable reasons, and prune merged/orphaned branches (local + remote) to keep the repo clean. Plans first, executes only on approval, delivers file changes via one signed PR.
argument-hint: "[--execute] [--include-stale] [--milestone M6]   default: PLAN only, no --include-stale"
---

# /repo-clean — Repository Truth-Pass & Clean Snapshot

Bring every artifact the agents and humans rely on into agreement with the **actual** state of the
repository — then leave it well-linked and traceable. This is a *truth-pass*, not a refactor.

> **Why this exists.** Doc drift is silent: no CI gate flags a stale milestone marker or a canvas
> `Status:` that lags reality, so it compounds every orchestration iteration. The external
> architectural review made this a first-class concern — recommendation **C1: "Align documentation
> with reality"**, classified under **Trust / truthfulness**
> ([docs/architecture/2026-05-18-external-architectural-review.md:708](docs/architecture/2026-05-18-external-architectural-review.md#L708)).
> The platform-engineering review opens the same way: *"Reported honestly so the team can trust the
> rest"* ([docs/architecture/2026-05-22-platform-engineering-review.md:382](docs/architecture/2026-05-22-platform-engineering-review.md#L382)).
> The canonical reference run is commit `ec22a25` (*"docs: reconcile M6 status surfaces"*, #1014) —
> this command generalises that one-off into a repeatable pass.

> **Rules are not restated.** Domain and contribution rules live in [AGENTS.md](AGENTS.md),
> [CLAUDE.md](CLAUDE.md), and [docs/git-workflow.md](docs/git-workflow.md). The reconcile logic
> mirrors [.claude/commands/m6-issue-generate.md](.claude/commands/m6-issue-generate.md) STEP 6.
> This file is the *cleanup loop* only.

---

## Operating contract (read before doing anything)

- **Two phases.** Default is **PLAN** — read live state, compute divergence, print a traceable plan,
  and **stop**. Mutations (issue closes, **branch deletions**, the reconcile PR) run **only** when
  invoked with `--execute` *or* when the human explicitly says "go" after seeing the plan. Never
  close an issue, delete a branch, or push a branch in PLAN phase.
- **Live state is the source of truth — never memory or the previous doc snapshot.** Every decision
  is driven by `gh issue list` / `gh pr list` / `gh api .../milestones` / `make check-images`.
- **File changes go through PRs (never `main` directly).** Branch protection requires SSH commit
  signing, DCO `Signed-off-by`, and **squash** merge. A pass produces up to **two** PRs, kept
  separate by concern: (1) the **truth-pass PR** — doc surfaces + `images.yaml`; (2) the
  **expert-learnings PR** — opened by `/m6-learn --apply` for `.claude/commands/experts/*`. Issue
  closes happen directly via `gh` and **reference the truth-pass PR**.
- **Hard constraints (from [AGENTS.md §Hard Constraints](AGENTS.md#hard-constraints) + repo memory):**
  - Commit type must be `docs:` for a pure doc reconcile, `ci:`/`chore:` if it only touches
    `images.yaml`/automation. Never invent a type — only `feat|fix|refactor|docs|test|ci|chore`.
  - Every commit: a DCO `Signed-off-by:` trailer (your configured git identity — see
    [docs/git-workflow.md](docs/git-workflow.md)) and `Assisted-by: Claude/<model>`.
    **Never** `Co-Authored-By` for AI. **Never** disable
    signing (`-c commit.gpgsign=false`).
  - Merge is squash-only (`gh pr merge --squash`); `--rebase` is rejected by `required_signatures`.
  - PR size budget (CLAUDE.md): a reconcile PR is docs-heavy and usually well under 200 lines;
    if `images.yaml` regen balloons it, note it in the PR body.
- **`AGENTS.md` is never touched by this command.** By its own charter it holds immutable
  principles, not milestone status. Same for ADRs under [docs/adr/](docs/adr/) — those are decisions,
  not state.

---

## STEP 0 — Isolated worktree (leave the user's checkout untouched)

Run all git operations in a throwaway worktree detached at `origin/main`, exactly like the
orchestrator does — the user's working directory is left exactly as they had it.

```bash
RUN_ID="$(date +%s)-$$"
REPO=$(git rev-parse --show-toplevel)
WT="/tmp/zynax-repo-clean-${RUN_ID}"
git -C "$REPO" worktree remove "$WT" --force 2>/dev/null || true
rm -rf "$WT" 2>/dev/null || true
git -C "$REPO" fetch origin --prune
git -C "$REPO" worktree add "$WT" origin/main   # detached at latest main
cd "$WT"
```

Parse args: `--execute` (skip the approval gate), `--include-stale` (also propose closing
non-AUTO stale/superseded issues — off by default because it's the highest-judgment action),
`--milestone <M?>` (override; default = active milestone read in STEP 1).

---

## STEP 1 — Snapshot live state

Read **only** these, and read them fresh. Determine the active milestone from the repo, do not
assume M6.

```bash
# Active milestone + per-surface current markers
sed -n '1,40p' state/current-milestone.md          # Status Summary table → active milestone
# Live issue / PR / milestone state
gh issue list  --state open --limit 300 --json number,title,labels,milestone,createdAt,updatedAt,author
gh pr    list  --state open --limit 100 --json number,title,headRefName,isDraft,author
gh api repos/:owner/:repo/milestones --jq '.[] | "\(.title)\topen:\(.open_issues)\tclosed:\(.closed_issues)\tstate:\(.state)"'
# Image source-of-truth drift (images/images.yaml is the SoT — ADR-024)
make check-images 2>&1 | tail -40 || true
```

Group the open `[AUTO]` issues by **family** (title prefix), newest-first within each family:

| Family | Title prefix | Created by |
|--------|--------------|-----------|
| digest-drift | `[AUTO] images.yaml digest drift detected` | [post-merge-completeness.yml](.github/workflows/post-merge-completeness.yml) drift job |
| smoke-fail | `[AUTO] Post-merge image smoke test failed` | post-merge image smoke job |
| size-breach | `[AUTO] Post-merge image size guard breached` | post-merge size guard |
| security-rescan | `[AUTO] Post-merge security rescan failed` | post-merge security job |

> These workflows have **no dedup** — every post-merge run opens a *fresh* issue, so they pile up
> (e.g. #1035–#1054 in one day). The cleanup is to keep the signal, drop the noise.

---

## STEP 2 — Compute divergence (no mutations yet)

### A. `[AUTO]` issue triage — *Fix + dedup-close*
For each family:
1. **digest-drift:** run `make check-images`. **If drift is real**, run `make sync-images` and stage
   `images/images.yaml` into the reconcile PR (STEP 4). **If no real drift** (transient / already
   fixed on `main`), the issues are pure noise. Either way: keep the **newest** issue as the anchor,
   plan to close **all** drift issues with a comment referencing the fix PR (real drift) or the
   newest run (no drift).
2. **smoke-fail / size-breach / security-rescan:** these are CI signals, not work items. Keep the
   **newest** of each family open *only if* the failure still reproduces against current `main`
   (re-pull the image / re-read the latest post-merge run); close the older duplicates with a
   "superseded by latest post-merge run #<newest>" reference. If the newest no longer reproduces,
   close the whole family as transient.

### B. Documentation drift — the reconcile core
Compare each status surface to live milestone state from STEP 1. Surfaces and what they must say:

| Surface | What must agree |
|---------|-----------------|
| [README.md](README.md) | Milestone table marker + per-service status table |
| [ROADMAP.md](ROADMAP.md) | Milestone section: active marker, delivered EPICs checked with #refs |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Milestone table + runtime diagram service markers (🟢/🟡) |
| [CLAUDE.md](CLAUDE.md) | Per-milestone table (In scope / Out of scope) |
| [state/current-milestone.md](state/current-milestone.md) | Status Summary + active-milestone progress section + "as of" date |
| [docs/milestones/M6-planning.md](docs/milestones/M6-planning.md) (or active milestone's plan) | EPIC rows ⬜→✅, "Last updated" line |
| [docs/spdd/*/canvas.md](docs/spdd/) | `**Status:**` line — `Aligned`→`Implemented` when the EPIC's last O-step has merged |

Marker vocabulary: `📅 Planned → 🚧 Active → ✅ Complete` (milestones);
`📋 Planned → 🟡 In progress → ✅ Implemented` (services / canvases).

> **Heavy-read tip.** If the surface set is large, delegate the *read-and-diff* to a single
> `Explore` subagent ("report every milestone/service/canvas marker that disagrees with this live
> state: <paste STEP 1 summary>") so the main context stays lean. All edits stay in the main agent.

### C. Knowledge drift — synthesize session learnings (`/m6-learn`)
The orchestrate/issue-generate flows append `## Session Learnings` blocks to
[docs/ai-learnings/*.md](docs/ai-learnings/). A clean snapshot folds those into the expert guides.
Invoke the synthesizer in its default (human-gated) mode:

```
/m6-learn
```

It clusters the accumulated learnings, dedupes against existing
[.claude/commands/experts/*](.claude/commands/experts/), and writes proposed additions to
[docs/ai-learnings/APPLY_LOG.md](docs/ai-learnings/APPLY_LOG.md) as **PENDING** — it **never
auto-commits in synthesis mode**. Read back the new PENDING rows and fold their one-line summaries
into the PLAN (STEP 3) so the human can approve/reject them in the same review.

> **`/m6-learn` owns its own git.** It commits expert-file changes on its own
> `docs/expert-learnings-*` branch and opens a **separate** PR (different files, different concern —
> expert prompts, not status surfaces). Do **not** fold expert-guide edits into the truth-pass doc
> PR. Because it manages its own branch, invoke it as a skill against the real checkout, not inside
> this command's `/tmp` worktree.

### D. Stale / superseded normal issues — *only with `--include-stale`*
Propose (never auto-decide) closing issues that are demonstrably done or obsolete:
- An EPIC whose milestone is ✅ Complete with all child O-steps merged.
- An issue whose work shipped under a different PR/issue (cite it).
- An issue contradicted by current reality (the feature now exists; cite the file/commit).
Each proposal carries an explicit reason and reference. Apply `status: stale` / `duplicate` /
`wontfix` labels per [docs/labels.md](docs/labels.md) where they fit. When in doubt, leave it open —
this is the one place to be conservative.

### E. Consistency check (catches drift the row-flip missed)
```bash
grep -nE 'M[1-8]|🚧|📅|✅|🟡|📋' README.md ROADMAP.md ARCHITECTURE.md CLAUDE.md \
  state/current-milestone.md docs/milestones/*-planning.md | grep -iE 'planned|active|complete'
for c in docs/spdd/*/canvas.md; do grep -H -m1 '^\*\*Status:\*\*' "$c"; done
```

### F. Branch hygiene — local + remote
Stale branches accumulate when a merge didn't auto-delete the head, or when work was abandoned.
A clean snapshot leaves only `main` plus protected/release branches and any branch currently
checked out in a worktree.

```bash
git fetch origin --prune     # sync remote-tracking refs; drop refs for branches deleted on the remote
PROTECTED='^(origin/)?(main|master|HEAD|release/.*)$'
# Branches checked out in ANY worktree must never be deleted (it would break that tree):
git worktree list --porcelain | awk '/^branch /{sub(/refs\/heads\//,"",$2); print $2}'
git branch    --format='%(refname:short)'   # local branches
git branch -r --format='%(refname:short)'   # remote branches (origin/*)
```

> **Squash-merge caveat (this repo merges squash-only — see repo memory).** `git branch --merged
> origin/main` will **not** flag a squash-merged branch: squash makes a new commit, so the branch
> tip is never an ancestor of `main`. The reliable merged-signal is the branch's **PR state**, not
> ancestry — always cross-check `gh pr list --head <branch> --state all`.

Classify every non-protected, non-worktree branch (local **and** remote):
- **Merged → delete.** Its PR is `MERGED` *or* its tip is reachable from `origin/main`
  (`git merge-base --is-ancestor <tip> origin/main`). These are orphans left behind after merge.
- **Unmerged with an OPEN PR → leave.** Live work; note the PR number, do not touch.
- **Orphaned (unique commits, no merged/open PR) → FLAG, never auto-delete.** Report ahead/behind
  vs `main`, last-commit age, and "no PR"; recommend an action (open a PR to land it on main /
  rebase / delete) but leave the decision to the human. Deleting unique unmerged commits is
  irreversible — this is the conservative line.

```bash
# Context for each branch in the report:
gh pr list --head "$b" --state all --json number,state,mergedAt --jq '.[0] // "no PR"'
git log -1 --format='%ci  %an  %s' "origin/$b"
git rev-list --left-right --count "origin/main...origin/$b"   # behind <TAB> ahead
```

---

## STEP 3 — PLAN output (then STOP unless `--execute`)

Print one traceable plan. Stop here and wait for the human's "go" unless `--execute` was passed.

```
## /repo-clean plan — <date>, active milestone <M?>, against main@<short-sha>

### Issues to close (N)
| # | family / kind | action | reason | reference |
|---|---------------|--------|--------|-----------|
| 1052 | digest-drift | close | superseded; drift fixed | PR #<new> |
| 1049 | smoke-fail   | close | transient, not reproducing on main@<sha> | run #<newest> |
...

### Doc surfaces to reconcile (M)
| surface | was | now | driver |
|---------|-----|-----|--------|
| ARCHITECTURE.md | M6 "📅 Planned" | 🚧 Active | gh: M6 111/136 closed |
...

### File changes in the truth-pass PR
- images/images.yaml  (only if real drift)
- README.md, ROADMAP.md, ... (list)

### Proposed expert-guide learnings (separate /m6-learn PR)
| # | domain | proposed addition | sessions |
|---|--------|-------------------|----------|
| .. | ci-release | ... | ... |
(PENDING in APPLY_LOG.md — human marks applied/rejected before /m6-learn --apply)

### Branches to clean (local L / remote R)
| branch | where | state | action | reason |
|--------|-------|-------|--------|--------|
| chore/proto-regen-x | L+R | PR #123 MERGED | delete | orphan left after squash-merge |
| spike/foo | R | no PR · +3/−40 · 88d old | FLAG | unique commits — human decides |
(After cleanup, only `main` + protected branches remain.)

### NOT touched
- AGENTS.md (immutable principles), docs/adr/* (decisions)
```

---

## STEP 4 — EXECUTE (on approval / `--execute`)

1. **Apply `images.yaml` fix** if STEP 2.A found real drift: `make sync-images` (stage the result).
2. **Reconcile every doc surface** from the STEP 2.B table — edit markers, EPIC rows, service
   statuses, the "as of"/"Last updated" dates, and canvas `Status:` lines. Drive each value from the
   live numbers, not the old text.
3. **Commit + PR** from a branch in the worktree:
   ```bash
   BR="chore/repo-clean-$(date +%Y%m%d)"      # docs/ branch name; type matches the dominant change
   git checkout -b "$BR"
   git add -A
   git commit -s -m "docs: reconcile status surfaces to live state (truth-pass)" \
     -m "<short body: what was stale → now, driven by gh state>" \
     -m "Assisted-by: Claude/claude-opus-4-8"
   git push -u origin "$BR"
   gh pr create --title "docs: repo truth-pass — reconcile status surfaces to live state" \
     --body "<ec22a25-style table: Surface | Was | Now; plus the [AUTO]-issue triage summary and the consistency-check result>"
   ```
   - If the diff is purely `images.yaml`/automation, use a `ci:` or `chore:` type instead of `docs:`.
   - The PR body **must** include the "What was stale (and is now fixed)" table and the
     `make check-images` result, mirroring #1014.
4. **Close the planned issues**, each with a one-line reason **and a reference** so the trail is
   traceable:
   ```bash
   gh issue close <N> --comment "Closed by /repo-clean truth-pass: <reason>. See PR #<new> / superseded by #<newest>."
   ```
   Apply `duplicate` / `status: stale` / `wontfix` labels where they were proposed in the plan.
5. **Apply approved learnings.** For every `APPLY_LOG.md` entry the human marked approved
   (`applied` / `pending-commit`), run:
   ```
   /m6-learn --apply
   ```
   This edits the expert guides, marks the log entries `committed`, and opens its **own**
   `docs/expert-learnings-*` PR — kept separate from the truth-pass doc PR. If no entries were
   approved, skip (synthesis-only proposals stay PENDING for the next pass).
6. **Branch hygiene.** Delete only branches the plan marked *merged orphan* — never a FLAGGED
   unmerged branch, a protected branch, or one checked out in a worktree:
   ```bash
   git branch -d  <merged-local>            # -d is safe: refuses unless reachable from HEAD's upstream
   git push origin --delete <merged-remote>  # remote counterpart
   ```
   A **squash-merged** orphan (PR `MERGED` but tip unreachable) makes `-d` refuse — use
   `git branch -D <b>` **only after** confirming `gh pr list --head <b> --state merged` is non-empty.
   Leave every FLAGGED branch untouched; they go in the report for the human to decide.
7. **Squash-merge is the human's call** — leave both PRs open for review unless the user explicitly
   asks to merge, then `gh pr merge <#> --squash` each.

---

## STEP 5 — Verify & report

```bash
# Re-run the consistency grep — must show every surface agreeing on the active milestone marker.
grep -nE 'M[1-8]|🚧|📅|✅|🟡|📋' README.md ROADMAP.md ARCHITECTURE.md CLAUDE.md \
  state/current-milestone.md docs/milestones/*-planning.md | grep -iE 'planned|active|complete'
gh issue list --state open --json title --jq '[.[]|select(.title|startswith("[AUTO]"))]|length'  # expect 0 or the kept anchors
# Branch state — expect only main (+ protected) once cleanup + PR merges settle:
git fetch origin --prune && git branch -r --format='%(refname:short)' | grep -vE '^origin/(main|HEAD)$'
```

Final report to the user:
- Truth-pass PR: `#<n>` — link. Expert-learnings PR (if any): `#<m>` — link.
- Issues closed: count + the `#`s, each with its reason/reference.
- Surfaces reconciled + the consistency-check result (zero remaining disagreements).
- Learnings: PENDING proposals written to `APPLY_LOG.md` (count), and how many were applied.
- Branches deleted: local + remote counts, each with its merged-PR reference; branches **FLAGGED**
  (unmerged, no PR) listed with ahead/behind + age for your decision. Remaining = `main` (+ protected).
- Anything intentionally left open and **why** (e.g. a smoke-fail still reproducing → real bug,
  filed/kept as a work item, not closed).

Then clean up the worktree:
```bash
cd "$REPO" && git worktree remove "$WT" --force 2>/dev/null || true
```

---

## Guardrails

- **Never** close an issue that represents un-shipped work just because it's old. AUTO/CI-noise and
  demonstrably-done work only.
- **Never** edit `AGENTS.md`, ADRs, or `.feature` contract files in a truth-pass.
- **Never** bypass signing/DCO or push to `main` directly.
- **Never** delete a branch that is protected (`main`/release), checked out in a worktree, or has
  unique unmerged commits without a `MERGED` PR. Detect merges by **PR state**, not `git branch
  --merged` (squash-merge hides merges from ancestry). FLAG unmerged orphans; never auto-delete them.
- If `make check-images` / `make sync-images` can't run (no Docker), say so and fall back to
  dedup-close only for digest-drift — do not hand-edit digests.
- One truth-pass = one doc PR + (optionally) one `/m6-learn` PR. These two are separate **by
  design** — status surfaces vs. expert prompts. Don't fold either into the other, and don't fold
  unrelated code fixes into either.
