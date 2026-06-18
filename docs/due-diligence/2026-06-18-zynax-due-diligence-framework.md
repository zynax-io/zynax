<!--
SPDX-License-Identifier: Apache-2.0
-->

# Zynax — Investment-Grade Technical & Strategic Due-Diligence Framework

> **Document type:** Meta-framework + executable agent prompt library
> **Date:** 2026-06-18
> **Target:** The Zynax repository, evaluated as a candidate for a tens-of-millions
> investment, full acquisition, CNCF Sandbox sponsorship, or Fortune-500 platform adoption.
> **Status:** Framework v1.0 — *not a completed diligence report.* This document is the
> machinery that **produces** the report.

---

## 0. How To Use This Framework

### 0.1 What this is — and is not

This document is **not** a due-diligence report. It is the **framework and prompt library**
that generates one. Running it end-to-end (one orchestrator + 26 specialized agents) produces
a defensible, evidence-graded report capable of exceeding 100 pages.

The framework is engineered to the standard expected by **VC/PE investment committees,
Fortune-500 enterprise-architecture boards, CNCF Technical Oversight Committee reviewers, Open
Source Program Offices, security consultancies, and Staff+/Distinguished engineers.**

### 0.2 The operating model

```
                          ┌─────────────────────────────────────┐
                          │  PART 1 — Repository Understanding   │
                          │  (shared context packet — read-only) │
                          └──────────────────┬──────────────────┘
                                             │ distributed to every agent
                                             ▼
   ┌─────────────────────────────────────────────────────────────────────────┐
   │              PART 4 — MASTER ORCHESTRATOR (single agent)                  │
   │  assigns scope · prevents overlap · collects · resolves contradictions ·  │
   │  weights & aggregates scores · writes executive summary & verdict         │
   └───────────────┬──────────────────────────────────────┬───────────────────┘
                   │ dispatches (Waves A→D)                │ merges
                   ▼                                       ▲
   ┌───────────────────────────────────────────────────────────────────────────┐
   │     PART 5 — 26 SPECIALIZED AGENTS (run independently, in parallel)        │
   │  Architecture · Security · Product · Market · Engineering · Performance …  │
   │  each returns a structured finding packet (score · confidence · evidence)  │
   └───────────────────────────────────────────────────────────────────────────┘
```

### 0.3 How to run it

1. **Read Part 1** — the reconstructed company/product/technology picture. Every agent gets
   this verbatim as its context packet.
2. **Read Part 2** — the methodology: dimensions, the 0–10 scoring scale, confidence bands,
   and the evidence taxonomy. These are binding on all agents.
3. **Read Part 3** — the investigation strategy: which agent owns which files, the
   parallelization waves, and the anti-overlap matrix.
4. **Dispatch using Part 4** — paste the orchestrator prompt into a controller session.
   The orchestrator owns wave sequencing and final synthesis.
5. **Run the Part 5 prompts** — each is self-contained and portable: paste it into a fresh
   Claude session (or the Agent tool) together with the Part 1 context packet.
6. **Assemble using Parts 6–10** — report template, risk-scoring, investment-recommendation,
   executive-presentation outline, and the final 100+ page document structure.

### 0.4 The non-negotiable evidence rule (binding on every agent)

> **Every factual claim must carry an evidence citation in the form `repo-path:line`
> (or `repo-path` for a whole file), a command + its output, or an external URL.
> Any claim that cannot be evidenced must be written as `UNKNOWN — not found`,
> never asserted. Aspirational/roadmap statements must be labelled `CLAIMED`, and
> separated from `VERIFIED` facts. The single fastest way to fail this diligence is to
> repeat a marketing claim as if it were a verified fact.**

This rule exists because the subject repository has a **documented history of
delivery-vs-narrative drift** (see Part 1, §1.10) that was later corrected by an internal
"Truth Pass." Detecting and scoring that class of drift is a first-class objective here.

### 0.5 Table of contents

- **Part 1 — Repository Understanding Summary** (shared context packet)
- **Part 2 — Due-Diligence Methodology** (dimensions, scoring, evidence)
- **Part 3 — Investigation Strategy** (agent waves, ownership matrix, dry-run)
- **Part 4 — Master Orchestration Prompt**
- **Part 5 — 26 Specialized Agent Prompts**
  - 5.1 Architecture · 5.2 Security · 5.3 Product · 5.4 Market · 5.5 Engineering
  - 5.6 Performance · 5.7 Testing · 5.8 Open Source · 5.9 DevOps · 5.10 Documentation
  - 5.11 Developer Experience · 5.12 AI Workflow · 5.13 Governance · 5.14 Technical Debt
  - 5.15 Maintainability · 5.16 Scalability · 5.17 Investment Analysis · 5.18 Business Strategy
  - 5.19 Competitive Analysis · 5.20 Enterprise Adoption · 5.21 CNCF Readiness
  - 5.22 OpenSSF Readiness · 5.23 Risk Assessment · 5.24 Repository Health
  - 5.25 Future Roadmap · 5.26 Innovation
- **Part 6 — Standard Report Template**
- **Part 7 — Risk Scoring Framework**
- **Part 8 — Investment Recommendation Framework**
- **Part 9 — Executive Presentation Outline**
- **Part 10 — Final Due-Diligence Document Structure**
- **Appendices** — A: Master scoring rubric · B: Evidence standard · C: Glossary · D: Contradiction register

---

# Part 1 — Repository Understanding Summary

> **Status of this section:** Reconstructed from a multi-source repository sweep (README,
> ROADMAP, AGENTS.md, 36 ADRs, `docs/product/strategy.md`, six dated architecture/competitive
> reviews, `state/current-milestone.md`, ~30 SPDD canvases, CI workflows, governance docs).
> This is the **shared context packet**: hand it to every Part 5 agent verbatim. Agents must
> still **independently verify** any claim they rely on — this summary is a map, not ground
> truth.

### 1.1 One-paragraph thesis

**Zynax is a declarative control plane for AI-agent workflows** — positioned as "what
Kubernetes is to containers" for AI workflows (`README.md`). Users author a workflow once in
YAML (Layer 1), it compiles to an **engine-neutral Workflow Intermediate Representation (IR)**
(Layer 2 contracts), and executes on any pluggable engine — Temporal, Argo, others — or routes
to any capability provider behind a stable gRPC contract (Layer 3). The core promise is
**portability and decoupling**: escape per-engine and per-framework lock-in. Evidence:
`README.md`, `docs/product/strategy.md`, `AGENTS.md`, `docs/adr/ADR-012-workflow-ir.md`,
`docs/adr/ADR-015-pluggable-workflow-engines.md`.

### 1.2 Mission & problem statement

The problem: "Workflows defined for Temporal cannot run on LangGraph. Without a control plane,
every engine requires a different workflow definition" (paraphrase, `docs/architecture/`
principal-architect review). Zynax's mission is to provide a versionable, declarative,
engine-neutral API so organizations write AI workflows once and run them anywhere, with
GitOps-native authoring and no mandatory SDK. Primary sources: `README.md`,
`docs/product/strategy.md` §1–§2, `ROADMAP.md` preamble.

### 1.3 Philosophy & engineering principles

- **Three-layer separation (non-negotiable):**
  - Layer 1 — **Intent**: `kind: Workflow | AgentDef | Policy | Capability` YAML in `spec/`.
    Never imported by services.
  - Layer 2 — **Communication**: gRPC contracts in `protos/zynax/v1/` + AsyncAPI events in
    `spec/asyncapi/`. No business logic.
  - Layer 3 — **Execution**: pluggable engines + adapters. Always behind an interface.
  - Source: `AGENTS.md` §The Three-Layer Separation; `docs/adr/ADR-011`, `ADR-012`, `ADR-015`.
- **Five non-negotiable mandates** (`AGENTS.md`): (1) no shared DB across services
  (`ADR-008`); (2) no Layer 1→3 coupling; (3) contracts before implementations (`ADR-019`);
  (4) declarative-first (`ADR-011`); (5) event-driven state machines over DAGs (`ADR-014`).
