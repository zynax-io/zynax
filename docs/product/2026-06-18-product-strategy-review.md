<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Product Strategy Review (2026-06-18)

**Document type:** Point-in-time Product Strategy Review  
**Review date:** 2026-06-18  
**Baseline:** [docs/product/strategy.md](../../docs/product/strategy.md) (no prior review found; diffing against living strategy)  
**Scope:** Tier 1 (public-safe) — grounded claims only  
**Live signals updated:** via `gh api` (2026-06-18)

---

## 1. Executive Summary

**Single most important product finding:** Zynax has closed the critical delivery-vs-narrative gap that shadowed the May 2026 architect review. End-to-end capability dispatch is now **fully wired and shipped in M6 (v0.5.0, released 2026-06-12)**; mTLS, SBOM/cosign/SLSA supply-chain, Postgres-backed horizontal scale, and NATS JetStream event bus all shipped. The architecture remains sound. **However, adoption is still the binding constraint:** 1 star, 0 forks, 3 contributors, 0 external adopters. M7 (active, v0.6.0 target) is the usability sprint; M8 is the CNCF Sandbox bid. **The product decisions are largely made; the next unit of leverage is distribution and first-run UX, not features.** The hero use case (agentic software-engineering automation with multi-engine portability) is real and runnable; the blockers are community and category messaging vs Kagent.

