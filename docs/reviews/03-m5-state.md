<!-- SPDX-License-Identifier: Apache-2.0 -->

# 03 — M5 Current State

**Date:** 2026-05-21  
**Branch:** docs/architecture-overhaul-m5  
**Source of truth:** `docs/milestones/M5-plan.md` (rev 30) + `state/current-milestone.md`  
**Purpose:** Phase 1 artifact — verified M5 scope, what's done, what's left, blockers.

---

## M5 Definition of Done (7 criteria)

| # | Criterion | Status |
|---|---|---|
| 1 | `make run-local && zynax apply spec/workflows/examples/code-review.yaml` → real state transitions + ≥1 capability dispatch | ❌ **Not done** — task-broker not in compose; agent-registry not implemented |
| 2 | v0.4.0 tag on GitHub with downloadable CLI binaries and GHCR service images | 🟡 **Partial** — CHANGELOG promoted; `git push origin v0.4.0` pending (user action required) |
| 3 | All 5 adapters (http ✅ + git + ci + llm + langgraph) merged | ❌ 1/5 — git/ci/llm/langgraph BDD done, implementations pending (#400–#418) |
| 4 | Python SDK `Agent` base class implemented (#474) | ✅ **Done** — #535 #536 #537 all merged |
| 5 | cel-go replaces bespoke guard evaluator (#476/#538) | ✅ **Done** — #538 #539 #540 all merged |
| 6 | SECURITY.md matches shipped reality | ✅ **Done** — truth pass completed 2026-05-20 |
| 7 | CI pipeline < 10 minutes per PR | 🟡 **Partial** — BATCH 0+1 done; #554 force-full-pipeline pending; target not yet verified |

**Overall:** 2.5/7 criteria met. **M5 is not yet complete.**

---

## Track Status

### M5.F — CI/CD Performance Sprint (#542) 🟡 In Progress

| Sub-track | Status |
|---|---|
| **BATCH 0** (concurrency, branch protection, release race) | ✅ All done: #547 #544 #548 #545 #589 #546 #557 #558 |
| **BATCH 1** (release pipeline, GHCR) | ✅ All done: #559–#566, #601, tools public |
| **Group A** (#554 force-full-pipeline) | ⬜ Open — next up |
| **Group B** (per-service change detection #549, #550, #220) | ⬜ Open |
| **Group C** (ci-runner: #551 ✅, #552 ✅; tools secure #358) | 🟡 Mostly done |
| **Group E** (DRY/KISS refactor #555) | ⬜ Open (P2) |

**M5.F.R** (Release Pipeline #556): ✅ Complete — all child issues done.

### M5.A — Truth Pass (#458) 🟡 In Progress

| Issue | Status |
|---|---|
| #472 Remove CNCF badge | ✅ Done |
| #473 Audit CHANGELOG phantom entries | ✅ Done |
| #474 Python SDK Agent base class | ✅ Done (BATCH 3 complete) |
| Fix SECURITY.md | ✅ Done (2026-05-20) |
| Add per-service status table to README (#579) | ⬜ Open |

### M5.B — Engine Correctness (#459) ✅ 3/4 Done

| Issue | Status |
|---|---|
| #475 resolveTemplate determinism | ✅ Done |
| #476 Guard evaluator cel-go epic | ✅ Done (#538 #539 #540) |
| #477 CompileWorkflow structured error list | ✅ Done |
| #478 SSE WriteTimeout fix | ✅ Done |

**M5.B is effectively complete.** Epic #476 parent may still show open (parent of closed children).

### M5.C — Capability Dispatch E2E (#460) 🔴 Critical Path — Blocked

This is the hardest dependency chain and the primary blocker for M5's E2E criterion.

#### task-broker (#479) — code complete, quality in progress

| Issue | Step | Status |
|---|---|---|
| PRs #520 #522 #523 | Core implementation | ✅ Merged (domain coverage 92.7%) |
| #530 | Update AGENTS.md | ✅ Done |
| #531 | BDD + godog step alignment | ✅ Done |
| #532 | Handler unit tests (gRPC error paths) | ⬜ **Open — do first** |

**Key gap:** task-broker is **not in the docker-compose stack**. Even with code merged, `make run-local` does not start it. Tracked by #481.

#### agent-registry (#480) — 0 LoC, pending

| Issue | Step | Status |
|---|---|---|
| #526 | Trim BDD to proto scope | ⬜ **Open — do first** |
| #527 | Domain layer | ⬜ Blocked on #526 |
| #528 | gRPC wiring + cmd + go.work | ⬜ Blocked on #527 |
| #481 | Compose wiring (task-broker + agent-registry) | ⬜ Blocked on #528 |

**Required for M5.C:** `agent-registry` must implement:
- `AgentRepository` port (in-memory backing for M5; Postgres in M6)
- `AgentRegistryService` application service (round-robin health tracking)
- Heartbeat timeout logic (mark unhealthy after 2 min without ping)

#### E2E exit criterion
`make run-local && zynax apply spec/workflows/examples/code-review.yaml` must produce observable state transitions AND ≥1 capability dispatch with a real (mock-data) agent response.

### M5.D — Security Baseline (#461) ✅ Complete
All 5 child issues merged: #482 (bearer auth) #483 (event publish warn) #484 (X-Request-ID) #485 (idempotent apply) #486 (compose consolidation).

### M5.E — Developer Experience Polish (#462) ✅ Complete
Child issues merged: #485 #486.

### Adapter Library (#377) 🟡 1/5 Done

| Adapter | Status |
|---|---|
| http-adapter (#380) | ✅ Complete — all step issues merged (#391–#397) |
| git-adapter (#381) | BDD done (#399); impl pending #400→#401→#402→#403 (blocked on #481) |
| ci-adapter (#382) | BDD done (#404); impl pending #405→#406→#407→#408 (blocked on #481) |
| llm-adapter (#383) | BDD done (#409); impl pending #410→#411→#412→#413 (blocked on #481) |
| langgraph-adapter (#384) | BDD done (#414); impl pending #415→#416→#417→#418 (blocked on #481) |

All 4 pending adapters are **blocked on #481 (compose wiring)**. They need a live agent-registry to register against.

---

## Architecture Gap Issues (filed post-review, status as of 2026-05-21)

| Gap | Issue | Status | Milestone |
|---|---|---|---|
| G1: Bearer non-constant-time | #567 | ⬜ Open | M5 |
| G2: ReadHeaderTimeout + MaxBytesReader | #568 | ⬜ Open | M5 |
| G3: Rate limiting POST /apply | #580 | ⬜ Open | M6 |
| G4: No RetryPolicy on Temporal Activities | #569 | ⬜ Open | M5 |
| G5: Watch polling load | #492 | ⬜ Open | M7 |
| G6: resolveTemplate bespoke | #584 | ⬜ Open | M6 |
| G7: mergePayload drops non-strings | #571 | ⬜ Open | M5 |
| G8: Action.Output parsed never consumed | #581 | ⬜ Open | M6 |
| G9: No CODEOWNERS | ~~#573~~ | Closed — file exists | — |
| G10: workflow-compiler retention contract violated | #572 | ⬜ Open | M5 |
| G11: No benchmarks | #493 | ⬜ Open | M7 |
| G12: No fuzz tests | #539 (partial) | 🟡 Partial (fuzz seed for CEL) | M5/M7 |
| G16: Background context goroutines | #570 | ⬜ Open | M5 |
| G17/H9: Stub services in SERVICE_LIST | #574 | ⬜ Open | M5 |
| G19: Kagent/Dapr positioning | #575 | ⬜ Open | M5 |
| G22: Summarizer phantom | #576 | ⬜ Open | M5 |
| G23: Phantom AGENT_LIST entries | #577 | ⬜ Open | M5 |
| G24: Compose missing task-broker/agent-registry | #481 | ⬜ Open (blocked on #528) | M5 |
| H8: ADR-021 scale plan | #578 | ⬜ Open | M6 |
| README status table | #579 | ⬜ Open | M5 |

### Promoted Gap Issues (M6 → M5 proposed by review)

| Gap | Issue | Reason to promote |
|---|---|---|
| G10/H1: workflow-compiler stateless (#466) | #466 | OOM risk in production (review R4); low effort (3–5 days) |

---

## Blockers Summary

| Blocker | Unblocks |
|---|---|
| #526 (Trim agent-registry BDD) | #527 → #528 → #481 |
| #527 (agent-registry domain) | #528 → #481 |
| #528 (agent-registry gRPC + go.work) | #481 |
| #481 (compose wiring) | All 4 adapters (#400–#418) + E2E criterion |
| v0.4.0 tag push (user action) | All install URLs; release assets |
| #532 (task-broker handler tests) | task-broker quality closure |
| #554 (force-full-pipeline trigger) | CI DX completion |

---

## Remaining Work to Close M5 (ordered by dependency)

1. **#532** — task-broker handler unit tests (XS, independent)
2. **#567** — bearer constant-time compare (XS, independent)
3. **#568** — ReadHeaderTimeout + MaxBytesReader (XS, independent)
4. **#569** — Temporal Activity RetryPolicy (S, independent)
5. **#570** — background-context goroutines in task-broker (S, independent)
6. **#571** — mergePayload non-string handling (S, independent)
7. **#572** — workflow-compiler retention doc fix (XS, independent)
8. **#574** — stub services AGENTS.md / SERVICE_LIST (XS, independent)
9. **#575** — Kagent positioning doc (S, independent)
10. **#576** — remove summarizer phantom (XS, independent)
11. **#577** — remove phantom AGENT_LIST entries (XS, independent)
12. **#579** — README per-service status table (XS, independent)
13. **#466** — stateless workflow-compiler / IR store TTL (S, promoted from M6)
14. **#526** → **#527** → **#528** → **#481** — agent-registry chain (M, critical path)
15. **Adapters** (#400–#418) — after #481
16. **#554** — force-full-pipeline trigger (S, after #552)
17. **v0.4.0 tag push** — user action; CHANGELOG already promoted

---

## What v0.4.0 Will Contain (when shipped)

Based on merged code as of 2026-05-21:

**Already shipped (waiting for tag):**
- task-broker in-memory MVP with 92.7% domain coverage
- Bearer-token auth + X-Request-ID propagation
- Idempotent apply with manifest hash
- http-adapter (full implementation)
- cel-go guard evaluator (replaces fail-open bespoke CEL)
- Python SDK Agent base class + @capability routing
- Unified release workflow
- CI runner container mode (ci-runner image)
- GHCR public images for all services
- SECURITY.md truth pass

**Not yet in v0.4.0 (pending M5.C):**
- E2E capability dispatch
- agent-registry service
- task-broker in compose stack
- git/ci/llm/langgraph adapters

---

*Re-verify against live GitHub issue state with `gh issue list --state open --milestone "Adapter Library (M5)"` before acting on this document.*
