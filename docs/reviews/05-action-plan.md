<!-- SPDX-License-Identifier: Apache-2.0 -->

# 05 — Action Plan & Backlog

**Date:** 2026-05-21  
**Branch:** docs/architecture-overhaul-m5  
**Purpose:** Phase 7 artifact — approved backlog for new/edited issues, milestone re-plan,
and dispositioned gaps from the 2026-05-20 principal architect review.

> **Approval status:** This plan reflects the reconciliation done in Phase 7 of the
> architecture overhaul. Issues are grouped by dependency order and milestone.

---

## Reconciliation Table (Review findings → existing issues)

| Finding | Issue | Status |
|---|---|---|
| G1: Bearer non-constant-time | #567 | ⬜ Open, M5 |
| G2: ReadHeaderTimeout | #568 | ⬜ Open, M5 |
| G3: Rate limiting | #580 | ⬜ Open, M6 |
| G4: No RetryPolicy | #569 | ⬜ Open, M5 |
| G5: Watch polling | #492 | ⬜ Open, M7 |
| G6: resolveTemplate | #584 | ⬜ Open, M6 |
| G7: mergePayload | #571 | ⬜ Open, M5 |
| G8: Action.Output | #581 | ⬜ Open, M6 |
| G9: CODEOWNERS | ~~#573~~ | ✅ Closed (file exists) |
| G10: compiler retention | #572 (doc) + #466 (code) | Open; #466 promote M6→M5 |
| G11: benchmarks | #493 | ⬜ Open, M7 |
| G12: fuzz tests | #539 | 🟡 Partial (cel-go seed) |
| G14/G15: hash ADR | #583 | ⬜ Open, M6 |
| G16: background context | #570 | ⬜ Open, M5 |
| G17/H9: stub services | #574 | ⬜ Open, M5 |
| G19: Kagent positioning | #575 | ⬜ Open, M5 |
| G20: pkg.go.dev | #582 | ⬜ Open, M6 |
| G21: Python SDK placeholder | #474 | ✅ Done (#535 #536 #537) |
| G22: summarizer phantom | #576 | ⬜ Open, M5 |
| G23: phantom AGENT_LIST | #577 | ⬜ Open, M5 |
| G24: compose missing services | #481 | ⬜ Open, M5 |
| H1: stateless compiler | #466/#490 | ⬜ Open, M6 → **promote to M5** |
| H2: OTel baseline | #491 | ⬜ Open, M7 → **consider M5/M6** |
| H3: TLS inter-service | #488 | ⬜ Open, M6 |
| H4: SBOM+cosign | #489/#465 | ⬜ Open, M6 |
| H5: bearer+header hardening | #567+#568 | ⬜ Open, M5 |
| H7: ADR-020 | #240 | ⬜ Open |
| H8: ADR-021 | #578 | ⬜ Open, M6 |
| H9: Unimplemented skeletons | #574 | ⬜ Open, M5 |
| README status table | #579 | ⬜ Open, M5 |

### Items identified NEW in this review (not yet filed)

| # | Gap | Action | Milestone |
|---|---|---|---|
| NEW-1 | gRPC client has no deadline/timeout on outgoing calls | File new issue | M5 |
| NEW-4 | `ZYNAX_API_KEY=""` bypasses auth without fatal at startup | File new issue | M5 |
| NEW-7 | No `read_only: true` rootfs in docker-compose | File new issue | M6 |
| H2 (promote) | OTel tracing to api-gateway + engine-adapter | Promote #491 M7→M5/M6 | M5/M6 |
| H1 (promote) | Stateless workflow-compiler | Promote #466 M6→M5 | M5 |

---

## New Issues to Create

### Group A: M5 Security Hardening (independent, can be done in any order)

**A1 — #567** (existing): Bearer constant-time compare  
Already filed. Add user story and link to #461 (M5.D parent).

**A2 — #568** (existing): ReadHeaderTimeout + MaxBytesReader  
Already filed. Add user story.

