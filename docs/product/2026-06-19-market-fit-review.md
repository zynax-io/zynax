<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax — Market-Fit Review (2026-06-19)

**Document type:** Product Strategy · Market Assessment · Longitudinal Positioning
**Date:** 2026-06-19 · **Baseline:** [docs/product/2026-06-18-market-fit-review.md](2026-06-18-market-fit-review.md) (prior dated review, T-1 day)
**Positioning SoT:** [docs/product/positioning.md](positioning.md) · **Competitive baseline:** [docs/architecture/2026-05-28-competitive-positioning.md](../architecture/2026-05-28-competitive-positioning.md)
**Related:** [docs/product/strategy.md](strategy.md) · [ROADMAP.md](../../ROADMAP.md) · [state/current-milestone.md](../../state/current-milestone.md)

---

## Executive Summary — Product-Market Fit Verdict

**Status:** Shipped/MVP · **Verdict (unchanged headline, sharpened constraint):** Promising category, credible architecture, and the **evaluation funnel just got materially cheaper to enter** — but the binding constraint has moved one notch downstream, from *evaluation friction* to *awareness / distribution*.

The one-day delta since [2026-06-18](2026-06-18-market-fit-review.md) is concentrated and real: the **First-run UX cluster (EPIC [#1370](https://github.com/zynax-io/zynax/issues/1370))** landed — 13 sub-issues closed, of which ~11 are functional first-run fixes, merged across ≥9 PRs on 2026-06-18 (`gh pr list --state merged --search merged:2026-06-18` → 32 PRs that day; 9 touch the first-run funnel). A brand-new user can now clone the repo and run a **real LLM code-review workflow locally with zero secrets and zero cost** — a bundled Ollama service ([infra/docker-compose/docker-compose.ollama.yml](../../infra/docker-compose/docker-compose.ollama.yml)) serving **Qwen2.5-Coder 3B** ([#1386](https://github.com/zynax-io/zynax/issues/1386)) behind the `llm-adapter`, driving a runnable, CLI-completable workflow ([spec/workflows/examples/code-review-ollama.yaml](../../spec/workflows/examples/code-review-ollama.yaml), [#1376](https://github.com/zynax-io/zynax/issues/1376)). Before this cluster, `make run-local` **crashed without secrets** ([#1375](https://github.com/zynax-io/zynax/issues/1375), [#1371](https://github.com/zynax-io/zynax/issues/1371)); now the no-secret path is the default happy path.

**What this does to the competitive position vs Kagent:** it removes the single highest-cost step in the evaluation funnel — the OpenAI key + paid-token wall — and does so on a surface Kagent's K8s-only model makes heavier (Kind + cluster vs `docker compose up`). Zynax's evaluation cost is now *plausibly lower* than the named lead competitor's. That is a top-of-funnel improvement.

**What it does NOT yet change:** the awareness signals are flat (1 star, 0 forks, 0 watchers, 1 maintainer — `gh api repos/:owner/:repo`). Lower evaluation friction converts traffic you already have; it does not generate traffic. **The PMF score holds at 7.0/10** because the technology-vs-adoption split is unchanged in aggregate — but the *evaluation* sub-dimension improves and the *awareness* sub-dimension is now the clear single bottleneck.

**Still missing for the verdict to move (explicitly):**
- **No asciinema hero cast** — confirmed absent (`find . -iname '*.cast'` → empty; no `asciinema`/`make demo` reference in [README.md](../../README.md) or [docs/quickstart.md](../../docs/quickstart.md) at review time). [#1360](https://github.com/zynax-io/zynax/issues/1360) is CLOSED but the README-first cast did **not** ship; the cast is a flagged human follow-up. The strategy doc's own §7.1 target — "the hero demo the literal first thing in the README, with an asciinema cast" — remains unmet.
- **Zero-Temporal engine ([#1359](https://github.com/zynax-io/zynax/issues/1359)) is still OPEN** (`status: ready`), deferred to M-dx. Temporal is therefore **still a production prerequisite**, and the local stack still spins up Temporal. (Correction to the 2026-06-18 review, which stated "#1359 merged; engine in active development" — it is not merged.)

---

## 1. Category Definition

**The control plane for AI agent workflows**, at the intersection of three maturing markets (per [strategy.md](strategy.md) §1 and the prior review §1):

1. **AI agent orchestration** (LangGraph, CrewAI, AutoGen, Kagent) — fast, noisy, mostly Python, LLM-centric.
2. **Durable workflow engines** (Temporal, Restate, Argo Workflows, Dapr) — mature, infra-shaped, not AI-specific.
3. **AI/ML platforms** (Kubeflow, Flyte, SageMaker, Vertex) — heavyweight, model-lifecycle-centric, adjacent.

**Per-period read:** the category is **no more or less crowded than yesterday** — a one-day window does not move a category. But the First-run cluster shifts *where Zynax competes within it*: a zero-secret, zero-cost local LLM run is an **agent-orchestration evaluation experience** that most durable-workflow engines (slice 2) don't even attempt, and that the LLM-library cohort (slice 1) gives you only because they have no infra to stand up. Zynax now offers slice-1's low evaluation cost **with** slice-2's engine portability. That intersection is still genuinely rare.

**Positioning-discipline audit (per [positioning.md](positioning.md) guardrail):** the README hero still leads correctly with the wedge — *"Write your agent workflow once — run it on Temporal or Argo without a rewrite"* ([README.md](../../README.md):7) — and pairs it with the co-existence story. **One drift to flag:** the new zero-cost-Ollama capability is a powerful *evaluation* hook but it competes on **parity turf** (Kagent has built-in ModelConfig; LangGraph is LLM-native). It must live in the body as a **try-it-free** on-ramp, never displace the portability wedge in the headline. Recommendation #2 below treats this as a positioning item, not a feature.

---

## 2. Competitive Landscape (Lead: Zynax vs Kagent)

### 2.1 Direct Comparison — with the First-run delta

Both pitch "control plane for AI agents." The split (from [docs/architecture/2026-05-28-competitive-positioning.md](../architecture/2026-05-28-competitive-positioning.md), carried forward with the 2026-06-19 first-run delta marked **NEW**):

| Dimension | **Zynax (v0.5.0 + first-run cluster)** | **Kagent (CNCF Sandbox 2026)** |
|---|---|---|
| **Core abstraction** | Engine-agnostic Workflow IR (protobuf) | Kubernetes CRDs (Pod-per-agent) |
| **Engine portability** | ✅ Temporal + Argo (CI matrix) | ❌ K8s-only; ADK lock-in |
| **Deployment for evaluation** | ✅ `docker compose` / `make run-local` | ❌ Kind + cluster required |
| **Zero-secret first run** | ✅ **NEW** — no-secret happy path; adapters no longer crash ([#1375](https://github.com/zynax-io/zynax/issues/1375)) | external/unverified |
| **Zero-cost local LLM eval** | ✅ **NEW** — bundled Ollama + Qwen2.5-Coder 3B ([#1374](https://github.com/zynax-io/zynax/issues/1374), [#1386](https://github.com/zynax-io/zynax/issues/1386)) | ❌ built-in ModelConfig but expects a real provider/key |
| **Runnable LLM example from CLI** | ✅ **NEW** — [code-review-ollama.yaml](../../spec/workflows/examples/code-review-ollama.yaml) runs to terminal via `zynax apply` ([#1376](https://github.com/zynax-io/zynax/issues/1376)) | kubectl-applied CRDs |
| **CLI surfaces capability output** | ✅ **NEW** — `zynax get/logs/result` show result payloads ([#1378](https://github.com/zynax-io/zynax/issues/1378)) | UI-native |
| **Agent integration model** | Adapter-first, no SDK (any gRPC `AgentService`) | K8s agents + MCP tools |
| **GitOps native** | ✅ `zynax apply` idempotent via canonical hash | ❌ kubectl-imperative |
| **Web UI / hero cast** | ❌ no Web UI (M8); ❌ **no asciinema cast yet** | ✅ Web UI included |
| **Production engine** | 🟡 Temporal still required ([#1359](https://github.com/zynax-io/zynax/issues/1359) OPEN, deferred M-dx) | K8s-native |
| **CNCF status** | Not yet (M8 target) | ✅ Sandbox 2026 |

**Net competitive read:** the First-run cluster widens Zynax's lead on the **evaluation-cost axis** — the cheapest way for a developer to *feel* the product is now Zynax, not Kagent — while Kagent retains its leads on **CNCF backing, Web UI, and shipped-cast polish**. The contest has not flipped; the **entry ramp has.**

> **Competitor-facts caveat (unchanged):** Kagent's Sandbox status, Web UI, ModelConfig, and provider behavior are external and **not verifiable from this repo**; carried forward from the May positioning doc and to be re-validated against Kagent's public sources before external use.

### 2.2 Full Competitive Matrix

Unchanged from the 2026-06-18 review (no competitor moved in one day). Summary: **one direct competitor (Kagent)**; **one large technical overlap (Temporal)** where the value prop is portability, not replacement; Restate/Dapr/Argo/LangGraph/CrewAI/AutoGen/Flyte/Kubeflow as wrappable runtimes or adjacent. See [2026-06-18-market-fit-review.md §2.2](2026-06-18-market-fit-review.md).

---

## 3. TAM Framing

Three adjacent slices (per prior review §3 and [strategy.md](strategy.md)):

1. **Agent orchestration** (noisy growth) — Zynax competes here with LangGraph/Kagent/CrewAI; **the zero-cost local run is a wedge into this slice's evaluation behaviour** (developers expect `pip install` / `docker compose`, not a paid key, to try a thing).
2. **Workflow engine market** (mature) — Zynax is a *consumer* (runs on Temporal/Argo), not a seller.
3. **Platform-engineering / governance layer** (emerging) — **strongest wedge**; engine portability without SDK lock-in for platform teams.

**Wedge position:** beachhead = agentic software-engineering automation; expand to platform teams wanting one control plane across engines; defend against Kagent (K8s-native alt) and Temporal (direct adoption). **The first-run cluster strengthens the *beachhead* slice specifically** — the runnable example is a *code review*, the exact beachhead use case, now demonstrable in minutes at zero cost.

---

## 4. Personas & Beachhead Validation

Beachhead = **agentic software-engineering automation**. It still holds, and the first-run cluster *tightens the proof* for the hero persona.

| Persona | Fit today | First-run delta | Horizon |
|---|---|---|---|
| **AI-forward dev team** (hero) | **Strong** | ✅ **Improved** — a dev can now run a real LLM code review on a git diff, locally, no key, no spend ([code-review-ollama.yaml](../../spec/workflows/examples/code-review-ollama.yaml)); quickstart reconciled to the real CLI surface ([#1379](https://github.com/zynax-io/zynax/issues/1379)) | Now |
| **Platform engineer** | Strong arch, weak proof (no adopter) | → unchanged; Helm/multi-namespace/Postgres shipped (M6) | M7→M8 |
| **AI-infra team** | Good (adapter-first, multi-language) | → 5 adapters; Python SDK on PyPI (M6) | M7→M8 |
| **Enterprise governance** | Partial (policy + rate-limit shipped; RBAC/SSO not) | → unchanged | Post-M8 |

**Beachhead validation:**
- ✅ **Shipped & now trivially demonstrable:** code-review workflow runs end-to-end from the CLI against a real local model — the strongest concrete beachhead artifact to date.
- ✅ **Dogfooding:** DevAuto Wave 4 (M6 [#881](https://github.com/zynax-io/zynax/issues/881)) — unchanged.
- 🟡 **Still open inside the cluster:** declarative demo-scenario manifest ([#1385](https://github.com/zynax-io/zynax/issues/1385)) and context-injection ([#1387](https://github.com/zynax-io/zynax/issues/1387)) remain OPEN — the one-file "workflow + AgentDef + context" demo is not yet a single artifact.
- 🔴 **Gap (unchanged):** no external customer reference. Still the blocking gate for CNCF + enterprise.

---

## 5. Adoption-Funnel Reality (Live Metrics vs Targets)

### 5.1 Current State (live, 2026-06-19 — `gh api repos/:owner/:repo`)

| Metric | Current | Δ vs 2026-06-18 | Note |
|---|---|---|---|
| GitHub stars | 1 | → | No change. |
| GitHub forks | 0 | → | No external contributors. |
| Watchers | 0 | → | No engagement signal. |
| Contributors (unique) | 3 | → | Maintainer + 2 cloud/test agents. |
| External adopters (named) | 0 | → | CNCF blocker. |
| Maintainers (confirmed) | 1 | → | **Critical blocker.** |
| Latest release | v0.5.0 (2026-06-12) | → | M7 unreleased; first-run cluster post-v0.5.0. |
| Open issues | 43 | — | Active backlog. |

### 5.2 Funnel-Stage Read (the important delta)

The funnel has **four stages**: *Awareness → Evaluation → Activation → Adoption*. The first-run cluster is a pure **Evaluation→Activation** intervention.

| Funnel stage | Status 2026-06-18 | Status 2026-06-19 | Lever that moved it |
|---|---|---|---|
| **Awareness** | 🔴 flat (1★/0 forks) | 🔴 **flat (unchanged)** | None — needs distribution/cast |
| **Evaluation** (can I try it cheaply?) | 🟡 key + spend wall; adapters crashed without secrets | 🟢 **materially improved** — zero-secret, zero-cost local run | First-run UX cluster (#1370) |
| **Activation** (did it work for me?) | 🟡 reference YAML waited on external events | 🟢 **improved** — CLI-completable example runs to terminal; output surfaced | #1376, #1378, #1381 |
| **Adoption** (am I using it for real?) | 🔴 0 adopters | 🔴 **unchanged** — Temporal still a prod prereq ([#1359](https://github.com/zynax-io/zynax/issues/1359) open) | None yet |

**Honest assessment:** yesterday the funnel had two leaks — a high evaluation wall **and** zero inbound. The cluster fixed the wall. **The remaining leak is the top of the funnel: there is almost no traffic to convert.** Cheaper evaluation raises the *conversion rate* of whatever awareness exists; with awareness ≈ 0, the absolute effect on adopters this week is ≈ 0. The leverage point is unambiguously **distribution** (the hero cast + outreach), now that the thing it would point at finally works at zero cost.

---

## 6. Shipped vs Partial vs Aspirational (delta only)

New rows / status changes since 2026-06-18 (full table unchanged otherwise — see [prior review §6](2026-06-18-market-fit-review.md)):

| Capability | Status | Evidence | Note |
|---|---|---|---|
| **Zero-secret first run** | ✅ Shipped | [#1375](https://github.com/zynax-io/zynax/issues/1375), [#1371](https://github.com/zynax-io/zynax/issues/1371) | `make run-local` no longer needs keys |
| **Zero-cost local LLM eval (Ollama + Qwen2.5-Coder 3B)** | ✅ Shipped | [docker-compose.ollama.yml](../../infra/docker-compose/docker-compose.ollama.yml), [#1374](https://github.com/zynax-io/zynax/issues/1374), [#1386](https://github.com/zynax-io/zynax/issues/1386) | Reuses host-pulled models read-only; nothing exposed to LAN |
| **Runnable CLI code-review example** | ✅ Shipped | [code-review-ollama.yaml](../../spec/workflows/examples/code-review-ollama.yaml), [#1376](https://github.com/zynax-io/zynax/issues/1376) | Runs to terminal via `zynax apply` |
| **CLI surfaces capability output** | ✅ Shipped | [#1378](https://github.com/zynax-io/zynax/issues/1378) | `zynax get/logs/result` show payloads |
| **Quickstart reconciled to real CLI** | ✅ Shipped | [#1379](https://github.com/zynax-io/zynax/issues/1379) | Leads with runnable Ollama example |
| **One-command `make demo` + asciinema hero** | 🟡 **Partial** | [#1360](https://github.com/zynax-io/zynax/issues/1360) CLOSED; `make demo` target shipped, but no `.cast` recorded yet | **Cast still missing for the verdict** |
| **Zero-Temporal eval engine** | 📅 Aspirational | [#1359](https://github.com/zynax-io/zynax/issues/1359) OPEN, deferred M-dx | **Temporal still a prod prerequisite** |
| **One-file demo-scenario manifest** | 🟡 Partial | [#1385](https://github.com/zynax-io/zynax/issues/1385), [#1387](https://github.com/zynax-io/zynax/issues/1387) OPEN | Workflow+AgentDef+context not yet one artifact |

---

## 7. Product-Market Fit Verdict & Score

**Verdict:** **Shipped/MVP · Promising category, credible architecture; evaluation friction materially reduced, but awareness/distribution is now the single binding constraint.**

### Grounding (reconciled to the architect review's basis, no invented numbers)

1. **Architecture:** ~8.3/10 (per the companion 2026-06-19 architecture review) — unchanged in one day at the architectural level.
2. **PMF:** held at 7.0/10. The first-run cluster improves the **evaluation** sub-dimension but the aggregate technology-vs-adoption split is unchanged: technology problem solved, adoption problem unsolved, and *awareness* (the input to the now-cheaper evaluation funnel) has not moved.
3. **Live adoption signals (2026-06-19):** 1 star, 0 forks, 0 watchers, 1 maintainer; CNCF prerequisites unmet.

### Score (delta-marked)

| Dimension | Score | Δ | Rationale |
|---|---|---|---|
| Architecture quality | 8.3/10 | → | Per companion architecture review. |
| Engineering culture | 8.0/10 | → | ADR/BDD/Truth-Pass culture; first-run cluster shipped cleanly across ≥9 PRs in a day. |
| Category defensibility | 7.0/10 | → | Engine-agnostic + adapter-first unique; evaluation-cost lead widened but parity turf (LLM) not a moat. |
| Beachhead proof | **7.5/10** | **+0.5** | Code-review beachhead now runnable end-to-end at zero cost — strongest concrete artifact yet. |
| **Evaluation friction** (sub-dim) | **7.5/10** | **+1.5** | Zero-secret, zero-cost local run; was the highest-cost funnel step. |
| Product-market fit | 7.0/10 | → | Shipped, end-to-end, now cheap to try. Blocked on **awareness**, not evaluation. |
| CNCF readiness | 5.0/10 | → | Code/governance ready; community prerequisites unmet. |

**Combined PMF score: 7.0 / 10 · Status: Shipped/MVP.** (The improvement is real but localized to the funnel's middle; the overall gate — awareness/community — is unchanged, so the headline number holds.)

---

## 8. Recommendations (Prioritized by User Type + Adoption Lever)

### Awareness → Evaluation (the gate that is now binding)

1. **Record the asciinema hero cast (the `make demo` target already shipped)** [**audience: evaluators** | **lever: awareness / time-to-insight** | **type: docs**] — the first-run *capability* shipped; the first-run *narrative* (the recorded cast) did not. Record clone → `make demo` → `zynax result` → output; embed above the fold. **P0.** Effort: S. Status: [#1360](https://github.com/zynax-io/zynax/issues/1360) closed with the target but the cast is a flagged human follow-up (`docs/casts/` placeholder). **This is now the single highest-leverage open item.**
2. **Position the zero-cost local run as a *try-it-free* body hook, not a headline** [**audience: evaluators** | **lever: positioning** | **type: docs**] — keep the portability wedge in the hero ([positioning.md](positioning.md) guardrail); add a "Try it in 5 minutes, no API key" sub-section. **P1.** Effort: S.
3. **Establish public cadence + Discussions to create top-of-funnel traffic** [**audience: community** | **lever: visibility** | **type: chore**] — cheaper evaluation only matters with inbound. **P1.** Effort: S.

### Evaluation → Adoption (community + adopter proof)

4. **Recruit a second maintainer from a second org** [**audience: governance** | **lever: credibility + bus-factor** | **type: chore**] — CNCF prerequisite, unchanged. **P0.** Effort: L.
5. **Define + recruit the first named external adopter** [**audience: power users** | **lever: social proof** | **type: product**] — now easier to pitch: "run our beachhead use case locally for free in 5 minutes." **P0.** Effort: M.
6. **Land the zero-Temporal eval engine ([#1359](https://github.com/zynax-io/zynax/issues/1359))** [**audience: evaluators** | **lever: friction (deployment weight)** | **type: feat**] — the last heavy dependency in the local run; deferred to M-dx but it is the natural sequel to the Ollama work. **P1.** Effort: M.

### Adoption → Expansion (M8+)

7. **Complete the one-file demo-scenario manifest ([#1385](https://github.com/zynax-io/zynax/issues/1385)/[#1387](https://github.com/zynax-io/zynax/issues/1387))** [**audience: dev teams** | **lever: discoverability** | **type: feat**] — collapse workflow+AgentDef+context into one artifact for the demo. **P1.** Effort: M.
8. **Grow the example/template library around the SW-engineering beachhead (3→10)** [**audience: dev teams** | **lever: discoverability** | **type: docs**] — now seeded by the runnable Ollama example. **P1.** Effort: M.
9. **Decide monetization only after a traction signal** [**audience: strategy** | **lever: optionality** | **type: chore**] — unchanged; keep core Apache-2.0. **P1.** Effort: S.

### Classified for `/plan` handoff

| # | Recommendation | Type | Audience | Adoption Lever | Priority | Size |
|---|---|---|---|---|---|---|
| 1 | Asciinema hero cast (target shipped) | docs | Evaluators | Awareness / time-to-insight | **P0** | S |
| 2 | Try-it-free as body hook (not headline) | docs | Evaluators | Positioning | P1 | S |
| 3 | Public cadence + Discussions | chore | Community | Visibility | P1 | S |
| 4 | 2nd maintainer from 2nd org | chore | Governance | Credibility / bus-factor | **P0** | L |
| 5 | First named adopter | product | Power users | Social proof | **P0** | M |
| 6 | Zero-Temporal eval engine (#1359) | feat | Evaluators | Deployment friction | P1 | M |
| 7 | One-file demo manifest (#1385/#1387) | feat | Dev teams | Discoverability | P1 | M |
| 8 | Example library 3→10 | docs | Dev teams | Discoverability | P1 | M |
| 9 | Monetization after traction | chore | Strategy | Optionality | P1 | S |

---

## 9. Longitudinal Delta vs 2026-06-18 Review

### Funnel / adoption deltas

| Signal | 2026-06-18 | 2026-06-19 | Delta |
|---|---|---|---|
| Stars / forks / watchers | 1 / 0 / 0 | 1 / 0 / 0 | → (awareness unchanged) |
| Maintainers / external adopters | 1 / 0 | 1 / 0 | → (CNCF blockers unchanged) |
| **Evaluation friction** | 🟡 key + spend wall; adapters crash w/o secrets | 🟢 zero-secret, zero-cost local LLM run | **MATERIALLY IMPROVED** |
| **Beachhead demonstrability** | 3 examples; reference YAML waited on events | code-review runs to terminal via CLI, free, local | **STRONGER** |
| **Hero asciinema cast** | "in-progress" | **still absent** (`.cast` not in repo) | **No change — verdict gap persists** |
| **Zero-Temporal engine (#1359)** | stated "merged" (incorrect) | **OPEN, deferred M-dx** (corrected) | **Correction — Temporal still prod prereq** |

### Verdict delta

| | 2026-06-18 | 2026-06-19 |
|---|---|---|
| PMF score | 7.0/10 | 7.0/10 (held) |
| Binding constraint | "Adoption (community credibility, adopter, 2nd maintainer)" | **Sharpened: awareness/distribution** — evaluation friction is no longer the leak; the now-cheap funnel needs traffic |
| Strongest new lever | UX-closeout in flight | **First-run cluster shipped** — convert-rate up; activate distribution next |

**Summary:** in one day the team closed the most expensive step in the evaluation funnel — a zero-secret, zero-cost local run of the beachhead use case. That widens the evaluation-cost lead over Kagent and strengthens beachhead proof (+0.5). It does **not** move the headline PMF number, because the constraint simply advanced one stage: with the wall gone, **the missing pieces are the hero cast that draws traffic in and the community that converts it.** Technology and now evaluation are solved; **distribution is the work.**

---

## 10. Risk Longitudinal Track (delta)

| Risk | 2026-06-18 | 2026-06-19 | Mitigation |
|---|---|---|---|
| R2: Single maintainer / bus-factor | 🔴 Open | 🔴 **Open** | Recruit 2nd maintainer now |
| R6: Kagent absorbs category | 🟡 Open (complementary) | 🟡 **Open — evaluation-cost lead widened** | Lead with portability wedge; add try-it-free body hook |
| R10: Adoption stalled at awareness | 🔴 Open | 🔴 **Open — now the *single* binding constraint** | Hero cast (#1360 cast follow-up) + cadence + outreach |
| **New R11: Evaluation→Activation friction** | 🟡 Implicit | 🟢 **CLOSED** by first-run cluster (#1370) | — |
| **New R12: Temporal still a prod prerequisite** | (mis-stated closed) | 🟡 **Open** — [#1359](https://github.com/zynax-io/zynax/issues/1359) deferred M-dx | Land zero-Temporal eval engine |

---

## 11. Market-Fit Verdict Summary (One-Liner)

**As of 2026-06-19 Zynax can be evaluated for free, locally, with no secrets — running its own beachhead use case (LLM code review) end-to-end — which widens its evaluation-cost lead over Kagent and strengthens beachhead proof; the PMF score holds at 7.0/10 because the binding constraint has simply advanced from evaluation friction to awareness/distribution: the now-cheap funnel still needs an asciinema hero cast to draw traffic in, a second maintainer and a named adopter to convert it, and the zero-Temporal engine ([#1359](https://github.com/zynax-io/zynax/issues/1359)) to drop the last heavy production prerequisite.**

---

## Appendix — Sources & Methodology

### A. Grounding sources (cited throughout)
- Prior dated review: [docs/product/2026-06-18-market-fit-review.md](2026-06-18-market-fit-review.md)
- Positioning SoT: [docs/product/positioning.md](positioning.md)
- Competitive baseline: [docs/architecture/2026-05-28-competitive-positioning.md](../architecture/2026-05-28-competitive-positioning.md)
- Strategy: [docs/product/strategy.md](strategy.md) · Roadmap: [ROADMAP.md](../../ROADMAP.md) · Milestone: [state/current-milestone.md](../../state/current-milestone.md)
- First-run artifacts: [infra/docker-compose/docker-compose.ollama.yml](../../infra/docker-compose/docker-compose.ollama.yml) · [infra/docker-compose/ollama/llm-adapter.config.yaml](../../infra/docker-compose/ollama/llm-adapter.config.yaml) · [spec/workflows/examples/code-review-ollama.yaml](../../spec/workflows/examples/code-review-ollama.yaml) · [README.md](../../README.md) · [docs/quickstart.md](../../docs/quickstart.md)

### B. Live signals (pulled 2026-06-19)

| Signal | Method | Result |
|---|---|---|
| Stars / forks / watchers | `gh api repos/:owner/:repo` | 1 / 0 / 0 |
| Contributors | `gh api repos/:owner/:repo/contributors --jq length` | 3 (maintainer + agents) |
| Open issues | `gh api repos/:owner/:repo` | 43 |
| Latest release | `gh release list` | v0.5.0 (2026-06-12) |
| EPIC #1370 closed sub-issues | `gh issue list --search "1370 in:body" --state closed` | 13 |
| First-run PRs merged 2026-06-18 | `gh pr list --state merged --search merged:2026-06-18` | 32 total; ≥9 touch the first-run funnel |
| #1359 / #1360 state | `gh issue view` | #1359 OPEN (deferred M-dx); #1360 CLOSED (cast a flagged follow-up) |
| Asciinema cast present? | `find . -iname '*.cast'`; grep README/quickstart | **None (placeholder only)** |

### C. Mark-up conventions
- ✅ Shipped · 🟡 Partial · 📅 Aspirational · 🔴 Blocker · 🟢 Mitigated/Improved.
- Competitor figures (Kagent Sandbox/UI/ModelConfig) are external/unverified — re-confirm against primary sources before external publication.
