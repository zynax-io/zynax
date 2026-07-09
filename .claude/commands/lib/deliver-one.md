---
description: "Fully autonomous single-story delivery — claims the issue, dispatches the model-routed domain agent (canvas agent first for feat: without an Aligned canvas), waits for CI + queue merge, verifies post-merge artifacts, and marks done. Cross-machine safe."
argument-hint: "<story-or-epic-issue-number>"
---

# /lib:deliver-one — Autonomous single-story delivery (building block of /deliver)

> **Building block** — invoked by `/deliver <#issue>`, not run directly.\n> **Scope contract:** milestone-agnostic; the caller may pass `--milestone M` as a filter.\n

End-to-end, unattended delivery of a single story issue. This command is a **single-issue
dispatcher**: it guards, soft-claims, and routes — the implementation itself always runs inside
the matching model-routed agent from `.claude/agents/` (implementation → Opus `xhigh`;
canvas → Fable `high`; post-merge → Haiku), never in this session's context. That keeps model
routing enforced regardless of which model the driving session runs.

> **Rules are not restated here.** Commit format, DCO + `Assisted-by` trailers, `GOWORK=off`,
> PR-size limits, coverage gates, and the SPDD workflow live in **`AGENTS.md`**, **`CLAUDE.md`**,
> and the shared agent protocol (`docs/patterns/delivery-agent-protocol.md`). This file is the
> *routing loop only*.

> **Canvas auto-alignment policy.** For `feat:` issues with no Aligned canvas, this command
> dispatches the `spdd-canvas` agent first. If its security review PASSes, Status is set to
> `Aligned` and implementation proceeds. If it **FAILs** (Tier 2 findings that cannot be
> resolved inline), stop and report — never proceed from a failed review (ADR-019).

---

## Cross-machine claim protocol (non-negotiable)

Two layers prevent duplicate work across concurrent sessions on any machine:

1. **Soft claim** — this command adds the `status: in-progress` label + self-assigns the issue
   on GitHub (visible immediately to all sessions and to `/deliver`). A *signal*, not a lock.
2. **Hard claim** — the dispatched agent pushes the empty deterministic branch `<type>/<N>`
   before writing any code (shared protocol §4). Only one `git push -u origin <type>/<N>` wins
   when two sessions race; a rejected push means the story is taken → the agent stops and
   reports "claim lost".

Always check both before starting. Never assume an issue is free just because you read it as open.

---

## STEP 0 — Pre-flight

```bash
# Agent roster must exist (model-routing PR merged; restart session once if the
# .claude/agents/ directory was created after this session started).
ls .claude/agents/

# Active-milestone config (SSoT: state/milestone.yaml) — single helper call, never inline awk.
eval "$(bash automation/milestone-env.sh)"
# → MILESTONE_NAME MILESTONE_TITLE MILESTONE_NUMBER MILESTONE_VERSION
#   PLANNING_DOC MILESTONE_LABEL GH_MILESTONE

# Planning state only — no code files in this session's context.
cat state/current-milestone.md           # active blockers, health
cat "$PLANNING_DOC"                      # dependency table, EPIC status
```

---

## STEP 1 — Guard: is this issue already claimed or done?

