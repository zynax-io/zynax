---
description: Milestone delivery planner — reads live GitHub + canvas state, computes dependency-aware parallel groups, and outputs /deliver commands to run. Respects in-progress issues across all machines.
argument-hint: "[--verbose]"
---

# /lib:sequence — Dependency-aware delivery planner (building block of /deliver)

> **Building block** — invoked by `/deliver` to sequence ready work, not run directly.\n> **Scope contract:** repo-wide `status: ready` work by default; `--milestone M` filters.\n

Read the active milestone's live state, build a dependency graph, detect in-progress sessions on any machine,
and output the next set of `/deliver` commands to run — in parallel where safe,
sequentially where dependencies require it.

> **This command is read-only.** It never writes code, creates branches, or modifies issues.
> Its only output is a prioritised command plan for the human to execute.

---

## STEP 1 — Sync and read canonical state

```bash
git fetch origin --prune
git checkout main && git pull --rebase origin main 2>/dev/null || true

# ── Active-milestone config (SSoT: state/milestone.yaml) ────────────────────
# Loaded at runtime; no milestone name, number, or label is hardcoded in this
# file. Updated only by /milestone close and /milestone open.
CFG=state/milestone.yaml
MILESTONE_NAME=$(awk '/^active:/{f=1} f && /^  name:/{print $2; exit}' "$CFG")
MILESTONE_TITLE=$(awk -F'"' '/^active:/{f=1} f && /^  title:/{print $2; exit}' "$CFG")
MILESTONE_NUMBER=$(awk '/^active:/{f=1} f && /^  github_milestone_number:/{print $2; exit}' "$CFG")
MILESTONE_VERSION=$(awk '/^active:/{f=1} f && /^  version:/{print $2; exit}' "$CFG")
PLANNING_DOC=$(awk '/^active:/{f=1} f && /^  planning_doc:/{print $2; exit}' "$CFG")
MILESTONE_LABEL=$(awk -F'"' '/^    milestone:/{print $2; exit}' "$CFG")
GH_MILESTONE="${MILESTONE_TITLE} (${MILESTONE_NAME})"   # GitHub milestone title
# ─────────────────────────────────────────────────────────────────────────────

# Read planning docs
cat "$PLANNING_DOC"
cat state/current-milestone.md
```

---

## STEP 2 — Snapshot all active-milestone issues from GitHub

```bash
# All open milestone issues (stories + EPICs)
OPEN=$(gh issue list \
  --milestone "$GH_MILESTONE" \
  --state open \
  --limit 300 \
  --json number,title,body,labels,assignees,state)

# All closed milestone issues
CLOSED=$(gh issue list \
  --milestone "$GH_MILESTONE" \
  --state closed \
  --limit 300 \
  --json number,title,labels)

# All open PRs (to detect hard-claimed branches)
OPEN_PRS=$(gh pr list \
  --state open \
  --limit 100 \
  --json number,title,headRefName,author,statusCheckRollup,mergeStateStatus)

# Remote branches (parallel-session hard claims)
git fetch origin --prune
REMOTE_BRANCHES=$(git ls-remote origin 'refs/heads/*' \
  | awk '{print $2}' | sed 's|refs/heads/||')
```

---

## STEP 3 — Classify each open story issue

For each open issue, determine its state:

| State | Condition |
|-------|-----------|
| `IN_PROGRESS` | Has label `status: in-progress` OR a remote branch matching `<type>/<N>-*` exists OR an open PR with head-ref matching `<type>/<N>-*` |
| `BLOCKED` | One or more dependency issues are still open (from "Pending #N" or "Dependencies:" in issue body) |
| `READY` | Open, not in-progress, all dependencies closed |
| `EPIC` | Has label `type: epic` — EPICs are resolved to their story children, not run directly |

```bash
# For each issue N in $OPEN:
#   1. Check labels for "status: in-progress"  → IN_PROGRESS
#   2. Check $REMOTE_BRANCHES for pattern "^[a-z]+/${N}-"  → IN_PROGRESS (hard claim)
#   3. Check $OPEN_PRS headRefName for pattern "^[a-z]+/${N}-"  → IN_PROGRESS (PR open)
#   4. Parse body for dependency refs: "Pending #NNN", "Depends on #NNN", "Dependencies: #NNN"
#      For each dep #D: check if D is in $CLOSED → satisfied; if in $OPEN → blocking
#   5. If all deps satisfied and not in-progress → READY
```

Extract dependency references from issue bodies:
```bash
# Pattern matches: "Pending #NNN", "Depends on #NNN", "Dependency: #NNN", "after #NNN"
echo "$OPEN" | jq -r '.[] | "\(.number) \(.body)"' \
  | grep -oP '(?:ending|epends on|ependency|after) #\K\d+' \
  | sort -nu
```

---

## STEP 4 — Read canvas states for all feat: EPICs

```bash
for CANVAS in docs/spdd/*/canvas.md; do
  EPIC_N=$(basename "$(dirname "$CANVAS")" | grep -oP '^\d+')
  STATUS=$(grep -m1 '^Status:' "$CANVAS" | awk '{print $2}')
  echo "EPIC #$EPIC_N canvas: $STATUS  ($CANVAS)"
done
```

For EPICs with canvas `Status: Draft` or no canvas at all, `/deliver` will run the SPDD
pipeline automatically — but note them as "canvas work needed" in the plan output.

---

## STEP 5 — Build parallel execution groups

Group READY issues into parallel batches based on independence. Two issues are **independent** if:
- Neither references the other in its dependency list
- They touch different services/files (use issue title scope hints: `(api-gateway)` vs `(infra)`)

**Priority order** comes from the EPIC table in `$PLANNING_DOC` — top-to-bottom table order
is the priority. Never hardcode EPIC or issue numbers in this file; the planning doc and the
issue bodies are the source of truth.

