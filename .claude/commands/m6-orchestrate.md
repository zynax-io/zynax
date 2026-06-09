---
description: Parallel M6 orchestrator — reads state, claims up to 3 issues per batch, routes each to the right domain expert subagent, runs them in parallel, collects results and learnings. Orchestrator never reads code files directly.
argument-hint: "[--batch-size N]  default: 3"
---

# /m6-orchestrate — Parallel M6 Delivery Orchestrator

Thin coordination layer: read state → claim issues → fan out to expert subagents in parallel →
collect results → persist learnings → report.

> **Context budget discipline.** The orchestrator's context is ~8K tokens maximum. It reads
> planning state only — never code files, never canvas body, never test output. Those all live
> in the expert subagent's isolated context. This is the point: each expert starts fresh with
> only what it needs, leaving room for deep domain expertise injection.

> **Rules are not restated here.** See `AGENTS.md`, `CLAUDE.md`, and the individual expert
> files under `.claude/commands/experts/` for domain rules. This file is the coordination
> loop only.

---

## STEP 0 — Read expert files (load routing table)

```bash
# Expert files live under .claude/commands/experts/
ls .claude/commands/experts/
# go-services.md | infra-helm.md | ci-release.md | spdd-canvas.md | python-adapters.md | bdd-contract.md | post-merge.md
```

Do not read the expert file contents now — they are injected into subagents at dispatch time.
Just verify they exist.

---

## STEP 1 — Read planning state (orchestrator context budget: ~8K tokens)

```bash
# Per-invocation run id — namespaces every worktree and the crash-recovery sweep (STEP 7).
# Two concurrent orchestrator runs get distinct ids, so neither can ever touch the other's trees.
ORCH_RUN_ID="$(date +%s)-$$"
export ORCH_RUN_ID

# Coordinator worktree — the orchestrator's OWN git operations (this sync, the STEP 2 merge
# pass, the STEP 8 learnings PR) run here, never in the user's primary checkout. The user's
# working directory is left exactly as they had it.
REPO=$(git rev-parse --show-toplevel)
COORD_WT="/tmp/zynax-orch-coord-${ORCH_RUN_ID}"
git -C "$REPO" worktree remove "$COORD_WT" --force 2>/dev/null || true
rm -rf "$COORD_WT" 2>/dev/null || true
git -C "$REPO" fetch origin --prune
git -C "$REPO" worktree add "$COORD_WT" origin/main   # detached at origin/main
cd "$COORD_WT"

# Read only these four files — nothing else (from the coordinator worktree)
cat state/current-milestone.md           # blockers, active work
cat docs/milestones/M6-planning.md       # EPIC status + dependency table
```

```bash
# Snapshot GitHub state
BATCH_SIZE=${ARGUMENTS:-3}               # default 3 parallel issues

OPEN=$(gh issue list \
  --milestone "K8s Production-Ready (M6)" \
  --state open --limit 300 \
  --json number,title,body,labels,assignees)

OPEN_PRS=$(gh pr list --state open --limit 100 \
  --json number,headRefName,author,mergeStateStatus,statusCheckRollup)

git fetch origin --prune
REMOTE_BRANCHES=$(git ls-remote origin 'refs/heads/*' \
  | awk '{print $2}' | sed 's|refs/heads/||')
```

---

## STEP 2 — Quick merge pass (≤60 s)

Before claiming new work, merge any open PRs that are already CLEAN. This runs inside the
coordinator worktree `$COORD_WT` (created in STEP 1) — never the user's checkout. Use
`git checkout --detach origin/main`, not `git checkout main`: `main` is checked out in the
user's primary worktree and git refuses to check it out a second time.

```bash
OPEN_PRS_JSON=$(gh pr list --author "@me" --state open \
  --json number,headRefName,mergeStateStatus,statusCheckRollup \
  --jq 'sort_by(.number) | .[]')

while IFS= read -r PR; do
  PR_N=$(echo "$PR" | jq -r .number)
  MERGE_STATE=$(echo "$PR" | jq -r .mergeStateStatus)
  FAILED=$(echo "$PR" | jq '[.statusCheckRollup[]? | select(.isRequired==true) | .conclusion] | any(. == "FAILURE" or . == "ERROR")')
  if [[ "$MERGE_STATE" == "CLEAN" ]] && [[ "$FAILED" == "false" ]]; then
    BR=$(echo "$PR" | jq -r .headRefName)
    git checkout -B "$BR" "origin/$BR" && git rebase origin/main && git push --force-with-lease
    gh pr merge "$PR_N" --squash
    until [ "$(gh pr view "$PR_N" --json state --jq .state)" = "MERGED" ]; do sleep 10; done
    git push origin --delete "$BR" 2>/dev/null || true
    git fetch origin --prune && git checkout --detach origin/main
  fi
done <<< "$OPEN_PRS_JSON"
```

