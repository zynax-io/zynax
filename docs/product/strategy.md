<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Product Strategy, Adoption & Sustainability

**Document type:** Product Strategy · Adoption Research · Sustainability Thesis
**Status:** Working draft for the maintainer — opinionated, evidence-grounded
**Scope:** Tier 1 (public-safe). Where this doc makes strategic claims, it cites the
repository artifact it is grounded in so nothing here drifts from shipped reality.

> **Messaging principle:** the operational rule for *how* this positioning is expressed — lead with
> the engine-portability wedge, ruthlessly and consistently, with the co-existence story — is the
> single source of truth in [docs/product/positioning.md](positioning.md). This strategy doc is the
> *why*; that doc is the *how we say it*.

> **How to read this.** This is a rewrite and hardening of an earlier informal
> competitive analysis. The earlier draft was directionally interesting but made
> several claims that contradict Zynax's own decision record. This version replaces
> guesswork with the project's ground truth: the [README](../../README.md) positioning,
> the [2026-05-20 principal architect review](../architecture/2026-05-20-principal-architect-review.md),
> the [2026-05-28 competitive positioning](../architecture/2026-05-28-competitive-positioning.md),
> the [ADR register](../adr/INDEX.md), and the [live milestone state](../../state/current-milestone.md).
> Every section separates what is **shipped**, **partial**, and **aspirational**.

---

## Table of contents

