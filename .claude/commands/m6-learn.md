---
description: Expert knowledge synthesizer — reads docs/ai-learnings/*.md, spawns a Learning Synthesizer subagent to cluster patterns and propose expert prompt additions. Always stops for human review before committing.
argument-hint: "[--domain go-services|infra-helm|ci-release|spdd-canvas|python-adapters|bdd-contract]  default: all"
---

# /m6-learn — Expert Knowledge Synthesizer

Read accumulated session learnings → spawn synthesizer subagent → output proposed additions
to expert prompt files → stop for human review.

> **This command never auto-commits.** All proposed changes are printed to the terminal.
> The human reviews, then commits approved additions manually or via a `docs:` PR.

---

## STEP 1 — Read all learning files

```bash
DOMAIN=${ARGUMENTS:-all}

if [ "$DOMAIN" = "all" ]; then
  DOMAINS="go-services infra-helm ci-release spdd-canvas python-adapters bdd-contract"
else
  DOMAINS="$DOMAIN"
fi

# Check all learning files exist
for D in $DOMAINS; do
  [ -f "docs/ai-learnings/$D.md" ] || { echo "Missing: docs/ai-learnings/$D.md"; exit 1; }
done

# Collect all content
for D in $DOMAINS; do
  echo "=== $D ==="
  cat "docs/ai-learnings/$D.md"
  echo ""
done
```

Count the learning entries to decide if synthesis is worthwhile:
```bash
ENTRY_COUNT=$(grep -c "^- \*\*" docs/ai-learnings/*.md 2>/dev/null || echo 0)
echo "Total learning entries across all domains: $ENTRY_COUNT"
[ "$ENTRY_COUNT" -lt 5 ] && {
  echo "Fewer than 5 entries — not enough data for synthesis yet."
  echo "Run /m6-orchestrate for a few more sessions first."
  exit 0
}
```

---

## STEP 2 — Read current expert files (for deduplication)

```bash
# Read all expert files so the synthesizer knows what's already there
for D in $DOMAINS; do
  # Map domain name to expert file name
  case "$D" in
    go-services)     F="go-services"    ;;
    infra-helm)      F="infra-helm"     ;;
    ci-release)      F="ci-release"     ;;
    spdd-canvas)     F="spdd-canvas"    ;;
    python-adapters) F="python-adapters" ;;
    bdd-contract)    F="bdd-contract"   ;;
  esac
  echo "=== expert: $F ==="
  cat ".claude/commands/experts/$F.md"
  echo ""
done
```

---

## STEP 3 — Spawn the Learning Synthesizer subagent

Spawn a fresh subagent with **only** the learning files + expert files content — no codebase
context, no issue history, no planning state.

```
Agent({
  description: "M6 learning synthesizer",
  subagent_type: "claude",
  prompt: """
    You are a learning synthesizer for the Zynax engineering AI system.

    You have been given:
    1. Accumulated learnings from expert sessions (docs/ai-learnings/*.md)
    2. Current content of each expert prompt file (.claude/commands/experts/*.md)

    Your task: identify patterns in the learnings and propose additions to the expert files.

    ## Rules

    - Only propose ADDITIONS — never deletions or rewrites of existing content.
    - Only propose a pattern if it appears in ≥2 separate sessions (recurrence rule).
    - Deduplicate: if a proposed addition is already covered by existing expert content, skip it.
    - Rank by value: start with patterns that prevented bugs or CI failures.
    - Maximum 5 proposed additions per domain.
    - Write each proposed addition in the exact format used by the target section of the expert file.
    - Cite the issues/sessions that support the proposal.

    ## Output format

    For each domain that has worthwhile additions:

    ```
    ### Domain: <name> → .claude/commands/experts/<name>.md

    #### Proposed addition to "Effective patterns"
    - **<Pattern name>:** <description — one paragraph max>
      Seen in: #NNN #NNN (N sessions). Recurrence: <count>.

    #### Proposed addition to "Edge cases discovered"
    - **<Edge case>:** <description>
      Resolution: <how to handle it>
      Seen in: #NNN. Recurrence: <count>.
    ```

    Only include sections where you have proposals. Do not output empty sections.
    End with a summary: N total proposed additions across M domains.

    ## The learning data

    <paste content of all learning files here>

    ## The current expert content (for deduplication)

    <paste content of all expert files here>
  """
})
```

---

## STEP 4 — Present proposals to human

Print the synthesizer output directly. Then print application instructions:

```
=== Learning Synthesizer Output ===
[synthesizer output]

=== How to apply approved additions ===

For each approved addition:
1. Open the target expert file: .claude/commands/experts/<name>.md
2. Add the text to the appropriate section (Effective patterns / Edge cases / Failed approaches)
3. Stage and commit:

   git checkout -b docs/expert-learnings-$(date +%Y%m%d)
   # edit the expert files
   git add .claude/commands/experts/
   git commit -s -m "docs(automation): apply expert learnings — <domain1> <domain2>

   Synthesized from N sessions via /m6-learn.

   Assisted-by: Claude/claude-sonnet-4-6"
   git push -u origin HEAD
   gh pr create --title "docs(automation): apply expert learnings — $(date +%Y-%m-%d)" \
     --body "Applying approved learning synthesizer output from /m6-learn run." \
     --label "type: docs,milestone: M6"

Do NOT apply all proposals blindly — reject any that:
- Contradict existing AGENTS.md rules
- Are too repo-specific to be generalisable (one-off workarounds)
- Duplicate content already in the expert file
```

---

## Learning loop lifecycle

```
/m6-orchestrate             — expert sessions run → ## Session Learnings blocks produced
                               → orchestrator appends to docs/ai-learnings/<domain>.md
                               → opens docs: PR for the appended raw learnings

After ≥5 new entries:
/m6-learn                   — synthesizer reads all learnings → proposes expert additions
                               → human reviews proposals
                               → human opens docs: PR with approved additions to expert files

After PR merged:
                              → expert files improved for next /m6-orchestrate batch
                              → context budget for next batch includes better expert knowledge
```

---

## What the synthesizer never does

- Read production code files
- Propose changes to AGENTS.md, CLAUDE.md, or ADRs (those require human decision)
- Auto-commit or push anything
- Propose deletions of existing expert content
- Propose changes based on a single session (recurrence rule: ≥2 sessions)
