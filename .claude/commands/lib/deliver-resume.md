---
description: "Resume work on the active milestone (state/milestone.yaml) — one canvas per EPIC, /lib:spdd-story creates all story issues in GitHub, stories are delivered via model-routed agents (one O-step per PR), stop after the cluster is merged."
argument-hint: "[optional: epic issue number or story issue number to prefer, e.g. 765 766]"
---

# /lib:deliver-resume — Resume an epic cluster (building block of /deliver)

> **Building block** — invoked by `/deliver <#epic>` to resume a cluster, not run directly.\n> **Scope contract:** the caller provides scope; milestone is optional (`--milestone` filter).\n

Pick the next ready EPIC, decompose it into story issues via `/lib:spdd-story`, dispatch each story
to its routed domain agent (one PR per O-step), merge them in O-step order, leave every state file
consistent, then stop.

> **Parallel-session safety.** Multiple sessions may run concurrently. Two mechanisms prevent
> duplicate work: (1) pre-filters in STEP 2 and STEP 3C that skip EPICs/stories whose claim branch
> or open PR already exists on the remote; (2) the atomic claim — an empty-branch push of the bare
> ref `<type>/<N>` — performed by the **dispatched agent** as its first act (shared protocol §4,
> `docs/patterns/delivery-agent-protocol.md`). Only one push wins when two sessions race; an agent
> reporting a rejected claim means the story is taken → return to STEP 3C and pick the next
> available one. Never assume a story is free just because you read it as open.

**EPIC-canvas model:** every `feat:` EPIC has exactly **one** REASONS Canvas at
`docs/spdd/<epic-issue>-<slug>/canvas.md`. That canvas's O steps map 1-to-1 to story PRs. Story
issues are created in GitHub by `/lib:spdd-story` — they reference the parent EPIC and carry full
labels, milestone, and a test plan template. `/lib:spdd-generate` always operates on the EPIC canvas,
not a per-story canvas.

> **Rules are not restated here.** Commit/PR format, conventional types, DCO + `Assisted-by`
> trailers, anti-patterns, `GOWORK=off`, PR-size, hexagonal layout, coverage gates, claim/branch
> mechanics, and the SPDD requirement all live in **`AGENTS.md`** (constitution), **`CLAUDE.md`**
> (dev loop), and the shared delivery protocol
> (**`docs/patterns/delivery-agent-protocol.md`**) that every dispatched agent reads at startup.
> This file is only the *session loop*.

> **Context budget.** This session reads planning state, issue bodies, and canvas status lines —
> never code files, test output, or expert-guide/agent-definition contents. Implementation context
> belongs inside the dispatched agents (each `.claude/agents/<name>.md` pins model, effort, tools).

**Session policy.** One **EPIC** per session. Each O-step ships as its **own** PR (= one story
issue), delivered end-to-end by a routed domain agent (claim → implement → gates → PR → CI →
queue merge → cleanup). Dispatch the cluster, collect the agents' results, and **stop** — do not
pick another EPIC. The STEP 1.5 merge pass at the start of the *next* session merges any leftover
green PRs in O-step order. `Closes #<story-N>` in the PR body closes the story issue automatically
on squash-merge. Every PR flips its own story-issue row in `"$PLANNING_DOC"` and updates
`state/current-milestone.md` in its own diff.

---

## Branch discipline (non-negotiable — ADR-023)

- Arm the queue with `gh pr merge <PR> --squash --auto` — `BEHIND` is cosmetic; the queue validates
  against current main (never rebase for freshness; a force-push ejects a queued PR).
- Only `DIRTY` (real conflicts) rebases: `git rebase --signoff origin/main` + `git push --force-with-lease`.
- `--squash` only — never `--merge` (`required_linear_history`) or `--rebase` (`required_signatures`).
- Resume nuance: arm/merge story PRs in **O-step order** (lowest story number first).
- No direct commits to `main` · delete remote branches after merge · never reopen a closed PR/branch.
- Full rules: `docs/patterns/delivery-agent-protocol.md` §6 + ADR-047.

---

## STEP 1 — Orient & resume (run first)

