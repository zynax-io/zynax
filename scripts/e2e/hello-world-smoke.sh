#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# hello-world-smoke.sh — kind happy-path smoke for the zero-dependency
# hello-world workflow (#1493, canvas O23). Asserts that
# spec/workflows/examples/hello-world.yaml runs to COMPLETED over the
# kind-deployed stack with NO model and NO secret, and that the echoed output
# is visible from the api-gateway result. The echo-worker that satisfies the
# "echo" capability is deployed by scripts/e2e/cluster-up.sh (#1492), so this
# script only submits the manifest and asserts the terminal state + payload.
#
# It deliberately mirrors scripts/e2e/e2e-happy.sh's submit + poll pattern (same
# api-gateway endpoints, the SAME terminal-status alias set) but stays minimal:
# it does NOT assert the NATS/memory planes — e2e-happy.sh owns those. Run it as
# a fast, zero-dependency confidence check on a cluster created by cluster-up.sh.
#
# Requires a running kind cluster created by cluster-up.sh.
#
# Usage:
#   scripts/e2e/hello-world-smoke.sh
#
# Environment overrides:
#   CLUSTER_NAME   kind cluster name                 (default: zynax-e2e)
#   NAMESPACE      release namespace                 (default: zynax)
#   API_GW_URL     api-gateway base URL              (default: http://localhost:8080)
#   ZYNAX_API_KEY  bearer token (empty = read secret)(default: "")
#   POLL_TIMEOUT   max seconds to wait for COMPLETED (default: 120)
#   POLL_INTERVAL  seconds between status polls      (default: 5)
#   WORKFLOW_FILE  path to the hello-world YAML       (default: the bundled example)
#
# Exit codes:
#   0  the workflow reached COMPLETED and the echoed output was found
#   1  an assertion failed or a required tool is missing

set -euo pipefail

# ── configuration ──────────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

CLUSTER_NAME="${CLUSTER_NAME:-zynax-e2e}"
NAMESPACE="${NAMESPACE:-zynax}"
API_GW_URL="${API_GW_URL:-http://localhost:8080}"
ZYNAX_API_KEY="${ZYNAX_API_KEY:-}"
POLL_TIMEOUT="${POLL_TIMEOUT:-120}"
POLL_INTERVAL="${POLL_INTERVAL:-5}"
WORKFLOW_FILE="${WORKFLOW_FILE:-${REPO_ROOT}/spec/workflows/examples/hello-world.yaml}"
# The literal message in hello-world.yaml — asserted to appear in the echoed
# output so a green run also proves the payload round-tripped (AC2).
EXPECTED_ECHO="${EXPECTED_ECHO:-Hello from Zynax}"

_PF_PIDS=()

# ── helpers ────────────────────────────────────────────────────────────────────

log()  { printf '\033[1;34m[hello-world]\033[0m %s\n' "$*"; }
pass() { printf '\033[1;32m[hello-world][PASS]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[hello-world][FAIL]\033[0m %s\n' "$*" >&2; exit 1; }
warn() { printf '\033[1;33m[hello-world][WARN]\033[0m %s\n' "$*" >&2; }

require() {
  command -v "$1" >/dev/null 2>&1 || fail "required tool not found on PATH: $1"
}

