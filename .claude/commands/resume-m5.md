---
description: Resume Zynax M5 work — pick a cluster of ≤3 related issues, ship each as its own PR, merge in order, leave state consistent, stop.
argument-hint: "[optional: issue numbers to prefer, e.g. 530 531]"
---

# Resume M5 — Adapter Library (v0.4.0)

Pick the next ready cluster, ship each issue as its own PR, merge them in order, leave every
state file consistent, then stop. **Stateless & compaction-safe:** derive all position from
`gh`/`git`/state files, never from session memory. Safe to re-run after `/compact`, crash, or a
fresh session.

> **Rules are not restated here.** Commit/PR format, conventional types, DCO + `Assisted-by`
> trailers, anti-patterns, `GOWORK=off`, PR-size, hexagonal layout, coverage gates, and the SPDD
> requirement all live in **`AGENTS.md`** (constitution) and **`CLAUDE.md`** (dev loop). Read them;
> obey them. This file is only the *session loop*.

**Session policy.** One **cluster of ≤3 related issues**. Each ships as its **own** PR. Open all
PRs in parallel, then merge them **strictly in order** — auto-merge enabled on PR_i only after
PR_(i-1) is MERGED and PR_i is rebased onto fresh `origin/main`. Never enable auto-merge on >1 PR
at once. Never merge by hand (`--admin` forbidden) — auto-merge + the Merge Queue run the required
checks. The **pre-merge required checks are the gate; do NOT chase post-merge `main` CI** (publish/
release run on `main` and aren't your concern). Every PR flips its own `M5-plan.md` row and updates
`current-milestone.md` **in its own diff**.

---

## STEP 1 — Orient & resume (run first)

**1.1 Sync main**
```bash
git fetch origin --prune && git checkout main && git pull --rebase origin main
[ "$(git rev-parse main)" = "$(git rev-parse origin/main)" ] || { echo "main diverged — STOP, ask"; exit 1; }
[ -z "$(git status --porcelain)" ] || { echo "dirty tree — STOP, ask"; git status; exit 1; }
```

**1.2 Read** — mandatory: `state/current-milestone.md`, `docs/milestones/M5-plan.md` (scoped: Track
Overview + current batch + gap table), `CLAUDE.md`. On demand by trigger:

| Read when | File |
|---|---|
| gap status / quality lens (G1–G24) | `docs/reviews/04-architecture-gaps.md` |
| forming a cluster from findings / re-planning | `docs/reviews/05-action-plan.md` |
| verifying M5 Definition of Done | `docs/reviews/03-m5-state.md` |
| any `docs:`/truth-pass (G2) | `docs/reviews/02-reality-vs-docs.md` |
| touching an ADR'd decision | `docs/reviews/01-decision-ledger.md` |
| feat / issue lacks acceptance criteria | the canvas covering it (see STEP 2) |

Run `/help` to confirm SPDD commands are available.

**1.3 Reconcile milestone ⟷ plan** (must agree)
```bash
gh issue list --milestone "Adapter Library (M5)" --state open   --limit 100 --json number,title,state
gh issue list --milestone "Adapter Library (M5)" --state closed --limit 200 --json number,title,state
```
Open issue ⇒ ⬜ row; closed issue ⇒ ✅ row. Any mismatch → fix `M5-plan.md` **and**
`current-milestone.md` now (small `docs:` commit, no PR; bump "Last updated").

**1.4 Detect in-flight + health gate**
```bash
gh pr list --author "@me" --state open --json number,title,headRefName,statusCheckRollup,mergeStateStatus,autoMergeRequest
git branch -a | grep -E "^\* (feat|fix|refactor|docs|test|ci|chore)/" ; git status --porcelain
```
**Health gate:** an open PR with red required checks won't auto-merge — fix it before advancing or
starting a new cluster. Then apply the **Resumption tree** (bottom) and report the matching row to
the user in one paragraph; wait for `go`/`stop`.

---

## STEP 2 — Pick a cluster (1–3 related issues)

"Related" = same area/service/layer or same task kind; may be a dependency chain (A→B→C) or
independent siblings. Rules: draw from the **same BATCH** in order **0→1→2→3→4→5→6**; prefer
lower-numbered issues; exclude any issue whose dependency is ⬜ open **and outside this cluster**
(intra-cluster deps are fine — handled by merge order); skip XL (>900 — split first); never exceed
3; mixed types OK if genuinely related.

```bash
gh issue view <N> --json number,title,body,labels,state,milestone,assignees,comments  # never bare view (deprecated projectCards → exit 1)
grep -rln -E "(#|issue[: ]*)<N>\b" docs/spdd/*/canvas.md   # canvas may be a PARENT-EPIC, not <N>-*
ls docs/spdd/ | grep -E "^<N>-"
```
If a canvas covers an issue (by name or O-step reference), that canvas is the scoping contract —
record which canvas + O-step each member maps to (or "no canvas"). **State the cluster + merge
order + shared context before coding.**

Batch map: `0` CI-unblock · `1` release · `2` engine-correctness · `3` SDK+truth · `4` dispatch-E2E
(ordered: task-broker→agent-registry→compose #481) · `5` CI-DX · `6` adapters (after #481). Live
⬜/✅ + issue #s are in `M5-plan.md` — never hardcode from memory.

---

## STEP 3 — Entry path (per issue)

Scope is fixed before any code: by an **Aligned** canvas for `feat`, by the issue body for
everything else. Implement exactly that one logical change — no cleanup, no extras, no drift.

| Type | Path |
|---|---|
| `feat` | Canvas MUST be `**Status:** Aligned` before code → `/spdd-generate <canvas>` (scopes to one O-step, then stops). Draft → align first (`/spdd-reasons-canvas`). No canvas referencing #N → not ready as feat. |
| `fix` `test` `ci` `chore` `refactor` | Implement directly (SPDD-exempt). |
| `docs` | Edit; truth-pass (G2); no mTLS/SBOM/cosign claims (M6+). |

Non-`feat`: still read a covering canvas O-step if one exists, and commit the `.feature` file
**before** implementation when a public boundary is touched.

---

## STEP 4 — Scope + fresh main + branch (per issue)

```bash
git fetch origin --prune && git checkout main && git pull --rebase origin main
[ "$(git rev-parse main)" = "$(git rev-parse origin/main)" ] || { echo "main diverged"; exit 1; }
git checkout -b <type>/<issue-N>-<short-slug>
```
**Branching:** independent siblings each branch off `main` (base `main`). Dependency chain A→B→C:
**stack** branches (B off A, C off B); each PR base = branch below; bottom PR base = `main`
(GitHub auto-retargets on merge).

**Size:** XS<50 / S 50–150 / M 150–400 → proceed · L 401–900 → justify + flag user · **XL>900 →
STOP**, scope to first 400 lines, open follow-up. (PR-size thresholds: `AGENTS.md`.)

**Gap lens** on every file touched (full list: `docs/reviews/04-architecture-gaps.md`): **G1**
const-time bearer compare · **G4** Temporal Activity `RetryPolicy` · **G7** `mergePayload` scalar
type-asserts · **G16** derived ctx + request-ID propagation. <10-line fix → note `Opportunistic
fix: G<N>` in PR body; else open a follow-up.

---

## STEP 5 — Implement

If a canvas covers the issue, implement via `/spdd-generate <canvas>` (read the O-step first).
Otherwise hand-implement the single logical change. `GOWORK=off` for every `go` command in
`services/*/`, `cmd/zynax/`, `protos/tests/` (ADR-017). Run the relevant checks and capture
evidence for the test plan:

| Check | Command | When |
|---|---|---|
| lint | `make lint-fix` / `make lint-go` / `make lint-protos` / `make lint-agents` | per change type |
| unit | `GOWORK=off go test ./... -race -timeout 60s` | always |
| domain cov ≥90% | `make test-coverage` | `internal/domain/` |
| adapter cov ≥80% | `make test-coverage-adapters` | adapter changes |
| BDD | `make test-bdd` | `.feature`/contract |
| security | `make security` | always |
| AI-context | `zynax-ci check ai-context` | AI-context files |

Tool missing locally → its box stays unchecked → do **not** enable auto-merge; open for human
review and surface it.

---

## STEP 6 — State consistency (per PR, in its own diff)

Each PR updates **only its own issue's row** (avoids sibling/stacked conflicts):
1. `docs/milestones/M5-plan.md` — flip this issue's row ⬜→✅; bump "Last updated"; update Track
   Overview if a track is now fully ✅. **Mandatory — not N/A.**
2. `state/current-milestone.md` — update the track row; remove the done issue from Active/IMMEDIATE;
   move any newly-unblocked downstream issue to "ready". **Mandatory.**
3. Canvas (feat only) — mark the O-step ✅; `/spdd-sync` if impl diverged.
4. `services/<svc>/AGENTS.md` — only if a new endpoint/type/capability was added (ADR-016).

Bundle into the implementation commit (or a follow-up commit on the **same branch**).

---

## STEP 7 — Commit

```bash
echo -n "<type>(<scope>): <subject>" | wc -c   # ≤ 72
git commit -s -m "<type>(<scope>): <subject>

<why; issue + Canvas O-step if feat>

Closes #<N>
Assisted-by: Claude/<model-id-from-this-session>"
```
Exact trailer spec (`Signed-off-by` + `Assisted-by`, never `Co-Authored-By` for AI):
`AGENTS.md §Hard Constraints`.

---

## STEP 8 — Open ALL cluster PRs in parallel (no auto-merge yet)

Per PR: assemble `pr-body-<N>.md` from the template (bottom), run every check, paste evidence,
flip all boxes.
```bash
git push -u origin HEAD
grep -cE '^- \[ \]' pr-body-<N>.md   # MUST be 0 before any auto-merge
echo -n "<type>(<scope>): <subject>" | wc -c   # ≤ 72
gh pr create --base <main|branch-below> --title "<type>(<scope>): <subject>" --assignee "@me" \
  --label "type: <kind>,milestone: M5,area: <area>" --body-file pr-body-<N>.md
```
Body sections: Summary · Why (issue link) · Cluster (sibling PRs + merge order) · State files
updated · Architecture gaps (G-IDs) · Test plan · Out of scope · `Closes #N`. Don't `gh pr edit`
the title afterward. Open **all** cluster PRs before STEP 9. Confirm no *other* in-flight PR is red.

---

## STEP 9 — Ordered, in-sync merge (one PR at a time)

For `i = 1…n` in merge order (bottom-up for stacked):
```bash
PR=<pr_i> ; BR=<branch_i> ; ISSUE=<issue_i>
# 1) sync main (incl. PR_1..PR_(i-1)) and rebase THIS PR onto it
git fetch origin --prune && git checkout main && git pull --rebase origin main
git checkout "$BR" && gh pr edit "$PR" --base main 2>/dev/null || true
git rebase origin/main || { echo "rebase conflict on $BR — resolve or stop+ask"; exit 1; }
git push --force-with-lease
# 2) wait for required checks green on the rebased head (red → smallest fix → re-watch; same check fails twice → stop+ask)
gh pr checks "$PR" --watch --interval 30
# 3) NOW enable auto-merge — only this PR
gh pr merge "$PR" --auto --squash --subject "<subject> (#$PR)" --body "Closes #$ISSUE

Assisted-by: Claude/<model-id-from-this-session>"
# 4) wait until it lands (do NOT query post-merge main CI)
until [ "$(gh pr view "$PR" --json state --jq '.state')" = "MERGED" ]; do sleep 30; done
# 5) reconcile (9.R) before PR_(i+1)
```

**9.R — reconcile after each merge**
```bash
gh issue view "$ISSUE" --json number,state,milestone --jq '{n:.number,state,ms:.milestone.title}'  # expect CLOSED
[ "$(gh issue view "$ISSUE" --json state --jq .state)" = "CLOSED" ] || gh issue close "$ISSUE" --reason completed
git fetch origin && git checkout main && git pull --rebase origin main
grep -nE "#?$ISSUE\b" docs/milestones/M5-plan.md   # row must be ✅ (merged in the PR diff)
```
Row not ✅ on merged `main` → open a tiny `docs:` fix PR now.

---

## STEP 10 — Verify, summarize, stop

```bash
git fetch origin --prune && git checkout main && git pull --rebase origin main
for n in <issue_1> <issue_2> <issue_3>; do gh issue view "$n" --json number,state,milestone --jq '{n:.number,state,ms:.milestone.title}'; done  # all CLOSED
for p in <pr_1> <pr_2> <pr_3>;       do gh pr view "$p" --json number,state,mergedAt --jq '{p:.number,state,mergedAt}'; done                    # all MERGED
gh issue list --milestone "Adapter Library (M5)" --state closed --limit 200 --json number --jq '.[].number' | sort -n  # cross-check ✅ rows
gh issue list --milestone "Adapter Library (M5)" --state open   --limit 200 --json number --jq '.[].number' | sort -n  # cross-check ⬜ rows
gh pr list --author "@me" --state open --json number,title   # none from this cluster
git status --porcelain                                        # clean
```
Any mismatch → fix `M5-plan.md` + `current-milestone.md` (`docs:` commit) **before** the summary.
Post the **Session summary** (below) on each merged PR and as the final chat message. **One cluster
per session — end the turn.**

---

## Resumption tree (apply in STEP 1.4)

Health gate first, then resume in-progress, then start new.

| Observed | Action |
|---|---|
| Any open PR of mine with **red** required checks | Fix first — debug → smallest fix → push → re-watch. Don't advance/start. |
| ≥1 cluster PR merged, others open | Resume STEP 9 on the remaining PRs in order. |
| All cluster PRs open, none merged | Resume STEP 9 from the bottom of the order. |
| Cluster fully merged | STEP 10 summary. STOP. |
| Local branch w/ uncommitted work, PRs not all open | Finish STEP 5→8, then 9. |
| Local branch clean, no PR | Inspect last commit; open PR (→8) or delete branch. |
| No in-flight work, batch has ready issues | Form a cluster (STEP 2) → 5→8 → 9 → 10. STOP. |
| Closed issues not yet ✅ in plan | Flip ✅ (small `docs:` commit), then form next cluster. |
| All M5 batches exhausted | Report M5 exit-criteria status; ask user re: milestone close / M6. |

**Ordering:** A→B if B touches A's code, needs A's type/endpoint, or `M5-plan.md` lists A as B's
dependency; else independent. When unsure, treat as dependent (stack + merge after).

---

## Test plan template (`pr-body-<N>.md`)

> Every box `- [x]` with evidence before auto-merge. N/A → `- [x] (N/A — reason)`. CI required
> checks (DCO, lint, unit, security, pr-size) are the merge-queue gate, not duplicated here.

```markdown
### Local pre-flight
- [ ] `make lint-go`/`lint-protos`/`lint-agents` — exit 0 (or N/A)  [evidence]
- [ ] `GOWORK=off go test ./... -race` — pass  [evidence]
- [ ] `make test-coverage` — domain ≥90% (or N/A)  [evidence]
- [ ] `make test-coverage-adapters` ≥80% / `make test-bdd` / `make security` (or N/A)  [evidence]
### Acceptance (issue body / Canvas O-step)
- [ ] <criterion>  [evidence: test / file:line / log]
### Engineering hygiene
- [ ] **`M5-plan.md` row flipped ⬜→✅ in this diff** (mandatory)
- [ ] **`current-milestone.md` updated in this diff** (mandatory)
- [ ] Branched from fresh `origin/main` · PR ≤900 lines · trailers on every commit
- [ ] Gap scan done (G1/G4/G7/G16); opportunistic fixes noted
- [ ] Canvas O-step ✅ (feat) · `AGENTS.md` updated if new capability · `.feature` before impl
- [ ] No out-of-scope edits · no mTLS/SBOM/cosign docs claims
```

---

## Session summary (comment on each merged PR + final message)

```markdown
## Session summary — <YYYY-MM-DD HH:MM TZ>
**Outcome:** <CLUSTER-MERGED | CLUSTER-PARTIAL (n/m) | CLUSTER-OPENED-NOT-MERGED | STATE-RECONCILED-ONLY | STOPPED-HEALTH-GATE>
**Cluster:** BATCH/track · merge order #A→#B→#C (dependent|independent) · shared context: <one line>

| Issue | PR | type | size | gaps | state |
|---|---|---|---|---|---|
| #A | <url> | <feat/…> | <S> | <G-IDs> | MERGED |

**State files:** M5-plan rows ⬜→✅ <…> · current-milestone <change> · canvas <…/N/A> · AGENTS.md <svc/N/A>
**Verify (all ✓ to continue safely):** issues CLOSED ✓ · PRs MERGED ✓ · milestone⟷plan lockstep ✓ · no stray PRs/branches, tree clean ✓ · milestone now <X closed / Y open>
**Repo state:** main HEAD <sha> "<subject>" · my open PRs <list/none> · health <green | red on #PR> · branches <list/none>
**Next:** recommended cluster <#,#,# — BATCH X> (or "resume STEP 9") · decision-tree row <one line> · review first <bullets/nothing>
```