**1.1 Sync main + load milestone config**
```bash
git fetch origin --prune && git checkout main && git pull --rebase origin main
[ "$(git rev-parse main)" = "$(git rev-parse origin/main)" ] || { echo "main diverged — STOP"; exit 1; }
[ -z "$(git status --porcelain)" ] || { echo "dirty tree — STOP"; git status; exit 1; }

# Active-milestone config (SSoT: state/milestone.yaml) — single helper call, never inline awk.
eval "$(bash automation/milestone-env.sh)"
# → MILESTONE_NAME MILESTONE_TITLE MILESTONE_NUMBER MILESTONE_VERSION PLANNING_DOC MILESTONE_LABEL GH_MILESTONE

# Per-invocation run id — namespaces every dispatched agent worktree
# (canvas dispatch STEP 3A, story dispatch STEP 5, post-merge dispatch + sweep STEP 6).
# Two concurrent sessions get distinct ids, so neither can ever touch the other's trees.
RESUME_RUN_ID="$(date +%s)-$$"
export RESUME_RUN_ID
```

**1.2 Read** — mandatory every session:
- `state/current-milestone.md`
- `"$PLANNING_DOC"` (epic/story tables + risk table)
- `CLAUDE.md`

| Read when | File |
|---|---|
| Any ADR-governed decision | `docs/adr/INDEX.md` — check the ADR that governs the area before changing it |
| EPIC-specific caveats | The EPIC's row + notes in `"$PLANNING_DOC"` — milestone-specific guidance lives there, never here |

Domain references (proto/BDD patterns, Helm charts, `images/images.yaml` SoT) are read by the
dispatched agents per their definitions — not by this session.

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
gh pr merge <PR> --squash --auto   # merge queue handles freshness (ADR-047);
                                   # remote branch auto-deleted on merge
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

# Arm the merge queue on green PRs (ADR-047): CLEAN or BEHIND both qualify —
# the queue handles freshness; never rebase for freshness (a force-push
# ejects a queued PR). DIRTY needs a manual --signoff rebase first.
while IFS=$'\t' read -r PR_N BR MERGE_STATE REQ_CONCLUSIONS; do
  if [[ "$MERGE_STATE" == "CLEAN" || "$MERGE_STATE" == "BEHIND" ]] && ! echo "$REQ_CONCLUSIONS" | grep -qE 'FAILURE|ERROR|TIMED_OUT'; then
    echo "Arming queue merge on PR #$PR_N ($BR) — all required checks green"
    gh pr merge "$PR_N" --squash --auto
  else
    echo "Skipping PR #$PR_N ($BR) — not yet green (state=$MERGE_STATE)"
  fi
done <<< "$OPEN_PRS"
# Fallback (no merge-queue rule on main — pre-cutover or rollback): rebase
# each green branch onto origin/main + --force-with-lease before arming.
```

After the pass: if all your PRs merged and the EPIC is complete → STEP 7 + stop.
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

# For each EPIC candidate (in priority order), check if any story claim branch is on remote.
# Claim refs are the bare <type>/<story-issue-N> (protocol §4 — slug applied only post-claim);
# match slugged variants too. Pattern: ^<type>/<N>(-|$).
# Example — EPIC #765 has stories #779–#792; skip #765 if any of those branches exist:
echo "$CLAIMED" | grep -E "^(feat|fix|refactor|docs|test|ci|chore)/(779|780|781|782|783|784|785|786|787|788|789|790|791|792)(-|$)"
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
- No canvas, EPIC is `feat:` → go to STEP 3A (synchronous `spdd-canvas` dispatch)
- EPIC is `refactor:/ci:/chore:` → go to STEP 3D (SPDD-exempt)
- EPIC is BLOCKED → report blocker, pick a different EPIC

---

## STEP 3 — EPIC canvas + story decomposition

### 3A — Canvas not yet Aligned (synchronous `spdd-canvas` dispatch)

The SPDD pipeline (analysis → canvas → security-review) runs inside the `spdd-canvas` agent —
never inline in this session. Dispatch it **synchronously** (`run_in_background: false`): an
Aligned canvas must exist before any story dispatch.

```
Agent({
  description: "EPIC #<EPIC_N> canvas — <epic title>",
  subagent_type: "spdd-canvas",
  run_in_background: false,
  prompt: """
    ISSUE: #<EPIC_N> — <epic title>
    REPO:  <REPO>
    WT:    /tmp/zynax-resume-<RESUME_RUN_ID>-<EPIC_N>     (your literal private worktree path)

    Issue body:
    <full issue body from gh issue view EPIC_N>

    Context files (read these before writing the canvas):
    <2-3 specific repo paths named in the issue body>

    Produce the EPIC canvas per your agent definition (analysis → canvas →
    security-review, must PASS) at docs/spdd/<EPIC_N>-<slug>/canvas.md. Read
    docs/patterns/delivery-agent-protocol.md and your expert guide first.
    Leave the canvas at `Status: Draft` and STOP for human review — do NOT
    auto-align it even if the security review PASSes; the human alignment
    gate is the only path to `Status: Aligned`. Report the canvas path and
    the security-review verdict. End with the ## Result and
    ## Session Learnings blocks.
  """
})
```

Canvas content requirements (R/E/A/S/O/N sections, one O-step per story PR each ≤400 lines,
observable infra DoD, Tier 2 findings → `canvas.private.md`) live in the `spdd-canvas` agent
definition and the SPDD guide — not restated here.

**[Human reviews and sets Status: Aligned — then re-run `/deliver <EPIC_N>` to resume at
STEP 3B.]** The canvas comes back `Status: Draft` (human gate pending — the agent never
auto-aligns): stop the session and report the canvas path + review verdict — never dispatch
stories from an unaligned canvas (`/lib:spdd-generate` inside the domain agents refuses it
anyway).

### 3B — Create story issues in GitHub (run once per EPIC)

Check whether story issues already exist before creating:
```bash
gh issue list --milestone "$GH_MILESTONE" --state all \
  --json number,title,body --jq '.[] | select(.body | test("#<EPIC_N>"))' | head -40
