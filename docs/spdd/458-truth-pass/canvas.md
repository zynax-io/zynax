<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M5.A Truth Pass

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content belongs in `canvas.private.md` (gitignored).

**Issue:** #458
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-18
**Status:** Implemented

**Child issues:** #472 (remove CNCF badge + milestone status) · #473 (CHANGELOG audit) · #474 (Python SDK decision)

---

## R — Requirements

**Problem:** The 2026-05 external architectural review (`docs/architecture/2026-05-external-architectural-review.md` §1.3, §5.1) identifies a severe reality-claim divergence. README, CLAUDE.md, and AGENTS.md assert M1–M4 are "Complete" with v0.3.0 as the shipped version. In fact, 5 of 7 declared platform services have zero Go implementation (agent-registry, task-broker, memory-service, event-bus, and memory-service). The Python SDK is a 5-line empty `__init__.py`. Every `zynax apply` fails at capability dispatch. The "CNCF Sandbox Candidate" badge implies official CNCF status the project does not hold.

**Definition of done — observable outcomes:**
- No `shields.io/badge/CNCF-Sandbox` string anywhere in the repo.
- Milestone table in README.md and CLAUDE.md accurately marks M3 and M4 as partial with explanatory notes.
- CHANGELOG.md contains no entries referencing files or features that do not exist in `git ls-files`.
- Python SDK: a deliberate decision is documented and actioned (implement or remove from docs).
- `make gitleaks` passes on every changed file.

---

## E — Entities

- **README.md** — primary public face of the project; contains milestone status table and CNCF badge.
- **CLAUDE.md** — AI assistant context file; contains milestone status table referenced by automated tooling.
- **CHANGELOG.md** — release history; currently claims shipped features (Helm charts) that do not exist.
- **`agents/sdk/src/zynax_sdk/__init__.py`** — 5-line empty package claimed as a working Python SDK.
- **CNCF badge** — shields.io badge string in README.md:7 asserting "Sandbox Candidate" status.
- **Milestone status table** — appears in README.md and CLAUDE.md; lists M1–M4 as "Complete".

---

## A — Approach

**What we WILL do:**
- Remove the self-applied CNCF badge and replace with an honest signal (e.g. "Built with CNCF-graduated technologies").
- Update milestone status to reflect partial completion with clear "blocked by M5.C" notes.
- Audit CHANGELOG against `git ls-files`; remove or flag phantom entries.
- Make a deliberate documented decision on the Python SDK.

**What we WON'T do:**
- Implement task-broker or agent-registry here (that is M5.C #460).
- Change the M1 or M2 milestone status (those are correctly marked Complete).
- Remove the SPDD methodology or reduce CI gates.

**ADR references:**
- ADR-018: AI knowledge base authorization model — governs changes to CLAUDE.md and AGENTS.md.
- ADR-019: SPDD prompt governance — Canvas before code applies to `feat:` changes; this is a `docs:` EPIC.

---

## S — Structure

**Files touched:**
- `README.md` — badge removal, milestone table update
- `CLAUDE.md` — milestone table update
- `CHANGELOG.md` — phantom-entry removal / flagging
- `agents/sdk/src/zynax_sdk/__init__.py` — implement minimal Agent OR add "planned" notice
- `agents/sdk/pyproject.toml` — update description if SDK is deferred

**No gRPC contracts touched. No proto changes. No service code changes.**

---

## O — Operations

1. **[#472]** Remove CNCF Sandbox Candidate badge from README.md; update milestone status table in README.md and CLAUDE.md to reflect M3 ⚠ partial (no task-broker) and M4 ⚠ partial (no agent-registry). Add explanatory note.
2. **[#473]** Audit CHANGELOG.md against `git ls-files`; remove or flag entries referencing Helm charts and other unshipped features; add a v0.3.x-actual section documenting what was actually shipped.
3. **[#474]** Decision: implement minimal Python `Agent` base class (Option A, ~200 LOC) OR remove SDK promise from docs (Option B, ~10 LOC). Document the decision in the issue; action whichever option is chosen.

---

## N — Norms

- `docs:` PR type for all changes in this EPIC — no `feat:` or `fix:` needed.
- Every commit carries `Signed-off-by` and `Assisted-by: Claude/claude-sonnet-4-6` trailers per CLAUDE.md §AI attribution.
- `make gitleaks` must pass — no email addresses, private paths, or credentials in changed files.
- PR size ≤ 200 LOC per issue (these are documentation changes; all three fit in S/M).
- Changes to CLAUDE.md and AGENTS.md require the `gitleaks-ai-context` CI gate to pass (ADR-018).

---

## S — Safeguards

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards
- Never introduce claims about features that do not exist in `git ls-files`.
- Never reduce the CI gate count or weaken `make gitleaks` to pass a change.
- Never mark a milestone "Complete" unless the end-to-end path runs without error.
- Changes to ADR-018-governed files (CLAUDE.md, AGENTS.md) must pass the gitleaks-ai-context gate.
