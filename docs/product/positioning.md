<!-- SPDX-License-Identifier: Apache-2.0 -->

# Positioning & Messaging Principle — the engine-portability wedge

> **This is the single source of truth for how Zynax positions itself.** Every stage of the
> project — planning, review, execution, issue authoring, implementation, and documentation —
> points here rather than restating the rule. Change the principle *here*, and the pointers follow.
>
> Canonical grounding: [strategy.md](strategy.md) · competitive analysis in
> [docs/architecture/2026-05-28-competitive-positioning.md](../architecture/2026-05-28-competitive-positioning.md).

---

## The principle (one line)

**Lead with the one thing a Kubernetes-locked competitor cannot do — run the *same* agent
workflow on multiple engines without a rewrite — and say it with total consistency, everywhere.**

The enemy is not Kagent. The enemy is **generic positioning** — describing Zynax in the contested
"control plane for AI agents" language that maps it onto the category leader and forces a comparison
on axes where a first-mover, CNCF-backed, UI-equipped project wins today.

---

## The wedge — what we lead with

These are the differentiators that are **real, shipped, and structurally unavailable to a
K8s-locked competitor**. Lead with these, in this priority order:

| # | Wedge | Status | Why a K8s-locked tool can't match it |
|---|-------|--------|--------------------------------------|
| 1 | **Engine portability** — the same declarative workflow runs on **Temporal _or_ Argo** without a rewrite | ✅ Shipped (both legs in the e2e CI matrix) | It is bound to one runtime / one agent model |
| 2 | **Runs on Compose _and_ K8s/Helm** | ✅ Shipped | It requires Kubernetes |
| 3 | **Adapter-first, no SDK required** — any gRPC `AgentService` is a capability | ✅ Shipped | It needs its own agent/SDK model |
| 4 | **GitOps-native** — `zynax apply` is idempotent via a canonical hash | ✅ Shipped | It is kubectl-imperative |

**The one sentence to lead with everywhere:**

> **"Write your agent workflow once — run it on Temporal or Argo without a rewrite."**

That is the sentence the competitor cannot say. If a headline could be lifted onto a competitor's
page unchanged, it is not differentiated — rewrite it.

---

## "Ruthless" = the discipline

"Ruthless" does **not** mean aggressive or attacking. It means *disciplined and unsentimental*:

1. **Lead with the wedge, every time.** Hero copy, talk titles, issue descriptions, blog intros,
   CLI help, error strings — all open on engine portability, not "a control plane for AI agents."
2. **Cut parity features from the headline.** Multi-LLM support, observability, a workflow DSL —
   table stakes a competitor also has. True, useful, *not differentiating* → they live in the body,
   never the lead.
3. **Don't claim the contested category.** Stop leading with "control plane for AI agents"
   (the competitor's anchored turf). Lead as **"the portability layer for agent workflows."**
4. **Consistency over polish.** One perfect README and ten issue/PR/doc strings that drift back to
   generic language defeats the point. The same wedge, every surface, every time.
5. **Drop messaging you're attached to** if it doesn't differentiate — even when it's true and
   sounds good.

---

## The co-existence story (not an attack)

Always pair the wedge with co-existence — it neutralizes an incumbent's advantage instead of
fighting it head-on. A K8s-native agent can register as a Zynax capability via the gRPC
`AgentService` contract, so the message is:

> **"Already standardized on a K8s-native agent tool? Keep it. Zynax orchestrates across your
> engines — including those agents — so you're not locked to one runtime."**

"We orchestrate across engines, including yours" beats "we're better than them": it removes the
rip-and-replace objection while still owning the portability narrative.

---

## Before → after (grounded in our own copy)

- ❌ *"Zynax is a control plane that abstracts the execution layer behind a declarative API."*
  → collides with the category leader; the reader compares on CNCF backing / UI and we lose.
- ✅ *"Zynax runs the same declarative agent workflow on Temporal or Argo without a rewrite — the
  engine-portability layer for agentic automation."*
  → no competitor answer; the reader compares on portability and we win.

---

## Two guardrails (non-negotiable)

1. **Ruthless ≠ dishonest.** Differentiators must stay grounded in what is actually shipped, with
   the [Zynax truth-pass](../architecture/2026-05-22-platform-engineering-review.md) discipline:
   mark **shipped / partial / aspirational**. A web UI and LangGraph-as-a-full-engine are
   aspirational; never imply otherwise. Competitor facts are external — flag them as unverified
   rather than inventing figures. Overclaiming to differentiate backfires and violates our culture.
2. **This is a distribution lever, not an engineering task.** It changes *how we describe* work, not
   *what we build*. The technical risks are closed; adoption is the binding constraint. Classify
   positioning recommendations as **(audience, adoption-lever = positioning)** — not as features.

---

## How this applies at every stage

This principle is wired into each control surface. Each surface carries a one-line check that points
back here:

| Stage | Surface | The check |
|-------|---------|-----------|
| **Constitution** | [AGENTS.md](../../AGENTS.md) §What Is Zynax? | The North-Star differentiator + link here; every agent inherits it. |
| **Plan / intake** | `.github/ISSUE_TEMPLATE/feature_request.md`, `epic.md`; [/plan](../../.claude/commands/plan.md) | "Positioning fit" — does this advance the wedge? Is user-facing copy lead-with-the-wedge? |
| **Canvas** | [docs/spdd/CANVAS_TEMPLATE.md](../spdd/CANVAS_TEMPLATE.md) §A — Approach | For user-facing features: how the change advances the wedge; user-facing copy leads with it. |
| **Execute / PR** | [docs/contributing/pr-templates.md](../contributing/pr-templates.md) | Any new user-facing string (README/CLI help/docs/error copy) leads with the wedge, not the contested category. |
| **Review** | `/lib:product-review`, `/lib:market-fit-review` | Does every user-facing surface lead with the wedge? Flag copy that competes on the contested category. |
| **Documentation** | [README.md](../../README.md), `docs/` | Hero + quickstart lead with portability; co-existence story present. |

**Rule of thumb for any contributor or agent:** before merging anything a user will read — a
README line, CLI help, an error message, a doc heading, an issue title — ask *"could a
Kubernetes-locked competitor put this exact sentence on their page?"* If yes, it is not
differentiated. Rewrite it to lead with the wedge.
