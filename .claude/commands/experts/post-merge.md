# Expert: Post-Merge Verifier

You verify that a recently merged PR's CI workflows completed successfully, confirm Docker images
were published to GHCR, update all digest pins, consolidate open digest-bump issues, and publish
structured evidence of every action taken.

**Expert tag:** `post-mrg`

---

## Activity log (emit at every phase transition)

```
[post-mrg PR#<N> <HH:MM:SS>] <PHASE>: <one-line description>  [ctx: ~<X>K | compress=<C> | msgs=<M>]
```

| Phase | When to emit |
|-------|-------------|
| `START` | First line — inputs received |
| `FIND_RUNS` | Before querying GitHub for workflow runs |
| `WAIT_CI` | Entering the polling loop for pending runs |
| `IMAGE_VERIFY` | Before checking GHCR for each affected image |
| `DIGEST_UPDATE` | Before updating digest pins in files |
| `ISSUE_TRIAGE` | Before processing open digest-bump issues |
| `COMMIT` | Before git add / git commit |
| `PR` | Before gh pr create |
| `CI_WAIT` | Polling for the digest-bump PR's own CI |
| `DONE` | All steps complete — print evidence block |
| `SKIP` | Nothing to do (no affected services in matrix) |
| `ERROR` | Any unrecoverable failure |

---

## Inputs (injected by orchestrator at dispatch time)

```
PR_NUMBER=<N>         # merged PR number
MERGE_SHA=<sha>       # merge commit SHA (full or 12-char short)
ISSUE_NUMBER=<N>      # the story issue this PR closed
SESSION_DATE=<YYYY-MM-DD>
```

---

## Phase 1 — Identify affected services

```bash
[post-mrg PR#${PR_NUMBER} $(date +%H:%M:%S)] START: post-merge verify for PR #${PR_NUMBER}  [ctx: ~10K | compress=0 | msgs=1]

# Get changed paths
CHANGED_FILES=$(gh pr view ${PR_NUMBER} --json files --jq '[.files[].path]')

# Map paths to services in the release.yml build matrix.
# Matrix services (as of 2026-06-08):
#   api-gateway, engine-adapter, workflow-compiler, task-broker, agent-registry, http-adapter
# NOT in matrix (no Docker image built): event-bus, memory-service
MATRIX_SERVICES="api-gateway engine-adapter workflow-compiler task-broker agent-registry http-adapter"

AFFECTED_SERVICES=""
for SVC in $MATRIX_SERVICES; do
  if echo "$CHANGED_FILES" | grep -q "\"services/${SVC}/"; then
    AFFECTED_SERVICES="$AFFECTED_SERVICES $SVC"
  fi
done
AFFECTED_SERVICES=$(echo "$AFFECTED_SERVICES" | xargs)

# Also flag infra changes that require base image checks
INFRA_CHANGED=false
if echo "$CHANGED_FILES" | grep -qE '"(Makefile|images/images\.yaml|\.github/workflows/)'; then
  INFRA_CHANGED=true
fi

echo "Affected services: ${AFFECTED_SERVICES:-none}"
echo "Infra changed: $INFRA_CHANGED"
```

> **Warning (as of 2026-06-08):** `event-bus` and `memory-service` are NOT in the release.yml
> build matrix. If only those services changed and INFRA_CHANGED is false, emit SKIP and stop.

If `$AFFECTED_SERVICES` is empty AND `$INFRA_CHANGED` is false, emit SKIP evidence block and stop.

---

## Phase 2 — Find and wait for CI workflow runs

```bash
[post-mrg PR#${PR_NUMBER} $(date +%H:%M:%S)] FIND_RUNS: querying workflow runs for merge commit ${MERGE_SHA}  [ctx: ~12K | compress=0 | msgs=3]

RUNS=$(gh run list --commit "${MERGE_SHA}" --limit 20 \
  --json databaseId,name,status,conclusion,workflowName,createdAt \
  --jq '.[]')

RELEASE_RUN=$(echo "$RUNS" | jq -r 'select(.workflowName == "Release") | .databaseId' | head -1)
TOOLS_RUN=$(echo "$RUNS" | jq -r 'select(.workflowName | test("tools|ci-runner"; "i")) | .databaseId' | head -1)
```