---

## STEP 3 — Select READY batch

Using the same classification logic as `/m6-plan`:

```bash
# For each open issue: classify as READY / IN_PROGRESS / BLOCKED
# Priority order from M6-planning.md EPIC table
# Select top $BATCH_SIZE READY issues for this session

# Quick filter:
# 1. Has "status: in-progress" label → IN_PROGRESS (skip)
# 2. Has remote branch matching the claim key <type>/<N> (bare) OR a slugged
#    <type>/<N>-* variant → IN_PROGRESS (skip). Match with `^<type>/<N>(-|$)`.
# 3. Has "Pending #X" or "Depends on #X" in body where X is open → BLOCKED (skip)
# 4. Otherwise → READY
```

Report the selected batch and their expert routing before dispatching:

```bash
# Build the issues list string for log tags
ISSUES_LIST=$(echo "$BATCH_ISSUES" | tr ' ' ',' | sed 's/^/#/;s/,/,#/g')

echo ""
echo "=== [orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] BATCH SELECTED — $BATCH_SIZE issues ==="
# For each issue in the batch, print: #N → <expert-tag>  <title>
# e.g.:
#   #823 → go-svc      feat(event-bus): service scaffold
#   #865 → ci-rel      ci(infra): OCI manifest annotations
#   #875 → ci-rel      chore(automation): expert mesh YAML configs
echo "==="
```

---

## STEP 4 — Claim all batch issues (soft claim)

```bash
# Ensure status: in-progress label exists
gh label create "status: in-progress" --color "FBCA04" \
  --description "Actively being implemented" 2>/dev/null || true

# Claim each selected issue
for N in $BATCH_ISSUES; do
  gh issue edit "$N" --add-label "status: in-progress" --add-assignee "@me"
done
```

---

## STEP 5 — Route issues to expert files

Issue routing rules (apply in order — first match wins):

| Issue title pattern | Expert file |
|---|---|
| `(api-gateway)` / `(workflow-compiler)` / `(engine-adapter)` / `(task-broker)` / `(agent-registry)` / `(event-bus)` / `(memory-service)` | `experts/go-services.md` |
| `(infra)` / `helm` / `k8s` in title (case-insensitive) | `experts/infra-helm.md` |
| `(ci)` / `actions` / `workflow` / `images.yaml` in title | `experts/ci-release.md` |
| `feat:` type AND no `Status: Aligned` canvas found | `experts/spdd-canvas.md` first, then domain expert |
| `(agents)` / `(sdk)` / `python` / `adapter` in title | `experts/python-adapters.md` |
| `test:` type OR `protos/tests` OR `.feature` in issue body | `experts/bdd-contract.md` |

Multi-expert issues: run `spdd-canvas` synchronously first (it must produce an Aligned canvas
before implementation can start), then dispatch the domain expert for the implementation.

After applying the routing table, emit one log line per issue:

```bash
# For each issue N with resolved expert tag E:
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] ROUTE: #$N → $E  ($ISSUE_TITLE)"
```

### Pre-spawn reconcile (idempotency layer 2 — dispatch-time early-out)

