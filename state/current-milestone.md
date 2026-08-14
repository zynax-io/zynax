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
| M9.A — agent-registry push-path hard-removal (ADR-039) | [#1674](https://github.com/zynax-io/zynax/issues/1674) ✅ **closed** | `docs/spdd/1674-agent-registry-push-removal/` — Implemented (#1699) | #1697 ✅ → #1698 ✅ (stateless scheduler: repos + DB gone, resync verified live) → #1598 ✅ (RPCs gone from the contract; file-scoped `buf breaking` exception per ADR-048 §4; shipped as 7 disjoint slices #1757–#1764) → #1699 ✅ (docs sweep + retired config keys/CLI alias; spike branch deleted; EPIC closed) |
| M9.B — EventBusService facade hard-removal (ADR-046) | [#1675](https://github.com/zynax-io/zynax/issues/1675) ✅ **closed** | `docs/spdd/1675-event-bus-facade-removal/` — Implemented (#1703) | #1700 ✅ → #1701 ✅ (facade tree + build/release wiring deleted) → #1702 ✅ (proto + stubs removed) → #1703 ✅ (AsyncAPI + docs truth pass; EPIC closed) |
| M9.C — named engine-conformance suite | [#1692](https://github.com/zynax-io/zynax/issues/1692) ✅ **closed** | `docs/spdd/1692-engine-conformance-suite/` — Implemented (#1775) | #1620 ✅ (CRD reconcile assertion now runs on both legs — argo-leg CRD-name collision fixed, verified live) → #1773 ✅ (ZECS defined in `docs/conformance/` with honest pass criteria + gap list; membership drift guard) → #1774 ✅ (`zecs-matrix.json` emitted per e2e run + `make conformance-matrix ENGINE=<engine>`; a leg that did not run renders NOT_RUN/SKIPPED, never PASS) → #1775 ✅ (matrix + rendered table published per release, each leg with its enforcement; how-to for adapter authors; EPIC closed) |
| M8.I tail (carried over) — merge-queue fork-canary evidence | [#1680](https://github.com/zynax-io/zynax/issues/1680) — ✅ closed 2026-07-10 | `docs/spdd/1680-merge-queue/` — Implemented | all 5 stories closed; fork-canary PR #1668 merged through the queue unattended (evidence on #1685) |

All three M9 epics are closed (2026-08-14); what remains for the milestone is the v0.8.0
release itself. Also riding alongside: ADR proposals #1693–#1696 (ADR-048..051 — API
versioning, OIDC edge auth, fuzz strategy, load/SLO).

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
   `libs/zynaxevents` suites, which stay the conventions' contract of record.

8. ✅ M9.B step 3 delivered (as of 2026-08-04): `protos/zynax/v1/event_bus.proto` and its
   Go/Python stubs are deleted ([#1702](https://github.com/zynax-io/zynax/issues/1702)) —
   the first proto hard-removal in the repo. `buf breaking` carries a documented
   intentional-removal exception in `protos/buf.yaml` scoped to that single file
   (ADR-048 §Decision 4); a control run proved another deleted proto still fails the gate.

9. ✅ **M9.B complete — EPIC [#1675](https://github.com/zynax-io/zynax/issues/1675) closed**
   (as of 2026-08-04): step 4 ([#1703](https://github.com/zynax-io/zynax/issues/1703)) dropped
   the `x-zynax-deprecated` gRPC access path from `spec/asyncapi/zynax-events.yaml` (channels
   byte-identical — they are the contract, realised by `libs/zynaxevents`) and truth-passed the
   eventing surfaces: `docs/patterns/direct-jetstream-events.md`, ARCHITECTURE §8 async path,
   the README service table, and `services/task-broker/AGENTS.md` (which documented
   `ZYNAX_BROKER_EVENTBUS_ADDR`, an env var that never existed in code — the real gate is
   `ZYNAX_BROKER_NATS_URL`). Historical records (CHANGELOG, ADR-022/046, M1/M5/M6/M8 planning,
   past canvases, `docs/ai-learnings/`) were left intact by design. `make validate-spec` green.
   ADR-046 is now fully executed: deploy → code → contract → spec.

10. ✅ M9.A step 3 delivered ([#1598](https://github.com/zynax-io/zynax/issues/1598), as of
    2026-08-04): the five deprecated `AgentRegistryService` RPCs and their nine
    request/response messages are gone from `protos/zynax/v1/agent_registry.proto`;
    `AgentDef`/`CapabilityDef`/`AgentStatus` stay (scheduler.proto reuses them) so the file
    keeps its name and every importer still resolves. `buf breaking` was **not** disabled —
    `protos/buf.yaml` carries a documented `ignore_only` stanza scoped to this one file and
    the exact two rules the removal trips (`SERVICE_NO_DELETE`, `MESSAGE_NO_DELETE`), the
    ADR-048 §Decision 4 mechanism #1756 established for `event_bus.proto`. All six caller
    surfaces went first: the five Go/Python adapter registration clients with their
    `UNIMPLEMENTED` tolerance shims, the SDK + contract BDD suites, and the api-gateway's
    `kind: AgentDef` route (now 400 `UNSUPPORTED_KIND`, not 410 — the migration window is
    closed). 5115 counted lines shipped as seven disjoint slices
    (#1757, #1758, #1759, #1760, #1762, #1763 → #1764) with no `split-not-possible` label.
    Runtime-verified: every adapter boots `SERVING` twice with the registry endpoint pointed
    at a dead port, and a live gateway rejects `kind: AgentDef` across a restart.

11. ✅ DLQ forwarding delivered ([#1653](https://github.com/zynax-io/zynax/issues/1653), as of
    2026-08-04): `libs/zynaxevents` now moves an exhausted message into `DLQ_<src>` instead of
    only provisioning the stream. The opt-in `Client.StartDLQForwarder` consumes
    `$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>`, fetches the message by sequence and
    republishes it byte-for-byte on the reserved exact `zynax.dlq.<prefix>.dead` subject; it
    never deletes the source, and a deterministic `DLQ_<stream>:<sequence>` message id makes
    replays, retries and a second forwarder collapse onto one DLQ message. Not started by
    `Subscribe` — the advisory subject is server-global and the per-identity NATS policy
    (ADR-046 Decision #4) grants no publisher advisory-subscribe or `zynax.dlq.>` publish, so
    an implicit mover would regress every subscriber. The BDD scenario no longer stops at the
    advisory: it asserts the message lands on `DLQ_<src>`, byte-identical to the still-present
    source, and that replaying the advisory does not duplicate it. Verified twice on one
    file-backed JetStream volume; the golden byte-compat fixtures are unchanged.

12. ✅ M9.A step 4 delivered ([#1699](https://github.com/zynax-io/zynax/issues/1699), as of
     2026-08-04) — **EPIC #1674 closed, M9.A complete**. Docs/status truth pass plus the two
     items #1598 deferred: the dead *required* config keys `registry_endpoint` (adk/ci/git/http/
     llm) and `REGISTRY_ADDR` (langgraph) are gone, and `zynax agent publish` is a hidden
     retirement stub. The directions are **not** symmetric, and CI proved it: a NEW image with
     the key still set boots fine (Go `yaml.Unmarshal` is non-strict; the langgraph model is
     `extra="ignore"`), but an OLD image with the key *deleted* fails startup validation — the
     first push dropped `REGISTRY_ADDR` from `scripts/e2e/manifests/echo-worker.yaml` and both
     e2e legs timed out because that Deployment pins `langgraph-adapter:main`, an image the PR
     does not rebuild. So the code stops reading the keys here and the deployed manifests keep
     them behind an explanatory comment; a follow-up drops them once the rebuilt `:main` images
     ship. Documented as an ordering rule in `docs/patterns/agent-crd-migration.md`. `zynax init expert` /
     `agent init` and `spec/schemas/agent-def.schema.json` were **kept**: an AgentDef manifest is
     still the expert-authoring format (ADR-028/ADR-033), only its api-gateway push is gone.
     `spike/adr-039-crd-scheduler-proof` deleted (local + remote, last SHA `1c30b17`) — the
     standing "keep the spike" instruction is retired. History (CHANGELOG, ADRs, past canvases,
     M1–M8 plans, `docs/reviews/`, `docs/due-diligence/`, `docs/ai-learnings/`) left intact.
     **Two pre-existing breakages found and NOT fixed here** (runtime-confirmed against a booted
     api-gateway, follow-up filed): `zynax apply <scenario-dir>` aborts because a Scenario's
     `apply_order` POSTs its AgentDef members to `/api/v1/apply` (400 `UNSUPPORTED_KIND`), and
     `infra/packages/code-review-rank/apply-job.yaml` does the same while shipping no `Agent` CR.

13. ✅ M9.C step 2 delivered ([#1773](https://github.com/zynax-io/zynax/issues/1773), as of
     2026-08-05): the suite is named and defined — **ZECS v0.8.0** in `docs/conformance/`
     (definition + `scenarios.yaml` membership manifest over `spec/workflows/examples/`, no
     shadow copy, no second harness). The pass criteria are transcribed from what
     `scripts/e2e/*.sh` actually assert, so the definition ships with a **gap list**: ZECS runs
     4 scenarios, only **1** (`e2e-demo.yaml`) on **both** legs; the two legs assert different
     depths for it (the enforced intersection is terminal status — the argo leg asserts no
     CloudEvents and no outputs); declared-output equality and the failure path are
     temporal-only; the CRD scenario asserts dispatch, not completion; and the `argo` leg is
     still advisory, not required, on `main`. Gaps G1–G6 are written into the reference doc for
     a follow-up story. New drift guard `zynax-ci check conformance` (`make check-conformance`,
     unconditional `conformance-check` job in `ci.yml`) reconciles membership ↔ corpus ↔ the
     engine-adapter's selectable engines ↔ the e2e matrix legs — no engine name is hardcoded in
     it, so engine N+1 turns the suite red until the leg really runs. PR cadence unchanged.

14. ✅ M9.C step 3 delivered ([#1774](https://github.com/zynax-io/zynax/issues/1774), as of
     2026-08-14): a ZECS run now produces a machine-readable result. Every e2e leg records its
     per-scenario step outcomes and a fan-in `ZECS matrix` job renders one `zecs-matrix.json`
     covering **every** engine the engine-adapter can be configured with (90-day retention;
     per-leg inputs 14 days). The correctness property is asserted, not intended: only an
     observed success renders `PASS`, a leg that never ran is present as `NOT_RUN` rather than
     omitted, and `not_in_leg` (the manifest says this leg does not run it) stays distinct from
     `not_executed` (it was meant to and produced nothing → the leg is `INCOMPLETE` and the run
     `complete: false`). Enforcement is carried per leg with a separate `enforced_result`, so the
     advisory argo leg (gap G6) can never read as merge-blocking.
     `make conformance-matrix ENGINE=<engine>` reproduces the same document locally by running
     that leg's manifest runners. The job is advisory by construction: it reports, it never
     gates, and it cannot turn an e2e leg red — the PR e2e gate is byte-for-byte unchanged.

15. ✅ M9.C step 4 delivered and **EPIC #1692 closed** — the last open M9 epic
     ([#1775](https://github.com/zynax-io/zynax/issues/1775), as of 2026-08-14): the portability
     claim is now checkable from a release page. Every release attaches `zecs-matrix.json` plus
     `zecs-conformance.md` and carries the rendered section in its notes, headed by the one-line
     claim (`ZECS v0.8.0 — argo PASS (advisory), temporal PASS (required) · required legs: PASS
     · all legs: PASS`). The table is rendered from the attached document by
     `zynax-ci conformance render` — never hand-written — and the e2e job summary now uses the
     same rendering over the same JSON, so the human view and the machine view cannot drift.
     Each leg is published with its `enforcement` and both aggregates, so an advisory PASS never
     reads as merge-blocking and the open branch-protection decision
     ([#1778](https://github.com/zynax-io/zynax/issues/1778)) changes the line with no template
     edit. A release may cite only a run whose revision is an ancestor of the released commit
     (PR-head runs are excluded by construction); when none qualifies — e2e is PR-triggered and
     path-conditional, artifacts expire at 90 days — the same tool emits an all-`NOT_RUN`,
     `complete: false` document and the notes lead with "INCOMPLETE run — not a conformance
     result", rather than omitting the matrix or presenting a partial run as complete. Adapter
     authors get a Diátaxis how-to (`docs/conformance/how-to-run-zecs.md`) beside the reference.
     Step 3's deferral is closed too: `ci_step` keys resolve statically against the e2e job's
     step ids, so a renamed step fails at PR time instead of inside a published matrix.
     All three M9 epics (#1674, #1675, #1692) are now closed; M9 stays **Active** until the
     v0.8.0 release itself is cut.

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