```bash
ISSUE_N=$ARGUMENTS   # e.g. 793

# Check closed (done)
STATE=$(gh issue view "$ISSUE_N" --json state --jq .state)
[ "$STATE" = "CLOSED" ] && { echo "Issue #$ISSUE_N is already CLOSED — nothing to do."; exit 0; }

# Check soft-claimed (in-progress on another session/machine)
IN_PROGRESS=$(gh issue view "$ISSUE_N" --json labels \
  --jq '[.labels[].name] | any(. == "status: in-progress")')
[ "$IN_PROGRESS" = "true" ] && {
  ASSIGNEE=$(gh issue view "$ISSUE_N" --json assignees --jq '[.assignees[].login] | join(", ")')
  echo "Issue #$ISSUE_N is already claimed (status: in-progress, assignee: $ASSIGNEE)."
  echo "If this is stale (session crashed), remove the label manually and re-run."
  exit 1
}

# Check if a branch for this issue already exists on remote (hard-claimed by another session).
# Match BOTH the bare deterministic claim key `<type>/<N>` and any post-claim slugged
# variant `<type>/<N>-<slug>` — the trailing boundary is end-of-ref OR a hyphen.
git fetch origin --prune
EXISTING_BRANCH=$(git ls-remote origin 'refs/heads/*' | awk '{print $2}' \
  | sed 's|refs/heads/||' | grep -E "^[a-z]+/${ISSUE_N}(-|$)" | head -1)
[ -n "$EXISTING_BRANCH" ] && {
  echo "Branch $EXISTING_BRANCH already exists on remote — story #$ISSUE_N is taken."
  exit 1
}
```

---

## STEP 2 — Claim: add label + self-assign

```bash
# Ensure the label exists (create once, idempotent)
gh label create "status: in-progress" --color "FBCA04" --description "Actively being implemented" \
  --repo "$(gh repo view --json nameWithOwner --jq .nameWithOwner)" 2>/dev/null || true

# Soft-claim: label + assign to self
gh issue edit "$ISSUE_N" --add-label "status: in-progress" --add-assignee "@me"
echo "Claimed issue #$ISSUE_N (label added, self-assigned)."
```

---

## STEP 3 — Read the issue

```bash
ISSUE=$(gh issue view "$ISSUE_N" --json number,title,body,labels,state,milestone)
echo "$ISSUE" | jq .

# Extract commit type from issue title (e.g. "feat(scope): title" → "feat")
ISSUE_TITLE=$(echo "$ISSUE" | jq -r .title)
COMMIT_TYPE=$(echo "$ISSUE_TITLE" | grep -oP '^(feat|fix|refactor|docs|test|ci|chore)' || echo "chore")

# Detect if this is an EPIC (type: epic label)
IS_EPIC=$(echo "$ISSUE" | jq '[.labels[].name] | any(. == "type: epic")')

# Detect if SPDD canvas is required (feat: type)
NEEDS_CANVAS=false
[ "$COMMIT_TYPE" = "feat" ] && NEEDS_CANVAS=true

echo "Issue type: $COMMIT_TYPE | Is EPIC: $IS_EPIC | Needs canvas: $NEEDS_CANVAS"
```

**If this is an EPIC (`IS_EPIC = true`):** resolve the EPIC to a story issue before proceeding.
Go to **STEP 3-EPIC**. Otherwise skip to **STEP 4**.

---

## STEP 3-EPIC — Resolve EPIC to next story issue

