#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# cluster-up.sh — bring up a kind cluster and deploy the full Zynax stack.
#
# EPIC G (#770) step 1 / #809. Creates a reproducible, ephemeral Kubernetes
# cluster via kind and installs the `zynax-umbrella` Helm chart so that the
# e2e assertion scripts (#810–#813) run against a real cluster.
#
# Idempotent: re-running against an existing cluster reuses it and performs a
# `helm upgrade --install`, so the script is safe to invoke repeatedly.
#
# Usage:
#   scripts/e2e/cluster-up.sh
#
# Environment overrides:
#   CLUSTER_NAME          kind cluster name           (default: zynax-e2e)
#   NAMESPACE             release namespace           (default: zynax)
#   RELEASE_NAME          Helm release name           (default: zynax)
#   CERT_MANAGER_VERSION  cert-manager release tag    (default: v1.14.5)
#   KIND_NODE_IMAGE       kind node image (digest-pinnable)
#   WAIT_TIMEOUT          per-resource rollout wait   (default: 600s)
#   E2E_IMAGE_TAG         service image tag override — set by e2e-smoke.yml to
#                         pr-<head-sha> so the cluster runs the exact staging
#                         images built pre-merge (#1118 / ADR-027). Unset =
#                         values-e2e.yaml default (:main lane).
#   E2E_IMAGE_PREFIX      registry prefix for E2E_IMAGE_TAG
#                         (default: ghcr.io/zynax-io/zynax/staging)
#
# Minimum host resources: 4 CPU, 8 GB RAM (see scripts/e2e/README.md).

set -euo pipefail

# ── configuration ───────────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

CLUSTER_NAME="${CLUSTER_NAME:-zynax-e2e}"
NAMESPACE="${NAMESPACE:-zynax}"
RELEASE_NAME="${RELEASE_NAME:-zynax}"
CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.14.5}"
# kind node image — pin to a digest in CI for reproducibility. The default tag
# tracks the kind release that the harness is validated against.
KIND_NODE_IMAGE="${KIND_NODE_IMAGE:-kindest/node:v1.29.2}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-600s}"

KIND_CONFIG="${SCRIPT_DIR}/kind-config.yaml"
UMBRELLA_CHART="${REPO_ROOT}/helm/zynax-umbrella"

# The Zynax service Deployments that must reach a healthy rollout. Only the 5
# services in the release.yml build matrix have a published image; event-bus and
# memory-service are not built yet (no GHCR image), so they are excluded from the
# e2e deploy (disabled in values-e2e.yaml) and from this assertion list. Re-add
# them once their images ship.
# Deployment names are pinned via fullnameOverride in values-e2e.yaml to
# `zynax-<svc>` (so the umbrella's inter-service addresses resolve), not the
# release-prefixed `zynax-zynax-<svc>` default.
SERVICE_DEPLOYMENTS=(
  "zynax-api-gateway"
  "zynax-workflow-compiler"
  "zynax-engine-adapter"
  "zynax-task-broker"
  "zynax-agent-registry"
)

# ── helpers ──────────────────────────────────────────────────────────────────────

log()  { printf '\033[1;34m[cluster-up]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[cluster-up]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[cluster-up]\033[0m %s\n' "$*" >&2; exit 1; }

require() {
  command -v "$1" >/dev/null 2>&1 || die "required tool not found on PATH: $1"
}

# ── preflight ──────────────────────────────────────────────────────────────────

require kind
require kubectl
require helm
require openssl

[[ -f "${KIND_CONFIG}" ]]   || die "kind config not found: ${KIND_CONFIG}"
[[ -d "${UMBRELLA_CHART}" ]] || die "umbrella chart not found: ${UMBRELLA_CHART}"

# ── 1. create (or reuse) the kind cluster ────────────────────────────────────────

if kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  log "kind cluster '${CLUSTER_NAME}' already exists — reusing (idempotent)."
else
  log "creating kind cluster '${CLUSTER_NAME}' (node image: ${KIND_NODE_IMAGE})…"
  kind create cluster \
    --name "${CLUSTER_NAME}" \
    --image "${KIND_NODE_IMAGE}" \
    --config "${KIND_CONFIG}" \
    --wait "${WAIT_TIMEOUT}"
fi

