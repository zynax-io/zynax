<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M8.A Governance and Community Readiness

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #470
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-18
**Status:** Aligned

**Child issues:** #494 (MAINTAINERS.md + GOVERNANCE.md simplification) · #495 (troubleshooting guide)

---

## R — Requirements

**Problem:** The 2026-05 architectural review (§16) scores community health at 2.5/10. `MAINTAINERS.md` does not exist. `GOVERNANCE.md` (451 lines) describes supermajority voting and RFC processes for an organisation of 1 person. There are no external contributors, no troubleshooting guide, and no curated entry points for first-time contributors. CNCF Sandbox requires 2+ maintainers from different organisations — impossible to satisfy without active community building.

**Definition of done:**
- `MAINTAINERS.md` exists listing current maintainer(s) with GitHub handle and affiliation.
- `GOVERNANCE.md` deferred sections clearly marked "when >5 contributors".
- `docs/troubleshooting.md` covers ≥5 known failure modes with diagnosis and fix.
- 5 `good first issue` issues open with complete acceptance criteria.
- CNCF Landscape PR filed.

---

## E — Entities

- **`MAINTAINERS.md`** — CNCF-template maintainer list; required for Sandbox submission.
- **`GOVERNANCE.md`** — simplified to defer supermajority/RFC/lazy-consensus sections.
- **`docs/troubleshooting.md`** — new file covering known failure modes from the architectural review.
- **`good first issue` label** — existing label; applied to 5 new issues with bounded, documented scope.
- **CNCF Landscape** — `cncf/landscape` GitHub repo; PR to add Zynax to the landscape.

---

## A — Approach

**What we WILL do:**
- Create `MAINTAINERS.md` following the CNCF project-template format.
- Add "deferred — requires 5+ contributors" gates to GOVERNANCE.md's supermajority, RFC, and lazy-consensus sections.
- Write `docs/troubleshooting.md` drawing on the architectural review's failure mode catalog.
- Write 5 bounded, well-documented `good first issue` tasks.
- File the CNCF Landscape PR.

**What we WON'T do:**
- Remove the SPDD methodology (deferred to M8.B per review §16.4 recommendation).
- Recruit external maintainers directly (that happens via community engagement, not code changes).

**ADR references:**
- ADR-018: AI knowledge base authorization model — changes to AGENTS.md and CLAUDE.md require gitleaks-ai-context gate.
- ADR-019: SPDD prompt governance — discussed for deferral in M8.B canvas.

---

## S — Structure

**New files:**
- `MAINTAINERS.md`
- `docs/troubleshooting.md`

**Modified files:**
- `GOVERNANCE.md` — deferred-section annotations
- `docs/local-dev.md` — cross-link to troubleshooting guide

---

## O — Operations

1. **[#494]** Create `MAINTAINERS.md`; annotate `GOVERNANCE.md` deferred sections.
2. **[#495]** Write `docs/troubleshooting.md` with ≥5 failure modes; cross-link from `docs/local-dev.md`.
3. Open 5 `good first issue` tasks with full acceptance criteria (no canvas needed — `docs:` or `chore:` type).
4. File CNCF Landscape PR (no repo change — external action).

---

## N — Norms

- `docs:` PR type for all changes in this EPIC.
- `make gitleaks` must pass — no email addresses or credentials in new docs.
- `MAINTAINERS.md` must follow the CNCF template format for Sandbox submission compatibility.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content
- [x] No PII (MAINTAINERS.md lists GitHub handles and organisation names — public information)
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed

### Feature Safeguards
- Never list internal infrastructure details (hostnames, IPs, credentials) in any new documentation.
- `good first issue` tasks must have bounded scope (≤ 200 LOC) — not open-ended research tasks.
- CNCF Landscape PR must not re-add the Sandbox Candidate badge until the TOC acknowledges the submission.
