---
description: Resume work on the active milestone (state/milestone.yaml) — one canvas per EPIC, /spdd-story creates all story issues in GitHub, /spdd-generate implements one O-step at a time, stop after the cluster is merged.
argument-hint: "[optional: epic issue number or story issue number to prefer, e.g. 765 766]"
---

# Resume Milestone — active milestone from state/milestone.yaml

Pick the next ready EPIC, decompose it into story issues via `/spdd-story`, ship each story as its
own PR in O-step order, merge them in order, leave every state file consistent, then stop.

> **Parallel-session safety.** Multiple sessions may run concurrently. Two mechanisms prevent
> duplicate work: (1) a pre-filter in STEP 2 and STEP 3C that skips EPICs/stories whose branch or
> open PR already exists on the remote; (2) an atomic claim in STEP 4 that pushes the empty branch
> to GitHub immediately — only one `git push -u origin $BRANCH` wins when two sessions race.
> If your push is rejected (branch already exists), treat that story as claimed and return to STEP 3C
> to pick the next available one. Never assume a story is free just because you read it as open.

**EPIC-canvas model:** every `feat:` EPIC has exactly **one** REASONS Canvas at
`docs/spdd/<epic-issue>-<slug>/canvas.md`. That canvas's O steps map 1-to-1 to story PRs. Story
issues are created in GitHub by `/spdd-story` — they reference the parent EPIC and carry full
labels, milestone, and a test plan template. `/spdd-generate` always operates on the EPIC canvas,
not a per-story canvas.

> **Rules are not restated here.** Commit/PR format, conventional types, DCO + `Assisted-by`
> trailers, anti-patterns, `GOWORK=off`, PR-size, hexagonal layout, coverage gates, and the SPDD
> requirement all live in **`AGENTS.md`** (constitution) and **`CLAUDE.md`** (dev loop). Read them;
> obey them. This file is only the *session loop*.

**Session policy.** One **EPIC** per session. Each O-step ships as its **own** PR (= one story
issue). Open all story-PRs in the cluster, enable auto-merge on the first PR, and **stop** — do not
block waiting for CI. The STEP 1.5 merge pass at the start of the *next* session merges green PRs in
O-step order and enables auto-merge on the next one. `Closes #<story-N>` in the PR body closes the
story issue automatically on squash-merge. Every PR flips its own story-issue row in
`"$PLANNING_DOC"` and updates `state/current-milestone.md` in its own diff.

---

## Branch discipline (non-negotiable — ADR-023)

- **Rebase before every merge.** `git rebase origin/main` immediately before `gh pr merge`.
  Never merge a branch that has diverged from `main`. Resolve all conflicts, then
  `git push --force-with-lease` before merging.
- **Merge strategy: `--squash` only.** Use `gh pr merge <PR> --squash`. Never `--merge`
  (creates a merge commit, violates `required_linear_history`). `--rebase` is blocked by
  `required_signatures` — GitHub cannot auto-sign replayed commits.
- **Delete the remote branch after every merge:**
  ```bash
  git push origin --delete <branch>
  ```
  No merged or closed branch should remain on the remote.
- **Never reopen a closed PR or branch.** If commits from a closed branch are still
  wanted, cherry-pick or rebase them onto a fresh branch off current `main`, open a
  new PR, let CI run green, then squash-merge.
- **No direct commits to `main` — including one-line doc fixes.** All changes go
  through a branch → PR → CI green → `gh pr merge --squash` → branch deleted.

---

## STEP 1 — Orient & resume (run first)

