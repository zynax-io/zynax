<!-- SPDX-License-Identifier: Apache-2.0 -->

# Zynax e2e harness

Reproducible end-to-end test harness for the full Zynax platform. These scripts
spin up an ephemeral [kind](https://kind.sigs.k8s.io/) (Kubernetes IN Docker)
cluster and deploy the complete stack via the `helm/zynax-umbrella` chart, so
that e2e assertions run against a real Kubernetes cluster rather than mocks.

Part of EPIC G (#770). This directory delivers cluster bootstrap + teardown
(**step 1, #809**) plus the assertion scripts: happy-path (**#810**), Argo-path
(**#811**), and failure-path (**#812**). The Helm upgrade/rollback script and
gated CI job (**#813**) land in the final step.

## Minimum resource requirements

The full stack — 7 Zynax services plus NATS JetStream, Postgres 16, and
Temporal — needs real headroom. The cluster will evict pods under memory
pressure below these limits:

| Resource | Minimum |
|----------|---------|
| CPU      | **4 cores** |
| RAM      | **8 GB** |
| Disk     | 20 GB free |

On CI this maps to the `ubuntu-latest` runner (4 vCPU / 16 GB), with Docker
configured for Docker-in-Docker. Locally, ensure Docker Desktop is allocated at
least 4 CPU and 8 GB RAM.

## Prerequisites

| Tool      | Notes |
|-----------|-------|
| `docker`  | running daemon (Docker Desktop or DinD) |
| `kind`    | cluster bootstrap |
| `kubectl` | cluster interaction |
| `helm`    | chart deployment (v3.14+) |

`cluster-up.sh` installs upstream **cert-manager** into the cluster itself —
the `zynax-cert-manager` subchart only creates `Certificate` / `ClusterIssuer`
resources and assumes cert-manager CRDs already exist (ADR-020).

## Usage

```bash
# Bring up the cluster and deploy the full stack (idempotent).
scripts/e2e/cluster-up.sh

# Run e2e assertions against the live cluster.
scripts/e2e/e2e-happy.sh      # Temporal happy-path: workflow.completed + memory KV
scripts/e2e/e2e-argo.sh       # Argo engine happy-path: Workflow CR reaches Succeeded
scripts/e2e/e2e-failure.sh    # failure-path: capability timeout → workflow.failed
scripts/e2e/helm-upgrade.sh   # Helm upgrade --atomic + rollback (#813)

# Tear the cluster down (idempotent).
scripts/e2e/cluster-down.sh
```

### Failure-path assertion (`e2e-failure.sh`, #812)

`e2e-failure.sh` submits a workflow whose initial state invokes an
**unreachable capability** (one no deployed agent serves). It then asserts that:

1. the workflow reaches a terminal `failed` state (reaching `succeeded` is the
   test's failure condition);
2. the failure is a capability dispatch timeout, bounded by
   `ZYNAX_CAPABILITY_TIMEOUT`;
3. the `zynax.workflow.failed` CloudEvent is consumed off the
   `ZYNAX_WORKFLOW` NATS JetStream stream.

The workflow fixture is generated at runtime (and removed on exit) rather than
committed under `spec/workflows/examples/`, to avoid publishing an intentionally
broken workflow as a reference. The script exits `0` only when the failure path
behaves as expected.

Both scripts are idempotent: `cluster-up.sh` reuses an existing cluster and
performs a `helm upgrade --install`; `cluster-down.sh` succeeds even if no
cluster is present.

### Configuration

Override defaults via environment variables (see each script header for the
full list):

| Variable               | Default            | Purpose |
|------------------------|--------------------|---------|
| `CLUSTER_NAME`         | `zynax-e2e`        | kind cluster name |
| `NAMESPACE`            | `zynax`            | release namespace |
| `RELEASE_NAME`         | `zynax`            | Helm release name |
| `CERT_MANAGER_VERSION` | `v1.14.5`          | cert-manager chart version |
| `KIND_NODE_IMAGE`      | `kindest/node:v1.29.2` | kind node image (digest-pinnable in CI) |
| `WAIT_TIMEOUT`         | `600s`             | per-resource rollout wait |
| `ZYNAX_CAPABILITY_TIMEOUT` | `30s`          | capability dispatch timeout asserted by `e2e-failure.sh` |

## What `cluster-up.sh` does

1. Creates a 3-node kind cluster (1 control-plane + 2 workers) from
   [`kind-config.yaml`](kind-config.yaml), exposing the api-gateway REST port on
   host `:8080`.
2. Installs cert-manager (CRDs + controllers).
3. Deploys `zynax-umbrella` with `event-bus` and `memory-service` enabled using
   **placeholder images** (real implementations: EPIC I #772 / J #773).
4. Waits for all **7 service Deployments** to reach a healthy rollout with their
   startup / liveness / readiness probes passing.

## What `helm-upgrade.sh` does

Validates that the platform can be safely upgraded in production (G.5 / #813):

1. Ensures the release is installed via `helm upgrade --install --atomic`.
2. Captures the current Helm revision.
3. Runs `helm upgrade --atomic` (forces a fresh, zero-downtime rolling update);
   `--atomic` auto-rolls-back the release on any failure.
4. Asserts all **7 service Deployments** are healthy after the upgrade.
5. Runs `helm rollback` to the pre-upgrade revision.
6. Asserts all 7 service Deployments are healthy again after the rollback.

A green exit proves `helm upgrade --atomic` succeeds without service interruption
and that rollback restores every service to a healthy state.

## CI

Live cluster bring-up is exercised by the gated `e2e-smoke.yml` workflow added
in #770 step 5 (#813). It runs `cluster-up.sh` then `helm-upgrade.sh`, triggers
only on changes to `helm/**`, `services/**`, or `engine-adapter/**` (plus manual
`workflow_dispatch`), and is **not** a required gate on every PR. All Actions are
SHA-pinned. The kind cluster is ephemeral per job run; kubeconfigs are never
committed.
