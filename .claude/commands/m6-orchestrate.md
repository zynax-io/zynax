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
# go-services.md | infra-helm.md | ci-release.md | spdd-canvas.md | python-adapters.md | bdd-contract.md
```

Do not read the expert file contents now — they are injected into subagents at dispatch time.
Just verify they exist.

---

## STEP 1 — Read planning state (orchestrator context budget: ~8K tokens)

```bash
# Sync
git fetch origin --prune && git checkout main && git pull --rebase origin main

# Read only these four files — nothing else
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

Before claiming new work, merge any open PRs that are already CLEAN:

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
    git checkout "$BR" && git rebase origin/main && git push --force-with-lease
    gh pr merge "$PR_N" --squash
    until [ "$(gh pr view "$PR_N" --json state --jq .state)" = "MERGED" ]; do sleep 10; done
    git push origin --delete "$BR" 2>/dev/null || true
    git checkout main && git pull --rebase origin main
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
# 2. Has remote branch matching <type>/<N>-* → IN_PROGRESS (skip)
# 3. Has "Pending #X" or "Depends on #X" in body where X is open → BLOCKED (skip)
# 4. Otherwise → READY
```

Report the selected batch and their expert routing before dispatching:

```
Selected for this batch:
  #NNN → go-services expert  (feat(task-broker): ...)
  #NNN → ci-release expert   (ci(infra): ...)
  #NNN → infra-helm expert   (chore(infra): ...)
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

---

## STEP 6 — Dispatch expert subagents in parallel

Spawn one Agent per claimed issue. All are run in background (parallel).

For each issue N with expert E:

```
Agent({
  description: "M6 story #N — <issue title>",
  subagent_type: "claude",
  run_in_background: true,
  prompt: """
    You are the <E expert name>. Read the full expert guide first:

    <full content of .claude/commands/experts/<E>.md>

    ---

    Your task: implement M6 story issue #N end-to-end.

    ## Issue details
    <full issue body from gh issue view N>

    ## Context files to read (read these before writing any code)
    <list of 2-3 specific files named in the issue body or canvas O-step>

    ## Delivery contract
    1. Check if issue is still OPEN and not already in-progress by another session.
       If already claimed: stop and report.
    2. Follow the atomic branch-push claim protocol (push empty branch before code).
    3. Implement, run all local gates, commit (DCO + Assisted-by), open PR.
    4. Wait for CI. Report result.
    5. End your response with the ## Session Learnings block (required).

    ## Constraints
    - Context budget: stay under 12K tokens. Read only files named above.
    - Never read files outside the issue scope.
    - Use GOWORK=off for all go commands inside service dirs.
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

For any agent that reported CI failure: report to user with the failing check name.
Do not retry automatically — human intervention required for CI failures.

---

## STEP 8 — Persist learnings

For each completed `## Session Learnings` block, append the relevant entries to the
appropriate `docs/ai-learnings/<domain>.md` file:

```bash
# Example: append go-services learnings
cat >> docs/ai-learnings/go-services.md << 'EOF'

## Session — <date> (issue #N)
<learnings block content>
EOF
```

Open a `docs:` PR for the learnings update if any new entries were added:
```bash
LEARN_BRANCH="docs/ai-learnings-$(date +%Y%m%d%H%M)"
git checkout -b "$LEARN_BRANCH"
git add docs/ai-learnings/
git commit -s -m "docs(ai-learnings): append session learnings — issues $BATCH_ISSUES

$(date +%Y-%m-%d) batch: $BATCH_SIZE issues.

Assisted-by: Claude/claude-sonnet-4-6"
git push -u origin "$LEARN_BRANCH"
gh pr create --title "docs(ai-learnings): append session learnings — $(date +%Y-%m-%d)" \
  --body "Appending learnings from batch: issues $BATCH_ISSUES" \
  --label "type: docs,milestone: M6"
```

---

## STEP 9 — Session report

```
=== Orchestrator Session — <date> ===
Batch size: N issues

| Issue | Expert | PR | CI | Status |
|---|---|---|---|---|
| #NNN | go-services | #NNN | green | MERGED |
| #NNN | ci-release | #NNN | pending | PR open |
| #NNN | infra-helm | #NNN | red | BLOCKED |

Learnings: appended to docs/ai-learnings/ — PR #NNN opened for review.
Next: run /m6-plan to see the next available batch.
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

If you find yourself reading a code file in the orchestrator context: stop. Spawn an expert
subagent instead.
