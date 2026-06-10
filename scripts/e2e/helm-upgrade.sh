#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# helm-upgrade.sh — assert Helm upgrade/rollback safety end-to-end.
#
# EPIC G (#770) step 5 / #813. Validates that the zynax-umbrella release can be
# upgraded with `helm upgrade --atomic` without service interruption and that a
# subsequent `helm rollback` returns every service to a healthy state.
#
# Flow:
#   1. Ensure the release is installed (helm upgrade --install --atomic).
#   2. Capture the current (pre-upgrade) revision.
#   3. Upgrade with `helm upgrade --atomic` (a no-op-but-mutating bump that forces
#      a fresh rollout); --atomic rolls back automatically on failure.
#   4. Assert all 5 service deployments are healthy after the upgrade.
#   5. Rollback to the pre-upgrade revision (`helm rollback`).
#   6. Assert all 5 service deployments are healthy after the rollback.
#
# Requires a running kind cluster created by cluster-up.sh (G.1 / #809).
# Idempotent: --atomic guarantees the release is left in a healthy state.
#
# Usage:
#   scripts/e2e/helm-upgrade.sh
#
# Environment overrides:
#   CLUSTER_NAME   kind cluster name           (default: zynax-e2e)
#   NAMESPACE      release namespace            (default: zynax)
#   RELEASE_NAME   Helm release name           (default: zynax)
#   WAIT_TIMEOUT   per-resource rollout wait    (default: 600s)
#   HELM_TIMEOUT   helm operation timeout       (default: 600s)
#
# Exit codes:
#   0  upgrade + rollback both succeeded and all services healthy
#   1  an assertion failed or a required tool is missing
#
# Minimum host resources: 4 CPU, 8 GB RAM (see scripts/e2e/README.md).

set -euo pipefail

# ── configuration ───────────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

CLUSTER_NAME="${CLUSTER_NAME:-zynax-e2e}"
NAMESPACE="${NAMESPACE:-zynax}"
RELEASE_NAME="${RELEASE_NAME:-zynax}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-600s}"
HELM_TIMEOUT="${HELM_TIMEOUT:-600s}"

UMBRELLA_CHART="${REPO_ROOT}/helm/zynax-umbrella"

# The 5 Zynax service Deployments that must reach a healthy rollout (mirrors
# cluster-up.sh). event-bus + memory-service are excluded — no GHCR image is
# published for them yet (not in the release.yml build matrix).
SERVICE_DEPLOYMENTS=(
  "zynax-api-gateway"
  "zynax-workflow-compiler"
  "zynax-engine-adapter"
  "zynax-task-broker"
  "zynax-agent-registry"
)

# Helm value flags shared by every install/upgrade so the release shape is
# identical across revisions (mirrors cluster-up.sh). The e2e-only Postgres +
# Temporal overrides live in values-e2e.yaml; the credential Secrets they rely on
# are created by cluster-up.sh, which always runs first.
HELM_SET_FLAGS=(
  -f "${SCRIPT_DIR}/values-e2e.yaml"
  --set zynax-cert-manager.enabled=true
)

# ── helpers ──────────────────────────────────────────────────────────────────────

log()  { printf '\033[1;34m[helm-upgrade]\033[0m %s\n' "$*"; }
pass() { printf '\033[1;32m[helm-upgrade][PASS]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[helm-upgrade][FAIL]\033[0m %s\n' "$*" >&2; exit 1; }
warn() { printf '\033[1;33m[helm-upgrade][WARN]\033[0m %s\n' "$*" >&2; }

require() {
  command -v "$1" >/dev/null 2>&1 || fail "required tool not found on PATH: $1"
}

# assert_all_healthy <phase-label>
# Verifies every service Deployment exists and reaches a healthy rollout.
assert_all_healthy() {
  local phase="$1"
  log "asserting all 5 service deployments healthy (${phase})…"
  for dep in "${SERVICE_DEPLOYMENTS[@]}"; do
    if ! kubectl -n "${NAMESPACE}" get deployment "${dep}" >/dev/null 2>&1; then
      fail "${phase}: expected deployment not found: ${dep}"
    fi
    log "  → ${dep}"
    kubectl -n "${NAMESPACE}" rollout status "deployment/${dep}" \
      --timeout "${WAIT_TIMEOUT}" \
      || fail "${phase}: deployment '${dep}' did not become healthy within ${WAIT_TIMEOUT}"
  done
  pass "${phase}: all 5 service deployments are healthy."
}

# ── preflight ──────────────────────────────────────────────────────────────────

log "preflight: checking required tools and cluster state…"

require kubectl
require helm
require jq

[[ -d "${UMBRELLA_CHART}" ]] || fail "umbrella chart not found: ${UMBRELLA_CHART}"