- **SPDD — Structured Prompt-Driven Development** (`ADR-019`, `docs/patterns/spdd-guide.md`):
  every `feat:` PR requires a **REASONS Canvas** committed *before* implementation code; the
  canvas moves Draft → Aligned (human) → Implemented. "Prompt-first" rule: requirements change
  → update canvas → then patch code, never the reverse. ~30 canvases live under `docs/spdd/`.
  The pipeline is driven by a consolidated, milestone-agnostic command set (PR #1400): five
  verbs — `/plan` (analysis→story→canvas→security-review→align), `/deliver`
  (implement→PR→CI→merge→verify), `/review`, `/reconcile`, `/learn` — plus a `/milestone`
  lifecycle command, with the SPDD/delivery/review building blocks under `.claude/commands/lib/`
  and domain personas under `.claude/commands/experts/`. Commands are safe-by-default (PLAN
  unless `--execute`).
- **Hexagonal services:** each Go service has `internal/{domain,api,infrastructure}`; domain
  packages have zero proto/gRPC imports. Source: `services/AGENTS.md`.
- **Layered testing (`ADR-016`):** BDD at gRPC boundaries (`protos/tests/`), unit ≥90% on
  `internal/domain/`, `buf breaking` as a CI gate. `.feature` files committed before code.
- **Engineering manifesto:** 15 enforced principles + DORA targets in
  `docs/contributing/engineering-manifesto.md`.

### 1.4 Target users, personas, market

Hero beachhead (from `docs/product/strategy.md` §6): "Ship a multi-agent
code-review / feature-implementation / CI workflow as a YAML manifest, run it on Temporal *or*
Argo without a rewrite, and watch it execute end-to-end with traces in Uptrace — in under 15
minutes."

| Persona | Want | Fit today | Horizon |
|---|---|---|---|
| AI-forward dev team (hero) | Automate code review/PR/CI with agents in PR-reviewable YAML | Strong | Now |
| Platform engineer | One engine-agnostic control plane, no per-team SDK lock-in | Strong arch, weak proof (no named adopter) | M7→M8 |
| AI-infra / agent-platform team | Shared inference + workflow substrate, multi-language agents | Good (adapter-first) | M7→M8 |
| Enterprise governance buyer | RBAC, policy, audit, GitOps, approval gates | Partial (policy yes; RBAC/SSO no) | Post-M8 |

Category: intersection of (1) AI-agent orchestration (LangGraph, CrewAI, AutoGen, OpenAI
Agents, **Kagent**), (2) durable workflow engines (Temporal, Restate, Argo, Dapr Workflows),
(3) adjacent AI/ML platforms (Kubeflow, Flyte). Zynax's claimed category: *the control plane
above (1) that orchestrates across (2), decoupled from any single engine.*

### 1.5 Differentiators & moat (as claimed in `docs/product/strategy.md` §5)

| Differentiator | Claimed defensibility | Claimed status |
|---|---|---|
| Engine-agnostic IR + multi-engine dispatch | HIGH | Shipped (Temporal+Argo in CI matrix) |
| Capability routing via stable `AgentService` gRPC | HIGH (integration moat) | Shipped (`ADR-010`, `ADR-013`) |
| Event-driven state machines (loops, human-in-the-loop) | Medium-high | Shipped (`ADR-014`) |
| Declarative + GitOps-native YAML, idempotent apply | Medium | Shipped |
| No mandatory SDK | Medium | Shipped (`ADR-013`) |

Commoditized (low moat, acknowledged): the adapter library (http/git/ci/llm/langgraph), and
policy/governance/quotas (enterprise table-stakes).

### 1.6 Named competitor — Kagent

**Kagent (CNCF Sandbox 2026) is the explicitly named direct competitor** with a near-identical
tagline. Source: `docs/architecture/2026-05-28-competitive-positioning.md`; `README.md`.

| Dimension | Zynax | Kagent |
|---|---|---|
| Core abstraction | Engine-agnostic Workflow **IR** | Kubernetes **CRDs** (pod-per-agent) |
| Engine portability | Temporal + Argo (LangGraph planned) | ADK / K8s lock-in |
| Deployment | Compose **and** K8s/Helm | K8s-only |
| Agent integration | Adapter-first, no SDK (any gRPC service) | Kubernetes-native agents |
| GitOps | Workflow YAML in git, `zynax apply` | kubectl-imperative |
| Web UI | None (M8) | Included |
| CNCF status | Not yet (M8 target) | Sandbox 2026 |

Stated co-existence story: a Kagent agent can register as a Zynax capability via `AgentService`
gRPC — "complementary, not rivals." **Diligence must stress-test whether this framing survives
contact with a buyer who has already adopted Kagent.**

### 1.7 Roadmap & status (M1–M8)

| Milestone | Status | Version | Scope (delivered/claimed) |
|---|---|---|---|
| M1 | Complete | v0.1.0 | 8 gRPC contracts, AsyncAPI, 140+ BDD scenarios, CI gates |
| M2 | Complete | v0.1.0 | Workflow IR compiler (YAML→protobuf), JSON Schemas, validation |
| M3 | **Partial** | v0.2.0 | Temporal engine (`IRInterpreterWorkflow`, dispatch); task-broker/agent-registry slipped to M5.C |
| M4 | **Partial** | v0.3.0 | api-gateway REST, `zynax` CLI, compose; end-to-end dispatch not wired until M5.C |
| M5 | Complete | v0.4.0 | 5 adapters, task-broker MVP, agent-registry MVP, Python SDK base, end-to-end dispatch, e2e-demo green |
| M6 | Complete | v0.5.0 | K8s readiness: mTLS, SBOM/cosign/SLSA, Postgres repos, Helm, NATS event-bus, memory-service, ArgoEngine, PyPI SDK, gRPC health, Prometheus /metrics |
| M7 | **Active** | v0.6.0 (target) | Usable workflows + observability: data-flow bindings, log/event streaming, OTEL+Uptrace, context propagation, git MCP shim, expert-agent substrate, templates, test rigor, supply-chain, docs |
| M8 | Aspirational | v1.0.0 | CNCF Sandbox submission: ≥2 maintainers from 2+ orgs, external security audit, production reference deployment |

Live status: `state/current-milestone.md`, `state/milestone.yaml`, `ROADMAP.md`,
`CLAUDE.md` §Per-Milestone Scope. As of mid-June 2026, M7 has ~57 closed / ~34 open issues with
several EPICs complete.

### 1.8 Goals, non-goals, assumptions

- **Explicit goals:** engine-agnostic IR + pluggable execution; capability routing via stable
  gRPC; declarative GitOps; K8s production-readiness; usability (data-flow) + observability
  (M7); CNCF Sandbox (M8).
- **Implicit goals (from strategy doc):** distribution > features ("the next unit of leverage
  is distribution, not design"); community traction; *proven* portability; supply-chain
  integrity; self-hosting of its own dev automation.
- **Non-goals (ADR-backed):** not an LLM framework (`ADR-011`); not DAG-based (`ADR-014`); no
  mandatory SDK (`ADR-013`); no single-engine lock-in (`ADR-015`); web UI deferred to M8;
  true multi-tenant isolation deferred post-M8; LangGraph is a capability provider, not a full
  engine adapter.
- **Key assumptions to test:** Temporal-as-default deters evaluators (no zero-Temporal mode);
  state-machines beat DAGs; multi-engine portability is actually valued by buyers; gRPC-only
  inter-service is acceptable; YAML-only authoring is acceptable at launch.

### 1.9 Governance & open-source posture

- License: Apache-2.0 (`LICENSE`, `ADR-005`, SPDX headers enforced).
- `GOVERNANCE.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md` all present.
- DCO sign-off enforced; SSH commit signing required; branch protection w/ required_signatures;
  squash-only merge (`ADR-023`); conventional commits (7 types only).
- CNCF structural alignment strong (license, CoC, ADRs/RFCs, SBOM/cosign/SLSA, OpenSSF
  Scorecard badge). **The gating gap is social, not technical**: single-maintainer bus factor,
  zero named external adopters, no recruited TOC sponsor (per `docs/product/strategy.md` §8).

### 1.10 Contradiction register (doc vs doc vs implementation vs roadmap)

This is the most important diligence artifact in Part 1. The project has a **documented
history** of claims preceding delivery, which an internal "Truth Pass" (M5.A, issue #458) and
the M6 delivery later reconciled. Diligence must verify each row against *current* HEAD, since
several were since fixed.

| # | Claim | Asserted in | Reality at time | Status now |
|---|---|---|---|---|
| C1 | "M3 & M4 Complete" | early README | Partial — no end-to-end dispatch until M5.C | Re-labelled Partial — **verify** |
| C2 | "mTLS between all services" | early SECURITY.md | inter-service used insecure creds | Claimed fixed M6 (`ADR-020`) — **verify enforced, not just configurable** |
| C3 | "SBOM per release" | early SECURITY.md | no SBOM in M3/M4 CI | Claimed fixed M6 (`ADR-025`) — **verify** |
| C4 | "cosign-signed images" | early SECURITY.md | no cosign step | Claimed fixed M6 — **verify signatures exist in GHCR** |
| C5 | "CloudEvents publishing" | early README/M3 | log-stub only | Claimed fixed M6 (NATS) — **verify** |
| C6 | "agent-registry implemented" | early README | 0 LoC M1–M4 | Delivered M5.C, Postgres M6 — **verify** |
| C7 | "stateless workflow-compiler" | CLAUDE.md | unbounded in-memory map M1–M5 | Claimed refactored M6 — **verify** |
| C8 | "cel-go guard evaluator" | M5.B scope | bespoke fail-open evaluator | Claimed replaced M6 — **verify** |

Design tensions (not errors, but diligence-relevant): DAGs vs state machines; Temporal
dependency as evaluation friction; YAML-only authoring; SDK vs no-SDK; whether the Kagent
"complementary" framing holds.

Live doc-vs-tooling drift (a worked drift-test example): the `.claude/commands/` surface was
consolidated to the five verbs above (PR #1400), but `CLAUDE.md` §SPDD and
`docs/patterns/spdd-guide.md` still cite the pre-consolidation command names — a reconciliation
follow-up. Agents must treat the live `.claude/commands/` tree as ground truth and flag the
lagging docs, not the reverse.

### 1.11 Repository "where things live" index (for agent routing)

| Concern | Primary locations |
|---|---|
| Constitution / rules | `AGENTS.md`, nested `*/AGENTS.md`, `CLAUDE.md` |
| Decisions | `docs/adr/` (36 ADRs + `INDEX.md` + `TEMPLATE.md`), `docs/rfcs/`, `docs/decisions/` |
| Product & market | `docs/product/strategy.md`, `docs/product/README.md`, `docs/architecture/2026-04-30-competitive-analysis.md`, `2026-05-28-competitive-positioning.md` |
| Architecture reviews | `docs/architecture/` (6 dated reviews 2026-04-30 → 2026-05-28), `ARCHITECTURE.md`, `docs/architecture/fitness-functions.md` |
| Roadmap & state | `ROADMAP.md`, `state/current-milestone.md`, `state/milestone.yaml`, `state/milestone.schema.json`, `docs/milestones/` |
| Services (Go) | `services/{api-gateway,workflow-compiler,engine-adapter,task-broker,agent-registry,memory-service,event-bus}` |
| Agents (Python) | `agents/sdk/`, `agents/adapters/`, `agents/examples/` |
| Contracts | `protos/zynax/v1/`, `protos/tests/` (BDD), `protos/generated/{go,python}/` |
| Specs | `spec/schemas/`, `spec/asyncapi/`, `spec/workflows/examples/`, `spec/templates/` |
| Infra | `infra/helm/`, `infra/docker/`, `infra/docker-compose/`, `images/images.yaml` |
| CI/CD | `.github/workflows/` (21 files), `Makefile`, `cmd/zynax-ci/` |
| AI workflow | `docs/patterns/spdd-guide.md`, `docs/spdd/` (~30 canvases), `.claude/commands/` (`README.md` + 5 verbs `plan`/`deliver`/`review`/`reconcile`/`learn` + `milestone.md` + `lib/` building blocks + `experts/` personas), `automation/`, `docs/ai-learnings/` |
| Governance | `GOVERNANCE.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `.github/CODEOWNERS`, `.github/ISSUE_TEMPLATE/`, `.github/PULL_REQUEST_TEMPLATE.md` |
| User CLI | `cmd/zynax/`, `docs/quickstart.md`, `docs/developer-guide.md` |

---

# Part 2 — Due-Diligence Methodology

### 2.1 Dimension model

The diligence is organized into **16 dimension groups**, each owned by one or more of the 26
Part-5 agents. Every group decomposes into the sub-dimensions below; an agent must address
every sub-dimension in its scope or mark it `UNKNOWN`.

| # | Dimension group | Sub-dimensions (must all be addressed or marked UNKNOWN) | Owning agent(s) |
|---|---|---|---|
| D0 | **Executive Summary** | Investment recommendation, confidence, risk profile, maturity, readiness, overall score | Orchestrator (Part 4) |
| D1 | **Product** | Vision, market fit, differentiation, use cases, personas, user journey, value prop, pricing potential, adoption barriers, community growth, completeness, DX, install experience, docs quality, enterprise readiness, roadmap realism, feature prioritization | 5.3, 5.20 |
| D2 | **Market** | Comparables, OSS competitors, commercial competitors, emerging tech, timing, SWOT, moat, defensibility, network effects, platform strategy, expansion, TAM framing | 5.4, 5.19 |
| D3 | **Architecture** | Quality, layer separation, dependency graph, complexity, modularity, maintainability, evolvability, tech debt, scalability, extensibility, interfaces, contracts, API design, state mgmt, failure isolation, performance arch, resource efficiency, concurrency, memory, storage, config, versioning, compatibility, migration | 5.1, 5.16 |
| D4 | **Engineering** | Code quality, patterns, anti-patterns, refactoring opportunities, duplication, naming, folder org, boundaries, consistency, complexity metrics, comments, ergonomics, readability | 5.5, 5.14, 5.15 |
| D5 | **Security** | Threat model, supply chain, secrets, SBOM, dependencies, update policy, container security, sandboxing, least privilege, authn, authz, input validation, unsafe code, Go safety, attack surface, dependency risk, reproducible builds, signing, release integrity, CI security | 5.2, 5.22 |
| D6 | **Performance** | Architecture bottlenecks, scalability, CPU, memory, IO, lock contention, parallelism, latency, benchmark quality, perf testing, future bottlenecks | 5.6, 5.16 |
| D7 | **Open Source** | License, governance, contribution model, bus factor, community health, onboarding, docs, issue mgmt, release cadence, transparency, decision-making, CNCF/LF/OpenSSF readiness | 5.8, 5.13, 5.21, 5.22 |
| D8 | **DevOps** | CI/CD, release engineering, automation, testing, reproducibility, build speed, dev workflow, tooling, versioning, quality gates, observability, monitoring | 5.9 |
| D9 | **Testing** | Coverage, unit, integration, E2E, regression, benchmarking, mutation, property, fuzzing, chaos, reliability | 5.7 |
| D10 | **AI Workflow** | Claude workflow, agent architecture, prompt quality, automation, parallel dev, context mgmt, planning, validation, hallucination prevention, repo guidance | 5.12, 5.26 |
| D11 | **Documentation** | Coverage, accuracy, consistency, arch docs, dev docs, user docs, install, tutorials, examples, maintenance burden | 5.10 |
| D12 | **Governance** | Roadmap, milestones, planning quality, execution strategy, decision logs, risk mgmt, ownership, long-term sustainability | 5.13, 5.25 |
| D13 | **Financial** | Est. engineering cost, maintenance cost, infra cost, operational cost, community cost, commercialization potential, monetization options, support burden | 5.17, 5.18 |
| D14 | **Enterprise Readiness** | Compliance, auditing, traceability, multi-platform, LTS, upgrade strategy, operational maturity, supportability | 5.20 |
| D15 | **Acquisition Readiness** | Fortune-500 adoptability, VC investability, hyperscaler acquirability, CNCF incubability, enterprise trust, maintainer sustainability | 5.17, 5.18, 5.23 |
| D16 | **Repository Health & Innovation** | Activity, hygiene, churn, branch/PR discipline, drift, novelty, defensible IP | 5.24, 5.26 |

### 2.2 Universal scoring scale (0–10, binding on all agents)

| Score | Label | Meaning |
|---|---|---|
| 9–10 | Exceptional | Best-in-class; a competitive asset; nothing material to fix. |
| 7–8 | Strong | Above market norm; minor gaps; production-credible. |
| 5–6 | Adequate | Meets baseline; notable gaps; works but not differentiated. |
| 3–4 | Weak | Below norm; material gaps that block adoption or scale. |
| 1–2 | Poor | Fundamentally deficient; high remediation cost. |
| 0 | Absent / Misrepresented | Claimed but not present, or actively misleading. |

A score must be accompanied by **3–7 evidence citations** and a one-line justification. Scores
ending in `*` denote low confidence (see §2.3).

### 2.3 Confidence bands

| Band | When to use |
|---|---|
| **High** | Directly verified in code/CI/config with citations; reproducible. |
| **Medium** | Inferred from strong but indirect evidence (e.g., docs + partial code). |
| **Low** | Based on claims not independently verifiable, or limited access. |

Every agent reports a per-dimension confidence and an overall confidence. The orchestrator
**down-weights low-confidence scores** in aggregation (Part 7).

### 2.4 Evidence taxonomy (strongest → weakest)

1. **E1 — Executed proof:** a command run with its output (test pass, `cosign verify`, build).
2. **E2 — Source code:** `repo-path:line` to the implementing code.
3. **E3 — Configuration / CI:** workflow YAML, Makefile target, Helm values pinning behavior.
4. **E4 — Contract / schema:** proto, JSON Schema, AsyncAPI definition.
5. **E5 — First-party doc:** ADR, AGENTS.md, design doc (records intent, not proof of delivery).
6. **E6 — Marketing / README claim:** lowest weight; must be corroborated by E1–E4 to count as VERIFIED.
7. **E7 — External:** third-party benchmark, CVE database, competitor docs, ecosystem data.

A `VERIFIED` fact requires E1–E4. A claim resting only on E5–E6 is `CLAIMED`, not verified.

### 2.5 Red/green flag convention

Each agent must produce explicit **Red flags** (deal risks, ordered by severity) and **Green
flags** (defensible strengths). The orchestrator promotes the most severe red flags into the
executive summary and risk register (Part 7).

### 2.6 The drift test (mandatory cross-cut)

Because of the history in Part 1 §1.10, **every agent must run the drift test** within its
scope: pick the 3 boldest claims in its area, attempt to verify each against code/CI, and
report `VERIFIED / PARTIAL / CONTRADICTED / UNKNOWN`. Contradictions feed the orchestrator's
contradiction-resolution step (Part 4) and the contradiction register (Appendix D).

---

# Part 3 — Investigation Strategy

### 3.1 Anti-overlap ownership matrix

To prevent two agents double-covering (and double-counting) the same evidence, each repository
zone has a **primary owner** (writes the authoritative finding) and **consumers** (may cite the
primary's finding but must not re-score it).

| Repository zone | Primary owner | Consumers |
|---|---|---|
| `protos/`, `spec/schemas/`, API/contract design | 5.1 Architecture | 5.5, 5.16, 5.7 |
| `services/*/internal/domain` code quality | 5.5 Engineering | 5.14, 5.15 |
| Security controls, `SECURITY.md`, mTLS, secrets | 5.2 Security | 5.22, 5.9, 5.20 |
| Supply chain (SBOM/cosign/SLSA), `images/images.yaml` | 5.2 Security | 5.9, 5.22 |
| `.github/workflows/`, `Makefile`, `cmd/zynax-ci/` | 5.9 DevOps | 5.7, 5.2, 5.24 |
| `protos/tests/`, coverage, fuzz, bench | 5.7 Testing | 5.6, 5.5 |
| Scalability/perf architecture | 5.16 Scalability | 5.6, 5.1 |
| `docs/` accuracy & coverage | 5.10 Documentation | 5.3, 5.11 |
| `docs/product/`, positioning, personas | 5.3 Product | 5.4, 5.19, 5.20 |
| `docs/architecture/competitive*`, Kagent | 5.19 Competitive | 5.4, 5.3 |
| `GOVERNANCE.md`, ADR process, ownership | 5.13 Governance | 5.8, 5.21, 5.25 |
| `.claude/`, `docs/spdd/`, `automation/`, SPDD | 5.12 AI Workflow | 5.26, 5.11 |
| CNCF criteria mapping | 5.21 CNCF | 5.8, 5.22, 5.13 |
| OpenSSF Scorecard / best-practices | 5.22 OpenSSF | 5.2, 5.21 |
| Git history, churn, branch/PR hygiene | 5.24 Repository Health | 5.13, 5.9 |
| Cost models, monetization, valuation | 5.17 Investment | 5.18, 5.15 |
| GTM, business model, expansion | 5.18 Business Strategy | 5.4, 5.17 |

**Rule:** if an agent finds material evidence outside its primary zone, it records it as a
*cross-reference note* for the owning agent rather than scoring it.

### 3.2 Parallelization waves

Agents run in four waves. Within a wave, all agents are independent and parallelizable. Later
waves consume earlier waves' findings (so they should be dispatched after, or given the prior
outputs).

- **Wave A — Ground truth (parallel, no dependencies):**
  5.1 Architecture · 5.2 Security · 5.5 Engineering · 5.7 Testing · 5.9 DevOps ·
  5.10 Documentation · 5.24 Repository Health · 5.12 AI Workflow.
- **Wave B — Derived technical (consume Wave A):**
  5.6 Performance · 5.14 Technical Debt · 5.15 Maintainability · 5.16 Scalability ·
  5.22 OpenSSF · 5.26 Innovation.
- **Wave C — Product/market/governance (parallel; lightly consume A):**
  5.3 Product · 5.4 Market · 5.19 Competitive · 5.13 Governance · 5.8 Open Source ·
  5.20 Enterprise Adoption · 5.21 CNCF · 5.25 Future Roadmap · 5.11 Developer Experience.
- **Wave D — Synthesis (consume everything):**
  5.23 Risk Assessment · 5.17 Investment Analysis · 5.18 Business Strategy, then the
  Orchestrator (Part 4) writes D0 Executive Summary.

### 3.3 Shared-context packet contract

Every agent receives, verbatim and read-only: **Part 1** (repository understanding), **Part 2
§2.2–§2.6** (scoring/confidence/evidence/drift rules), and its **own Part 5 prompt**. Agents
must not assume any context beyond this packet; if they need a fact not in it, they must derive
it from the repository with a citation.

### 3.4 Per-agent handoff format (what each agent returns to the orchestrator)

```yaml
agent: "<agent name>"
wave: "<A|B|C|D>"
dimension_groups: ["D3", "D6", ...]
overall_score: <0-10>
overall_confidence: "<High|Medium|Low>"
sub_scores:
  - dimension: "<sub-dimension>"
    score: <0-10>
    confidence: "<...>"
    justification: "<one line>"
    evidence: ["path:line", "cmd→output", "url"]
drift_test:
  - claim: "<bold claim tested>"
    result: "<VERIFIED|PARTIAL|CONTRADICTED|UNKNOWN>"
    evidence: ["..."]
red_flags:   [{severity: "<Critical|High|Medium|Low>", finding: "...", evidence: ["..."]}]
green_flags: [{strength: "...", evidence: ["..."]}]
open_questions: ["..."]
unknowns: ["<what could not be verified and why>"]
cross_references: [{to_agent: "5.x", note: "...", evidence: ["..."]}]
recommendations: [{priority: "<P0|P1|P2>", action: "...", rationale: "..."}]
```

### 3.5 Dry-run dispatch example (validate the loop before running all 26)

> **Goal:** confirm a single agent + the context packet produces a grounded, scored section.
> Pick the **Security** agent (5.2) because it has the clearest verifiable/falsifiable claims.

1. Open a fresh Claude session (or use the Agent tool with `subagent_type: Explore` for
   read-only, or `general-purpose` if you want it to run verification commands).
2. Paste, in order: **Part 1** (context packet) → **Part 2 §2.2–§2.6** → **Prompt 5.2**.
3. The agent should attempt to **verify C2–C4 from the contradiction register** (mTLS, SBOM,
   cosign) by:
   - `grep -rn "insecure.NewCredentials" services/` (is insecure transport still wired?),
   - inspecting `.github/workflows/release.yml` for a `cosign sign` step,
   - inspecting `images/images.yaml` and CI for SBOM generation,
   - optionally `cosign verify <ghcr-image>` if network access is available.
4. Acceptance: the returned packet has an `overall_score`, per-sub-dimension scores with
   `path:line` evidence, a completed `drift_test` for C2–C4 marked
   VERIFIED/PARTIAL/CONTRADICTED, and explicit `unknowns`. If any score lacks evidence, the
   prompt has failed and must be tightened before scaling to 26 agents.

This dry-run is the cheapest way to catch a framework defect before spending 26 agent-runs.

---

# Part 4 — Master Orchestration Prompt

> Paste this into a controller session. It coordinates the 26 agents, resolves their
> contradictions, aggregates scores, and emits the executive summary + investment verdict.
> It assumes it can dispatch the Part 5 prompts (via the Agent tool, sub-sessions, or by your
> hand) and receive their YAML handoff packets (§3.4).

```text
ROLE
You are the Lead Diligence Partner orchestrating a 26-agent technical and strategic due
diligence of the Zynax repository for a potential tens-of-millions investment or acquisition.
You are accountable for a defensible, evidence-graded final recommendation. You personally
write no code findings; you assign, collect, reconcile, weight, and synthesize.

INPUTS YOU HOLD
- Part 1 (Repository Understanding) — the shared context packet. Distribute verbatim.
- Part 2 (Methodology) — scoring scale (0-10), confidence bands, evidence taxonomy, drift test.
- Part 3 (Investigation Strategy) — ownership matrix, waves, handoff schema.
- Part 5 (26 agent prompts) — one per agent.
- Parts 6-10 — report template, risk scoring, investment framework, exec outline, final TOC.

OPERATING PRINCIPLES
1. Evidence over assertion. Reject any agent finding whose score lacks E1-E4 evidence
   (Part 2 §2.4); send it back once with "INSUFFICIENT EVIDENCE — re-verify or mark UNKNOWN".
2. No double-counting. Enforce the §3.1 ownership matrix: a fact is scored by its primary
   owner only; others cite it.
3. Confidence-weighted aggregation. Down-weight Low-confidence scores (Part 7 §7.4).
4. Drift is a finding, not a footnote. Treat every CONTRADICTED drift-test result as a
   potential deal risk; reconcile it explicitly (see CONTRADICTION RESOLUTION).
5. Separate VERIFIED from CLAIMED everywhere. The thesis ("engine-agnostic portability",
   "production-ready", "mTLS everywhere") must be re-derived from code/CI, not README.

EXECUTION SEQUENCE
Step 1 — Assign. For each of the 26 agents, emit a dispatch record: agent id, wave,
  primary zone(s), the exact Part 5 prompt, and the context packet. Respect Part 3 waves:
  dispatch Wave A first; Wave B after A returns; C in parallel with B; D last.
Step 2 — Collect. Receive each agent's §3.4 YAML handoff. Validate schema completeness and
  evidence sufficiency. Log every UNKNOWN and every CONTRADICTED claim.
Step 3 — Contradiction resolution. Build the contradiction register (Appendix D). For each
  conflict (agent-vs-agent, or agent-vs-Part-1, or claim-vs-code):
    a. State both positions with their evidence.
    b. Apply the evidence hierarchy: E1 > E2 > E3 > E4 > E5 > E6. Executed proof and code beat
       docs and README.
    c. Record a RESOLVED verdict + residual uncertainty. If unresolved, mark OPEN and route to
       Risk Assessment (5.23) as a diligence gap requiring management Q&A.
Step 4 — Score aggregation. Compute per-dimension-group scores (Part 2 §2.1) as the
  confidence-weighted mean of contributing agents' sub-scores. Compute the overall score using
  the Part 7 weights. Never average away a Critical red flag — Critical findings cap the
  overall maturity score regardless of the mean (see Part 8 §8.3 gating rules).
Step 5 — Synthesize. Produce, in order:
    (i)   Executive Summary (D0): recommendation, confidence, risk profile, maturity,
          readiness, overall score (use Part 8).
    (ii)  Top 10 red flags and top 10 green flags across all agents, de-duplicated and ranked.
    (iii) The aggregate risk register (Part 7).
    (iv)  The investment recommendation with deal-structure options (Part 8).
    (v)   The unknowns & assumptions ledger (what diligence could NOT verify; what management
          must answer; what a confirmatory technical session should target).
    (vi)  The final report assembled per Part 10; the slide outline per Part 9.

CONTRADICTION RESOLUTION TEMPLATE (use per conflict)
  CONFLICT: <short title>
  POSITION A (<agent/source>): <claim> — evidence <...> — tier <E?>
  POSITION B (<agent/source>): <claim> — evidence <...> — tier <E?>
  RESOLUTION: <which holds, by evidence hierarchy> — RESIDUAL RISK: <...> — STATUS: RESOLVED|OPEN

QUALITY BARS BEFORE YOU FINALIZE
- Every dimension group D1-D16 has a score + confidence + ≥3 evidence citations.
- Every C1-C8 row from Part 1 §1.10 has a current-HEAD verdict.
- The recommendation states what would change it (the swing factors).
- The unknowns ledger is non-empty and honest (a diligence with zero unknowns is not credible).
- No marketing claim survives in the report as VERIFIED without E1-E4 backing.

OUTPUT
Emit the full report following Part 10's structure, opening with the Part 6 executive report
template, and a one-page Part 9 slide outline. Then emit the machine-readable scorecard:
  { overall_score, overall_confidence, recommendation, risk_profile,
    dimension_scores: {D1..D16}, critical_red_flags: [...], swing_factors: [...],
    unknowns: [...] }
```

---

# Part 5 — 26 Specialized Agent Prompts

> Each prompt is **portable and self-contained**. To run one: open a fresh Claude session (or
> the Agent tool), paste **Part 1** + **Part 2 §2.2–§2.6**, then the prompt below. Each prompt
> follows the 13-part contract: Role · Mission · Context · Repository locations · Investigation
> checklist · Expected evidence · Output format · Scoring rubric · Red flags · Green flags ·
> Questions to answer · Confidence & unknowns · Recommendations. All return the §3.4 YAML packet.

---

## 5.1 — Architecture Agent

```text
ROLE: Distinguished Engineer / Enterprise Architect specializing in distributed control planes.
MISSION: Assess whether Zynax's architecture is sound, modular, evolvable, and defensible — and
  whether the three-layer separation and engine-agnostic IR are real or aspirational.
CONTEXT: You hold Part 1. The thesis is "Kubernetes for AI workflows": Layer 1 YAML intent →
  Layer 2 gRPC/AsyncAPI contracts + Workflow IR → Layer 3 pluggable engines/adapters.
REPOSITORY LOCATIONS:
  - AGENTS.md, services/AGENTS.md, agents/AGENTS.md, protos/AGENTS.md (constitution)
  - docs/adr/ (esp. 001 gRPC, 006 monorepo, 008 no-shared-DB, 010-015 runtime/IR/engines,
    029 data-flow, 031 context-propagation, 034 manifest-id collisions, 035 adapter boundary)
  - protos/zynax/v1/*.proto (the contracts), spec/schemas/*.json (Layer 1 schemas)
  - services/*/internal/{domain,api,infrastructure} (hexagonal layering)
  - services/engine-adapter (Temporal IR interpreter), services/workflow-compiler (YAML→IR)
  - docs/architecture/ (6 dated reviews), ARCHITECTURE.md, docs/architecture/fitness-functions.md
INVESTIGATION CHECKLIST:
  [ ] Verify layer separation is enforced, not just documented: do any services import Layer-1
      YAML types? Is there a CI/layer-boundary check? Does domain/ import proto/gRPC?
  [ ] Inspect the Workflow IR (protos + workflow-compiler): is it genuinely engine-neutral, or
      does it leak Temporal semantics? How are loops/human-in-the-loop modeled (ADR-014)?
  [ ] Map the dependency graph across the 7 services; confirm gRPC-only, no shared DB (ADR-008).
  [ ] Assess extensibility: how hard is adding a 3rd engine (beyond Temporal/Argo)? Is there a
      clean WorkflowEngine interface? Verify ArgoEngine exists and shares the interface.
  [ ] Evaluate API/contract design quality: versioning (zynax.v1), buf-breaking discipline,
      backward-compat rules, idempotent apply (ManifestWorkflowID), error modeling.
  [ ] Failure isolation, state management, migration strategy (Postgres repos, ADR-021/026).
  [ ] DRIFT TEST: verify "engine-agnostic — runs on Temporal OR Argo without rewrite" against
      code (is the same IR truly dispatched to both? is there a CI matrix proving it?).
EXPECTED EVIDENCE: proto definitions, interface declarations (path:line), dependency imports,
  CI layer checks, the engine adapter abstraction, ADR cross-refs.
SCORING RUBRIC (0-10): 9-10 textbook control-plane with proven multi-engine portability and
  enforced layering; 7-8 strong, minor leaks; 5-6 sound design but portability partly
  aspirational; 3-4 layering leaks or single-engine reality; 0-2 monolith-in-disguise.
RED FLAGS: IR leaks engine specifics; only one engine actually wired; domain imports proto;
  shared DB; god-services; no migration path.
GREEN FLAGS: enforced layer boundaries; clean WorkflowEngine interface; two real engines;
  buf-breaking gate; hexagonal domain with zero infra imports.
QUESTIONS TO ANSWER: Is the moat ("portability") architecturally real today? What is the cost
  to add an engine? Where will this architecture break at 10x scale?
CONFIDENCE & UNKNOWNS: state what you verified vs. inferred; list what needs a live demo.
RECOMMENDATIONS: P0/P1/P2 architectural actions with rationale.
RETURN: the §3.4 YAML packet (dimension_groups: D3, contributes to D6/D16).
```

---

## 5.2 — Security Agent

```text
ROLE: Principal Security Engineer / offensive+defensive, supply-chain specialist.
MISSION: Establish the real security posture — authn/authz, transport, secrets, supply chain,
  container hardening, CI security — and separate shipped controls from documented intentions.
CONTEXT: Part 1 §1.10 lists historical security-claim drift (mTLS/SBOM/cosign) reportedly fixed
  in M6. Your job is to verify against current HEAD, not trust SECURITY.md.
REPOSITORY LOCATIONS:
  - SECURITY.md, docs/adr/ADR-020-zero-trust-auth.md, ADR-025-slsa-provenance-attestation.md,
    ADR-024-image-reference-management.md, ADR-018-ai-kb-authorization-model.md
  - services/*/cmd/*/main.go (TLS/cred wiring), services/api-gateway/internal/domain (bearer auth)
  - .github/workflows/ (ci.yml, pr-checks.yml, release.yml, weekly-audit.yml) — gitleaks,
    govulncheck, bandit, pip-audit, dependency-review, Trivy, SBOM (syft), cosign
  - images/images.yaml (digest pinning SoT), infra/docker/Dockerfile.* (non-root, distroless)
  - infra/helm/ (NetworkPolicy, securityContext, cert-manager), .pre-commit-config.yaml
INVESTIGATION CHECKLIST:
  [ ] AUTH: how do services authenticate to each other? grep for insecure.NewCredentials across
      services/ — is mTLS enforced or merely configurable? Is dev insecure and prod mTLS?
  [ ] api-gateway authz: bearer token — constant-time compare? what's unauthenticated (GET)?
  [ ] SUPPLY CHAIN: confirm SBOM generation (syft) + cosign signing in release.yml. If network
      allows, run `cosign verify` / `cosign verify-attestation` against a published image.
  [ ] Verify SLSA provenance (ADR-025) and image digest pinning (images.yaml + check-images gate).
  [ ] SECRETS: gitleaks config + baseline; any secrets/PII in repo? envconfig-only? .env.example?
  [ ] DEPENDENCIES: govulncheck (Go), pip-audit/bandit (Python) wired and blocking? any ignores?
  [ ] CONTAINERS: runAsNonRoot, drop ALL caps, read-only rootfs, seccomp, distroless — verify in
      Dockerfiles + Helm securityContext.
  [ ] ATTACK SURFACE: exposed ports, NetworkPolicy default-deny, input validation at gateway/compiler.
  [ ] DRIFT TEST: verify C2 (mTLS), C3 (SBOM), C4 (cosign) from Part 1 §1.10 against HEAD.
EXPECTED EVIDENCE: path:line of cred wiring, workflow steps, Dockerfile directives, Helm
  securityContext, command outputs (cosign/grep), CVE-gate config.
SCORING RUBRIC: 9-10 enforced mTLS + signed+SBOM'd+provenanced images + blocking CVE gates +
  hardened containers, all verified; 7-8 strong, small gaps; 5-6 controls exist but enforcement
  or coverage partial; 3-4 claims exceed reality; 0-2 insecure-by-default or misrepresented.
RED FLAGS: insecure transport in prod path; unsigned images; SBOM/cosign claimed not present;
  secrets in repo; CVE gates non-blocking; root containers; SECURITY.md overstates posture.
GREEN FLAGS: cosign-verifiable images; SBOM attached; blocking govulncheck/gitleaks; non-root
  distroless; cert-manager mTLS; DCO + signed commits + branch protection.
QUESTIONS: Could a Fortune-500 security review pass this today? What is the residual attack
  surface? Is the supply chain tamper-evident end-to-end?
CONFIDENCE & UNKNOWNS: note anything requiring a running cluster or registry access.
RECOMMENDATIONS: prioritized, mapped to CNCF/OpenSSF expectations.
RETURN: §3.4 YAML packet (dimension_groups: D5; contributes D7/D8/D14). Put any unfixed-vuln
  exploit detail in a SEPARATE private note, never in the shared report (mirror repo policy).
```

---

## 5.3 — Product Agent

```text
ROLE: Principal Product Manager for developer/infrastructure platforms.
MISSION: Judge product vision, market fit, completeness, value proposition, adoption barriers,
  and whether the "usable today" story is real for the hero persona.
CONTEXT: Part 1 §1.4 hero use case: ship a multi-agent code-review/CI workflow as YAML, run on
  Temporal OR Argo, see traces in Uptrace, in <15 min. Distribution > features is the stated bet.
REPOSITORY LOCATIONS:
  - README.md, docs/product/strategy.md, docs/product/README.md, ROADMAP.md
  - docs/quickstart.md, docs/developer-guide.md, docs/faq.md, docs/authoring/, docs/migration-v0.6.md
  - spec/workflows/examples/ (are there real runnable workflows?), spec/templates/
  - agents/examples/ (reference agents), cmd/zynax/ (the user CLI)
INVESTIGATION CHECKLIST:
  [ ] Can the hero journey actually be completed from docs alone? Trace quickstart end-to-end.
  [ ] Inventory runnable example workflows; are they real (compile + dispatch) or illustrative?
  [ ] Value proposition clarity vs. Kagent (Part 1 §1.6) — is differentiation legible to a buyer?
  [ ] Product completeness: what's missing for the hero persona (data-flow bindings, observability)?
  [ ] Adoption barriers: Temporal dependency, YAML-only authoring, Day-0 friction, no web UI.
  [ ] Roadmap realism (M7/M8): is the sequencing credible given delivery history?
  [ ] DRIFT TEST: verify the "<15 min, two engines, traces" promise is achievable at HEAD.
EXPECTED EVIDENCE: quickstart steps, example manifests (path), CLI commands, ADR-029 data-flow
  status, observability wiring (ADR-030).
SCORING RUBRIC: 9-10 compelling, complete-for-beachhead, frictionless; 7-8 strong with known
  gaps; 5-6 promising but incomplete for the hero journey; 3-4 vision>product; 0-2 vaporware.
RED FLAGS: hero journey blocked; examples don't run; differentiation unclear; roadmap ignores
  delivery history; no usable workflow at HEAD.
GREEN FLAGS: runnable end-to-end demo; crisp differentiation; honest roadmap; low time-to-value.
QUESTIONS: Who buys this today and why? What is the single biggest adoption blocker?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS (classified by persona + adoption lever).
RETURN: §3.4 YAML packet (D1; contributes D14).
```

---

## 5.4 — Market Agent

```text
ROLE: Market analyst / category strategist for cloud-native + AI infrastructure.
MISSION: Size and characterize the market, map comparables and competitors, assess timing,
  SWOT, moat, network effects, platform strategy, and expansion opportunities.
CONTEXT: Category = control plane above AI-agent orchestration, across durable engines. Named
  rival: Kagent (CNCF Sandbox). Adjacent: Temporal, Argo, Restate, Dapr, LangGraph, Flyte.
REPOSITORY LOCATIONS:
  - docs/product/strategy.md (§3-§5 market/moat), docs/architecture/2026-04-30-competitive-
    analysis.md, docs/architecture/2026-05-28-competitive-positioning.md, README.md
  - ROADMAP.md (expansion sequencing), docs/product/README.md
INVESTIGATION CHECKLIST:
  [ ] Define the category and TAM/SAM/SOM framing (be explicit about assumptions; this is E7).
  [ ] Comparables matrix: OSS competitors, commercial competitors, emerging tech — positioning.
  [ ] Market timing: is "engine-agnostic AI workflow control plane" early/right/late?
  [ ] SWOT; moat & defensibility (re-derive from §1.5, but pressure-test independently).
  [ ] Network effects / platform strategy: does the adapter/capability model create lock-in or
      flywheel? Where are the expansion vectors (templates, marketplace, managed offering)?
  [ ] DRIFT TEST: is the "complementary to Kagent" story a real co-existence or a hedge?
EXPECTED EVIDENCE: repo competitive docs (E5), external ecosystem data (E7, cite sources),
  feature-comparison grounded in code where claimed.
SCORING RUBRIC: 9-10 large/growing market, clear timing, defensible category; 7-8 attractive
  with competition; 5-6 real but crowded/uncertain timing; 3-4 niche or mistimed; 0-2 no market.
RED FLAGS: dominated by a CNCF-backed rival; commoditizing fast; no defensibility; mistimed.
GREEN FLAGS: genuine category creation; portability tailwind; multi-vector expansion.
QUESTIONS: Why now? Why does a control plane win vs. a dominant single engine? What kills it?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D2; contributes D13/D15).
```

---

## 5.5 — Engineering Agent

```text
ROLE: Staff Software Engineer / code-quality reviewer (Go + Python).
MISSION: Assess code quality, patterns/anti-patterns, consistency, naming, boundaries,
  complexity, comments, ergonomics, and refactoring opportunities across services + agents.
CONTEXT: Go services (hexagonal), Python SDK + adapters. ≥90% domain coverage claimed; lint via
  golangci-lint, ruff, mypy --strict. GOWORK=off required for go commands (ADR-017).
REPOSITORY LOCATIONS:
  - services/*/internal/{domain,api,infrastructure}, services/*/cmd/*/main.go
  - agents/sdk/, agents/adapters/{http,git,ci,llm,langgraph}/, agents/examples/
  - tools/golangci-lint.yml, Makefile (lint targets), docs/engineering/best-practices/
  - docs/patterns/ (go-service-patterns, python-agent-guide, bdd-contract-testing)
INVESTIGATION CHECKLIST:
  [ ] Sample 4-6 domain packages: complexity, cohesion, error handling, naming, comment density.
  [ ] Anti-patterns: business logic in main.go/api layer; god functions; copy-paste duplication;
      premature abstraction; ignored errors; context misuse (esp. Temporal determinism).
  [ ] Consistency across services (do they share conventions? a common config/obs lib?).
  [ ] Python adapter quality: typing, async correctness, Protocol-based runtime (ADR-010).
  [ ] Lint configuration strictness vs. what actually passes; any blanket nolint/ignore.
  [ ] Refactoring opportunities ranked by leverage.
  [ ] DRIFT TEST: verify the ≥90% domain coverage and lint-clean claims (run targets if able).
EXPECTED EVIDENCE: path:line for representative good/bad code, lint config, coverage output.
SCORING RUBRIC: 9-10 exemplary, idiomatic, consistent; 7-8 strong with minor debt; 5-6 mixed;
  3-4 inconsistent/duplicated/under-tested; 0-2 poor.
RED FLAGS: logic leaking across layers; large duplication; ignored errors; lint suppressed.
GREEN FLAGS: clean domain layers; shared libs; strict lint actually enforced; readable code.
QUESTIONS: Would this pass a Staff+ review at a top eng org? Where is the highest-leverage cleanup?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D4; contributes D3/D9, hands debt items to 5.14).
```

---

## 5.6 — Performance Agent

```text
ROLE: Performance / systems engineer.
MISSION: Assess performance architecture, expected scalability, benchmark quality, and likely
  future bottlenecks. (Wave B — consume 5.1 Architecture and 5.7 Testing first.)
CONTEXT: M7 EPIC R adds benchmarks/fuzz; bench baseline in tools/bench-baseline.txt; BENCH_SERVICES
  includes workflow-compiler, engine-adapter.
REPOSITORY LOCATIONS:
  - services/*/internal/domain/*_bench_test.go, tools/bench-baseline.txt, Makefile (bench/fuzz)
  - services/engine-adapter (dispatch hot path), services/workflow-compiler (compile path)
  - services/task-broker (routing), event-bus (NATS throughput), infra/helm/ (HPA, resources)
INVESTIGATION CHECKLIST:
  [ ] Identify hot paths: compile, IR interpretation, capability dispatch, event publish.
  [ ] Benchmark coverage & quality: are baselines meaningful? benchstat in CI? regressions gated?
  [ ] Concurrency model: goroutine usage, lock contention, context cancellation, backpressure.
  [ ] Memory: was the in-memory compiler map bounded (Part 1 §1.10 C7)? any unbounded growth?
  [ ] IO/latency: gRPC overhead, Temporal round-trips, Postgres/NATS calls per workflow step.
  [ ] Scalability config: HPA, PDB, resource requests/limits, statelessness.
  [ ] DRIFT TEST: any published throughput/latency numbers — verifiable?
EXPECTED EVIDENCE: bench files + baseline, profiling if runnable, Helm resource config, code paths.
SCORING RUBRIC: 9-10 measured, gated, scalable; 7-8 good with gaps; 5-6 plausible but unmeasured;
  3-4 likely bottlenecks; 0-2 no perf discipline.
RED FLAGS: no benchmarks on hot paths; unbounded memory; sync blocking; no regression gate.
GREEN FLAGS: bench baselines + benchstat gate; bounded state; horizontal scale design.
QUESTIONS: What breaks first at 10x/100x? Are perf claims measured or assumed?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D6; contributes D3/D16).
```

---

## 5.7 — Testing Agent

```text
ROLE: Test architect / SDET.
MISSION: Assess test coverage and quality across the pyramid: BDD contracts, unit, integration,
  E2E, regression, benchmarks, fuzz, property, chaos — and whether gates are real.
CONTEXT: ADR-016 layered strategy: BDD at gRPC boundaries (protos/tests/, godog+bufconn), unit
  ≥90% on domain/, buf breaking as gate. ADR-017: GOWORK=off for all go tests.
REPOSITORY LOCATIONS:
  - protos/tests/features/*.feature + protos/tests/*/steps_test.go (BDD)
  - services/*/internal/domain/*_test.go (unit), *_test.go //go:build integration (integration)
  - agents/sdk/tests/, agents/examples/*/tests/ (Python pytest)
  - .github/workflows/ci.yml, e2e-smoke.yml; Makefile (test-* targets); tools/bench-baseline.txt
INVESTIGATION CHECKLIST:
  [ ] Confirm every gRPC method has ≥1 BDD scenario (ADR-016) — count features vs. RPCs.
  [ ] Verify the ≥90% domain coverage gate exists AND blocks (not advisory). Run if able.
  [ ] Integration tests: do they use real backings (testcontainers) and run in CI?
  [ ] E2E: does e2e-smoke actually apply a workflow and assert execution?
  [ ] Fuzz/bench presence and whether they gate. Any mutation/property testing? Chaos?
  [ ] Test smells: over-mocking, asserting on internals, flaky-skip patterns, GOWORK misuse.
  [ ] DRIFT TEST: verify "140+ BDD scenarios" and "≥90% coverage" claims at HEAD.
EXPECTED EVIDENCE: feature/test file paths, coverage output, CI gate config, test counts.
SCORING RUBRIC: 9-10 rigorous multi-tier with blocking gates + fuzz; 7-8 strong; 5-6 decent unit
  but thin integration/E2E; 3-4 coverage theater; 0-2 minimal/aspirational.
RED FLAGS: coverage gate advisory not blocking; BDD missing for RPCs; E2E is a no-op; skipped tests.
GREEN FLAGS: BDD-before-code discipline; real testcontainers integration; fuzz in CI; benchstat gate.
QUESTIONS: Do the gates actually block a bad PR? What is untested and risky?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D9; contributes D6/D8).
```

---

## 5.8 — Open Source Agent

```text
ROLE: Open Source Program Office (OSPO) lead.
MISSION: Assess license, governance, contribution model, bus factor, community health,
  onboarding, issue management, release cadence, transparency, and decision-making.
CONTEXT: Apache-2.0; single-maintainer baseline; CNCF Sandbox is the M8 goal but the gating gap
  is social (maintainers/adopters), not technical (Part 1 §1.9).
REPOSITORY LOCATIONS:
  - LICENSE, GOVERNANCE.md, CONTRIBUTING.md, CODE_OF_CONDUCT.md, SECURITY.md
  - .github/CODEOWNERS, .github/ISSUE_TEMPLATE/, .github/PULL_REQUEST_TEMPLATE.md
  - docs/adr/ (decision transparency), docs/rfcs/, git history (contributors), CHANGELOG.md
INVESTIGATION CHECKLIST:
  [ ] License hygiene: Apache-2.0 + SPDX headers (ADR-005), NOTICE, third-party compliance.
  [ ] Governance maturity: is decision-making documented and followable? maintainer roles?
  [ ] BUS FACTOR: how many distinct human committers in git log? single-maintainer risk?
  [ ] Contribution funnel: is onboarding (CONTRIBUTING) approachable or a wall (Day-0 friction)?
  [ ] Issue/PR management: triage labels, templates, responsiveness, release cadence + tags.
  [ ] Transparency: ADRs/RFCs public, roadmap public, honest status (Truth Pass culture).
  [ ] DRIFT TEST: any "community" claims vs. actual stars/forks/contributors.
EXPECTED EVIDENCE: governance docs, CODEOWNERS, `git shortlog -sne` style contributor counts,
  release tags, issue/PR metadata.
SCORING RUBRIC: 9-10 healthy multi-org community + mature governance; 7-8 solid solo/small but
  well-run; 5-6 good docs, thin community; 3-4 single point of failure; 0-2 unsustainable.
RED FLAGS: single maintainer / bus factor 1; no external contributors; stale issues; license gaps.
GREEN FLAGS: documented governance; honest transparency culture; clean license; good templates.
QUESTIONS: Is this sustainable without the founder? What's the path to a 2nd maintainer?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D7; contributes D12/D15, feeds 5.21 CNCF).
```

---

## 5.9 — DevOps Agent

```text
ROLE: Staff DevOps / Release Engineer.
MISSION: Assess CI/CD, release engineering, automation, reproducibility, build speed, quality
  gates, tooling, versioning, and operational observability of the delivery pipeline itself.
CONTEXT: 21 workflows; build-once-promote-by-retag (ADR-027); image digest SoT (ADR-024);
  shift-left security; weekly audit; CI logic moving to a tested Go CLI (ADR-036, cmd/zynax-ci).
REPOSITORY LOCATIONS:
  - .github/workflows/ (all 21: ci.yml, pr-checks.yml, pr-size.yml, release.yml, proto-generate.yml,
    weekly-audit.yml, e2e-smoke.yml, sdk-publish.yml, cli-release.yml, tools-*.yml, helm-lint.yml)
  - Makefile (41 targets), cmd/zynax-ci/, images/images.yaml, infra/docker/, renovate.json
INVESTIGATION CHECKLIST:
  [ ] Map required vs. advisory gates. Which actually block merge? (dco, lint, test-unit, security…)
  [ ] Reproducibility: is build-once/promote-by-retag real? digest pinning enforced (check-images)?
  [ ] Release engineering: semver tags, multi-arch builds, SDK→PyPI (trusted publisher), signing.
  [ ] Automation hygiene: post-merge digest bot (loop-safety via skip-ci marker), proto regen.
  [ ] Build speed & caching; Docker-only dev model; GOWORK=off enforcement.
  [ ] Pipeline observability: are failures actionable? flaky management? weekly audit value.
  [ ] DRIFT TEST: verify "21 workflows / build-once-promote" claims; confirm gates block.
EXPECTED EVIDENCE: workflow YAML path:line, Makefile targets, branch-protection implications,
  release.yml retag steps, images.yaml + check-images.
SCORING RUBRIC: 9-10 best-in-class supply-chain-grade pipeline; 7-8 strong; 5-6 functional with
  advisory gaps; 3-4 fragile/manual; 0-2 ad hoc.
RED FLAGS: critical gates advisory; rebuild-on-release (tamper risk); manual release steps; flaky CI.
GREEN FLAGS: blocking gates; promote-by-retag; signed+SBOM'd; digest SoT; CI-as-tested-code.
QUESTIONS: Can this ship safely multiple times a day? What is the most fragile automation?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D8; contributes D5/D9/D16).
```

---

## 5.10 — Documentation Agent

```text
ROLE: Principal technical writer / docs architect.
MISSION: Assess documentation coverage, accuracy (vs. code), consistency, and maintenance
  burden across architecture, developer, user, install, tutorial, and example docs.
CONTEXT: The repo distributes intent across many docs (the diligence had to reconstruct vision).
  Honest "Truth Pass" culture exists; verify docs match HEAD.
REPOSITORY LOCATIONS:
  - README.md, ROADMAP.md, ARCHITECTURE.md, CLAUDE.md, AGENTS.md (+ nested)
  - docs/ (adr, architecture, product, patterns, engineering, contributing, milestones,
    quickstart.md, developer-guide.md, faq.md, authoring/, observability/, infra/, git-workflow.md)
INVESTIGATION CHECKLIST:
  [ ] Coverage map: which subsystems are documented vs. dark? install/quickstart/tutorials/examples.
  [ ] ACCURACY: sample 5-8 doc claims and verify against code/CI (this is the core test).
  [ ] Consistency: do README/ROADMAP/CLAUDE/state agree on milestone status and capabilities?
  [ ] Maintenance burden: dated review docs, duplication, drift risk, single source of truth.
  [ ] Onboarding path: can a new engineer get productive from docs alone?
  [ ] DRIFT TEST: pick the 3 boldest capability claims in README; verify VERIFIED/PARTIAL/CONTRADICTED.
EXPECTED EVIDENCE: doc path:line vs. code path:line pairs; inconsistency citations.
SCORING RUBRIC: 9-10 comprehensive, accurate, consistent; 7-8 strong minor drift; 5-6 good but
  uneven/aging; 3-4 inaccurate/inconsistent; 0-2 misleading.
RED FLAGS: docs assert unshipped features; cross-doc status contradictions; dead quickstart.
GREEN FLAGS: accurate dated reviews; consistent status surfaces; runnable examples; clear ADRs.
QUESTIONS: Could a stranger adopt this from docs alone? Where do docs lie or lag?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D11; contributes D1/D7).
```

---

## 5.11 — Developer Experience Agent

```text
ROLE: DX engineer / DevRel.
MISSION: Assess the contributor and user experience: setup, build, local run, feedback loops,
  tooling ergonomics, error messages, and time-to-first-success.