The batch was selected from a once-read snapshot (STEP 1). Before spawning an agent for
issue `N`, re-query **live** state — a concurrent session on another machine may have closed
or merged it in the interval. This is a cheap early-out, not the authoritative check (that is
STEP 7's merge-SHA dedupe, which operates on a merge fact and is immune to GitHub API lag).

```bash
DISPATCH_ISSUES=""
for N in $BATCH_ISSUES; do
  STATE=$(gh issue view "$N" --json state --jq .state)
  MERGED_PR=$(gh pr list --state merged --search "$N in:body" \
    --json number --jq '.[0].number // empty')
  if [ "$STATE" = "CLOSED" ] || [ -n "$MERGED_PR" ]; then
    echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] RECONCILE_SKIP: #$N already delivered (state=$STATE${MERGED_PR:+, PR #$MERGED_PR}) — dropping soft claim"
    gh issue edit "$N" --remove-label "status: in-progress" 2>/dev/null || true
    continue
  fi
  DISPATCH_ISSUES="$DISPATCH_ISSUES $N"
done
BATCH_ISSUES=$(echo "$DISPATCH_ISSUES" | xargs)   # only reconciled-live issues proceed
```

---

## STEP 6 — Dispatch expert subagents in parallel

Spawn one Agent per claimed issue. All are run in background (parallel).

For each issue N with expert E, log before spawning:

```bash
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] DISPATCH: #$N → $E — $ISSUE_TITLE"
```

Then spawn:

```
Agent({
  description: "M6 story #N — <issue title>",
  subagent_type: "claude",
  run_in_background: true,
  prompt: """
    You are the <E expert name>. Read the full expert guide first:

    <full content of .claude/commands/experts/<E>.md>

    ---

    ## Isolated worktree — this is your FIRST action, before reading or writing anything
    Run this as your very first Bash call. Every read, edit, build, and commit after this
    happens inside your private tree — branch switches, git add, and lint here are invisible
    to sibling agents and theirs are invisible to you.

    ```bash
    REPO=$(git rev-parse --show-toplevel)   # remember the main checkout
    WT=/tmp/zynax-orch-<RUN_ID>-<N>
    git -C "$REPO" worktree remove "$WT" --force 2>/dev/null || true
    rm -rf "$WT" 2>/dev/null || true
    git -C "$REPO" fetch origin --prune
    git -C "$REPO" worktree add "$WT" origin/main
    cd "$WT"
    ```

    Never cd out of "$WT" until cleanup. Do NOT run `git checkout <branch>` defensively, do
    NOT verify the branch before each Bash call, do NOT avoid `git add .`, do NOT avoid
    `git stash` — none of that is needed: this tree is yours alone.

    ---

    Your task: implement M6 story issue #N end-to-end.

    ## Issue details
    <full issue body from gh issue view N>

    ## Context files to read (read these before writing any code)
    <list of 2-3 specific files named in the issue body or canvas O-step>

    ## Delivery contract
    1. Check if issue is still OPEN and not already in-progress by another session.
       If already claimed: remove your worktree (cleanup below), stop, and report.
    2. From inside "$WT", claim with the DETERMINISTIC KEY before any code: the branch
       ref is `<type>/<N>` — a pure function of the issue number, NO slug. Push it empty
       (atomic hard claim). This is the same key /m6-issue-generate derives, so it is the
       sole mutex: if a sibling already pushed `<type>/<N>` your push is rejected → stop,
       run cleanup, report "claim lost". Apply any human-readable slug only AFTER the push
       wins (rename + push the slugged ref); never let a slug into the claim push.
    3. Implement, run all local gates, commit (DCO + Assisted-by), open PR.
    4. Wait for CI. Report result.
    5. Cleanup — your LAST action, always, success or failure:
       ```bash
       cd "$REPO"
       git worktree remove "$WT" --force 2>/dev/null || true
       rm -rf "$WT" 2>/dev/null || true
       ```
    6. End your response with the ## Session Learnings block (required, template below).

    ## Result format (required — orchestrator parses these for post-merge dispatch)
    ```
    ## Result
    - Issue: #NNN
    - PR: #NNN
    - Merge SHA: <full sha of squash merge commit on main, or "not merged">
    - CI: green / red / pending
    - Affected services: <comma-separated list, e.g. "memory-service,event-bus" or "none">
    ```

    ## Session Learnings (required — emit verbatim in this shape so /m6-learn can parse it)
    ```
    ## Session Learnings
    - domain: <go-services|ci-release|infra-helm|python-adapters|bdd-contract|spdd-canvas>
    - issue: #NNN
    - date: YYYY-MM-DD

    ### Effective patterns
    - <pattern>: <why it worked>

    ### Edge cases discovered
    - <what>: <resolution>

    ### Failed approaches
    - <what>: <why it failed>

    ### Proposed expert prompt update
    - Rule: <exact text>
      Category: domain | structural-workaround
      Reason: <why permanent — for structural-workaround, name the shared-tree problem it works around>
    ```
    Mark Category `structural-workaround` for any rule that only exists to survive a shared
    working tree (branch resets, git add pollution, stash hazards, ref locks, cherry-pick
    rescues). Mark `domain` for genuine engineering knowledge (API shapes, query planner,
    proto field names, test patterns).

    ## Constraints
    - Context budget: stay under 12K tokens. Read only files named above.
    - Never read files outside the issue scope.
    - Use GOWORK=off for all go commands inside service dirs (a worktree is a normal checkout).
    - Commit format: <type>(<scope>): <subject> ≤72 chars, -s flag, Assisted-by trailer.
  """
})
```

---

## STEP 7 — Collect results (wait for all background agents)

As each agent completes, extract:
1. Issue number + PR URL
2. CI status (green / red / pending)
3. `## Session Learnings` block
4. `## Result` block — especially `Merge SHA` and `Affected services`

### Completion-time merge-SHA dedupe (idempotency layer 3 — authoritative)

This is the **authoritative** idempotency check, not layers 1–2. The deterministic claim key
(layer 1) and pre-spawn reconcile (layer 2) are best-effort early-outs that a stale snapshot or
GitHub API lag can slip past; this layer operates on a **merge fact** (a SHA on `origin/main`),
which cannot be wrong. Keep two in-context sets and consult them on every completion **before**
queuing the PR's STEP 7.5 post-merge verifier:

```bash
# Initialize once, before collecting any results:
SEEN_ISSUES=""        # issue numbers already delivered by a merged PR this session
SEEN_MERGE_SHAS=""    # squash-merge SHAs already handed to a post-merge verifier
```

On each agent completion reporting a merged PR (`Merge SHA` present), reconcile against live state:

```bash
# Inputs from the agent's ## Result block: ISSUE_N, PR_N, MERGE_SHA
# Re-query the authoritative merged PR for this issue (may differ from the one this agent opened).
WINNER_PR=$(gh pr list --state merged --search "$ISSUE_N in:body" \
  --json number,mergeCommit --jq 'sort_by(.number) | .[0]')
WINNER_PR_N=$(echo "$WINNER_PR" | jq -r '.number // empty')
WINNER_SHA=$(echo "$WINNER_PR" | jq -r '.mergeCommit.oid // empty')

# Dedupe by issue: a DIFFERENT PR already delivered this issue → this agent's PR is the loser.
if echo " $SEEN_ISSUES " | grep -q " $ISSUE_N " \
   || { [ -n "$WINNER_PR_N" ] && [ "$WINNER_PR_N" != "$PR_N" ]; }; then
  echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] DEDUPE: #$ISSUE_N already delivered by PR #$WINNER_PR_N — closing loser PR #$PR_N, skipping its post-merge dispatch"
  # Close the redundant loser PR if it is still open (idempotent; never touches the winner or main).
  if [ "$(gh pr view "$PR_N" --json state --jq .state 2>/dev/null)" = "OPEN" ]; then
    gh pr close "$PR_N" --delete-branch \
      --comment "Superseded by PR #$WINNER_PR_N, which already delivered #$ISSUE_N. Closing the redundant duplicate (idempotent dispatch, layer 3)." 2>/dev/null || true
  fi
  continue   # do NOT queue a post-merge verifier for a loser PR
fi

# Dedupe by merge SHA: never hand the same merge commit to two verifiers.
if echo " $SEEN_MERGE_SHAS " | grep -q " $MERGE_SHA "; then
  echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] DEDUPE: merge SHA $MERGE_SHA already verified — skipping duplicate post-merge dispatch"
  continue
fi

# First time we see this issue + SHA — record it; STEP 7.5 will dispatch exactly one verifier.
SEEN_ISSUES="$SEEN_ISSUES $ISSUE_N"
SEEN_MERGE_SHAS="$SEEN_MERGE_SHAS $MERGE_SHA"
```

Emit a log line as each result arrives. Extract context stats from the agent's Session Learnings
block (look for `ctx_peak` and `compressions` fields if the expert emitted them):

```bash
# On success:
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] DONE:  #$N ($E) — PR #$PR_N CI:$CI_STATUS  [agent ctx: ~${AGENT_CTX_PEAK}K peak | compress=${AGENT_COMPRESSIONS} | msgs=${AGENT_MSGS}]"
# On failure:
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] FAIL:  #$N ($E) — $FAIL_REASON  [agent ctx: ~${AGENT_CTX_PEAK}K peak | compress=${AGENT_COMPRESSIONS}]"
```

Parse `AGENT_CTX_PEAK`, `AGENT_COMPRESSIONS`, `AGENT_MSGS` from the last `[ctx: ...]` line in the
agent's output (fall back to `?` if the agent didn't emit ctx stats).