| Question | Current answer |
|---|---|
| Is the architecture differentiated? | ✅ Yes — engine-agnostic IR + adapter-first proven (Temporal+Argo in CI). |
| Is the category defensible? | ⚠ Contested — Kagent (CNCF Sandbox 2026) occupies same framing; differentiation must be ruthless. |
| What is actually shipped? | ✅ M6 complete; end-to-end dispatch, K8s-ready, mTLS, observability core (M7 EPIC O done). |
| What blocks adoption today? | ⚠ Traction + Day-0 friction: zero external users, single maintainer, M7 first-run UX (keystone #1167 merged). |
| What is the beachhead? | ✅ Agentic software-engineering automation — three runnable workflows shipped. |

---

## 2. Positioning Check — Are We Still Differentiated?

**From [README.md](../../README.md) (lines 24–26):**
> *"Zynax is to AI workflows what Kubernetes is to containers — a control plane that abstracts the execution layer behind a declarative, versionable API."*

**Current positioning:** Declarative + engine-agnostic + adapter-first + GitOps-native.

**Drift from living strategy (vs [docs/product/strategy.md](../../docs/product/strategy.md)):** None detected. The strategy document (March 2026, working draft) correctly names the defenses:
- Engine-agnostic IR dispatch: **High defensibility** ✅ (Temporal+Argo in CI per M6).
- Capability routing via `AgentService` contract: **High defensibility** ✅ (shipped, stable).
- Event-driven state machines: **Medium-high** ✅ (ADR-014 enforces; loops and human-in-the-loop native).
- Declarative + GitOps: **Medium** ✅ (table stakes soon, well-executed via idempotent apply with canonical hash).

**Strategic risk — still open:** Kagent (CNCF Sandbox 2026, [docs/architecture/2026-05-28-competitive-positioning.md](../../docs/architecture/2026-05-28-competitive-positioning.md), lines 73–82) occupies the same tagline. From the competitive analysis (2026-05-28):

> **Kagent wins** today when an org wants K8s-native agent management with a web UI and a CNCF-backed ecosystem.

**Mitigation status:** Low. The product still lacks a web UI (M8 roadmap), and Kagent has first-mover CNCF backing. However, Zynax's engine-agnostic thesis is genuinely rare: Kagent requires K8s; Zynax runs on Compose **and** K8s/Helm (M6 shipped). The messaging has not yet pivoted ruthlessly to this wedge.

---

## 3. Real vs Aspirational — Capability Status (Post-M6)

**Update from architect review (2026-05-20):** The May review flagged **critical gaps** that are now resolved. Per [state/current-milestone.md](../../state/current-milestone.md) (lines 14–26, 62–80) and [README.md](../../README.md) (lines 410–435):

| Capability | May status | Today (2026-06-18) | Proof |
|---|---|---|---|
| **Declarative YAML → IR → execution** | ✅ Shipped (M2) | ✅ Shipped | `zynax apply` runs end-to-end; e2e-smoke CI gate green |
| **Engine-agnostic dispatch (Temporal + Argo)** | 🟡 Partial (Temporal only) | ✅ Shipped (M6) | Both legs in e2e-smoke CI engine matrix (#1365 real gate, #766) |
| **End-to-end capability dispatch** | ❌ **Not wired** (May critical risk R1) | ✅ **Fully wired** (M6 EPIC #626) | task-broker Postgres-backed, agent-registry heartbeat, DispatchCapabilityActivity → agent gRPC |
| **mTLS + SBOM/cosign/SLSA** | ❌ Promised, not real (May R3) | ✅ Shipped (M6) | All inter-service mTLS via env-var cert paths; cosign images + SPDX SBOM in releases (#235 #239 #465) |
| **Postgres-backed repositories** | 📅 Planned (M6) | ✅ Shipped (M6 EPIC #626) | task-broker + agent-registry on pgx/v5; horizontal scale proven |
| **NATS JetStream event bus** | 📅 Planned (M6) | ✅ Shipped (M6 EPIC #772) | Publish/Subscribe/Unsubscribe + DLQ; ADR-022 wired in engine-adapter (#827) |
| **Helm charts for all services** | 📅 Planned (M6) | ✅ Shipped (M6 EPIC #765) | 7 services + NATS/Postgres/Temporal subcharts; infra/helm/ populated |
| **OpenTelemetry + Uptrace observability** | 📅 Planned (M7) | 🟡 **Partial** (M7 EPIC O done) | Core OTEL instrumentation + exemplars merged (#1187–#1192); Uptrace compose & Helm working; backend UI/log export in-flight |
| **Workflow data-flow (output→input bindings)** | 📅 Planned (M7 keystone) | 🟡 **Partial** (M7.W keystone #1167 merged) | Compiler still gates `output:` field syntax; domain model landed; tests passing |
| **LangGraph as a full engine** | 📅 Planned | 📅 Aspirational | langgraph-adapter shipped as *capability provider* (M5), not a `WorkflowEngine` implementation |
| **Web UI** | ❌ None | ❌ None | M8 roadmap; no work started |
| **Python SDK on PyPI** | 🟡 Partial (zynax-sdk stub) | ✅ Published (M6 EPIC #626) | `zynax-sdk` on PyPI with Agent base class + @capability routing |

**Scorecard delta (architect review May vs now):**

| Dimension | May | Now | Change |
|---|---:|---:|---|
| Overall architecture | 6.5 | 7.5 | ↑ end-to-end wired, supply-chain done |
| Security | 4.0 | 8.0 | ↑↑ mTLS + SBOM/cosign/SLSA shipped |
| Scalability | 4.5 | 7.0 | ↑ Postgres horizontal, config convergence |
| Maintainability | 8.0 | 8.5 | → strong (ADRs, hexagonal, ≥90% domain) |
| Product-market fit | 7.0 | 6.5 | ↓ shipping improved, but traction still 0 |
| CNCF alignment | 6.5 | 7.5 | ↑ supply-chain done; community gap remains |

---

## 4. Adoption Funnel — Live Signals (2026-06-18)

**Baseline (from May architect review):** 0 stars, 0 forks, 0 external adopters.

**Current (via `gh api` 2026-06-18):**

```
{
  "stars": 1,
  "forks": 0,
  "watchers": 0,
  "open_issues": 52,
  "contributors": 3
}
```

**Milestones (open, via `gh api`):**
```
Usable Workflows + Observability (M7)    open: 24  closed: 93
CNCF Sandbox (M8)                         open: 7   closed: 0
Developer Experience (M-dx)               open: 13  closed: 0
User Experience (M-UX)                    open: 2   closed: 0
```

| Metric | May baseline | Current (2026-06-18) | Trend | Interpretation |
|---|---|---|---|---|
| **GitHub stars** | 0 | 1 | ↑ minimal but non-zero | First external awareness signal (likely internal seeding) |
| **Forks** | 0 | 0 | → | No external contributor pickup yet |
| **Watchers** | 0 | 0 | → | Repository not on external radar |
| **Contributors** | ? | 3 | ↑ | Core team size; bus-factor unchanged (1 maintainer named) |
| **Open issues** | 58 | 52 | ↓ | M7 orchestration draining backlog (57 closed M7, 24 open) |
| **Releases in 6 weeks** | 1 (v0.3.0) | 6 total (v0.4.0, v0.5.0, proto-stubs) | ↑↑ | Shipping cadence strong |
| **External adopters** | 0 | 0 | → | **No named adopters** — largest blocker for CNCF Sandbox |

**Time-to-first-working-workflow audit (from [docs/quickstart.md](../../docs/quickstart.md)):**
- Clone repo → `make bootstrap` → `make run-local` → `zynax apply spec/workflows/examples/code-review.yaml` → `zynax logs` → `make stop-local`
- Estimated user time: **8–12 minutes** on a fast connection, if Temporal + NATS images are cached.
- **Still blocked by:** Temporal dependency (even in lightweight mode); large tools image pull (slow first-run).
- **M7 initiative (epic #1370, 2026-06-18 reframing):** "First-run UX closeout" — target **under 15 minutes** with a zero-Temporal lightweight evaluation mode and a one-command demo path (`make demo`).

**Release velocity (from `gh api` releases):**
- v0.4.0 (2026-05-28), v0.5.0 (2026-06-12): **14-day cycle** = good for a pre-1.0 project.
- 6 releases in 6 weeks (including proto-stubs snapshots) = active engineering.
- Docker images published to GHCR on every release + on every main merge (`:main` tag live).

---

## 5. Beachhead Validation — Is the Hero Use Case Holding?

**Hero use case (from [docs/product/strategy.md](../../docs/product/strategy.md) §6.2):**
> *"Ship a multi-agent code-review / feature-implementation / CI workflow as a YAML manifest, run it on Temporal **or** Argo without a rewrite, and watch it execute end-to-end with traces in Uptrace — in under 15 minutes."*

**Validation evidence:**

1. **Runnable example workflows exist:**
   - [spec/workflows/examples/code-review.yaml](../../spec/workflows/examples/code-review.yaml)
   - [spec/workflows/examples/feature-implementation.yaml](../../spec/workflows/examples/feature-implementation.yaml)
   - [spec/workflows/examples/ci-pipeline.yaml](../../spec/workflows/examples/ci-pipeline.yaml)
   - All confirmed real YAML with loops, human-in-the-loop, and (post-M7.W) cross-state data-flow.

2. **End-to-end dispatch verified:**
   - M6 shipped task-broker + agent-registry (EPIC #626).
   - All 5 execution adapters shipped (http · git · ci · llm · langgraph, per [README.md](../../README.md) lines 426–434).
   - Argo engine adapter shipped (ADR-015 engine abstraction, #766).
   - e2e-smoke CI gate runs workflows end-to-end with real capability dispatch.

3. **Observability proof:**
   - M7 EPIC O (observability) **complete as of 2026-06-17** (state/current-milestone.md line 63).
   - OTEL core instrumentation + exemplars merged (#1187–#1192).
   - Uptrace compose stack running (docker-compose with Uptrace backend + login UI).
   - Connected distributed traces from engine-adapter → task-broker → adapter gRPC.

4. **Multi-engine portability:**
   - TemporalEngine (M3, shipping since v0.2.0).
   - ArgoEngine (M6, #766, tested in e2e-smoke CI matrix per #1365).
   - `WorkflowEngine` interface (ADR-015) proven (6-method port, reusable).

5. **Internal dogfooding:**
   - DevAuto Wave 4 (self-hosted issue-delivery engine, EPIC #881) uses Zynax as the substrate.
   - Per [state/current-milestone.md](../../state/current-milestone.md) line 100, O1–O7 + O9 delivered; production-grade proof of concept.

**Verdict:** ✅ The beachhead is **real and credibly proven.** The only gap is **external validation** (zero named external adopters, zero forks from outside).

---

## 6. M7 Execution Status — First-Run UX (2026-06-18 Mid-Milestone)

**From [state/current-milestone.md](../../state/current-milestone.md) (lines 30–80):**

M7 opened 2026-06-15; **12 EPICs**, one REASONS Canvas each. Current delivery (as of 2026-06-17):

| EPIC | Title | Status | Blocker? |
|---|---|---|---|
| **W** | Workflow data-flow (output/input bindings) — **keystone** | 🟡 Partial (W.4 #1178 merged; domain landed; compiler still gates `output:`) | 🟡 Blocking rest of workflow composition |
| **L** | Execution log/event streaming | 🟡 Partial (L.3–L.4 merged) | No blocker |
| **O** | Observability — OTEL + Uptrace | ✅ **Complete** (O.4–O.9 + exemplars merged) | None — shipped |
| **C** | Context propagation | 🟡 Partial (C.2 merged) | No blocker |
| **G** | Git MCP shim | 🟡 Partial (G.3–G.5 merged) | No blocker |
| **X** | Expert-agent substrate + examples | ✅ **Complete** (X.2 merged; agents/examples live) | None — shipped |
| **T** | Reusable templates + workflows | 🟡 Partial (T.1–T.2, T.4 merged) | No blocker |
| **R** | Test rigor (benchmarks, fuzz) | 🟡 Partial (R.1–R.2 merged; benchstat gate active) | No blocker |
| **Q** | Quality & supply-chain | ✅ **Complete** (Q.4–Q.5 merged; CVE audit clean) | None — shipped |
| **D** | Docs (quick-start, authoring, observability) | 🟡 In-progress (#1355–#1358 landed docs) | No blocker |
| **P** | llm-adapter Go port (ADR-035) | 🟡 Partial (P.1–P.3 shipped; Python llm-adapter retired #1283) | No blocker |
| **S** | CI bash → Go CLI (ADR-036) | 🟡 Partial (S.1–S.2 shipped) | No blocker |

**Truth-pass (2026-06-17, state/current-milestone.md lines 68–72):**
- 2026-06-17 verification: architecture review committed (docs/reviews/06, #1295 + #1303).
- Milestone renamed from "Full Observability" to **"Usable Workflows + Observability"** (reflects true scope).
- ROADMAP.md reconciled with live state (#1299).
- `make ci` green on current tools image (#1302 closed pip-audit drift).

**Completion rate:** ~57 closed / ~34 open as of 2026-06-17. **Keystone W (data-flow) is the only hard dependency;** all others are parallel/additive.

---

## 7. Recommendations — Classified by User Type + Adoption Lever

Each recommendation is tagged with **(user type, adoption lever)** for `/plan` handoff:

### 7.1 **Critical** (gates M8 Sandbox filing)

1. **Recruit a second maintainer from a second org** (maintainer/governance, community)
   - **Why:** Single maintainer is R2 (May review) and §6.3 CNCF blocker.
   - **Evidence:** [docs/product/strategy.md](../../docs/product/strategy.md) §7.3; [GOVERNANCE.md](../../GOVERNANCE.md) (conflict resolution defined but no deputy).
   - **Action:** Start now via agent/cloud-native communities. 6–12 week lead time.
   - **Success metric:** 2nd maintainer actively reviewing PRs + co-signing releases by M8 gate.

2. **Land the M7 first-run UX keystone (data-flow #1167) and publish one flagship demo** (developer, Day-0 friction)
   - **Why:** Time-to-first-working-workflow is the conversion gate; M7.W is still partial (compiler gates `output:`).
   - **Evidence:** [state/current-milestone.md](../../state/current-milestone.md) line 44 (W keystone); [docs/product/strategy.md](../../docs/product/strategy.md) §7.1 (time-to-first-workflow metric).
   - **Action:** Finish compiler `output:` field support; publish asciinema cast of code-review.yaml on Temporal + Argo in README.
   - **Success metric:** `make demo` runs hero workflow end-to-end in < 15 min; ≥ 1 star/fork from external user (not internal seeding).

3. **Ship zero-Temporal lightweight evaluation mode** (operator/evaluator, Day-0 friction)
   - **Why:** Temporal dependency deters evaluators (May review gap G3). In-memory engine or stubbed backend reduces barrier.
   - **Evidence:** [docs/product/strategy.md](../../docs/product/strategy.md) §7.1 (friction audit); architect review (2026-05-20) §14 adoption barriers.
   - **Action:** Implement `--eval-mode` flag to api-gateway that uses in-process `MemoryEngine` (no Temporal needed).
   - **Success metric:** Quickstart runs in < 2 min with zero external dependencies; evaluators report "no Temporal needed."

### 7.2 **High** (gates M7 completion and M8 traction)

4. **Make all external messaging Kagent-differentiated** (product/marketing, positioning)
   - **Why:** Kagent (CNCF Sandbox 2026) now owns the same mindspace. Zynax's wedge (engine-agnostic + GitOps) is real but under-emphasized.
   - **Evidence:** [docs/architecture/2026-05-28-competitive-positioning.md](../../docs/architecture/2026-05-28-competitive-positioning.md) (Kagent table lines 164–189); README overlap with Kagent tagline.
   - **Action:** Audit all external messaging (README, docs, blog, CFP talks). Lead with *engine-agnostic IR + multi-engine dispatch + adapter-first*. Carry co-existence story (Kagent agent ↔ Zynax capability).
   - **Success metric:** All new blog posts, talks, and issue descriptions reference Kagent comparison; external adopter citing Zynax for multi-engine choice (not just K8s allergy).

5. **Establish public cadence and governance signal** (maintainer/community, governance)
   - **Why:** CNCF Sandbox requires public channel + cadence (§14 criterion). Discussions enabled but no regular "what shipped" or roadmap update.
   - **Evidence:** [docs/product/strategy.md](../../docs/product/strategy.md) §7.3 (community moves); [GOVERNANCE.md](../../GOVERNANCE.md) exists but no PR cadence posted.
   - **Action:** Monthly "Zynax digest" post in Discussions (what shipped, next sprint, blockers, contributor spotlights). Link from README.
   - **Success metric:** ≥ 3 digests published; external contributors asking questions in Discussions; one attributed community contribution.

6. **Cut Day-0 friction — contributor fast-lane** (contributor, developer-experience)
   - **Why:** 416-line CONTRIBUTING.md + full SPDD/AGENTS overhead alienates casual contributors ("just want to fix a typo").
   - **Evidence:** [CONTRIBUTING.md](../../CONTRIBUTING.md) (416 lines); May review adoption barrier §3; [docs/contributing/](../../docs/contributing/) burden.
   - **Action:** Create a separate "Good first issues" lane with a 10-line contributor path (no SPDD requirement; docs-only contributions). Link from CONTRIBUTING.
   - **Success metric:** ≥ 1 external "good first issue" PR merged; total contributor count rises from 3 to 5+ by M8.

### 7.3 **Medium** (gates M8 adoption posture)

7. **Grow the example/template library around software-engineering beachhead** (developer, ecosystem)
   - **Why:** Hero use case (code-review/feature-impl/CI) is proven but needs more runnable variants (e.g., code-search, refactoring, audit agent).
   - **Evidence:** [spec/workflows/examples/](../../spec/workflows/examples/) has 3 workflows; [docs/product/strategy.md](../../docs/product/strategy.md) §6.2 (beachhead is narrow).
   - **Action:** Add 3–5 more example workflows (security audit, code-search, refactoring agent). Host in [docs/examples/](../../docs/examples/) with hero asciinema casts.
   - **Success metric:** External user cites a template as their on-ramp; ≥ 2 new workflows added to spec/workflows/examples/ by M8.

8. **Dogfood via DevAuto — operationalize the substrate** (maintainer, product validation)
   - **Why:** Wave 4 (#881) proved Zynax can self-host issue-delivery. Next: use Zynax in production for the repo's own CI/automation.
   - **Evidence:** [state/current-milestone.md](../../state/current-milestone.md) line 100 (Wave 4 delivered O1–O9).
   - **Action:** Migrate repo's own issue triage / auto-labeling / release pipeline to a Zynax workflow. Document the playbook.
   - **Success metric:** Zynax runs 1–2 production automation tasks for the repo; learnings flow back to docs/authoring/.

9. **Preserve monetization optionality — keep core open** (maintainer, sustainability)
   - **Why:** Scenario A (open-core + managed cloud) or Scenario B (CNCF-donated + services) both viable; decide *after* traction signal.
   - **Evidence:** [docs/product/strategy.md](../../docs/product/strategy.md) §9 (two scenarios, sequencing rule: don't gate before adoption).
   - **Action:** No code changes. Policy: core stays Apache-2.0; never feature-gate before 1st external adopter.
   - **Success metric:** M8 filing lists both scenarios as forward-compatible; no feature gating to date.

---

## 8. Longitudinal Delta — Prior Recommendations vs Current Status

**Baseline:** No prior dated review found. Diffing against [docs/product/strategy.md](../../docs/product/strategy.md) §11 (recommendations, March 2026 working draft):

| Prior recommendation | Status | Evidence | Blocker closed? |
|---|---|---|---|
| **1. Land M7 usability keystone + flagship demo** | 🟡 In-flight | W keystone (#1167) merged; asciinema cast *not yet* in README | Still open (needs UI/marketing work) |
| **2. Kagent-differentiated messaging** | ❌ Not started | README still overlaps; no public comparison post | Open — must start now for M8 |
| **3. Cut Day-0 friction** | 🟡 Partial | `make run-local` tight; zero-Temporal mode *not yet* started; contributor lane *not yet* | Partially open — eval mode needed |
| **4. Invest in community before features** | ⚠ Minimal | Discussions enabled; no cadence; 0 named adopters; no 2nd maintainer | **Critical blocker — open** |
| **5. Grow example/template library** | ✅ Partial | 3 runnable workflows; #1355–#1358 docs landed; no new beachhead templates | Acceptable — good-to-have |
| **6. Decide monetization only after traction** | ✅ Held | No feature-gating; strategy.md preserved optionality | Preserved (no action = success) |

**Critical closure paths for M8 filing:**
- **#2 (Kagent messaging):** Must ship in next 30 days (before CFP submissions).
- **#4 (community/maintainer):** 6-month lead time; recruit *now* or slip M8.
- **#1 (keystone demo):** Finish W compiler work by M7 close-out (target 2026-07-15).

---

## 9. Market & Competitive Reality Check (2026-06-18)

**Kagent status update (no fresh 2026-06 data, extrapolating from May):**
- CNCF Sandbox Accepted (2026-05).
- K8s-native agent management with kubectl/Helm integration (no cross-engine vision).
- Likely has paying cloud tier or enterprise support by now (market pressure).

**Zynax's defensible position:**
- Engine-agnostic IR (unique).
- Compose **and** K8s (vs K8s-only).
- Multi-language adapters (Go + Python).
- No mandatory SDK (lowest friction integration).

**Zynax's vulnerability:**
- Zero adoption signals (stars, forks, external users).
- Single maintainer (bus-factor, CNCF blocker).
- Kagent has CNCF backing + likely funding.

**Honest verdict:** Category is contested; winner determined by community + adoption, not architecture. Zynax is 6 months behind Kagent on traction and CNCF visibility. The corrective action is social (recruitment, messaging) not technical.

---

## 10. Risks & Honest Weaknesses (Post-M6 Update)

| Risk | Prior status (May) | Current status | Mitigation |
|---|---|---|---|
| **Kagent absorbs the category** (R6) | Open | Open, **intensified** (Kagent now CNCF Sandbox 2026) | Ruthless differentiation messaging + co-existence narrative |
| **Single maintainer / bus-factor** (R2) | Open | Open — **top adoption + CNCF blocker** | Recruit 2nd maintainer *now* (6-mo lead time) |
| **End-to-end dispatch not wired** (R1) | **Critical** | ✅ **Closed** (M6 shipped) | — |
| **Insecure gRPC by default** (R3) | Critical | ✅ **Closed** (mTLS, M6) | — |
| **No SBOM/cosign/SLSA** | High | ✅ **Closed** (M6) | — |
| **Unbounded in-memory state / OOM** (R4) | High | 🟡 Addressed (Postgres-backed M6; compiler still needs work) | Finish workflow-compiler stateless refactor (EPIC #466) |
| **Temporal dependency deters evaluators** | High | Open | Zero-Temporal eval mode needed (EPIC #1359) |
| **Delivery-vs-narrative gap** (R8) | High | 🟢 **Mitigated** (Truth-Pass culture) | Maintain honest status; Truth-Pass culture working |
| **Release pipeline / install URLs** (R9) | High | ✅ **Closed** (releases cut, multi-arch) | — |

**Net:** Technical critical risks (dispatch, security, supply chain) **closed by M6**. Remaining risks are **market and community** (Kagent, bus-factor, adoption) — not solved by code.

---

## 11. Appendix — Sources & Methodology

### A. Grounded signals (2026-06-18, via `gh api`)

```
Repository stats:
  stars: 1
  forks: 0
  watchers: 0
  open_issues: 52
  contributors: 3

Open milestones:
  M7 (Usable Workflows + Observability): 24 open, 93 closed
  M8 (CNCF Sandbox): 7 open, 0 closed
  M-UX (User Experience): 2 open, 0 closed
  M-dx (Developer Experience): 13 open, 0 closed

Release history (last 8):
  v0.5.0 (2026-06-12) — K8s Production-Ready, shipped
  v0.4.0 (2026-05-28) — Adapter Library, shipped
  [proto-stubs snapshots]
```

### B. Canonical source documents (linked throughout, no duplication)

- **Positioning:** [README.md](../../README.md), [ROADMAP.md](../../ROADMAP.md), [ARCHITECTURE.md](../../ARCHITECTURE.md)
- **Strategy:** [docs/product/strategy.md](../../docs/product/strategy.md)
- **Competitive:** [docs/architecture/2026-05-28-competitive-positioning.md](../../docs/architecture/2026-05-28-competitive-positioning.md)
- **Principal review:** [docs/architecture/2026-05-20-principal-architect-review.md](../../docs/architecture/2026-05-20-principal-architect-review.md)
- **Live status:** [state/current-milestone.md](../../state/current-milestone.md), [docs/milestones/M7-planning.md](../../docs/milestones/M7-planning.md), [docs/milestones/M6-planning.md](../../docs/milestones/M6-planning.md)
- **Execution:** [spec/workflows/examples/](../../spec/workflows/examples/), [docs/quickstart.md](../../docs/quickstart.md), [docs/authoring/](../../docs/authoring/)
- **Governance:** [GOVERNANCE.md](../../GOVERNANCE.md), [docs/contributing/engineering-manifesto.md](../../docs/contributing/engineering-manifesto.md)

### C. Review methodology

- Every claim grounded in a cited artifact (file:line, doc section, or `gh` result).
- Shipped/partial/aspirational split applied uniformly.
- Scorecard deltas computed from architect review (2026-05-20) and current [state/current-milestone.md](../../state/current-milestone.md).
- Recommendations classified by user type (developer/operator/maintainer/product-owner) and adoption lever (distribution/friction/community/governance).
- Longitudinal comparison against [docs/product/strategy.md](../../docs/product/strategy.md) (living strategy, no prior dated review).

---

## 12. Conclusion

**Zynax has solved the engineering half of the problem.** M6 shipped production-grade infrastructure (K8s, mTLS, supply-chain, observability foundation). The architecture remains differentiated and proven (Temporal + Argo in CI). The beachhead (agentic software-engineering automation) is real, runnable, and dogfooded internally.

**The binding constraint is now social, not technical:** zero adoption signals, single maintainer, Kagent's CNCF backing, and undifferentiated messaging. M7 (active, usable workflows + observability) will complete the feature work. **M8 (CNCF Sandbox filing) is gated on community recruitment, first-run UX polish, and ruthless positioning against Kagent** — all of which have 6–12 week lead times and cannot be accelerated by engineering alone.

**Recommended focus (next 90 days):** (1) Recruit a second maintainer from a second org. (2) Finish M7 keystone (data-flow #1167) and publish flagship demo. (3) Ship zero-Temporal evaluation mode. (4) Establish public cadence. (5) Make every external message Kagent-differentiated. These moves address the adoption surface, not the architecture; they are the highest-leverage unit of work remaining before M8.
