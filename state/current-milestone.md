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
| M3 — Temporal Execution | ✅ Complete | v0.2.0 |
| **M4 — YAML System + CLI** | **In Progress** | v0.3.0 |

---

## M4 — What's Needed

Goal: `zynax apply workflow.yaml` compiles, submits, and returns a `run_id`. Users can
manage workflow runs from the terminal via the `zynax` CLI.

- [ ] `api-gateway` service: Go implementation — `/api/v1/apply` + `/api/v1/workflows/{id}` (#315)
- [ ] `api-gateway` `kind: AgentDef` routing via `AgentRegistryService` (#316)
- [ ] `zynax` CLI: `apply`, `get`, `delete`, `status` commands (#317)
- [ ] `zynax logs`: streaming `WatchWorkflow` events (#318)
- [ ] Local Docker Compose runner — `make run-local` (#319)
- [ ] `zynax gitops watch <dir>` sub-command (#320)

See [ROADMAP.md §M4](../ROADMAP.md) and [Canvas #314](../docs/spdd/314-yaml-system-cli/canvas.md).

---

## Active PRs (update when state changes)

| PR | Title | Status |
|----|-------|--------|
| #321 | docs: add M4 YAML System + CLI REASONS Canvas (#314) | Awaiting merge |

---

## Known Blockers

None. M4 implementation begins with issue #315.

---

## Recently Closed

- M3 (Temporal Execution) — engine-adapter fully implemented; smoke tests pass
- See Epic #214 and Canvas `docs/spdd/214-temporal-execution/canvas.md` (status: Implemented)
