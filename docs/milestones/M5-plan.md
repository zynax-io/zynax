<!-- SPDX-License-Identifier: Apache-2.0 -->

# M5 — Adapter Library Execution Plan

**Milestone:** Adapter Library (M5) · v0.4.0
**GitHub Milestone:** [Adapter Library (M5)](https://github.com/zynax-io/zynax/milestone/5)
**Parent epic:** [#377](https://github.com/zynax-io/zynax/issues/377)
**Status:** In Progress
**Last updated:** 2026-05-19

---

## Structure

M5 delivers five parallel tracks:

| Track | Epic | Title | Status |
|-------|------|-------|--------|
| M5.A | [#458](https://github.com/zynax-io/zynax/issues/458) | Truth Pass — documentation alignment | In Progress (2/3 done) |
| M5.B | [#459](https://github.com/zynax-io/zynax/issues/459) | Engine Correctness Hardening | In Progress (3/4 done) |
| M5.C | [#460](https://github.com/zynax-io/zynax/issues/460) | Capability Dispatch End-to-End | In Progress |
| M5.D | [#461](https://github.com/zynax-io/zynax/issues/461) | Control Plane Security Baseline | ✅ Complete |
| M5.E | [#462](https://github.com/zynax-io/zynax/issues/462) | Developer Experience Polish | ✅ Complete |
| Adapters | [#377](https://github.com/zynax-io/zynax/issues/377) | Adapter Library (http ✅ + git + ci + llm + langgraph) | In Progress (1/5 done) |
| Tooling | [#442](https://github.com/zynax-io/zynax/issues/442) | Fully Containerized Makefile Dev Workflow | ✅ Complete |

---

## M5.A — Truth Pass (#458)

**Canvas:** [docs/spdd/458-truth-pass/canvas.md](../spdd/458-truth-pass/canvas.md)

Aligns all documentation with implementation reality following the 2026-05 architectural review.

| Issue | Title | Status |
|-------|-------|--------|
| [#472](https://github.com/zynax-io/zynax/issues/472) | Remove CNCF badge + update milestone status | ✅ Done |
| [#473](https://github.com/zynax-io/zynax/issues/473) | Audit CHANGELOG for phantom entries | ✅ Done |
| [#474](https://github.com/zynax-io/zynax/issues/474) | Python SDK decision | ⬜ Open |

---

## M5.B — Engine Correctness Hardening (#459)

**Canvas:** [docs/spdd/459-engine-correctness/canvas.md](../spdd/459-engine-correctness/canvas.md)

Fixes four production-incident-grade bugs identified in the 2026-05 architectural review.

| Issue | Title | Status |
|-------|-------|--------|
| [#475](https://github.com/zynax-io/zynax/issues/475) | resolveTemplate map-iteration determinism | ✅ Done |
| [#476](https://github.com/zynax-io/zynax/issues/476) | Replace bespoke guard parser with cel-go | ⬜ Open |
| [#477](https://github.com/zynax-io/zynax/issues/477) | CompileWorkflow structured error list | ✅ Done |
| [#478](https://github.com/zynax-io/zynax/issues/478) | SSE WriteTimeout fix | ✅ Done |

---

## M5.C — Capability Dispatch End-to-End (#460)

**Canvas:** [docs/spdd/460-capability-dispatch/canvas.md](../spdd/460-capability-dispatch/canvas.md)

Delivers the two missing services that make `zynax apply → capability dispatch` work end-to-end.

### task-broker MVP (#479)

**Canvas:** [docs/spdd/479-task-broker/canvas.md](../spdd/479-task-broker/canvas.md)

| Issue | Canvas step | Title | Status |
|-------|-------------|-------|--------|
| [#530](https://github.com/zynax-io/zynax/issues/530) | O6 | Update AGENTS.md | ⬜ Open |
| [#531](https://github.com/zynax-io/zynax/issues/531) | O7 | Align service BDD + godog steps | ⬜ Open |
| [#532](https://github.com/zynax-io/zynax/issues/532) | O8 | Handler unit tests (grpcErr coverage) | ⬜ Open |

Implementation merged: PRs #520, #522, #523. Domain coverage: 92.7%.

### agent-registry MVP (#480)

**Canvas:** [docs/spdd/480-agent-registry/canvas.md](../spdd/480-agent-registry/canvas.md)

Ordered delivery — step 1 must be CI-green before step 2 begins (ADR-016).

| Issue | Canvas step | Title | Status |
|-------|-------------|-------|--------|
| [#526](https://github.com/zynax-io/zynax/issues/526) | O1 | Trim BDD to proto scope | ⬜ Open |
| [#527](https://github.com/zynax-io/zynax/issues/527) | O2 | Domain layer | ⬜ Open (blocked on #526) |
| [#528](https://github.com/zynax-io/zynax/issues/528) | O3 | gRPC wiring + cmd + go.work | ⬜ Open (blocked on #527) |

### compose wiring (#481)

| Issue | Title | Status |
|-------|-------|--------|
| [#481](https://github.com/zynax-io/zynax/issues/481) | Add task-broker + agent-registry to docker-compose | ⬜ Open (blocked on #528) |

---

## M5.D — Control Plane Security Baseline (#461) ✅ Complete

**Canvas:** [docs/spdd/461-security-baseline/canvas.md](../spdd/461-security-baseline/canvas.md)

All 5 child issues merged: #482 #483 #484 #485 #486.

---

## M5.E — Developer Experience Polish (#462) ✅ Complete

**Canvas:** [docs/spdd/462-dx-polish/canvas.md](../spdd/462-dx-polish/canvas.md)

Both child issues merged: #485 #486.

---

## Adapter Library (#377)

**Canvas:** [docs/spdd/377-adapter-library/canvas.md](../spdd/377-adapter-library/canvas.md)

| Adapter | Epic | Canvas | Step issues | Status |
|---------|------|--------|-------------|--------|
| http | [#380](https://github.com/zynax-io/zynax/issues/380) | [380 canvas](../spdd/380-http-adapter/canvas.md) | #391–#397 | ✅ Done |
| git | [#381](https://github.com/zynax-io/zynax/issues/381) | [381 canvas](../spdd/381-git-adapter/canvas.md) | #399–#403 | BDD done; impl pending |
| ci | [#382](https://github.com/zynax-io/zynax/issues/382) | [382 canvas](../spdd/382-ci-adapter/canvas.md) | #404–#408 | BDD done; impl pending |
| llm | [#383](https://github.com/zynax-io/zynax/issues/383) | [383 canvas](../spdd/383-llm-adapter/canvas.md) | #409–#413 | BDD done; impl pending |
| langgraph | [#384](https://github.com/zynax-io/zynax/issues/384) | [384 canvas](../spdd/384-langgraph-adapter/canvas.md) | #414–#418 | BDD done; impl pending |

---

## Tooling (#442) ✅ Complete

**Canvas:** [docs/spdd/442-containerized-make/canvas.md](../spdd/442-containerized-make/canvas.md)

All 4 child issues merged: #443 #444 #445 #446.

---

## Blocked / Parking

- **#474** (Python SDK decision) — deliberate decision required before any implementation
- **#476** (guard parser) — Option A (cel-go) vs Option B (rename + fail-closed) to be decided in issue