For any agent that reported CI failure: report to user with the failing check name.
For any agent with `compress >= 1` in its result: flag it — that expert may need splitting next time.
Do not retry automatically — human intervention required for CI failures.

### Leftover worktree sweep (crashed-agent cleanup)

A subagent removes its own `/tmp/zynax-orch-<RUN_ID>-<N>` last. If it crashed, the path leaks.
After collecting all results, reclaim only **this run's** leftovers — never glob-all, which would
delete a concurrent orchestrator run's live trees:

```bash
for WT in /tmp/zynax-orch-${ORCH_RUN_ID}-* /tmp/zynax-postmerge-${ORCH_RUN_ID}-*; do
  [ -d "$WT" ] || continue
  echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] WT_SWEEP: reclaiming leaked worktree $WT"
  git -C "$COORD_WT" worktree remove "$WT" --force 2>/dev/null || true
  rm -rf "$WT" 2>/dev/null || true
done
git -C "$COORD_WT" worktree prune

# Stale trees from PRIOR crashed runs (age-based; never glob-all live runs):
find /tmp -maxdepth 1 \( -name 'zynax-orch-*' -o -name 'zynax-postmerge-*' \) -mmin +180 \
  -exec rm -rf {} + 2>/dev/null || true
```

---

## STEP 7.5 — Post-merge verification (dispatch one post-mrg agent per merged PR)

