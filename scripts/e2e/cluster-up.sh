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

# The 7 Zynax service Deployments that must reach a healthy rollout. event-bus
# and memory-service run as placeholder images until EPIC I (#772) / J (#773).
SERVICE_DEPLOYMENTS=(
  "${RELEASE_NAME}-zynax-api-gateway"
  "${RELEASE_NAME}-zynax-workflow-compiler"
  "${RELEASE_NAME}-zynax-engine-adapter"
  "${RELEASE_NAME}-zynax-task-broker"
  "${RELEASE_NAME}-zynax-agent-registry"
  "${RELEASE_NAME}-zynax-event-bus"
  "${RELEASE_NAME}-zynax-memory-service"
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

# ── 3. deploy the full Zynax stack via the umbrella chart ────────────────────────

# Build chart dependencies if the packaged subcharts are missing (e.g. a fresh
# checkout where `helm dependency build` has not yet run). No-op if present.
if [[ ! -d "${UMBRELLA_CHART}/charts" ]] || \
   [[ -z "$(ls -A "${UMBRELLA_CHART}/charts" 2>/dev/null)" ]]; then
  log "building umbrella chart dependencies…"
  helm dependency build "${UMBRELLA_CHART}"
fi

log "deploying zynax-umbrella as release '${RELEASE_NAME}' in namespace '${NAMESPACE}'…"
# event-bus + memory-service are enabled with placeholder images so all 7
# service pods schedule; real implementations land via EPIC I (#772) / J (#773).
helm upgrade --install "${RELEASE_NAME}" "${UMBRELLA_CHART}" \
  --namespace "${NAMESPACE}" \
  --create-namespace \
  --set zynax-event-bus.enabled=true \
  --set zynax-memory-service.enabled=true \
  --set zynax-cert-manager.enabled=true \
  --wait \
  --timeout "${WAIT_TIMEOUT}"

# ── 4. wait for all 7 service deployments to become healthy ──────────────────────

log "waiting for all 7 service deployments to roll out…"
for dep in "${SERVICE_DEPLOYMENTS[@]}"; do
  if ! kubectl -n "${NAMESPACE}" get deployment "${dep}" >/dev/null 2>&1; then
    die "expected deployment not found: ${dep} (umbrella values out of sync?)"
  fi
  log "  → ${dep}"
  kubectl -n "${NAMESPACE}" rollout status "deployment/${dep}" \
    --timeout "${WAIT_TIMEOUT}"
done

log "all 7 service deployments are healthy."
kubectl -n "${NAMESPACE}" get pods -o wide

log "cluster '${CLUSTER_NAME}' is up. api-gateway REST is reachable on host port 8080."
log "tear down with: scripts/e2e/cluster-down.sh"
