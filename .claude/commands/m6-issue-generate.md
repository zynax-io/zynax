---
description: Fully autonomous M6 story delivery — claims issue via GitHub label, runs SPDD pipeline for feat: issues, implements, waits for CI, verifies Docker artifacts, merges, and marks done. Cross-machine safe.
argument-hint: "<story-or-epic-issue-number>"
---

# /m6-issue-generate — Autonomous M6 Story Delivery

End-to-end, unattended delivery of a single M6 issue: claim → canvas (if feat:) → implement →
local checks → push → PR → wait for CI → verify artifacts → squash-merge → cleanup → done.

> **Rules are not restated here.** Commit format, DCO + `Assisted-by` trailers, `GOWORK=off`,
> PR-size limits, hexagonal layout, coverage gates, and the SPDD workflow all live in **`AGENTS.md`**
> and **`CLAUDE.md`**. Read them before starting. This file is the *execution loop only*.

> **Canvas auto-alignment policy.** For `feat:` issues this skill auto-runs `/spdd-analysis`,
> `/spdd-reasons-canvas`, and `/spdd-security-review`. If the security review PASSes, Status is set
> to `Aligned` automatically and implementation proceeds. If it **FAILs** (Tier 2 findings that
> cannot be resolved inline), the skill stops and reports — do not proceed from a failed review.

---

## Cross-machine claim protocol (non-negotiable)

Two layers prevent duplicate work across concurrent sessions on any machine:

1. **Soft claim** — add `status: in-progress` label + self-assign the issue on GitHub (visible
   immediately to all sessions and to `/m6-plan`). This is a *signal*, not a lock.
2. **Hard claim** — push an empty branch to GitHub before writing any code. Only one `git push -u
   origin $BRANCH` wins when two sessions race. A rejected push means the story is taken → stop.

Always check both before starting. Never assume an issue is free just because you read it as open.

---

## STEP 0 — Pre-flight: read the rules

```bash
# Mandatory reads every run — do not skip
cat CLAUDE.md                            # dev loop, PR-size, SPDD rules
cat AGENTS.md                            # constitution: layer boundaries, mandates, anti-patterns
cat state/current-milestone.md           # active blockers, health
cat docs/milestones/M6-planning.md       # dependency table, EPIC status
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

# Check if a branch for this issue already exists on remote (hard-claimed by another session)
git fetch origin --prune
EXISTING_BRANCH=$(git ls-remote origin 'refs/heads/*' | awk '{print $2}' \
  | sed 's|refs/heads/||' | grep -E "^[a-z]+/${ISSUE_N}-" | head -1)
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

## STEP 2.5 — Create isolated worktree

Create a throw-away checkout so the rest of the skill runs on a guaranteed clean tree,
completely isolated from the caller's working directory.

```bash
WORKTREE_PATH="/tmp/zynax-auto-${ISSUE_N}"

# Remove any leftover from a previous crashed run
git worktree remove "$WORKTREE_PATH" --force 2>/dev/null || true
rm -rf "$WORKTREE_PATH" 2>/dev/null || true

# Create a fresh worktree based on current origin/main
git fetch origin --prune
git worktree add "$WORKTREE_PATH" origin/main

# All subsequent steps run from this directory
cd "$WORKTREE_PATH"
echo "Worktree ready: $WORKTREE_PATH"
```

> Every file read, edit, build, and commit from this point on happens inside
> `$WORKTREE_PATH`. The caller's workspace is untouched.

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
Go to **STEP 3-EPIC**. Otherwise skip to **STEP 3.5**.

---

## STEP 3.5 — Identify expert persona and start activity log

Determine which expert persona applies to this issue (same routing table as `/m6-orchestrate`):

```bash
EXPERT_TAG="general"
EXPERT_NAME="General"
case "$ISSUE_TITLE" in
  *"(api-gateway)"*|*"(workflow-compiler)"*|*"(engine-adapter)"*|\
  *"(task-broker)"*|*"(agent-registry)"*|*"(event-bus)"*|*"(memory-service)"*)
    EXPERT_TAG="go-svc"; EXPERT_NAME="Go Services Engineer" ;;
  *"(infra)"*|*helm*|*k8s*)
    EXPERT_TAG="infra"; EXPERT_NAME="Infrastructure / SRE Engineer" ;;
  *"(ci)"*|*actions*|*images.yaml*)
    EXPERT_TAG="ci-rel"; EXPERT_NAME="CI / Release Engineer" ;;
  *"(agents)"*|*"(sdk)"*|*python*|*adapter*)
    EXPERT_TAG="py-adapter"; EXPERT_NAME="Python Adapter Engineer" ;;
  test:*)
    EXPERT_TAG="bdd"; EXPERT_NAME="BDD / Contract Engineer" ;;