Poll until complete (max 20 minutes, 60 s intervals):

```bash
[post-mrg PR#${PR_NUMBER} $(date +%H:%M:%S)] WAIT_CI: release=${RELEASE_RUN:-N/A} tools=${TOOLS_RUN:-N/A}  [ctx: ~12K | compress=0 | msgs=4]

DEADLINE=$(($(date +%s) + 1200))
while [ "$(date +%s)" -lt "$DEADLINE" ]; do
  ALL_DONE=true
  for RUN_ID in $RELEASE_RUN $TOOLS_RUN; do
    [ -z "$RUN_ID" ] && continue
    RESULT=$(gh run view "$RUN_ID" --json status,conclusion \
      --jq '"status=\(.status) conclusion=\(.conclusion)"')
    echo "  run ${RUN_ID}: $RESULT"
    CONCLUSION=$(gh run view "$RUN_ID" --json conclusion --jq .conclusion)
    if [ "$CONCLUSION" = "failure" ] || [ "$CONCLUSION" = "cancelled" ]; then
      echo "[post-mrg PR#${PR_NUMBER}] ERROR: run ${RUN_ID} ${CONCLUSION} — emit evidence and stop"
      exit 1
    fi
    STATUS=$(gh run view "$RUN_ID" --json status --jq .status)
    [ "$STATUS" != "completed" ] && ALL_DONE=false
  done
  $ALL_DONE && break
  sleep 60
done
```

---

## Phase 3 — Verify GHCR image publication

```bash
[post-mrg PR#${PR_NUMBER} $(date +%H:%M:%S)] IMAGE_VERIFY: checking GHCR for affected services  [ctx: ~13K | compress=0 | msgs=5]

VERIFIED_IMAGES=""
for SVC in $AFFECTED_SERVICES; do
  LATEST=$(gh api "/orgs/zynax-io/packages/container/zynax%2F${SVC}/versions" \
    --jq '.[0] | {digest: .name, tags: .metadata.container.tags, created: .created_at}' 2>/dev/null)

  if [ -z "$LATEST" ]; then
    echo "  ${SVC}: NOT FOUND in GHCR"
    IMAGE_VERIFY_STATUS="FAIL"
  else
    DIGEST=$(echo "$LATEST" | jq -r .digest)
    TAGS=$(echo "$LATEST" | jq -r '.tags | join(", ")')
    echo "  ${SVC}: ${DIGEST:0:19}... tags=[${TAGS}]"
    eval "SVC_DIGEST_$(echo $SVC | tr - _)=${DIGEST}"
  fi
done
```

---

## Phase 4a — Update service image digest pins in docker-compose

```bash
[post-mrg PR#${PR_NUMBER} $(date +%H:%M:%S)] DIGEST_UPDATE: updating docker-compose.services.yml  [ctx: ~14K | compress=0 | msgs=6]

COMPOSE_FILE="infra/docker-compose/docker-compose.services.yml"
COMPOSE_PINS_UPDATED=""

for SVC in $AFFECTED_SERVICES; do
  VARNAME="SVC_DIGEST_$(echo $SVC | tr - _)"
  NEW_DIGEST="${!VARNAME}"
  [ -z "$NEW_DIGEST" ] && continue

  if grep -q "zynax-io/zynax/${SVC}@sha256:" "$COMPOSE_FILE" 2>/dev/null; then
    OLD_DIGEST=$(grep "zynax-io/zynax/${SVC}@sha256:" "$COMPOSE_FILE" \
      | grep -oP 'sha256:[a-f0-9]+' | head -1)
    if [ "$OLD_DIGEST" != "$NEW_DIGEST" ]; then
      sed -i "s|zynax-io/zynax/${SVC}@${OLD_DIGEST}|zynax-io/zynax/${SVC}@${NEW_DIGEST}|g" \
        "$COMPOSE_FILE"
      echo "  ${SVC}: updated ${OLD_DIGEST:0:19}... -> ${NEW_DIGEST:0:19}..."
      COMPOSE_PINS_UPDATED="${COMPOSE_PINS_UPDATED} ${SVC}"
    else
      echo "  ${SVC}: already current"
    fi
  fi
done
```

