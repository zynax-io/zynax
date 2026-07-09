---
description: "Parallel milestone orchestrator — reads state, claims up to 3 issues per batch, routes each to the matching model-routed agent under .claude/agents/, runs them in parallel, collects results and learnings. Orchestrator never reads code files directly."
argument-hint: "[--batch-size N]  default: 3"
---

# /lib:deliver-batch — Parallel delivery orchestrator (building block of /deliver)

> **Building block** — invoked by `/deliver` (no-arg or `--batch N`), not run directly.\n> **Scope contract:** the caller provides scope — repo-wide `status: ready` work by default, or a\n> `--milestone M` filter. Where this says "active milestone", read "the caller-provided scope".\n

Thin coordination layer: read state → claim issues → fan out to model-routed subagents in
parallel → collect results → persist learnings → report.

> **Context budget discipline.** The orchestrator reads planning state only — never code files,
> never canvas body, never test output, never expert-guide contents. Everything an agent needs
> lives in its own definition (`.claude/agents/<name>.md` — model, effort, tools) plus two files
> it reads itself at startup: `docs/patterns/delivery-agent-protocol.md` and its domain guide
> under `.claude/commands/experts/`. Dispatch prompts stay under ~15 lines.

> **Rules are not restated here.** See `AGENTS.md`, `CLAUDE.md`, the shared protocol
> (`docs/patterns/delivery-agent-protocol.md`), and the expert guides for domain rules. This
> file is the coordination loop only.

---

## STEP 0 — Verify the agent roster exists

```bash
ls .claude/agents/
# go-services.md | python-adapters.md | bdd-contract.md | infra-helm.md |
# ci-release.md  | spdd-canvas.md     | post-merge.md
```

Do not read their contents — the harness loads them at dispatch. Just verify they exist; if the
directory is missing, stop and report (the model-routing PR not yet merged, or the session
started before `.claude/agents/` existed — restart the session once).

---

## STEP 1 — Read planning state

```bash
# Per-invocation run id — namespaces every worktree and the crash-recovery sweep (STEP 7).
# Two concurrent orchestrator runs get distinct ids, so neither can ever touch the other's trees.
ORCH_RUN_ID="$(date +%s)-$$"
export ORCH_RUN_ID

# Coordinator worktree — the orchestrator's OWN git operations (this sync, the STEP 2 merge
# pass, the STEP 8 learnings PR) run here, never in the user's primary checkout.
REPO=$(git rev-parse --show-toplevel)
COORD_WT="/tmp/zynax-orch-coord-${ORCH_RUN_ID}"
git -C "$REPO" worktree remove "$COORD_WT" --force 2>/dev/null || true
rm -rf "$COORD_WT" 2>/dev/null || true
git -C "$REPO" fetch origin --prune
git -C "$REPO" worktree add "$COORD_WT" origin/main   # detached at origin/main
cd "$COORD_WT"

# Active-milestone config (SSoT: state/milestone.yaml) — single helper call, never inline awk.
eval "$(bash automation/milestone-env.sh)"
# → MILESTONE_NAME MILESTONE_TITLE MILESTONE_NUMBER MILESTONE_VERSION
#   PLANNING_DOC MILESTONE_LABEL GH_MILESTONE

# Read only these two files — nothing else (from the coordinator worktree)
cat state/current-milestone.md           # blockers, active work
cat "$PLANNING_DOC"                      # EPIC status + dependency table
```

```bash
# Snapshot GitHub state
BATCH_SIZE=${ARGUMENTS:-3}               # default 3 parallel issues

OPEN=$(gh issue list \
  --milestone "$GH_MILESTONE" \
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

Before claiming new work, arm the merge queue on any open PRs that are already green
(ADR-047). `BEHIND` is fine — the queue validates each PR against current main on its turn;
never rebase for freshness (a force-push ejects a queued PR). Only `DIRTY` (real conflicts)
needs a manual `--signoff` rebase. *Fallback (no merge-queue rule on main — pre-cutover or
rollback): rebase `origin/main` + `--force-with-lease` in `$COORD_WT` before arming, as
before.*

```bash
OPEN_PRS_JSON=$(gh pr list --author "@me" --state open \
  --json number,mergeStateStatus,statusCheckRollup \
  --jq 'sort_by(.number) | .[]')

while IFS= read -r PR; do
  PR_N=$(echo "$PR" | jq -r .number)
  MERGE_STATE=$(echo "$PR" | jq -r .mergeStateStatus)
  FAILED=$(echo "$PR" | jq '[.statusCheckRollup[]? | select(.isRequired==true) | .conclusion] | any(. == "FAILURE" or . == "ERROR")')
  if [[ "$MERGE_STATE" == "CLEAN" || "$MERGE_STATE" == "BEHIND" ]] && [[ "$FAILED" == "false" ]]; then
    gh pr merge "$PR_N" --squash --auto   # enqueue; the queue merges it on its turn
  fi
