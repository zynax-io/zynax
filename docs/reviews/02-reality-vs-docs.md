<!-- SPDX-License-Identifier: Apache-2.0 -->

# 02 — Reality vs Documentation

**Date:** 2026-05-21  
**Branch:** docs/architecture-overhaul-m5  
**Purpose:** Phase 1 artifact — every place where documentation disagrees with
code, merged PRs, open issues, or the 2026-05-20 principal architect review.

Format: **Doc claims X → Code/evidence shows Y → Fix required.**

---

## Summary

| Severity | Count | Description |
|---|---|---|
| **Critical** | 4 | Doc asserts system works end-to-end / milestone is complete when it is not |
| **High** | 8 | Doc asserts a security or capability property that doesn't exist |
| **Medium** | 6 | Stale milestone status, broken links, missing docs |
| **Low** | 5 | Minor inaccuracies, naming drift |

---

## Critical Discrepancies

### C1 — ARCHITECTURE.md milestone table is stale by 4 milestones
**Location:** `ARCHITECTURE.md` lines 9–19 (§Milestone Status at top of file)  
**Doc claims:** M1 Complete; M2 "Next"; M3 "Planned"; M4 "Planned"  
**Reality:** M2 ✅ Complete; M3 ⚠ Partial; M4 ⚠ Partial; M5 In Progress  
**Evidence:** `state/current-milestone.md`, `CLAUDE.md`, `README.md` milestone table  
**Fix:** Rewrite entire `ARCHITECTURE.md` — see Phase 3.

### C2 — ARCHITECTURE.md §13 milestones table completely wrong
**Location:** `ARCHITECTURE.md` lines 498–510  
**Doc claims:** M2 "Next", M5–M7 "Planned", wrong architecture component descriptions  
**Reality:** M2–M4 delivered; M5 in progress; component descriptions outdated  
**Fix:** Part of full ARCHITECTURE.md rewrite.

### C3 — README M4 description claims agent-registry routing exists
**Location:** `README.md` lines 377–379  
**Doc claims:** "M4 delivered ... `kind: AgentDef` routing via `AgentRegistryService`"  
**Reality:** `services/agent-registry/` has 0 LoC, no go.mod — does not exist. AgentDef routing is not functional.  
**Evidence:** `ls services/agent-registry/` shows only AGENTS.md + BDD feature files  
**Fix:** Correct README M4 description; add caveat.