---

## Phase 4b — Update base image digests in images/images.yaml

Run only when `$INFRA_CHANGED` is true OR `$TOOLS_RUN` is non-empty:

```bash
IMAGES_FILE="images/images.yaml"
YAML_PINS_UPDATED=""

CI_RUNNER_LATEST=$(gh api \
  "/orgs/zynax-io/packages/container/zynax%2Fci-runner/versions" \
  --jq '.[0] | {digest: .name, tag: .metadata.container.tags[0]}' 2>/dev/null)

if [ -n "$CI_RUNNER_LATEST" ]; then
  NEW_DIGEST=$(echo "$CI_RUNNER_LATEST" | jq -r .digest)
  NEW_TAG=$(echo "$CI_RUNNER_LATEST" | jq -r .tag)
  OLD_DIGEST=$(grep -A3 'name: ci-runner' "$IMAGES_FILE" | grep digest | awk '{print $2}')

  if [ "$OLD_DIGEST" != "$NEW_DIGEST" ]; then
    yq -i "(.images[] | select(.name == \"ci-runner\") | .digest) = \"${NEW_DIGEST}\"" \
      "$IMAGES_FILE"
    yq -i "(.images[] | select(.name == \"ci-runner\") | .tag) = \"${NEW_DIGEST}\"" \
      "$IMAGES_FILE"
    echo "  ci-runner: ${OLD_DIGEST:0:19}... -> ${NEW_DIGEST:0:19}..."
    YAML_PINS_UPDATED="${YAML_PINS_UPDATED} ci-runner"
    make sync-images
  fi
fi
```

> **Note:** Never add service images (api-gateway, etc.) to `images/images.yaml`.
> That file is only for base/toolchain images: ci-runner, golang-alpine, distroless-static, python-slim.

---

## Phase 5 — Triage digest-bump issues

```bash
[post-mrg PR#${PR_NUMBER} $(date +%H:%M:%S)] ISSUE_TRIAGE: consolidating open digest-bump issues  [ctx: ~15K | compress=0 | msgs=7]

BUMP_ISSUES=$(gh issue list --state open --limit 100 \
  --json number,title,createdAt \
  --jq '[.[] | select(.title | test("bump.*digest|digest.*bump"; "i"))] | sort_by(.number)')

BUMP_COUNT=$(echo "$BUMP_ISSUES" | jq length)
STALE_ISSUES=""
IMPLEMENT_ISSUE=""

if [ "$BUMP_COUNT" -gt 1 ]; then
  NEWEST_N=$(echo "$BUMP_ISSUES" | jq -r '.[-1].number')
  STALE_ISSUES=$(echo "$BUMP_ISSUES" | jq -r '.[:-1][].number' | tr '\n' ' ')
  for OLD_N in $STALE_ISSUES; do
    gh issue comment "$OLD_N" --body "Superseded by #${NEWEST_N} — closing."
    gh issue close "$OLD_N" --reason "not planned"
    echo "  Closed #${OLD_N} (superseded by #${NEWEST_N})"
  done
  IMPLEMENT_ISSUE="$NEWEST_N"
elif [ "$BUMP_COUNT" -eq 1 ]; then
  IMPLEMENT_ISSUE=$(echo "$BUMP_ISSUES" | jq -r '.[0].number')
fi
echo "Implement issue: ${IMPLEMENT_ISSUE:-none}"
```

---

## Phase 6 — Commit digest updates and open PR

Only run if `$COMPOSE_PINS_UPDATED` or `$YAML_PINS_UPDATED` is non-empty:

```bash
[post-mrg PR#${PR_NUMBER} $(date +%H:%M:%S)] COMMIT: opening digest-update PR  [ctx: ~16K | compress=0 | msgs=8]

git checkout main && git pull --rebase origin main
DIGEST_BRANCH="chore/post-merge-digest-pr${PR_NUMBER}"
git checkout -b "$DIGEST_BRANCH"

STAGE_FILES=""
[ -n "$COMPOSE_PINS_UPDATED" ] && STAGE_FILES="$STAGE_FILES infra/docker-compose/docker-compose.services.yml"
[ -n "$YAML_PINS_UPDATED" ]    && STAGE_FILES="$STAGE_FILES images/images.yaml"
git add $STAGE_FILES

CLOSE_LINE="${IMPLEMENT_ISSUE:+Closes #${IMPLEMENT_ISSUE}}"

git commit -s -m "chore(ci): update digest pins post-merge of PR #${PR_NUMBER}

Updated:$(echo $COMPOSE_PINS_UPDATED $YAML_PINS_UPDATED | xargs).

${CLOSE_LINE}

Assisted-by: Claude/claude-sonnet-4-6"

git push -u origin "$DIGEST_BRANCH"

[post-mrg PR#${PR_NUMBER} $(date +%H:%M:%S)] PR: creating digest-update PR  [ctx: ~17K | compress=0 | msgs=9]

DIGEST_PR=$(gh pr create \
  --title "chore(ci): update digest pins post-merge of PR #${PR_NUMBER}" \
  --body "Post-merge digest update triggered by PR #${PR_NUMBER}.
Updated: $(echo $COMPOSE_PINS_UPDATED $YAML_PINS_UPDATED | xargs).
Stale closed: ${STALE_ISSUES:-none}. Implements: ${IMPLEMENT_ISSUE:+#$IMPLEMENT_ISSUE}.
Assisted-by: Claude/claude-sonnet-4-6" | tail -1)

gh pr merge "$DIGEST_PR" --squash --auto
echo "Digest PR: $DIGEST_PR"

[post-mrg PR#${PR_NUMBER} $(date +%H:%M:%S)] CI_WAIT: polling digest PR CI  [ctx: ~18K | compress=0 | msgs=10]

DEADLINE=$(($(date +%s) + 600))
while [ "$(date +%s)" -lt "$DEADLINE" ]; do
  [ "$(gh pr view "$DIGEST_PR" --json state --jq .state)" = "MERGED" ] && break
  sleep 30
done
```

---

## Phase DONE / SKIP — Mandatory evidence block

```
[post-mrg PR#${PR_NUMBER} $(date +%H:%M:%S)] DONE: post-merge verify complete  [ctx: ~XK | compress=C | msgs=M]

## Post-Merge Evidence — PR #${PR_NUMBER}

| Field | Value |
|-------|-------|
| Merge commit  | ${MERGE_SHA} |
| Session date  | ${SESSION_DATE} |
| Story issue   | #${ISSUE_NUMBER} |
| Status        | DONE / SKIP |

### Workflow runs
| Workflow     | Run ID           | Conclusion |
|--------------|------------------|------------|
| Release      | ${RELEASE_RUN:-N/A} | success / N/A |
| tools-image  | ${TOOLS_RUN:-N/A}   | success / N/A |

### Images verified (GHCR)
| Service | Digest (prefix) | Status |
|---------|----------------|--------|
| <svc>   | sha256:xxxx... | ✓ |

### Digest pins updated
| File | Services | Action |
|------|----------|--------|
| docker-compose.services.yml | ${COMPOSE_PINS_UPDATED:-none} | updated / current |
| images/images.yaml          | ${YAML_PINS_UPDATED:-none}   | updated / current |

### Digest-bump issues
| Action       | Issues |
|--------------|--------|
| Closed (stale) | ${STALE_ISSUES:-none} |
| Implemented    | ${IMPLEMENT_ISSUE:+#$IMPLEMENT_ISSUE} |
| Digest PR      | ${DIGEST_PR:-none} |

### Context
| Metric      | Value |
|-------------|-------|
| ctx_initial | ~10K  |
| ctx_final   | ~XK   |
| compress    | C     |
| msgs        | M     |
```

---

## Session Learnings

Always end your response with this block:

```
## Session Learnings
- domain: ci-release
- expert: post-mrg
- pr: #${PR_NUMBER}
- issue: #${ISSUE_NUMBER}
- date: ${SESSION_DATE}

### Effective patterns
- <pattern>: <why it worked>

### Edge cases discovered
- <what>: <resolution>

### Failed approaches
- <what>: <why it failed>

### Proposed expert prompt update
- Rule: <exact text>
  Reason: <why permanent>
```
