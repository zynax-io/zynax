# Security Review — EPIC X: Expert-Agent Substrate + agents/examples

**Canvas:** `docs/spdd/1170-expert-substrate/canvas.md` (Status: Draft → Aligned)
**Issue:** #1170 · **Date:** 2026-06-16 · **Reviewer:** SPDD Canvas expert (pre-alignment gate)
**Authority:** `docs/knowledge-base-policy.md` (Tier 1/2/3), ADR-019 (SPDD governance)

## Verdict: PASS

No Tier-2 content, prompt injection, abstraction leak, or authority violation found in the
canvas. No companion `canvas.private.md` exists or is required. None of the notes below block the
human alignment decision; they are implementation-time guards already bound as Feature Safeguards.

## Five-check results

| Check | Result | Notes |
|-------|--------|-------|
| 1. Tier-2 content scan | PASS | No real hostnames/IPs/TLDs/namespaces, no credentials, no deployment specifics, no OpSec. The paths (`agents/examples/`, `automation/workflows/experts/`, `spec/workflows/examples/`) are repo code paths, not infrastructure topology. |
| 2. Prompt-injection scan | PASS | All prose is human-facing user stories + ADR references; no AI-directed instructions, override attempts, or conditional triggers. |
| 3. Abstraction check | PASS | E (Entities) and O (Operations) describe code structure, intent, and patterns — not a specific environment. Nothing internal is inferable by a stranger. |
| 4. Authority hierarchy | PASS | N/S sections reinforce AGENTS.md (DCO, `Assisted-by`, gitleaks PII gate, least-privilege capability scope); no Canvas content contradicts or overrides an AGENTS.md rule. |
| 5. Completeness | PASS-with-note | All 7 REASONS sections present (R, E, A, S-structure, O, N, S-safeguards); Context Security checklist present; `**Status:**` valid. Status was `Draft` (expected — the human owns the align flip, performed alongside this review). |

## Notes (implementation-time guards — already bound as canvas Safeguards)

- **Least-privilege capability scope (X.3 #1203):** the runtime `kind: AgentDef` expert must never
  be granted broader capability scope than it declares — enforce least-privilege per capability at
  registration time.
- **No silent runtime↔authoring drift (X.5 #1205):** the mapping table is the SoT and must be
  CI-checked so a runtime expert and its authoring counterpart cannot diverge unnoticed.
- **SDK-optional (X.2 #1202):** `agents/examples/*` use the SDK, but adapters must remain
  SDK-optional per ADR-013 — examples may depend on the SDK, the platform contract may not.
- **PII hygiene:** agent examples and their tests/fixtures must not embed literal email addresses
  (gitleaks PII gate); reference commit-hygiene rules by name per AGENTS.md §Hard Constraints.