1. [Executive summary](#1-executive-summary)
2. [What Zynax actually is — and what it is not](#2-what-zynax-actually-is--and-what-it-is-not)
3. [Market & category](#3-market--category)
4. [Competitive landscape (corrected)](#4-competitive-landscape-corrected)
5. [Differentiators — defensible vs commoditized](#5-differentiators--defensible-vs-commoditized)
6. [Personas & the beachhead](#6-personas--the-beachhead)
7. [Adoption strategy](#7-adoption-strategy)
8. [The CNCF path](#8-the-cncf-path)
9. [Sustainability & ecosystem (monetization)](#9-sustainability--ecosystem-monetization)
10. [Risks & honest weaknesses](#10-risks--honest-weaknesses)
11. [Recommendations (prioritized)](#11-recommendations-prioritized)
12. [Appendix — scorecard, sources, glossary](#12-appendix--scorecard-sources-glossary)

---

## 1. Executive summary

**What Zynax is (verbatim, [README](../../README.md)):** *"The declarative control plane
for AI agent workflows."* *"Zynax is to AI workflows what Kubernetes is to containers — a
control plane that abstracts the execution layer behind a declarative, versionable API."*

**The one defensible wedge.** Zynax compiles a `kind: Workflow` YAML manifest into an
engine-neutral **Workflow IR** and dispatches it to a **pluggable engine** (Temporal and
Argo are shipped and tested in the CI engine matrix; LangGraph is planned), invoking
capabilities through an **adapter-first, no-SDK** contract. The combination —
**declarative + engine-agnostic + adapter-first + GitOps-native** — is genuinely rare and
is the only thing competitors cannot trivially copy. Everything else (an adapter library, a
YAML schema, a CLI) is commoditized.

**The single biggest risk is not technical — it is category + traction.** The
"control plane for AI agents" category is now crowded, and Zynax has a *named direct
competitor it must out-position*: **Kagent**, a CNCF Sandbox project (2026) using almost
the same tagline. Meanwhile the project's own
[architect review](../architecture/2026-05-20-principal-architect-review.md) recorded
**zero stars, zero forks, and a single maintainer** as the baseline. For an
open-source control plane, *that* — not architecture — is what gates adoption and a CNCF
Sandbox bid.

**Where the engineering actually is.** M6 (v0.5.0) shipped K8s production-readiness: mTLS,
SBOM/cosign/SLSA supply-chain, Postgres-backed horizontal scale, NATS JetStream event bus,
Helm charts, and — critically — the **end-to-end capability dispatch path** that the May
review flagged as missing. **M7 (v0.6.0, active)** is the usability milestone: workflow
data-flow bindings (the keystone), execution log/event streaming, and OpenTelemetry +
Uptrace observability. **M8** is the CNCF Sandbox submission. See
[state/current-milestone.md](../../state/current-milestone.md).

**The recommended focus, in one line:** *pick one hero use case, make the first 15 minutes
unforgettable, and recruit a community before adding features.* The architecture is already
ahead of the adoption surface; the next unit of leverage is **distribution, not design**.

| Question | Short answer |
|---|---|
| Is the architecture differentiated? | **Yes** — engine-agnostic IR + adapter-first is real and partly proven (Temporal+Argo in CI matrix). |
| Is the category defensible? | **Contested** — Kagent occupies the same framing with CNCF backing. Messaging must be ruthless. |
| What blocks adoption today? | **Traction & Day-0 friction**: zero external users, single maintainer, Temporal dependency, YAML-only authoring. |
| What is the beachhead? | **Agentic software-engineering automation** — the only use case with shipped, runnable proof. |
| Can it monetize? | **Yes, later** — two viable models, but both are gated on adoption and interact with CNCF neutrality. Don't monetize before traction. |

---

## 2. What Zynax actually is — and what it is not

The earlier draft framed Zynax as *"Kubernetes + Crossplane + Argo + Backstage, but for
AI."* **That overstates the scope and contradicts the project's own one-way-door
decisions.** Zynax is deliberately *narrower*: it is a **control plane**, not a container
runtime, not a developer portal, and not a workflow engine.

This is not an accident — it is enforced by the architecture mandates in
[AGENTS.md](../../AGENTS.md) and locked by ADRs ([docs/adr/INDEX.md](../adr/INDEX.md)):

- **ADR-011** — declarative YAML control plane (intent, not code).
- **ADR-012** — Workflow IR as the engine-agnostic intermediate representation.
- **ADR-014** — event-driven **state machines**, not DAGs (enables loops and
  human-in-the-loop natively).
- **ADR-015** — **pluggable** engines behind a `WorkflowEngine` interface; Zynax never
  *becomes* an engine.

### The three-layer model (the real architecture)

| Layer | What it is | Where | Boundary rule |
|---|---|---|---|
| **1 — Intent** | `kind: Workflow` / `AgentDef` / `Policy` YAML | `spec/` | Never imported by services |
| **2 — Communication** | gRPC contracts + AsyncAPI events | `protos/` | No business logic |
| **3 — Execution** | Engines (Temporal, Argo, …) + adapters | `services/`, `agents/` | Always behind an interface |

### Shipped vs partial vs aspirational (be honest)

| Capability | Status | Evidence |
|---|---|---|
| Declarative YAML → IR → execution | ✅ Shipped | M2–M6; `zynax apply` runs end-to-end on the local stack |
| Engine-agnostic dispatch (Temporal **+** Argo) | ✅ Shipped | Both legs in the e2e-smoke CI engine matrix |
| Adapter-first, no-SDK capabilities (5 adapters) | ✅ Shipped | http · git · ci (Go) · llm · langgraph (Python) |
| mTLS, SBOM/cosign/SLSA, Postgres scale, event bus | ✅ Shipped | M6 / v0.5.0 |
| OpenTelemetry + Uptrace observability | 🟡 Partial | M7 EPIC O — core instrumentation merged; backend UI/log export landing |
| Workflow data-flow (output→input bindings) | 🟡 Partial | M7 EPIC W **keystone**; compiler still gates `output:` ([state](../../state/current-milestone.md)) |
| LangGraph as a full **engine** | 📅 Aspirational | langgraph-adapter exists as a *capability provider*, not a `WorkflowEngine` |
| Web UI | 📅 Aspirational | M8 roadmap; none today |

> **Why this matters for strategy.** The honest "real vs aspirational" split is itself a
> credibility asset. The project ran an explicit **"Truth Pass"** (M5.A) that removed a
> premature CNCF badge and phantom changelog entries. Lead with that integrity — it is rare
> and it directly addresses the "delivery-vs-narrative gap" the architect review named as
> its single most important finding.

---

## 3. Market & category

Zynax sits at the intersection of three maturing markets, and its positioning depends on
which one a buyer thinks they are shopping in:

1. **AI agent orchestration** (LangGraph, CrewAI, AutoGen, OpenAI Agents, Kagent) — fast,
   noisy, framework-centric, mostly Python, mostly library-shaped.
2. **Durable workflow / execution engines** (Temporal, Restate, Argo Workflows, Dapr
   Workflows) — mature, infrastructure-shaped, not AI-specific.
3. **AI/ML platforms & MLOps** (Kubeflow, Flyte, SageMaker, Vertex) — heavyweight, model-
   lifecycle-centric, **adjacent not direct** (Zynax is about *agentic workflows*, not
   model training/serving).

**The category Zynax is creating** is the *control plane that sits above (1) and (3) and
orchestrates across (2)*: write the workflow once, run it on whatever engine the org
already operates, invoke whatever agent/framework already exists, govern it with policy,
ship it as GitOps YAML.

**The category is real but crowded.** The
[architect review](../architecture/2026-05-20-principal-architect-review.md) is blunt:
*"'Control plane for AI agent workflows' is now a crowded category. Zynax's unique angle …
is genuine but must be ruthlessly defended in messaging."* The strategic implication is
that **category education and differentiation are now more valuable than additional
features.**

---

## 4. Competitive landscape (corrected)

> **The earlier draft's largest error was omitting Kagent entirely** while comparing Zynax
> to Flyte/Kubeflow/Argo. Per the repo's own
> [competitive positioning doc](../architecture/2026-05-28-competitive-positioning.md) and
> the architect review (risk **R6**, gap **G19**), **Kagent is the direct competitor.**
> Flyte and Kubeflow are *adjacent MLOps*, not direct.

### 4.1 Lead comparison: Zynax vs Kagent

Both pitch a "control plane for AI agents." The distinction is sharp and must be the
headline of all external messaging:

| Dimension | **Zynax** | **Kagent** |
|---|---|---|
| Core abstraction | Engine-agnostic Workflow **IR** | Kubernetes **CRDs** (Pod-per-agent) |
| Engine portability | ✅ Temporal + Argo (LangGraph planned) | ❌ ADK / K8s lock-in |
| Deployment | Compose **and** K8s/Helm | K8s-only (Kind required) |
| Agent integration | Adapter-first, **no SDK** (any gRPC service) | Kubernetes-native agents |
| Workflow authoring | Declarative YAML, compile-time IR validation | kubectl-native CRDs |
| GitOps | ✅ Workflow YAML in git, `zynax apply` | ❌ kubectl-imperative |
| Web UI | ❌ (M8) | ✅ included |
| Multi-LLM out of the box | Via llm-adapter | ✅ built-in ModelConfig |
| CNCF status | Not yet (M8 target) | ✅ Sandbox 2026 |
| Production-proven | No (v0.5.0) | Partial |

**Positioning takeaways:**
- **Zynax wins** when an org refuses engine lock-in, runs agents in multiple languages,
  wants GitOps-native YAML reviewed in PRs, and wants no-SDK registration.
- **Kagent wins** today when an org wants K8s-native agent management with a web UI and a
  CNCF-backed ecosystem.
- **They are complementary**, not strictly rivals: a Kagent-managed agent can register as a
  Zynax capability via the `AgentService` gRPC contract. Use this as a co-existence story,
  not a fight Zynax cannot yet win on ecosystem size.

### 4.2 Full landscape

| Project | Relationship | Why it matters | CNCF |
|---|---|---|---|
| **Kagent** | **Direct competitor** | Same framing, CNCF-backed — the strategic risk to beat | Sandbox |
| **Temporal** | Wrapped runtime | Largest technical overlap; Zynax runs *on* it. Many buyers will use Temporal directly | No |
| **Restate** | Wrappable runtime | Durable execution; a future engine adapter | No |
| **Dapr Workflows** | Adjacent | General-purpose, K8s-first; not AI-specific | Incubating |
| **Argo Workflows** | Engine (shipped) | DAG-based; Zynax integrates it as an engine, doesn't replace it | Graduated |
| **LangGraph / CrewAI / AutoGen** | **Wrapped, not rivals** | Frameworks Zynax invokes as capabilities (ADR-011 keeps Zynax out of framework territory) | No |
| **Flyte / Kubeflow** | Adjacent (MLOps) | Model lifecycle, not agentic workflows — *not* direct competitors | OSS / Incubating |
| **Backstage / Crossplane** | Vision analogy only | Inspirations for "platform" and "control plane" framing; **not** what Zynax builds | CNCF |

---

## 5. Differentiators — defensible vs commoditized

The earlier draft listed six differentiators as if all were equally strong. They are not.
The external and principal reviews both warn that **the adapter library is commoditized
infrastructure** — anyone can wrap an API. The durable moat is narrower and must be
defended deliberately.

| Differentiator | Defensibility | Status |
|---|---|---|
| **Engine-agnostic IR + multi-engine dispatch** | **High** — hard to copy, proven in CI matrix | ✅ Shipped (Temporal+Argo) |
| **Capability routing via a stable `AgentService` contract** | **High** — the integration moat | ✅ Shipped |
| **Event-driven state machines (loops, human-in-the-loop)** | Medium-high — design-level edge over DAG tools | ✅ Shipped (ADR-014) |
| **Declarative + GitOps-native YAML** | Medium — table stakes soon, but well-executed (idempotent apply via canonical hash) | ✅ Shipped |
| **Policy / governance / quotas** | Medium — enterprise value, partially built | 🟡 Partial (policy + rate-limit shipped) |
| **Adapter library (http/git/ci/llm/langgraph)** | **Low — commoditized** | ✅ Shipped, but not a moat |
| **No mandatory SDK** | Medium — reduces adoption friction; a *go-to-market* asset more than a moat | ✅ Shipped (ADR-013) |

**Strategic implication:** market the *portability + capability-routing* story relentlessly;
treat the adapter library as a convenience and an ecosystem on-ramp, **not** as the pitch.

---

## 6. Personas & the beachhead

The earlier draft proposed "Platform Engineering teams" as the target. That is a credible
*long-term* buyer, but it is the wrong **beachhead** today: it implies a long enterprise
sales cycle, and Zynax has **no reference adopter** to anchor it. The beachhead should be
where Zynax can *show, not tell* — and the only place with shipped, runnable proof is
**agentic software-engineering automation**.

### Personas

| Persona | What they want | Zynax fit today | Horizon |
|---|---|---|---|
| **AI-forward dev team** (hero) | Automate code review / PR / CI with agents, in YAML they can PR-review | **Strong** — three runnable example workflows | Now |
| **Platform engineer** | One engine-agnostic control plane for many teams; no per-team SDK lock-in | Strong architecture, weak proof (no adopter) | M7→M8 |
| **AI-infra / agent-platform team** | Shared inference + workflow substrate, multi-language agents | Good (adapter-first, multi-language) | M7→M8 |
| **Enterprise governance buyer** | RBAC, policy, audit, GitOps, approval gates | Partial (policy/quota shipped; RBAC/SSO not) | Post-M8 |

### The hero use case

> **"Ship a multi-agent code-review / feature-implementation / CI workflow as a YAML
> manifest, run it on Temporal *or* Argo without a rewrite, and watch it execute end-to-end
> with traces in Uptrace — in under 15 minutes."**

This is credible *because the artifacts already exist*:
[spec/workflows/examples/code-review.yaml](../../spec/workflows/examples/code-review.yaml),
[feature-implementation.yaml](../../spec/workflows/examples/feature-implementation.yaml),
and [ci-pipeline.yaml](../../spec/workflows/examples/ci-pipeline.yaml) are real, runnable
workflows that exercise loops, human-in-the-loop, and (with M7) cross-state data-flow.
There is also a self-hosted **DevAuto** automation track using the same substrate — Zynax
dogfooding its own dev automation is a powerful proof point and a content engine.

---

## 7. Adoption strategy

This is the heart of the document. The architecture is ahead of the adoption surface; the
next leverage is **distribution and Day-0 experience**, not features.

### 7.1 Day-0 experience audit

The actual quickstart ([docs/quickstart.md](../../docs/quickstart.md)) is roughly eight
steps: bootstrap → install CLI → `make run-local` → (optional) observability stack → apply
a workflow → status → logs → teardown. That is good, but the
[architect review](../architecture/2026-05-20-principal-architect-review.md) named concrete
friction that still depresses conversion:

| Friction | Impact | Recommended reduction |
|---|---|---|
| **Temporal dependency** | "Needs a Temporal cluster" makes evaluators bounce | Ship a **zero-Temporal lightweight / in-process engine mode** for evaluation |
| **Large tools image pull** | Slow first run | A slim, one-command demo path (`make demo`) that pulls minimal images |
| **416-line CONTRIBUTING** + AI-context/SPDD/AGENTS overhead | Alienates casual contributors | A 10-line "fix a typo / first PR" fast lane separate from the full contract |
| **YAML-only authoring** | Power users want imperative composition | Document the trade-off; consider a thin builder later (not a priority) |
| **Time-to-first-workflow** | The single most important metric | Make the hero demo the literal first thing in the README, with an asciinema cast |

### 7.2 Measure adoption, not features

Replace feature-count thinking with an adoption funnel. Baseline (per the May review):
**0 stars, 0 forks, 0 external adopters.**

| Metric | Why | Target shape |
|---|---|---|
| **Time-to-first-working-workflow** | The conversion gate | < 15 min, one command |
| GitHub stars / forks / Discussions | Awareness & intent | First non-zero, then steady slope |
| External adopters (named) | CNCF prerequisite | ≥ 1 before M8 filing |
| Ecosystem adapters (community-built) | Extensibility proof | First community adapter |
| Contributors / second maintainer | Bus-factor & CNCF | ≥ 2 maintainers from ≥ 2 orgs |

### 7.3 Community & governance (the #1 blocker)

The review's risk **R2** and gap **G18** are explicit: a single maintainer and no community
channel block both adoption and CNCF Sandbox. Concrete moves:

- **Recruit a second maintainer from a second org** — start now, via the agent/cloud-native
  communities; this is a *social* process with long lead time.
- **Lower the contribution bar** — a "good first issue" lane; a short contributor path that
  doesn't require reading the full SPDD/AGENTS machinery.
- **Establish a public cadence** — Discussions for RFCs/roadmap (already enabled), a visible
  changelog of demos, and a regular "what shipped" note.
- **Lead with the Truth-Pass culture** — honesty about what's shipped is a trust asset for
  early adopters and CNCF reviewers alike.

### 7.4 Sequenced adoption plan (milestone-anchored)

| Phase | Gate | Adoption action |
|---|---|---|
| **Now (M7)** | Workflows become *usable + observable* (data-flow + Uptrace) | Publish the hero demo (blog + asciinema); zero-Temporal eval mode; README-first quickstart |
| **M7 → M8** | First external adopter + 2nd maintainer | Community push, conference/CFP, ecosystem adapter program |
| **M8** | CNCF Sandbox submission | File once traction + governance prerequisites are met |
| **Post-M8** | Production references | Enterprise-readiness (RBAC/SSO/multi-tenancy), operator runbooks |

---

## 8. The CNCF path

Structural alignment is strong; **community traction is the blocking gap, not technology.**
From the review's §14 criteria table, updated for post-M6 reality:

| CNCF Sandbox criterion | Status |
|---|---|
| Apache-2.0 license | ✅ |
| Code of Conduct | ✅ |
| Public roadmap + ADRs/RFCs | ✅ ([ROADMAP](../../ROADMAP.md), 36+ ADRs) |
| SBOM / cosign / SLSA / mTLS | ✅ (delivered M6 — was a gap in May) |
| OpenSSF Scorecard | ✅ |
| **≥ 2 maintainers from ≥ 2 orgs** | ❌ single maintainer |
| **≥ 1 external adopter** | ❌ (0 stars/forks baseline) |
| **Public community channel / cadence** | ❌/🟡 (Discussions enabled; no cadence) |
| **Trademark policy** | ❌ |
| CNCF TOC sponsor | ❌ not yet recruited |

**M8 planning** already scopes the governance artifacts (MAINTAINERS file, simplified
GOVERNANCE, trademark policy) under M8.A and the Sandbox filing under M8.B. The honest
verdict: **do not file until the community/maintainer/adopter prerequisites are real** —
filing early and being rejected is worse than filing late and being accepted.

---

## 9. Sustainability & ecosystem (monetization)

> **Framing note.** Zynax aims for CNCF Sandbox (M8), which favors vendor-neutral stewardship.
> This section is written as *project sustainability and ecosystem*, not as a single-vendor
> pricing sheet, precisely so it stays compatible with that goal. Two viable models are
> presented; the recommendation is about **sequencing**, not picking a winner now.

### Scenario A — Open-core + managed cloud (Temporal / Astronomer pattern)

- **OSS core** (Apache-2.0): the control plane, IR, engines, adapters — everything shipped.
- **Enterprise tier**: SSO/RBAC, multi-tenancy isolation, SLAs, advanced policy/quota
  (note: basic **policy and rate-limit already exist**, so this is an extension, not new
  ground), audit/compliance reporting.
- **Zynax Cloud**: a hosted control plane so teams skip the Temporal/K8s operational burden
  — directly attacks the Day-0 "needs a cluster" friction.
- **Pros:** clear revenue, funds maintainers. **Cons:** vendor-capture optics complicate a
  neutral CNCF donation; risks feature-gating the core too early.

### Scenario B — CNCF-donated core + services

- **Donate the core** to CNCF for genuine neutrality; grow the broadest possible adoption.
- **Revenue via services**: support, training, certification, certified-partner program,
  and *managed add-ons* (e.g. hosted observability) that don't gate core features.
- **Pros:** maximizes adoption and neutrality; cleanest CNCF story. **Cons:** services
  revenue scales with headcount, not software; slower to monetize.

### The tension, made explicit

CNCF Sandbox neutrality and a single-vendor commercial model pull in opposite directions.
The market has worked examples of threading this needle (Temporal, Astronomer/Airflow,
GitLab) — typically by keeping a genuinely useful neutral core and monetizing *operational
convenience and enterprise governance* rather than crippling the OSS product.

### Recommendation (sequencing)

1. **Now → M8:** optimize for *adoption and neutrality*. Keep the core fully open and
   donation-friendly. **Do not feature-gate before there is traction** — gating an
   un-adopted product just suppresses the adoption you need.
2. **Preserve optionality:** nothing about the current architecture forecloses Scenario A
   later (a hosted control plane and enterprise governance are natural future tiers).
3. **Decide the model *after* a traction signal** (first adopters, first community
   contributors). Monetization posture should follow evidence of demand, not precede it.

---

## 10. Risks & honest weaknesses

Folded from the review's risk register, updated to current (post-M6) state:

| Risk | Then (May) | Now | Mitigation |
|---|---|---|---|
| **Kagent absorbs the category** (R6) | Open | Open — **top strategic risk** | Ruthless engine-agnostic differentiation in all messaging; co-existence story |
| **Single maintainer / bus-factor** (R2) | Open | Open — **top adoption + CNCF blocker** | Recruit 2nd maintainer from 2nd org now |
| End-to-end dispatch not wired (R1) | **Critical** | ✅ Closed (M6) | — |
| Insecure gRPC by default (R3) | Critical | ✅ Closed (mTLS, M6) | — |
| No SBOM/cosign/SLSA (supply chain) | High | ✅ Closed (M6) | — |
| Unbounded in-memory state / OOM (R4) | High | 🟡 Partially addressed (Postgres-backed repos M6; compiler store revisited in M7) | Finish stateless compiler |
| Temporal dependency deters evaluators | High | Open | Zero-Temporal lightweight eval mode |
| Delivery-vs-narrative gap (R8) | High | 🟢 Mitigated — Truth-Pass culture | Keep status tables honest |
| Release pipeline / install URLs (R9) | High | ✅ Closed (releases cut, multi-arch) | — |

**Net:** the May review's *technical* critical risks (dispatch, security, supply chain) are
largely closed by M6. What remains are **market and community** risks — which are exactly
the ones that don't get solved by writing more code.

---

## 11. Recommendations (prioritized)

Milestone-anchored, replacing the earlier draft's generic list:

1. **Land the M7 usability keystone (data-flow bindings) and publish one flagship
   end-to-end demo.** The hero code-review/feature-implementation workflow, running on
   Temporal *and* Argo, with Uptrace traces, as an asciinema cast in the README. *Nothing
   matters more for conversion than a believable first 15 minutes.*
2. **Make all external messaging Kagent-differentiated.** Lead every pitch with
   *engine-agnostic + adapter-first + GitOps*; carry the co-existence story; never echo
   Kagent's tagline.
3. **Cut Day-0 friction.** Ship a zero-Temporal lightweight evaluation mode, a one-command
   demo, and a short contributor fast-lane separate from the full SPDD/AGENTS machinery.
4. **Invest in community before features.** Recruit a second maintainer from a second org;
   establish a public cadence; court the first named adopter. This is the long-lead-time
   work that gates M8.
5. **Grow the example/template library** around the software-engineering beachhead, and
   dogfood via the DevAuto track as a content engine.
6. **Decide monetization posture only after a traction signal** — keep the core open and
   neutral until then; preserve Scenario-A optionality.

---

## 12. Appendix — scorecard, sources, glossary

### A. Grounded scorecard (May 2026 review, annotated with post-M6 deltas)

| Dimension | May score | Post-M6 reality |
|---|---:|---|
| Overall architecture | 6.5 / 10 | ↑ end-to-end dispatch now wired |
| Security | 4.0 / 10 | ↑↑ mTLS + SBOM/cosign/SLSA shipped (M6) |
| Performance | 4.0 / 10 | → benchmarks/fuzz/load are M7 EPIC R |
| Scalability | 4.5 / 10 | ↑ Postgres-backed horizontal scale (M6) |
| Maintainability | 8.0 / 10 | → still strong (ADRs, hexagonal, ≥90% domain coverage) |
| Product-market fit | 7.0 / 10 | → "promising category, premature claim"; gated on traction |
| CNCF alignment | 6.5 / 10 | ↑ supply-chain done; community remains the gap |

> These are the project's *own* assessment scores, not invented here. The earlier draft's
> 1–10 adoption table was ungrounded and has been intentionally **not** reproduced.

### B. Source documents (do not duplicate — these are the canonical references)

- Positioning & overview: [README.md](../../README.md), [ROADMAP.md](../../ROADMAP.md),
  [ARCHITECTURE.md](../../ARCHITECTURE.md), [AGENTS.md](../../AGENTS.md)
- Competitive: [docs/architecture/2026-05-28-competitive-positioning.md](../architecture/2026-05-28-competitive-positioning.md)
- Strategic review (scores, risks, CNCF table): [docs/architecture/2026-05-20-principal-architect-review.md](../architecture/2026-05-20-principal-architect-review.md)
- Scope guardrails: [docs/adr/INDEX.md](../adr/INDEX.md) (ADR-011/012/014/015)
- Live status: [state/current-milestone.md](../../state/current-milestone.md),
  [docs/milestones/M7-planning.md](../milestones/M7-planning.md),
  [docs/milestones/M6-planning.md](../milestones/M6-planning.md)
- Day-0 & hero use case: [docs/quickstart.md](../../docs/quickstart.md),
  [docs/developer-guide.md](../../docs/developer-guide.md),
  [docs/authoring/workflows.md](../authoring/workflows.md),
  [docs/authoring/experts.md](../authoring/experts.md),
  [spec/workflows/examples/](../../spec/workflows/examples/)
- Community / CNCF readiness: [docs/contributing/engineering-manifesto.md](../contributing/engineering-manifesto.md),
  [CONTRIBUTING.md](../../CONTRIBUTING.md), [GOVERNANCE.md](../../GOVERNANCE.md)

### C. Glossary

- **Control plane** — the layer that decides *what* runs and *in what order*, abstracting
  the engine that *executes*. Zynax is this layer; it is not an engine.
- **Workflow IR** — the engine-neutral intermediate representation a `kind: Workflow`
  compiles to (ADR-012); the portability moat.
- **Adapter / capability** — any gRPC service implementing the `AgentService` contract
  becomes an invokable capability — no SDK required (ADR-013).
- **Engine** — a runtime that executes the IR (Temporal, Argo shipped; LangGraph planned),
  behind the `WorkflowEngine` interface (ADR-015).
- **Beachhead** — the narrow first market where the product can win decisively before
  expanding. Here: agentic software-engineering automation.
