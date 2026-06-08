---
description: Expert knowledge synthesizer — reads docs/ai-learnings/*.md, spawns a Learning Synthesizer subagent to cluster patterns and propose expert prompt additions. Writes proposals to APPLY_LOG.md. Stops for human review before committing.
argument-hint: "[--domain go-services|ci-release|...] [--apply]  default: synthesize all"
---

# /m6-learn — Expert Knowledge Synthesizer

Read accumulated session learnings → spawn synthesizer subagent → output proposed additions
to expert prompt files → write pending entries to APPLY_LOG.md → stop for human review.

Two modes:
- **`/m6-learn`** (default) — synthesize and write proposals to APPLY_LOG.md as PENDING.
- **`/m6-learn --apply`** — apply PENDING entries already approved by the human in APPLY_LOG.md,
  then commit expert files + log update together.

> **This command never auto-commits in synthesis mode.** Human edits APPLY_LOG.md entries
> (`pending` → `applied` / `rejected`) then runs `/m6-learn --apply` to commit.

---

## APPLY mode (`/m6-learn --apply`)

Check if any PENDING entries in APPLY_LOG.md were approved (`applied`) by the human.
If yes: edit the expert files to add approved text, mark entries as `committed` in the log,
commit everything, open a PR, and stop.

```bash
LOG="docs/ai-learnings/APPLY_LOG.md"
[ ! -f "$LOG" ] && echo "No APPLY_LOG.md found — run /m6-learn first." && exit 1

# Find the most recent run with approved-but-not-yet-committed entries
APPROVED=$(python3 - "$LOG" <<'PYEOF'
import sys, re

log = open(sys.argv[1]).read()
# Find approved lines: | N | domain | text | sessions | applied | ... |
approved = re.findall(
    r'^\|\s*(\d+)\s*\|\s*(\S+)\s*\|\s*(.+?)\s*\|\s*(.+?)\s*\|\s*applied\s*\|\s*pending-commit\s*\|',
    log, re.MULTILINE
)
for row in approved:
    print('\t'.join(row))
PYEOF
)

if [ -z "$APPROVED" ]; then
  echo "No entries marked 'applied / pending-commit' in APPLY_LOG.md."
  echo "Edit APPLY_LOG.md: set Status='applied' and Delta='pending-commit' for entries to commit."
  exit 0
fi

echo "$APPROVED" | while IFS=$'\t' read -r NUM DOMAIN PATTERN SESSIONS; do
  echo "Applying: [$DOMAIN] $PATTERN (from $SESSIONS)"
done
```

Then: for each approved entry, the human has already edited the expert file.
Validate the expert files are modified, update APPLY_LOG.md entries to `committed`, commit:

```bash
LEARN_BRANCH="docs/expert-learnings-$(date +%Y%m%d%H%M)"
git checkout -b "$LEARN_BRANCH"
git add .claude/commands/experts/ docs/ai-learnings/APPLY_LOG.md
git commit -s -m "docs(automation): apply expert learnings — $(date +%Y-%m-%d)

Synthesized from session learnings via /m6-learn.
See docs/ai-learnings/APPLY_LOG.md for applied entries.

Assisted-by: Claude/claude-sonnet-4-6"
git push -u origin "$LEARN_BRANCH"
gh pr create \
  --title "docs(automation): apply expert learnings — $(date +%Y-%m-%d)" \
  --body "Applying approved /m6-learn proposals. See APPLY_LOG.md for details." \
  --label "type: docs,milestone: M6"
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
  description: "M6 learning synthesizer",
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

    ## Output format

    First, for each domain with worthwhile proposals:

    ### Domain: <name> → .claude/commands/experts/<name>.md

    #### Proposed addition to "<Section name>"
    ```
    - **<Pattern name>:** <description — one paragraph max>
      Seen in: #NNN, #NNN (N sessions). Recurrence: N.
    ```

    Then, output a APPLY_LOG_ENTRY block (used by /m6-learn to append to the log):

    ```apply-log
    ## Run <YYYY-MM-DD HH:MM> — domains: <list>

    | # | Domain | Pattern | Source sessions | Status | Delta |
    |---|--------|---------|-----------------|--------|-------|
    | 1 | go-services | <pattern name, ≤60 chars> | #NNN, #NNN | pending | — |
    | 2 | ci-release  | <pattern name, ≤60 chars> | #NNN       | pending | — |

    **Summary:** N proposed | 0 applied | 0 rejected | N pending
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
   - Defer:    leave Status as "pending" (will re-appear in next /m6-learn run)

3. After editing expert file(s) and APPLY_LOG.md, run:
   /m6-learn --apply
   This stages, commits, and opens a PR for the approved entries.

=== APPLY_LOG.md written to docs/ai-learnings/APPLY_LOG.md ===
```

---

## Learning loop lifecycle

```
/m6-orchestrate
  → expert sessions run → ## Session Learnings blocks produced
  → orchestrator appends raw learnings to docs/ai-learnings/<domain>.md
  → opens docs: PR for the raw learnings

After ≥3 new session blocks:
/m6-learn
  → reads learnings + expert files + APPLY_LOG.md
  → synthesizer clusters patterns (recurrence ≥2, dedup vs applied)
  → prints proposals
  → appends PENDING run entry to APPLY_LOG.md

Human reviews:
  → edits APPLY_LOG.md: pending → applied/rejected
  → manually edits expert files to add approved text

/m6-learn --apply
  → finds approved-but-not-committed entries in APPLY_LOG.md
  → updates log entries to committed
  → commits expert files + APPLY_LOG.md
  → opens docs: PR
```

---

## APPLY_LOG.md format reference

```markdown
# Expert Learning Apply Log

> Append-only. Each run of /m6-learn adds one entry. Human edits Status column.
> Delta column records what changed in the expert file (filled in by /m6-learn --apply).

---

## Run 2026-06-10 14:30 — domains: go-services, ci-release

| # | Domain | Pattern | Source sessions | Status | Delta |
|---|--------|---------|-----------------|--------|-------|
| 1 | go-services | Shell state reset (branch/bash) | #818, #826 | committed | go-services.md +8L |
| 2 | go-services | handler.go carried from prior step | #818 | rejected | — |
| 3 | ci-release | gh run list --commit not supported | #947 | committed | ci-release.md +4L |
| 4 | ci-release | make sync-images requires Docker | #947 | committed | post-merge.md +12L |

**Summary:** 4 proposed | 3 committed | 1 rejected | 0 pending
```

Status lifecycle: `pending` → `applied` (human sets) → `committed` (/m6-learn --apply sets)
Or: `pending` → `rejected` (human sets, stays rejected)

---

## What the synthesizer never does

- Read production code files
- Propose changes to AGENTS.md, CLAUDE.md, or ADRs (those require human decision)
- Auto-commit or push anything
- Propose deletions of existing expert content
- Propose changes based on a single session (recurrence rule: ≥2 sessions)
