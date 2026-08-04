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
| M9.A — agent-registry push-path hard-removal (ADR-039) | [#1674](https://github.com/zynax-io/zynax/issues/1674) | `docs/spdd/1674-agent-registry-push-removal/` — Aligned (#1734) | #1697 ✅ → #1698 ✅ (stateless scheduler: repos + DB gone, resync verified live) → #1598 → #1699 |
| M9.B — EventBusService facade hard-removal (ADR-046) | [#1675](https://github.com/zynax-io/zynax/issues/1675) | `docs/spdd/1675-event-bus-facade-removal/` — Aligned (#1734) | #1700 ✅ → #1701 ✅ (facade tree + build/release wiring deleted) → #1702 → #1703 |
| M9.C — named engine-conformance suite | [#1692](https://github.com/zynax-io/zynax/issues/1692) | `docs/spdd/1692-engine-conformance-suite/` — Aligned (#1734) | #1620 ✅ (CRD reconcile assertion now runs on both legs — argo-leg CRD-name collision fixed, verified live) → (steps 2–4 filed via `/lib:spdd-story`) |
| M8.I tail (carried over) — merge-queue fork-canary evidence | [#1680](https://github.com/zynax-io/zynax/issues/1680) — ✅ closed 2026-07-10 | `docs/spdd/1680-merge-queue/` — Implemented | all 5 stories closed; fork-canary PR #1668 merged through the queue unattended (evidence on #1685) |

The three M9 epics are mutually parallel; #1620 has no gate and can merge first. Also riding
alongside: ADR proposals #1693–#1696 (ADR-048..051 — API versioning, OIDC edge auth, fuzz
strategy, load/SLO).

### Delivery status — `/deliver` unblocked 2026-07-10

1. ✅ The three M9 canvases aligned (PR #1734); all nine stories at `status: ready`.
2. ✅ Fork-canary done: PR #1668 merged through the queue unattended; evidence on
   [#1685](https://github.com/zynax-io/zynax/issues/1685); epic #1680 closed.

3. ✅ PyPI publish resolved ([#1732](https://github.com/zynax-io/zynax/issues/1732)):
   trusted publisher registered (maintainer) + dist-staging workflow fix (PR #1736);
   `zynax-sdk 0.1.0` live on PyPI since 2026-07-10 (dispatched run 29082343455).
   Sigstore bundles attach on the next platform tag (v0.8.0).

4. ✅ M9.C step 1 delivered ([#1620](https://github.com/zynax-io/zynax/issues/1620),
   2026-07-10): the Workflow CRD reconcile e2e assertion now runs on **both** engine legs.
   The argo leg was never dispatch-broken — the assertion's unqualified `kubectl get workflow`
   resolved to the co-installed Argo CRD; the script now pins `workflow.zynax.io` and the
   `matrix.engine == 'temporal'` guard is dropped. Verified on a live argo kind cluster.

5. ✅ Hard-removal PRs unblocked ([#1363](https://github.com/zynax-io/zynax/issues/1363),
   2026-08-04): `.github/workflows/pr-size.yml` now honors the documented
   `split-not-possible` label, downgrading the > 900-line hard failure to a warning. M9
   removals are irreducible by construction (PR #1750 for #1701 counts 4,068 lines, 4,038
   of them deletions; the smallest indivisible unit is 949) — they now merge under the
   labelled exception instead of an `--admin` gate bypass. Apply the label, then push a
   commit so the gate re-evaluates (a re-run replays the pre-label event payload).

6. ✅ M9.A step 2 delivered ([#1698](https://github.com/zynax-io/zynax/issues/1698),
   as of 2026-08-04): agent-registry is now physically stateless — memory + Postgres
   `AgentRepository` adapters, the push handler shim, the push-era domain package, and the
   umbrella DB wiring are deleted; `pgx`/`golang-migrate`/`testcontainers` left its `go.mod`
   and `agent-registry` left the `test-integration` allowlist. Restart-recovery-by-informer-
   resync verified twice on one live kind cluster. Next: #1598 (proto RPC removal).

   **Landed as three PRs, not one** (#1751 → #1752 → #1754): at 1846 counted lines the
   single-PR removal was over the 900-line gate, and this delivery predated item 5's
   `split-not-possible` escape hatch. Unlike #1701, this removal *was* reducible — the
   slices are disjoint file sets merged in compile order (handler shim → repo adapters →
   domain package), so splitting was the honest fix rather than a labelled exception.
   Rule of thumb for the remaining removals (#1598 proto, #1675/B `services/event-bus/`):
   split when the layers are compile-independent; label when they genuinely are not.

7. ✅ M9.B steps 1–2 delivered (as of 2026-08-04): the `zynax-event-bus` chart, umbrella
   block, cert identity and 50054 egress are gone ([#1700](https://github.com/zynax-io/zynax/issues/1700),
   PR #1741), and `services/event-bus/` plus its build/release wiring (`go.work`,
   `images/images.yaml`, release + build-images + cleanup matrices, `.dockerignore`,
   CODEOWNERS) are deleted ([#1701](https://github.com/zynax-io/zynax/issues/1701)). The
   `EventBusService` protos/tests BDD suite is retired with a pointer to the
   `libs/zynaxevents` suites, which stay the conventions' contract of record. Remaining:
   #1702 (proto + stubs), #1703 (AsyncAPI + docs truth pass, epic close).

---

## v0.7.0 close ritual — completed 2026-07-10

1. ✅ Tail resolved: #1650 closed (delivered by PR #1673; epic #1576 already closed);
   M8.I epic #1680 moved to M9 (canvas gates its close on the pending fork-canary);
   #1420 (load/SLO harness) moved to unscheduled pending ADR-051 (#1696).
2. ✅ Signed `v0.7.0` tag pushed; Release workflow green (CLI + zynax-ci binaries,
   SBOMs, retag-promoted images per ADR-027); GitHub Release published.
   ⚠️ `SDK PyPI Publish` failed — pre-existing config gap, tracked in #1732
   (resolved 2026-07-10: publisher registered + PR #1736; package live).
3. ✅ GitHub milestones #7 and #8 closed; `state/milestone.yaml` rotated (M7+M8 →
   history, M9 active) in this PR.

---

## Known drift being reconciled

- Stale M-dx epic nest: #173 ⊃ #205 ⊃ #148 ⊃ #146 — consolidation recommended (see the
  triage comments on those issues, 2026-07-08).
- Local branches with unique unmerged commits and no PR, flagged by the 2026-07-10
  `/reconcile` for a human decision: `feat/1492-kind-demo-lifecycle`, `pr-1447`,
  `wavec-rebuild` (1 commit each, 2026-06-19 → 06-25 era; land or delete).
- `images/images.yaml` api-gateway pin is stale (`sha256:c663e687…`, from #1728): the
  2026-07-10 merge-queue batch orphaned #1740's api-gateway staging image because the
  Release retag job promoted only the batch head (#1741). Fixed by [#1742](https://github.com/zynax-io/zynax/issues/1742)
  (batch-aware retag walk) — the first post-merge Release run re-promotes #1740 and
  the digest-sync bot commit lands the current pin. Verify the pin flips after merge.

Resolved by the 2026-07-10 `/reconcile` truth-pass: CLAUDE.md / ROADMAP.md / README /
ARCHITECTURE / M7–M9 planning docs now reflect the v0.7.0 close; #233/#234 label drift
fixed (kept `status: backlog`).