CONTEXT: Docker-only model (make bootstrap pulls a tools image); 41 Make targets; GOWORK=off
  gotcha; SPDD/canvas overhead; CONTRIBUTING reportedly long (Day-0 friction risk).
REPOSITORY LOCATIONS:
  - Makefile, docs/local-dev.md, docs/github-setup.md, CONTRIBUTING.md, docs/quickstart.md
  - infra/docker-compose/, .pre-commit-config.yaml, cmd/zynax/ (UX of the CLI), docs/faq.md
INVESTIGATION CHECKLIST:
  [ ] Time-to-first-success: from clone to running stack / first workflow — count steps & footguns.
  [ ] Local dev loop: compose up, logs, reset; speed of inner loop; Docker-only friction.
  [ ] Tooling ergonomics: make help quality, error messages, the GOWORK=off trap, pre-commit hooks.
  [ ] Contributor friction: is SPDD/canvas a moat for quality or a wall for casual contributors?
  [ ] CLI UX: command discoverability, help, error handling (cmd/zynax).
  [ ] DRIFT TEST: verify the "<15 min" / one-command-demo claims by tracing the documented steps.
EXPECTED EVIDENCE: Make targets, doc steps, CLI help output, hook config.
SCORING RUBRIC: 9-10 delightful, fast, low-friction; 7-8 good with known traps; 5-6 workable but
  heavy; 3-4 high friction; 0-2 hostile.
