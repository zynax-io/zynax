# Zynax Roadmap Realignment — User-Experience Program (2026-06-18)

> **Status:** Proposal (executed in part — see §16). **Author:** Oscar Gómez Manresa.
> **Scope:** product/roadmap planning only — no implementation code.
> **Principle:** preserve existing work; extend, reorganize, merge duplicates, rebalance.
> Companion to [docs/product/strategy.md](strategy.md) and [ROADMAP.md](../../ROADMAP.md).

This document realigns the whole roadmap around one outcome: **a first-time user clones (or
doesn't clone), runs one command, and immediately understands and experiences Zynax's value** —
then configures their **own** scenario declaratively. It is grounded in the **live 2026-06-18
validation run** (real workflow driven end-to-end on a local model) and in current GitHub state.

---

## 1. Roadmap audit (grounded in live state)

| Milestone | GH | State | Reality |
|-----------|----|-------|---------|
| M1–M6 | — | ✅ Complete (v0.1–v0.5) | Shipped |
| **M7 — Usable Workflows + Observability** | #7 | 🚧 Active, ~complete | 90 closed; only the first-run UX cluster (#1370 + stories) remained open |
| **M-dx — Developer Experience** | #9 | Open (13) | Mixes *user* onboarding (#1359, #1360) with *developer/contributor* work (#1361, #1363, #1366, #205, #173, #148) |
| **M8 — CNCF Sandbox** | #8 | Open (7) | Governance/community/CNCF (#470, #471, #494–496) + infra (#244, #245) |

**Findings:** (a) M7 is effectively a UX-closeout milestone now — its remaining work *is* the
first-run experience. (b) M-dx conflated **user** and **developer** experience — #1359/#1360 are
user-onboarding, misfiled. (c) Governance/community already has epics (#470/#471) — do **not**
duplicate. (d) Several experiences the brief names (SDK, Plugin, Enterprise, Release-Eng,
Docs-Portal) have **no** epic yet.

---

## 2. Milestone realignment (the map)

- **M7 (reframed in place)** — in-place **first-run UX closeout**. Canonical epic **#1370**.
  Absorbs #1359 (zero-Temporal Day-0) and #1360 (one-command demo) from M-dx. *Reframed, not
  renamed — its 90 closed workflow/observability issues stay truthful history.*
- **M-UX — User Experience (#10, NEW)** — the **forward** UX program: no-clone try-it, intelligent
  context-loading at scale, Documentation Portal. Epics **#1389**, **#1390**.
- **M-dx — Developer Experience (#9)** — Contributor + SDK/Adapter-author experience and AI-tooling.
  Epics **#1391** (Contributor), **#1392** (SDK), plus existing #205/#173/#148.
- **M8 — CNCF Sandbox (#8)** — Governance/Community/Maintainer (#470/#471) + CNCF + infra (#244/#245).
- **Unscheduled (future)** — **#1393** Plugin/Extension, **#1394** Enterprise Adoption,
  **#1395** Release-Engineering — sequenced once M7/M-UX land.

Nothing is discarded; misfiled work is re-homed; missing experiences get epics; duplicates are
mapped to existing issues.

---

## 3. Epic realignment

| Epic | Milestone | Change |
|------|-----------|--------|
| **#1370** First-run UX | M7 | **Reframed** from "awesome quickstart" → canonical UX epic (one-command demo + declarative scenario config). New AC/scope/deliverables. |
| #1389 Forward UX | M-UX | **New** — no-clone try-it + intelligent context-loading |
| #1390 Documentation Portal | M-UX | **New** — Diátaxis restructure |
| #1391 Contributor Experience | M-dx | **New** — wraps #1361, #1363, #1366, #1368, #1369 |
| #1392 SDK & Adapter-Author | M-dx | **New** |
| #1393 Plugin/Extension · #1394 Enterprise · #1395 Release-Eng | unscheduled | **New** placeholders |
| #470/#471 Governance/Community | M8 | **Kept** — mapped, not duplicated |
| #205/#173/#148 SPDD/TechExcellence/AI-KB | M-dx | **Kept** — AI methodology stays here |

---

## 4. Story creation (under #1370, M7)

| # | Story | Why |
|---|-------|-----|
| #1385 | Declarative demo-scenario manifest (workflow + AgentDef + context in one file) | User configures their **own** scenario declaratively (the brief's core ask) |
| #1386 | Qwen2.5-Coder 3B default reference model (configurable) | Standard demo/validation model |
| #1387 | Declarative context-injection for demo scenarios | Ground the scenario in real content, no code |
| #1388 | Human-validation guide standard + template | Every user-visible story is testable by a stranger |

Existing correctness stories (#1371–#1381) and moved-in #1359/#1360 are retained — see #1370.

---

## 5. Dependency graph (implementation order)

```
Correctness gate (any first run must succeed)
  #1371 advertise-addr ─┐
  #1372 snake→event     ├─► #1374 local-model overlay ─► #1386 Qwen default
  #1375 graceful boot   │        │
  #1373 logs stream ────┘        ▼
  #1381 fast-fail        #1376 runnable example ─► #1360 make demo ─► #1359 Day-0 engine
  #1380 healthz                 │                        │
  #1378 surface output ─────────┘                        ▼
  #1377 event-injection                      #1385 declarative scenario ─► #1387 context-injection
                                                          │
                                             #1379 quickstart docs ─► #1388 validation guide
M-UX (after M7): #1389 no-clone + context-loading ─► #1390 Docs Portal
```
**Critical path:** correctness → local model → runnable example → `make demo` → declarative scenario.

---

## 6. Canvas updates

- **#1370 canvas** (`docs/spdd/1370-awesome-quickstart/canvas.md`) — updated to the expanded
  first-run-UX scope (declarative scenario config, Qwen default, human validation). Treated as an
  SPDD **prompt-update** (requirements changed → Canvas first).
- New feat: stories that cross a gRPC/spec boundary (#1385, #1387) require their own REASONS Canvas
  before implementation (ADR-019); generate via `/spdd-reasons-canvas` when scheduled.

---

## 7. RFC updates

- **New RFC — "One-Command UX & Declarative Scenarios"** in [docs/rfcs/](../rfcs/): formalizes the
  `make demo` contract (§14) and the declarative `Scenario` manifest (workflow + AgentDef + context).
- **New RFC — "Context-Loading Architecture"** (§12): document-metadata schema + loading policy.
- Both RFCs are referenced by #1385/#1387/#1389 and gate their canvases.

---

## 8. Documentation restructuring (Diátaxis) — epic #1390

Reorganize `docs/` into the four Diátaxis modes (preserve content; reorganize, don't discard):

| Mode | Houses |
|------|--------|
| **Tutorials** | Quick Start, one-command demo, first scenario |
| **How-to** | Configure a scenario, add an AgentDef, switch models, troubleshoot |
| **Reference** | CLI, manifest schemas, capability/event grammar, ports |
| **Explanation** | Architecture, concepts, ADRs, experts, workflows |

Cross-cutting: Concepts · Workflows · Experts · Integrations · Troubleshooting · Performance ·
Security · Governance · Release Process. Landing page surfaces the **user journey first**.

---

## 9. M7 redesign centered on #1370

M7 answers exactly one question: *how does someone clone (or not), run one command, see value, and
configure their own scenario declaratively — with no repo knowledge?* Everything in M7 serves that.
Contributor/maintainer/governance/enterprise work is explicitly **out** of M7 (→ §10). Full redesign
in epic **#1370**.

---

## 10. Future-experience epics (postponed, sequenced)

| Experience | Epic | Home |
|------------|------|------|
| Contributor | #1391 | M-dx |
| Developer / SDK / Adapter-author | #1392 | M-dx |
| Maintainer · Governance · Community | #470, #471 | M8 (existing — mapped) |
| Documentation Portal | #1390 | M-UX |
| Forward UX (no-clone, context-loading) | #1389 | M-UX |
| Plugin / Extension | #1393 | unscheduled |
| Enterprise Adoption | #1394 | unscheduled |
| Release Engineering | #1395 | unscheduled |

---

## 11. User onboarding redesign

```
clone (or no-clone) → one command → verify prereqs → install/prepare → configure
→ select workflow → load AgentDefs + inject context (declarative) → launch services
→ run demo on local model (Qwen2.5-Coder 3B) → meaningful visible result
→ guided explanation → suggested next actions → offer cleanup
```
No repository knowledge required. Deterministic and repeatable. Owned by #1370 (clone path) and
#1389 (no-clone path).

---

## 12. Context-loading architecture (epic #1389; RFC §7)

Each document/context pack exposes machine-readable front-matter:

```yaml
purpose:        # one line
audience:       # user | developer | operator | maintainer | agent
required_by:    # [issue/epic/agent ids]
dependencies:   # [doc paths]
priority:       # required | optional | lazy
token_estimate: # int
confidence:     # high | medium | low
owner:          # role
related_files:  # [paths]
```
**Loading policy:** agents load `required` automatically; `optional` on demand; `lazy` (large docs)
only when referenced — minimizing context size without reducing correctness. Builds on existing
[docs/context/](../context/) and ADR-028 (context-slice) / ADR-031 (context propagation).

---

## 13. Human-validation standard (epic story #1388)

Every user-visible story ships a guide a stranger can execute: **Purpose · Prerequisites · Expected
duration · Commands · Expected output · Screenshots (when useful) · Troubleshooting · Rollback ·
Feedback questions · Bug-reporting checklist.** Template lives under `docs/contributing/` (or
`docs/authoring/`) and is referenced by each story's acceptance criteria.

---

## 14. One-command UX specification (RFC §7; stories #1360, #1359)

A single entry point — `make demo` (alias `./scripts/demo`, `task demo`) — that:

1. verifies prerequisites · 2. downloads missing assets · 3. prepares the environment ·
4. launches services (or the zero-Temporal Day-0 engine for first contact) · 5. loads the demo
**scenario** (workflow + AgentDefs + injected context, declaratively) · 6. runs on the default local
model (**Qwen2.5-Coder 3B**) · 7. prints progress · 8. shows a **meaningful visible result** ·
9. prints guided explanation + suggested next actions + URLs/credentials · 10. offers cleanup.

**Deterministic and repeatable.** Default model is configurable; Qwen2.5-Coder 3B is the reference
unless a documented technical reason dictates otherwise.

---

## 15. AI-first planning metadata

Every planning artifact (epic/story/canvas/RFC) exposes machine-readable fields so agents can plan
and execute autonomously:

```yaml
objective: …        inputs: […]         outputs: […]
acceptance: […]     required_experts: […]   optional_experts: […]
dependencies: […]   manual_validation: <guide path>
risk: …  confidence: …  future_work: […]  related_files: […]
context_packs: […]  large_context_refs: […]
```
Adopt incrementally: new epics/stories created here carry the linkage; the schema is formalized in
the AI-planning section of the Context-Loading RFC (§7).

---

## 16. Prioritized implementation roadmap

| Phase | Milestone | Work | Gate |
|-------|-----------|------|------|
| **P0 — Make any first run succeed** | M7 | #1371, #1372, #1373, #1375, #1380, #1381 | a fresh `run-local` works end-to-end |
| **P1 — Zero-secret local value** | M7 | #1374, #1386, #1376, #1378 | runnable example completes on Qwen, output visible |
| **P2 — One command + declarative scenario** | M7 | #1360, #1359, #1385, #1387, #1377 | `make demo`; user configures own scenario declaratively |
| **P3 — Docs that match reality + validation** | M7 | #1379, #1388 | quickstart followable verbatim; validation guides shipped |
| **P4 — Forward UX** | M-UX | #1389, #1390 | no-clone try-it; Diátaxis portal |
| **P5 — Other experiences** | M-dx / M8 / future | #1391, #1392, #470, #471, #1393, #1394, #1395 | sequenced after UX lands |

**Execution model:** P0–P3 are SPDD-routed within M7 via `/milestone-orchestrate` (stories are
`status: ready`, canvas Aligned). `feat:` stories crossing a boundary (#1385/#1387) get a REASONS
Canvas first.

---

### What was executed in this pass (GitHub)

- Created milestone **User Experience (M-UX) #10**.
- **Reframed #1370** into the canonical first-run UX epic (new title/scope/AC/deliverables).
- **Moved #1359, #1360** from M-dx → M7, linked under #1370.
- Created stories **#1385–#1388** (M7, under #1370).
- Created epics **#1389–#1395** (M-UX / M-dx / unscheduled) with bidirectional links to wrapped
  issues (#1361/#1363/#1366/#1368/#1369 → #1391; #244/#245 → #1394; #470 confirmed as gov/community).

Full traceability: **Milestone → Epic → Story → Issue → PR → Docs → Validation → Release notes →
Future milestones.** Nothing orphaned.
