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
| **M3 — Temporal Execution** | **Next** | v0.2.0 |
| M4 — YAML System + CLI | Planned | v0.3.0 |

---

## M3 — What's Needed

Goal: WorkflowIR executes on Temporal. Engine abstraction proven end-to-end.

- [ ] `engine-adapter` service: Go implementation of `WorkflowEngine` interface
- [ ] `TemporalEngine` adapter: Submit, Signal, Query, Cancel, Watch
- [ ] Generic Temporal "state machine worker" that interprets IR at runtime
- [ ] `DispatchCapabilityActivity`: Temporal Activity that calls task-broker
- [ ] End-to-end test: YAML → IR → Temporal → capability dispatch → result

See [ROADMAP.md §M3](../ROADMAP.md) for the full checklist.
See [Epic #101](https://github.com/zynax-io/zynax/issues/101) for the M2 closure context.

---

## Active PRs (update when state changes)

| PR | Title | Status |
|----|-------|--------|
| #201 | ci: fix proto-generate.yml YAML syntax error | Awaiting merge |
| #202 | docs: update README and ROADMAP to reflect M1 and M2 completion | Awaiting merge |

---

## Known Blockers

None at this time. M3 planning has not started.

---

## Recently Closed

- M2 (Workflow IR) — all 13 issues merged; see Epic #101
- CI Infrastructure Epic #14 — all 8 ACs done; see issues #185–#188