RED FLAGS: many manual steps; hidden gotchas (GOWORK); broken quickstart; contributor wall.
GREEN FLAGS: one-command setup; fast loop; great make help; thoughtful CLI.
QUESTIONS: How long to first green workflow? What's the biggest friction for a new contributor?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS (classified by user type + adoption lever).
RETURN: §3.4 YAML packet (D1; contributes D7/D8).
```

---

## 5.12 — AI Workflow Agent

```text
ROLE: Applied-AI / agentic-systems engineer evaluating AI-assisted development at scale.
MISSION: Assess the SPDD pipeline, agent architecture, prompt quality, context management,
  planning, validation, hallucination prevention, and repository guidance — and whether this
  AI-native development model is a genuine asset or a liability.
CONTEXT: SPDD (ADR-019) requires a REASONS Canvas before code; ~30 canvases in docs/spdd/. The
  command surface was consolidated (PR #1400) to five verbs — /plan, /deliver, /review,
  /reconcile, /learn — plus a /milestone lifecycle command; SPDD/delivery/review building blocks
  live under .claude/commands/lib/ and domain personas under .claude/commands/experts/.
  Milestone-agnostic, safe-by-default (PLAN unless --execute); self-hosting ambition via automation/.
REPOSITORY LOCATIONS:
  - docs/patterns/spdd-guide.md, docs/spdd/ (canvases), docs/spdd/CANVAS_TEMPLATE.md
  - .claude/commands/README.md (self-guided command guide), the 5 verbs
    (plan/deliver/review/reconcile/learn) + milestone.md, .claude/commands/lib/ (19 building
    blocks: spdd-*, deliver-*, sequence, plan-infer, *-review, milestone-*),
    .claude/commands/experts/ (8 personas), .claude/settings*, automation/ (AgentDef workflows)
  - docs/adr/ADR-019-spdd-prompt-governance.md, ADR-018-ai-kb-authorization-model.md,
    ADR-028, ADR-033 (expert-agent substrate); docs/ai-learnings/, docs/ai-assistant-setup.md
INVESTIGATION CHECKLIST:
  [ ] How does SPDD actually constrain AI output? Is canvas-before-code enforced (gate) or convention?
  [ ] Prompt/command quality: are the .claude/commands verbs, lib/ blocks, and experts well-engineered?
  [ ] Command-model coherence: does the 5-verb + lib/ + experts/ structure reduce cognitive load
      vs. the prior 20+ command sprawl? Is .claude/commands/README.md a faithful self-guided map?
  [ ] Context management: KB tiering (ADR-018), AI context budget, knowledge-base policy, leak controls.
  [ ] Hallucination/quality prevention: security-review step, drift/sync blocks, learnings loop.
  [ ] Is the AI workflow a defensible IP asset, a productivity multiplier, or process overhead?
  [ ] Risk: does heavy AI authorship correlate with quality issues elsewhere (cross-ref 5.5/5.7)?
  [ ] DRIFT TEST: verify canvases map to shipped features; verify CLAUDE.md §SPDD and
      docs/patterns/spdd-guide.md command names match the live .claude/commands/ surface (known
      lag after PR #1400); is the "self-hosting automation" real?
EXPECTED EVIDENCE: command/canvas paths, the canvas-freshness gate config, KB policy, learnings.
SCORING RUBRIC: 9-10 novel, disciplined, quality-positive AI-native process; 7-8 strong; 5-6
  promising but partly ceremony; 3-4 overhead>value or quality risk; 0-2 chaotic AI sprawl.
RED FLAGS: AI process is ceremony not gate; KB leak risk; quality issues traceable to AI churn.
GREEN FLAGS: enforced canvas governance; KB security tiering; learnings feedback loop; novel IP.
QUESTIONS: Is this a competitive advantage or a maintenance tax? Does it improve or risk quality?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D10; contributes D11/D16, feeds 5.26 Innovation).
```

---

## 5.13 — Governance Agent

```text
ROLE: Engineering governance / program management lead.
MISSION: Assess roadmap, milestones, planning quality, execution strategy, decision logs, risk
  management, ownership, and long-term sustainability of the development process.
CONTEXT: 36 ADRs; milestone state machine (state/milestone.yaml + schema); SPDD; squash-only +
  signed merges (ADR-023). Documented delivery-vs-narrative history then corrected.
REPOSITORY LOCATIONS:
  - GOVERNANCE.md, docs/adr/ (+ INDEX.md, TEMPLATE.md), docs/decisions/, docs/rfcs/
  - ROADMAP.md, state/current-milestone.md, state/milestone.yaml, state/milestone.schema.json,
    docs/milestones/, docs/contributing/engineering-manifesto.md
INVESTIGATION CHECKLIST:
  [ ] Decision discipline: are one-way doors ADR'd? Is the ADR process followed and current?
  [ ] Planning quality: milestone scoping, exit criteria, realism vs. actual delivery cadence.
  [ ] Execution strategy: how is work claimed/tracked/closed? cross-machine safety? truth-pass.
  [ ] Risk management: is there an explicit risk register? how are blockers surfaced?
  [ ] Ownership & sustainability: CODEOWNERS, succession, founder dependency.
  [ ] DRIFT TEST: compare a past milestone's "complete" claim with what shipped (Part 1 §1.10).
EXPECTED EVIDENCE: ADR INDEX completeness, milestone schema/state, manifesto principles, merge rules.
SCORING RUBRIC: 9-10 mature, self-correcting governance; 7-8 strong; 5-6 documented but
  founder-dependent; 3-4 ad hoc/optimistic planning; 0-2 ungoverned.
RED FLAGS: optimistic milestone labeling; ADRs not followed; no risk register; founder-only ownership.
GREEN FLAGS: disciplined ADRs; honest milestone re-labeling; manifesto enforced; truth-pass habit.
QUESTIONS: Does planning predict reality? Is the process robust to the founder leaving?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D12; contributes D7/D15, feeds 5.25).
```

---

## 5.14 — Technical Debt Agent

```text
ROLE: Engineering due-diligence specialist focused on liabilities and remediation cost.
MISSION: Inventory technical debt, quantify remediation effort, and flag debt that threatens
  scale, security, or velocity. (Wave B — consume 5.1, 5.5, 5.7.)
