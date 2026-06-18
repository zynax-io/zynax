---
description: Expert knowledge synthesizer — reads docs/ai-learnings/*.md, spawns a Learning Synthesizer subagent to cluster patterns and propose expert prompt additions. Writes proposals to APPLY_LOG.md. Stops for human review before committing.
argument-hint: "[--domain go-services|ci-release|...] [--apply]  default: synthesize all"
---

# /learn — Expert Knowledge Synthesizer

> **Milestone-agnostic.** Synthesize learnings on demand from `docs/ai-learnings/*.md`; nothing here
> depends on a milestone. `/deliver` and `/reconcile` append the raw session learnings this digests.

Read accumulated session learnings → spawn synthesizer subagent → output proposed additions
to expert prompt files → write pending entries to APPLY_LOG.md → stop for human review.

Two modes:
- **`/learn`** (default) — synthesize and write proposals to APPLY_LOG.md as PENDING.
- **`/learn --apply`** — apply PENDING entries already approved by the human in APPLY_LOG.md,
  then commit expert files + log update together.

> **This command never auto-commits in synthesis mode.** Human edits APPLY_LOG.md entries
> (`pending` → `applied` / `rejected`) then runs `/learn --apply` to commit.

---

## APPLY mode (`/learn --apply`)

Check if any PENDING entries in APPLY_LOG.md were approved (`applied`) by the human.
If yes: edit the expert files to add approved text, mark entries as `committed` in the log,
commit everything, open a PR, and stop.

```bash
LOG="docs/ai-learnings/APPLY_LOG.md"
[ ! -f "$LOG" ] && echo "No APPLY_LOG.md found — run /learn first." && exit 1

# Find the most recent run with approved-but-not-yet-committed entries
APPROVED=$(python3 - "$LOG" <<'PYEOF'
import sys, re

log = open(sys.argv[1]).read()
# Find approved lines: | N | domain | text | category | sessions | applied | pending-commit |
# (also tolerates legacy 6-column rows without the Category column)
approved = re.findall(
    r'^\|\s*(\d+)\s*\|\s*(\S+)\s*\|\s*(.+?)\s*\|\s*(domain|structural-workaround)\s*\|\s*(.+?)\s*\|\s*applied\s*\|\s*pending-commit\s*\|',
    log, re.MULTILINE
)
for row in approved:
    print('\t'.join(row))
PYEOF
)

if [ -z "$APPROVED" ]; then
  echo "No entries marked 'applied / pending-commit' in APPLY_LOG.md."
  echo "Edit APPLY_LOG.md: set Status='applied' and Delta='pending-commit' for entries to commit."
  echo "(structural-workaround entries are suppressed by default — leave them 'rejected')."
  exit 0
fi

echo "$APPROVED" | while IFS=$'\t' read -r NUM DOMAIN PATTERN CATEGORY SESSIONS; do
  echo "Applying: [$DOMAIN/$CATEGORY] $PATTERN (from $SESSIONS)"
done
```

Then: for each approved entry, the human has already edited the expert file.
Validate the expert files are modified, update APPLY_LOG.md entries to `committed`, commit:

```bash
LEARN_BRANCH="docs/expert-learnings-$(date +%Y%m%d%H%M)"
git checkout -b "$LEARN_BRANCH"
git add .claude/commands/experts/ docs/ai-learnings/APPLY_LOG.md
git commit -s -m "docs(automation): apply expert learnings — $(date +%Y-%m-%d)

Synthesized from session learnings via /learn.
See docs/ai-learnings/APPLY_LOG.md for applied entries.

Assisted-by: Claude/<model-id-of-this-session>"
git push -u origin "$LEARN_BRANCH"
gh pr create \
  --title "docs(automation): apply expert learnings — $(date +%Y-%m-%d)" \
  --body "Applying approved /learn proposals. See APPLY_LOG.md for details." \
  --label "type: docs" --label "$MILESTONE_LABEL"
```

**Stop here if mode is `--apply`.** Do not continue to synthesis steps.

---

## SYNTHESIZE mode (default)

### STEP 1 — Read APPLY_LOG to get already-applied patterns

```bash
LOG="docs/ai-learnings/APPLY_LOG.md"
ALREADY_APPLIED=""
if [ -f "$LOG" ]; then
  # Extract patterns already committed — skip these in synthesis
  ALREADY_APPLIED=$(grep -oP '(?<=\| \d{1,3} \| \S+ \| ).*?(?= \|)' "$LOG" \
    | grep -B1 'committed' || true)
  APPLIED_COUNT=$(grep -c '| committed |' "$LOG" 2>/dev/null || echo 0)
  echo "Patterns already applied: $APPLIED_COUNT"
fi
```

---

### STEP 2 — Read all learning files

```bash
DOMAIN=${ARGUMENTS:-all}

if [ "$DOMAIN" = "all" ]; then
  DOMAINS="go-services infra-helm ci-release spdd-canvas python-adapters bdd-contract"
else
  DOMAINS="$DOMAIN"
fi

for D in $DOMAINS; do
  [ -f "docs/ai-learnings/$D.md" ] || { echo "Missing: docs/ai-learnings/$D.md"; }
done

for D in $DOMAINS; do
  echo "=== $D ==="
  cat "docs/ai-learnings/$D.md"
  echo ""
done
```

Count entries to decide if synthesis is worthwhile:
```bash
ENTRY_COUNT=$(grep -c "^### " docs/ai-learnings/*.md 2>/dev/null || echo 0)
echo "Session blocks across all learning files: $ENTRY_COUNT"
[ "$ENTRY_COUNT" -lt 3 ] && {
  echo "Fewer than 3 session blocks — not enough data yet."
  exit 0
}
```

---

### STEP 3 — Read current expert files (for deduplication)

```bash
for D in $DOMAINS; do
  case "$D" in
    go-services)     F="go-services"    ;;
    infra-helm)      F="infra-helm"     ;;
    ci-release)      F="ci-release"     ;;
    spdd-canvas)     F="spdd-canvas"    ;;
    python-adapters) F="python-adapters" ;;
    bdd-contract)    F="bdd-contract"   ;;
    post-merge)      F="post-merge"     ;;
  esac
  [ -f ".claude/commands/experts/$F.md" ] || continue
  echo "=== expert: $F ==="
  cat ".claude/commands/experts/$F.md"
  echo ""
done
```

---

### STEP 4 — Spawn the Learning Synthesizer subagent

```
Agent({
  description: "Expert learning synthesizer",
  subagent_type: "claude",
  prompt: """
    You are a learning synthesizer for the Zynax engineering AI system.

    You have been given:
    1. Accumulated session learnings from docs/ai-learnings/*.md
    2. Current content of each expert prompt file
    3. A list of patterns already applied (from APPLY_LOG.md) — do NOT re-propose these

    ## Rules

    - Only propose ADDITIONS — never deletions or rewrites of existing content.
    - Recurrence rule: only propose a pattern seen in ≥2 separate sessions.
    - Deduplicate against existing expert content AND against already-applied list.
    - Rank by value: start with patterns that prevented bugs or CI failures.
    - Maximum 5 proposed additions per domain.
    - Write each proposed addition in the exact format used by the target section.
    - Cite the issues/sessions that support the proposal.
    - Classify every proposal as `domain`, `structural-workaround`, or `env-constraint`:
      - `structural-workaround` — only exists to survive a SHARED working tree: defensive
        `git checkout`/branch-verify before Bash calls, "never `git add .`", stash-avoidance,
        ref-lock recovery, cherry-pick rescue, "checkout target file to undo a sibling's edit".
        These are made unnecessary by worktree isolation → default `rejected`.
      - `env-constraint` — a RUNTIME/sandbox constraint that worktree isolation does NOT fix:
        e.g. background-subagent Bash denies compound/chained commands, shell state not
        persisting between calls (use literal paths, not vars), `env` prefix denied, multiline
        `-m` denied (use `git commit -F`), CI-wait must be `gh pr checks --watch` not a loop.
        These are REAL and must persist, but they belong in the **dispatch-prompt preamble of
        milestone-orchestrate.md / issue-deliver.md** (injected into every agent), NOT scattered
        into the expert guides. Do NOT auto-reject: surface as `pending` with a note that the
        fix is a manual command-file edit (outside `/learn --apply`'s expert-file scope).
      - `domain` — genuine engineering knowledge (API shape, query planner behaviour, proto
        field name, test pattern, gRPC code mapping).
      Prefer the `Category:` line the session already emitted; if absent, infer it.
    - Do NOT promote `structural-workaround` proposals to the expert guides while worktree
      isolation is in effect (milestone-orchestrate STEP 6 / STEP 7.5; EPIC #1001). List them under a
      separate "Structural (suppressed — root cause fixed by worktree isolation)" heading so
      the human can confirm, but default their Status to `rejected` in the apply-log.

    ## Output format

    First, for each domain with worthwhile proposals:

    ### Domain: <name> → .claude/commands/experts/<name>.md

    #### Proposed addition to "<Section name>"
    ```
    - **<Pattern name>:** <description — one paragraph max>
      Seen in: #NNN, #NNN (N sessions). Recurrence: N.
    ```

    Then, output a APPLY_LOG_ENTRY block (used by /learn to append to the log):

    ```apply-log
    ## Run <YYYY-MM-DD HH:MM> — domains: <list>

    | # | Domain | Pattern | Category | Source sessions | Status | Delta |
    |---|--------|---------|----------|-----------------|--------|-------|
    | 1 | go-services | <pattern name, ≤60 chars> | domain | #NNN, #NNN | pending | — |
    | 2 | go-services | <shared-tree workaround> | structural-workaround | #NNN, #NNN | rejected | — |

    **Summary:** N proposed | 0 applied | M rejected (structural) | P pending
    ```

    Only include sessions where you have proposals. End with the apply-log block always.

    ## The learning data

    <paste content of all learning files here>

    ## Already-applied patterns (skip these — do NOT re-propose)

    <paste APPLY_LOG.md content or "none" here>

    ## Current expert content (for deduplication)

    <paste content of all expert files here>
  """
})
```

---

### STEP 5 — Append run entry to APPLY_LOG.md

Extract the `apply-log` block from the synthesizer output and append it to
`docs/ai-learnings/APPLY_LOG.md`:

```bash
# The apply-log block from the synthesizer is fenced with ```apply-log ... ```
# Extract and append it (strip the fences)
SYNTHESIZER_LOG_BLOCK="<extracted apply-log block content>"

cat >> docs/ai-learnings/APPLY_LOG.md << EOF

$SYNTHESIZER_LOG_BLOCK
EOF

echo ""
echo "Run entry written to docs/ai-learnings/APPLY_LOG.md"
```

---

### STEP 6 — Present proposals and instructions

```
=== Learning Synthesizer Output ===
[synthesizer output — proposals only, not the log block]

=== How to apply ===

1. Review each proposal above.

2. Edit docs/ai-learnings/APPLY_LOG.md — for each proposal row:
   - Approve:  change Status from "pending" to "applied"
               change Delta from "—" to "pending-commit"
               then manually edit the target expert file to add the text
   - Reject:   change Status from "pending" to "rejected"
               leave Delta as "—"
   - Defer:    leave Status as "pending" (will re-appear in next /learn run)

3. After editing expert file(s) and APPLY_LOG.md, run:
   /learn --apply
   This stages, commits, and opens a PR for the approved entries.

=== APPLY_LOG.md written to docs/ai-learnings/APPLY_LOG.md ===
```

---

## Learning loop lifecycle

```
/deliver
  → expert sessions run → ## Session Learnings blocks produced
  → orchestrator appends raw learnings to docs/ai-learnings/<domain>.md
  → opens docs: PR for the raw learnings

After ≥3 new session blocks:
/learn
  → reads learnings + expert files + APPLY_LOG.md
  → synthesizer clusters patterns (recurrence ≥2, dedup vs applied)
  → prints proposals
  → appends PENDING run entry to APPLY_LOG.md

Human reviews:
  → edits APPLY_LOG.md: pending → applied/rejected
  → manually edits expert files to add approved text

/learn --apply
  → finds approved-but-not-committed entries in APPLY_LOG.md
  → updates log entries to committed
  → commits expert files + APPLY_LOG.md
  → opens docs: PR
```

---

## APPLY_LOG.md format reference

```markdown
# Expert Learning Apply Log

> Append-only. Each run of /learn adds one entry. Human edits Status column.
> Delta column records what changed in the expert file (filled in by /learn --apply).

---

## Run 2026-06-10 14:30 — domains: go-services, ci-release

| # | Domain | Pattern | Category | Source sessions | Status | Delta |
|---|--------|---------|----------|-----------------|--------|-------|
| 1 | go-services | Shell state reset (branch/bash) | structural-workaround | #818, #826 | rejected | — |
| 2 | go-services | handler.go carried from prior step | domain | #818 | rejected | — |
| 3 | ci-release | gh run list --commit not supported | domain | #947 | committed | ci-release.md +4L |
| 4 | ci-release | make sync-images requires Docker | domain | #947 | committed | post-merge.md +12L |

**Summary:** 4 proposed | 2 committed | 2 rejected (1 structural) | 0 pending
```

> The `Category` column is required (added under EPIC #1001). `structural-workaround` rows are
> defaulted to `rejected` while worktree isolation is in effect — they describe bandages the
> shared-tree root-cause fix made unnecessary, so they must not re-enter the expert guides.
> `env-constraint` rows (runtime/sandbox constraints worktree isolation does NOT fix — e.g.
> background-subagent compound-Bash denial, non-persistent shell state) are NOT auto-rejected:
> leave them `pending` and apply them by hand to the dispatch-prompt preamble in
> `milestone-orchestrate.md` / `issue-deliver.md` — they are outside `/learn --apply`'s
> expert-file scope, so `--apply` will not commit them.

Status lifecycle: `pending` → `applied` (human sets) → `committed` (/learn --apply sets)
Or: `pending` → `rejected` (human sets, stays rejected)

---

## What the synthesizer never does

- Read production code files
- Propose changes to AGENTS.md, CLAUDE.md, or ADRs (those require human decision)
- Auto-commit or push anything
- Propose deletions of existing expert content
- Propose changes based on a single session (recurrence rule: ≥2 sessions)
