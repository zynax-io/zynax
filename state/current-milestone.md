<!-- Canonical status file. Updated by /milestone open|close, delivery PRs, and /reconcile. Do not edit by hand outside those flows. -->

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
| M3 — Temporal Execution | ✅ Complete (task-broker landed M5.C) | v0.2.0 |
| M4 — YAML System + CLI | ✅ Complete (agent-registry landed M5.C) | v0.3.0 |
| M5 — Adapter Library | ✅ Complete | v0.4.0 |
| M6 — K8s Production-Ready | ✅ Complete | v0.5.0 |
| M7 — Usable Workflows + Observability | ✅ Complete | v0.7.0¹ |
| M8 — CNCF Sandbox + thin-Zynax reduction | ✅ Complete | v0.7.0¹ |
| **M9 — Hard Removals + Conformance** | 🚧 **Active** (GitHub milestone #11) | **v0.8.0 (target)** |
| M-dx — Developer Experience (GitHub #9) · M-UX — User Experience (GitHub #10) | 📅 Planned buckets | see ROADMAP version plan |

¹ **M7 and M8 shipped together as the single signed v0.7.0 release on 2026-07-10**
([release](https://github.com/zynax-io/zynax/releases/tag/v0.7.0)): signed tag, GitHub
Release with CLI/zynax-ci binaries + per-service SBOMs, milestones #7 and #8 closed,
`state/milestone.yaml` rotated. v0.6.0 was skipped to keep tags monotonic; v1.0.0 stays
reserved for CNCF acceptance.

---

## M9 — Hard Removals + Conformance (GitHub milestone #11, target v0.8.0) — ACTIVE

Plan: **[docs/milestones/M9-planning.md](../docs/milestones/M9-planning.md)** ·
Goal: delete the paths M8 deprecated, per each ADR's removal clause, and formalise the
dual-engine e2e into a named conformance suite.

| EPIC | Issue | Canvas | Stories (in delivery order) |
|------|-------|--------|------------------------------|
| M9.A — agent-registry push-path hard-removal (ADR-039) | [#1674](https://github.com/zynax-io/zynax/issues/1674) | `docs/spdd/1674-agent-registry-push-removal/` — Draft | #1697 → #1698 → #1598 → #1699 |
| M9.B — EventBusService facade hard-removal (ADR-046) | [#1675](https://github.com/zynax-io/zynax/issues/1675) | `docs/spdd/1675-event-bus-facade-removal/` — Draft | #1700 → #1701 → #1702 → #1703 (v0.7.0 gate now satisfied) |
| M9.C — named engine-conformance suite | [#1692](https://github.com/zynax-io/zynax/issues/1692) | `docs/spdd/1692-engine-conformance-suite/` — Draft | #1620 → (steps 2–4 filed on alignment) |
| M8.I tail (carried over) — merge-queue fork-canary evidence | [#1680](https://github.com/zynax-io/zynax/issues/1680) | `docs/spdd/1680-merge-queue/` — Aligned, delivered | all 5 stories closed; epic open only for the fork-PR dry run (maintainer-armed, candidate PR #1668) |

The three M9 epics are mutually parallel; #1620 has no gate and can merge first. Also riding
alongside: ADR proposals #1693–#1696 (ADR-048..051 — API versioning, OIDC edge auth, fuzz
strategy, load/SLO).

### Blockers / human actions before delivery can start

1. **Align the three M9 canvases** (Draft → Aligned), then flip their stories
   `status: backlog → status: ready`, then `/deliver`.
2. **Fork-canary for #1680:** arm a green fork PR (candidate #1668) with
   `gh pr merge --squash --auto`; evidence goes on #1685, then close #1680.
3. **PyPI trusted publisher never registered** ([#1732](https://github.com/zynax-io/zynax/issues/1732)):
   `zynax-sdk` has never actually published (v0.5.0 and v0.7.0 runs both failed
   `invalid-publisher`; the package 404s on PyPI). Register the pending publisher on
   pypi.org (project `zynax-sdk`, workflow `sdk-publish.yml`, environment `pypi`), then
   re-run the v0.7.0 `SDK PyPI Publish` run.

---

## v0.7.0 close ritual — completed 2026-07-10

1. ✅ Tail resolved: #1650 closed (delivered by PR #1673; epic #1576 already closed);
   M8.I epic #1680 moved to M9 (canvas gates its close on the pending fork-canary);
   #1420 (load/SLO harness) moved to unscheduled pending ADR-051 (#1696).
2. ✅ Signed `v0.7.0` tag pushed; Release workflow green (CLI + zynax-ci binaries,
   SBOMs, retag-promoted images per ADR-027); GitHub Release published.
   ⚠️ `SDK PyPI Publish` failed — pre-existing config gap, tracked in #1732.
3. ✅ GitHub milestones #7 and #8 closed; `state/milestone.yaml` rotated (M7+M8 →
   history, M9 active) in this PR.

---

## Known drift being reconciled

- `CLAUDE.md` / `ROADMAP.md` milestone tables still describe the v0.7.0 close as pending —
  next `/reconcile` truth-pass updates them (state/* is authoritative as of this PR).
- Docs claiming "SDK on PyPI" (M6 deliverable) are aspirational until #1732 is fixed —
  the package has never been on PyPI.
- Stale M-dx epic nest: #173 ⊃ #205 ⊃ #148 ⊃ #146 — consolidation recommended (see the
  triage comments on those issues, 2026-07-08).
- Label drift: #233/#234 carry both `status: ready` and `status: backlog` (backlog is the
  newer intent — flagged 2026-07-10).
