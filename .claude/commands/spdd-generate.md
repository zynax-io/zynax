# /spdd-generate

Execute Canvas Operations step-by-step, generating code that strictly follows the Canvas Structure, Norms, and Safeguards.

## Instructions

Read the Canvas at $ARGUMENTS (path to canvas.md). Then:

1. Confirm the Canvas status is `Aligned` (not `Draft`) — refuse to generate from an unaligned Canvas
2. List all Operations steps and ask which step to execute (or start from step 1 if unambiguous)
3. For the selected step:
   a. Read all files listed in the Canvas Structure section that are relevant to this step
   b. Check the Safeguards section — if the step would violate any safeguard, halt and report
   c. Check the Norms section — apply all norms to the generated code
   d. Generate the code change for this step only (never generate multiple steps at once)
   e. Review the generated output against: layer boundaries (ADR-001), no panic in production, GOWORK=off for go commands, BDD feature file before implementation if touching a gRPC boundary
4. After generating, summarize: what was changed, which Canvas Operation step is now complete, what the next step is
5. Do NOT proceed to the next step automatically — wait for human review

## Safeguards Check (run before every step)

- Does this step require hardcoding an engine name? → Halt (ADR-015)
- Does this step add a new gRPC method without a .feature file? → Halt (ADR-016)
- Does this step import from another service's internal/? → Halt (ADR-008)
- Does this step embed Tier 2 context (hostnames, credentials) in code comments? → Halt

## Input

$ARGUMENTS — path to the aligned canvas.md (e.g., docs/spdd/205-spdd/canvas.md)