# Point kubectl/helm at the cluster regardless of how it was created.
kubectl config use-context "kind-${CLUSTER_NAME}" >/dev/null

# ── 2. install cert-manager (CRDs + controllers) ─────────────────────────────────

# The umbrella chart's zynax-cert-manager subchart creates Certificate /
# ClusterIssuer resources but does NOT install cert-manager itself (ADR-020).
# Install upstream cert-manager so those CRDs exist; idempotent via upgrade.
log "installing cert-manager ${CERT_MANAGER_VERSION}…"
helm upgrade --install cert-manager cert-manager \
  --repo https://charts.jetstack.io \
  --namespace cert-manager \
  --create-namespace \
  --version "${CERT_MANAGER_VERSION}" \
  --set installCRDs=true \
  --wait \
  --timeout "${WAIT_TIMEOUT}"

# ── 2.5 provision Postgres + Temporal credentials ───────────────────────────────

# The Temporal schema Job and server connect to the bundled Postgres using the
# `temporal-db` Secret, and the bitnami Postgres subchart reads its password from
# the `zynax-postgres-creds` Secret (values-e2e.yaml sets auth.existingSecret).
# Pre-create both with a single shared password so they always match. The schema
# Job runs `temporal-sql-tool create-database`, which only the Postgres superuser
# can do — values-e2e.yaml points Temporal at user `postgres`, so temporal-db
# carries the superuser password.
#
# Idempotent: the password is generated once and reused on re-runs so it never
# diverges from an already-initialised Postgres data volume.
log "provisioning Postgres + Temporal credentials in namespace '${NAMESPACE}'…"
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

if kubectl -n "${NAMESPACE}" get secret zynax-postgres-creds >/dev/null 2>&1; then
  log "  reusing existing zynax-postgres-creds Secret (idempotent)."
  PG_PASSWORD="$(kubectl -n "${NAMESPACE}" get secret zynax-postgres-creds \
    -o jsonpath='{.data.postgres-password}' | base64 -d)"
else
  PG_PASSWORD="$(openssl rand -hex 16)"
  kubectl -n "${NAMESPACE}" create secret generic zynax-postgres-creds \
    --from-literal=postgres-password="${PG_PASSWORD}" \
    --from-literal=password="${PG_PASSWORD}"
fi

if ! kubectl -n "${NAMESPACE}" get secret temporal-db >/dev/null 2>&1; then
  kubectl -n "${NAMESPACE}" create secret generic temporal-db \
    --from-literal=password="${PG_PASSWORD}"
fi

# api-gateway refuses to start without ZYNAX_GW_API_KEY (it reads the `api-key`
# key from the Secret named by api-gateway's apiKeySecretName, default
# `zynax-gw-api-key`). Provision a throwaway key for the smoke cluster.
if ! kubectl -n "${NAMESPACE}" get secret zynax-gw-api-key >/dev/null 2>&1; then
  kubectl -n "${NAMESPACE}" create secret generic zynax-gw-api-key \
    --from-literal=api-key="$(openssl rand -hex 16)"
fi

# ── 3. deploy the full Zynax stack via the umbrella chart ────────────────────────

# Build chart dependencies if the packaged subcharts are missing (e.g. a fresh
# checkout where `helm dependency build` has not yet run). No-op if present.
if [[ ! -d "${UMBRELLA_CHART}/charts" ]] || \
   [[ -z "$(ls -A "${UMBRELLA_CHART}/charts" 2>/dev/null)" ]]; then
  log "building umbrella chart dependencies…"
  helm dependency build "${UMBRELLA_CHART}"
fi

