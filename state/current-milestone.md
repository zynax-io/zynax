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
| M7 — Usable Workflows + Observability | ✅ Delivered — close ritual pending | v0.7.0¹ |
| M8 — CNCF Sandbox + thin-Zynax reduction | ✅ Delivered (tail: M8.I) — close ritual pending | v0.7.0¹ |
| **M9 — Hard Removals + Conformance** | 🚧 **Active on GitHub** (milestone #11) — yaml rotation pending | **v0.8.0 (target)** |
| M-dx — Developer Experience (GitHub #9) · M-UX — User Experience (GitHub #10) | 📅 Planned buckets | see ROADMAP version plan |

¹ **M7 and M8 ship together as one signed v0.7.0 release.** M7 finished at 0 open issues but
was never tagged; its v0.6.0 target was skipped to keep tags monotonic. v1.0.0 stays reserved
for CNCF acceptance. **As of 2026-07-08 the v0.7.0 tag/Release does not exist yet** — see
"Pending close ritual" below.

> ⚠️ **Truth note:** `state/milestone.yaml` still shows `active: M7` because the
> `/milestone close` → `/milestone open M9` rotation was interrupted (its uncommitted work on
> `docs/m8-close-truth-pass` was lost). GitHub is ahead: milestone #11 (M9) exists and is
> populated. This file describes the GitHub truth; only the sanctioned commands rotate the yaml.

---

## Pending close ritual (human runbook, in order)

1. Close the delivered-but-open M8 tail: #1650 and epic #1576 (their work merged in PR #1673);
   decide M8.I (merge-queue epic #1680, stories #1681–#1685, all open) — finish it inside M8 or
   move it out so the milestone can close.
2. `/milestone close` — signed **v0.7.0** tag + GitHub Release covering M7+M8; closes GitHub
   milestones **#7** and **#8**; rotates both into `state/milestone.yaml` history.
3. `/milestone open M9` — activates M9 (GitHub milestone **#11**, label `milestone: M9`,
   target **v0.8.0**); the planning doc `docs/milestones/M9-planning.md` is already in place.
4. Review + align the three M9 canvases (Draft → Aligned), flip their stories
   `status: backlog → status: ready`, then `/deliver`.

---

## M9 — Hard Removals + Conformance (GitHub milestone #11, target v0.8.0)

Plan: **[docs/milestones/M9-planning.md](../docs/milestones/M9-planning.md)** ·
Goal: delete the paths M8 deprecated, per each ADR's removal clause, and formalise the
dual-engine e2e into a named conformance suite.

| EPIC | Issue | Canvas | Stories (in delivery order) |
|------|-------|--------|------------------------------|
| M9.A — agent-registry push-path hard-removal (ADR-039) | [#1674](https://github.com/zynax-io/zynax/issues/1674) | `docs/spdd/1674-agent-registry-push-removal/` — Draft | #1697 → #1698 → #1598 → #1699 |
| M9.B — EventBusService facade hard-removal (ADR-046) | [#1675](https://github.com/zynax-io/zynax/issues/1675) | `docs/spdd/1675-event-bus-facade-removal/` — Draft | #1700 → #1701 → #1702 → #1703 (gated on v0.7.0 published) |
| M9.C — named engine-conformance suite | [#1692](https://github.com/zynax-io/zynax/issues/1692) | `docs/spdd/1692-engine-conformance-suite/` — Draft | #1620 → (steps 2–4 filed on alignment) |

The three epics are mutually parallel; #1620 has no gate and can merge first. Critical path:
close ritual → M9.B chain. Also riding alongside: ADR proposals #1693–#1696
(ADR-048..051 — API versioning, OIDC edge auth, fuzz strategy, load/SLO).

---

## M8 — CNCF Sandbox + thin-Zynax reduction (GitHub milestone #8) — delivered, closing

Plan: [docs/milestones/M8-planning.md](../docs/milestones/M8-planning.md). Delivered 2026-07-03 → 07-07:
governance (M8.A/B), CRD-native scheduler (M8.C #1571, ADR-039), Compose-runtime removal
(M8.D #1572, ADR-041), thin Workflow CRD front-end (M8.E #1573, ADR-043), Envoy Gateway edge
auth+rate-limit (M8.F #1574, ADR-044), ValidatingAdmissionPolicy allow-list (M8.G #1575,
ADR-045), direct NATS JetStream + facade deprecation (M8.H #1576, ADR-046, final step PR #1673).
**Open tail:** #1650/#1576 (delivered, need closing) + **M8.I merge queue** (#1680, stories
#1681–#1685 — ADR-047 reserved). CNCF submission filing remains a maintainer action
([docs/cncf/sandbox-submission.md](../docs/cncf/sandbox-submission.md)).

## M7 — Usable Workflows + Observability (GitHub milestone #7) — delivered, closing

All 12 EPICs closed; milestone at **0 open / 172 closed**. Ships in v0.7.0 (v0.6.0 skipped).
Plan + full delivery log: [docs/milestones/M7-planning.md](../docs/milestones/M7-planning.md).
Deferred out of M7 with their own canvases: #1385 (scenario manifest), #1387 (context
injection); #1359 (zero-Temporal engine) superseded by #1456 (ADR-037 Rejected).

---

## Known drift being reconciled (2026-07-08 truth pass)

- `state/milestone.yaml` active block stale (M7) — rotation owned by `/milestone close|open`.
- Delivered-but-open issues: #1650, #1576 (recommend close; see PR #1673).
- Stale M-dx epic nest: #173 ⊃ #205 ⊃ #148 ⊃ #146 — consolidation recommended (see the
  triage comments on those issues, 2026-07-08).
- Label drift fixed 2026-07-08: #1620 (`milestone: M8` → `M9`), #1524 (stale `milestone: M7`).