**A3 — NEW**: gRPC client outgoing call deadlines

```
Title: fix(services): add context.WithTimeout to all outgoing gRPC calls
Milestone: M5
Labels: fix, area/api-gateway, area/engine-adapter, area/task-broker

User story: As a platform operator, I want all inter-service gRPC calls to
have explicit deadlines so that slow downstream services don't cascade into
thread pool exhaustion and silent hangs.

Acceptance criteria:
- Given a gRPC call to workflow-compiler from api-gateway
  When the compiler takes >30s to respond
  Then the api-gateway returns HTTP 504 (deadline exceeded)
- Given a gRPC call to task-broker from engine-adapter
  When task-broker is unavailable
  Then the Activity returns Temporal's retriable error within 30s

Definition of Done:
- context.WithTimeout(ctx, 30s) on all outgoing gRPC client calls
  (api-gateway→compiler, api-gateway→engine-adapter, engine-adapter→task-broker,
   task-broker→agent-registry)
- Unit test for timeout path
- make lint test green

Dependencies: None
Blocks: Nothing (quality improvement)
```

**A4 — NEW**: ZYNAX_API_KEY empty behavior

```
Title: fix(api-gateway): refuse to start in production mode without ZYNAX_API_KEY
Milestone: M5
Labels: fix, area/api-gateway, security

User story: As a platform operator, I want the api-gateway to fail fast at
startup when ZYNAX_API_KEY is empty so that auth bypasses are immediately
visible rather than silent.

Acceptance criteria:
- Given ZYNAX_API_KEY="" and ZYNAX_DEV_INSECURE not set
  When api-gateway starts
  Then it exits with a clear error message within 1 second
- Given ZYNAX_DEV_INSECURE=1 and ZYNAX_API_KEY=""
  When api-gateway starts
  Then it starts with a loud WARN log about disabled auth

Definition of Done:
- os.Exit(1) on startup if key is empty and dev-insecure not set
- WARN log if dev-insecure flag is set
- Test for startup failure
- make lint test green

Dependencies: None
Blocks: Nothing
```

### Group B: M5 Engine Correctness (independent)

**B1 — #569** (existing): Temporal Activity RetryPolicy  
Already filed. Add user story: "As a platform operator, I want capability dispatch
retries to be bounded so that permanent failures (capability not found) don't retry
indefinitely consuming Temporal's thread pool."

**B2 — #570** (existing): Background-context goroutines  
Already filed.

**B3 — #571** (existing): mergePayload non-strings  
Already filed.

### Group C: M5 Truth Pass (independent)

**C1 — #572** (existing): workflow-compiler retention doc  
Already filed.

**C2 — #574** (existing): stub services / SERVICE_LIST  
Already filed.

**C3 — #575** (existing): Kagent positioning doc  
Already filed.

**C4 — #576** (existing): summarizer phantom  
Already filed.

**C5 — #577** (existing): phantom AGENT_LIST  
Already filed.

**C6 — #579** (existing): README per-service status table  
Already filed.

### Group D: M5 Compiler Promotion (promote M6→M5)

**D1 — #466** (existing, milestone change): Stateless workflow-compiler

Promote from M6 → M5. Rationale: OOM risk (R4 in review); low effort (3–5 days);
unblocks horizontal scale story before v0.4.0 ships.

Action: `gh issue edit 466 --milestone "Adapter Library (M5)"` + add M5 milestone label.

### Group E: M5.C Critical Path (ordered, each blocks the next)

These issues already exist. Ordering and dependencies are correct in M5-plan.md:

```
#526 (Trim agent-registry BDD)
  → #527 (agent-registry domain)
    → #528 (agent-registry gRPC + go.work)
      → #481 (compose wiring)
        → #400→#401→#402→#403 (git-adapter)
        → #405→#406→#407→#408 (ci-adapter)
        → #410→#411→#412→#413 (llm-adapter)
        → #415→#416→#417→#418 (langgraph-adapter)
```

