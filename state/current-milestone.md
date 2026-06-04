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
| **M5 — Adapter Library** | ✅ **Complete** | **v0.4.0** |
| **M6 — K8s Production-Ready** | 🚀 **Active** | — |

M3/M4 are partial because task-broker and agent-registry were not delivered in those milestones.
Both completed under M5.C (#460). CloudEvents publishing is log-only (not wired to NATS).
v0.4.0 tag pushed 2026-05-29; GitHub Release live at https://github.com/zynax-io/zynax/releases/tag/v0.4.0
GitHub milestone "Adapter Library (M5)" closed 2026-05-29; 5 deferred issues (#235 #239 #376 #466 #656) moved to M6.
See [docs/milestones/M5-plan.md](../docs/milestones/M5-plan.md).

---

## M5 — Progress

M5 is structured into seven tracks. See full execution plan: **[docs/milestones/M5-plan.md](../docs/milestones/M5-plan.md)**.

### Track Overview

| Track | Epic | Status |
|-------|------|--------|
| **M5.F CI Sprint** | [#542](https://github.com/zynax-io/zynax/issues/542) | ✅ Complete (closed) — #555 ✅ all child issues done |
| **M5.F.R Release Pipeline** | [#556](https://github.com/zynax-io/zynax/issues/556) | ✅ Complete (closed) |
| M5.A Truth Pass | [#458](https://github.com/zynax-io/zynax/issues/458) | ✅ Complete — #472 ✅ #473 ✅ #474 ✅ #572 ✅ #579 ✅ |
| M5.B Engine Correctness | [#459](https://github.com/zynax-io/zynax/issues/459) | ✅ Complete (closed) — #475 ✅ #476 ✅ #477 ✅ #478 ✅ |
| M5.C Capability Dispatch | [#460](https://github.com/zynax-io/zynax/issues/460) | ✅ Complete — e2e-demo.yaml created; run `make run-local && zynax apply spec/workflows/examples/e2e-demo.yaml` |
| M5.D Security Baseline | [#461](https://github.com/zynax-io/zynax/issues/461) | ✅ Complete (closed) |
| M5.E DX Polish | [#462](https://github.com/zynax-io/zynax/issues/462) | ✅ Complete (closed) |
| Adapter Library | [#377](https://github.com/zynax-io/zynax/issues/377) | ✅ Complete (closed) — all five adapters merged |
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
| ~~[#382](https://github.com/zynax-io/zynax/issues/382)~~ | Adapters | ci-adapter impl | ✅ Closed — all steps done (#404–#408) |
| ~~[#383](https://github.com/zynax-io/zynax/issues/383)~~ | Adapters | llm-adapter impl | ✅ Closed — #409 BDD #410 #411 #412 #413 (PR #742) all merged |
| ~~[#384](https://github.com/zynax-io/zynax/issues/384)~~ | Adapters | langgraph-adapter impl | ✅ Closed — #414 BDD #415 #416 #417 #418 (PR #743) all merged |

---

## Known Blockers

- **git-adapter coverage** — ✅ complete; #714–#718 all merged; coverage ≥85% live on CI; git re-added to GO_ADAPTER_LIST.
- **adapter implementations** (#405–#418) — ✅ all done; git/ci/llm/langgraph all merged.
- **E2E demo** — ✅ `e2e-demo.yaml` created; langgraph `echo` capability wired; run `make run-local && zynax apply spec/workflows/examples/e2e-demo.yaml` to observe dispatch + completion in Temporal UI (http://localhost:7088).
- **v0.4.0 tag** — ✅ pushed 2026-05-29; GitHub Release live with CLI binaries, GHCR service images, and SBOMs.

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
| ~~G19: competitive positioning doc (Kagent/Dapr) (#575)~~ | ✅ fixed |
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

## Active Work (BATCH 7.1 — git-adapter coverage — ✅ COMPLETE)

| Issue | Title | Status |
|-------|-------|--------|
| [#715](https://github.com/zynax-io/zynax/issues/715) | Cover requestReview + progressEvent | ✅ Done — PR #722 merged |
| [#716](https://github.com/zynax-io/zynax/issues/716) | Cover execute/sanitise/githubErrCode/parsePayload | ✅ Done — PR #723 merged |
| [#717](https://github.com/zynax-io/zynax/issues/717) | Cover RegisterAgent retry + isTransient + cmd | ✅ Done — PR #724 merged |
| [#718](https://github.com/zynax-io/zynax/issues/718) | Re-add git to GO_ADAPTER_LIST | ✅ Done — PR #725 merged |

## Active Work (BATCH 7 — adapter O5 step) — ✅ COMPLETE

| Issue | Title | Status |
|-------|-------|--------|
| [#412](https://github.com/zynax-io/zynax/issues/412) | llm-adapter registry client + bootstrap | ✅ Done — merged |
| [#417](https://github.com/zynax-io/zynax/issues/417) | langgraph-adapter registry client + bootstrap | ✅ Done — merged |
| [#413](https://github.com/zynax-io/zynax/issues/413) | llm-adapter Dockerfile + docker-compose + AGENTS.md | ✅ Done — PR #742 merged |
| [#418](https://github.com/zynax-io/zynax/issues/418) | langgraph-adapter Dockerfile + docker-compose + AGENTS.md | ✅ Done — PR #743 merged |

## Active Work (BATCH 8 — Code Quality)

| Issue | Title | Status |
|-------|-------|--------|
| [#373](https://github.com/zynax-io/zynax/issues/373) | Thread ctx in workflow-compiler gRPC handlers | ✅ Done |
| [#374](https://github.com/zynax-io/zynax/issues/374) | ctx-first mandate in services/AGENTS.md + Temporal comment | ✅ Done |
| [#375](https://github.com/zynax-io/zynax/issues/375) | Enable ruff D (Google docstrings) in agents/sdk | ✅ Done |

## Active Work (BATCH 9 — Documentation Quality)

| Issue | Title | Status |
|-------|-------|--------|
| [#232](https://github.com/zynax-io/zynax/issues/232) | Architecture fitness functions doc | ✅ Done — PR #750 merged |
| [#248](https://github.com/zynax-io/zynax/issues/248) | AI-output review checklist in PR template | ✅ Done — PR #751 pending |
| [#228](https://github.com/zynax-io/zynax/issues/228) | Google-style docstrings on SDK public symbols | ✅ Done |
| [#229](https://github.com/zynax-io/zynax/issues/229) | Strip explanatory comments in agents/sdk | ✅ Done |

---

## M6 — Active Work

### M6.A Health Probe Semantics (#463) — ✅ COMPLETE

| Story | Issue | Status |
|-------|-------|--------|
| A.1 split probes in api-gateway | [#487](https://github.com/zynax-io/zynax/issues/487) | ✅ Merged (#821) |

Canvas: `docs/spdd/463-health-probes/canvas.md` — Status: Implemented

### M6.D Stateless Compiler (#466) — ✅ COMPLETE

| Story | Issue | Status |
|-------|-------|--------|
| D.1 drop in-memory IR store | [#490](https://github.com/zynax-io/zynax/issues/490) | ✅ Merged (#774) |

Canvas: `docs/spdd/466-stateless-compiler/canvas.md` — Status: Implemented

### EBUS-DECISION (#764) — ✅ RESOLVED

**ADR-022 accepted — Option 1: Full gRPC EventBusService wrapping NATS JetStream.** (PR #822)
EPIC I (#772) unblocked. Stories created: #823 #824 #825 #826 #827 #828.
Next: `/spdd-reasons-canvas 772` → canvas Aligned → implement I.1 (#823) first.

### M6.B Inter-Service mTLS (#464) — ✅ COMPLETE

| Story | Issue | Status |
|-------|-------|--------|
| O1 mTLS env-var cert paths + credential wiring | [#488](https://github.com/zynax-io/zynax/issues/488) | ✅ Merged (#831) |

Canvas: `docs/spdd/464-mtls/canvas.md` — Status: Implemented

### M6.C Supply Chain Hardening (#465) — ✅ COMPLETE

| Story | Issue | Status |
|-------|-------|--------|
| O1 cosign signing + SBOM (SPDX) + multi-arch in release workflows | [#489](https://github.com/zynax-io/zynax/issues/489) | ✅ Merged (#833) |

Canvas: `docs/spdd/465-supply-chain/canvas.md` — Status: Implemented

### M6.Helm — Helm Charts for All Services (#765) — 🔄 In Progress

Canvas: `docs/spdd/765-helm-charts/canvas.md` — Status: **Aligned** ✅

| Story | Issue | Status |
|-------|-------|--------|
| A.0 feat(infra): shared zynax-lib library chart | [#779](https://github.com/zynax-io/zynax/issues/779) | ✅ Merged (#872) |
| A.1 feat(infra): Helm chart for api-gateway | [#780](https://github.com/zynax-io/zynax/issues/780) | ✅ Merged (#886) |
| A.2 feat(infra): Helm chart for workflow-compiler | [#781](https://github.com/zynax-io/zynax/issues/781) | ✅ Merged |
| A.3 feat(infra): Helm chart for engine-adapter | [#782](https://github.com/zynax-io/zynax/issues/782) | ✅ Merged |
| A.4–A.13: remaining stories | [#783](https://github.com/zynax-io/zynax/issues/783)–[#792](https://github.com/zynax-io/zynax/issues/792) | ⬜ Pending |

### M6.Images — Single Source of Truth for Container Image References (#855) — ⬜ Canvas Aligned, not started

Canvas: `docs/spdd/855-images-sot/canvas.md` — Status: **Aligned** ✅

| Story | Issue | Status |
|-------|-------|--------|
| O1 chore(ci): images/images.yaml schema + initial population | [#856](https://github.com/zynax-io/zynax/issues/856) | ⬜ Open |
| O2 feat(zynax-ci): images sync + check subcommands | [#857](https://github.com/zynax-io/zynax/issues/857) | ⬜ Open |
| O3 ci: wire drift-check into CI | [#858](https://github.com/zynax-io/zynax/issues/858) | ⬜ Open (**ships with O2**) |
| O4 chore(infra): Dockerfile ARG migration | [#859](https://github.com/zynax-io/zynax/issues/859) | ⬜ Open |
| O5 chore(ci): bump flow rewrite (closes #844) | [#860](https://github.com/zynax-io/zynax/issues/860) | ⬜ Open |
| O6 docs: propagation | [#861](https://github.com/zynax-io/zynax/issues/861) | ⬜ Open |
| O7 docs: ADR-024 | [#862](https://github.com/zynax-io/zynax/issues/862) | ⬜ Open |

**Keystone**: O2 + O3 must ship in the same sprint; neither is done without the other.

### M6.Images — GHCR Package Hygiene — ⬜ Ready to implement

Investigation confirmed (2026-06-03): all 8 GHCR images have `"annotations": null` on their OCI index manifests → "No description" in GHCR UI. Two `unknown/unknown` rows per image are SLSA provenance attestations (expected). No retention policy exists.

Delivery order (each is its own PR):

| Story | Issue | Status |
|-------|-------|--------|
| docs(adr): ADR-025 — keep vs disable SLSA attestations | [#868](https://github.com/zynax-io/zynax/issues/868) | ⬜ Open |
| ci: OCI manifest annotations (fix "no description") | [#865](https://github.com/zynax-io/zynax/issues/865) | ⬜ Open — depends on #868 |
| ci: description-present gate + size-budget check | [#866](https://github.com/zynax-io/zynax/issues/866) | ⬜ Open — depends on #865 |
| chore(ci): GHCR retention cap (last 5 builds) | [#867](https://github.com/zynax-io/zynax/issues/867) | ⬜ Open |
| docs: document unknown/unknown attestation manifests | [#869](https://github.com/zynax-io/zynax/issues/869) | ⬜ Open — depends on #868 |

### M6 Infra / Tooling — ✅ Complete

Process-health work (ADR-023, merge-policy, image-bump tooling, /resume-m6 fix).
All PRs merged 2026-06-03.

| Work Item | Issue | PR | Status |
|-----------|-------|-----|--------|
| ADR-023 — restrict direct pushes to main | — | #847 | ✅ Merged |
| chore(ci): bump-ci-runner script + make target | [#843](https://github.com/zynax-io/zynax/issues/843) | #848 | ✅ Merged |
| ci(infra): post-build bump issue from tools-image.yml | [#844](https://github.com/zynax-io/zynax/issues/844) | #849 | ✅ Merged |
| chore(claude): /resume-m6 rewrite — FF discipline | [#845](https://github.com/zynax-io/zynax/issues/845) | #850 | ✅ Merged |
| docs(contributing): merge policy | [#846](https://github.com/zynax-io/zynax/issues/846) | #851 | ✅ Merged |
| docs(milestones): tracking + ROADMAP expansion | — | #852 | ✅ Merged |

### M6.DevAuto — Self-hosting dev-automation (#873) — 🔄 In Progress

Canvas: SPDD-exempt (docs:/chore:/ci: stories only, until Wave 4 #881 which is BLOCKED on #626 + #772)

| Story | Issue | Status |
|-------|-------|--------|
| DevAuto.1 docs(automation): STATUS-AND-DIRECTION.md | [#874](https://github.com/zynax-io/zynax/issues/874) | ✅ Merged (#884) |
| DevAuto.2 chore(automation): expert mesh YAML configs | [#875](https://github.com/zynax-io/zynax/issues/875) | ⬜ Next |
| DevAuto.3 chore(automation): orchestrator config | [#876](https://github.com/zynax-io/zynax/issues/876) | ⬜ Pending |
| DevAuto.4–7 ci: Waves 0–3 | [#877](https://github.com/zynax-io/zynax/issues/877)–[#880](https://github.com/zynax-io/zynax/issues/880) | ⬜ Pending |
| DevAuto.8 feat: Wave 4 aspirational | [#881](https://github.com/zynax-io/zynax/issues/881) | ⬜ **BLOCKED on #626 + #772** |
| DevAuto.9 test: xfail gate | [#882](https://github.com/zynax-io/zynax/issues/882) | ⬜ Pending |
| DevAuto.10 docs: AGENTS.md pointer + README | [#883](https://github.com/zynax-io/zynax/issues/883) | ⬜ Pending |

---

## Next Session Queue (priority order)

Remaining open M5 non-epic issues:
- **#228** (docs: SDK docstrings, S) — ✅ Done
- **#229** (refactor: strip comments, S) — ✅ Done
- **#376** (docs: SDK docstrings step 2) — BLOCKED on SDK modules (M6+ scope)
- **#235**, **#239** (SBOM/SLSA) — superseded by M6.C #489; close when M6 activates
- **#656** (feat: gRPC Health Checking, L) — M6 prep; defer to M6
