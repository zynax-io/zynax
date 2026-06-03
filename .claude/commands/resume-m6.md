---
description: Resume Zynax M6 work — one canvas per EPIC, /spdd-story creates all story issues in GitHub, /spdd-generate implements one O-step at a time, stop after the cluster is merged.
argument-hint: "[optional: epic issue number or story issue number to prefer, e.g. 765 766]"
---

# Resume M6 — K8s Production-Ready (v0.5.0)

Pick the next ready EPIC, decompose it into story issues via `/spdd-story`, ship each story as its
own PR in O-step order, merge them in order, leave every state file consistent, then stop.

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
issue). Open all story-PRs in the cluster in parallel after stories are created, then merge them
**strictly in O-step order**. Never enable auto-merge on >1 PR at once. Never merge by hand. Every
PR flips its own story-issue row in `docs/milestones/M6-planning.md` and updates
`state/current-milestone.md` in its own diff.

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

**1.1 Sync main**
```bash
git fetch origin --prune && git checkout main && git pull --rebase origin main
[ "$(git rev-parse main)" = "$(git rev-parse origin/main)" ] || { echo "main diverged — STOP"; exit 1; }
[ -z "$(git status --porcelain)" ] || { echo "dirty tree — STOP"; git status; exit 1; }
```

**1.2 Read** — mandatory every session:
- `state/current-milestone.md`
- `docs/milestones/M6-planning.md` (M6 epic/story tables + risk table)
- `CLAUDE.md`