Also: **#532** (task-broker handler tests) is independent and should be done in parallel
with the agent-registry chain.

### Group F: M5 CI Sprint (ordered)

Existing issues. Next action: **#554** (force-full-pipeline trigger), then #549 → #550.
#555 (DRY/KISS refactor) is P2; do last.

### Group G: Architecture Overhaul Tracking Issue

**G1 — NEW**: Track this documentation overhaul as an issue so reviewers can see
the scope and link to this branch.

```
Title: docs: architecture overhaul — reconcile M5 reality, gap analysis, best practices
Milestone: M5
Labels: docs, area/docs

Description:
Comprehensive documentation reconciliation based on the 2026-05-20 principal
architect review. Branch: docs/architecture-overhaul-m5.

Deliverables:
- ARCHITECTURE.md rewrite (was M1-era, now reflects M5 state)
- README.md, AGENTS.md, CLAUDE.md, SECURITY.md, ROADMAP.md fixes
- docs/reviews/ (00–05): inventory, decision ledger, reality vs docs,
  M5 state, gap analysis, action plan
- docs/engineering/best-practices/: go, python, dockerfiles, github-ci,
  architecture-patterns
- docs/milestones/M5-engineering-review.md

Resolves: D1–D10 from docs/reviews/02-reality-vs-docs.md
```

---

## Milestone Re-Plan

### M5 — Adapter Library (v0.4.0)

**Promote to M5 (from M6):**
- #466 — stateless workflow-compiler (OOM risk R4; 3–5 days effort)

**New M5 issues (from this review):**
- A3 — gRPC call deadlines (new issue)
- A4 — ZYNAX_API_KEY startup guard (new issue)
- G1 — arch-overhaul tracking issue (new issue)

**Existing M5 issues — no milestone change needed:**
#526, #527, #528, #481, #532, #554, #549, #550, #567, #568, #569, #570, #571, #572, #574, #575, #576, #577, #579

**Remain in M6 (per review recommendation):**
#580 (rate limit), #584 (resolveTemplate), #581 (Action.Output), #583 (hash ADR),
#582 (pkg.go.dev), #578 (ADR-021 scale), #488 (mTLS), #489 (SBOM+cosign), #465 (supply chain),
#487 (health probes → but M6.A epic exists), #466/#490 (stateless compiler via #490 code issue)

**Remain in M7:**
#491 (OTel — consider M5/M6), #492 (Watch polling), #493 (benchmarks)

### M6 — K8s Production-Ready (v0.5.0)

No changes to scope. ADR-020 (#240) + ADR-021 (#578) are the anchoring decisions.

### M7 — Full Observability (v0.6.0)

Consider promoting #491 (OTel baseline for api-gateway + engine-adapter) to M5 or M6.
The 2026-05-20 review rates observability absence as "High" gap. A workflow run is
untraceable end-to-end today. Recommendation: promote #491 to M6 at minimum.

---

## Labels to Create/Verify

Existing label categories (from `docs/labels.md`):
- `type`: feat/fix/refactor/docs/test/ci/chore
- `area`: per service/layer
- `milestone`: M5/M6/M7/M8
- `priority`: P0/P1/P2

New labels needed (if not present):
- `security` — for security-specific issues (#567, #568, A3, A4)
- `area/architecture` — for cross-cutting architecture issues

---

## Readiness Statement

After the actions above are executed, every accepted direction from the 2026-05-20
review will have:
- ✅ Epic (existing M5.C/D/E/F epics)
- ✅ Issue with user story + acceptance criteria + milestone + labels
- ✅ Canvas (for feat: issues — agent-registry #480, adapters #381–#384 all have canvases)
- ✅ ADR links (ADR-020 for security, ADR-021 for scale)
- ✅ First `.feature` named in issue (all M5.C/adapters have BDD feature files)

An implementer can pick any item from the ordered dependency chain and start coding
immediately under the new architecture/code/test approaches.
