# Security Review — EPIC T: Reusable Templates + First Real Workflows

**Canvas:** `docs/spdd/1171-templates-real-workflows/canvas.md` (Status: Draft → Aligned)
**Issue:** #1171 · **Date:** 2026-06-16 · **Reviewer:** SPDD Canvas expert (pre-alignment gate)
**Authority:** `docs/knowledge-base-policy.md` (Tier 1/2/3), ADR-019 (SPDD governance)

## Verdict: PASS

No Tier-2 content, prompt injection, abstraction leak, or authority violation found in the
canvas. No companion `canvas.private.md` exists or is required. None of the notes below block the
human alignment decision; they are implementation-time guards already bound as Feature Safeguards.

## Five-check results

| Check | Result | Notes |
|-------|--------|-------|
| 1. Tier-2 content scan | PASS | No real hostnames/IPs/TLDs/namespaces, no credentials, no deployment specifics, no OpSec. The paths (`spec/templates/`, `spec/workflows/examples/`, `cmd/zynax/`, `docs/authoring/`) are repo code paths, not infrastructure topology. |
| 2. Prompt-injection scan | PASS | All prose is human-facing user stories + ADR references; no AI-directed instructions, override attempts, or conditional triggers. |
| 3. Abstraction check | PASS | E (Entities) and O (Operations) describe template mechanics, CLI commands, and workflow intent — not a specific environment. Nothing internal is inferable by a stranger. |
| 4. Authority hierarchy | PASS | N/S sections reinforce AGENTS.md (DCO, `Assisted-by`, one-logical-change-per-commit, `make validate-spec`, schema backward-compatibility); no Canvas content contradicts or overrides an AGENTS.md rule. |
| 5. Completeness | PASS-with-note | All 7 REASONS sections present; Context Security checklist present; `**Status:**` valid; O-section stories T.1–T.4 are created as issues #1206–#1209 and linked. Status was `Draft` (expected — the human owns the align flip, performed alongside this review). |

## Notes (implementation-time guards — already bound as canvas Safeguards)

- **No real workflow on unimplemented features (T.3 #1208):** the three real workflows
  (`code-review`, `ci-pipeline`, `feature-implementation`) must run on M7 capabilities only; they
  depend on EPIC W data-flow (W.5) and EPIC X experts (X.3) — do not ship a workflow that requires
  a not-yet-delivered capability.
- **No secrets in templates (T.1 #1206):** templates must parameterize credentials via
  inputs/secret-refs — never bake a literal secret or token into a `spec/templates/*` or
  `spec/workflows/examples/*` manifest (gitleaks PII/secret gate; reference commit-hygiene rules by
  name per AGENTS.md §Hard Constraints).
- **Schema backward-compatibility (T.1/T.2):** the `version:` field gates manifest-schema
  evolution; never break existing manifest schema compatibility — additive/versioned changes only.