cleanup() {
  for pid in "${_PF_PIDS[@]+"${_PF_PIDS[@]}"}"; do
    kill "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT

# port_forward <resource> <local_port> <remote_port> — background kubectl
# port-forward, wait up to PF_TIMEOUT (default 30s) for the port to accept.
PF_TIMEOUT="${PF_TIMEOUT:-30}"
port_forward() {
  local resource="$1" local_port="$2" remote_port="$3"
  local pf_log; pf_log=$(mktemp)
  kubectl -n "${NAMESPACE}" port-forward "${resource}" \
    "${local_port}:${remote_port}" >"${pf_log}" 2>&1 &
  local pf_pid=$!
  _PF_PIDS+=("$pf_pid")
  local i=0
  while ! (echo >"/dev/tcp/127.0.0.1/${local_port}") 2>/dev/null; do
    if ! kill -0 "${pf_pid}" 2>/dev/null; then
      fail "port-forward ${resource}:${remote_port} exited: $(cat "${pf_log}")"
    fi
    i=$((i + 1))
    [[ $i -ge ${PF_TIMEOUT} ]] && fail "port-forward ${resource}:${remote_port} not ready in ${PF_TIMEOUT}s"
    sleep 1
  done
  rm -f "${pf_log}"
  log "port-forward ready: localhost:${local_port} → ${resource}:${remote_port}"
}

# api_curl <method> <path> [extra curl args...] — call the api-gateway with the
# bearer token when set.
api_curl() {
  local method="$1" path="$2"; shift 2
  local auth_args=()
  [[ -n "${ZYNAX_API_KEY}" ]] && auth_args=(-H "Authorization: Bearer ${ZYNAX_API_KEY}")
  curl --silent --show-error --fail \
    -X "${method}" "${auth_args[@]+"${auth_args[@]}"}" "$@" "${API_GW_URL}${path}"
}

# ── preflight ──────────────────────────────────────────────────────────────────

log "preflight: checking required tools and cluster state…"
require kubectl
require curl
require jq

[[ -f "${WORKFLOW_FILE}" ]] || fail "workflow file not found: ${WORKFLOW_FILE}"

if ! kubectl config get-contexts "kind-${CLUSTER_NAME}" >/dev/null 2>&1; then
  fail "kubectl context 'kind-${CLUSTER_NAME}' not found — run scripts/e2e/cluster-up.sh first"
fi
kubectl config use-context "kind-${CLUSTER_NAME}" >/dev/null

kubectl -n "${NAMESPACE}" get deployment "zynax-api-gateway" >/dev/null 2>&1 \
  || fail "api-gateway deployment not found in namespace '${NAMESPACE}' — run cluster-up.sh first"
# The echo-worker satisfies the "echo" capability (cluster-up.sh deploys it).
kubectl -n "${NAMESPACE}" get deployment "echo-worker" >/dev/null 2>&1 \
  || fail "echo-worker deployment not found — required for the 'echo' capability (cluster-up.sh deploys it)"

# Resolve the gateway bearer key cluster-up.sh provisioned (avoids a 401).
if [[ -z "${ZYNAX_API_KEY}" ]]; then
  ZYNAX_API_KEY=$(kubectl -n "${NAMESPACE}" get secret zynax-edge-apikey \
    -o jsonpath='{.data.zynax-cli}' 2>/dev/null | base64 -d || true)
  [[ -n "${ZYNAX_API_KEY}" ]] && log "using api-gateway key from the zynax-edge-apikey secret."
fi

log "preflight passed."

# Tunnel the gateway by default (the kind NodePort can be reset by kube-proxy;
# a port-forward is environment-independent). Honour a non-default API_GW_URL.
if [[ "${API_GW_URL}" == "http://localhost:8080" ]]; then
  GW_LOCAL_PORT="${GW_LOCAL_PORT:-18080}"
  port_forward "svc/zynax-api-gateway" "${GW_LOCAL_PORT}" 8080
  API_GW_URL="http://localhost:${GW_LOCAL_PORT}"
fi

# ── 1. submit hello-world via api-gateway ──────────────────────────────────────

log "step 1: submitting hello-world via api-gateway at ${API_GW_URL}…"

APPLY_RESPONSE=$(api_curl POST /api/v1/apply \
  -H "Content-Type: application/x-yaml" \
  --data-binary "@${WORKFLOW_FILE}" 2>&1) \
  || fail "POST /api/v1/apply failed. Is api-gateway reachable at ${API_GW_URL}? Response: ${APPLY_RESPONSE}"

log "apply response: ${APPLY_RESPONSE}"
RUN_ID=$(printf '%s' "${APPLY_RESPONSE}" | jq -r '.run_id // empty')
[[ -n "${RUN_ID}" ]] || fail "apply response did not contain run_id. Full response: ${APPLY_RESPONSE}"
pass "step 1: hello-world submitted. run_id=${RUN_ID}"

# ── 2. poll status until COMPLETED ─────────────────────────────────────────────

log "step 2: polling GET /api/v1/workflows/${RUN_ID} for a terminal success (timeout=${POLL_TIMEOUT}s)…"

ELAPSED=0
FINAL_STATUS=""
while [[ $ELAPSED -lt $POLL_TIMEOUT ]]; do
  STATUS_RESPONSE=$(api_curl GET "/api/v1/workflows/${RUN_ID}" 2>/dev/null) || {
    warn "status poll failed at ${ELAPSED}s — will retry"
    sleep "${POLL_INTERVAL}"; ELAPSED=$((ELAPSED + POLL_INTERVAL)); continue
  }
  FINAL_STATUS=$(printf '%s' "${STATUS_RESPONSE}" | jq -r '.status // empty')
  log "  [${ELAPSED}s] status=${FINAL_STATUS}"
  # SAME alias set as e2e-happy.sh — keep the two in lockstep.
  case "${FINAL_STATUS}" in
    succeeded|completed|*COMPLETED|*SUCCEEDED) break ;;
    failed|error|*FAILED|*ERROR|*CANCELED|*TERMINATED|*TIMED_OUT)
      fail "hello-world reached terminal failure state '${FINAL_STATUS}'. Response: ${STATUS_RESPONSE}" ;;
  esac
  sleep "${POLL_INTERVAL}"; ELAPSED=$((ELAPSED + POLL_INTERVAL))
