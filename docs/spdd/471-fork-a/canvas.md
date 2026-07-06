<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M8.B Post-M8 Process Transition: Fork A + CNCF Sandbox Submission

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #471
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-18
**Status:** Implemented

**Child issues:** #496 (ROADMAP Fork A positioning)

> **Delivered (2026-07-06, M8.B) — with two DoD items synced to repo reality:**
> - README §1 Fork A/wedge framing: already delivered by the M7 positioning pass
>   (docs/product/positioning.md is the canonical rule; README leads with
>   "write once — run on Temporal or Argo").
> - ROADMAP: wedge framing + engine-portability conformance section + M9
>   conformance-suite milestone + honest version plan (v0.7.0 for M8; v1.0.0
>   reserved for CNCF acceptance).
> - **Sync note:** the original DoD said "conformance on Temporal and
>   LangGraph" — LangGraph was superseded by **Argo** as the second engine
>   (ADR-015; M6 ArgoEngine). The standing conformance evidence is the
>   dual-engine e2e matrix (same manifests through temporal + argo legs on
>   every infra/service PR); formalising it into a *named* suite is the M9
>   roadmap item.
> - ADR-019 amended: Canvas required for multi-PR feat: epics, exempt for
>   single-PR feat: (AGENTS.md/CLAUDE.md/pr-checks message updated in the
>   same diff).
> - CNCF Sandbox application + landscape entry PREPARED at
>   docs/cncf/sandbox-submission.md — **filing is a maintainer action**,
>   surfaced on #471; not claimed as done.

---

## R — Requirements

**Problem:** The 2026-05 architectural review (§23) identifies two divergent long-term forks and recommends Fork A — "The Honest YAML Layer": Zynax as the best declarative YAML manifest layer for hybrid workflow engines, with semantic equivalence proved across Temporal and LangGraph on a conformance suite. The current ROADMAP and README use "Kubernetes of AI workflows" framing that is overreaching for a project at this stage. Filing a CNCF TOC submission is the gating event for M8 completion.

**Definition of done:**
- README §1 uses Fork A framing.
- ROADMAP.md includes a conformance suite milestone.
- At least 2 workflows pass a conformance check on both Temporal and LangGraph.
- CNCF TOC submission PR filed (link in #471).
- ADR-019 amended: SPDD Canvas is optional, not mandatory, for `feat:` PRs.

---

## E — Entities

- **Fork A** — "The Honest YAML Layer" positioning: one YAML manifest runs on any supported workflow engine with identical observable `StateTransitionEvent` output.
- **Conformance suite** — 20 `.feature` files asserting byte-identical state-transition event sequences across Temporal and LangGraph for the same input YAML.
- **LangGraph engine adapter** — second engine adapter implementing `WorkflowEngine` interface (M5 adapter work); required before conformance can be demonstrated.
- **CNCF TOC submission** — a PR to `cncf/toc` using the Sandbox project proposal template; requires MAINTAINERS.md (M8.A), 2+ maintainers, and community signals.
- **ADR-019 amendment** — changes the SPDD mandate from "required for all `feat:` PRs" to "recommended for complex features; optional otherwise".

---

## A — Approach

**What we WILL do:**
- Update README and ROADMAP to Fork A framing and the 90-day strategic roadmap from review §22.
- Define the 20-workflow conformance suite as `.feature` files.
- Amend ADR-019 to make SPDD optional for external contributors.
- File the CNCF Landscape + TOC submission when community prerequisites are met (M8.A).

**What we WON'T do:**
- Pursue Fork B (full agent platform) — that requires a funded team and is out of scope.
- Remove SPDD entirely — it remains useful for complex internal features.
- File the CNCF submission before M8.A prerequisites are met (MAINTAINERS.md, 2+ maintainers, community signals).

**ADR references:**
- ADR-019: SPDD prompt governance — this canvas amends it; the amendment is a `docs:` PR opening a new ADR-020.
- ADR-015: Pluggable workflow engines — Fork A depends on this abstraction being honest (already verified).

---

## S — Structure

**Files modified:**
- `README.md` — Fork A positioning in §1
- `ROADMAP.md` — conformance suite milestone; 90-day strategic plan from review §22
- `docs/adr/ADR-019-spdd-prompt-governance.md` — amend mandate to optional
- `docs/adr/INDEX.md` — update ADR-019 entry

**New files:**
- `protos/tests/features/conformance/` — 20 conformance scenario `.feature` files

---

## O — Operations

1. **[#496]** Update README and ROADMAP to Fork A framing; amend ADR-019 to optional SPDD.
2. Define 20-workflow conformance suite as `.feature` files (separate PR, no canvas needed — `test:` type).
3. File CNCF Landscape PR and TOC submission (external action, no repo change for the filing itself).

---

## N — Norms

- `docs:` PR type for README/ROADMAP updates and ADR amendment.
- `test:` PR type for conformance `.feature` files.
- `make gitleaks` must pass on all changes.
- ADR-019 amendment must preserve the Canvas tooling and slash commands — only the mandate level changes.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed

### Feature Safeguards
- Never file the CNCF TOC submission until M8.A prerequisites are met (MAINTAINERS.md with 2+ orgs, active community signals).
- Never re-add the Sandbox Candidate badge until the TOC acknowledges the submission.
- ADR-019 amendment must not break existing Canvas tooling — only the enforcement level changes.