done <<< "$OPEN_PRS_JSON"
```

---

## STEP 3 — Select READY batch

Using the same classification logic as `/deliver`:

```bash
# For each open issue: classify as READY / IN_PROGRESS / BLOCKED
# Priority order from the EPIC table in "$PLANNING_DOC"
# Select top $BATCH_SIZE READY issues for this session

# Quick filter:
# 1. Has "status: in-progress" label → IN_PROGRESS (skip)
# 2. Has remote branch matching the claim key <type>/<N> (bare) OR a slugged
#    <type>/<N>-* variant → IN_PROGRESS (skip). Match with `^<type>/<N>(-|$)`.
# 3. Has "Pending #X" or "Depends on #X" in body where X is open → BLOCKED (skip)
# 4. Otherwise → READY
```

Report the selected batch and their agent routing before dispatching:

```bash
ISSUES_LIST=$(echo "$BATCH_ISSUES" | tr ' ' ',' | sed 's/^/#/;s/,/,#/g')

echo ""
echo "=== [orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] BATCH SELECTED — $BATCH_SIZE issues ==="
# For each issue in the batch, print: #N → <agent>  <title>
#   #823 → go-services      feat(event-bus): service scaffold
#   #865 → ci-release       ci(infra): OCI manifest annotations
echo "==="
```

---

## STEP 4 — Claim all batch issues (soft claim)

```bash
gh label create "status: in-progress" --color "FBCA04" \
  --description "Actively being implemented" 2>/dev/null || true

for N in $BATCH_ISSUES; do
  gh issue edit "$N" --add-label "status: in-progress" --add-assignee "@me"
done
```

---

## STEP 5 — Route issues to agents

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

Multi-agent issues: run `spdd-canvas` **synchronously first** (`run_in_background: false` — it
must produce an Aligned canvas before implementation can start), then dispatch the domain agent.

After applying the routing table, emit one log line per issue:

```bash
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] ROUTE: #$N → $AGENT  ($ISSUE_TITLE)"
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

## STEP 6 — Dispatch agents in parallel

Spawn one Agent per claimed issue, all in background (parallel). The agent's definition
supplies its model, effort, tools, and instructions to read the shared protocol
(`docs/patterns/delivery-agent-protocol.md`) and its domain guide — the dispatch prompt
carries **only the per-issue facts**:

```bash
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] DISPATCH: #$N → $AGENT — $ISSUE_TITLE"
```

```
Agent({
  description: "Story #N — <issue title>",
  subagent_type: "<agent from STEP 5, e.g. go-services>",
  run_in_background: true,
  prompt: """
    ISSUE: #N — <issue title>
    REPO:  <REPO>
    WT:    /tmp/zynax-orch-<ORCH_RUN_ID>-<N>     (your literal private worktree path)

    Issue body:
    <full issue body from gh issue view N>

    Context files (read these before writing any code):
    <2-3 specific repo paths named in the issue body or canvas O-step>

    Deliver this story end-to-end per your agent definition: read
    docs/patterns/delivery-agent-protocol.md and your expert guide first, then
    claim → implement → gates → PR → CI → queue merge → cleanup. End with the
    ## Result and ## Session Learnings blocks (the orchestrator parses both).
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

Emit a log line as each result arrives:

```bash
# On success:
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] DONE:  #$N ($AGENT) — PR #$PR_N CI:$CI_STATUS"
# On failure:
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] FAIL:  #$N ($AGENT) — $FAIL_REASON"
```

For any agent that reported CI failure: report to user with the failing check name.
For any agent that reported a mid-run context split: flag it — the issue may need slicing.
Do not retry automatically — human intervention required for CI failures.

### Crashed-agent delivery recovery (finalize BEFORE sweeping)

An agent that crashes mid-delivery (transient API 5xx/529, OOM, classifier outage) leaves its work
in its worktree and may already have pushed the claim branch — **do not sweep it blindly or the
work is lost.** For every agent whose result is missing or an API-error (no `## Result` block) AND
whose claim branch `<type>/<N>` exists on origin, **recover the delivery from the coordinator**
before the sweep below:

```bash
# For each crashed/stalled agent N (literal worktree path):
WT="/tmp/zynax-orch-${ORCH_RUN_ID}-${N}"
# The claim may be the BARE key or the post-slug ref — protocol §4 deletes the bare
# key after the slug rename, so match both (same boundary as STEP 3's filter).
CLAIM_REF=$(git ls-remote origin 'refs/heads/*' | awk '{print $2}' \
  | sed 's|refs/heads/||' | grep -E "^<type>/${N}(-|$)" | head -1)
[ -n "$CLAIM_REF" ] || continue   # no claim (bare or slugged) → nothing to recover
PR=$(gh pr list --head "$CLAIM_REF" --state open --json number --jq '.[0].number // empty')
if [ -n "$PR" ]; then
  # PR already open — just finish it: gh pr checks "$PR" --watch --interval 30, then squash-merge.
  echo "RECOVER: #$N has open PR #$PR — finishing"; continue
fi
# No PR yet → inspect the worktree and finish the delivery:
git -C "$WT" log origin/main..HEAD --oneline    # committed-but-unpushed work?
git -C "$WT" status --short                      # uncommitted work?
#   uncommitted → review the diff, run the local gates, then `git -C "$WT" commit -s -F <msgfile>`
#   committed   → already done; just push
git -C "$WT" push -u origin "HEAD:refs/heads/${CLAIM_REF}"   # fast-forward onto the surviving
#   claim ref (bare or slugged); if rejected because the ref on origin advanced → rebase onto
#   the remote branch + push --force-with-lease (claim-commit recovery; main-freshness is the
#   merge queue's job — ADR-047, see the shared protocol §6).
# Then open the PR (canonical template body), wait for CI, squash-merge. NOW the tree is safe to sweep.
```

Recovery is the orchestrator's job, not the dead agent's: validate the recovered diff against the
issue's acceptance criteria yourself — CI is the safety net (a broken/incomplete diff red-walls and
you fix it or re-dispatch). Only worktrees with **no** recoverable work (no claim branch, empty tree)
fall through to the sweep below.

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

## STEP 7.5 — Post-merge verification (one `post-merge` agent per merged PR)

For every **deduped** merged PR — i.e. each merge SHA recorded in `SEEN_MERGE_SHAS` by STEP 7's
layer-3 check, never a loser PR that was closed there — dispatch a `post-merge` agent
**in background** (its definition pins Haiku: mechanical GitHub/GHCR verification only).
Exactly one verifier per merge SHA. Run all post-merge agents in parallel.

