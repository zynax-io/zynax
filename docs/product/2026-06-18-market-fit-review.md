<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Market-Fit Review (2026-06-18)

**Document type:** Product Strategy · Market Assessment · Longitudinal Positioning  
**Date:** 2026-06-18 · **Baseline:** [docs/architecture/2026-05-28-competitive-positioning.md](../architecture/2026-05-28-competitive-positioning.md) and [docs/product/strategy.md](strategy.md) — first dated market-fit review  
**Related:** [2026-05-20 Principal Architect Review](../architecture/2026-05-20-principal-architect-review.md) · [ROADMAP.md](../../ROADMAP.md) · [state/current-milestone.md](../../state/current-milestone.md)

---

## Executive Summary — Product-Market Fit Verdict

**Status:** Shipped/MVP · **Verdict:** Promising category, credible architecture, community adoption remains the blocking gate.

Zynax v0.5.0 (M6, released 2026-06-12) ships production-grade infrastructure — mTLS, SBOM/cosign supply chain, Postgres-backed horizontal scale, Helm charts, and NATS event bus — closing May's critical security/scalability gaps. The **end-to-end capability dispatch path is now wired** (task-broker + agent-registry MVP in compose; all 5 adapters shipped). A developer can run `make run-local && zynax apply spec/workflows/examples/e2e-demo.yaml` and see a multi-step workflow execute end-to-end with traces in Uptrace. **This is a system, not a collection of stubs.**

The **market position is defensible:** engine-agnostic IR + adapter-first + GitOps-native beats Kagent (K8s-locked Pod-per-agent) for orgs wanting engine portability and co-existence story. Temporal/Argo are shipping in the e2e CI matrix. One beachhead use case (agentic code-review / feature-automation workflows) has runnable examples + dogfooding via DevAuto.

