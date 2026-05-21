<!-- SPDX-License-Identifier: Apache-2.0 -->

# M5 — Adapter Library Engineering Review

> **Status:** In Progress — review current as of 2026-05-21  
> **Version:** v0.4.0 (pending tag push)  
> **Epic:** [#377](https://github.com/zynax-io/zynax/issues/377)  
> **Plan:** [docs/milestones/M5-plan.md](M5-plan.md) (authoritative execution detail)

---

## Executive Summary

M5 is substantially progressed but **not yet complete** as of 2026-05-21. The release
pipeline is fixed, CI is faster, and four of seven M5 tracks are done. Two critical items
remain: the agent-registry service (#480) and the compose wiring (#481) that block the
first end-to-end capability dispatch and the full 5-adapter library.

The 2026-05-20 principal architect review (`docs/architecture/2026-05-20-principal-architect-review.md`)
gives M5's underlying architecture **6.5/10** overall — excellent design, partial execution,
and a delivery-vs-narrative gap that M5.A has been systematically closing.

**What changed from M4 to M5 (so far):**
- task-broker in-memory MVP with 92.7% domain coverage
- cel-go replaces bespoke CEL evaluator (fail-open bug fixed)
- Python SDK Agent base class + @capability routing
- Unified release pipeline + first GHCR public images
- CI runner container image (ci-runner)
- Bearer-token auth, X-Request-ID propagation, idempotent apply
- SECURITY.md truth pass (removed false mTLS/SBOM/cosign claims)

**What must still land to close M5:**
- agent-registry domain (#527) → gRPC wiring (#528) → compose (#481)
- 4 remaining adapters (#381–#384), each blocked on #481
- Several hardening issues: #532 #567 #568 #569 #570 #571 #572 #574 #579
- Force-full-pipeline trigger (#554)
- v0.4.0 tag push (user action)

---

## Track Completion Status

| Track | Epic | Status | Done criteria |
|---|---|---|---|
| M5.F CI Sprint | [#542](https://github.com/zynax-io/zynax/issues/542) | 🟡 BATCH 0+1 done | #554 pending |
| M5.F.R Release Pipeline | [#556](https://github.com/zynax-io/zynax/issues/556) | ✅ Complete | All issues merged |
| M5.A Truth Pass | [#458](https://github.com/zynax-io/zynax/issues/458) | 🟡 2.5/3 done | #579 README status table pending |
| M5.B Engine Correctness | [#459](https://github.com/zynax-io/zynax/issues/459) | ✅ Complete | #475 #477 #478 #538 #539 #540 done |
| M5.C Dispatch E2E | [#460](https://github.com/zynax-io/zynax/issues/460) | 🔴 Critical path | #526→#527→#528→#481 pending |
| M5.D Security Baseline | [#461](https://github.com/zynax-io/zynax/issues/461) | ✅ Complete | #482 #483 #484 #485 #486 done |
| M5.E DX Polish | [#462](https://github.com/zynax-io/zynax/issues/462) | ✅ Complete | #485 #486 done |
| Adapters (#377) | [#377](https://github.com/zynax-io/zynax/issues/377) | 🟡 1/5 done | http ✅; 4 blocked on #481 |

---

## What Was Built (M5 deliverables, merged as of 2026-05-21)

### task-broker MVP (#479)
- **PRs:** #520, #522, #523
- **Capability:** 5 RPCs (`DispatchTask`, `AcknowledgeTask`, `GetTask`, `ListTasks`, `CancelTask`)
- **Coverage:** 92.7% domain
- **Architecture:** Hexagonal (domain/api/infra), in-memory repo, round-robin agent selection
- **Gap:** Not in compose stack (#481); handler unit tests (#532) incomplete

### cel-go Guard Evaluator (#476 / #538)
- **PRs:** #538, #539, #540
- **Change:** Replaced 80-line bespoke string-matcher (fail-open) with `cel-go` (fail-closed)
- **Correctness:** Unrecognized guard expressions now return `false`, not `true`
- **Added:** Fuzz seed corpus for guard evaluation

### Python SDK Agent base class (#474 / #535–#537)
- **PRs:** #535 (implementation), #536 (tests), #537 (docs)
- **Capability:** `Agent` base class + `@capability` decorator; gRPC streaming task events
- **Coverage:** ≥ 85%
- **Architecture:** Option A (minimal, no framework lock-in) per ADR-013

### http-adapter (#380)
- **PRs:** #391–#397
- **Capability:** REST API proxy; config-only route mapping; registry client with backoff
- **First completed adapter**

### Release pipeline fixes (#556)
- **PRs:** #557 (unified release), #559–#566 (pipeline + GHCR visibility)
- **State:** All service/adapter images now public on GHCR; unified `release.yml`
- **CLI:** v0.4.0 binaries ready; tag push pending

### CI runner image (#542)
- **PRs:** #551 (Dockerfile.ci-runner), #552 (switch all jobs)
- **Impact:** No tool downloads at CI run-time; reproducible environments

### Security hardening (#461)
- **Bearer-token auth** (#482): `ZYNAX_GW_API_KEY` environment variable
- **X-Request-ID** (#484): correlation ID propagated api-gateway → compiler → engine-adapter
- **Idempotent apply** (#485): SHA-256 manifest hash → stable `run_id`

---

## Known Gaps Identified (post-review)

See `docs/reviews/04-architecture-gaps.md` for the full ranked list. Top gaps for v0.4.0:

| Priority | Gap | Issue | Effort |
|---|---|---|---|
| P0 | agent-registry chain (#526→#527→#528→#481) | #480 | M |
| P1 | Bearer constant-time compare | #567 | XS |
| P1 | ReadHeaderTimeout | #568 | XS |
| P1 | Temporal Activity RetryPolicy | #569 | S |
| P1 | Background-context goroutines | #570 | S |
| P1 | workflow-compiler unbounded IR store | #466 | S (promote M6→M5) |
| P2 | README per-service status table | #579 | XS |

---

## Architecture Assessment (from 2026-05-20 review, updated)

| Dimension | Score (2026-05-20) | Change since review |
|---|---|---|
| Architectural soundness | 7.5/10 | No change |
| Security | 4.0/10 | +0.5 (SECURITY.md truth pass, auth hardening) |
| Maintainability | 8.0/10 | No change |
| Testing | 7.5/10 | +0.5 (cel-go fuzz seed, Python SDK tests) |
| CI/CD | 6.5/10 | +1.0 (unified release, ci-runner image, public GHCR) |
| Performance | 4.0/10 | No change (unbounded IR store still open) |

---

## M5 Definition of Done — Progress

| Criterion | Status |
|---|---|
| 1. E2E `make run-local && zynax apply code-review.yaml` with ≥1 dispatch | ❌ Pending #481 |
| 2. v0.4.0 tag + downloadable artifacts | 🟡 Pipeline ready; tag push pending |
| 3. All 5 adapters merged | ❌ 1/5 (http ✅) |
| 4. Python SDK Agent base class | ✅ |
| 5. cel-go guard evaluator | ✅ |
| 6. SECURITY.md truthful | ✅ |
| 7. CI < 10 min/PR | 🟡 Improved; not yet verified |

---

## Exit Criteria Checklist (for M5 closure)

- [ ] #526 (Trim agent-registry BDD) merged
- [ ] #527 (agent-registry domain) merged
- [ ] #528 (agent-registry gRPC + go.work) merged
- [ ] #481 (compose wiring) merged
- [ ] `make run-local && zynax apply spec/workflows/examples/code-review.yaml` produces
      observable state transitions and ≥1 capability dispatch with mock agent response
- [ ] #532 (task-broker handler tests) merged
- [ ] #567 (constant-time bearer) merged
- [ ] #569 (Temporal RetryPolicy) merged
- [ ] #466 (stateless workflow-compiler) merged (promoted from M6)
- [ ] At least one more adapter merged (git-adapter preferred: #400→#403)
- [ ] v0.4.0 tag pushed → GitHub Release created → download URLs live
- [ ] CI consistently runs < 10 minutes per PR

---

## What Should NOT Change

These architectural elements are confirmed as "crown jewels" by the 2026-05-20 review
and must not be modified without an ADR:

- Three-layer separation (Intent / Communication / Execution)
- Hexagonal `internal/{api,domain,infrastructure}` per service
- Proto-first + BDD-first discipline (ADR-016)
- Apache-2.0 + DCO
- `WorkflowEngine` 6-method interface (ADR-015)
- Per-service AGENTS.md pattern
- ADR culture (ADR-001–ADR-019)

---

*This review will be updated when M5 closes. See `docs/milestones/M5-plan.md` for live execution detail.*