For every **deduped** merged PR — i.e. each merge SHA recorded in `SEEN_MERGE_SHAS` by STEP 7's
layer-3 check, never a loser PR that was closed there — dispatch a `post-merge` expert subagent
**in background**. Exactly one verifier per merge SHA. Run all post-merge agents in parallel.

```bash
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] POST_MERGE: dispatching verifiers for merged PRs"
```

For each merged PR N with merge SHA S and affected services A, log before spawning:

```bash
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] POST_MERGE_DISPATCH: PR #$PR_N (merge=$S affected=$A)"
```

Then spawn:

```
Agent({
  description: "Post-merge verify PR #PR_N (issue #N)",
  subagent_type: "claude",
  run_in_background: true,
  prompt: """
    You are the Post-Merge Verifier. Read the full expert guide first:

    <full content of .claude/commands/experts/post-merge.md>

    ---

    ## Isolated worktree — this is your FIRST action
    You are mostly read-only (GitHub + GHCR APIs) but may push a digest-pin commit. Work
    entirely inside your own tree so any branch you create cannot stomp a sibling's checkout.

    ```bash
    REPO=$(git rev-parse --show-toplevel)
    WT=/tmp/zynax-postmerge-<RUN_ID>-<PR_N>
    git -C "$REPO" worktree remove "$WT" --force 2>/dev/null || true
    rm -rf "$WT" 2>/dev/null || true
    git -C "$REPO" fetch origin --prune
    git -C "$REPO" worktree add "$WT" origin/main
    cd "$WT"
    ```

    ---

    Your task: verify post-merge CI, artifacts, and digest pins for:

    PR_NUMBER:    <PR_N>
    MERGE_SHA:    <S>
    ISSUE_NUMBER: <issue N>
    SESSION_DATE: <date>

    ## Delivery contract
    1. Identify affected services from PR file changes (gh pr view PR_N --json files).
    2. Find and wait for post-merge workflow runs (release.yml, tools-image.yml) — max 20 min.
    3. Verify GHCR images for services in the release.yml matrix.
    4. Update digest pins in docker-compose.services.yml if stale.
    5. Update images/images.yaml if ci-runner or a base image was rebuilt; run make sync-images.
    6. Find all open "bump <image> digest" issues; close stale duplicates; implement newest.
    7. Commit all digest updates as a single chore(ci) PR; squash-merge.
    8. Output the full ## Post-Merge Evidence block.
    9. Cleanup — your LAST action, always:
       ```bash
       cd "$REPO"
       git worktree remove "$WT" --force 2>/dev/null || true
       rm -rf "$WT" 2>/dev/null || true
       ```
    10. End with ## Session Learnings.

    ## Constraints
    - Context budget: stay under 20K tokens.
    - Your tree is private: no defensive `git checkout` before Bash calls is needed.
    - Never add service images to images/images.yaml — only base images belong there.
    - `gh pr merge --squash` only.
    - If no images were built and no digest issues are open: emit SKIP with evidence, run cleanup, exit.
  """
})
```