| Read when | File |
|---|---|
| Any EPIC touching event-bus | ADR-022 Accepted (#764 closed) — EPIC I (#772) needs canvas via `/spdd-reasons-canvas 772` |
| Any memory-service work | Confirm single-store choice is in J.2 canvas safeguards |
| Proto or BDD boundary touched | `protos/AGENTS.md` + `docs/patterns/bdd-contract-testing.md` |
| Helm chart work | `infra/AGENTS.md` + `docs/patterns/helm-charts.md` |
| Any ADR-governed decision | `docs/adr/INDEX.md` — ADR-022 (event-bus) Accepted; all 22 ADRs stable |

Run `/help` to confirm all SPDD commands are available.

**1.3 Reconcile M6 milestone ⟷ planning doc**
```bash
gh issue list --milestone "K8s Production-Ready (M6)" --state open   --limit 200 --json number,title,labels,state
gh issue list --milestone "K8s Production-Ready (M6)" --state closed --limit 200 --json number,title,labels,state
```
Open issue ⇒ ⬜ row; closed issue ⇒ ✅ row. Any mismatch → fix via a dedicated `docs:` branch
and PR (never a direct commit to `main`):
```bash
git checkout -b docs/milestone-sync-$(date +%Y%m%d)
# edit M6-planning.md and/or current-milestone.md
git add docs/milestones/M6-planning.md state/current-milestone.md
git commit -s -m "docs(milestones): reconcile M6 planning table — <what changed>

Assisted-by: Claude/<model>"
git push -u origin HEAD
gh pr create --title "docs(milestones): reconcile M6 planning table" \
  --body "Reconcile open/closed issue state with delivery table." \
  --label "type: docs,milestone: M6"
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

---

## STEP 2 — Pick an EPIC

**Priority order** (highest first — but always check blockers first):

| Priority | EPIC | Issue | Status gate |
|---|---|---|---|
| ✅ | M6.A Health probes | #463 | **COMPLETE** — #487 merged in PR #821 |
| ✅ | M6.D Stateless compiler | #466 | **COMPLETE** — #490 merged in PR #774 |
| 1 | M6.B mTLS | #464 | canvas `Aligned`; child #488 |
| 2 | M6.C Supply chain | #465 | canvas `Aligned`; child #489 |
| 3 | M6.Helm | #765 | canvas `Aligned` `docs/spdd/765-helm-charts/canvas.md`; children #779–#792 |
| 4 | M6.H Postgres repos | #626 | canvas `Aligned` `docs/spdd/626-postgres-repos/canvas.md`; children #793 #794 |
| 5 | M6.F Config convergence | #670 | refactor/ci — SPDD-exempt; children #667 #668 #669 |
| 6 | M6.NS Multi-namespace | #767 | canvas `Aligned` `docs/spdd/767-multi-namespace/canvas.md`; children #799 #800; D.1 done (#774) |
| 7 | M6.Argo | #766 | canvas `Aligned` `docs/spdd/766-argo-engine/canvas.md`; children #795–#798 |
| 8 | M6.SDK PyPI | #769 | canvas `Aligned` `docs/spdd/769-sdk-pypi/canvas.md`; children #805–#808 |
| 9 | M6.Policy | #768 | canvas `Aligned` `docs/spdd/768-policy-enforcement/canvas.md`; children #801–#804 |
| 10 | M6.J memory-service | #773 | canvas `Aligned` `docs/spdd/773-memory-service/canvas.md`; children #814–#819; **BLOCKED on #626** |
| 11 | M6.I event-bus | #772 | ADR-022 Accepted (#764 closed) — no canvas yet; run `/spdd-reasons-canvas 772` first |
| 12 | M6.G e2e harness | #770 | canvas `Aligned` `docs/spdd/770-e2e-harness/canvas.md`; children #809–#813; BLOCKED on EPIC A + I + J + B |

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

**Canvas must include** for M6 EPICs:
- **R**: exact K8s DoD (observable outcomes, not just "charts exist")
- **E**: every new resource type, every gRPC contract touched
- **A**: what we WILL do and what we WON'T (e.g. "no OTel traces — that is M7")
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
gh issue list --milestone "K8s Production-Ready (M6)" --state all \
  --json number,title,body --jq '.[] | select(.body | test("#<EPIC_N>"))' | head -40
```

If stories are missing, run:
```bash
/spdd-story <EPIC_N>
```

`/spdd-story` will create one GitHub issue per O-step. **Each story issue MUST have:**
- Title: `feat(<scope>): <story-title> (#<EPIC_N>, step <N>)` — conventional-commit form
- Labels: `type: feature`, `area: <area>`, `milestone: M6`, `size/<S|M|L>`, `spdd: canvas-step`
- Milestone: `K8s Production-Ready (M6)`
- Body includes:
  - Story (as-a / I-want / so-that)
  - Canvas reference: `docs/spdd/<EPIC_N>-<slug>/canvas.md` O-step N
  - Scope (what this PR changes)
  - Acceptance criteria (≥3, each concrete and testable)
  - Out of scope (what the next O-step handles)
  - Size estimate
  - Dependencies (previous O-step issue, if any)
  - Test plan checklist (see template below)
  - `Assisted-by: Claude/claude-sonnet-4-6`

Verify all stories were created:
```bash
gh issue list --label "milestone: M6" --state open --json number,title \
  --jq '.[] | select(.title | test("#<EPIC_N>"))'
```

### 3C — Pick the next unmerged O-step

```bash
# Find which O-steps are already merged (look for closed story issues)
gh issue list --label "milestone: M6" --state closed --json number,title,body \
  --jq '.[] | select(.body | test("#<EPIC_N>.*step")) | {n:.number,title}'

# Pick the lowest open step number
gh issue list --label "milestone: M6" --state open --json number,title,body \
  --jq '.[] | select(.body | test("#<EPIC_N>.*step")) | {n:.number,title}' | head -10
```

**One O-step at a time.** The cluster for this session = one EPIC's next ≤3 O-steps (if they
are independent) or the next single O-step (if each depends on the previous).

### 3D — SPDD-exempt EPICs (refactor:/ci:/chore:)

No canvas required. Implement directly from the issue body. Each story issue still needs the
test plan template and labels — create them via:
```bash
gh issue create --title "<type>(<scope>): <title> (#<EPIC_N>, step <N>)" \
  --label "type: <refactor|ci|chore>,area: <area>,milestone: M6,size/<S|M>" \
  --milestone "K8s Production-Ready (M6)" \
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

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"
```

---

## STEP 4 — Scope + branch (per O-step story)

