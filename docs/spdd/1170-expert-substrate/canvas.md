# REASONS Canvas — EPIC X: Expert-Agent Substrate + agents/examples

> Tier 1 (public-safe). Tier 2 → `canvas.private.md`. Run `/spdd-security-review` before committing.

**Issue:** #1170 · **Milestone:** M7 (v0.6.0)
**Author:** M7 program plan · **Date:** 2026-06-15 · **Status:** Aligned

---

## R — Requirements
- **Problem:** `agents/examples/` does not exist; there is no canonical "write-your-own-agent"
  reference and no runtime **expert** pattern. The brief wants experts on **two substrates**:
  runtime AgentDef agents (in-workflow) and Claude Code experts (authoring loop).
- **Done when:** `agents/examples/` builds/lints/tests; a runtime expert (`go-review-expert`)
  registers and is dispatchable in a workflow with a trace; every authoring expert declares its
  runtime mapping.

## E — Entities
```
agents/examples/{echo, summarizer, go-review-expert}  ← reference agents on the SDK
RuntimeExpert (kind: AgentDef)                          ← registered + dispatched + OTEL-traced
AuthoringExpert (automation/workflows/experts/*.yaml)   ← Claude Code delivery-loop experts
ExpertMapping table                                     ← runtime ↔ authoring (or authoring-only)
```

## A — Approach
**We will:** create `agents/examples/` with three SDK-based reference agents; define a runtime
expert pattern (`kind: AgentDef` template + capability schema + registration); map each authoring
expert to its runtime counterpart (or mark "authoring-only"); wire `make lint-agent`/`test-unit-agent`
to discover `agents/examples/*`.
**We will NOT:** ship the full 14-expert library or RAG/memory strategies — **deferred to M-dx**.
**Governing ADRs:** ADR-033 (expert substrate — this EPIC), ADR-010 (pluggable agent runtime), ADR-013 (adapter-first).

## S — Structure (first S)
```
agents/examples/echo/ · summarizer/ · go-review-expert/   ← pyproject + handler + tests
agents/sdk/                                                 ← expert helpers (capability schema)
spec/workflows/examples/agent-def-expert.yaml               ← runtime expert AgentDef template
automation/workflows/experts/*.yaml                         ← add `runtime_mapping:` field
Makefile (lint-agent/test-unit-agent discovery)             ← glob agents/examples/*
docs/experts/                                               ← authoring guide + mapping table
```
Config: Python 3.12 + uv (ADR-002/003).

## O — Operations (stories — `spdd-story` form)

**GitHub issues:** X.1 #1201 · X.2 #1202 · X.3 #1203 · X.4 #1204 · X.5 #1205 (epic #1170)
**X.1 — ADR: expert substrate + mapping** · S · `adr-proposal`
- As a `maintainer`, I want the dual-substrate model + mapping recorded so runtime/authoring don't drift.
- AC: [ ] ADR-033 committed (runtime AgentDef + authoring expert, mapping table, drift guard). Deps: none.

**X.2 — `agents/examples/` reference agents** · M · `feat`
- As an `agent author`, I want canonical SDK examples so I can copy a working pattern.
- AC: [ ] `echo`, `summarizer`, `go-review-expert` build/lint/test; [ ] each has a capability schema + BDD. Deps: X.1.

**X.3 — Runtime expert pattern (AgentDef)** · M · `feat`
- As a `workflow author`, I want a registerable expert so an expert can run inside a workflow.
- AC: [ ] `kind: AgentDef` expert template; [ ] registers in agent-registry; [ ] dispatchable + OTEL-traced. Deps: X.2, O.6.

**X.4 — Make discovery for examples** · XS · `ci`
- As a `contributor`, I want CI to lint/test `agents/examples/*` so they stay green.
- AC: [ ] `make lint-agent AGENT=` + `test-unit-agent AGENT=` discover examples; [ ] CI runs them. Deps: X.2.

**X.5 — Authoring↔runtime mapping** · S · `docs`/`feat`
- As a `maintainer`, I want each authoring expert mapped to a runtime expert so the system is coherent.
- AC: [ ] `runtime_mapping:` added to `experts/*.yaml`; [ ] CI check that every authoring expert declares it; [ ] mapping table in docs. Deps: X.1.

**Order:** X.1 → X.2 → {X.3, X.4, X.5}.

## N — Norms
- Python: ruff + mypy clean; ≥90% coverage on new agent code; uv-managed (ADR-002/003).
- `Signed-off-by:` + `Assisted-by:`; `.claude/commands` files need `git add -f`; no literal emails (gitleaks PII gate).

## S — Safeguards (second S)
### Context Security
- [x] No Tier 2 content; [x] no PII / no literal emails; [x] no prompt-injection; [x] `/spdd-security-review` — PASS (2026-06-16, see `SECURITY-REVIEW.md`)

### Feature Safeguards
- Never grant an expert broader capability scope than declared — least-privilege per capability.
- Never let runtime and authoring experts drift silently — the mapping table is the SoT (CI-checked).
- Never require the SDK for agents — examples use it but adapters remain SDK-optional (ADR-013).