Collect post-merge agent results the same way as domain agents (wait for completion).

Emit on each post-merge agent completing:

```bash
# Success (with updates):
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] POST_MERGE_DONE: PR #$PR_N — digest-PR:#$D_PR workflows:$W_CONCLUSION  [ctx: ~${PMG_CTX_INIT}K→~${PMG_CTX_FINAL}K | compress=${PMG_COMPRESSIONS} | msgs=${PMG_MSGS}]"
# Skip (no images, no open bump issues):
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] POST_MERGE_SKIP: PR #$PR_N — $SKIP_REASON"
# Failure:
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] POST_MERGE_FAIL: PR #$PR_N — $FAIL_REASON"
```

Note: both `ctx_initial` and `ctx_final` are reported (evidence of growth across GHCR/workflow API calls).

---

## STEP 8 — Persist learnings

For each completed `## Session Learnings` block (domain experts + post-merge experts), append
the relevant entries to the appropriate `docs/ai-learnings/<domain>.md` file.
Post-merge learnings go to `docs/ai-learnings/ci-release.md`:

```bash
# Example: append go-services learnings
cat >> docs/ai-learnings/go-services.md << 'EOF'

## Session — <date> (issue #N)
<learnings block content>
EOF
```

Open a `docs:` PR for the learnings update if any new entries were added. This runs in the
coordinator worktree `$COORD_WT` from STEP 1 (branch off `origin/main`, never local `main`):

```bash
LEARN_BRANCH="docs/ai-learnings-$(date +%Y%m%d%H%M)"
git checkout -B "$LEARN_BRANCH" origin/main
git add docs/ai-learnings/
git commit -s -m "docs(ai-learnings): append session learnings — issues $BATCH_ISSUES

$(date +%Y-%m-%d) batch: $BATCH_SIZE issues.

Assisted-by: Claude/claude-sonnet-4-6"
git push -u origin "$LEARN_BRANCH"
LEARN_PR=$(gh pr create --title "docs(ai-learnings): append session learnings — $(date +%Y-%m-%d)" \
  --body "Appending learnings from batch: issues $BATCH_ISSUES" \
  --label "type: docs,milestone: M6" | tail -1)
gh pr merge "$LEARN_PR" --squash --auto
```

---

## STEP 9 — Session report

```
=== Orchestrator Session — <date> ===
Batch size: N issues

### Domain Delivery

| Issue | Expert | PR | CI | Status |
|---|---|---|---|---|
| #NNN | go-services | #NNN | green | MERGED |
| #NNN | ci-release  | #NNN | pending | PR open |
| #NNN | infra-helm  | #NNN | red | BLOCKED |

### Post-Merge Verification

| PR | Workflows | Images verified | Digest pins updated | Bump issues | Digest PR | ctx initial→final |
|---|---|---|---|---|---|---|
| #NNN | release.yml: success | api-gateway ✅ | docker-compose.services.yml ✅ | #912,#917 closed; #931 → PR #NNN | #NNN merged | ~10K→~18K |
| #NNN | none (docs-only) | — | — | — | — | SKIP |

Learnings: appended to docs/ai-learnings/ — PR #NNN opened for review.
Next: run /m6-plan to see the next available batch.
```

After the report, remove the coordinator worktree (the per-run agent/post-merge trees were
already swept in STEP 7):

```bash
cd /tmp   # leave the worktree before removing it
git -C "$REPO" worktree remove "$COORD_WT" --force 2>/dev/null || true
rm -rf "$COORD_WT" 2>/dev/null || true
git -C "$REPO" worktree prune
```

---

## Context budget — enforced invariants

| What orchestrator reads | What it NEVER reads |
|---|---|
| `state/current-milestone.md` | Any `services/*/` Go files |
| `docs/milestones/M6-planning.md` | Any canvas body |
| GitHub issue list (JSON) | Any test output |
| Open PR list (JSON) | Any workflow file contents |
| Remote branch list | Any proto definitions |