```bash
git fetch origin --prune && git checkout main && git pull --rebase origin main
[ "$(git rev-parse main)" = "$(git rev-parse origin/main)" ] || { echo "main diverged"; exit 1; }
git checkout -b <type>/<story-issue-N>-<short-slug>
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
1. `docs/milestones/M6-planning.md` — flip this story's row ⬜→✅; bump "Last updated"
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

## STEP 8 — Open all story PRs in parallel (no auto-merge yet)

```bash
git push -u origin HEAD
echo -n "<type>(<scope>): <subject>" | wc -c   # ≤ 72
gh pr create --base <main|branch-below> \
  --title "<type>(<scope>): <subject>" \
  --assignee "@me" \
  --label "type: <kind>,milestone: M6,area: <area>" \
  --body-file pr-body-<N>.md
```

Open **all** cluster story PRs before STEP 9. Verify no other open PR of yours has red checks.

---

## STEP 9 — Ordered merge (one PR at a time)

For `i = 1…n` in O-step order:
```bash
PR=<pr_i>; BR=<branch_i>; STORY=<story_issue_i>
git fetch origin --prune && git checkout main && git pull --rebase origin main
git checkout "$BR" && gh pr edit "$PR" --base main 2>/dev/null || true
git rebase origin/main || { echo "rebase conflict — resolve or stop+ask"; exit 1; }
git push --force-with-lease
gh pr checks "$PR" --watch --interval 30
gh pr merge "$PR" --squash
until [ "$(gh pr view "$PR" --json state --jq .state)" = "MERGED" ]; do sleep 30; done
git push origin --delete "$BR" 2>/dev/null || true
```

**After each merge (9.R):**
```bash
[ "$(gh issue view "$STORY" --json state --jq .state)" = "CLOSED" ] || gh issue close "$STORY" --reason completed
git fetch origin && git checkout main && git pull --rebase origin main
grep -nE "#?$STORY\b" docs/milestones/M6-planning.md   # row must be ✅
```

---

## STEP 10 — EPIC completion check + stop

When ALL O-steps of an EPIC are merged:
```bash
# Check all story issues for this EPIC are closed
gh issue list --label "milestone: M6" --state open --json number,title,body \
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
| Open PR of mine with red required checks | Fix first; do not advance or start new cluster |
| Some story PRs merged, others open for same EPIC | Resume STEP 9 on remaining in O-step order |
| All story PRs open, none merged | Resume STEP 9 from O-step 1 |
| EPIC fully merged | STEP 10 summary. STOP. |
| Local branch w/ uncommitted work, no PR | Finish STEP 5→8 |
| No in-flight work, EPIC has stories created | Pick next O-step → STEP 4 |
| EPIC has canvas Aligned but no story issues | STEP 3B: create stories, then STEP 4 |
| EPIC has no canvas | STEP 3A: full pipeline |
| EPIC is BLOCKED (#764 for EPIC I, #626 for EPIC J) | Pick next unblocked EPIC |
| All EPICs exhausted | Report M6 exit-criteria; ask re: milestone close / M7 |

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
- [ ] **`M6-planning.md` row ⬜→✅ in this diff** (mandatory)
- [ ] **`current-milestone.md` updated in this diff** (mandatory)
- [ ] Canvas O-step <N> marked ✅; `/spdd-sync` run if impl diverged
- [ ] Branched from fresh `origin/main` · PR ≤900 lines · trailers on every commit
- [ ] No out-of-scope edits · observability (traces/dashboards) deferred to M7
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

**State files:** M6-planning rows ⬜→✅ <list> · current-milestone <change> · canvas O-steps ✅ <list> · AGENTS.md <svc/N/A>
**Verify (all ✓ to continue):** story issues CLOSED ✓ · PRs MERGED ✓ · M6-planning lockstep ✓ · no stray PRs/branches, tree clean ✓
**Blockers:** <list any blocking issues — e.g. "#764 EBUS-DECISION unresolved — EPIC I stays blocked">
**Repo state:** main HEAD <sha> "<subject>" · my open PRs <list/none> · health <green | red on #PR>
**Next:** EPIC <#N title> · O-step <N> (#story-issue) · review first: <canvas / issue / N/A>
```