```

If stories are missing, run:
```bash
/lib:spdd-story <EPIC_N>

# Locked decision #1107: /lib:spdd-story is milestone-agnostic — the CALLER injects
# the active milestone label + GitHub milestone on every story it just created
# (idempotent — re-running re-applies the same label/milestone):
for STORY in $(gh issue list --state open --limit 100 --json number,body \
  --jq '.[] | select(.body | test("#<EPIC_N>")) | .number'); do
  gh issue edit "$STORY" --add-label "$MILESTONE_LABEL" --milestone "$GH_MILESTONE"
done
```

`/lib:spdd-story` will create one GitHub issue per O-step. **Each story issue MUST have:**
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
# For each candidate story #N: check if the bare claim ref or a slugged variant is in $CLAIMED
# echo "$CLAIMED" | grep -E "^[a-z]+/<N>(-|$)"
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

## STEP 4 — Route + soft-claim the cluster (per O-step story)

Verify the agent roster exists — do **not** read the files (the harness loads them at dispatch):
```bash
ls .claude/agents/
# go-services.md | python-adapters.md | bdd-contract.md | infra-helm.md |
# ci-release.md  | spdd-canvas.md     | post-merge.md
```
Directory missing → stop and report (the model-routing PR not yet merged, or the session started
before `.claude/agents/` existed — restart the session once).

Issue routing rules (apply in order — first match wins). Agent names are the
`.claude/agents/<name>.md` definitions; each pins its own model and effort
(implementation → Opus `xhigh`; canvas → Fable `high`; post-merge → Haiku):

| Issue title pattern | Agent (`subagent_type`) |
|---|---|
| `(api-gateway)` / `(workflow-compiler)` / `(engine-adapter)` / `(task-broker)` / `(agent-registry)` / `(event-bus)` / `(memory-service)` | `go-services` |
| `(infra)` / `helm` / `k8s` in title (case-insensitive) | `infra-helm` |
| `(ci)` / `actions` / `workflow` / `images.yaml` in title | `ci-release` |
| `feat:` type AND no `Status: Aligned` canvas found | `spdd-canvas` first, then domain agent |
| `(agents)` / `(sdk)` / `python` / `adapter` in title | `python-adapters` |
| `test:` type OR `protos/tests` OR `.feature` in issue body | `bdd-contract` |

In this loop the `spdd-canvas`-first row is satisfied by STEP 3A — the synchronous canvas
dispatch always precedes any story dispatch.

**Soft claim** (label + assignee). The *atomic* claim — the empty-branch push of the bare ref
`<type>/<N>` — is performed by the dispatched agent itself as its first act (protocol §4), never
by this session:

```bash
gh label create "status: in-progress" --color "FBCA04" \
  --description "Actively being implemented" 2>/dev/null || true

for N in $CLUSTER_STORIES; do
  gh issue edit "$N" --add-label "status: in-progress" --add-assignee "@me"