**1.1 Sync main + load milestone config**
```bash
git fetch origin --prune && git checkout main && git pull --rebase origin main
[ "$(git rev-parse main)" = "$(git rev-parse origin/main)" ] || { echo "main diverged — STOP"; exit 1; }
[ -z "$(git status --porcelain)" ] || { echo "dirty tree — STOP"; git status; exit 1; }

# ── Active-milestone config (SSoT: state/milestone.yaml) ────────────────────
# Loaded at runtime; no milestone name, number, or label is hardcoded in this
# file. Updated only by /milestone-close and /milestone-new.
CFG=state/milestone.yaml
MILESTONE_NAME=$(awk '/^active:/{f=1} f && /^  name:/{print $2; exit}' "$CFG")
MILESTONE_TITLE=$(awk -F'"' '/^active:/{f=1} f && /^  title:/{print $2; exit}' "$CFG")
MILESTONE_NUMBER=$(awk '/^active:/{f=1} f && /^  github_milestone_number:/{print $2; exit}' "$CFG")
MILESTONE_VERSION=$(awk '/^active:/{f=1} f && /^  version:/{print $2; exit}' "$CFG")
PLANNING_DOC=$(awk '/^active:/{f=1} f && /^  planning_doc:/{print $2; exit}' "$CFG")
MILESTONE_LABEL=$(awk -F'"' '/^    milestone:/{print $2; exit}' "$CFG")
GH_MILESTONE="${MILESTONE_TITLE} (${MILESTONE_NAME})"   # GitHub milestone title
# ─────────────────────────────────────────────────────────────────────────────
```

**1.2 Read** — mandatory every session:
- `state/current-milestone.md`
- `"$PLANNING_DOC"` (epic/story tables + risk table)
- `CLAUDE.md`

| Read when | File |
|---|---|
| Proto or BDD boundary touched | `protos/AGENTS.md` + `docs/patterns/bdd-contract-testing.md` |
| Helm chart work | `infra/AGENTS.md` + `docs/patterns/helm-charts.md` |
| Any ADR-governed decision | `docs/adr/INDEX.md` — check the ADR that governs the area before changing it |
| Image / GHCR / release work | `images/images.yaml` (SoT — ADR-024) + ADR-025 (attestations) + ADR-027 (retag model) |
| EPIC-specific caveats | The EPIC's row + notes in `"$PLANNING_DOC"` — milestone-specific guidance lives there, never here |

Run `/help` to confirm all SPDD commands are available.

**1.3 Reconcile GitHub milestone ⟷ planning doc**
```bash
gh issue list --milestone "$GH_MILESTONE" --state open   --limit 200 --json number,title,labels,state
gh issue list --milestone "$GH_MILESTONE" --state closed --limit 200 --json number,title,labels,state
```
Open issue ⇒ ⬜ row; closed issue ⇒ ✅ row. Any mismatch → fix via a dedicated `docs:` branch
and PR (never a direct commit to `main`):
```bash
git checkout -b docs/milestone-sync-$(date +%Y%m%d)
# edit "$PLANNING_DOC" and/or current-milestone.md
git add "$PLANNING_DOC" state/current-milestone.md
git commit -s -m "docs(milestones): reconcile ${MILESTONE_NAME} planning table — <what changed>

Assisted-by: Claude/<model>"
git push -u origin HEAD
gh pr create --title "docs(milestones): reconcile ${MILESTONE_NAME} planning table" \
  --body "Reconcile open/closed issue state with delivery table." \
  --label "type: docs" --label "$MILESTONE_LABEL"
gh pr checks <PR> --watch
git fetch origin main && git rebase origin/main && git push --force-with-lease
gh pr merge <PR> --squash
git push origin --delete docs/milestone-sync-$(date +%Y%m%d)
```

**1.4 Detect in-flight + health gate**
```bash
gh pr list --author "@me" --state open --json number,title,headRefName,statusCheckRollup,mergeStateStatus,autoMergeRequest
git branch -a | grep -E "^\* (feat|fix|refactor|docs|test|ci|chore)/"; git status --porcelain
```
Apply the **Resumption tree** (bottom) and report the matching row before taking any action.

**Health gate rule:** only block the session if a required check is **RED** (failed/error). Checks
that are pending or running are not a blocker — proceed to STEP 1.5 and then pick new work. Never
`--watch` CI here.

**1.5 — Quick merge pass (non-blocking, run every session)**

Before picking new work, sweep all your open PRs and merge any that are already green. This clears
the queue in ≤30 s without blocking on CI.

