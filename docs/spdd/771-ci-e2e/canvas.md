<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M6.CI-E2E: e2e smoke + upgrade CI gate

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #771
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-10
**Status:** Implemented

**Child issues:** #1070 (H.1) · #1071 (H.2)

---

## R — Requirements

**Problem:** EPIC G (#770) delivered the full e2e harness — an ephemeral kind cluster, the `zynax-umbrella` chart deploy, and assertion scripts for the Temporal happy-path (`e2e-happy.sh`), the Argo happy-path (`e2e-argo.sh`), the failure-path (`e2e-failure.sh`), and Helm upgrade/rollback (`helm-upgrade.sh`). The CI gate that productionizes it, `.github/workflows/e2e-smoke.yml` (PR #1064, #813), currently runs only `cluster-up.sh → helm-upgrade.sh → cluster-down.sh`. Two gaps remain against this epic's intent:

1. The gate proves the stack *installs and upgrades* but never asserts a workflow actually **runs to completion or fails correctly** — the `e2e-happy.sh` / `e2e-failure.sh` assertions exist but are not invoked in CI.
2. Only the default (Temporal) engine is exercised. The pluggable-engine guarantee (ADR-015) is not validated end-to-end — there is no Temporal-vs-Argo matrix, so the Argo path (`e2e-argo.sh`, #811) has no CI coverage.

**Definition of done (observable):**
- The gated `e2e-smoke` job runs the happy-path and failure-path assertions against the live kind cluster and fails the job (with readable logs) when an assertion fails.
- The job runs a 2-leg `engine: [temporal, argo]` matrix (`fail-fast: false`); the Temporal leg runs `e2e-happy.sh`, the Argo leg runs `e2e-argo.sh`, each against an umbrella deployed with engine-selecting values.
- The job stays **advisory** — path-gated on `helm/** · services/** · engine-adapter/** · scripts/e2e/**` plus `workflow_dispatch`, excluded from the branch-protection required-check set, with `cluster-down.sh` in `if: always()` per leg.

---

## E — Entities

> Tier 1 abstractions only.

```
e2e-smoke (GitHub Actions workflow)
   └── job: e2e (strategy.matrix.engine ∈ {temporal, argo}, fail-fast: false)
         ├── cluster-up.sh        → ephemeral kind cluster + zynax-umbrella deploy (engine-selecting values)
         ├── e2e-happy.sh         → [temporal leg] assert workflow.completed + memory KV read
         ├── e2e-argo.sh          → [argo leg]     assert Argo Workflow CR → Succeeded
         ├── e2e-failure.sh       → assert capability timeout → workflow.failed CloudEvent
         ├── helm-upgrade.sh      → assert helm upgrade --atomic + rollback
         └── cluster-down.sh      → idempotent teardown (if: always())

Engine selection  → helm/zynax-umbrella values axis (Temporal | Argo)
Harness scripts   → scripts/e2e/* (delivered by EPIC G #770; unchanged here)
```

No new domain entities, no proto/gRPC contracts. The only new artefacts are CI-workflow structure (a matrix leg) and, if needed, an engine-selecting umbrella values overlay.

---

## A — Approach

**Will do:**
- Extend `.github/workflows/e2e-smoke.yml` to invoke the existing assertion scripts after `cluster-up.sh` (H.1, #1070).
- Add a `strategy.matrix.engine: [temporal, argo]` axis that deploys the umbrella with engine-specific values and runs the engine-appropriate happy-path per leg (H.2, #1071).
- Keep the gate advisory and path-filtered; preserve `concurrency` cancel-in-progress and `if: always()` teardown.

**Won't do:**
- No new harness scripts — all assertions exist from EPIC G #770.
- No promotion to a required PR check (live-cluster e2e is deliberately advisory to avoid flakiness blocking merges — ADR-016 keeps e2e above the required BDD tier).
- No observability/traces/dashboards (M7 scope, explicitly out per the epic).
- No engine-name hardcoding — engine selection flows through umbrella values (ADR-015).

**Governing ADRs:** ADR-015 (pluggable engines — matrix validates the abstraction), ADR-016 (layered testing — e2e is the advisory top tier), ADR-020 (mTLS/cert-manager — cluster bring-up installs cert-manager CRDs), ADR-023 (rebase-merge discipline), ADR-024 (image refs via `images.yaml`; Actions SHA-pinned).

---

## S — Structure

| Path | Change |
|------|--------|
| `.github/workflows/e2e-smoke.yml` | H.1: run `e2e-happy.sh` + `e2e-failure.sh` after `cluster-up.sh`. H.2: add `strategy.matrix.engine: [temporal, argo]`, `fail-fast: false`, per-leg happy-path + engine-selecting values. |
| `helm/zynax-umbrella/values*.yaml` | H.2 (only if an engine toggle is not already exposed): an engine-selecting values overlay for the Argo leg. |
| `scripts/e2e/*` | Unchanged — consumed as-is. |

No service, proto, or gRPC-contract changes. CI-only blast radius.

---

## O — Operations

> Each step = one reviewable PR. Order: H.1 → H.2.

1. **H.1 (#1070)** — ✅ merged. In `e2e-smoke.yml`, run `scripts/e2e/e2e-happy.sh` and `scripts/e2e/e2e-failure.sh` after `cluster-up.sh`; keep `helm-upgrade.sh` and `cluster-down.sh` (`if: always()`); job stays advisory. Verify a forced assertion failure fails the job with readable logs.
2. **H.2 (#1071)** — ✅ delivered-pending-merge (branch `ci/1071-engine-matrix-e2e`). Add `strategy.matrix.engine: [temporal, argo]` (`fail-fast: false`); deploy the umbrella with engine-selecting values per leg; Temporal leg runs `e2e-happy.sh`, Argo leg runs `e2e-argo.sh`; ensure the Argo Workflows controller/CRDs are present for the Argo leg. Verify both legs run independently and tear down per leg. *Delivered as:* `E2E_ENGINE` axis in `cluster-up.sh` (installs the pinned argo-helm `argo-workflows` chart 0.47.5 + the `zynax-ir-interpreter` WorkflowTemplate for the argo leg), `scripts/e2e/values-e2e-argo.yaml` engine-selecting overlay, and additive Argo env plumbing in `helm/zynax-engine-adapter` (gated on `activeEngine=argo` — Temporal renders byte-identical). Per-leg verification = the PR's own gate runs (`e2e smoke (temporal)` / `e2e smoke (argo)`).

---

## N — Norms

> Cross-cutting standards (root AGENTS.md Hard Constraints, infra/AGENTS.md, CI conventions).

- **Commit hygiene:** `Signed-off-by` (DCO) + `Assisted-by: Claude/<model>` trailers; never `Co-Authored-By` for AI.
- **PR type:** `ci:` (workflow change) — one of the seven CI-enforced types; scope `infra`.
- **PR size:** ≤ 200 lines ideal; `.github/workflows/` is excluded from the size budget but keep the diff focused (one story per PR).
- **Merge discipline:** rebase-merge only; branch deleted after merge (ADR-023).
- **Infra-as-code (infra/AGENTS.md):** no manual `kubectl`/console steps — everything reproducible from the workflow + `scripts/e2e/`.
- **Action pinning:** all third-party Actions pinned to a full 40-char commit SHA (existing `e2e-smoke.yml` convention).
- **Image references (ADR-024):** any image refs added to the workflow resolve from `images/images.yaml` pinned digests.

---

## S — Safeguards (second S)

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal email addresses; author attribution only (matches repo convention)
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file (verdict WARN — Draft status only; no Tier 2 / injection / abstraction / authority findings)

### Feature Safeguards
- **Never** hardcode an engine name in the workflow or scripts — engine selection must flow through `helm/zynax-umbrella` values (ADR-015).
- **Never** promote the e2e-smoke job to a branch-protection required check — it stays advisory/gated to keep live-cluster flakiness off the merge path (ADR-016: e2e sits above the required BDD tier).
- **Never** leave a cluster running — `cluster-down.sh` must run in `if: always()` for every matrix leg.
- **Never** run on every PR — preserve the path filter (`helm/** · services/** · engine-adapter/** · scripts/e2e/** · the workflow file`) + `workflow_dispatch`.
- **Never** introduce un-pinned third-party Actions or unmanaged image references (ADR-024).