CONTEXT: Known deferrals: rate limiting, multi-tenancy isolation, some observability; historical
  in-memory compiler state and bespoke guard evaluator reportedly remediated in M6.
REPOSITORY LOCATIONS:
  - Whole repo via grep: TODO/FIXME/HACK/XXX, //nolint, # type: ignore, deprecated markers
  - docs/milestones/ (deferral notes), state/current-milestone.md (blockers), ADRs marked Proposed
  - services/*/internal (incomplete impls), agents/adapters (stubbed capabilities)
INVESTIGATION CHECKLIST:
  [ ] Quantify debt markers (counts by type, concentration by package).
  [ ] Architectural debt: deferred isolation/RBAC, single-engine reality gaps, stubbed features.
  [ ] Test/lint debt: suppressed checks, skipped tests, coverage exemptions.
  [ ] Dependency debt: outdated deps (renovate backlog), pinned-around CVEs.
  [ ] Estimate remediation cost (eng-weeks) and classify Now/Next/Later.
  [ ] DRIFT TEST: verify the M6 remediations (C7 compiler state, C8 cel-go) actually landed.
EXPECTED EVIDENCE: grep counts with paths, deferral notes, proposed-ADR backlog, dep ages.
SCORING RUBRIC (debt — invert: high score = LOW debt): 9-10 minimal, well-tracked; 7-8 modest;
  5-6 moderate, mostly known; 3-4 significant/under-tracked; 0-2 crushing.
RED FLAGS: large untracked debt; deferred security/isolation; suppressed gates; stubbed core features.
GREEN FLAGS: debt explicitly tracked + dated; low marker density; remediations actually shipped.
QUESTIONS: What debt blocks scale or a sale? What's the remediation bill?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS (with eng-week estimates).
RETURN: §3.4 YAML packet (D4; contributes D3/D15).
```

---

## 5.15 — Maintainability Agent

```text
ROLE: Long-term-ownership / sustainability reviewer.
MISSION: Assess how cheaply and safely this codebase can be evolved by a NEW team — modularity,
  readability, change-amplification, onboarding cost, and knowledge concentration.
CONTEXT: Hexagonal services, contract-first, heavy AI-assisted authorship; single-maintainer.
REPOSITORY LOCATIONS:
  - services/*/internal (module boundaries), protos/ (contract stability), docs/patterns/
  - AGENTS.md hierarchy (is guidance enough to onboard?), docs/ai-learnings/ (tacit knowledge)