log "deploying zynax-umbrella as release '${RELEASE_NAME}' in namespace '${NAMESPACE}'…"
# values-e2e.yaml carries the e2e-only overrides (shared with helm-upgrade.sh so
# the release shape is identical across revisions): service image tags pinned to
# `main`, event-bus/memory-service disabled (no image yet), and the Postgres /
# Temporal credential wiring.
# E2E_IMAGE_TAG (set by e2e-smoke.yml for docker-touching PRs) repoints the 5
# deployed services at the pre-merge staging lane — helm-upgrade.sh applies the
# same overrides so the release shape stays identical across revisions.
IMAGE_OVERRIDES=()
if [[ -n "${E2E_IMAGE_TAG:-}" ]]; then
  E2E_IMAGE_PREFIX="${E2E_IMAGE_PREFIX:-ghcr.io/zynax-io/zynax/staging}"
  log "service image lane override: ${E2E_IMAGE_PREFIX}/<svc>:${E2E_IMAGE_TAG}"
  for svc in api-gateway workflow-compiler engine-adapter task-broker agent-registry; do
    IMAGE_OVERRIDES+=(
      --set "zynax-${svc}.image.repository=${E2E_IMAGE_PREFIX}/${svc}"
      --set "zynax-${svc}.image.tag=${E2E_IMAGE_TAG}"
    )
  done
fi
# NOTE: no --wait here. engine-adapter cannot become Ready until the Temporal
# `default` namespace is registered (step 3.5), and a Helm --wait would deadlock
# waiting on it. Readiness is asserted explicitly by the rollout loop in step 4.
helm upgrade --install "${RELEASE_NAME}" "${UMBRELLA_CHART}" \
  --namespace "${NAMESPACE}" \
  --create-namespace \
  -f "${SCRIPT_DIR}/values-e2e.yaml" \
  --set zynax-cert-manager.enabled=true \
  "${IMAGE_OVERRIDES[@]}"

# ── 3.5 register the Temporal 'default' namespace ────────────────────────────────

# The temporalio/temporal chart does NOT auto-register a namespace, but
# engine-adapter connects to namespace 'default' on startup and crash-loops with
# "Namespace default is not found" until it exists. Wait for the Temporal frontend
# to roll out, then register it via the admintools pod. Idempotent: skip if it is
# already present (e.g. a re-run against a reused cluster).
log "waiting for Temporal frontend, then registering the 'default' namespace…"
kubectl -n "${NAMESPACE}" rollout status \
  "deployment/${RELEASE_NAME}-temporal-frontend" --timeout "${WAIT_TIMEOUT}"

ADMINTOOLS="deployment/${RELEASE_NAME}-temporal-admintools"
namespace_ready=""
for _ in $(seq 1 30); do
  if kubectl -n "${NAMESPACE}" exec "${ADMINTOOLS}" -- \
       temporal operator namespace describe default >/dev/null 2>&1; then
    namespace_ready="yes"
    break
  fi
  kubectl -n "${NAMESPACE}" exec "${ADMINTOOLS}" -- \
    temporal operator namespace create default >/dev/null 2>&1 || true
  sleep 5
done
[[ -n "${namespace_ready}" ]] || die "Temporal 'default' namespace did not register"
log "Temporal 'default' namespace is registered."

# ── 4. wait for all 5 service deployments to become healthy ──────────────────────

log "waiting for all 5 service deployments to roll out…"
for dep in "${SERVICE_DEPLOYMENTS[@]}"; do
  if ! kubectl -n "${NAMESPACE}" get deployment "${dep}" >/dev/null 2>&1; then
    die "expected deployment not found: ${dep} (umbrella values out of sync?)"
  fi
  log "  → ${dep}"
  kubectl -n "${NAMESPACE}" rollout status "deployment/${dep}" \
    --timeout "${WAIT_TIMEOUT}"
done

log "all 5 service deployments are healthy."

# ── 5. deploy the echo capability worker (#1088) ────────────────────────────────

# The umbrella deploys only the platform services — no capability provider — so
# dispatched tasks are never claimed. Deploy a minimal langgraph-adapter that
# registers the "echo" capability with agent-registry and completes its tasks,
# letting spec/workflows/examples/e2e-demo.yaml reach a terminal succeeded state.
# agent-registry is healthy by now (step 4), so the worker registers on startup.
log "deploying echo capability worker…"
kubectl -n "${NAMESPACE}" apply -f "${SCRIPT_DIR}/manifests/echo-worker.yaml"
kubectl -n "${NAMESPACE}" rollout status deployment/echo-worker --timeout "${WAIT_TIMEOUT}"
log "echo-worker is healthy and registered."

kubectl -n "${NAMESPACE}" get pods -o wide

log "cluster '${CLUSTER_NAME}' is up. api-gateway REST is reachable on host port 8080."
log "tear down with: scripts/e2e/cluster-down.sh"
