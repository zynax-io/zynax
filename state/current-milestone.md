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

## M4 — Progress

Goal: `zynax apply workflow.yaml` compiles, submits, and returns a `run_id`. Users can
manage workflow runs from the terminal via the `zynax` CLI.

- [x] api-gateway service: HTTP REST layer — `/api/v1/apply` + `/api/v1/workflows/{id}` (#315, merged)
- [x] api-gateway: `kind: AgentDef` routing via `AgentRegistryService` (#316, merged)
- [x] `zynax` CLI: `apply`, `get`, `delete`, `status` commands (#317, #330, merged)
- [x] `zynax logs`: SSE streaming `WatchWorkflow` events (#318, #338, merged)
- [x] Local Docker Compose runner — `make run-local` / `make stop-local` (#319, PR #340 open)
- [x] CLI release CI — multi-platform binaries published to GitHub Releases on `v*.*.*` tag
- [ ] `zynax gitops watch <dir>` sub-command (#320)
- [ ] `zynax validate` — Go-based manifest/canvas/schema validation (#331 epic, steps #332–#336)

See [Canvas](../docs/spdd/314-yaml-system-cli/canvas.md) and [Epic #314](https://github.com/zynax-io/zynax/issues/314).

---

## Active PRs

| PR | Title | Status |
|----|-------|--------|
| #340 | feat(infra): Docker Compose local runner + service Dockerfiles (#319) | Open — awaiting review |

---

## Known Blockers

None.

---

## Recently Closed

- M3 (Temporal Execution) — all 5 step issues (#301–#305) merged; all BDD scenarios pass.
  Canvas: `docs/spdd/214-temporal-execution/canvas.md` (status: Implemented).
- M4 Step 1 (#315): api-gateway HTTP REST layer merged.
- M4 Step 2 (#316): api-gateway AgentDef routing merged.
- M4 Step 3 (#317, #330): `zynax` CLI apply/get/delete/status merged.
- M4 Step 4 (#318, #338): `zynax logs` SSE streaming merged.