```bash
# Get all open PRs sorted by number (= O-step order)
OPEN_PRS=$(gh pr list --author "@me" --state open \
  --json number,headRefName,statusCheckRollup,mergeStateStatus \
  --jq 'sort_by(.number) | .[] |
    [.number, .headRefName, .mergeStateStatus,
     ([ .statusCheckRollup[]? | select(.isRequired==true) | .conclusion ] | unique | tostring)
    ] | @tsv')

# Merge only PRs where mergeStateStatus=="CLEAN" and no required check failed
while IFS=$'\t' read -r PR_N BR MERGE_STATE REQ_CONCLUSIONS; do
  if [[ "$MERGE_STATE" == "CLEAN" ]] && ! echo "$REQ_CONCLUSIONS" | grep -qE 'FAILURE|ERROR|TIMED_OUT'; then
    echo "Merging PR #$PR_N ($BR) — all required checks green"
    git fetch origin --prune
    git checkout "$BR" && git rebase origin/main && git push --force-with-lease
    gh pr merge "$PR_N" --squash
    until [ "$(gh pr view "$PR_N" --json state --jq .state)" = "MERGED" ]; do sleep 10; done
    git push origin --delete "$BR" 2>/dev/null || true
    git checkout main && git pull --rebase origin main
  else
    echo "Skipping PR #$PR_N ($BR) — not yet green (state=$MERGE_STATE)"
  fi
done <<< "$OPEN_PRS"
```

After the pass: if all your PRs merged and the EPIC is complete → STEP 10 + stop.
Otherwise pick new work below.

---

## STEP 2 — Pick an EPIC

**Priority order** lives in the EPIC table of `"$PLANNING_DOC"` — top-to-bottom table order is
the priority. Never hardcode EPIC or issue numbers in this command file; the planning doc and
the live issue bodies are the source of truth.

```bash
# Live EPIC list (cross-check against the planning-doc table; this is the
# runtime truth when the static open_epics hint in state/milestone.yaml is stale)
gh issue list --milestone "$GH_MILESTONE" --label "type: epic" --state open \
  --limit 50 --json number,title,labels
```

