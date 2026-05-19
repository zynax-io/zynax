# Current Milestone State

> This file tracks the active execution state. Update it when milestones close,
> blockers change, or active work shifts. Do NOT use this file for architecture
> decisions — those belong in `docs/adr/`. Do NOT accumulate history here.

---

## Status Summary

| Milestone | Status | Version |
|-----------|--------|---------|
| M1 — Contracts Foundation | ✅ Complete | v0.1.0 |
| M2 — Workflow IR | ✅ Complete | v0.1.0 |
| M3 — Temporal Execution | ⚠ Partial | v0.2.0 |
| M4 — YAML System + CLI | ⚠ Partial | v0.3.0 |
| **M5 — Adapter Library** | **In Progress** | v0.4.0 |

M3/M4 are partial because task-broker and agent-registry were not delivered in those milestones.
Both are in-progress under M5.C (#460). See [docs/milestones/M5-plan.md](../docs/milestones/M5-plan.md).

---

## M5 — Progress

M5 is structured into five parallel tracks. See full execution plan: **[docs/milestones/M5-plan.md](../docs/milestones/M5-plan.md)**.

### Track Overview

| Track | Epic | Status |
|-------|------|--------|
| M5.A Truth Pass | [#458](https://github.com/zynax-io/zynax/issues/458) | In Progress — 2/3 children done; #474 open |
| M5.B Engine Correctness | [#459](https://github.com/zynax-io/zynax/issues/459) | In Progress — 3/4 children done; #476 open |
| M5.C Capability Dispatch | [#460](https://github.com/zynax-io/zynax/issues/460) | In Progress — task-broker code merged; agent-registry pending |
| M5.D Security Baseline | [#461](https://github.com/zynax-io/zynax/issues/461) | ✅ Complete (closed) |
| M5.E DX Polish | [#462](https://github.com/zynax-io/zynax/issues/462) | ✅ Complete (closed) |
| Adapter Library | [#377](https://github.com/zynax-io/zynax/issues/377) | In Progress — http ✅; git/ci/llm/langgraph BDD done, impl pending |
| Containerized Make | [#442](https://github.com/zynax-io/zynax/issues/442) | ✅ Complete (closed) |

---

## Active Work (M5.C)

### task-broker (#479) — code complete, quality in progress

Implementation PRs #520, #522, #523 merged. Domain coverage: 92.7%.

**Open cleanup issues (M5.C, `track: M5.C`):**

| Issue | Step | Status |
|-------|------|--------|
| [#530](https://github.com/zynax-io/zynax/issues/530) | Update AGENTS.md | ready |
| [#531](https://github.com/zynax-io/zynax/issues/531) | Align service BDD + godog steps | ready |
| [#532](https://github.com/zynax-io/zynax/issues/532) | Handler unit tests | ready |

### agent-registry (#480) — pending implementation

Canvas aligned. Ordered delivery: #526 → #527 → #528 → #481.

| Issue | Step | Status |
|-------|------|--------|
| [#526](https://github.com/zynax-io/zynax/issues/526) | Trim BDD to proto scope | ready (do first) |
| [#527](https://github.com/zynax-io/zynax/issues/527) | Domain layer | blocked on #526 |
| [#528](https://github.com/zynax-io/zynax/issues/528) | gRPC wiring + go.work | blocked on #527 |
| [#481](https://github.com/zynax-io/zynax/issues/481) | Compose wiring | blocked on #528 |

---

## Active Work (Other Tracks)

| Issue | Track | Title | Status |
|-------|-------|-------|--------|
| [#474](https://github.com/zynax-io/zynax/issues/474) | M5.A | Python SDK decision | open — decision required |
| [#476](https://github.com/zynax-io/zynax/issues/476) | M5.B | Guard parser (cel-go) | open — option A/B to decide |
| [#381](https://github.com/zynax-io/zynax/issues/381) | Adapters | git-adapter impl | open (#399 BDD done; #400+ pending) |
| [#382](https://github.com/zynax-io/zynax/issues/382) | Adapters | ci-adapter impl | open (#404 BDD done; #405+ pending) |
| [#383](https://github.com/zynax-io/zynax/issues/383) | Adapters | llm-adapter impl | open (#409 BDD done; #410+ pending) |
| [#384](https://github.com/zynax-io/zynax/issues/384) | Adapters | langgraph-adapter impl | open (#414 BDD done; #415+ pending) |

---

## Known Blockers

- **agent-registry (#480)** — BDD trim (#526) must merge before domain implementation (#527) begins (ADR-016).
- **compose wiring (#481)** — depends on #528 (agent-registry gRPC wiring) landing first.
- **#476 (guard parser)** — requires architectural decision (cel-go vs simple rename) documented in the issue.
- **#474 (Python SDK)** — requires a deliberate decision before any implementation.

---

## Recently Closed

- **#461 M5.D** — Control Plane Security Baseline: all 5 child issues merged (#482–#486).
- **#462 M5.E** — Developer Experience Polish: all child issues merged (#485–#486).
- **#442** — Fully Containerized Makefile: all 4 child issues merged (#443–#446).
- **#529** — docs(agent-registry): REASONS Canvas for #480.
- **#533** — docs(task-broker): REASONS Canvas for #479.
