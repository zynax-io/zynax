# /spdd-prompt-update

Incrementally update a REASONS Canvas when requirements change mid-implementation. Updates the Canvas before any code changes.

## Instructions

Read the Canvas at $ARGUMENTS. Then receive the requirement change description (either from the current session context or ask the user to describe it).

1. Identify which REASONS sections are invalidated by the change:
   - New or changed business rules → R (Requirements), O (Operations)
   - New entities or changed relationships → E (Entities)
   - Strategy change → A (Approach)
   - New files/services touched → S (Structure)
   - New cross-cutting standards → N (Norms)
   - New constraints or relaxed constraints → S (Safeguards)

2. For each invalidated section, propose the updated text

3. Highlight the diff between old and new in a clear before/after format

4. Check that the updated Canvas remains Tier 1 (no sensitive content introduced by the change)

5. Update Canvas status to `Draft` (it needs re-alignment after a requirements change)

6. Run /spdd-security-review on the updated Canvas

## Output Format

**Affected sections:** R, O (example)

**Before → After for each section:**
Show only the changed lines with context.

**Alignment needed:** List the specific decisions the human must confirm before the Canvas returns to `Aligned` status.

## Fundamental Rule

Requirements change → Canvas update → THEN code change. Never the reverse.

## Input

$ARGUMENTS — path to canvas.md (e.g., docs/spdd/205-spdd/canvas.md)
