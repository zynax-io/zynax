# /spdd-sync

Synchronise a REASONS Canvas with the current code after refactoring. Updates the Canvas to reflect implementation reality without changing logic.

## Instructions

Read the Canvas at $ARGUMENTS. Then read the implementation files listed in the Canvas **S — Structure** section.

1. Compare current implementation against Canvas O (Operations) and S (Structure):
   - Are the Operation steps still accurate? (Did any step get split or merged?)
   - Are the file paths in Structure still correct? (Did any file get moved or renamed?)
   - Did any new helper types or functions emerge that should be in E (Entities)?
   - Did any cross-cutting norm emerge from the refactor that should be in N (Norms)?

2. For each discrepancy, propose a Canvas update that documents the current reality

3. Do NOT change R (Requirements), A (Approach), or S (Safeguards) — those reflect intent, not implementation details. If they need to change, use /spdd-prompt-update instead.

4. Update Canvas status to `Synced` when complete

5. This command is ONLY for non-behavioural changes (refactoring). If the logic changed, use /spdd-prompt-update first.

## Output Format

**Discrepancies found:** N

For each discrepancy:
- **Section**: which REASONS section
- **Current Canvas says**: <text>
- **Implementation shows**: <reality>
- **Proposed update**: <new Canvas text>

**Canvas status after sync:** Synced

## Input

$ARGUMENTS — path to canvas.md (e.g., docs/spdd/205-spdd/canvas.md)
