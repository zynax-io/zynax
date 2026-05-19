<!-- SPDX-License-Identifier: Apache-2.0 -->

# M5 — Adapter Library Execution Plan

**Milestone:** Adapter Library (M5) · v0.4.0
**GitHub Milestone:** [Adapter Library (M5)](https://github.com/zynax-io/zynax/milestone/5)
**Parent epic:** [#377](https://github.com/zynax-io/zynax/issues/377)
**Status:** In Progress
**Last updated:** 2026-05-19 (rev 2)

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
| [#474](https://github.com/zynax-io/zynax/issues/474) | **Python SDK** — Agent base class epic | ⬜ Epic (see below) |

### Python SDK epic (#474) — promoted

**Canvas:** [docs/spdd/474-python-sdk/canvas.md](../spdd/474-python-sdk/canvas.md)
**Decision:** Option A chosen — implement minimal `Agent` base class.

| Issue | Step | Title | Status |
|-------|------|-------|--------|
| [#535](https://github.com/zynax-io/zynax/issues/535) | O1 | Implement Agent base class | ⬜ Open |
| [#536](https://github.com/zynax-io/zynax/issues/536) | O2 | Unit tests (≥ 85% coverage) | ⬜ Open (blocked on #535) |
| [#537](https://github.com/zynax-io/zynax/issues/537) | O3 | Docs update — README, ARCHITECTURE.md, AGENTS.md | ⬜ Open (blocked on #535+#536) |

---

## M5.B — Engine Correctness Hardening (#459)

**Canvas:** [docs/spdd/459-engine-correctness/canvas.md](../spdd/459-engine-correctness/canvas.md)

Fixes four production-incident-grade bugs identified in the 2026-05 architectural review.

| Issue | Title | Status |
|-------|-------|--------|
| [#475](https://github.com/zynax-io/zynax/issues/475) | resolveTemplate map-iteration determinism | ✅ Done |
| [#476](https://github.com/zynax-io/zynax/issues/476) | **Guard evaluator** — cel-go epic | ⬜ Epic (see below) |
| [#477](https://github.com/zynax-io/zynax/issues/477) | CompileWorkflow structured error list | ✅ Done |
| [#478](https://github.com/zynax-io/zynax/issues/478) | SSE WriteTimeout fix | ✅ Done |

### Guard evaluator epic (#476) — promoted

**Canvas:** [docs/spdd/476-guard-parser/canvas.md](../spdd/476-guard-parser/canvas.md)
**Decision:** Option A — integrate `github.com/google/cel-go` (fail-closed on eval error).

| Issue | Step | Title | Status |
|-------|------|-------|--------|
| [#538](https://github.com/zynax-io/zynax/issues/538) | O1 | Integrate cel-go into evalGuard | ⬜ Open |
| [#539](https://github.com/zynax-io/zynax/issues/539) | O2 | Test suite + fuzz seed | ⬜ Open (blocked on #538) |
| [#540](https://github.com/zynax-io/zynax/issues/540) | O3 | Remove CEL misrepresentation from docs | ⬜ Open (blocked on #538+#539) |

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

### http-adapter (#380) ✅ Complete

All step issues merged: #391 #392 #393 #394 #395 #396 #397.

### git-adapter (#381)

**Canvas:** [docs/spdd/381-git-adapter/canvas.md](../spdd/381-git-adapter/canvas.md) · Capabilities: `open_pr`, `request_review`, `get_diff`

| Issue | Step | Title | Status |
|-------|------|-------|--------|
| [#399](https://github.com/zynax-io/zynax/issues/399) | O1 | BDD contract feature file | ✅ Done |
| [#400](https://github.com/zynax-io/zynax/issues/400) | O2 | Go module scaffold + config layer | ⬜ Open |
| [#401](https://github.com/zynax-io/zynax/issues/401) | O3 | Capability handler (open_pr, request_review, get_diff) | ⬜ Open (blocked on #400) |
| [#402](https://github.com/zynax-io/zynax/issues/402) | O4 | Registry client + bootstrap | ⬜ Open (blocked on #401) |
| [#403](https://github.com/zynax-io/zynax/issues/403) | O5 | Dockerfile, docker-compose, AGENTS.md | ⬜ Open (blocked on #402) |

### ci-adapter (#382)

**Canvas:** [docs/spdd/382-ci-adapter/canvas.md](../spdd/382-ci-adapter/canvas.md) · Capabilities: `trigger_workflow`, `get_run_status`

| Issue | Step | Title | Status |
|-------|------|-------|--------|
| [#404](https://github.com/zynax-io/zynax/issues/404) | O1 | BDD contract feature file | ✅ Done |
| [#405](https://github.com/zynax-io/zynax/issues/405) | O2 | Go module scaffold + config layer | ⬜ Open |
| [#406](https://github.com/zynax-io/zynax/issues/406) | O3 | CIHandler + PollLoop | ⬜ Open (blocked on #405) |
| [#407](https://github.com/zynax-io/zynax/issues/407) | O4 | Registry client + bootstrap | ⬜ Open (blocked on #406) |
| [#408](https://github.com/zynax-io/zynax/issues/408) | O5 | Dockerfile, docker-compose, AGENTS.md | ⬜ Open (blocked on #407) |

### llm-adapter (#383)

**Canvas:** [docs/spdd/383-llm-adapter/canvas.md](../spdd/383-llm-adapter/canvas.md) · Capability: `chat_completion` · Python

| Issue | Step | Title | Status |
|-------|------|-------|--------|
| [#409](https://github.com/zynax-io/zynax/issues/409) | O1 | BDD contract feature file | ✅ Done |
| [#410](https://github.com/zynax-io/zynax/issues/410) | O2 | Module scaffold + ProviderConfig | ⬜ Open |
| [#411](https://github.com/zynax-io/zynax/issues/411) | O3 | Provider handlers (OpenAI, Bedrock, Ollama) | ⬜ Open (blocked on #410) |
| [#412](https://github.com/zynax-io/zynax/issues/412) | O4 | Registry client + bootstrap | ⬜ Open (blocked on #411) |
| [#413](https://github.com/zynax-io/zynax/issues/413) | O5 | Dockerfile, docker-compose, AGENTS.md | ⬜ Open (blocked on #412) |

### langgraph-adapter (#384)

**Canvas:** [docs/spdd/384-langgraph-adapter/canvas.md](../spdd/384-langgraph-adapter/canvas.md) · Maps LangGraph StateGraph to capabilities · Python

| Issue | Step | Title | Status |
|-------|------|-------|--------|
| [#414](https://github.com/zynax-io/zynax/issues/414) | O1 | BDD contract feature file | ✅ Done |
| [#415](https://github.com/zynax-io/zynax/issues/415) | O2 | Module scaffold + GraphMount config | ⬜ Open |
| [#416](https://github.com/zynax-io/zynax/issues/416) | O3 | GraphLoader + LangGraphHandler | ⬜ Open (blocked on #415) |
| [#417](https://github.com/zynax-io/zynax/issues/417) | O4 | Registry client + bootstrap | ⬜ Open (blocked on #416) |
| [#418](https://github.com/zynax-io/zynax/issues/418) | O5 | Dockerfile, docker-compose, AGENTS.md | ⬜ Open (blocked on #417) |

---

## Tooling (#442) ✅ Complete

**Canvas:** [docs/spdd/442-containerized-make/canvas.md](../spdd/442-containerized-make/canvas.md)

All 4 child issues merged: #443 #444 #445 #446.

---

## Blocked / Parking

- **#474** (Python SDK decision) — deliberate decision required before any implementation
- **#476** (guard parser) — Option A (cel-go) vs Option B (rename + fail-closed) to be decided in issue