```bash
# Find the canvas — read from origin/main (this checkout may be stale; STEP 1 fetched)
CANVAS_DIR=$(git ls-tree --name-only origin/main docs/spdd/ 2>/dev/null \
  | sed 's|docs/spdd/||' | grep -E "^${ISSUE_N}-" | head -1)

# Determine canvas state
if [ -n "$CANVAS_DIR" ]; then
  CANVAS_STATUS=$(git show "origin/main:docs/spdd/$CANVAS_DIR/canvas.md" 2>/dev/null \
    | grep -m1 "^Status:" | awk '{print $2}')
  echo "Canvas found: docs/spdd/$CANVAS_DIR/canvas.md — Status: $CANVAS_STATUS"
else
  CANVAS_STATUS="none"
  echo "No canvas found for EPIC #$ISSUE_N"
fi

# Find the next open story issue for this EPIC (lowest step number, not yet in-progress or done)
STORY_ISSUES=$(gh issue list --milestone "$GH_MILESTONE" --state open \
  --json number,title,body,labels \
  --jq ".[] | select(.body | test(\"#${ISSUE_N}\")) | {n:.number,title:.title,labels:[.labels[].name]}")

# Filter out already in-progress stories
NEXT_STORY=$(echo "$STORY_ISSUES" | jq -r 'select((.labels | any(. == "status: in-progress")) | not) | .n' \
  | sort -n | head -1)

if [ -z "$NEXT_STORY" ]; then
  echo "No available story issues for EPIC #$ISSUE_N (all in-progress or done)."
  gh issue edit "$ISSUE_N" --remove-label "status: in-progress"
  exit 0
fi

echo "Resolved EPIC #$ISSUE_N → story #$NEXT_STORY"
# Re-claim on the story issue, release EPIC claim
gh issue edit "$ISSUE_N" --remove-label "status: in-progress"
gh issue edit "$NEXT_STORY" --add-label "status: in-progress" --add-assignee "@me"

# Re-read story issue details
ISSUE=$(gh issue view "$NEXT_STORY" --json number,title,body,labels,state,milestone)
ISSUE_N=$NEXT_STORY
ISSUE_TITLE=$(echo "$ISSUE" | jq -r .title)
COMMIT_TYPE=$(echo "$ISSUE_TITLE" | grep -oP '^(feat|fix|refactor|docs|test|ci|chore)' || echo "chore")
NEEDS_CANVAS=false
[ "$COMMIT_TYPE" = "feat" ] && NEEDS_CANVAS=true
```

---

## STEP 4 — Canvas gate (feat: only, when canvas not Aligned)

Skip this step if `COMMIT_TYPE != "feat"` or if the canvas is already `Aligned`.

```bash
# Find EPIC number referenced in story body (pattern: "EPIC #NNN" or "parent #NNN")
EPIC_N=$(echo "$ISSUE" | jq -r .body | grep -oP '(?<=#)\d+' | head -1)
[ -z "$EPIC_N" ] && EPIC_N="$ISSUE_N"   # fallback: issue is its own EPIC

# Read canvas state from origin/main — never from this possibly-stale checkout
CANVAS_DIR=$(git ls-tree --name-only origin/main docs/spdd/ 2>/dev/null \
  | sed 's|docs/spdd/||' | grep -E "^${EPIC_N}-" | head -1)
[ -n "$CANVAS_DIR" ] && CANVAS_STATUS=$(git show "origin/main:docs/spdd/$CANVAS_DIR/canvas.md" 2>/dev/null \
  | grep -m1 "^Status:" | awk '{print $2}')
```

If no canvas exists or Status ≠ `Aligned`, dispatch the `spdd-canvas` agent **synchronously**
(foreground — implementation cannot start before it returns Aligned):

```
Agent({
  description: "Canvas for EPIC #EPIC_N",
  subagent_type: "spdd-canvas",
  run_in_background: false,
  prompt: """
    ISSUE: EPIC #EPIC_N — <epic title>   (story being delivered: #ISSUE_N)
    REPO:  <repo root>
    WT:    /tmp/zynax-auto-canvas-<EPIC_N>-<ISSUE_N>     (your literal private worktree path —
           per-story suffix so two sessions on different stories of one EPIC never collide)

    Produce or align the REASONS Canvas for this EPIC per your agent definition:
    analysis → canvas (Status: Draft) → security review. On PASS, set Status: Aligned
    and create story issues if none exist (label them "$MILESTONE_LABEL", milestone
    "$GH_MILESTONE"). On FAIL, stop and report the Tier-2 findings.
    End with ## Result (canvas path + Status) and ## Session Learnings.
  """
})
```

On FAIL: remove the soft claim (`gh issue edit "$ISSUE_N" --remove-label "status: in-progress"`)
and stop — report the findings. On PASS: proceed with `CANVAS_STATUS=Aligned`.

```bash
# Locked decision #1107: /lib:spdd-story is milestone-agnostic — the CALLER injects the
# active milestone label + GitHub milestone on every story of this EPIC (idempotent
# re-assert, also covers stories that pre-existed without labels):
for STORY in $(gh issue list --state open --limit 100 --json number,body \
  --jq ".[] | select(.body | test(\"#${EPIC_N}\")) | .number"); do
  gh issue edit "$STORY" --add-label "$MILESTONE_LABEL" --milestone "$GH_MILESTONE"
done
```

