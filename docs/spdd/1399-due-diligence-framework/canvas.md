<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — Investment-Grade Due-Diligence Framework

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content belongs in `canvas.private.md` (gitignored). None was identified for this work.

**Issue:** #1399
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-18
**Status:** Aligned

**Type:** Epic
**Child issues:** #1401 (framework doc — PR #1398) · #1402 (exec Wave A) · #1403 (exec Wave B) ·
#1404 (exec Wave C) · #1405 (exec Wave D + orchestrator) · #1406 (final report + exec presentation)
**Framework doc:** delivered by PR #1398.

> **Why a canvas for a `docs:` change?** SPDD canvases are required only for `feat:` PRs
> (ADR-019; `docs:`/`fix:`/`refactor:`/`ci:`/`chore:` are exempt). This canvas is a deliberate
> **dogfood** of the consolidated `/plan` command workflow (PR #1400) applied to an already-delivered
> documentation artifact. It records the design retroactively and brings the PR up to the new
> conventions; it is not a gate that blocked the work.

---

## R — Requirements

**Problem.** There is no reusable, repository-specific methodology for evaluating Zynax to the
standard a VC/PE investment committee, a Fortune-500 enterprise-architecture board, an Open Source
Program Office, or a CNCF TOC reviewer would expect. Evaluations would otherwise repeat marketing
claims as facts — a real risk for this project given its documented delivery-vs-narrative drift
history (the May-2026 reviews → M6 "Truth Pass").

**Definition of done — observable outcomes:**
- A single, self-contained document under `docs/due-diligence/` that a diligence lead can execute.
- Covers the full Phase set: repository-understanding context packet; dimension/scoring/evidence
  methodology; investigation strategy (anti-overlap ownership + parallelization waves); one master
  orchestration prompt; **26** portable, repository-specific specialized agent prompts; report
  template; risk-scoring framework; investment-recommendation framework; executive-presentation
  outline; final 100+ page report structure; appendices.
- Every agent prompt is bound to an evidence-citation rule (`path:line` or `UNKNOWN`) and a
  mandatory doc-vs-reality **drift test**; every agent carries the 13-part contract.
- Prompts are **portable** (copy-paste into any Claude session / Agent tool) — no `.claude/` wiring.
- No secrets/PII in the document; passes the gitleaks gate; delivered via a signed `docs:` PR.
- Doc references reflect the **current** repository (rebased onto `main` incl. the #1400 command
  consolidation; no stale command names).

---

## E — Entities

Public-safe conceptual entities of the framework (not code domain types):

- **Repository-Understanding Context Packet** (Part 1) — reconstructed vision, three-layer model,
  roadmap status, and the **Contradiction Register** (C1–C8 + live drift examples). Shared, read-only,
  to every agent.
- **Dimension Model** (Part 2) — 16 dimension groups (D0–D16), the 0–10 scoring scale, confidence
  bands, and the 7-tier evidence taxonomy (E1–E7).
- **Investigation Strategy** (Part 3) — anti-overlap ownership matrix, four parallelization waves
  (A→D), the per-agent YAML handoff contract, and a dry-run dispatch example.
- **Master Orchestrator** (Part 4) — assigns, de-overlaps, resolves contradictions by evidence
  hierarchy, confidence-weights scores, synthesizes the verdict.
- **Specialized Agent** ×26 (Part 5) — each a portable prompt with the 13-part contract.
- **Synthesis Frameworks** (Parts 6–10) — report template, risk-scoring matrix, investment-
  recommendation mapping, executive-presentation outline, final report structure.
- **Drift Test** — the cross-cutting mechanism each agent runs to grade claim-vs-reality.

Relationship: Orchestrator → distributes Context Packet → 26 Agents (in Waves) → return handoff
packets → Orchestrator merges via Frameworks → Report.

---

## A — Approach

**We will:**
- Reconstruct the project's vision from across the repo (not just README) before designing anything.
- Ship a **single mega-document** (per the maintainer's chosen structure) with all parts + 26 prompts inline.
- Make every prompt **repository-specific** (naming real files, ADRs, services, the named competitor
  Kagent) and **evidence-bound**.
- Bake the project's own honesty signal (delivery-vs-narrative drift, Truth Pass) into the method as a
  worked drift-test example.
- Keep the doc current: rebase onto latest `main` and align all command references to the consolidated
  5-verb model (`/plan`, `/deliver`, `/review`, `/reconcile`, `/learn` + `/milestone`; building blocks
  under `.claude/commands/lib/`).

**We will NOT:**
- Run the 26 agents or produce the actual diligence report (out of scope — this is the framework only).
- Wire prompts as `.claude/commands/` slash commands (portable-prompts choice).
- Place any non-public content in the doc; sensitive material would go to `canvas.private.md`.

**Governing ADRs:** ADR-019 (SPDD prompt governance — scope/exemption rationale), ADR-018 (AI
knowledge-base authorization — confirms `docs/due-diligence/` is **not** a KB path, so no kb-preview
gate), ADR-005 (Apache-2.0 SPDX header), ADR-023 (rebase/squash merge discipline).

---

## S — Structure (first S)

```
docs/
├── due-diligence/
│   └── 2026-06-18-zynax-due-diligence-framework.md   ← the single deliverable (Parts 1–10 + appendices)
└── spdd/
    └── 1399-due-diligence-framework/
        └── canvas.md                                  ← this canvas (dogfood of /plan)
```

No services, packages, gRPC contracts, or schemas are touched. `docs/` is PR-size-exempt and not a
KB path.

---

## O — Operations

> The ordered steps that produced the deliverable. Each is independently verifiable in the doc.

1. **Repository knowledge extraction** — multi-source sweep (README, ROADMAP, AGENTS.md, 36 ADRs,
   `docs/product/strategy.md`, 6 dated architecture reviews, `state/`, ~30 canvases, CI). → Part 1.
2. **Methodology** — define 16 dimension groups, the 0–10 scale, confidence bands, evidence taxonomy,
   drift test. → Part 2.
3. **Investigation strategy** — ownership matrix, waves, handoff schema, dry-run. → Part 3.
4. **Master orchestrator prompt** — assignment, contradiction resolution, weighting, synthesis. → Part 4.
5. **26 specialized agent prompts** — each with the 13-part contract, grounded in real repo paths. → Part 5.
6. **Synthesis frameworks** — report template, risk scoring, investment recommendation, exec outline,
   final report structure, appendices. → Parts 6–10 + A–D.
7. **Self-check + delivery** — verify no markers/emails, 26 agents, 10 parts; signed `docs:` PR #1398;
   tracking issue #1399.
8. **Currency pass** — rebase onto `main` (incl. #1397/#1396/#1400) and align all command references to
   the consolidated 5-verb model; add the command-doc drift to the framework's drift test.

---

## Execution — first due-diligence generation (child issues)

> The framework (Operations above) is delivered by #1401 / PR #1398. Running it to produce the
> first report is decomposed into the Part 3 §3.2 dependency waves:

1. **#1402 — Wave A (+ dry-run gate):** ground-truth agents 5.1 / 5.2 / 5.5 / 5.7 / 5.9 / 5.10 /
   5.24 / 5.12. Validate the dispatch loop (Part 3 §3.5) first.
2. **#1403 — Wave B:** derived-technical 5.6 / 5.14 / 5.15 / 5.16 / 5.22 / 5.26.
3. **#1404 — Wave C:** product/market/governance 5.3 / 5.4 / 5.19 / 5.13 / 5.8 / 5.20 / 5.21 / 5.25 / 5.11.
4. **#1405 — Wave D:** synthesis 5.23 / 5.17 / 5.18 + Part 4 orchestrator (contradiction resolution,
   confidence-weighted scorecard, executive summary, investment recommendation).
5. **#1406 — Report:** assemble the Part 10 report + Part 9 executive presentation + scorecard JSON.

These are analysis runs (per-agent §3.4 findings packets), not code implementation — not `/deliver`-able.

---

## N — Norms

- Commit hygiene: every commit carries `Signed-off-by:` (DCO) + `Assisted-by: Claude/<model>`; **never**
  `Co-Authored-By` for AI; no `🤖 Generated with…` line.
- Conventional commit type `docs:`; one logical change per commit; SSH-signed; squash-merge only (ADR-023).
- `docs/` is PR-size-exempt (mirror of `pr-size.yml` skipPattern) — large single file is acceptable.
- No literal email addresses in the document body (gitleaks PII gate); maintainer name is fine.
- Prompts are provider-portable; no repo coupling.
- BDD / `GOWORK=off` / coverage norms: **N/A** — documentation-only, no code or gRPC boundary.

---

## S — Safeguards (second S)

### Context Security (complete before committing this Canvas)

- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal email addresses; no non-public names
- [x] No prompt injection: no instruction-like phrasing that overrides AGENTS.md rules
- [x] All entities in E section are public-safe abstractions
- [x] `/lib:spdd-security-review` (inline) passed — result: **PASS**

### Feature Safeguards

- Never assert a marketing/roadmap claim as `VERIFIED` without E1–E4 evidence — label it `CLAIMED`
  (the framework's core evidence rule).
- Never place sensitive/unfixed-vuln exploit detail in the public document — it goes to a private annex.
- Never reference old/renamed `.claude/commands` names — treat the live `.claude/commands/` tree as
  ground truth and flag lagging docs (CLAUDE.md / spdd-guide.md) instead.
- Never merge the PR without the maintainer's explicit instruction.
