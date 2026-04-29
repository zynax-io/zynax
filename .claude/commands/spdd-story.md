# /spdd-story

Break a raw feature description or GitHub issue into INVEST-compliant user stories.

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

## Output Format

For each story:

**Story N: <title>**
- As a `<actor>`, I want `<capability>` so that `<outcome>`.
- Size estimate: XS / S / M / L
- Acceptance criteria:
  - [ ] <concrete, testable criterion>
  - [ ] <concrete, testable criterion>
- Out of scope: <what this story explicitly does NOT cover>

End with a recommended implementation order and any dependency between stories.

## Input

$ARGUMENTS — GitHub issue number, issue URL, or raw feature description