Post-merge subagents read only GitHub API + GHCR API + the two digest-pin files
(`infra/docker-compose/docker-compose.services.yml` and `images/images.yaml`).

If you find yourself reading a code file in the orchestrator context: stop. Spawn an expert
subagent instead.

---

## Worktree isolation invariants

Every git working tree in a session is private and run-scoped — there is no shared mutable
tree, so cross-agent branch/staging/commit corruption is structurally impossible.

| Tree | Path | Owner | Lifecycle |
|------|------|-------|-----------|
| Coordinator | `/tmp/zynax-orch-coord-<RUN_ID>` | orchestrator (STEP 2 merge pass, STEP 8 learnings PR) | created STEP 1, removed after STEP 9 |
| Domain agent | `/tmp/zynax-orch-<RUN_ID>-<N>` | one expert subagent | created first / removed last by the agent; swept in STEP 7 if it crashed |
| Post-merge | `/tmp/zynax-postmerge-<RUN_ID>-<PR_N>` | one post-merge subagent | created first / removed last by the agent; swept in STEP 7 if it crashed |

- **Run-scoped, never bare-issue:** paths carry `<RUN_ID>` so the STEP 7 sweep and the
  coordinator can only ever reclaim *this* run's trees. Globbing `/tmp/zynax-orch-*` is forbidden.
- **User's checkout is never mutated:** the orchestrator does all its own git work in the
  coordinator worktree; `main` stays checked out (untouched) in the user's primary worktree.
- Distinct from `/m6-issue-generate`'s `/tmp/zynax-auto-<N>` — no namespace collision.

---

## Idempotent dispatch invariants

The same issue must never ship two pull requests, even under stale snapshots and GitHub API
lag. Three layers enforce this as **defense-in-depth** — not one check, and the cheap layers do
not replace the authoritative one:

| Layer | Where | Mechanism | Strength |
|-------|-------|-----------|----------|
| 1 — Deterministic claim key | STEP 6 dispatch prompt (+ `/m6-issue-generate` STEP 5) | Branch ref `<type>/<N>` is a pure function of the issue number; the atomic empty-branch push is the **sole mutex** shared by both entry points. Slug applied only post-claim. | Prevents two live branches for one issue. |
| 2 — Pre-spawn reconcile | STEP 5 | Re-query `gh issue view` + merged-PR search before spawning; skip + drop soft claim if already delivered. | Cheap early-out; can be defeated by API lag. |
| 3 — Completion-time merge-SHA dedupe | STEP 7 | `SEEN_ISSUES` / `SEEN_MERGE_SHAS`; on each completion re-query the authoritative merged PR; close the loser PR and skip its verifier when a different PR already delivered the issue. | **Authoritative** — operates on a merge fact (a SHA on `main`), immune to API lag. |

- **Completion-time is authoritative.** Layers 1–2 reduce the race window; only layer 3 acts on
  a fact that cannot be wrong. Never treat the claim key or the pre-spawn reconcile as sufficient
  on its own.
- **The claim key is the single mutex across both entry points.** `/m6-orchestrate` and
  `/m6-issue-generate` derive the identical `<type>/<N>` ref, so a race between them collides on
  one push. A slug must never enter the claim push.
- **A loser PR is closed, never merged.** When two PRs target one issue, layer 3 keeps the first
  merged (the winner) and closes the redundant one; `gh pr merge` flags and the push-to-main
  policy (ADR-023) are unchanged.

---

## Orchestrator context tracking

The orchestrator's own context is intentionally tiny (planning state only, no code).
Track it with the same format as experts:

```bash
# Emit at the start and after each major step:
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] <PHASE>: <desc>  [ctx: ~<X>K | agents:<A>/<B> running]"
```

Heuristics:
- After STEP 1 (reading 2 files + GitHub JSON): **~15K**
- Each expert subagent result added: **+2–5K**
- Each post-merge subagent result added: **+3–6K**
- Expected peak for a 3-agent batch + 3 post-merge verifiers: **~40–60K**

### Orchestrator split thresholds

| Condition | Action |
|-----------|--------|
| `CTX_TOKENS > 60K` | Stop claiming new issues — only collect existing agent results |
| `CTX_TOKENS > 100K` OR `CTX_COMPRESSIONS >= 1` | **STOP. Report collected results so far.** Let the human run `/m6-plan` for the next batch. |