done
```

**Size gate:** XS<50 / S 50–200 / M 200–400 → dispatch · L 401–900 → agent justifies in PR body ·
**XL>900 → STOP**, split the O-step, open a follow-up issue *before* dispatching.

**BDD gate (ADR-016):** a story touching a gRPC boundary must land its `.feature` file before
implementation — the routed agent enforces this per its guide; name the feature path in the
dispatch context files.

---

## STEP 5 — Dispatch the routed domain agent (replaces inline implementation)

No code is written in this session. Spawn one Agent per cluster story. **Independent O-steps
(≤3):** dispatch all in background, in parallel. **Dependent O-steps:** dispatch one at a time in
O-step order — start the next only after the previous agent reports its PR merged (this replaces
branch stacking). The agent's definition supplies its model, effort, tools, and instructions to
read the shared protocol (`docs/patterns/delivery-agent-protocol.md`) and its domain guide — the
dispatch prompt carries **only the per-story facts**:

```
Agent({
  description: "Story #N — <story title>",
  subagent_type: "<agent from STEP 4, e.g. go-services>",
  run_in_background: true,
  prompt: """
    ISSUE: #N — <story title>
    REPO:  <REPO>
    WT:    /tmp/zynax-resume-<RESUME_RUN_ID>-<N>     (your literal private worktree path)

    Issue body:
    <full issue body from gh issue view N>

    Context files (read these before writing any code):
    docs/spdd/<EPIC_N>-<slug>/canvas.md — O-step <N> of EPIC #<EPIC_N>
    <1-2 specific repo paths named in the issue body or canvas O-step>

    Deliver this story end-to-end per your agent definition: read
    docs/patterns/delivery-agent-protocol.md and your expert guide first, then
    claim → implement → gates → PR → CI → queue merge → cleanup. End with the
    ## Result and ## Session Learnings blocks (the orchestrator parses both).
  """
})
```

Worktree paths are run-scoped and private: `/tmp/zynax-resume-<RESUME_RUN_ID>-<N>` — created
first / removed last by the agent, distinct from `/tmp/zynax-orch-*` (`/lib:deliver-batch`) and
`/tmp/zynax-auto-*` (`/deliver`), so no namespace collision.

---

## STEP 6 — Collect results (wait for all background agents)

As each agent completes, extract:
1. Issue number + PR URL
2. CI status (green / red / pending)
3. `## Result` block — especially `Merge SHA` and `Affected services`
4. `## Session Learnings` block

**Per-story state contract** — verify on each merged PR (the agent ships these in its own diff):
1. `"$PLANNING_DOC"` — this story's row ⬜→✅; "Last updated" bumped
2. `state/current-milestone.md` — EPIC progress updated; noted if EPIC is now fully done
3. Epic canvas O-step marked ✅ (`/lib:spdd-sync <canvas>` run if implementation diverged)
4. `services/<svc>/AGENTS.md` — only if a new gRPC method, K8s resource type, or env var was added

A merged PR missing a state flip → small `docs:` follow-up branch → PR → queue merge (never a
direct commit to `main`).

- Agent reports **CI failure** → report the failing check name to the user; do not retry
  automatically — human intervention required.
- Agent reports **claim rejected** (branch `<type>/<N>` already on remote) → story owned by
  another session → return to STEP 3C and dispatch the next open O-step.
