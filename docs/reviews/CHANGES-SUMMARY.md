<!-- SPDX-License-Identifier: Apache-2.0 -->

# Architecture Overhaul — Changes Summary

**Branch:** `docs/architecture-overhaul-m5`  
**Date:** 2026-05-21  
**Tracking issue:** [#624](https://github.com/zynax-io/zynax/issues/624)  
**Source review:** `docs/architecture/2026-05-20-principal-architect-review.md`

---

## Files Created

### Review Artifacts (`docs/reviews/`)

| File | Purpose |
|------|---------|
| `00-inventory.md` | Repo map, CI workflow list, doc artifact census, AI context budget (948 lines / 2,000 budget), 10 known discrepancies D1–D10 |
| `01-decision-ledger.md` | All 19 ADRs with status, code-reflects evidence, rejected directions |
| `02-reality-vs-docs.md` | 23 reality-vs-docs items: 4 critical, 8 high, 6 medium, 5 low |
| `03-m5-state.md` | M5 snapshot: 7 DoD criteria (2.5/7 met), track status, critical-path chain |
| `04-architecture-gaps.md` | G1–G24 / H1–H9 / R1–R9 verified against HEAD; 7 new gaps NEW-1–NEW-7; ranked P0–P4 |
| `05-action-plan.md` | Reconciliation table (gaps → issues), new issues A3/A4/G1, dependency-ordered milestone re-plan |
| `CHANGES-SUMMARY.md` | This file |
| `DECISIONS-NEEDED.md` | Open decisions requiring human sign-off |

### Engineering Best Practices (`docs/engineering/best-practices/`)

| File | Key patterns |
|------|-------------|
| `go.md` | Service layout, context propagation, `crypto/subtle.ConstantTimeCompare`, `ReadHeaderTimeout`, `log/slog`, table-driven tests, gRPC deadlines with `context.WithTimeout` |
| `python.md` | `pyproject.toml` / `uv`, mypy --strict, `Agent` base class usage, Pydantic Settings, async gRPC patterns, bandit SAST rules |
| `dockerfiles.md` | Multi-stage template, distroless vs Alpine, HEALTHCHECK directive, Go toolchain alignment, `.dockerignore`, no secrets in layers |
| `github-ci.md` | SHA-pinned actions, least-privilege permissions, concurrency groups, required CI gates table, ci-runner container usage |
| `architecture-patterns.md` | Hexagonal arch, WorkflowEngine strategy, capability routing, BDD-first workflow, idempotent apply, Fowler event taxonomy, outbox pattern, circuit-breaker pattern |

### Milestone Document

| File | Purpose |
|------|---------|
| `docs/milestones/M5-engineering-review.md` | Live M5 review: track completion status, what was built, architecture assessment, DoD progress, exit criteria checklist |

---

## Files Modified

| File | Change | Why |
|------|--------|-----|
| `ARCHITECTURE.md` | **Complete rewrite** (was M1-era, showing M2 as "Next") | D1 — most critical discrepancy; 4 milestones behind reality |
| `AGENTS.md` | Fixed 3 broken knowledge-base links; added 13 new index entries for review docs and best practices | D3 — broken links; make new docs discoverable to agents |
| `SECURITY.md` | Fixed broken `AGENTS.md §7` cross-reference | D7 — broken anchor link |
| `README.md` | Corrected M3 description (CloudEvents stub caveat, cel-go added); fixed M4 (removed false agent-registry routing claim); added dispatch-not-wired warning above quickstart | D2, D4 — false capability claims |
| `ROADMAP.md` | Replaced empty M3/M4 checklists with accurate delivered/not-delivered lists; M5 section updated with per-track status | D6 — empty checklists showed no delivery history |
| `CLAUDE.md` | Corrected "CEL guards" → "cel-go guards"; added CloudEvents stub caveat | D8 — imprecise AI instructions |
| `docs/milestones/M5-plan.md` | Rev 31: added BATCH 6 (security hardening); added #622/#623/#624/#466 to gaps table; fixed blocked/parking section; updated header | Align plan with review findings and new issues |

---

## GitHub Actions

### New Issues Created

| Issue | Title | Milestone |
|-------|-------|-----------|
| [#622](https://github.com/zynax-io/zynax/issues/622) | fix(services): add context.WithTimeout to all outgoing gRPC calls | M5 |
| [#623](https://github.com/zynax-io/zynax/issues/623) | fix(api-gateway): refuse to start in production mode without ZYNAX_API_KEY | M5 |
| [#624](https://github.com/zynax-io/zynax/issues/624) | docs: architecture overhaul tracking issue | M5 |

### Milestone Changes

| Issue | Change | Rationale |
|-------|--------|-----------|
| [#466](https://github.com/zynax-io/zynax/issues/466) | M6 → M5 | Risk R4: OOM from unbounded IR store; 3–5 day effort; unblocks horizontal scale |

### Issue Cross-Links Added (comments)

Cross-link comments were added to: #567, #568, #569, #570, #571, #572, #574, #575, #576, #577, #579, #466  
Each comment links to the relevant gap in `docs/reviews/04-architecture-gaps.md` and to M5-plan.md batches.

### Milestone Description Updated

M5 "Adapter Library (M5)" description now references:
- `docs/milestones/M5-plan.md`
- `docs/milestones/M5-engineering-review.md`
- `docs/reviews/04-architecture-gaps.md`
- `docs/reviews/05-action-plan.md`

---

## Discrepancy Resolution Status (from `02-reality-vs-docs.md`)

| ID | Discrepancy | Status |
|----|------------|--------|
| D1 | ARCHITECTURE.md 4 milestones stale | ✅ Fixed (complete rewrite) |
| D2 | README M3 false CloudEvents claim | ✅ Fixed |
| D3 | AGENTS.md 3 broken knowledge-base links | ✅ Fixed |
| D4 | README M4 false agent-registry routing claim | ✅ Fixed |
| D5 | Quickstart missing dispatch-not-wired warning | ✅ Fixed |
| D6 | ROADMAP.md empty milestone checklists | ✅ Fixed |
| D7 | SECURITY.md broken AGENTS.md anchor link | ✅ Fixed |
| D8 | CLAUDE.md imprecise AI instructions | ✅ Fixed |
| D9 | Bearer non-constant-time (auth.go) | ⬜ Tracked (#567) |
| D10 | CloudEvents log stub not wired to NATS | ⬜ Tracked (known, M6+) |