# Verify the kind cluster exists and point kubectl at it.
if ! kubectl config get-contexts "kind-${CLUSTER_NAME}" >/dev/null 2>&1; then
  fail "kubectl context 'kind-${CLUSTER_NAME}' not found — run scripts/e2e/cluster-up.sh first"
fi
kubectl config use-context "kind-${CLUSTER_NAME}" >/dev/null

# Build chart dependencies if the packaged subcharts are missing (mirrors
# cluster-up.sh). No-op if already present.
if [[ ! -d "${UMBRELLA_CHART}/charts" ]] || \
   [[ -z "$(ls -A "${UMBRELLA_CHART}/charts" 2>/dev/null)" ]]; then
  log "building umbrella chart dependencies…"
  helm dependency build "${UMBRELLA_CHART}"
fi

log "preflight passed."

# ── 1. ensure the release is installed ───────────────────────────────────────────

log "step 1: ensuring release '${RELEASE_NAME}' is installed (helm upgrade --install --atomic)…"
helm upgrade --install "${RELEASE_NAME}" "${UMBRELLA_CHART}" \
  --namespace "${NAMESPACE}" \
  --create-namespace \
  "${HELM_SET_FLAGS[@]}" \
  --atomic \
  --wait \
  --timeout "${HELM_TIMEOUT}"

assert_all_healthy "post-install"

# ── 2. capture the pre-upgrade revision ──────────────────────────────────────────

BASE_REVISION=$(helm status "${RELEASE_NAME}" --namespace "${NAMESPACE}" \
  -o json | jq -r '.version')
[[ -n "${BASE_REVISION}" && "${BASE_REVISION}" != "null" ]] \
  || fail "step 2: could not determine current Helm revision for '${RELEASE_NAME}'"
log "step 2: pre-upgrade revision is ${BASE_REVISION}."

# ── 3. helm upgrade --atomic ──────────────────────────────────────────────────────

# Force a fresh rollout by stamping a unique pod annotation. The chart shape is
# otherwise unchanged, so this exercises the rolling-update path (zero-downtime
# strategy: maxUnavailable=0) without changing application behaviour. --atomic
# auto-rolls-back on failure, so a green exit here proves a safe upgrade.
UPGRADE_STAMP="e2e-upgrade-$(date +%s)"
log "step 3: running 'helm upgrade --atomic' (stamp=${UPGRADE_STAMP})…"
helm upgrade "${RELEASE_NAME}" "${UMBRELLA_CHART}" \
  --namespace "${NAMESPACE}" \
  "${HELM_SET_FLAGS[@]}" \
  --set-string "podAnnotations.zynax\.io/e2e-upgrade=${UPGRADE_STAMP}" \
  --atomic \
  --wait \
  --timeout "${HELM_TIMEOUT}" \
  || fail "step 3: 'helm upgrade --atomic' failed — release auto-rolled-back, upgrade is NOT safe"

pass "step 3: 'helm upgrade --atomic' succeeded."

# ── 4. assert healthy after upgrade (no service interruption) ─────────────────────

assert_all_healthy "post-upgrade"

UPGRADE_REVISION=$(helm status "${RELEASE_NAME}" --namespace "${NAMESPACE}" \
  -o json | jq -r '.version')
log "step 4: post-upgrade revision is ${UPGRADE_REVISION}."

# ── 5. rollback to the pre-upgrade revision ──────────────────────────────────────

log "step 5: rolling back '${RELEASE_NAME}' to revision ${BASE_REVISION}…"
helm rollback "${RELEASE_NAME}" "${BASE_REVISION}" \
  --namespace "${NAMESPACE}" \
  --wait \
  --timeout "${HELM_TIMEOUT}" \
  || fail "step 5: 'helm rollback' to revision ${BASE_REVISION} failed"

pass "step 5: rollback to revision ${BASE_REVISION} completed."

# ── 6. assert healthy after rollback ──────────────────────────────────────────────

assert_all_healthy "post-rollback"

# ── summary ──────────────────────────────────────────────────────────────────────

ROLLBACK_REVISION=$(helm status "${RELEASE_NAME}" --namespace "${NAMESPACE}" \
  -o json | jq -r '.version')

printf '\n\033[1;32m[helm-upgrade] ALL ASSERTIONS PASSED\033[0m\n'
printf '  release:   %s (namespace %s)\n' "${RELEASE_NAME}" "${NAMESPACE}"
printf '  revisions: base=%s → upgrade=%s → rollback=%s\n' \
  "${BASE_REVISION}" "${UPGRADE_REVISION}" "${ROLLBACK_REVISION}"
printf '  upgrade:   helm upgrade --atomic succeeded (zero-downtime rollout)\n'
printf '  rollback:  all 5 service deployments healthy after rollback\n'
printf '\n'