INVESTIGATION CHECKLIST:
  [ ] Change amplification: does a typical change touch 1 module or many? (proto change blast radius)
  [ ] Modularity & coupling: service independence, shared-lib coupling, interface stability.
  [ ] Readability & onboarding: can a new owner navigate via AGENTS.md + docs without the founder?
  [ ] Knowledge concentration / bus factor (cross-ref 5.8): is critical knowledge only in one head?
  [ ] AI-authorship risk: is the code understandable/owned by humans, or only regenerable by AI?
  [ ] DRIFT TEST: pick a recent feature; trace how many files/contracts a change required.
EXPECTED EVIDENCE: dependency/coupling observations with paths, AGENTS.md coverage, change traces.
SCORING RUBRIC: 9-10 easy to own & evolve by a new team; 7-8 good; 5-6 evolvable but
  founder-/AI-dependent; 3-4 high change cost; 0-2 unmaintainable without originators.
RED FLAGS: high coupling; proto changes cascade; knowledge in one head; AI-only-comprehensible code.
GREEN FLAGS: clean boundaries; stable contracts; self-serve onboarding docs; low change amplification.
QUESTIONS: Could an acquirer's team own this in 90 days? What raises the change cost?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D4; contributes D3/D15).
```

---

## 5.16 — Scalability Agent

```text
ROLE: Distributed-systems / SRE scalability architect. (Wave B — consume 5.1, 5.6.)
MISSION: Assess horizontal/vertical scaling, statefulness, data layer, multi-tenancy, and
  failure behavior under load and partition.
CONTEXT: Postgres-backed repos (ADR-021), NATS JetStream event bus (ADR-022), Temporal cluster,
  Helm HPA/PDB; multi-tenancy isolation deferred post-M8; namespace field reportedly cosmetic.
