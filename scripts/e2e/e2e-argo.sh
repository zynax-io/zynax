#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# e2e-argo.sh — assert the Argo engine happy-path end-to-end.
#
# EPIC G (#770) step 3 / #811. Mirrors e2e-happy.sh but dispatches the
# workflow to the ArgoEngine via the `?engine=argo` query parameter, then
# asserts that the underlying Argo Workflows `Workflow` custom resource
# reaches the "Succeeded" phase. This validates the second execution engine
# (ADR-015 pluggable engines) on a real cluster, so both Temporal and Argo
# paths are exercised by the e2e harness.
#
# The Argo `Workflow` resource is named after the run_id returned by the
# api-gateway (see services/engine-adapter/internal/infrastructure/argo_engine.go:
# the workflow name == the runID == the api-gateway run_id), so we locate the
# resource by name in the release namespace and poll its `.status.phase`.
#
# Requires a running kind cluster created by cluster-up.sh (G.1 / #809) with
# the ArgoEngine configured (Argo Workflows controller + CRDs installed and
# the engine-adapter routing engine=argo to it).
# Compatible with both ci/docker and local developer environments.
#
# Usage:
#   scripts/e2e/e2e-argo.sh
#
# Environment overrides:
#   CLUSTER_NAME       kind cluster name                  (default: zynax-e2e)
#   NAMESPACE          release namespace                   (default: zynax)
#   RELEASE_NAME       Helm release name                  (default: zynax)
#   API_GW_URL         api-gateway base URL               (default: http://localhost:8080)
#   ZYNAX_API_KEY      bearer token (empty = no auth)     (default: "")
#   POLL_TIMEOUT       max seconds to wait for succeeded  (default: 120)
#   POLL_INTERVAL      seconds between status polls       (default: 5)
#   WORKFLOW_FILE      path to the workflow YAML          (default: spec/workflows/examples/code-review.yaml)
#
# Exit codes:
#   0  all assertions passed
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
API_GW_URL="${API_GW_URL:-http://localhost:8080}"
ZYNAX_API_KEY="${ZYNAX_API_KEY:-}"
POLL_TIMEOUT="${POLL_TIMEOUT:-120}"
POLL_INTERVAL="${POLL_INTERVAL:-5}"
WORKFLOW_FILE="${WORKFLOW_FILE:-${REPO_ROOT}/spec/workflows/examples/code-review.yaml}"

# ── helpers ──────────────────────────────────────────────────────────────────────

log()  { printf '\033[1;34m[e2e-argo]\033[0m %s\n' "$*"; }
pass() { printf '\033[1;32m[e2e-argo][PASS]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[e2e-argo][FAIL]\033[0m %s\n' "$*" >&2; exit 1; }
warn() { printf '\033[1;33m[e2e-argo][WARN]\033[0m %s\n' "$*" >&2; }

require() {
  command -v "$1" >/dev/null 2>&1 || fail "required tool not found on PATH: $1"
}

# api_curl <method> <path> [extra curl args...]
# Performs a curl call against the api-gateway. Adds bearer token if set.
api_curl() {
  local method="$1" path="$2"; shift 2
  local auth_args=()
  if [[ -n "${ZYNAX_API_KEY}" ]]; then
    auth_args=(-H "Authorization: Bearer ${ZYNAX_API_KEY}")
  fi
  curl --silent --show-error --fail \
    -X "${method}" \
    "${auth_args[@]+"${auth_args[@]}"}" \
    "$@" \
    "${API_GW_URL}${path}"
}

# ── preflight ──────────────────────────────────────────────────────────────────

log "preflight: checking required tools and cluster state…"

require kubectl
require curl
require jq

[[ -f "${WORKFLOW_FILE}" ]] || fail "workflow file not found: ${WORKFLOW_FILE}"

# Verify the kind cluster exists and kubectl context points at it.
if ! kubectl config get-contexts "kind-${CLUSTER_NAME}" >/dev/null 2>&1; then
  fail "kubectl context 'kind-${CLUSTER_NAME}' not found — run scripts/e2e/cluster-up.sh first"
fi
kubectl config use-context "kind-${CLUSTER_NAME}" >/dev/null

# Verify the api-gateway deployment is healthy.
if ! kubectl -n "${NAMESPACE}" get deployment \
    "zynax-api-gateway" >/dev/null 2>&1; then
  fail "api-gateway deployment not found in namespace '${NAMESPACE}' — run cluster-up.sh first"
fi

# Verify the Argo Workflows CRD is installed — the engine path cannot work
# without it. This is the precondition that distinguishes this test from the
# Temporal happy-path.
if ! kubectl get crd workflows.argoproj.io >/dev/null 2>&1; then
  fail "Argo Workflows CRD 'workflows.argoproj.io' not found — cluster is not configured with ArgoEngine"
fi

log "preflight passed."

# ── 1. Submit workflow via api-gateway with engine=argo ──────────────────────────