done

case "${FINAL_STATUS}" in
  succeeded|completed|*COMPLETED|*SUCCEEDED) ;;
  *) fail "hello-world did not reach COMPLETED within ${POLL_TIMEOUT}s. Last status: '${FINAL_STATUS}'" ;;
esac
pass "step 2: hello-world reached terminal success state '${FINAL_STATUS}' (run_id=${RUN_ID})."

# ── 3. assert the echo capability round-tripped (AC2) ──────────────────────────
#
# The workflow only transitions greet → done (terminal) when the "echo.completed"
# event arrives, so a COMPLETED status already proves the echo-worker satisfied
# the dispatch. We additionally confirm the run's event stream reached
# WorkflowExecutionCompleted (the terminal lifecycle event) so a green smoke ties
# the success to the actual echo round-trip, not just a status flip. Step 4 then
# reads the declared workflow output over REST — the M7.U output path.

log "step 3: asserting the run reached the terminal completion event (echo round-trip)…"

LOGS=$(api_curl GET "/api/v1/workflows/${RUN_ID}/logs" 2>/dev/null || true)
if printf '%s' "${LOGS}" | grep -q "WorkflowExecutionCompleted"; then
  pass "step 3: run reached WorkflowExecutionCompleted — the echo capability round-tripped."
else
  warn "  WorkflowExecutionCompleted not seen in the streamed logs."
  warn "  logs tail: $(printf '%s' "${LOGS}" | tail -c 400)"
  fail "step 3: run did not reach the terminal completion event (echo dispatch unverified)."
fi

# ── 4. read the declared workflow output over REST (M7.U O.8/O.9, #1103 gap #4) ─
#
# hello-world.yaml declares a terminal output `message: $.states.greet.output.message`
# (the greet echo action publishes echo.message). GET /outputs must return it — the
# live apply → COMPLETED → read-outputs proof that closes #1103 platform gap #4.

log "step 4: reading GET /api/v1/workflows/${RUN_ID}/outputs (declared output read path)…"

OUTPUTS=$(api_curl GET "/api/v1/workflows/${RUN_ID}/outputs" 2>/dev/null) \
  || fail "step 4: GET /api/v1/workflows/${RUN_ID}/outputs failed"
OUT_MESSAGE=$(printf '%s' "${OUTPUTS}" | jq -r '.message // empty')
if [[ "${OUT_MESSAGE}" == "${EXPECTED_ECHO}" ]]; then
  pass "step 4: /outputs returned message=\"${OUT_MESSAGE}\" — apply→COMPLETED→read-outputs proven (gap #4 closed)."
else
  fail "step 4: /outputs did not return the declared message. Expected \"${EXPECTED_ECHO}\", got: ${OUTPUTS}"
fi

# ── summary ────────────────────────────────────────────────────────────────────

printf '\n\033[1;32m[hello-world] SMOKE PASSED\033[0m\n'
printf '  workflow: run_id=%s status=%s sent-message="%s"\n' "${RUN_ID}" "${FINAL_STATUS}" "${EXPECTED_ECHO}"
printf '\n'