---

## STEP 5 — Route to the domain agent

Routing rules aligned with `/lib:deliver-batch` STEP 5 (first match wins). Two deliberate
differences: the `feat:`-without-Aligned-canvas row lives in STEP 4 here (already handled
before routing), and this table adds a fallback row for unmatched titles:

| Issue title pattern | Agent (`subagent_type`) |
|---|---|
| `(api-gateway)` / `(workflow-compiler)` / `(engine-adapter)` / `(task-broker)` / `(agent-registry)` / `(event-bus)` / `(memory-service)` | `go-services` |
| `(infra)` / `helm` / `k8s` (case-insensitive) | `infra-helm` |
| `(ci)` / `actions` / `workflow` / `images.yaml` | `ci-release` |
| `(agents)` / `(sdk)` / `python` / `adapter` | `python-adapters` |
| `test:` type OR `protos/tests` OR `.feature` in body | `bdd-contract` |
| anything else | `go-services` (closest general implementer) — flag the routing gap in the report |

```bash
echo "ROUTE: #$ISSUE_N → $AGENT  ($ISSUE_TITLE)"
```

---

## STEP 6 — Dispatch the domain agent (foreground — this command blocks until done)

Unlike `/lib:deliver-batch`, this command waits: its contract is end-to-end autonomous
delivery, not fire-and-forget. The agent performs the **hard claim** (deterministic key
`<type>/<N>`, shared protocol §4), implements, reconciles all status surfaces in the same diff
(protocol §5), runs gates + runtime smoke, opens the PR from the canonical template, waits for
CI in the foreground, arms the queue merge, and cleans up its worktree.

```
Agent({
  description: "Story #ISSUE_N — <issue title>",
  subagent_type: "<agent from STEP 5>",
  run_in_background: false,
  prompt: """
    ISSUE: #ISSUE_N — <issue title>
    REPO:  <repo root>
    WT:    /tmp/zynax-auto-<ISSUE_N>     (your literal private worktree path)
    CANVAS: docs/spdd/<EPIC_N>-<slug>/canvas.md — Status: Aligned   (feat: only; use
            /lib:spdd-generate semantics — implement the single O-step this story covers)

    Issue body:
    <full issue body>

    Context files (read these before writing any code):
    <2-3 specific repo paths named in the issue body or canvas O-step>

    Deliver this story end-to-end per your agent definition: read
    docs/patterns/delivery-agent-protocol.md and your expert guide first, then
    claim → implement → gates → PR → CI → queue merge → cleanup. End with the
    ## Result and ## Session Learnings blocks.
  """
})
```

If the agent reports **"claim lost"**: another session won the race — remove the soft claim and
exit 0 (not an error). If it reports a red gate or CI failure it could not fix in scope: leave
the hard claim branch for inspection, remove the soft claim, and report the failing check.

---

## STEP 7 — Verify issue closed + EPIC completion

```bash
# Parse the agent's ## Result block: PR_N, MERGE_SHA, CI, AFFECTED_SERVICES
sleep 5   # allow GitHub to process Closes #N from the squash-merge commit

ISSUE_STATE=$(gh issue view "$ISSUE_N" --json state --jq .state)
if [ "$ISSUE_STATE" != "CLOSED" ]; then
  gh issue close "$ISSUE_N" --reason completed \
    --comment "Closed by squash-merge of PR #$PR_N. All acceptance criteria met."
fi
echo "Issue #$ISSUE_N is CLOSED."

# Remove soft claim
gh issue edit "$ISSUE_N" --remove-label "status: in-progress" 2>/dev/null || true

# EPIC completion check: if all stories for the parent EPIC are now closed, close the EPIC too
if [ -n "$EPIC_N" ] && [ "$EPIC_N" != "$ISSUE_N" ]; then
  OPEN_STORIES=$(gh issue list --milestone "$GH_MILESTONE" --state open \
    --json body --jq "[.[] | select(.body | test(\"#${EPIC_N}\"))] | length")
  if [ "$OPEN_STORIES" -eq 0 ]; then
    gh issue close "$EPIC_N" --reason completed \
      --comment "All O-steps merged. Canvas status: Implemented."
    # Flip the canvas Status: Aligned → Implemented via a small docs: PR
    # (branch off origin/main in a throw-away worktree; commit -s with the
    # Assisted-by trailer; label "type: docs" + "$MILESTONE_LABEL"; squash auto-merge).
  fi
fi
```

