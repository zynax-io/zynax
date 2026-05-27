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
Both are in-progress under M5.C (#460). CloudEvents publishing is log-only (not wired to NATS).
See [docs/milestones/M5-plan.md](../docs/milestones/M5-plan.md).

---

## M5 — Progress

M5 is structured into seven tracks. See full execution plan: **[docs/milestones/M5-plan.md](../docs/milestones/M5-plan.md)**.

### Track Overview

| Track | Epic | Status |
|-------|------|--------|
| **M5.F CI Sprint** | [#542](https://github.com/zynax-io/zynax/issues/542) | 🟡 In Progress — all done except #555 (DRY/KISS L, P2) |
| **M5.F.R Release Pipeline** | [#556](https://github.com/zynax-io/zynax/issues/556) | ✅ Complete (closed) |
| M5.A Truth Pass | [#458](https://github.com/zynax-io/zynax/issues/458) | In Progress — #472 ✅ #473 ✅ #474 ✅ #579 ⬜ open |
| M5.B Engine Correctness | [#459](https://github.com/zynax-io/zynax/issues/459) | ✅ Complete (closed) — #475 ✅ #476 ✅ #477 ✅ #478 ✅ |
| M5.C Capability Dispatch | [#460](https://github.com/zynax-io/zynax/issues/460) | ✅ Compose wired — all 3 services in stack; E2E dispatch pending adapters |
| M5.D Security Baseline | [#461](https://github.com/zynax-io/zynax/issues/461) | ✅ Complete (closed) |
| M5.E DX Polish | [#462](https://github.com/zynax-io/zynax/issues/462) | ✅ Complete (closed) |
| Adapter Library | [#377](https://github.com/zynax-io/zynax/issues/377) | In Progress — http ✅; git/ci/llm/langgraph BDD done, impl pending |
| Containerized Make | [#442](https://github.com/zynax-io/zynax/issues/442) | ✅ Complete (closed) |

---

## IMMEDIATE — Adapters (P2, unblocked by #481 ✅)

### BATCH 0 — ✅ All done
~~#547 #544 #548 #545 #589 #546 #557 #558 #559 #560~~

### BATCH 1 — ✅ All done

| Issue | Title | Size | Status |
|-------|-------|------|--------|
| ~~[#561](https://github.com/zynax-io/zynax/issues/561)~~ | ~~Push service/adapter images to GHCR on every main merge~~ | S | ✅ Done |
| ~~[#601](https://github.com/zynax-io/zynax/issues/601)~~ | ~~Fix Go builder base image 1.25→1.26.3-alpine in service Dockerfiles~~ | XS | ✅ Done |
| ~~[#562](https://github.com/zynax-io/zynax/issues/562)~~ | ~~Make GHCR service/adapter images publicly readable~~ | XS | ✅ Done — 5 service/adapter images public; zynax/tools deleted (see below) |
| (admin) | **Confirm zynax/tools published + set public** | — | ✅ Done — tools-image.yml succeeded 2026-05-20; package set public |
| [#563](https://github.com/zynax-io/zynax/issues/563) | Deduplicate tools image — remove tools-publish.yml + delete old zynax-tools package | XS | ✅ Done |
| [#566](https://github.com/zynax-io/zynax/issues/566) | README packages section with GHCR image pull commands | S | ✅ Done |
| [#552](https://github.com/zynax-io/zynax/issues/552) | Switch all GH Actions jobs to ci-runner container mode | M | ✅ Done |

---

## Active Work (M5.C)

### task-broker (#479) ✅ Complete (closed)

All steps done: impl PRs #520 #522 #523, quality #530 ✅ #531 ✅ #532 ✅. Domain 92.7%.

### agent-registry (#480) ✅ Complete (closed)

All steps done: #526 ✅ #527 ✅ #528 ✅ #481 ✅. Compose wired.

---

## Active Work (Other Tracks)

| Issue | Track | Title | Status |
|-------|-------|-------|--------|
| [#381](https://github.com/zynax-io/zynax/issues/381) | Adapters | git-adapter impl | ✅ Complete — #400 #401 #402 #403 all merged |
| [#713](https://github.com/zynax-io/zynax/issues/713) | Adapters | git-adapter quality epic (coverage ≥85%) | ✅ Complete — #714 ✅ #715 ✅ #716 ✅ #717 ✅ #718 ✅ all merged |
| [#382](https://github.com/zynax-io/zynax/issues/382) | Adapters | ci-adapter impl | open — #404 BDD done; #405+ next (canvas Aligned) |
| [#383](https://github.com/zynax-io/zynax/issues/383) | Adapters | llm-adapter impl | open — #409 BDD done; #410+ pending |
| [#384](https://github.com/zynax-io/zynax/issues/384) | Adapters | langgraph-adapter impl | open — #414 BDD done; #415+ pending |

---

## Known Blockers

- **git-adapter coverage** — ✅ complete; #714–#718 all merged; coverage ≥85% live on CI; git re-added to GO_ADAPTER_LIST.
- **adapter implementations** (#405–#418) — unblocked by #481 ✅; git-adapter impl ✅; ci/llm/langgraph pending.
- **E2E demo** — compose wired (#481 ✅); needs an adapter registered for capability dispatch.
- **v0.4.0 tag** — CHANGELOG promoted; run `git tag -a v0.4.0 -m "M5 Adapter Library" && git push origin v0.4.0` on main to trigger the release workflow and create GitHub Release assets.

---

## Architecture Gaps (open issues to file)

The 2026-05-20 principal architect review identified gaps not yet tracked as issues.
See `docs/milestones/M5-plan.md §Architecture Gaps` for the full list.
Priority gaps to file immediately:

| Gap | Severity |
|-----|----------|
| ~~G1: constant-time bearer compare~~ | ✅ fixed #567 |
| ~~G2: ReadHeaderTimeout + MaxBytesReader~~ | ✅ fixed #568 |
| ~~G4: no RetryPolicy on Temporal Activities (#569)~~ | ✅ fixed #569 |
| ~~G16: background-context goroutines in task-broker (#570)~~ | ✅ fixed #570 |
| G19: competitive positioning doc (Kagent/Dapr) (#575) | High |
| ~~G17: stub services in SERVICE_LIST (#574)~~ | ✅ fixed |
| ~~G23: Phantom AGENT_LIST entries (#577)~~ | ✅ fixed |
| ~~G22: Summarizer phantom (#576)~~ | ✅ fixed |

---

## Recently Closed

- **#567** — Bearer constant-time compare (G1 fix) ✅. **#568** — ReadHeaderTimeout + MaxBytesReader (G2 fix) ✅.
- **#461 M5.D** — Control Plane Security Baseline: all 5 child issues merged (#482–#486).
- **#462 M5.E** — Developer Experience Polish: all child issues merged (#485–#486).
- **#442** — Fully Containerized Makefile: all 4 child issues merged (#443–#446).
- **#529** — docs(agent-registry): REASONS Canvas for #480.
- **#533** — docs(task-broker): REASONS Canvas for #479.
- **#526** — BDD trim (agent-registry), **#532** — handler unit tests (task-broker), **#554** — force-full-pipeline trigger.
- **SECURITY.md** — false mTLS/SBOM/cosign claims removed (2026-05-20, part of M5.A truth pass).

## Recently Closed (this session)

- **#474** #459 #479 #480 #543 #556 — epics closed; all children had merged prior sessions
- **#577** — remove phantom researcher/calculator agents from AGENT_LIST
- **#574** — remove memory-service/event-bus from SERVICE_LIST; add git to GO_ADAPTER_LIST
- **#576** — remove summarizer phantom; delete agents/examples/summarizer/
- **#712** (PR) — summarizer phantom removal, merged 2026-05-26
- **#714** — revert git from GO_ADAPTER_LIST until coverage gate met ✅

## Active Work (BATCH 7.1 — git-adapter coverage)

| Issue | Title | Status |
|-------|-------|--------|
| [#715](https://github.com/zynax-io/zynax/issues/715) | Cover requestReview + progressEvent | ✅ Done — PR #722 merged |
| [#716](https://github.com/zynax-io/zynax/issues/716) | Cover execute/sanitise/githubErrCode/parsePayload | ✅ Done — PR #723 merged |
| [#717](https://github.com/zynax-io/zynax/issues/717) | Cover RegisterAgent retry + isTransient + cmd | ✅ Done — PR #724 merged |
| [#718](https://github.com/zynax-io/zynax/issues/718) | Re-add git to GO_ADAPTER_LIST | 🟡 In Progress — PR open (this PR) |

## Next Session Queue (priority order)

| Priority | Issue | Title | Note |
|----------|-------|-------|------|
| P1 | [#405](https://github.com/zynax-io/zynax/issues/405) | ci-adapter Go module scaffold + config layer (O2) | S, feat, canvas Aligned |
| P2 | [#579](https://github.com/zynax-io/zynax/issues/579) | README per-service status table | S, docs, M5.A |
| P2 | [#555](https://github.com/zynax-io/zynax/issues/555) | DRY/KISS refactor — reusable workflows, composite actions | L, ci, needs-design |
| M6 prep | [#656](https://github.com/zynax-io/zynax/issues/656) | gRPC Health Checking Protocol in all services | L, deferred |