log "step 1: submitting code-review workflow via api-gateway at ${API_GW_URL} (engine=argo)…"

APPLY_RESPONSE=$(api_curl POST "/api/v1/apply?engine=argo" \
  -H "Content-Type: application/x-yaml" \
  --data-binary "@${WORKFLOW_FILE}" 2>&1) \
  || fail "POST /api/v1/apply?engine=argo failed. Is api-gateway reachable at ${API_GW_URL}? Response: ${APPLY_RESPONSE}"

log "apply response: ${APPLY_RESPONSE}"

RUN_ID=$(printf '%s' "${APPLY_RESPONSE}" | jq -r '.run_id // empty')
[[ -n "${RUN_ID}" ]] || fail "apply response did not contain run_id. Full response: ${APPLY_RESPONSE}"

pass "step 1: workflow submitted to Argo engine. run_id=${RUN_ID}"

# ── 2. Poll workflow status until succeeded ──────────────────────────────────────

log "step 2: polling GET /api/v1/workflows/${RUN_ID} for status=succeeded (timeout=${POLL_TIMEOUT}s)…"

ELAPSED=0
FINAL_STATUS=""
while [[ $ELAPSED -lt $POLL_TIMEOUT ]]; do
  STATUS_RESPONSE=$(api_curl GET "/api/v1/workflows/${RUN_ID}" 2>/dev/null) || {
    warn "status poll failed at ${ELAPSED}s — will retry"
    sleep "${POLL_INTERVAL}"
    ELAPSED=$((ELAPSED + POLL_INTERVAL))
    continue
  }
  FINAL_STATUS=$(printf '%s' "${STATUS_RESPONSE}" | jq -r '.status // empty')
  log "  [${ELAPSED}s] status=${FINAL_STATUS}"

  case "${FINAL_STATUS}" in
    succeeded|completed)
      break
      ;;
    failed|error)
      fail "workflow reached terminal failure state '${FINAL_STATUS}'. Response: ${STATUS_RESPONSE}"
      ;;
  esac
  sleep "${POLL_INTERVAL}"
  ELAPSED=$((ELAPSED + POLL_INTERVAL))
done

if [[ "${FINAL_STATUS}" != "succeeded" && "${FINAL_STATUS}" != "completed" ]]; then
  fail "workflow did not reach succeeded within ${POLL_TIMEOUT}s. Last status: '${FINAL_STATUS}'"
fi

pass "step 2: workflow reached terminal success state '${FINAL_STATUS}' (run_id=${RUN_ID})."

# ── 3. Assert the Argo Workflow resource reached the Succeeded phase ──────────────
#
# The api-gateway run_id equals the name of the Argo `Workflow` custom resource
# (see argo_engine.go). We poll its `.status.phase` directly via kubectl to
# confirm the assertion at the engine's own source of truth — not just via the
# gateway projection — proving the workflow really ran on Argo.

log "step 3: asserting Argo Workflow resource '${RUN_ID}' reaches phase 'Succeeded'…"

ARGO_ELAPSED=0
ARGO_PHASE=""
while [[ $ARGO_ELAPSED -lt $POLL_TIMEOUT ]]; do
  ARGO_PHASE=$(kubectl -n "${NAMESPACE}" get workflow.argoproj.io "${RUN_ID}" \
    -o jsonpath='{.status.phase}' 2>/dev/null || true)
  log "  [${ARGO_ELAPSED}s] argo workflow phase=${ARGO_PHASE:-<none>}"

  case "${ARGO_PHASE}" in
    Succeeded)
      break
      ;;
    Failed|Error)
      WF_DUMP=$(kubectl -n "${NAMESPACE}" get workflow.argoproj.io "${RUN_ID}" \
        -o jsonpath='{.status.message}' 2>/dev/null || true)
      fail "Argo Workflow '${RUN_ID}' reached terminal failure phase '${ARGO_PHASE}'. Message: ${WF_DUMP}"
      ;;
  esac
  sleep "${POLL_INTERVAL}"
  ARGO_ELAPSED=$((ARGO_ELAPSED + POLL_INTERVAL))
done

if [[ "${ARGO_PHASE}" != "Succeeded" ]]; then
  fail "Argo Workflow '${RUN_ID}' did not reach 'Succeeded' within ${POLL_TIMEOUT}s. Last phase: '${ARGO_PHASE:-<none>}'"
fi

pass "step 3: Argo Workflow '${RUN_ID}' reached phase 'Succeeded'."

# ── summary ──────────────────────────────────────────────────────────────────────

printf '\n\033[1;32m[e2e-argo] ALL ASSERTIONS PASSED\033[0m\n'
printf '  engine:    argo\n'
printf '  workflow:  run_id=%s  gateway_status=%s\n' "${RUN_ID}" "${FINAL_STATUS}"
printf '  argo:      workflow=%s  phase=%s\n' "${RUN_ID}" "${ARGO_PHASE}"
printf '\n'