For each candidate EPIC, the status gate is determined live:
- canvas at `docs/spdd/<EPIC_N>-*/canvas.md` with `Status: Aligned` → implementable
- canvas `Status: Draft` or missing (and EPIC is `feat:`) → STEP 3A first
- `refactor:/ci:/chore:` EPIC → SPDD-exempt → STEP 3D
- body names an open blocker ("Depends on #N" / "Pending #N" with #N open) → BLOCKED, skip

**Pre-filter: skip EPICs already claimed by another session**

```bash
# Collect all open remote branches and open PR head-refs (includes drafts)
git fetch origin --prune
CLAIMED_BRANCHES=$(git ls-remote origin 'refs/heads/*' | awk '{print $2}' | sed 's|refs/heads/||')
CLAIMED_PRS=$(gh pr list --state open --json headRefName --jq '.[].headRefName')
CLAIMED=$(printf '%s\n%s\n' "$CLAIMED_BRANCHES" "$CLAIMED_PRS" | sort -u)

# For each EPIC candidate (in priority order), check if any story branch is already on remote.
# Branch names follow: <type>/<story-issue-N>-<slug>
# Pattern: any branch whose name contains the story issue numbers for this EPIC
# Example — EPIC #765 has stories #779–#792; skip #765 if any of those branches exist:
echo "$CLAIMED" | grep -E "^(feat|fix|refactor|docs|ci|chore)/(779|780|781|782|783|784|785|786|787|788|789|790|791|792)-"
# If any match → EPIC #765 is in-flight in another session → move to next priority
```

After confirming the EPIC is unclaimed:

```bash
gh issue view <EPIC_N> --json number,title,body,labels,state,milestone,comments
ls docs/spdd/ | grep -E "^<EPIC_N>-"   # check if EPIC canvas already exists
```

**Determine which canvas state the EPIC is in:**
- Canvas exists + `Status: Aligned` → go to STEP 3B (story check) or STEP 3C (implement)
- Canvas exists + `Status: Draft` → go to STEP 3A to align
- No canvas, EPIC is `feat:` → go to STEP 3A (full SPDD pipeline)
- EPIC is `refactor:/ci:/chore:` → go to STEP 3D (SPDD-exempt)
- EPIC is BLOCKED → report blocker, pick a different EPIC

---

## STEP 3 — EPIC canvas + story decomposition

### 3A — Canvas not yet Aligned (full SPDD pipeline)

Run in order, stopping between each step for inspection:

```bash
/spdd-analysis <EPIC_N>
```
Review output: verify ADR constraints, tier classification, Tier 2 flags.

```bash
/spdd-reasons-canvas <EPIC_N>
```
Canvas is written to `docs/spdd/<EPIC_N>-<slug>/canvas.md` (Status: Draft).

**Canvas must include** for infra EPICs:
- **R**: exact K8s DoD (observable outcomes, not just "charts exist")
- **E**: every new resource type, every gRPC contract touched
- **A**: what we WILL do and what we WON'T (e.g. "no OTel traces — that is a later milestone")
- **S-Structure**: every file created or modified, K8s resource kind for infra EPICs
- **O**: one O-step per story PR, each ≤400 lines — number them to match story issue titles
- **N**: GOWORK=off, DCO+Assisted-by, test coverage gate, liveness threshold env var etc.
- **S-Safeguards**: architecture invariants, state-minimization rule (where applicable)

```bash
/spdd-security-review docs/spdd/<EPIC_N>-<slug>/canvas.md
```
Must PASS before committing. Any Tier 2 findings → move to `canvas.private.md`.

**[Human reviews and sets Status: Aligned — then run STEP 3B]**

### 3B — Create story issues in GitHub (run once per EPIC)

Check whether story issues already exist before creating:
```bash
gh issue list --milestone "$GH_MILESTONE" --state all \
  --json number,title,body --jq '.[] | select(.body | test("#<EPIC_N>"))' | head -40
```

If stories are missing, run:
```bash
/spdd-story <EPIC_N>
```

`/spdd-story` will create one GitHub issue per O-step. **Each story issue MUST have:**
- Title: `feat(<scope>): <story-title> (#<EPIC_N>, step <N>)` — conventional-commit form
- Labels: `type: feature`, `area: <area>`, `$MILESTONE_LABEL`, `size/<S|M|L>`, `spdd: canvas-step`
- Milestone: `$GH_MILESTONE`
- Body includes:
  - Story (as-a / I-want / so-that)
  - Canvas reference: `docs/spdd/<EPIC_N>-<slug>/canvas.md` O-step N
  - Scope (what this PR changes)
  - Acceptance criteria (≥3, each concrete and testable)
  - Out of scope (what the next O-step handles)
  - Size estimate
  - Dependencies (previous O-step issue, if any)
  - Test plan checklist (see template below)
  - `Assisted-by: Claude/<model-id-of-this-session>`

Verify all stories were created:
```bash
gh issue list --label "$MILESTONE_LABEL" --state open --json number,title \
  --jq '.[] | select(.title | test("#<EPIC_N>"))'
```

### 3C — Pick the next unmerged O-step

```bash
# Find which O-steps are already merged (look for closed story issues)
gh issue list --label "$MILESTONE_LABEL" --state closed --json number,title,body \
  --jq '.[] | select(.body | test("#<EPIC_N>.*step")) | {n:.number,title}'

# Collect all open/draft PR head-refs and remote branches (parallel-session filter)
git fetch origin --prune
CLAIMED=$(printf '%s\n%s\n' \
  "$(git ls-remote origin 'refs/heads/*' | awk '{print $2}' | sed 's|refs/heads/||')" \
  "$(gh pr list --state open --json headRefName --jq '.[].headRefName')" \
  | sort -u)

# List open story issues for this EPIC (lowest step number first), skip already-claimed ones
gh issue list --label "$MILESTONE_LABEL" --state open --json number,title,body \
  --jq '.[] | select(.body | test("#<EPIC_N>.*step")) | {n:.number,title}' \
  | head -10
# For each candidate story #N: check if any branch matching <type>/<N>-* is in $CLAIMED
# echo "$CLAIMED" | grep -E "^[a-z]+/<N>-"
# If a match is found → another session owns that story → skip to the next step number
```

**One O-step at a time.** The cluster for this session = one EPIC's next ≤3 O-steps (if they
are independent) or the next single O-step (if each depends on the previous).

