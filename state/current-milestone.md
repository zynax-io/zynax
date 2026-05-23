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
| **M5.F CI Sprint** | [#542](https://github.com/zynax-io/zynax/issues/542) | 🟡 In Progress — #551 ✅ #552 ✅ #554 ✅ force-full-pipeline; next: **#549** per-service change-detection |
| **M5.F.R Release Pipeline** | [#556](https://github.com/zynax-io/zynax/issues/556) | 🟡 In Progress — #642 ✅ #641 ✅ #655 ✅; next: #549 per-service change-detection |
| M5.A Truth Pass | [#458](https://github.com/zynax-io/zynax/issues/458) | In Progress — 2/3 children done; #474 open |
| M5.B Engine Correctness | [#459](https://github.com/zynax-io/zynax/issues/459) | In Progress — #538 ✅ #539 ✅ #540 ✅; #476 parent open |
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

### task-broker (#479) — code complete, quality in progress

Implementation PRs #520, #522, #523 merged. Domain coverage: 92.7%.

**Open cleanup issues (M5.C):**

| Issue | Step | Status |
|-------|------|--------|
| [#530](https://github.com/zynax-io/zynax/issues/530) | Update AGENTS.md | ✅ Done |
| [#531](https://github.com/zynax-io/zynax/issues/531) | Align service BDD + godog steps | ✅ Done |
| [#532](https://github.com/zynax-io/zynax/issues/532) | Handler unit tests | ✅ Done — api 84.9%, domain 92.7% |

### agent-registry (#480) — pending implementation

Canvas aligned. Ordered delivery: #526 → #527 → #528 → #481.

| Issue | Step | Status |
|-------|------|--------|
| [#526](https://github.com/zynax-io/zynax/issues/526) | Trim BDD to proto scope | ✅ Done |
| [#527](https://github.com/zynax-io/zynax/issues/527) | Domain layer | ✅ Done |
| [#528](https://github.com/zynax-io/zynax/issues/528) | gRPC wiring + go.work | ✅ Done |
| [#481](https://github.com/zynax-io/zynax/issues/481) | Compose wiring | ✅ Done |

---

## Active Work (Other Tracks)

| Issue | Track | Title | Status |
|-------|-------|-------|--------|
| [#474](https://github.com/zynax-io/zynax/issues/474) | M5.A | Python SDK Agent base class | ✅ Complete — #535 ✅ #536 ✅ #537 ✅; BATCH 3 done |
| [#476](https://github.com/zynax-io/zynax/issues/476) | M5.B | Guard parser (cel-go) | ✅ closed — all 3 children merged |
| [#381](https://github.com/zynax-io/zynax/issues/381) | Adapters | git-adapter impl | open (#399 BDD done; #400+ pending, wait for #481) |
| [#382](https://github.com/zynax-io/zynax/issues/382) | Adapters | ci-adapter impl | open (#404 BDD done; #405+ pending, wait for #481) |
| [#383](https://github.com/zynax-io/zynax/issues/383) | Adapters | llm-adapter impl | open (#409 BDD done; #410+ pending, wait for #481) |
| [#384](https://github.com/zynax-io/zynax/issues/384) | Adapters | langgraph-adapter impl | open (#414 BDD done; #415+ pending, wait for #481) |

---

## Known Blockers

- **✅ #655 FIXED** — `tools/healthcheck` static binary added to all 6 distroless Dockerfiles; `docker-compose.yml` migrated from `CMD-SHELL` + `wget`/`nc` to `CMD /healthcheck`; override file removed.
- **#552 ✅ done** — all jobs now run in ci-runner container mode.
- **adapter implementations** (#400–#418) — unblocked by #481 ✅; adapters need a live registry to register against.
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
| G4: no RetryPolicy on Temporal Activities (#569) | Medium |
| G16: background-context goroutines in task-broker (#570) | Medium |
| G19: competitive positioning doc (Kagent/Dapr) (#575) | High |
| G17: stub services in SERVICE_LIST (#574) | Low |

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

- **#661** — COMPILER_ADDR default 50051→50054 in api-gateway (PR #675 ✅)
- **#662** — Makefile scan-image/sbom/build-svc repo-root context (PR #676 ✅)
- **#665** — http-adapter registry_endpoint port 9091→50052 (PR #677 ✅)

## Next Session Queue (priority order)

| Priority | Issue | Title | Note |
|----------|-------|-------|------|
| P1 | [#622](https://github.com/zynax-io/zynax/issues/622) | context.WithTimeout on all gRPC calls (NEW-1) | S, fix, 4 services |
| ~~P1~~ | ~~[#663](https://github.com/zynax-io/zynax/issues/663)~~ | ~~Derive GO_SERVICES from go.work~~ | ✅ Done |
| P1 | [#666](https://github.com/zynax-io/zynax/issues/666) | Align ZYNAX_ENGINE_ACTIVE_ENGINE to full-prefix | XS, fix |
| P1 | [#664](https://github.com/zynax-io/zynax/issues/664) | Correct Go 1.25+ / Helm chart claims in README | XS, docs |
| P1 | [#679](https://github.com/zynax-io/zynax/issues/679) | Translate Temporal NotFound → domain.ErrExecutionNotFound | S, fix — GetStatus/Signal/Cancel return INTERNAL instead of NOT_FOUND |
| P2 | [#549](https://github.com/zynax-io/zynax/issues/549) | Per-service change detection (CI test lanes) | M, ci |
| M6 prep | [#656](https://github.com/zynax-io/zynax/issues/656) | gRPC Health Checking Protocol in all services | L, deferred |
