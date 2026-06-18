# /lib:spdd-story

Break a raw feature description or GitHub issue into INVEST-compliant user stories,
then create one GitHub issue per story as a child of the parent epic.

## Instructions

Read the feature description or GitHub issue provided in $ARGUMENTS.

1. Identify the primary actors (who benefits from this feature)
2. Identify the core capability being requested
3. Identify the acceptance boundary (what is explicitly out of scope)
4. Break the work into 2–5 independent user stories following the INVEST principle:
   - **I**ndependent: can be delivered separately
   - **N**egotiable: details are flexible
   - **V**aluable: delivers observable value on its own
   - **E**stimable: can be sized relatively
   - **S**mall: fits in one PR (≤ 400 lines excluding generated code)
   - **T**estable: has clear, verifiable acceptance criteria
5. **Idempotency guard — never create duplicate stories.** Before creating anything,
   check whether the parent epic already has child issues (an epic created via
   `/lib:spdd-analysis`→issue-filing, or a re-run of this command, often already has them):
   - `gh issue list --search "<parent-#> in:body" --state all` and read the epic body's
     task-list / checklist for existing `#refs`.
   - If children already exist, **do NOT create new issues.** Instead: validate each
     existing child against INVEST, map it to an Operations step, ensure it carries the
     right labels + milestone + a canvas link, and report the reconciled set. Only create
     issues for genuinely missing stories.
6. After presenting the stories, create one GitHub issue **per still-missing story** using `gh issue create`.
   - Title format: `feat(<scope>): <story title> (#<parent-issue>, step <N>)`
   - Body includes: Story (as-a/I-want/so-that), Context (link to canvas if it exists), Scope, Acceptance criteria, Out of scope, Size estimate, Dependencies on other steps
   - End each body with: `Assisted-by: Claude/claude-sonnet-4-6`
   - Report the created (or reconciled) issue URL after each step

## Output Format

For each story (display first, then create issues):

**Story N: <title>**
- As a `<actor>`, I want `<capability>` so that `<outcome>`.
- Size estimate: XS / S / M / L
- Acceptance criteria:
  - [ ] <concrete, testable criterion>
  - [ ] <concrete, testable criterion>
- Out of scope: <what this story explicitly does NOT cover>

End with a recommended implementation order, any dependency between stories,
and a summary table of the created GitHub issues.

## Input

$ARGUMENTS — GitHub issue number, issue URL, or raw feature description