**The single blocking gate is not technical — it is adoption.** Live signals: 1 star, 0 forks, 0 external users, 1 maintainer. CNCF Sandbox prerequisites (≥2 maintainers from ≥2 orgs, ≥1 external adopter, public community cadence) remain unmet. M7 (active) is reframed as first-run UX closeout (#1370); M-UX (forward user experience) opens after. **Monetization posture should follow traction, not precede it.**

---

## 1. Category Definition

**The control plane for AI agent workflows** — sitting at the intersection of three maturing markets:

1. **AI agent orchestration** (LangGraph, CrewAI, AutoGen, Kagent) — fast, noisy, mostly Python libraries, mostly LLM-centric.
2. **Durable workflow engines** (Temporal, Restate, Argo Workflows, Dapr) — mature, infrastructure-shaped, not AI-specific.
3. **AI/ML platforms** (Kubeflow, Flyte, SageMaker, Vertex) — heavyweight, model-lifecycle-centric, adjacent not direct.

**Zynax's category (shipped, proven):** Write a workflow once in YAML; run it on whatever engine your org operates (Temporal, Argo); invoke any agent or capability (no SDK required); govern it with policies; ship it as GitOps manifest. The category is **real** — it solves a genuine adoption problem for enterprises running multiple execution engines. It is **contested** — Kagent entered CNCF Sandbox (2026) pitching almost identical positioning. It is **maturing** — three adjacent markets are all growing.

**Market signal:** The category is crowded but the wedge (engine-agnostic + adapter-first) is genuinely rare. Kagent's K8s-native Pod-per-agent model and Zynax's declarative IR model serve different buyer segments. **This is complementary positioning, not head-to-head conflict** — a Kagent-managed agent can register as a Zynax capability via the gRPC `AgentService` contract. Co-existence story is real and differentiated.

---

## 2. Competitive Landscape (Lead: Zynax vs Kagent)

### 2.1 Direct Comparison

Both pitch "control plane for AI agents." The split is sharp (from [docs/architecture/2026-05-28-competitive-positioning.md](../architecture/2026-05-28-competitive-positioning.md), updated with M6 deltas):

| Dimension | **Zynax (v0.5.0)** | **Kagent (CNCF Sandbox 2026)** |
|---|---|---|
| **Core abstraction** | Engine-agnostic Workflow IR (protobuf) | Kubernetes CRDs (Pod-per-agent model) |
| **Engine portability** | ✅ Temporal + Argo (LangGraph planned) — shipped in CI matrix | ❌ Kubernetes-only; ADK lock-in |
| **Deployment** | Docker Compose (M5) + K8s Helm (M6) | K8s-only (Kind required) |
| **Agent integration model** | Adapter-first, no SDK required (any gRPC `AgentService` is a capability) | Kubernetes agents + MCP tools |
| **Workflow authoring** | Declarative YAML, compile-time IR validation | Kubernetes CRDs, kubectl-native |
| **GitOps native** | ✅ `zynax apply` is idempotent via canonical hash | ❌ kubectl-imperative |
| **Web UI** | ❌ (M8 roadmap) | ✅ included |
| **Multi-LLM support** | Via `llm-adapter` (Go; OpenAI/Bedrock/Ollama) | ✅ built-in ModelConfig |
| **Multi-language agents** | ✅ Go + Python adapters, same control plane | ✅ Any container |
| **Observability** | ✅ OpenTelemetry + Uptrace (M7 core merged; dashboard/UI landing) | ✅ K8s-native Prometheus + Grafana |
| **CNCF status** | Not yet (M8 target) | ✅ Sandbox 2026 |
| **Production-proven** | 🟡 v0.5.0 (shipped mTLS/SBOM/scale, not yet deployed at scale) | Partial |

**Messaging gap:** Zynax currently conflates engine-agnosticism with Temporal-specific features. Fix: lead **every** pitch with "write once, run on Temporal or Argo without rewriting the workflow" — this is the differentiator Kagent cannot match.

### 2.2 Full Competitive Matrix

| Project | Role | Why it matters | CNCF | Risk level |
|---|---|---|---|---|
| **Kagent** | Direct competitor | Same positioning, CNCF-backed, K8s-native alternative | Sandbox | **Critical — strategic positioning** |
| **Temporal** | Wrapped runtime | Largest technical overlap; Zynax *runs on* Temporal. | No | Medium |
| **Restate** | Wrappable runtime | Durable execution alternative to Temporal; a future engine adapter. | No | Low |
| **Dapr Workflows** | Adjacent | General-purpose K8s orchestration; not AI-specific. | Incubating | Low |
| **Argo Workflows** | Engine (shipped) | DAG-based; Zynax ships ArgoEngine as an adapter ([#766](https://github.com/zynax-io/zynax/issues/766)). | Graduated | Low |
| **LangGraph / CrewAI / AutoGen** | Wrapped, not rivals | Frameworks Zynax invokes as capabilities via adapters (ADR-011). | No | Low |
| **Flyte / Kubeflow / SageMaker** | Adjacent MLOps | Model lifecycle, not agentic workflows. | No / Incubating / Proprietary | Low |
| **Backstage / Crossplane** | Vision analogy | Inspirations for "platform control plane" framing; Zynax is narrower. | CNCF | Low |

**Key insight:** Zynax has *one* direct competitor (Kagent) and *one* large technical overlap (Temporal) where Zynax's value prop is portability, not replacement.

> **Note on competitor facts:** Kagent's CNCF Sandbox status, web-UI, and feature claims are external and not verifiable from this repository; they are carried forward from the May competitive-positioning doc and should be re-validated against Kagent's public sources before external use.

---

## 3. TAM Framing

**Total addressable markets (three adjacent slices):**

1. **Agent Orchestration Market** (noisy growth) — market-size figures are external/unverified; Zynax does not own this slice — it competes with LangGraph/Kagent/CrewAI.
2. **Workflow Engine Market** (mature) — Temporal, Dapr (Incubating), Argo (graduated). Zynax is a *consumer* of this market, not a seller into it.
3. **Platform Engineering / Governance Layer** (emerging) — GitOps + policy enforcement + multi-team engine abstraction. **Zynax's wedge here is strongest** — platform teams wanting engine portability without SDK lock-in.

**Zynax's wedge position (where it can win decisively):**
- **Beachhead:** Agentic software-engineering automation (code-review, PR, feature-impl workflows).
- **Expand to:** Platform teams wanting a single control plane across multiple agents/engines/teams.
- **Defend against:** Kagent (K8s-native alternative) and Temporal (buying direct adoption instead of via a layer).

**Market dynamics:**
- **Shipped:** Zynax has the only engine-agnostic control plane with multi-engine proof in CI (Temporal + Argo). Kagent has CNCF backing but is K8s-only.
- **Aspirational:** LangGraph adds agents/teams layer later; Temporal considers a control plane; Kagent adds non-K8s support.
- **Risk:** Category becomes commoditized if "control plane" becomes table stakes. **Mitigation: ruthless messaging on the portability wedge.**

---

## 4. Personas & Beachhead Validation

From [docs/product/strategy.md](strategy.md) §6, with M6/M7 evidence:

| Persona | Zynax fit today | Beachhead evidence | Horizon |
|---|---|---|---|
| **AI-forward dev team** (hero persona) | **Strong** — workflows in YAML they can PR-review | ✅ Three runnable examples: [code-review.yaml](../../spec/workflows/examples/code-review.yaml), [feature-implementation.yaml](../../spec/workflows/examples/feature-implementation.yaml), [ci-pipeline.yaml](../../spec/workflows/examples/ci-pipeline.yaml) · DevAuto wave 4 dogfooding (M6 delivered to boundary) | Now |
| **Platform engineer** | Strong architecture, weak proof (no production adopter) | Helm charts shipped (M6); multi-namespace routing shipped (#799); Postgres-backed scale shipped | M7→M8 |
| **AI-infra team** | Good (adapter-first, multi-language) | 5 adapters shipped (http, git, ci, llm, langgraph); Python SDK on PyPI (M6 #805) | M7→M8 |
| **Enterprise governance** | Partial (policy + rate-limit shipped; RBAC/SSO not) | Policy routing + rate-limits shipped (M6 #802/#803/#804); audit trail via Postgres + CloudEvents | Post-M8 |

**Beachhead validation (agentic software-engineering automation):**
- ✅ **Shipped:** Three runnable workflows (code-review, feature-impl, ci-pipeline) with state machines, loops, human-in-the-loop.
- ✅ **Dogfooding:** DevAuto wave 4 (M6 #881) deployed 9 expert AgentDefs, orchestrator workflow, issue→plan→route→deliver→verify pipeline, learning-synthesizer. Deployed to boundary; O8 (#1103) deferred to M7 (platform gap: compiler `output:` binding).
- ✅ **Observable:** End-to-end trace in Uptrace (M7 EPIC O core merged; dashboard UI landing).
- **Gap:** No external customer reference. **Blocking gate for CNCF + enterprise sales.**

---

## 5. Adoption-Funnel Reality (Live Metrics vs Targets)

### 5.1 Baseline (May 2026 Architect Review)

| Metric | Baseline | Source |
|---|---|---|
| GitHub stars | 0 | Architect review §1.3 |
| GitHub forks | 0 | Architect review §1.3 |
| External adopters | 0 | Architect review §1.3 |
| Maintainers | 1 | Architect review §1.3 |
| Watchers | 0 | — |

### 5.2 Current State (2026-06-18)

| Metric | Current | Change | Note |
|---|---|---|---|
| GitHub stars | 1 | +1 (0→1) | Early signal, not traction yet. |
| GitHub forks | 0 | → | No external contributors yet. |
| Watchers | 0 | → | No engagement signal. |
| Contributors (unique) | 3 | +3 | Single maintainer + 2 cloud/test agents (not independent contributors). |
| External users | 0 | → | No named adopter. |
| Maintainers (confirmed) | 1 | → | **Critical blocker for CNCF.** |

### 5.3 Adoption Targets vs Reality (from [docs/product/strategy.md](strategy.md) §7.2)

| Target | Why | Reality | Gap |
|---|---|---|---|
| **Time-to-first-working-workflow** | Conversion gate | 15 min (quickstart) · M7 #1370 improving | 🟡 Partial — depends on Day-0 engine (#1359) |
| **GitHub stars / forks** | Awareness signal | 1 star; 0 forks | 🔴 Stalled — need high-quality demo + messaging |
| **External adopters (named)** | CNCF prerequisite | 0 | 🔴 Blocking — need hero use case proof + outreach |
| **Ecosystem adapters (community-built)** | Extensibility proof | 0 | 📅 Aspirational — Restate adapter planned (#1102) |
| **Contributors / second maintainer** | Bus-factor + CNCF | 1 maintainer, 0 external | 🔴 Blocking — long lead time (social process) |

**Honest assessment:** Adoption is stalled at awareness. The architecture is proven; the system works end-to-end; the code is clean. **The next unit of leverage is distribution, not engineering.** M7's UX-closeout (#1370 + stories #1371–#1388) directly targets conversion friction.

---

## 6. Shipped vs Partial vs Aspirational (as of 2026-06-18)

From [docs/product/strategy.md](strategy.md) §2, updated post-M6:

| Capability | Status | Evidence | Timeline |
|---|---|---|---|
| **Declarative YAML → IR → execution** | ✅ Shipped | M2–M6; `zynax apply` runs end-to-end | Stable |
| **Engine-agnostic dispatch (Temporal + Argo)** | ✅ Shipped | Both legs in e2e-smoke CI engine matrix (#771, #1071) | Stable |
| **Adapter-first, no-SDK capabilities** | ✅ Shipped | 5 adapters: http, git, ci, llm (Go, M7 #1278), langgraph (Python, M6) | Stable |
| **mTLS inter-service** | ✅ Shipped | M6 #464; env-var cert paths + gRPC credential wiring | Stable |
| **SBOM/cosign signing / SLSA L2** | ✅ Shipped | M6 #465, #489; multi-arch release images | Stable |
| **Postgres-backed horizontal scale** | ✅ Shipped | M6 EPIC #626; task-broker + agent-registry on pgx/v5 | Stable |
| **Event bus (NATS JetStream)** | ✅ Shipped | M6 EPIC #772; Publish/Subscribe/Unsubscribe + DLQ (ADR-022) | Stable |
| **Helm charts (all 7 services + subcharts)** | ✅ Shipped | M6 EPIC #765; 14 O-steps merged | Stable |
| **gRPC Health Checking Protocol** | ✅ Shipped | M6 #74/#656; grpc.health.v1 in all services | Stable |
| **Prometheus `/metrics` per-request** | ✅ Shipped | M6 #491; per-service instrumentation | Stable |
| **OpenTelemetry + Uptrace** | 🟡 Partial | M7 EPIC O core merged; dashboard login UI + helm landing | M7 active |
| **Workflow data-flow (output→input)** | 🟡 Partial | M7 EPIC W keystone (#1178, #1167); compiler still gates `output:` | M7 active |
| **LangGraph as a full engine** | 📅 Aspirational | langgraph-adapter exists as a *capability provider*, not a `WorkflowEngine` | M7+ |
| **Web UI** | 📅 Aspirational | M8 roadmap; none in M6 | M8 |
| **Python SDK on PyPI** | ✅ Shipped | M6 #805, #769; trusted publisher + TestPyPI dry-run | Stable |
| **Memory service (Redis KV + pgvector)** | ✅ Shipped | M6 EPIC #773; all 10 RPCs, namespace TTL | Stable |

**Trend:** M6 shifted from "aspirational infrastructure" to "shipped production-grade." M7 (active) is completing observability and workflow usability. **The system is a system.**

---

## 7. Product-Market Fit Verdict & Score

**Verdict:** **Shipped/MVP · Promising category, credible architecture, community adoption is the blocker.**

### Grounding

1. **Architecture (from 2026-05-20 Principal Architect Review):** May 6.5/10 → June ~7.5/10 (integration gap closed; security/scalability closed; observability partial).
2. **Product-market fit (from 2026-05-20 review §1.4):** May "Promising category, premature claim" (7.0/10). June: end-to-end demo exists and runnable; verdict holds at 7.0/10 because adoption is the remaining blocker, not technology.
3. **Adoption signals (live 2026-06-18):** 1 star, 0 forks, 0 external users, 1 maintainer; CNCF prerequisites unmet. **Technology problem solved + adoption problem unsolved.**

### Final Score

| Dimension | Score | Rationale |
|---|---|---|
| **Architecture quality** | 7.5/10 | Three-layer separation, hexagonal services, engine abstraction proven in CI. Observability core merged. |
| **Engineering culture** | 8.0/10 | ADRs, BDD-first, 2:1 test:code ratio, ≥90% domain coverage, Truth-Pass culture. |
| **Category defensibility** | 7.0/10 | Engine-agnostic + adapter-first is unique; Kagent is K8s-only alternative (complementary). |
| **Beachhead proof** | 7.0/10 | Three runnable workflows + DevAuto Wave 4 dogfooding. No external customer yet. |
| **Product-market fit** | 7.0/10 | Shipped, end-to-end, observable. Blocked on adoption. Technology no longer the gate. |
| **CNCF readiness** | 5.0/10 | Code quality + governance ready; community prerequisites unmet. |

**Combined PMF score: 7.0 / 10 · Status: Shipped/MVP.**

---

## 8. Recommendations (Prioritized by User Type + Adoption Lever)

### Awareness → Evaluation (Current gate: Day-0 friction)

1. **Ship zero-Temporal evaluation mode (#1359)** [**audience: evaluators** | **lever: friction reduction**] — in-process lightweight engine for laptop evaluation. Effort: M. Status: #1359 merged; engine in active development.
2. **Publish flagship demo as asciinema cast in README** [**audience: evaluators** | **lever: time-to-insight**] — real 15-min screencast (clone → run-local → apply → traces). Effort: S. Status: in-progress (M7 UX closeout).

### Evaluation → Adoption (Current gate: Community + adopter proof)

3. **Recruit a second maintainer from a second org** [**audience: contributors** | **lever: credibility + bus-factor**] — CNCF prerequisite. Effort: L (2–3 months). Status: not started; blocking M8.
4. **Establish public cadence + GitHub Discussions** [**audience: community** | **lever: visibility + async participation**] — weekly office hours + "what shipped" digest. Effort: S. Status: Discussions enabled; cadence not established.
5. **Define and recruit the first named external adopter** [**audience: power users** | **lever: social proof**] — onboard an org running agentic workflows; document case study. Effort: M. Status: not started.

### Adoption → Expansion (M8+)

6. **Grow example/template library around the software-engineering beachhead** [**audience: dev teams** | **lever: discoverability**] — 3→10 runnable workflows; Diátaxis docs. Effort: M. Status: three shipped; M7 extending.
7. **Make contribution fast-lane separate from full SPDD/AGENTS model** [**audience: contributors** | **lever: participation**] — 5-line first-PR path. Effort: S. Status: M-dx backlog (#1391).
8. **Decide monetization posture only after traction signals** [**audience: ecosystem** | **lever: sustainability**] — keep core Apache-2.0; don't feature-gate before adoption. Effort: S (decision). Status: both scenarios articulated in strategy §9; defer until M8.

---

## 9. Longitudinal Delta vs May Review

### Technology / Engineering Deltas

| Finding | May | June | Delta |
|---|---|---|---|
| **End-to-end dispatch wired** | ❌ Critical gap | ✅ e2e-demo succeeds (WORKFLOW_STATUS_COMPLETED) | **CLOSED** |
| **Security posture** | ❌ Insecure gRPC, no SBOM/cosign/mTLS | ✅ mTLS, SBOM/cosign/SLSA L2, constant-time bearer | **CLOSED** |
| **Scalability / state management** | ⚠ Unbounded in-memory IR store | ✅ Made stateless (#466/#490) | **CLOSED** |
| **Postgres horizontal scale** | ❌ In-memory | ✅ Both on pgx/v5 (EPIC #626) | **CLOSED** |
| **Event bus** | ❌ Stub | ✅ NATS JetStream + engine-adapter wiring (EPIC #772) | **CLOSED** |
| **Memory service** | ❌ Stub | ✅ Redis KV + pgvector (EPIC #773) | **CLOSED** |
| **Observability core** | ❌ Zero OpenTelemetry | 🟡 OTel + Uptrace core merged; dashboard landing | **Partial → complete in M7** |
| **Performance** | 0/10 — zero load tests | 🟡 Benchmarks + benchstat gate (#493) | **Partial** |
| **Workflow data-flow** | ❌ Not architected | 🟡 Keystone landed (#1178); compiler still gates | **Partial** |

**Summary:** All May critical + high-priority recommendations are closed or on track. **The technology is no longer the problem.**

### Market / Adoption Deltas

| Signal | May | June | Delta |
|---|---|---|---|
| **GitHub stars** | 0 | 1 | +1 (minimal) |
| **GitHub forks** | 0 | 0 | → |
| **External users** | 0 | 0 | → (CNCF blocker) |
| **Maintainers** | 1 | 1 | → (critical gap) |
| **Category crowding** | "Now crowded" (Kagent entering) | "Kagent + Zynax complementary positioning" | Refined |
| **Beachhead proof** | 3 example workflows | 3 examples + DevAuto Wave 4 to boundary | ✅ Stronger |
| **Product-market fit verdict** | 7.0/10: "premature claim" | 7.0/10: "Shipped/MVP, adoption blocked" | **Stable; gating clarified** |

**Summary:** Technology moved 6.5→7.5/10; adoption remains stuck at 0 signals. **The blocking gate shifted from "technical integration" to "community credibility."**

---

## 10. Recommendations by Adoption Lever (Classified for `/plan` Handoff)

| # | Recommendation | Type | Audience | Adoption Lever | Priority | Milestone | Size |
|---|---|---|---|---|---|---|---|
| 1 | Ship zero-Temporal evaluation mode | feat | Evaluators | Friction → 5-min demo | P0 | M7 | M |
| 2 | Publish flagship asciinema demo in README | docs | Evaluators | Awareness / time-to-insight | P0 | M7 | S |
| 3 | Recruit 2nd maintainer from 2nd org | chore | Governance | Community credibility | P0 | M7→M8 | L |
| 4 | Establish public cadence + Discussions | chore | Community | Visibility + participation | P1 | M7 | S |
| 5 | Define + recruit 1st named adopter | product | Power users | Social proof → CNCF | P0 | M7→M8 | M |
| 6 | Grow example/template library (3→10) | docs | Dev teams | Discoverability | P1 | M7 | M |
| 7 | Fast-lane contribution path | docs | Contributors | First-contributor onboarding | P1 | M-dx | S |
| 8 | Decide monetization only after traction | chore | Strategy | Neutrality + optionality | P1 | M8 | S |

**For `/plan` handoff:** Issues #1359 (Day-0 engine) and #1360 (make demo) are already in M7 active; #1370 is UX closeout epic. New issues for maintainer + adopter recruitment are governance-track, not feature-track.

---

## 11. Risk Longitudinal Track

From [docs/architecture/2026-05-20-principal-architect-review.md](../architecture/2026-05-20-principal-architect-review.md), updated:

| Risk | May Status | June Status | Mitigation |
|---|---|---|---|
| **R1: End-to-end dispatch not wired** | 🔴 Critical | ✅ **CLOSED** | — |
| **R2: Single maintainer / bus-factor** | 🔴 Open | 🔴 **OPEN (unchanged)** | Recruit 2nd maintainer — start now |
| **R3: Insecure gRPC by default** | 🔴 Critical | ✅ **CLOSED** (mTLS, M6) | — |
| **R4: Unbounded in-memory state / OOM** | 🔴 High | ✅ **CLOSED** | — |
| **R6: Kagent absorbs the category** | 🟡 Open | 🟡 **OPEN (refined)** — complementary, not conflicting | Ruthless messaging on engine-agnostic wedge |
| **R8: Delivery-vs-narrative gap** | 🟡 Mitigated | 🟡 **MITIGATED (maintained)** | Keep culture alive |
| **R9: Release pipeline / install URLs** | 🔴 High | ✅ **CLOSED** | — |
| **New R10: Adoption stalled at awareness** | N/A | 🔴 **OPEN (new finding)** | M7 UX closeout + maintainer + adopter recruitment |

**Net:** 4 critical risks closed by M6 engineering. 2 new strategic risks open: bus-factor + adoption awareness. Both are non-technical; both require long lead time.

---

## 12. Market-Fit Verdict Summary (One-Liner)

**Zynax v0.5.0 is a shipped, architecturally sound control plane for AI workflows with credible engine-agnostic IR + multi-engine proof; the blocking gate is adoption (community credibility, named adopter, second maintainer), not technology.**

---

## Appendix — Sources & Methodology

### A. Grounding sources (cited throughout)

- **Competitive positioning:** [docs/architecture/2026-05-28-competitive-positioning.md](../architecture/2026-05-28-competitive-positioning.md)
- **Strategic analysis:** [docs/product/strategy.md](strategy.md)
- **Architecture review:** [docs/architecture/2026-05-20-principal-architect-review.md](../architecture/2026-05-20-principal-architect-review.md)
- **Live milestone status:** [state/current-milestone.md](../../state/current-milestone.md)
- **Roadmap:** [ROADMAP.md](../../ROADMAP.md)
- **Shipped evidence:** [README.md](../../README.md) + github.com/zynax-io/zynax (live releases, CI matrix)

### B. Live signals (pulled 2026-06-18)

| Signal | Method | Result |
|---|---|---|
| GitHub stars/forks/watchers | `gh api repos/zynax-io/zynax` | 1 star, 0 forks, 0 watchers |
| Contributor count | `gh api repos/zynax-io/zynax/contributors` | 3 (maintainer + agents) |
| Latest release | github.com/zynax-io/zynax/releases | v0.5.0 (2026-06-12) |
| CI matrix | `.github/workflows/e2e-smoke.yml` | Temporal + Argo both green |

### C. Mark-up conventions

- ✅ **Shipped** — merged to main, in a release, or demonstrated end-to-end.
- 🟡 **Partial** — core landed, polish pending; roadmap-committed.
- 📅 **Aspirational** — designed, not yet implemented; M7+ roadmap.
- 🔴 **Blocker** — unmet gate for CNCF / adoption / enterprise sale.
- 🟢 **Mitigated** — was a risk, now managed.

> Competitor figures and market-size claims are external/unverified and should be re-confirmed against primary sources before external publication.