### C4 — Quickstart does not warn that first capability dispatch will fail
**Location:** `README.md` lines 280–300 (Quickstart section)  
**Doc claims:** `zynax apply spec/workflows/examples/code-review.yaml` works  
**Reality:** code-review.yaml dispatches capabilities; task-broker is not in the compose stack (#481 open); agent-registry does not exist. Every workflow with actions will fail at first dispatch.  
**Evidence:** `infra/docker-compose/docker-compose.yml` — task-broker service missing; review §4.1  
**Fix:** Add warning to quickstart: "Note: workflows with capability actions require task-broker and agent-registry, which are pending M5.C (#460). State transitions will log without capability dispatch until #481 lands."

---

## High Severity Discrepancies

### H1 — AGENTS.md has 3 broken file links
**Location:** `AGENTS.md` lines 205–212 (Knowledge Base Index)  
**Doc claims:**  
- `docs/architecture/execution-architecture.md`  
- `docs/architecture/competitive-analysis-2026.md`  
- `docs/architecture/2026-05-external-architectural-review.md`  
**Reality:** Actual files are:  
- `docs/architecture/2026-04-30-execution-architecture.md`  
- `docs/architecture/2026-04-30-competitive-analysis.md`  
- `docs/architecture/2026-05-20-principal-architect-review.md`  
**Fix:** Update 3 links in AGENTS.md Knowledge Base Index.

### H2 — AGENTS.md references non-existent §7
**Location:** `SECURITY.md` line: "See `AGENTS.md §7` and `docs/adr/`"  
**Reality:** AGENTS.md has no §7 (no section numbering at all in AGENTS.md)  
**Fix:** Remove or replace with specific link.

### H3 — Bearer-token compare is not constant-time (open security gap)
**Location:** `services/api-gateway/internal/api/auth.go:15`  
**Doc claims:** SECURITY.md lists "Constant-time bearer comparison" as "In Progress" tracked by #567  
**Reality:** `r.Header.Get("Authorization") != want` — string comparison is not constant-time; timing oracle exists  
**Evidence:** Confirmed by reading `auth.go` 2026-05-21; issue #567 filed but not yet addressed  
**Fix:** `crypto/subtle.ConstantTimeCompare` — see issue #567.

### H4 — All inter-service gRPC uses insecure credentials
**Location:** `services/*/internal/infrastructure/clients.go`  
**Doc claims:** SECURITY.md §Planned says "mTLS between all platform services"  
**Reality:** `insecure.NewCredentials()` everywhere; no TLS at all  
**Evidence:** Review §7 table row 1; review score 4.0/10 for security  
**Fix:** TLS-first dial helper (ADR-020 + #488); insecure gated behind `ZYNAX_DEV_INSECURE=1`.

### H5 — CloudEvents publishing is a log stub
**Location:** `services/engine-adapter/internal/infrastructure/activities.go`  
**Doc claims:** README M3 description says "CloudEvents lifecycle publishing (`zynax.workflow.state.entered/exited/completed/failed`)"  
**Reality:** `PublishLifecycleEventActivity` emits a WARN-level log entry and returns nil. No event is actually published to NATS or any bus.  
**Evidence:** Review §4.6; confirmed by description "only logs as warn in M5.B (#483)"  
**Note:** M5.B #483 fixed swallowed errors → logs as WARN now, but publishing is still a stub.  
**Fix:** Label M3 CloudEvents as "log-stub, not published" in docs; full implementation tracked in event-bus stub.

### H6 — workflow-compiler retains IRs unboundedly
**Location:** `services/workflow-compiler/internal/api/server.go:31`  
**Doc claims:** Proto comments describe 30-day retention  
**Reality:** `map[string]*zynaxv1.WorkflowIR` with no eviction, no TTL, no LRU bound, no persistence  
**Evidence:** Review §4.3; file line 31 confirmed  
**Fix:** #466 (promote to M5, inject Store port); correct proto contract doc.

### H7 — ARCHITECTURE.md describes NATS event bus as wired and working
**Location:** `ARCHITECTURE.md` §8, §Data Flows step 8, step 9  
**Reality:** `services/event-bus/` has 0 LoC; NATS is in the compose stack but nothing publishes to it; Memory Service is 0 LoC  
**Fix:** ARCHITECTURE.md must clearly label event-bus and memory-service as "stub — not yet implemented" with issue links.

### H8 — No HEALTHCHECK in service Dockerfiles
**Location:** `infra/docker/`, service `Dockerfile` files  
**Doc claims:** Definition of Done in AGENTS.md: "Health probes implemented"  
**Reality:** No `HEALTHCHECK` directive in service Dockerfiles (review §7.2)  
**Evidence:** Review §7.2 ❌ "No HEALTHCHECK directive in Dockerfiles"  
**Fix:** Add HEALTHCHECK to all service Dockerfiles; canvas #463 (health-probes) exists.

---

## Medium Severity Discrepancies

### M1 — ROADMAP.md M5 section is out of date
**Location:** `ROADMAP.md` lines ~129–163  
**Doc claims:** M5.A/B/C/D/E checkbox list shows most items as open  
**Reality:** M5.D ✅ complete, M5.E ✅ complete, M5.B cel-go ✅, Python SDK ✅, BATCH 0+1 all done  
**Fix:** Update ROADMAP.md M5 section with current completion state.

### M2 — ROADMAP.md M3/M4 marked incorrectly
**Location:** `ROADMAP.md`  
**Doc claims:** M3/M4 have full ✅ checkboxes  
**Reality:** M3/M4 are ⚠ Partial — task-broker and agent-registry not delivered  
**Fix:** Add "⚠ Partial" notation and link to M5.C blocker.

### M3 — M2, M3, M4 milestone reviews are missing
**Location:** `docs/milestones/`  
**Doc claims:** CLAUDE.md references `M1-engineering-review.md` and `M1-release-notes.md`  
**Reality:** Only M1 has review + release notes; M2/M3/M4 have none  
**Fix:** Create M2/M3/M4 reviews in Phase 6 (or log as deferred with DECISIONS-NEEDED).

### M4 — README Docker Images table incomplete
**Location:** `README.md` Docker Images section  
**Doc claims:** Lists api-gateway, workflow-compiler, engine-adapter, task-broker, http-adapter, tools  
**Reality:** Correct (agent-registry has no image, properly omitted). However: the table doesn't explain which images go into `make run-local` and which don't. Task-broker is listed but not in compose yet.  
**Fix:** Add footnote on task-broker: "published on GHCR; not yet in make run-local (M5.C #481)".

### M5 — agents/sdk version claims vs reality
**Location:** Historical — `agents/sdk/` previously had version placeholder "v0.1.0" for a 3-line stub  
**Reality:** Python SDK Agent base class is now implemented (#535 ✅)  
**Status:** Fixed in M5.A BATCH 3 — no longer an issue as of 2026-05-21.

### M6 — cmd/zynax is not in go.work
**Location:** README.md line 98: "From source (requires Go 1.25+): `cd cmd/zynax && GOWORK=off go build -o ~/bin/zynax .`"  
**Reality:** This is correct (`cmd/zynax` is a standalone module; `GOWORK=off` is required)  
**Status:** Not a discrepancy — the README is correct. Note it here for completeness.

---

## Low Severity Discrepancies

### L1 — ARCHITECTURE.md §WorkflowEngine interface shows wrong signature
**Location:** `ARCHITECTURE.md` lines 232–250  
**Doc claims:** Interface has `Query(ctx, id) (*ExecutionState, error)` method  
**Reality:** Actual method is `GetWorkflowStatus` per `engine_adapter.proto`; `Query` is an internal name  
**Fix:** Sync ARCHITECTURE.md interface snippet to actual code.

### L2 — ARCHITECTURE.md §Interoperability lists TypeScript client
**Location:** `ARCHITECTURE.md` lines 442–460  
**Doc claims:** "A TypeScript CI dashboard submits a workflow..."  
**Reality:** No TypeScript stubs exist; this is aspirational/illustrative text  
**Fix:** Label as "illustrative example of multi-language interoperability" or simplify to Go examples.

### L3 — AGENTS.md references `services/AGENTS.md` in knowledge base index
**Location:** `AGENTS.md` line 213: "Per-layer rules: `services/AGENTS.md`..."  
**Reality:** `services/AGENTS.md` exists but is a thin redirect to per-service AGENTS.md files; correct path is `services/*/AGENTS.md`  
**Fix:** Update the knowledge base index link.

### L4 — Phantom agents in AGENT_LIST
**Location:** `state/` or similar tracking files  
**Doc claims:** (per review G23) Phantom researcher/calculator agents in AGENT_LIST  
**Status:** Tracked by #577  
**Fix:** Remove from lists when #577 is actioned.

### L5 — Summarizer example has only a feature file
**Location:** `agents/examples/`  
**Doc claims:** (per review G22) Implies working summarizer agent  
**Reality:** Feature file only; no implementation  
**Status:** Tracked by #576  
**Fix:** Either implement or remove.

---

## Previously Fixed Discrepancies (2026-05-20 Truth Pass)

These were flagged in the 2026-05-20 review and have since been fixed:

| Item | Fixed by | PR/Issue |
|---|---|---|
| SECURITY.md claimed mTLS, SBOM, cosign | Rewritten to match reality | Truth-pass 2026-05-20 |
| CNCF Sandbox Candidate badge | Removed | #472 |
| Phantom CHANGELOG entries (Helm charts, Argo engine) | Removed | #473 |
| Bespoke CEL evaluator fail-open (evalGuard) | Replaced with cel-go | #538 |
| Python SDK was 3-line placeholder | Agent base class implemented | #535 |
| `resolveTemplate` map non-determinism | Fixed with sorted keys | #475 |
| Event publish errors silently swallowed | Now logged as WARN | #483 |
| `CompileWorkflow` returned only first error | Returns full error list | #477 |
| GOWORK=off documentation | Added to AGENTS.md, CLAUDE.md | ADR-017 |
| Non-constant-time bearer compare | Filed as #567 | Open |
| AGENTS.md footer "CNCF Sandbox Candidate" | Removed (badge dropped in #472, footer not updated) | docs/compress-ai-context |
| CLAUDE.md Go-1.22-specific anti-patterns | Removed (toolchain is Go 1.26.3) | docs/compress-ai-context |
| `docs/ai-assistant-setup.md` "AGENTS.md §12" | Fixed to "AGENTS.md §AI Anti-patterns" (§12 never existed) | docs/compress-ai-context |

---

*This file is a point-in-time snapshot (2026-05-21). Each item must be re-verified against
HEAD before it is actioned. "Fixed" items should be validated from the codebase, not
taken on faith from this document.*