### 3D — SPDD-exempt EPICs (refactor:/ci:/chore:)

No canvas required. Implement directly from the issue body. Each story issue still needs the
test plan template and labels — create them via:
```bash
gh issue create --title "<type>(<scope>): <title> (#<EPIC_N>, step <N>)" \
  --label "type: <refactor|ci|chore>,area: <area>" --label "$MILESTONE_LABEL" --label "size/<S|M>" \
  --milestone "$GH_MILESTONE" \
  --body "$(cat <<'EOF'
## Context
Closes the <N>th step of <EPIC_N> (<epic title>).

## Scope
<what this PR changes — exact files>

## Acceptance criteria
- [ ] <criterion>
- [ ] <criterion>

## Test plan
- [ ] `GOWORK=off go test ./... -race` — pass
- [ ] `make lint` — exit 0
- [ ] <additional check relevant to this story>

## Out of scope
<what comes in the next step>

## Size estimate: <XS/S/M>

Assisted-by: Claude/<model-id-of-this-session>
EOF
)"
```

---

## STEP 4 — Scope + branch (per O-step story)

```bash
git fetch origin --prune && git checkout main && git pull --rebase origin main
[ "$(git rev-parse main)" = "$(git rev-parse origin/main)" ] || { echo "main diverged"; exit 1; }
BRANCH=<type>/<story-issue-N>-<short-slug>
git checkout -b "$BRANCH"

# ── Atomic claim ─────────────────────────────────────────────────────────────
# Push the empty branch to GitHub NOW (before any code). GitHub serialises branch
# creation: only one push wins when two sessions race on the same branch name.
# A rejected push means another session already claimed this story → go back to
# STEP 3C and pick the next available O-step.
if ! git push -u origin "$BRANCH" 2>&1; then
  echo "CLAIMED: branch $BRANCH already on remote — story #<story-issue-N> taken by another session"
  git checkout main && git branch -D "$BRANCH"
  echo "→ return to STEP 3C and pick the next open O-step"
  exit 1
fi
echo "CLAIMED: story #<story-issue-N> → $BRANCH pushed to origin"
# ─────────────────────────────────────────────────────────────────────────────
```

**Stacking:** if O-step 2 depends on O-step 1's types/files, stack B off A (`git checkout -b ... A`).
If truly independent (different files, no shared types), branch each off `main`.

**Size gate:** XS<50 / S 50–200 / M 200–400 → proceed · L 401–900 → justify in PR body ·
**XL>900 → STOP**, split the O-step further, open follow-up issue.

**BDD gate (ADR-016):** if this O-step touches a gRPC boundary:
```bash
ls protos/tests/<service>/features/   # .feature file must exist before implementation
```
Missing `.feature` → write and commit it FIRST, in a separate commit on this branch.

---

## STEP 5 — Implement via /spdd-generate

For `feat:` O-steps with an Aligned epic canvas:
```bash
/spdd-generate docs/spdd/<EPIC_N>-<slug>/canvas.md
```
The skill reads the canvas, identifies the current O-step, generates the code for that step only,
and stops. Review the output; if it would violate a safeguard, halt and report.

After generating, run the evidence commands:

| Check | Command | Required |
|---|---|---|
| build | `GOWORK=off go build ./...` in touched service dirs | always |
| unit + race | `GOWORK=off go test ./... -race -timeout 60s` | always |
| domain coverage ≥90% | `GOWORK=off go test ./internal/domain/... -coverprofile=cov.out && go tool cover -func cov.out` | domain changes |
| lint | `make lint-go` (in Docker) | always |
| BDD | `make test-bdd` | contract changes |
| security | `make security` | always |

Capture all output — paste into PR body test plan checkboxes.

---

## STEP 6 — State consistency (per story PR, in its own diff)

Each story PR updates:
1. `"$PLANNING_DOC"` — flip this story's row ⬜→✅; bump "Last updated"
2. `state/current-milestone.md` — update EPIC progress; note if EPIC is now fully done
3. Epic canvas O-step — mark it ✅; run `/spdd-sync <canvas>` if implementation diverged
4. `services/<svc>/AGENTS.md` — only if a new gRPC method, K8s resource type, or env var was added