Apply **hard sequential constraints** derived from issue bodies: any chain expressed as
"Depends on #N" / "Pending #N" forms a strict order; an EPIC whose body names an open blocker
is BLOCKED for all its children (re-check the blocker live before skipping).

---

## STEP 6 — Output the plan

Produce output in this format:

```
=== ${MILESTONE_NAME} Delivery Plan — <date> ===

## In-progress (skip — running on a machine or session)
  #NNN  <title>  [assignee: <login> | branch: <branch>]
  ...

## READY — Parallel batch 1 (run simultaneously in separate terminals)
  Terminal 1:  /deliver NNN   # <commit-type>(<scope>): <title>
  Terminal 2:  /deliver NNN   # <commit-type>(<scope>): <title>
  Terminal 3:  /deliver NNN   # <commit-type>(<scope>): <title>

  Note: Batch 2 becomes available once Batch 1 issues are closed.

## READY — Parallel batch 2 (run after batch 1 completes)
  /deliver NNN   # depends on #NNN from batch 1

## BLOCKED — waiting on dependencies
  #NNN  <title>  [blocked by: #NNN (<title>)]
  ...

## BLOCKED — external decisions required (no code can unblock these)
  #NNN  <title>  [blocked by: <description of blocker>]
  ...

## Canvas work needed (run before /deliver for these EPICs)
  EPIC #NNN: no canvas → /deliver NNN will auto-create it
  EPIC #NNN: canvas Status: Draft → /deliver NNN will auto-align it
  ...

## EPIC completion summary
  EPIC #NNN <title>: N/M stories done (M-N remaining)
  ...

## ${MILESTONE_NAME} exit criteria progress
  [ ] All K8s EPIC stories merged (Helm ✓/✗, mTLS ✓, supply-chain ✓, ...)
  [ ] v0.5.0 release tag pushed
  [ ] GitHub milestone "${GH_MILESTONE}" closed
```

---

## STEP 7 — Stale claim detection

```bash
# Warn about issues labeled "status: in-progress" with no open PR and no remote branch
# (likely a crashed session that forgot to clean up)
STALE_CLAIMS=$(echo "$OPEN" | jq -r \
  '.[] | select(.labels[].name == "status: in-progress") | .number' \
  | while read N; do
    HAS_BRANCH=$(echo "$REMOTE_BRANCHES" | grep -cE "^[a-z]+/${N}-" || true)
    HAS_PR=$(echo "$OPEN_PRS" | jq -r '.[].headRefName' | grep -cE "^[a-z]+/${N}-" || true)
    [ "$HAS_BRANCH" -eq 0 ] && [ "$HAS_PR" -eq 0 ] && echo "$N"
  done)

if [ -n "$STALE_CLAIMS" ]; then
  echo ""
  echo "## WARNING — Possibly stale claims (label with no branch/PR)"
  for N in $STALE_CLAIMS; do
    ASSIGNEE=$(gh issue view "$N" --json assignees --jq '[.assignees[].login] | join(", ")')
    echo "  #$N (assignee: $ASSIGNEE) — if session crashed, remove label: gh issue edit $N --remove-label 'status: in-progress'"
  done
fi
```

---

## STEP 8 — Recommended next action

Based on the plan, provide a single clear recommendation:

- If in-progress count is < 3 and batch 1 has available issues → recommend which terminals to open
- If all batch 1 is in-progress → "wait for current sessions to complete, or start batch 2 if independent"
- If no READY issues → report the blocking bottleneck (EPIC, canvas, external decision)
- If the milestone is complete → report exit criteria and recommend tagging ${MILESTONE_VERSION}

Example output:
```
## Recommended next action
Run the following in 3 parallel terminals (all independent):
  Terminal 1:  /deliver 859
  Terminal 2:  /deliver 868
  Terminal 3:  /deliver 874

Then: once #868 closes, run /deliver 865 (GHCR hygiene chain).
```

---

## Shared-state contract with /deliver

| Event | Who updates | What changes |
|-------|-------------|-------------|
| Session starts on issue N | `/deliver` | Adds `status: in-progress` label + self-assign |
| Branch pushed (hard claim) | `/deliver` | Remote branch `<type>/<N>-*` appears |
| PR opened | `/deliver` | Open PR with headRefName `<type>/<N>-*` |
| CI completes + PR merged | `/deliver` | PR closed, issue closed, branch deleted, label removed |
| Session crashes mid-run | Manual cleanup | Remove `status: in-progress` label; delete stale branch |

`/deliver` reads all three signals (label + branch + PR) to detect in-progress work. The **branch**
is the authoritative hard claim; the **label** is a soft signal visible before any branch push.
Because label assignment is not atomic with branch push, `/deliver` always cross-checks both.

---

## Reusable detection snippet

```bash
# Returns "in-progress" | "done" | "ready" | "blocked" for a given issue N
issue_state() {
  local N=$1
  # Done?
  gh issue view "$N" --json state --jq .state | grep -q CLOSED && { echo "done"; return; }
  # In-progress by label?
  gh issue view "$N" --json labels --jq '[.labels[].name] | any(. == "status: in-progress")' \
    | grep -q true && { echo "in-progress"; return; }
  # In-progress by branch?
  git ls-remote origin "refs/heads/*${N}-*" | grep -q . && { echo "in-progress"; return; }
  # Blocked by open dependency?
  DEPS=$(gh issue view "$N" --json body --jq .body \
    | grep -oP '(?:ending|epends on|ependency) #\K\d+')
  for D in $DEPS; do
    gh issue view "$D" --json state --jq .state | grep -q OPEN && { echo "blocked:$D"; return; }
  done
  echo "ready"
}
```