REPOSITORY LOCATIONS:
  - services/* (statelessness), infra/helm/ (HPA, PDB, replicas, resources, NetworkPolicy)
  - docs/adr/ADR-021-horizontal-scale.md, ADR-022-event-bus-architecture.md, ADR-026-postgres-distribution.md
  - services/event-bus (NATS durable consumers), services/memory-service (KV/vector)
INVESTIGATION CHECKLIST:
  [ ] Statelessness: are services horizontally scalable? where does state live? sticky sessions?
  [ ] Data layer scale: Postgres single-instance vs. HA; connection pooling; per-service ownership.
  [ ] Event bus: NATS throughput, durable consumer semantics, replay, backpressure, ordering.
  [ ] Multi-tenancy: is namespace isolation real or cosmetic? noisy-neighbor risk.
  [ ] Failure isolation: partition behavior, retries, idempotency, Temporal durability.
  [ ] DRIFT TEST: verify "production-ready / horizontally scalable" against Helm + code reality.
EXPECTED EVIDENCE: Helm values (replicas/HPA), repo/state code paths, ADR claims vs. implementation.
SCORING RUBRIC: 9-10 proven scale-out + isolation; 7-8 strong with known limits; 5-6 designed but
  unproven / single-instance data; 3-4 scaling gaps; 0-2 won't scale.
RED FLAGS: hidden state; single Postgres SPOF; cosmetic multi-tenancy; no backpressure.
GREEN FLAGS: stateless services + HPA; per-service data ownership; durable event semantics.
QUESTIONS: What is the scaling ceiling today? Where does it fall over first?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D3/D6; contributes D14/D15).
```

---

## 5.17 — Investment Analysis Agent

```text
ROLE: VC/PE investment principal. (Wave D — consume all technical + market findings.)
MISSION: Translate the technical and market picture into an investment thesis: cost-to-build,
  cost-to-maintain, commercialization potential, monetization options, and valuation framing.
CONTEXT: Open-core infra; pre-traction (single maintainer, ~0 external adopters); strong eng
  discipline; M7 in progress; CNCF M8 ambition. Distribution is the stated bottleneck.
REPOSITORY LOCATIONS:
  - docs/product/strategy.md (monetization/sequencing), ROADMAP.md, git history (velocity/cost proxy)
  - Synthesizes outputs of 5.1-5.16 + 5.18/5.19/5.20 (do not re-score their zones; cite them).
INVESTIGATION CHECKLIST:
  [ ] Replacement cost: estimate eng-years embodied (services, contracts, CI, adapters, docs).
  [ ] Maintenance + infra + community cost to sustain.
  [ ] Commercialization paths: managed/cloud, enterprise (RBAC/SSO/audit), support, marketplace.
  [ ] Monetization timing: what must be true (traction signal) before monetizing?
  [ ] Risk-adjusted thesis: what's the bull case, bear case, and the swing factors?
  [ ] Deal structure fit: seed/Series-A invest vs. acqui-hire vs. asset acquisition vs. incubate.
EXPECTED EVIDENCE: cite the contributing agents' scores + evidence; external comps (E7) for
  valuation framing with explicit assumptions.
SCORING RUBRIC: 9-10 compelling risk-adjusted return; 7-8 attractive with conditions; 5-6
  interesting but pre-proof; 3-4 weak/early; 0-2 uninvestable.
RED FLAGS: no traction + crowded market + single maintainer; no clear monetization; high burn-to-proof.
GREEN FLAGS: defensible tech moat; low cost-to-replicate-elsewhere; clear commercialization vectors.
QUESTIONS: What is this worth, under what assumptions? What milestone de-risks the next round?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS (with explicit valuation assumptions).
RETURN: §3.4 YAML packet (D13/D15; primary input to Part 8).
```

---

## 5.18 — Business Strategy Agent

```text
ROLE: Corporate-development / GTM strategist. (Wave D.)
MISSION: Assess business model, go-to-market, expansion, partnerships, platform strategy, and
  the strategic narrative an acquirer or board would underwrite.
CONTEXT: OSS-first; beachhead = AI-forward dev teams (code-review/CI workflows); platform play
  via adapters/capabilities; CNCF as a distribution/credibility channel.
REPOSITORY LOCATIONS:
  - docs/product/strategy.md (§1, §7-§11 GTM/sequencing/monetization), ROADMAP.md, README.md
  - Synthesizes 5.4 Market, 5.19 Competitive, 5.20 Enterprise, 5.3 Product.
INVESTIGATION CHECKLIST:
  [ ] Business model options: open-core, managed service, enterprise add-ons, support — fit & timing.
  [ ] GTM: bottom-up developer adoption vs. top-down enterprise; beachhead → expansion sequence.
  [ ] Distribution strategy: CNCF, templates/marketplace, dogfooding, community flywheel.
  [ ] Partnerships: Temporal/Argo/cloud ecosystems — ally or dependency risk?
  [ ] Strategic narrative coherence: does the story hold for a board/IC?
  [ ] DRIFT TEST: is "distribution > features" backed by any distribution investment in the repo?
EXPECTED EVIDENCE: strategy doc sections, roadmap sequencing, repo signals of GTM/distribution work.
SCORING RUBRIC: 9-10 coherent, executable, multi-vector strategy; 7-8 strong; 5-6 plausible but
  unproven; 3-4 unclear model; 0-2 no viable business.
RED FLAGS: no GTM motion; dependency on a partner that could compete; incoherent narrative.
GREEN FLAGS: clear beachhead→expansion; credible distribution channels; defensible platform play.
QUESTIONS: How does this become a business? What's the wedge and the expansion?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D13/D2; primary input to Part 8).
```

---

## 5.19 — Competitive Analysis Agent

```text
ROLE: Competitive intelligence analyst (cloud-native + agentic AI).
MISSION: Deeply assess the competitive landscape, especially the named rival Kagent, and judge
  whether Zynax's differentiation is defensible and legible.
CONTEXT: Part 1 §1.6 — Kagent (CNCF Sandbox) uses a near-identical tagline; Zynax claims
  engine-agnostic IR + GitOps + no-SDK + compose-AND-k8s as its edge; "complementary" framing.
REPOSITORY LOCATIONS:
  - docs/architecture/2026-04-30-competitive-analysis.md, 2026-05-28-competitive-positioning.md
  - docs/product/strategy.md (§4-§5), README.md (positioning), external sources (E7) for rivals.
INVESTIGATION CHECKLIST:
  [ ] Build a rigorous feature/positioning matrix vs. Kagent, Temporal, Argo, LangGraph, Restate, Dapr.
  [ ] Pressure-test each claimed differentiator: is it real (cite Zynax code) AND not easily copied?
  [ ] Where does each competitor WIN? Be honest (Kagent: CNCF backing, web UI, K8s-native).
  [ ] Is the "complementary to Kagent" story credible to a buyer who already runs Kagent?
  [ ] Time-to-parity: how fast could a funded rival replicate the IR/portability moat?
  [ ] DRIFT TEST: verify each "we uniquely do X" claim against both Zynax code and competitor reality.
EXPECTED EVIDENCE: Zynax code path:line for each differentiator; cited competitor docs (E7).
SCORING RUBRIC: 9-10 clear, defensible, hard-to-copy edge; 7-8 real edge, some exposure; 5-6
  differentiated but contestable; 3-4 me-too; 0-2 outclassed by a backed rival.
RED FLAGS: rival has CNCF + UI + funding and similar value prop; differentiator easily copied;
  "complementary" is wishful.
GREEN FLAGS: genuinely unique portability proven in code; structural (not feature) moat.
QUESTIONS: Why does Zynax win a head-to-head? How long does the moat last?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D2; feeds 5.4/5.17/5.18).
```

---

## 5.20 — Enterprise Adoption Agent

```text
ROLE: Enterprise architect / buyer-side platform evaluator (Fortune-500 lens).
MISSION: Assess whether a large enterprise could adopt and operate Zynax: compliance, auditing,
  traceability, multi-platform support, LTS/upgrade strategy, operational maturity, supportability.
CONTEXT: Enterprise governance persona is post-M8; policy shipped, RBAC/SSO/audit not yet.
REPOSITORY LOCATIONS:
  - infra/helm/ (operability, NetworkPolicy, RBAC, secrets), SECURITY.md, docs/observability/
  - docs/adr/ (policy, mTLS, supply chain), spec/schemas/policy*, services/api-gateway (authz)
  - docs/infra/, docs/migration-v0.6.md (upgrade/migration), CHANGELOG.md
INVESTIGATION CHECKLIST:
  [ ] Compliance & audit: audit logging, traceability, policy enforcement, data residency posture.
  [ ] AuthN/Z for enterprise: RBAC, SSO/OIDC, multi-tenant isolation — present or roadmap?
  [ ] Operability: observability (OTEL/Uptrace), runbooks, upgrade/migration story, LTS posture.
  [ ] Multi-platform: cloud portability, air-gapped/on-prem feasibility, K8s + compose.
  [ ] Supportability: who supports a production incident at 2am? SLAs, docs, escalation.
  [ ] DRIFT TEST: verify enterprise-readiness claims (e.g., "production-ready") against gaps.
EXPECTED EVIDENCE: Helm/operability config, authz code, observability wiring, migration docs.
SCORING RUBRIC: 9-10 enterprise-deployable today; 7-8 close, minor gaps; 5-6 pilot-able not
  prod-enterprise; 3-4 missing core enterprise controls; 0-2 not enterprise-viable.
RED FLAGS: no RBAC/SSO/audit; cosmetic multi-tenancy; no upgrade/LTS story; no support model.
GREEN FLAGS: policy engine; mTLS; observability; clean upgrade path; on-prem + cloud.
QUESTIONS: Would a Fortune-500 platform team approve this for production? What blocks procurement?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D14; feeds D15).
```

---

## 5.21 — CNCF Readiness Agent

```text
ROLE: CNCF Technical Oversight Committee (TOC) reviewer.
MISSION: Map Zynax against CNCF Sandbox (and look-ahead Incubation) criteria and judge
  readiness honestly. (Consume 5.8 Open Source, 5.2 Security, 5.13 Governance.)
CONTEXT: M8 goal is CNCF Sandbox; structural alignment strong; the gating gap is social
  (≥2 maintainers from 2+ orgs, ≥1 named adopter, TOC sponsor). The project's own strategy
  doc advises NOT filing until prerequisites are real.
REPOSITORY LOCATIONS:
  - LICENSE, CODE_OF_CONDUCT.md, GOVERNANCE.md, CONTRIBUTING.md, SECURITY.md
  - docs/adr/ (public decisions), ROADMAP.md, README.md (OpenSSF badge), git history (maintainers)
  - docs/product/strategy.md §8 (the CNCF path)
INVESTIGATION CHECKLIST:
  [ ] Sandbox criteria checklist: license, CoC, public roadmap, governance, security disclosure,
      OWNERS/maintainers, healthy dev cadence, no trademark/IP blockers.
  [ ] Maintainer/adopter prerequisites: count distinct-org maintainers; any named adopters?
  [ ] Trademark/neutrality: is the project name/IP cleanly donatable? any company entanglement?
  [ ] Differentiation from existing CNCF projects (incl. Kagent) — TOC will ask "why not them".
  [ ] Honest verdict: file now / not yet / what's the critical path.
  [ ] DRIFT TEST: verify the OpenSSF/CNCF-alignment claims (badge, controls) at HEAD.
EXPECTED EVIDENCE: governance/security/license files, maintainer counts, badge config, ADR openness.
SCORING RUBRIC: 9-10 Sandbox-ready (likely Incubation path); 7-8 ready bar social prerequisites;
  5-6 structurally close, socially far; 3-4 multiple gaps; 0-2 not a candidate.
RED FLAGS: single-org maintainership; no adopters; trademark/IP entanglement; redundant with a
  CNCF project.
GREEN FLAGS: full structural compliance; honest self-assessment; clear differentiation.
QUESTIONS: Would the TOC accept this, and what's the one blocker? Why this over Kagent?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS (critical-path ordered).
RETURN: §3.4 YAML packet (D7; feeds D15).
```

---

## 5.22 — OpenSSF Readiness Agent

```text
ROLE: OpenSSF / Scorecard + Best-Practices assessor, supply-chain security focus.
MISSION: Score the project against OpenSSF Scorecard checks and the Best Practices Badge, and
  verify supply-chain integrity claims. (Consume 5.2 Security, 5.9 DevOps.)
CONTEXT: README reportedly carries an OpenSSF Scorecard badge; SBOM/cosign/SLSA (ADR-024/025);
  DCO + signed commits + branch protection + pinned digests.
REPOSITORY LOCATIONS:
  - .github/workflows/ (pinned actions? token permissions? CI tests? SAST?), README.md (badge)
  - images/images.yaml (digest pinning), .pre-commit-config.yaml, SECURITY.md (disclosure policy)
  - renovate.json (dependency update automation), branch protection (inferred from ADR-023)
INVESTIGATION CHECKLIST:
  [ ] Walk the Scorecard checks: Branch-Protection, Signed-Releases, Pinned-Dependencies,
      Token-Permissions, Dangerous-Workflow, SAST, Dependency-Update-Tool, Fuzzing, CI-Tests,
      Vulnerabilities, Code-Review, Maintained, Security-Policy, License, CII/Best-Practices.
  [ ] For each: VERIFIED/PARTIAL/FAIL with the config path:line that proves it.
  [ ] Are GitHub Actions pinned by SHA? Are workflow token permissions least-privilege?
  [ ] Signed releases (cosign) + provenance (SLSA) — verify presence and, if possible, validity.
  [ ] DRIFT TEST: verify the displayed Scorecard/badge level matches the actual config.
EXPECTED EVIDENCE: workflow permission blocks, action pin SHAs, signing steps, security policy,
  Scorecard-style per-check results.
SCORING RUBRIC: 9-10 high Scorecard + Gold/Silver badge-ready; 7-8 strong; 5-6 mid Scorecard with
  gaps (unpinned actions, broad tokens); 3-4 weak; 0-2 poor supply-chain hygiene.
RED FLAGS: unpinned actions; broad write tokens; unsigned releases; no SAST/fuzzing; stale deps.
GREEN FLAGS: SHA-pinned actions; least-privilege tokens; signed+SBOM'd releases; fuzzing in CI.
QUESTIONS: What's the realistic Scorecard score, and the top 3 fixes to raise it?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D5/D7; feeds 5.21).
```

---

## 5.23 — Risk Assessment Agent

```text
ROLE: Chief Risk Officer for the diligence. (Wave D — consume ALL agents.)
MISSION: Aggregate every red flag and open question into a single prioritized risk register
  with severity, likelihood, impact, and mitigation, per Part 7.
CONTEXT: You do not discover new facts; you synthesize the 25 other agents + the orchestrator's
  contradiction register into a coherent risk picture.
INPUTS: all §3.4 packets; the contradiction register (Appendix D); Part 1 §1.10.
INVESTIGATION CHECKLIST:
  [ ] De-duplicate and cluster red flags into risk themes (technical, security, market, people,
      execution, legal/IP, financial).
  [ ] For each risk: severity (Part 7 §7.1), likelihood, blast radius, detectability, mitigation,
      owner, and whether it is a deal-breaker, condition-precedent, or monitorable.
  [ ] Identify CONCENTRATION risks (e.g., single maintainer touches everything).
  [ ] Identify UNKNOWN-driven risks (things diligence couldn't verify) and required confirmations.
  [ ] Produce the aggregate risk profile (Critical/High/Medium/Low counts) for Part 8 gating.
EXPECTED EVIDENCE: cite the originating agent + their evidence for each risk (no new claims).
SCORING RUBRIC (risk — invert): 9-10 low, well-mitigated risk; 7-8 moderate, manageable; 5-6
  elevated with mitigations; 3-4 high; 0-2 severe/uninsurable.
RED FLAGS (meta): any Critical risk without a mitigation; clusters of correlated risks; reliance
  on unverified claims.
GREEN FLAGS: risks are known, bounded, and mitigated; honest unknowns ledger.
QUESTIONS: What single risk most threatens the investment? What must be true to proceed?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS (conditions precedent for a deal).
RETURN: §3.4 YAML packet + the full risk register (Part 7 format).
```

---

## 5.24 — Repository Health Agent

```text
ROLE: Repository forensics analyst.
MISSION: Assess objective repo-health signals: commit activity, contributor distribution,
  branch/PR hygiene, churn, drift artifacts, and automation noise.
CONTEXT: Strict merge discipline (squash-only, signed, ADR-023); auto pipelines (digest bot,
  proto regen, weekly audit) can generate [AUTO] issues/noise.
REPOSITORY LOCATIONS (commands, read-only):
  - `git log --oneline -50`, `git shortlog -sne` (contributors), `git branch -a` / remote branches
  - .github/workflows/weekly-audit.yml, proto-generate.yml; CHANGELOG.md; state/current-milestone.md
  - issue/PR metadata (if gh available): open vs closed, [AUTO] issue pile, stale branches
INVESTIGATION CHECKLIST:
  [ ] Commit cadence & recency; is the project actively maintained?
  [ ] Contributor distribution (bus factor signal — cross-ref 5.8/5.15).
  [ ] Branch/PR hygiene: stale branches, orphaned PRs, merge discipline adherence.
  [ ] Drift artifacts: digest-drift, smoke, size, security [AUTO] issues — volume and triage state.
  [ ] Repo cleanliness: build artifacts committed (e.g., coverage.out), gitignore hygiene.
  [ ] DRIFT TEST: does claimed velocity/cadence match the git log?
EXPECTED EVIDENCE: command outputs (git log/shortlog/branch), workflow configs, issue counts.
SCORING RUBRIC: 9-10 active, clean, disciplined; 7-8 healthy; 5-6 active but noisy/messy; 3-4
  stagnant or chaotic; 0-2 abandoned/unhygienic.
RED FLAGS: single committer; stale branch sprawl; untriaged [AUTO] pile; artifacts committed.
GREEN FLAGS: steady cadence; clean branches; disciplined signed/squash merges; low drift.
QUESTIONS: Is this a living, well-tended repo? Where's the hidden mess?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D16; feeds 5.8/5.13/5.23).
```

---

## 5.25 — Future Roadmap Agent

```text
ROLE: Technical strategist assessing forward execution.
MISSION: Judge the realism, sequencing, and risk of the M7/M8 roadmap, and the credibility of
  the path to v1.0 and CNCF. (Consume 5.13 Governance, 5.3 Product, 5.24 Health.)
CONTEXT: M7 (usable workflows + observability) active; M8 (CNCF, audit, reference deployment)
  aspirational. Delivery history shows past optimistic labeling later corrected.
REPOSITORY LOCATIONS:
  - ROADMAP.md, state/current-milestone.md, state/milestone.yaml, docs/milestones/
  - docs/spdd/ (canvases as forward-commitments), Proposed-status ADRs (029-036), docs/rfcs/
INVESTIGATION CHECKLIST:
  [ ] Map M7 EPICs to current completion; is the keystone (data-flow bindings, ADR-029) done?
  [ ] Sequencing realism: are dependencies ordered correctly? is observability/usability on track?
  [ ] Velocity vs. ambition: does historical delivery cadence support the M8 timeline?
  [ ] Roadmap honesty: are non-goals explicit? is scope creep controlled?
  [ ] Critical path to v1.0 + CNCF: what are the true blockers (technical vs. social)?
  [ ] DRIFT TEST: compare a prior milestone's plan vs. actual delivery to calibrate optimism.
EXPECTED EVIDENCE: milestone state, EPIC/canvas status, proposed-ADR backlog, git velocity.
SCORING RUBRIC: 9-10 credible, well-sequenced, de-risked roadmap; 7-8 solid; 5-6 ambitious but
  plausible; 3-4 optimistic/under-resourced; 0-2 fantasy.
RED FLAGS: keystone features slipping; M8 timeline ignores single-maintainer reality; scope creep.
GREEN FLAGS: realistic sequencing; explicit non-goals; calibrated-by-history planning.
QUESTIONS: Will v1.0/CNCF happen on the stated timeline? What's the real critical path?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D12; feeds D15).
```

---

## 5.26 — Innovation Agent

```text
ROLE: Distinguished engineer / technical innovation assessor.
MISSION: Identify what is genuinely novel and defensible (technical IP and process IP) vs.
  competent-but-commodity, and judge the innovation's durability.
CONTEXT: Candidate innovations: engine-agnostic Workflow IR; event-driven state-machine model;
  adapter-first no-SDK capability routing; and the SPDD AI-native development methodology.
REPOSITORY LOCATIONS:
  - protos/zynax/v1/ + services/workflow-compiler + engine-adapter (the IR + dispatch)
  - docs/adr/ADR-012, ADR-014, ADR-013, ADR-029 (IR/state-machine/no-SDK/data-flow)
  - docs/patterns/spdd-guide.md, docs/spdd/, .claude/commands/ (5 verbs + lib/ + experts/ + README),
    automation/ (the AI-native process)
INVESTIGATION CHECKLIST:
  [ ] For each candidate innovation: is it truly novel, or a reframing of existing art? Cite prior art.
  [ ] Is the novelty embodied in code (defensible) or only in docs (rhetorical)?
  [ ] Durability: how long until commoditized? what protects it (network effects, complexity, IP)?
  [ ] Process IP: is SPDD a transferable, valuable methodology or bespoke ceremony? (cross-ref 5.12)
  [ ] Rate innovation vs. execution: is the value in the idea or the disciplined build?
  [ ] DRIFT TEST: verify each "novel" claim against both the code and the prior-art landscape.
EXPECTED EVIDENCE: code embodying the innovation (path:line); ADR rationale; external prior art (E7).
SCORING RUBRIC: 9-10 genuinely novel + defensible + embodied; 7-8 meaningful innovation; 5-6
  incremental/competent; 3-4 mostly repackaging; 0-2 no real innovation.
RED FLAGS: "novelty" only in marketing; easily replicated; no embodiment in code.
GREEN FLAGS: novel mechanism proven in code; process IP that compounds; durable complexity moat.
QUESTIONS: What here could not be rebuilt by a competent team in 6 months? Why?
CONFIDENCE & UNKNOWNS · RECOMMENDATIONS.
RETURN: §3.4 YAML packet (D16/D10; feeds 5.17/5.19).
```

---

# Part 6 — Standard Report Template

> The orchestrator uses this for the consolidated report; each agent uses §6.2 for its section.

### 6.1 Consolidated report skeleton

```
ZYNAX DUE-DILIGENCE REPORT — <date>
0. Executive Summary
   - Recommendation (Part 8) · Overall score · Confidence · Risk profile · Maturity · Readiness
   - Top 5 reasons to proceed · Top 5 reasons for caution · Swing factors
1. Repository Understanding (from Part 1; updated with verified vs. claimed)
2. Dimension Findings (D1-D16), each: score · confidence · key evidence · red/green flags
3. Cross-Cutting Themes (portability reality · supply-chain integrity · bus factor · drift)
4. Contradiction Register & Resolutions (Appendix D)
5. Risk Register (Part 7)
6. Investment Recommendation (Part 8) with deal-structure options
7. Conditions Precedent & Confirmatory Diligence (what to verify in a management session)
8. Unknowns & Assumptions Ledger
Appendices: per-agent full findings · evidence index · scorecard JSON
```

### 6.2 Per-agent section template

```
## <Agent name> — Score: <0-10> (<confidence>)
Mission recap: <1 line>
Verdict: <2-3 sentences>
Sub-dimension scores: <table: sub-dim | score | confidence | evidence>
Drift test: <claim → VERIFIED/PARTIAL/CONTRADICTED/UNKNOWN → evidence>
Red flags: <severity-ordered list w/ evidence>
Green flags: <list w/ evidence>
Open questions / unknowns: <list>
Recommendations: <P0/P1/P2 with rationale>
Cross-references: <to other agents>
```

### 6.3 Hard rules for the report

- No claim without evidence (Part 2 §2.4); label VERIFIED vs CLAIMED everywhere.
- Every D1-D16 has a score, confidence, and ≥3 citations.
- The unknowns ledger must be non-empty and honest.
- A Critical red flag must appear in the Executive Summary.

---

# Part 7 — Risk Scoring Framework

### 7.1 Severity definitions

| Severity | Definition | Effect on deal |
|---|---|---|
| **Critical** | Threatens viability, security, or legality; or a core claim is contradicted. | Deal-breaker until resolved; caps maturity score (Part 8 §8.3). |
| **High** | Materially impairs adoption, scale, or value; costly to fix. | Condition precedent or price adjustment. |
| **Medium** | Notable but bounded; manageable post-close. | Monitor + remediation plan. |
| **Low** | Minor; cosmetic or easily fixed. | Note only. |

### 7.2 Likelihood × Impact matrix

```
            IMPACT →     Low        Medium      High        Severe
LIKELIHOOD ↓
  Almost certain         Medium     High        Critical    Critical
  Likely                 Low        Medium      High        Critical
  Possible               Low        Medium      High        High
  Unlikely               Low        Low         Medium      High
  Rare                   Low        Low         Low         Medium
```

### 7.3 Risk register format (one row per risk)

```
ID | Theme | Description | Severity | Likelihood | Impact | Detectability | Mitigation | Owner | Class(deal-breaker/CP/monitor) | Source agent + evidence
```

Themes: Technical · Security/Supply-chain · Market/Competitive · People/Bus-factor ·
Execution/Roadmap · Legal/IP/License · Financial/Commercial · Operational/Enterprise.

### 7.4 Aggregate risk profile & score weighting

- Aggregate profile = counts {Critical, High, Medium, Low}. **Any Critical → overall risk
  profile is at best "Elevated"; ≥1 unmitigated Critical → "High".**
- Confidence weighting for dimension aggregation: High = ×1.0, Medium = ×0.7, Low = ×0.4.
  Dimension group score = Σ(sub-score × confidence-weight) / Σ(confidence-weight).
- **Suggested overall weighting** (orchestrator may adjust with justification):

| Dimension group | Weight |
|---|---|
| D3 Architecture | 12% |
| D5 Security | 12% |
| D9 Testing | 8% |
| D4 Engineering | 8% |
| D8 DevOps | 7% |
| D1 Product | 9% |
| D2 Market | 9% |
| D7 Open Source | 6% |
| D12 Governance | 6% |
| D6 Performance | 4% |
| D11 Documentation | 4% |
| D10 AI Workflow | 3% |
| D14 Enterprise Readiness | 4% |
| D13 Financial | 4% |
| D16 Repo Health & Innovation | 4% |
| **Total** | **100%** |

(D15 Acquisition Readiness and D0 Executive Summary are synthesized outputs, not weighted inputs.)

---

# Part 8 — Investment Recommendation Framework

### 8.1 Score → recommendation mapping

| Weighted overall | Recommendation | Meaning |
|---|---|---|
| 8.0–10.0 | **Strong Proceed** | Invest/acquire with standard terms; high conviction. |
| 6.5–7.9 | **Proceed (Conditional)** | Invest/acquire subject to conditions precedent (Part 7). |
| 5.0–6.4 | **Conditional / Watch** | Not yet; re-evaluate after named milestones de-risk it. |
| 3.0–4.9 | **Pass (Revisit later)** | Decline now; track for a future round. |
| 0.0–2.9 | **Pass** | Decline; fundamental deficiencies. |

### 8.2 Confidence bands on the recommendation

State the recommendation with an explicit confidence (High/Medium/Low) and the **swing
factors** — the small set of facts that, if they flip, change the recommendation. A
recommendation with no stated swing factors is not credible.

### 8.3 Gating rules (override the numeric score)

- **≥1 unmitigated Critical risk** → recommendation cannot exceed **Conditional / Watch**
  regardless of weighted score.
- **A contradicted core thesis claim** (e.g., engine portability not real, or security controls
  misrepresented) → cap at **Pass (Revisit later)** until corrected and re-verified.
- **Bus factor = 1 with no succession plan** → cap acquisition-readiness sub-verdict at
  Conditional; flag as a key-person condition precedent.

### 8.4 Deal-structure options (recommend the best-fit, with rationale)

| Structure | When it fits | Key diligence focus |
|---|---|---|
| **Equity investment (seed/A)** | Strong tech + early market, pre-traction | Team, moat durability, distribution plan, milestone de-risking |
| **Acqui-hire** | Team/IP > product maturity; bus-factor risk | Key-person retention, knowledge transfer, IP assignment |
| **Asset/technology acquisition** | Tech valuable inside an acquirer's platform | Code ownership, license cleanliness, integration cost, maintainability (5.15) |
| **Strategic partnership / OEM** | Complementary to acquirer's stack | Contract stability, roadmap alignment, support model |
| **Incubate / sponsor (CNCF path)** | Ecosystem value > direct revenue | Governance neutrality, community plan, trademark/IP donatability |
| **Pass / monitor** | Below threshold or unresolved Critical | Define the re-evaluation trigger milestones |

### 8.5 Required recommendation statement shape

```
RECOMMENDATION: <one of §8.1> (confidence: <H/M/L>)
BEST-FIT STRUCTURE: <§8.4 option> — because <rationale>
OVERALL SCORE: <x.x>/10  | RISK PROFILE: <Low/Moderate/Elevated/High>
THESIS IN ONE LINE: <why yes/no>
SWING FACTORS (what would change this): <2-4 bullets>
CONDITIONS PRECEDENT: <if Conditional — the must-resolve list>
```

---

# Part 9 — Executive Presentation Outline

> 12–15 slides for an investment committee / board / acquirer exec team.

1. **Title & verdict** — recommendation, overall score, confidence (one line each).
2. **What Zynax is** — the "Kubernetes for AI workflows" thesis in one diagram (Part 1 §1.1).
3. **Why now** — market timing + the portability/lock-in pain (Part 5.4/5.19).
4. **The product today** — hero journey, what's real vs. roadmap (Part 5.3, with the drift test).
5. **Differentiation & moat** — vs. Kagent and the field; defensibility (Part 5.19/5.26).
6. **Architecture** — three layers + engine-agnostic IR; is portability real? (Part 5.1).
7. **Engineering & quality** — code, testing, CI/CD, supply-chain discipline (Part 5.5/5.7/5.9/5.2).
8. **Security & supply chain** — posture, signed/SBOM'd, the resolved drift history (Part 5.2/5.22).
9. **Open source & CNCF path** — governance, bus factor, the social gating gap (Part 5.8/5.21/5.13).
10. **Scorecard** — D1-D16 radar/heatmap with confidence (Part 7).
11. **Risk register** — top Critical/High risks + mitigations (Part 7).
12. **Financials & deal** — cost-to-build, commercialization, valuation framing, structure (Part 8).
13. **Swing factors & conditions precedent** — what would change the call.
14. **Unknowns & confirmatory diligence** — what a management session must answer.
15. **Recommendation & next steps**.

Design notes: lead with the verdict; one chart per slide; every quantitative claim footnoted to
its evidence; clearly mark VERIFIED vs CLAIMED; never hide a Critical risk.

---

# Part 10 — Final Due-Diligence Document Structure

> The assembled report (target 100+ pages when fully executed). Section page-budgets are
> indicative.

```
FRONT MATTER (2-3 pp)
  Cover · Confidentiality notice · Methodology & evidence standard (Part 2) · Scope & limits
  · Glossary (Appendix C)

1. EXECUTIVE SUMMARY (4-6 pp)
   Verdict · Overall + dimension scores · Risk profile · Top reasons pro/con · Swing factors

2. COMPANY / PRODUCT / TECHNOLOGY OVERVIEW (6-8 pp)
   Reconstructed vision (Part 1) · VERIFIED vs CLAIMED capability map · roadmap status

3. PRODUCT & MARKET (12-16 pp)
   3.1 Product (5.3) · 3.2 Market & TAM (5.4) · 3.3 Competitive incl. Kagent (5.19)
   3.4 Enterprise adoption (5.20) · 3.5 Developer experience (5.11)

4. TECHNOLOGY & ARCHITECTURE (20-28 pp)
   4.1 Architecture (5.1) · 4.2 Engineering (5.5) · 4.3 Scalability (5.16)
   4.4 Performance (5.6) · 4.5 Technical debt (5.14) · 4.6 Maintainability (5.15)
   4.7 Innovation & IP (5.26)

5. SECURITY & SUPPLY CHAIN (10-14 pp)
   5.1 Security posture (5.2) · 5.2 OpenSSF/Scorecard (5.22) · 5.3 Threat model & attack surface
   (private annex for any unfixed-vuln detail)

6. QUALITY & DELIVERY (10-14 pp)
   6.1 Testing (5.7) · 6.2 DevOps/CI-CD (5.9) · 6.3 Documentation (5.10)
   6.4 AI-native development / SPDD (5.12)

7. OPEN SOURCE, GOVERNANCE & CNCF (10-12 pp)
   7.1 Open source health (5.8) · 7.2 Governance (5.13) · 7.3 CNCF readiness (5.21)
   7.4 Repository health (5.24) · 7.5 Future roadmap realism (5.25)

8. RISK (8-10 pp)
   Full risk register (Part 7) · contradiction register & resolutions (Appendix D)
   · concentration & key-person risk · unknowns ledger

9. FINANCIAL & INVESTMENT (8-10 pp)
   Cost-to-build/maintain (5.17) · commercialization & GTM (5.18) · valuation framing
   · recommendation & deal structure (Part 8)

10. CONCLUSION & NEXT STEPS (3-4 pp)
    Final recommendation · conditions precedent · confirmatory-diligence checklist · timeline

APPENDICES
  A. Per-agent full findings (the 26 §3.4 packets)
  B. Evidence index (every citation) · C. Machine-readable scorecard JSON
  D. Tooling: the prompts used (this framework) · E. Glossary
```

---

# Appendix A — Master Scoring Rubric (quick reference)

| Score | Architecture | Security | Product | Market | Testing |
|---|---|---|---|---|---|
| 9-10 | Proven multi-engine portability, enforced layers | Enforced mTLS, signed+SBOM'd, hardened, verified | Complete-for-beachhead, frictionless | Large, well-timed, defensible category | Multi-tier + blocking gates + fuzz |
| 7-8 | Strong, minor leaks | Strong, small gaps | Strong w/ known gaps | Attractive w/ competition | Strong, minor gaps |
| 5-6 | Sound but portability partly aspirational | Controls exist, partial enforcement | Promising but incomplete | Real but crowded | Decent unit, thin integration/E2E |
| 3-4 | Layer leaks / single-engine reality | Claims exceed reality | Vision > product | Niche / mistimed | Coverage theater |
| 0-2 | Monolith-in-disguise | Insecure / misrepresented | Vaporware | No market | Minimal |

(Other dimensions follow the generic 0–10 scale in Part 2 §2.2 and each agent's rubric.)

# Appendix B — Evidence Standard (quick reference)

- Cite as `repo-path:line` (E2/E3/E4), `command → output` (E1), or `URL` (E7).
- `VERIFIED` requires E1–E4. `CLAIMED` rests on E5–E6 only and must be labelled.
- Unverifiable → `UNKNOWN — not found`; never assert.
- Aspirational/roadmap → label `CLAIMED` and separate from delivered.
- The drift test (Part 2 §2.6) is mandatory for every agent.

# Appendix C — Glossary

| Term | Meaning |
|---|---|
| Workflow IR | Engine-neutral intermediate representation a YAML workflow compiles to (ADR-012). |
| Layer 1/2/3 | Intent (YAML) / Communication (gRPC+AsyncAPI) / Execution (engines+adapters). |
| SPDD | Structured Prompt-Driven Development — REASONS Canvas before code (ADR-019). |
| REASONS Canvas | The pre-implementation design artifact for `feat:` PRs (docs/spdd/). |
| Adapter / Capability | A gRPC service implementing `AgentService`; how agents plug in without an SDK (ADR-013). |
| Engine adapter | A pluggable `WorkflowEngine` impl (Temporal, Argo) executing the IR (ADR-015). |
| Truth Pass | The internal exercise that reconciled claims with delivery (M5.A, issue #458). |
| Drift | Gap between documented claims and verified implementation. |
| Bus factor | Number of people whose loss would stall the project. |
| CNCF Sandbox | The entry tier for CNCF projects; the M8 ambition. |

# Appendix D — Contradiction Register (template; orchestrator fills at runtime)

| ID | Claim | Source A | Source B / Code reality | Evidence tiers | Resolution | Residual risk | Status |
|---|---|---|---|---|---|---|---|
| C1 | "M3/M4 Complete" | early README (E6) | re-labelled Partial; verify HEAD | E2/E6 | — | — | OPEN until verified |
| C2 | "mTLS everywhere" | SECURITY.md (E5) | services creds wiring | E2/E5 | — | — | OPEN until verified |
| C3 | "SBOM per release" | SECURITY.md (E5) | release.yml steps | E3/E5 | — | — | OPEN until verified |
| C4 | "cosign-signed images" | SECURITY.md (E5) | release.yml + `cosign verify` | E1/E3/E5 | — | — | OPEN until verified |
| C5 | "CloudEvents publishing" | early README (E6) | event-bus + NATS code | E2/E6 | — | — | OPEN until verified |
| C6 | "agent-registry implemented" | early README (E6) | services/agent-registry | E2/E6 | — | — | OPEN until verified |
| C7 | "stateless compiler" | CLAUDE.md (E5) | workflow-compiler code | E2/E5 | — | — | OPEN until verified |
| C8 | "cel-go guards" | M5.B scope (E5) | engine-adapter guard eval | E2/E5 | — | — | OPEN until verified |
| … | (agents append new conflicts here) | | | | | | |

---

> **End of framework v1.0.** This document is the machine; running Part 4 + Part 5 against the
> live repository produces the report described in Part 10. Maintain it by updating Part 1 when
> the repository's reality changes, and by appending new conflicts to Appendix D as agents find
> them.