---

## STEP 7 — Commit

```bash
echo -n "<type>(<scope>): <subject>" | wc -c   # ≤ 72 characters
git commit -s -m "<type>(<scope>): <subject>

<why — canvas O-step N of EPIC #EPIC_N; one sentence>

Closes #<story-issue-N>

Assisted-by: Claude/<model-id-from-this-session>"
```

---

## STEP 8 — Open all story PRs in parallel

```bash
# Branch already exists on remote from the STEP 4 claim — force-push the commits
git push --force-with-lease
echo -n "<type>(<scope>): <subject>" | wc -c   # ≤ 72
gh pr create --base <main|branch-below> \
  --title "<type>(<scope>): <subject>" \
  --assignee "@me" \
  --label "type: <kind>" --label "$MILESTONE_LABEL" --label "area: <area>" \
  --body-file pr-body-<N>.md
```

**Required in every PR body** (`pr-body-<N>.md`): include `Closes #<story-issue-N>` — this closes
the story issue automatically on squash-merge (do not rely solely on the commit message). Fill all
test plan checkboxes with evidence before opening.

Open **all** cluster story PRs before STEP 9. Verify no other open PR of yours has **red** checks
(running/pending checks are fine — the STEP 1.5 merge pass handles them next session).

---

## STEP 9 — Enable auto-merge + stop (do NOT block on CI)

Once all story PRs are open, rebase each branch off `origin/main`, enable auto-merge on the
**first** O-step PR, then **stop the session**. CI runs asynchronously. The STEP 1.5 merge pass
in the next session detects green PRs, merges them in O-step order, and enables auto-merge on the
next PR in sequence.

```bash
# Rebase every branch off current origin/main before enabling auto-merge
for BR in <branch_1> <branch_2> ...; do
  git fetch origin --prune
  git checkout "$BR"
  git rebase origin/main || { echo "CONFLICT on $BR — resolve before stopping"; exit 1; }
  git push --force-with-lease
done
git checkout main

# Enable auto-merge on O-step 1 only; subsequent PRs get auto-merge enabled by the merge pass
# after the preceding PR merges (to respect O-step order).
gh pr merge <pr_1> --auto --squash
echo "Auto-merge enabled on PR #<pr_1>. Session complete — CI is running."
```

**Why stop here?** `gh pr checks --watch` freezes the session slot (typically 5–15 min) without
doing useful work. In a parallel setup this means no new issues get picked up. The merge pass is
the right place to detect and act on CI results.

**If `--auto` is unavailable** (repo has it disabled): leave branches rebased and pushed; the
STEP 1.5 pass will check `mergeStateStatus` and merge when green.

**Post-merge cleanup happens automatically:**
- Story issue closes via `Closes #<N>` in PR body (squash-merge carries it)
- Planning-doc row ⬜→✅ is in the PR diff — merged with the code
- After the merge pass, verify: `grep -nE "#?$STORY\b" "$PLANNING_DOC"` — row must be ✅

---

## STEP 10 — EPIC completion check + stop

When ALL O-steps of an EPIC are merged:
```bash
# Check all story issues for this EPIC are closed
gh issue list --label "$MILESTONE_LABEL" --state open --json number,title,body \
  --jq '.[] | select(.body | test("#<EPIC_N>.*step"))'  # should be empty

# Update epic issue itself
gh issue close <EPIC_N> --reason completed --comment "All O-steps merged. Canvas status: Implemented."
```

Mark the EPIC canvas `Status: Implemented` (small docs: commit). Post the session summary (below).
**One EPIC per session — end the turn.**

---

## Resumption tree

