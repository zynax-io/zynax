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
| **M5.F CI Sprint** | [#542](https://github.com/zynax-io/zynax/issues/542) | 🟡 In Progress — #551 ✅ Dockerfile.ci-runner; #552 ✅ container-mode PR merged; next: **#554** force-full-pipeline trigger |
| **M5.F.R Release Pipeline** | [#556](https://github.com/zynax-io/zynax/issues/556) | 🟢 BATCH 1 complete — #566 ✅ Docker Images in README; GHCR image refs in docker-compose |
| M5.A Truth Pass | [#458](https://github.com/zynax-io/zynax/issues/458) | In Progress — 2/3 children done; #474 open |
| M5.B Engine Correctness | [#459](https://github.com/zynax-io/zynax/issues/459) | In Progress — #538 ✅ #539 ✅ #540 ✅; #476 parent open |
| M5.C Capability Dispatch | [#460](https://github.com/zynax-io/zynax/issues/460) | In Progress — task-broker code merged; agent-registry pending |
| M5.D Security Baseline | [#461](https://github.com/zynax-io/zynax/issues/461) | ✅ Complete (closed) |
| M5.E DX Polish | [#462](https://github.com/zynax-io/zynax/issues/462) | ✅ Complete (closed) |
| Adapter Library | [#377](https://github.com/zynax-io/zynax/issues/377) | In Progress — http ✅; git/ci/llm/langgraph BDD done, impl pending |
| Containerized Make | [#442](https://github.com/zynax-io/zynax/issues/442) | ✅ Complete (closed) |

---

## IMMEDIATE — #554 force-full-pipeline trigger (P1, unblocked by #552 ✅)

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
| [#530](https://github.com/zynax-io/zynax/issues/530) | Update AGENTS.md | ready |
| [#531](https://github.com/zynax-io/zynax/issues/531) | Align service BDD + godog steps | ready |
| [#532](https://github.com/zynax-io/zynax/issues/532) | Handler unit tests | ready |

### agent-registry (#480) — pending implementation

Canvas aligned. Ordered delivery: #526 → #527 → #528 → #481.

| Issue | Step | Status |
|-------|------|--------|
| [#526](https://github.com/zynax-io/zynax/issues/526) | Trim BDD to proto scope | ready (do first) |
| [#527](https://github.com/zynax-io/zynax/issues/527) | Domain layer | blocked on #526 |
| [#528](https://github.com/zynax-io/zynax/issues/528) | gRPC wiring + go.work | blocked on #527 |
| [#481](https://github.com/zynax-io/zynax/issues/481) | Compose wiring | blocked on #528 |

---

## Active Work (Other Tracks)

| Issue | Track | Title | Status |
|-------|-------|-------|--------|
| [#474](https://github.com/zynax-io/zynax/issues/474) | M5.A | Python SDK Agent base class | open — Option A chosen, impl pending |
| [#476](https://github.com/zynax-io/zynax/issues/476) | M5.B | Guard parser (cel-go) | open — #538 ✅ #539 ✅ #540 ✅; pending #476 parent close |
| [#381](https://github.com/zynax-io/zynax/issues/381) | Adapters | git-adapter impl | open (#399 BDD done; #400+ pending, wait for #481) |
| [#382](https://github.com/zynax-io/zynax/issues/382) | Adapters | ci-adapter impl | open (#404 BDD done; #405+ pending, wait for #481) |
| [#383](https://github.com/zynax-io/zynax/issues/383) | Adapters | llm-adapter impl | open (#409 BDD done; #410+ pending, wait for #481) |
| [#384](https://github.com/zynax-io/zynax/issues/384) | Adapters | langgraph-adapter impl | open (#414 BDD done; #415+ pending, wait for #481) |

---

## Known Blockers

- **#551 (Dockerfile.ci-runner) → #552 (switch all jobs)** — highest CI priority. Image must bake in every tool (Go 1.26.3, golangci-lint, govulncheck, buf, Python+uv, ruff, mypy, bandit, pip-audit, gitleaks, zynax-ci, etc.) so no CI step downloads tooling at run time. Reference: `infra/docker/Dockerfile.tools`. Publish to `ghcr.io/zynax-io/zynax/ci-runner:latest` on every Dockerfile change.
- **agent-registry (#480)** — BDD trim (#526) must merge before domain (#527) begins (ADR-016).
- **compose wiring (#481)** — depends on #528 (agent-registry gRPC wiring) landing first.
- **adapter implementations** — wait for #481 (compose wiring) so adapters have a live registry.
- **E2E demo** — blocked on #481 fully wired.
- **v0.4.0 tag** — CHANGELOG promoted; run `git tag -a v0.4.0 -m "M5 Adapter Library" && git push origin v0.4.0` on main to trigger the release workflow and create GitHub Release assets.

---

## Architecture Gaps (open issues to file)

The 2026-05-20 principal architect review identified gaps not yet tracked as issues.
See `docs/milestones/M5-plan.md §Architecture Gaps` for the full list.
Priority gaps to file immediately:

| Gap | Severity |
|-----|----------|
| G1: constant-time bearer compare in api-gateway | High |
| G4: no RetryPolicy on Temporal Activities | Medium |
| G16: background-context goroutines in task-broker | Medium |
| G19: competitive positioning doc (Kagent/Dapr) | High |
| G17: stub services in SERVICE_LIST | Low |

---

## Recently Closed

- **#461 M5.D** — Control Plane Security Baseline: all 5 child issues merged (#482–#486).
- **#462 M5.E** — Developer Experience Polish: all child issues merged (#485–#486).
- **#442** — Fully Containerized Makefile: all 4 child issues merged (#443–#446).
- **#529** — docs(agent-registry): REASONS Canvas for #480.
- **#533** — docs(task-broker): REASONS Canvas for #479.
- **SECURITY.md** — false mTLS/SBOM/cosign claims removed (2026-05-20, part of M5.A truth pass).