- Agent result **missing** (crash) but its claim branch exists on origin → finish the delivery
  **from the leaked worktree** per `/lib:deliver-batch` STEP 7 (crashed-agent recovery): inspect
  `/tmp/zynax-resume-<RESUME_RUN_ID>-<N>`, commit if needed, push HEAD onto the **surviving**
  claim ref — bare `<type>/<N>` OR slugged `<type>/<N>-*` — then open/finish the PR. Never
  delete the claim branch, and never blind re-dispatch while the claim ref exists (a
  re-dispatched agent's first act is the §4 empty-branch claim push, guaranteed rejected) —
  never sweep pushed work blindly.

**Post-merge verification (mirror of `/lib:deliver-batch` STEP 7.5).** For each **merged** PR
collected above — dedupe by merge SHA, exactly one verifier per merge SHA — dispatch one
`post-merge` agent **in background** (its definition pins Haiku: mechanical GitHub/GHCR
verification only). Run them in parallel:

```
Agent({
  description: "Post-merge verify PR #PR_N (issue #N)",
  subagent_type: "post-merge",
  run_in_background: true,
  prompt: """
    PR_NUMBER:    <PR_N>
    MERGE_SHA:    <S>
    ISSUE_NUMBER: <issue N>
    SESSION_DATE: <date>
    REPO:         <REPO>
    WT:           /tmp/zynax-resume-postmerge-<RESUME_RUN_ID>-<PR_N>   (your literal private worktree path)

    Verify post-merge CI, GHCR artifacts, and digest pins for this merge per your
    agent definition (read docs/patterns/delivery-agent-protocol.md and your expert
    guide first). Back-fill the originating PR's "Post-merge digest sync → main"
    Evidence placeholder. End with the ## Post-Merge Evidence block and
    ## Session Learnings.
  """
})
```

Collect the post-merge agents' results the same way as the domain agents' (wait for completion);
their `## Session Learnings` blocks join the persistence pass below.

Persist each `## Session Learnings` block (domain agents + post-merge agents) to
`docs/ai-learnings/<domain>.md` via a `docs:` PR (same pattern as `/lib:deliver-batch` STEP 8).

**Leftover worktree sweep** — crashed agents only, this run's namespace only (never glob-all,
which would delete a concurrent run's live trees):
```bash
for WT in /tmp/zynax-resume-${RESUME_RUN_ID}-* /tmp/zynax-resume-postmerge-${RESUME_RUN_ID}-*; do
  [ -d "$WT" ] || continue
  git worktree remove "$WT" --force 2>/dev/null || true
  rm -rf "$WT" 2>/dev/null || true
done
git worktree prune
```

---

## STEP 7 — EPIC completion check + stop

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
| Open PR of mine, all checks **green** (CLEAN) | STEP 1.5 merge pass → merge now; continue to remaining O-steps or STEP 7 |
| Some story PRs merged, others open for same EPIC | STEP 1.5 merge pass; if unmerged PRs still running, pick next unclaimed O-step |
| All story PRs open, none merged | STEP 1.5 merge pass; if none green yet, pick next unclaimed O-step if EPIC has more stories |
| EPIC fully merged | STEP 7 summary. STOP. |
| Remote claim branch `<type>/<N>` or slugged `<type>/<N>-*`, no PR, no live agent | Crashed delivery — finish it from the leaked worktree per `/lib:deliver-batch` STEP 7 (crashed-agent recovery): inspect, commit if needed, push HEAD onto the surviving claim ref, open/finish the PR. Never delete the claim branch, never blind re-dispatch while the claim ref exists (the re-dispatched agent's §4 claim push is guaranteed rejected); never sweep pushed work blindly |
| Agent reports claim push rejected (branch exists on remote) | Another session claimed that story — return to STEP 3C, pick next open O-step |
| No in-flight work, EPIC has stories created | Pick next O-step → STEP 4 |
| EPIC has canvas Aligned but no story issues | STEP 3B: create stories, then STEP 4 |
| EPIC has no canvas | STEP 3A: synchronous `spdd-canvas` dispatch |
| EPIC is BLOCKED (#764 for EPIC I, #626 for EPIC J, #626+#772 for DevAuto.8 #881) | Pick next unblocked EPIC |
| All EPICs exhausted | Report milestone exit-criteria; recommend /milestone close |

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
- [ ] Canvas O-step <N> marked ✅; `/lib:spdd-sync` run if impl diverged
- [ ] Branched from fresh `origin/main` · PR ≤900 lines · trailers on every commit
- [ ] No out-of-scope edits · observability (traces/dashboards) deferred to a later milestone
```

---

## Session summary

```markdown
## Session summary — <YYYY-MM-DD HH:MM TZ>
**Outcome:** <EPIC-COMPLETE | EPIC-PARTIAL (n/m O-steps) | STORIES-CREATED-NOT-IMPLEMENTED | STOPPED-BLOCKED | STOPPED-HEALTH-GATE>
**EPIC:** #<N> <title> · canvas `docs/spdd/<N>-<slug>/canvas.md` · O-steps this session: <X–Y of Z>

| Story | PR | agent | type | size | O-step | state |
|---|---|---|---|---|---|---|
| #A | <url> | <go-services/…> | <feat/…> | <S/M> | O-step N | MERGED |

**State files:** planning-doc rows ⬜→✅ <list> · current-milestone <change> · canvas O-steps ✅ <list> · AGENTS.md <svc/N/A>
**Verify (all ✓ to continue):** story issues CLOSED ✓ · PRs MERGED ✓ · planning-doc lockstep ✓ · no stray PRs/branches, tree clean ✓
**Blockers:** <list any blocking issues — e.g. "#764 EBUS-DECISION unresolved — EPIC I stays blocked">
**Repo state:** main HEAD <sha> "<subject>" · my open PRs <list/none> · health <green | red on #PR>
**Next:** EPIC <#N title> · O-step <N> (#story-issue) · review first: <canvas / issue / N/A>
```