```bash
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] POST_MERGE_DISPATCH: PR #$PR_N (merge=$S affected=$A)"
```

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
    WT:           /tmp/zynax-postmerge-<ORCH_RUN_ID>-<PR_N>   (your literal private worktree path)

    Verify post-merge CI, GHCR artifacts, and digest pins for this merge per your
    agent definition (read docs/patterns/delivery-agent-protocol.md and your expert
    guide first). Back-fill the originating PR's "Post-merge digest sync → main"
    Evidence placeholder. End with the ## Post-Merge Evidence block and
    ## Session Learnings.
  """
})
```

Collect post-merge agent results the same way as domain agents (wait for completion).

```bash
# Success (with updates):
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] POST_MERGE_DONE: PR #$PR_N — digest-PR:#$D_PR workflows:$W_CONCLUSION"
# Skip (no images, no open bump issues):
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] POST_MERGE_SKIP: PR #$PR_N — $SKIP_REASON"
# Failure:
echo "[orchestrator issues:${ISSUES_LIST} $(date +%H:%M:%S)] POST_MERGE_FAIL: PR #$PR_N — $FAIL_REASON"
```

---

## STEP 8 — Persist learnings

For each completed `## Session Learnings` block (domain agents + post-merge agents), append
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

Assisted-by: Claude/<model-id-of-this-session>"
git push -u origin "$LEARN_BRANCH"
LEARN_PR=$(gh pr create --title "docs(ai-learnings): append session learnings — $(date +%Y-%m-%d)" \
  --body "Appending learnings from batch: issues $BATCH_ISSUES" \
  --label "type: docs" --label "$MILESTONE_LABEL" | tail -1)
gh pr merge "$LEARN_PR" --squash --auto
```

---

## STEP 9 — Session report

```
=== Orchestrator Session — <date> ===
Batch size: N issues

### Domain Delivery

| Issue | Agent | PR | CI | Status |
|---|---|---|---|---|
| #NNN | go-services | #NNN | green | MERGED |
| #NNN | ci-release  | #NNN | pending | PR open |
| #NNN | infra-helm  | #NNN | red | BLOCKED |

### Post-Merge Verification

| PR | Workflows | Images verified | Digest pins updated | Bump issues | Digest PR |
|---|---|---|---|---|---|
| #NNN | release.yml: success | api-gateway ✅ | docker-compose.services.yml ✅ | #912,#917 closed; #931 → PR #NNN | #NNN merged |
| #NNN | none (docs-only) | — | — | — | SKIP |

Learnings: appended to docs/ai-learnings/ — PR #NNN opened for review.
Next: run /deliver to see the next available batch.
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
| `"$PLANNING_DOC"` | Any canvas body |
| GitHub issue list (JSON) | Any test output |
| Open PR list (JSON) | Any workflow file contents |
| Remote branch list | Any expert guide or agent definition contents |

If you find yourself reading a code file in the orchestrator context: stop. That work belongs
inside a dispatched agent.

Proactive guards (the load-bearing dedupe state must outlive the session's context):

- **One batch per session.** Never claim beyond `$BATCH_SIZE`, and never claim additional
  issues after the first agent result has been collected — the next batch gets a fresh session.
- **`SEEN_ISSUES` / `SEEN_MERGE_SHAS` are in-context state.** If a compaction occurred (or may
  have), do NOT trust them: rebuild both from live GitHub before dispatching any further
  post-merge verifier — for each batch issue re-run the layer-3 winner query
  (`gh pr list --state merged --search "<N> in:body"`) and re-mark the seen SHAs.
- If you notice your own context has been compacted mid-run: stop claiming new issues, rebuild
  the seen-sets as above, collect the results of agents already running, report, and let the
  human start the next batch fresh.

---

## Worktree isolation invariants

Every git working tree in a session is private and run-scoped — there is no shared mutable
tree, so cross-agent branch/staging/commit corruption is structurally impossible.

| Tree | Path | Owner | Lifecycle |
|------|------|-------|-----------|
| Coordinator | `/tmp/zynax-orch-coord-<RUN_ID>` | orchestrator (STEP 2 merge pass, STEP 8 learnings PR) | created STEP 1, removed after STEP 9 |
| Domain agent | `/tmp/zynax-orch-<RUN_ID>-<N>` | one delivery subagent | created first / removed last by the agent; swept in STEP 7 if it crashed |
| Post-merge | `/tmp/zynax-postmerge-<RUN_ID>-<PR_N>` | one post-merge subagent | created first / removed last by the agent; swept in STEP 7 if it crashed |

- **Run-scoped, never bare-issue:** paths carry `<RUN_ID>` so the STEP 7 sweep and the
  coordinator can only ever reclaim *this* run's trees. Globbing `/tmp/zynax-orch-*` is forbidden.
- **User's checkout is never mutated:** the orchestrator does all its own git work in the
  coordinator worktree; `main` stays checked out (untouched) in the user's primary worktree.
- Distinct from `/deliver`'s `/tmp/zynax-auto-<N>` — no namespace collision.

---

## Idempotent dispatch invariants

The same issue must never ship two pull requests, even under stale snapshots and GitHub API
lag. Three layers enforce this as **defense-in-depth** — not one check, and the cheap layers do
not replace the authoritative one:

| Layer | Where | Mechanism | Strength |
|-------|-------|-----------|----------|
| 1 — Deterministic claim key | shared protocol §4 (`docs/patterns/delivery-agent-protocol.md`, same key `/deliver` derives) | Branch ref `<type>/<N>` is a pure function of the issue number; the atomic empty-branch push is the **sole mutex** shared by both entry points. Slug applied only post-claim. | Prevents two live branches for one issue. |
| 2 — Pre-spawn reconcile | STEP 5 | Re-query `gh issue view` + merged-PR search before spawning; skip + drop soft claim if already delivered. | Cheap early-out; can be defeated by API lag. |
| 3 — Completion-time merge-SHA dedupe | STEP 7 | `SEEN_ISSUES` / `SEEN_MERGE_SHAS`; on each completion re-query the authoritative merged PR; close the loser PR and skip its verifier when a different PR already delivered the issue. | **Authoritative** — operates on a merge fact (a SHA on `main`), immune to API lag. |

- **Completion-time is authoritative.** Layers 1–2 reduce the race window; only layer 3 acts on
  a fact that cannot be wrong. Never treat the claim key or the pre-spawn reconcile as sufficient
  on its own.
- **The claim key is the single mutex across both entry points.** `/deliver` and this
  orchestrator derive the identical `<type>/<N>` ref, so a race between them collides on
  one push. A slug must never enter the claim push.
- **A loser PR is closed, never merged.** When two PRs target one issue, layer 3 keeps the first
  merged (the winner) and closes the redundant one; `gh pr merge` flags and the push-to-main
  policy (ADR-023) are unchanged.

---

## Model routing (why dispatch is per-agent)

| Agent | Model / effort | Rationale |
|---|---|---|
| `spdd-canvas` | Fable · `high` | Top-tier reasoning: epic decomposition, Tier-2 judgment, ADR fit |
| `go-services` `python-adapters` `bdd-contract` `infra-helm` `ci-release` | Opus · `xhigh` | Scoped implementation of an already-reasoned story — the tuned coding configuration |
| `post-merge` | Haiku | Mechanical GitHub/GHCR verification; escalates judgment calls back to the orchestrator |

The orchestrator itself runs on the session model (top tier for planning-heavy sessions) — it
makes routing decisions, never implementation ones.