| Observed | Action |
|---|---|
| Open PR of mine with **red** required checks | Fix first; do not advance or start new cluster |
| Open PR of mine with checks **running/pending** | STEP 1.5 merge pass (skip non-green PRs); then pick next unclaimed O-step → STEP 4 |
| Open PR of mine, all checks **green** (CLEAN) | STEP 1.5 merge pass → merge now; continue to remaining O-steps or STEP 10 |
| Some story PRs merged, others open for same EPIC | STEP 1.5 merge pass; if unmerged PRs still running, pick next unclaimed O-step |
| All story PRs open, none merged | STEP 1.5 merge pass; if none green yet, pick next unclaimed O-step if EPIC has more stories |
| EPIC fully merged | STEP 10 summary. STOP. |
| Local branch w/ uncommitted work, no PR | Finish STEP 5→8 |
| STEP 4 branch push rejected (branch exists on remote) | Another session claimed that story — return to STEP 3C, pick next open O-step |
| No in-flight work, EPIC has stories created | Pick next O-step → STEP 4 |
| EPIC has canvas Aligned but no story issues | STEP 3B: create stories, then STEP 4 |
| EPIC has no canvas | STEP 3A: full pipeline |
| EPIC is BLOCKED (#764 for EPIC I, #626 for EPIC J, #626+#772 for DevAuto.8 #881) | Pick next unblocked EPIC |
| All EPICs exhausted | Report milestone exit-criteria; recommend /milestone-close |

---

## Story issue test plan template

> Every box `- [x]` with evidence before auto-merge. N/A → `- [x] (N/A — reason)`.

```markdown
## Summary
<1-3 sentences — what changes and why>

## EPIC canvas
`docs/spdd/<EPIC_N>-<slug>/canvas.md` — O-step <N>

## Acceptance criteria
- [ ] <concrete observable outcome 1>
- [ ] <concrete observable outcome 2>
- [ ] <concrete observable outcome 3>

## Out of scope
<what the next O-step covers>

## Test plan

### Build & unit
- [ ] `GOWORK=off go build ./...` — exit 0  [evidence]
- [ ] `GOWORK=off go test ./... -race -timeout 60s` — all pass  [evidence]
- [ ] `GOWORK=off go test ./internal/domain/... -cover` — ≥90% on domain  [evidence / N/A]

### Lint & security
- [ ] `make lint-go` — exit 0  [evidence]
- [ ] `make security` — no new findings  [evidence]

### Contract (when gRPC boundary touched)
- [ ] `.feature` file committed before implementation  [evidence / N/A]
- [ ] `make test-bdd` — all scenarios pass  [evidence / N/A]

### Acceptance
- [ ] <criterion mapped to acceptance criteria above>  [evidence: test / file:line / log]

### Engineering hygiene
- [ ] **Planning-doc row ⬜→✅ in this diff** (mandatory)
- [ ] **`current-milestone.md` updated in this diff** (mandatory)
- [ ] Canvas O-step <N> marked ✅; `/spdd-sync` run if impl diverged
- [ ] Branched from fresh `origin/main` · PR ≤900 lines · trailers on every commit
- [ ] No out-of-scope edits · observability (traces/dashboards) deferred to a later milestone
```

---

## Session summary

```markdown
## Session summary — <YYYY-MM-DD HH:MM TZ>
**Outcome:** <EPIC-COMPLETE | EPIC-PARTIAL (n/m O-steps) | STORIES-CREATED-NOT-IMPLEMENTED | STOPPED-BLOCKED | STOPPED-HEALTH-GATE>
**EPIC:** #<N> <title> · canvas `docs/spdd/<N>-<slug>/canvas.md` · O-steps this session: <X–Y of Z>

| Story | PR | type | size | O-step | state |
|---|---|---|---|---|---|
| #A | <url> | <feat/…> | <S/M> | O-step N | MERGED |

**State files:** planning-doc rows ⬜→✅ <list> · current-milestone <change> · canvas O-steps ✅ <list> · AGENTS.md <svc/N/A>
**Verify (all ✓ to continue):** story issues CLOSED ✓ · PRs MERGED ✓ · planning-doc lockstep ✓ · no stray PRs/branches, tree clean ✓
**Blockers:** <list any blocking issues — e.g. "#764 EBUS-DECISION unresolved — EPIC I stays blocked">
**Repo state:** main HEAD <sha> "<subject>" · my open PRs <list/none> · health <green | red on #PR>
**Next:** EPIC <#N title> · O-step <N> (#story-issue) · review first: <canvas / issue / N/A>
```