esac
# feat: with no Aligned canvas → spdd first
[ "$NEEDS_CANVAS" = "true" ] && [ "$CANVAS_STATUS" != "Aligned" ] && \
  EXPERT_TAG="spdd→${EXPERT_TAG}" EXPERT_NAME="SPDD Canvas → ${EXPERT_NAME}"

echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║  EXPERT: $EXPERT_NAME"
echo "║  TAG:    $EXPERT_TAG   ISSUE: #$ISSUE_N"
echo "║  TITLE:  $ISSUE_TITLE"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""
echo "[$EXPERT_TAG #$ISSUE_N $(date +%H:%M:%S)] START: $ISSUE_TITLE"
```

Use this log format at every subsequent step:
```
[$EXPERT_TAG #$ISSUE_N $(date +%H:%M:%S)] <PHASE>: <one-line description>
```

---

## STEP 3-EPIC — Resolve EPIC to next story issue

```bash
# Find the canvas (if it exists)
CANVAS_DIR=$(ls docs/spdd/ 2>/dev/null | grep -E "^${ISSUE_N}-" | head -1)

# Determine canvas state
if [ -n "$CANVAS_DIR" ]; then
  CANVAS_STATUS=$(grep -m1 "^Status:" "docs/spdd/$CANVAS_DIR/canvas.md" | awk '{print $2}')
  echo "Canvas found: docs/spdd/$CANVAS_DIR/canvas.md — Status: $CANVAS_STATUS"
else
  CANVAS_STATUS="none"
  echo "No canvas found for EPIC #$ISSUE_N"
fi

# If canvas not Aligned, run SPDD pipeline (STEP 4-CANVAS will handle this)
# Find the next open story issue for this EPIC (lowest step number, not yet in-progress or done)
STORY_ISSUES=$(gh issue list --milestone "K8s Production-Ready (M6)" --state open \
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

## STEP 4-CANVAS — Run SPDD pipeline (feat: only, when canvas not Aligned)

Skip this step if `COMMIT_TYPE != "feat"` or if the canvas is already `Aligned`.

```bash
echo "[$EXPERT_TAG #$ISSUE_N $(date +%H:%M:%S)] CANVAS: running SPDD pipeline for EPIC #$EPIC_N"
```

```bash
# Find EPIC number referenced in story body (pattern: "EPIC #NNN" or "parent #NNN")
EPIC_N=$(echo "$ISSUE" | jq -r .body | grep -oP '(?<=#)\d+' | head -1)
[ -z "$EPIC_N" ] && EPIC_N="$ISSUE_N"   # fallback: issue is its own EPIC

CANVAS_DIR=$(ls docs/spdd/ 2>/dev/null | grep -E "^${EPIC_N}-" | head -1)

if [ -z "$CANVAS_DIR" ] || [ "$CANVAS_STATUS" != "Aligned" ]; then
  echo "Running SPDD pipeline for EPIC #$EPIC_N..."

  # Analysis — understand codebase impact, ADR constraints, Tier 2 flags
  /spdd-analysis "$EPIC_N"

  # Generate canvas (Status: Draft)
  /spdd-reasons-canvas "$EPIC_N"

  CANVAS_DIR=$(ls docs/spdd/ | grep -E "^${EPIC_N}-" | head -1)
  CANVAS_PATH="docs/spdd/$CANVAS_DIR/canvas.md"

  # Security review — MUST PASS before auto-alignment
  REVIEW_RESULT=$(/spdd-security-review "$CANVAS_PATH" 2>&1)
  echo "$REVIEW_RESULT"
  if echo "$REVIEW_RESULT" | grep -qi "FAIL\|Tier 2 finding\|BLOCKED"; then
    echo "Security review FAILED — cannot auto-align. Resolve Tier 2 findings and re-run."
    gh issue edit "$ISSUE_N" --remove-label "status: in-progress"
    exit 1
  fi

  # Auto-align: set Status: Aligned in canvas
  sed -i 's/^Status: Draft/Status: Aligned/' "$CANVAS_PATH"
  grep "^Status:" "$CANVAS_PATH"   # confirm
  echo "Canvas auto-aligned: $CANVAS_PATH"
fi

# Create story issues if not yet created for this EPIC
STORY_COUNT=$(gh issue list --milestone "K8s Production-Ready (M6)" --state all \
  --json body --jq "[.[] | select(.body | test(\"#${EPIC_N}\"))] | length")
[ "$STORY_COUNT" -eq 0 ] && /spdd-story "$EPIC_N"
```

---

## STEP 5 — Sync main + create branch (atomic hard claim)

```bash
echo "[$EXPERT_TAG #$ISSUE_N $(date +%H:%M:%S)] CLAIM: creating branch + hard claim on origin"
# Worktree was created from origin/main in STEP 2.5 — already clean and up to date.

# Derive branch name from issue title: <type>/<N>-<slug>
SLUG=$(echo "$ISSUE_TITLE" | sed 's|[^a-zA-Z0-9 ]||g' | tr '[:upper:]' '[:lower:]' \
  | tr ' ' '-' | sed 's/^[a-z]*-[a-z0-9]*-//' | cut -c1-40 | sed 's/-$//')
BRANCH="${COMMIT_TYPE}/${ISSUE_N}-${SLUG}"

git checkout -b "$BRANCH"

# Hard claim: push empty branch NOW — only one session wins
if ! git push -u origin "$BRANCH" 2>&1; then
  echo "HARD CLAIM FAILED: branch $BRANCH already on remote — story #$ISSUE_N taken by another session."
  git checkout main && git branch -D "$BRANCH"
  gh issue edit "$ISSUE_N" --remove-label "status: in-progress"
  exit 1
fi
echo "Hard-claimed: branch $BRANCH pushed to origin."
```

---

## STEP 6 — Implement

```bash
echo "[$EXPERT_TAG #$ISSUE_N $(date +%H:%M:%S)] CODE: implementing — reading issue scope and referenced files"
```

For `feat:` issues with an Aligned canvas, use `/spdd-generate`:

```bash
if [ "$NEEDS_CANVAS" = "true" ]; then
  CANVAS_PATH="docs/spdd/$CANVAS_DIR/canvas.md"
  # Identify which O-step this story covers (from story title "step N")
  STEP_N=$(echo "$ISSUE_TITLE" | grep -oP '(?<=step )\d+' | head -1)
  echo "Implementing canvas O-step $STEP_N via /spdd-generate"
  /spdd-generate "$CANVAS_PATH"
  # /spdd-generate stops after one O-step — verify it generated the right step
fi
```

For SPDD-exempt issues (`fix:`, `refactor:`, `ci:`, `chore:`), implement directly from the issue
body's scope and acceptance criteria. Read all referenced files before writing any code.

**After implementation, update state files in the same diff:**
1. `docs/milestones/M6-planning.md` — flip this story's row ⬜→✅
2. `state/current-milestone.md` — update EPIC progress
3. Canvas O-step — mark ✅; run `/spdd-sync <canvas>` if implementation diverged from spec
4. `services/<svc>/AGENTS.md` — only if a new gRPC method, K8s resource type, or env var added

---

## STEP 7 — Local verification gates

```bash
echo "[$EXPERT_TAG #$ISSUE_N $(date +%H:%M:%S)] TEST: running local gates (build, test, lint, security)"
```

Run all required checks before committing. Do not commit if any gate fails.

```bash
# Identify touched service directories
TOUCHED_DIRS=$(git diff --name-only | grep -oP '^services/[^/]+' | sort -u)

for SVC_DIR in $TOUCHED_DIRS; do
  echo "=== $SVC_DIR ==="
  (cd "$SVC_DIR" && GOWORK=off go build ./...)              || { echo "BUILD FAILED in $SVC_DIR"; exit 1; }
  (cd "$SVC_DIR" && GOWORK=off go test ./... -race -timeout 60s) || { echo "TESTS FAILED in $SVC_DIR"; exit 1; }
  # Domain coverage gate ≥90% (only if domain/ was touched)
  git diff --name-only | grep -q "$SVC_DIR/internal/domain/" && \
    (cd "$SVC_DIR" && GOWORK=off go test ./internal/domain/... -coverprofile=/tmp/cov.out \
      && go tool cover -func /tmp/cov.out | tail -1 | awk '{if ($3+0 < 90.0) exit 1}') \
    || { echo "DOMAIN COVERAGE BELOW 90% in $SVC_DIR"; exit 1; }
done

# Python adapters
TOUCHED_AGENTS=$(git diff --name-only | grep -oP '^agents/[^/]+' | sort -u)
[ -n "$TOUCHED_AGENTS" ] && make lint-python && make test-python

# Lint + security (runs in Docker)
make lint    || { echo "LINT FAILED"; exit 1; }
make security || { echo "SECURITY SCAN FAILED"; exit 1; }

# BDD (only if gRPC boundary touched)
git diff --name-only | grep -q '\.proto\|_pb2\|\.go.*grpc' && {
  # .feature file must exist before BDD test
  ls protos/tests/*/features/*.feature 2>/dev/null | head -1 || {
    echo "BDD: gRPC boundary touched but no .feature file found — create it first (ADR-016)"
    exit 1
  }
  make test-bdd || { echo "BDD TESTS FAILED"; exit 1; }
}

echo "All local gates passed."
```

---

## STEP 8 — Commit

```bash
echo "[$EXPERT_TAG #$ISSUE_N $(date +%H:%M:%S)] COMMIT: all gates green — staging and committing"
# Verify title length ≤ 72 chars
echo -n "${COMMIT_TYPE}(<scope>): <subject>" | wc -c   # replace before committing

git add -p   # stage intentionally (never git add -A)

# PR body file (used in STEP 9)
cat > /tmp/pr-body-${ISSUE_N}.md << 'EOF'
## Summary
<1-3 sentences — what changes and why, referencing the canvas O-step>

## EPIC canvas
`docs/spdd/<EPIC_N>-<slug>/canvas.md` — O-step <N>  (N/A for SPDD-exempt issues)

## Acceptance criteria
- [x] <criterion 1>  [evidence: test output / file:line / log]
- [x] <criterion 2>  [evidence]
- [x] <criterion 3>  [evidence]

## Test plan

### Build & unit
- [x] `GOWORK=off go build ./...` — exit 0  [evidence]
- [x] `GOWORK=off go test ./... -race -timeout 60s` — all pass  [evidence]
- [x] Domain coverage ≥90%  [evidence / N/A]

### Lint & security
- [x] `make lint` — exit 0  [evidence]
- [x] `make security` — no new findings  [evidence]

### Contract
- [x] `.feature` file committed before implementation  [evidence / N/A]
- [x] `make test-bdd` — all scenarios pass  [evidence / N/A]

### Engineering hygiene
- [x] `M6-planning.md` row ⬜→✅ in this diff
- [x] `current-milestone.md` updated in this diff
- [x] Canvas O-step ✅; `/spdd-sync` run if impl diverged
- [x] Branched off fresh `origin/main` · PR ≤900 lines · trailers on every commit
EOF

# Fill in evidence from STEP 7 output before committing PR body

git commit -s -m "$(cat <<EOF
${COMMIT_TYPE}(<scope>): <subject>

<why — one sentence referencing canvas O-step N of EPIC #EPIC_N>

Closes #${ISSUE_N}

Assisted-by: Claude/claude-sonnet-4-6
EOF
)"

git push --force-with-lease
```

---

## STEP 9 — Open PR

```bash
echo "[$EXPERT_TAG #$ISSUE_N $(date +%H:%M:%S)] PR: opening pull request against main"
echo -n "<title>" | wc -c   # must be ≤ 72 chars

PR_URL=$(gh pr create \
  --base main \
  --title "${COMMIT_TYPE}(<scope>): <subject>" \
  --assignee "@me" \
  --label "type: ${COMMIT_TYPE},milestone: M6,area: <area>" \
  --body-file "/tmp/pr-body-${ISSUE_N}.md")

PR_N=$(echo "$PR_URL" | grep -oP '\d+$')
echo "Opened PR #$PR_N: $PR_URL"
```

---

## STEP 10 — Wait for CI (blocking)

```bash
echo "[$EXPERT_TAG #$ISSUE_N $(date +%H:%M:%S)] CI_WAIT: PR #$PR_N — waiting for required checks"
```

Unlike `/resume-m6`, this command waits for CI to complete before merging. This is intentional —
the command's contract is end-to-end autonomous delivery, not a fire-and-forget push.

```bash
echo "Waiting for CI on PR #$PR_N (this may take 5–20 minutes)..."

# Poll every 60 s; timeout after 30 min (1800 s)
ELAPSED=0
CI_PASSED=false
while [ $ELAPSED -lt 1800 ]; do
  ROLLUP=$(gh pr view "$PR_N" --json statusCheckRollup,mergeStateStatus)
  MERGE_STATE=$(echo "$ROLLUP" | jq -r .mergeStateStatus)
  FAILED=$(echo "$ROLLUP" | jq '[.statusCheckRollup[]? | select(.isRequired==true) | .conclusion] | any(. == "FAILURE" or . == "ERROR" or . == "TIMED_OUT")')
  PENDING=$(echo "$ROLLUP" | jq '[.statusCheckRollup[]? | select(.isRequired==true) | .status] | any(. == "IN_PROGRESS" or . == "QUEUED" or . == "WAITING")')

  if [ "$FAILED" = "true" ]; then
    echo "CI FAILED on PR #$PR_N. Review failures before retrying."
    gh pr view "$PR_N" --web 2>/dev/null || true
    # Report which checks failed
    gh pr checks "$PR_N" | grep -E "fail|error" || true
    # Clean up soft claim — hard claim stays until branch is manually deleted
    gh issue edit "$ISSUE_N" --remove-label "status: in-progress"
    exit 1
  fi

  if [ "$PENDING" = "false" ] && [ "$MERGE_STATE" = "CLEAN" ]; then
    echo "All required CI checks passed. PR #$PR_N is CLEAN."
    CI_PASSED=true
    break
  fi

  echo "CI running... (${ELAPSED}s elapsed, state=$MERGE_STATE)"
  sleep 60
  ELAPSED=$((ELAPSED + 60))
done

[ "$CI_PASSED" = "false" ] && {
  echo "CI timed out after 30 minutes. Check PR #$PR_N manually."
  gh issue edit "$ISSUE_N" --remove-label "status: in-progress"
  exit 1
}
```

---

## STEP 11 — Verify Docker artifacts (when applicable)

```bash
# Only emitted when TOUCHES_IMAGE > 0:
echo "[$EXPERT_TAG #$ISSUE_N $(date +%H:%M:%S)] IMAGE_CHECK: verifying GHCR artifact publication"
```

Skip this step if the PR diff does not touch `Dockerfile*`, `.github/workflows/*image*`,
`*release*`, or `*publish*` files.

```bash
TOUCHES_IMAGE=$(git diff origin/main...HEAD --name-only \
  | grep -cE 'Dockerfile|workflows.*(image|release|push|publish)' || true)

if [ "$TOUCHES_IMAGE" -gt 0 ]; then
  echo "PR touches image-building files — waiting for post-merge image publication..."

  # Wait for merge first (STEP 12 below sets this)
  # Then verify image was published to ghcr.io

  REPO_OWNER="zynax-io"
  REPO_NAME="zynax"

  # List images that should have been updated (derive from workflow files touched)
  EXPECTED_IMAGES=$(git diff origin/main...HEAD --name-only \
    | grep -oP '(?<=workflows/).*(?=\.yml)' | grep -E 'image|release|push' | head -5)

  for IMG in $EXPECTED_IMAGES; do
    echo "Checking ghcr.io/$REPO_OWNER/$REPO_NAME/$IMG..."
    # Poll for new image version (up to 15 min post-merge)
    IMG_ELAPSED=0
    while [ $IMG_ELAPSED -lt 900 ]; do
      VERSION_COUNT=$(gh api "/orgs/$REPO_OWNER/packages/container/${REPO_NAME}%2F${IMG}/versions" \
        --jq 'length' 2>/dev/null || echo "0")
      [ "$VERSION_COUNT" -gt 0 ] && {
        LATEST_TAG=$(gh api "/orgs/$REPO_OWNER/packages/container/${REPO_NAME}%2F${IMG}/versions" \
          --jq '.[0].metadata.container.tags[0]' 2>/dev/null || echo "unknown")
        echo "Image ghcr.io/$REPO_OWNER/$REPO_NAME/$IMG:$LATEST_TAG confirmed."
        break
      }
      echo "Waiting for image publication... (${IMG_ELAPSED}s)"
      sleep 60
      IMG_ELAPSED=$((IMG_ELAPSED + 60))
    done
    [ $IMG_ELAPSED -ge 900 ] && echo "WARNING: image $IMG not confirmed after 15 min — check manually."
  done
fi
```

---

## STEP 12 — Rebase + squash-merge

```bash
echo "[$EXPERT_TAG #$ISSUE_N $(date +%H:%M:%S)] MERGE: CI green — rebasing and squash-merging PR #$PR_N"
git fetch origin --prune
git checkout "$BRANCH"
git rebase origin/main || {
  echo "CONFLICT during rebase on $BRANCH — resolve conflicts, then:"
  echo "  git rebase --continue && git push --force-with-lease"
  echo "Then re-run STEP 12."
  exit 1
}
git push --force-with-lease

gh pr merge "$PR_N" --squash
until [ "$(gh pr view "$PR_N" --json state --jq .state)" = "MERGED" ]; do
  echo "Waiting for merge to settle..."
  sleep 10
done
echo "PR #$PR_N merged."

# Delete the remote branch
git push origin --delete "$BRANCH" 2>/dev/null || true
git checkout main && git pull --rebase origin main
git branch -D "$BRANCH" 2>/dev/null || true
```

---

## STEP 13 — Verify issue closed

```bash
sleep 5   # allow GitHub to process Closes #N from squash-merge commit

ISSUE_STATE=$(gh issue view "$ISSUE_N" --json state --jq .state)
if [ "$ISSUE_STATE" != "CLOSED" ]; then
  # Manually close if Closes #N wasn't picked up (e.g. it was in the commit, not PR body)
  gh issue close "$ISSUE_N" --reason completed \
    --comment "Closed by squash-merge of PR #$PR_N. All acceptance criteria met."
fi
echo "Issue #$ISSUE_N is CLOSED."
```

---

## STEP 14 — Cleanup + done report

```bash
# Remove soft claim
gh issue edit "$ISSUE_N" --remove-label "status: in-progress" 2>/dev/null || true

# EPIC completion check: if all stories for the parent EPIC are now closed, close the EPIC too
if [ -n "$EPIC_N" ] && [ "$EPIC_N" != "$ISSUE_N" ]; then
  OPEN_STORIES=$(gh issue list --milestone "K8s Production-Ready (M6)" --state open \
    --json body --jq "[.[] | select(.body | test(\"#${EPIC_N}\"))] | length")
  if [ "$OPEN_STORIES" -eq 0 ]; then
    gh issue close "$EPIC_N" --reason completed \
      --comment "All O-steps merged. Canvas status: Implemented."
    # Mark canvas Implemented
    CANVAS_PATH=$(ls "docs/spdd/${EPIC_N}-"*/canvas.md 2>/dev/null | head -1)
    [ -n "$CANVAS_PATH" ] && sed -i 's/^Status: Aligned/Status: Implemented/' "$CANVAS_PATH" && {
      git checkout -b "docs/epic-${EPIC_N}-close-$(date +%Y%m%d%H%M)"
      git add "$CANVAS_PATH"
      git commit -s -m "docs(spdd): mark EPIC #${EPIC_N} canvas Implemented

      All O-steps merged for EPIC #${EPIC_N}.

      Assisted-by: Claude/claude-sonnet-4-6"
      git push -u origin HEAD
      gh pr create --title "docs(spdd): mark EPIC #${EPIC_N} canvas Implemented" \
        --body "All O-steps for EPIC #${EPIC_N} have been merged. Closing canvas." \
        --label "type: docs,milestone: M6"
    }
  fi
fi

echo "[$EXPERT_TAG #$ISSUE_N $(date +%H:%M:%S)] DONE: PR #$PR_N merged — issue #$ISSUE_N closed"

cat << EOF
=== DONE: Issue #${ISSUE_N} ===
Story:       $ISSUE_TITLE
Branch:      $BRANCH (deleted)
PR:          #$PR_N — MERGED
CI:          All required checks passed ✓
Artifacts:   $([ "$TOUCHES_IMAGE" -gt 0 ] && echo "Docker images verified ✓" || echo "N/A (no image-touching files)")
Issue state: CLOSED ✓
Next:        Run /m6-plan to see what to pick up next.
EOF

# Remove the isolated worktree — all work is merged, nothing to keep
cd /tmp   # leave the worktree directory before removing it
git -C "$OLDPWD" worktree remove "$WORKTREE_PATH" --force 2>/dev/null || true
rm -rf "$WORKTREE_PATH" 2>/dev/null || true
echo "Worktree $WORKTREE_PATH removed."
```

---

## Error handling reference

| Condition | Action |
|-----------|--------|
| Issue already CLOSED | Exit 0 — nothing to do |
| Issue has `status: in-progress` | Stop — report assignee; don't steal |
| Branch already on remote | Stop — hard-claimed by another session |
| Security review FAIL on canvas | Stop — report Tier 2 findings; remove soft claim |
| Local build/test failure | Stop — fix before committing; soft claim remains |
| CI red on PR | Stop — report failing checks; remove soft claim |
| CI timeout (>30 min) | Stop — check manually; remove soft claim |
| Rebase conflict | Stop — resolve manually; re-run STEP 12 |
| Branch delete fails | Continue — non-fatal; clean up manually |