---

## STEP 8 — Post-merge verification (dispatch the `post-merge` agent)

For the merged PR, dispatch the `post-merge` agent (pinned to Haiku — mechanical GitHub/GHCR
verification). Foreground, to honor this command's end-to-end contract; the agent SKIPs fast
when the diff built no images and no digest-bump issues are open.

```
Agent({
  description: "Post-merge verify PR #PR_N (issue #ISSUE_N)",
  subagent_type: "post-merge",
  run_in_background: false,
  prompt: """
    PR_NUMBER:    <PR_N>
    MERGE_SHA:    <MERGE_SHA>
    ISSUE_NUMBER: <ISSUE_N>
    SESSION_DATE: <date>
    REPO:         <repo root>
    WT:           /tmp/zynax-postmerge-auto-<PR_N>   (your literal private worktree path)

    Verify post-merge CI, GHCR artifacts, and digest pins per your agent definition.
    Back-fill the originating PR's "Post-merge digest sync → main" Evidence
    placeholder. End with ## Post-Merge Evidence and ## Session Learnings.
  """
})
```

---

## STEP 9 — Done report + learnings

Append **every** dispatched agent's `## Session Learnings` block to the matching
`docs/ai-learnings/<domain>.md`: the canvas agent from STEP 4 (→ `spdd-canvas.md`), the domain
agent, and the post-merge agent (→ `ci-release.md`) — same flow as `/lib:deliver-batch` STEP 8
(throw-away worktree off `origin/main`, `docs:` PR, squash auto-merge).

```
=== DONE: Issue #<ISSUE_N> ===
Story:       <ISSUE_TITLE>
Agent:       <agent> (model-routed)
PR:          #<PR_N> — MERGED (merge SHA <MERGE_SHA>)
CI:          All required checks passed ✓
Post-merge:  <digest PR / SKIP reason>
Issue state: CLOSED ✓
Next:        Run /deliver to see what to pick up next.
```

If the domain agent crashed mid-run (no `## Result` block): recover per
`/lib:deliver-batch` STEP 7's crashed-agent procedure (inspect `/tmp/zynax-auto-<ISSUE_N>`,
finish or re-dispatch, then sweep). Never sweep a tree that still holds unpushed work.

---

## Error handling reference

| Condition | Action |
|-----------|--------|
| Issue already CLOSED | Exit 0 — nothing to do |
| Issue has `status: in-progress` | Stop — report assignee; don't steal |
| Branch already on remote | Stop — hard-claimed by another session |
| Security review FAIL on canvas | Stop — report Tier 2 findings; remove soft claim |
| Agent reports "claim lost" | Exit 0 — another session won the race; remove soft claim |
| Agent reports red gate / CI failure | Stop — report failing check; remove soft claim; hard-claim branch stays for inspection |
| Agent crashed (no ## Result) | Recover from its worktree per deliver-batch STEP 7; never blind-sweep |
| Required check stuck pending (no ## Result after ~45 min) | Inspect the stuck check yourself; then treat as crashed-agent recovery — the agent's `--watch` has no timeout of its own |
| Ejected from merge queue twice | Stop — investigate the flaky queue check before re-queueing |
