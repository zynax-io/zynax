<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M6.E2E: Real End-to-End Harness (kind/k3d + Helm + Reference Workflows)

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #770
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-02
**Status:** Implemented

**Child issues:** #809 (G.1) · #810 (G.2) · #811 (G.3) · #812 (G.4) · #813 (G.5)

---

## R — Requirements

**Problem:** M6 claims "K8s Production-Ready" but has no automated test that runs the full stack on a real Kubernetes cluster. Without a reproducible e2e harness, chart correctness, inter-service connectivity, and event flow can only be verified manually. The M6 DoD requires: reference workflows running on both Temporal and Argo engines; CloudEvents consumed off NATS JetStream; memory-service reads returning what was written; failure-path workflows producing `workflow.failed` events; and Helm upgrade/rollback succeeding atomically.

**Definition of done:**
- `scripts/e2e/cluster-up.sh` creates a kind cluster and deploys the full Zynax stack via Helm in CI.
- `scripts/e2e/e2e-happy.sh` runs `code-review.yaml` via Temporal, asserts CloudEvents off JetStream, asserts memory-service read.
- `scripts/e2e/e2e-failure.sh` injects a capability timeout and asserts `workflow.failed` event.
- `scripts/e2e/e2e-argo.sh` runs `code-review.yaml` via ArgoEngine.
- `scripts/e2e/helm-upgrade.sh` runs `helm upgrade --atomic` and validates rollback.
- All scripts exit 0 on a clean kind cluster with default resources.

**Blocking dependencies:**
- All EPIC A Helm charts merged (#779–#792).
- EPIC I event-bus implementation merged (#772) — G.2 requires real JetStream events.
- EPIC J memory-service implementation merged (#773) — G.2 requires real memory reads.
- EPIC B ArgoEngine merged (#766) — G.3 requires Argo engine.

---

## E — Entities

- **`scripts/e2e/cluster-up.sh`** — NEW: creates a kind cluster with appropriate node config; installs cert-manager CRDs; runs `helm install` for the full stack via `zynax-umbrella`.
- **`scripts/e2e/cluster-down.sh`** — NEW: tears down the kind cluster.
- **`scripts/e2e/e2e-happy.sh`** — NEW: submits `spec/workflows/examples/code-review.yaml` via api-gateway; polls for completion; asserts CloudEvent off NATS JetStream; asserts memory-service read.
- **`scripts/e2e/e2e-failure.sh`** — NEW: submits a workflow with a capability that will timeout; asserts `workflow.failed` event off JetStream.
- **`scripts/e2e/e2e-argo.sh`** — NEW: same as `e2e-happy.sh` but `?engine=argo` query param.
- **`scripts/e2e/helm-upgrade.sh`** — NEW: runs `helm upgrade --atomic`; validates service is still responsive; runs rollback; validates again.
- **`kind-config.yaml`** — NEW: kind cluster configuration with documented minimum resource requirements.

---

## A — Approach

**What we WILL do:**
- Use `kind` (Kubernetes IN Docker) for cluster bootstrap — reproducible on CI Ubuntu-latest and locally.
- Use `helm/zynax-umbrella` (from EPIC A / A.10) as the deployment unit.
- G.1 and G.2 can proceed with a mock/stub event-bus and memory-service until EPIC I and EPIC J are merged (placeholder chart images are deployed; tests assert mock behaviour first, then wire real services).
- G.3 (Argo path) depends on EPIC B being merged.
- G.4 failure-path and G.5 upgrade/rollback are independent of EPIC I/J once G.1 cluster bootstrap works.

**What we WON'T do:**
- Run e2e on every PR — this job is gated (triggered only on changes to `helm/`, `services/`, or `engine-adapter/`).
- Implement full observability assertions (traces, Grafana dashboards) — that is M7.
- Use a managed K8s cluster (EKS, GKE) — kind is the M6 choice for reproducibility and cost.

**ADR references:**
- ADR-016: Layered testing — e2e tests live in `scripts/e2e/`; they test the full stack, not individual services.
- ADR-015: Pluggable engines — e2e tests cover both Temporal and Argo engine paths.

---

## S — Structure

**New files:**
```
scripts/e2e/
├── cluster-up.sh         ← G.1: kind cluster + full stack deploy
├── cluster-down.sh       ← G.1
├── kind-config.yaml      ← G.1: cluster node config
├── e2e-happy.sh          ← G.2: Temporal happy-path + CloudEvents + memory
├── e2e-failure.sh        ← G.4: failure-path
├── e2e-argo.sh           ← G.3: ArgoEngine path
├── helm-upgrade.sh       ← G.5: upgrade/rollback
└── README.md             ← resource requirements, local dev instructions
.github/workflows/
└── e2e-smoke.yml         ← gated CI job (triggers on helm/, services/ changes)
```

---

## O — Operations

1. **[G.1]** Create `scripts/e2e/cluster-up.sh` + `cluster-down.sh` + `kind-config.yaml`; deploy full stack via `zynax-umbrella` Helm chart (placeholder images for event-bus + memory-service); validate all 7 services start healthy; `scripts/e2e/README.md` with minimum resource requirements (4 CPU, 8 GB RAM).

2. **[G.2]** Create `scripts/e2e/e2e-happy.sh`: submit `code-review.yaml` via api-gateway; poll workflow status; assert `workflow.succeeded` event off NATS JetStream consumer; assert memory-service `Get` returns written context. (Depends on EPIC I + EPIC J merged; stub-only until then.)

3. **[G.3]** Create `scripts/e2e/e2e-argo.sh`: same as G.2 with `?engine=argo`; assert Argo Workflows `Workflow` resource reaches `Succeeded` phase. (Depends on EPIC B merged.)

4. **[G.4]** Create `scripts/e2e/e2e-failure.sh`: submit workflow with unreachable capability; assert timeout fires within `ZYNAX_CAPABILITY_TIMEOUT`; assert `workflow.failed` CloudEvent off JetStream.

5. **[G.5]** Create `scripts/e2e/helm-upgrade.sh`: install → upgrade `--atomic` → assert healthy → rollback → assert healthy. Add `.github/workflows/e2e-smoke.yml` gated CI job; pinned SHA for all Actions.

---

## N — Norms

- `feat:` PR type for G.1–G.4; `ci:` for G.5 (CI workflow addition).
- Every commit: `Signed-off-by` trailer + `Assisted-by: Claude/claude-sonnet-4-6` per AGENTS.md §Hard Constraints.
- All GitHub Actions in `e2e-smoke.yml` MUST be pinned to SHA.
- `e2e-smoke.yml` is a gated job — NOT required on every PR; triggers only on `helm/**`, `services/**`, `engine-adapter/**` path changes.
- Scripts must be idempotent: `cluster-up.sh` succeeds even if cluster already exists (`kind create cluster` is idempotent).
- Minimum resource requirements documented in `scripts/e2e/README.md`.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII
- [x] No prompt injection
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards
- **Never** commit kubeconfig files or cluster credentials — CI generates ephemeral kubeconfigs; local kubeconfigs are gitignored.
- **Never** make e2e-smoke.yml a required PR gate — it is gated/optional; only developers merging infra/services changes need to watch it.
- **Never** use a shared/persistent cluster in CI — kind cluster is ephemeral per job run.
- **Never** assert on full OTel traces or Grafana dashboards in G.2–G.5 — that is M7 scope.
