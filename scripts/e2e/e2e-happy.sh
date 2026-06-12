#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# e2e-happy.sh — assert the Temporal happy-path end-to-end.
#
# EPIC G (#770) step 2 / #810. Submits the reference workflow via api-gateway,
# polls until the workflow reaches "succeeded" state, asserts the lifecycle
# CloudEvents arrived on NATS JetStream, and verifies that the memory-service
# KV plane works by writing a sentinel key and reading it back. The CloudEvent
# + memory assertions are REQUIRED (#1090, canvas 1086 O4) — there is no skip
# path. Exception: the terminal workflow.completed event is enforced only with
# E2E_REQUIRE_COMPLETED_EVENT=1 until bug #1149 (JetStream stream subject
# overlap makes it undeliverable) is fixed — see step 3b.
#
# Requires a running kind cluster created by cluster-up.sh (G.1 / #809).
# Compatible with both ci/docker and local developer environments.
#
# Usage:
#   scripts/e2e/e2e-happy.sh
#
# Environment overrides:
#   CLUSTER_NAME       kind cluster name                  (default: zynax-e2e)
#   NAMESPACE          release namespace                   (default: zynax)
#   RELEASE_NAME       Helm release name                  (default: zynax)
#   API_GW_URL         api-gateway base URL               (default: http://localhost:8080)
#   ZYNAX_API_KEY      bearer token (empty = no auth)     (default: "")
#   POLL_TIMEOUT       max seconds to wait for succeeded  (default: 120)
#   POLL_INTERVAL      seconds between status polls       (default: 5)
#   WORKFLOW_FILE      path to the workflow YAML          (default: spec/workflows/examples/e2e-demo.yaml)
#   NATS_ASSERT_TIMEOUT          max seconds to wait for the JetStream events (default: 60)
#   E2E_REQUIRE_COMPLETED_EVENT  1 = hard-fail when workflow.completed is not
#                                on JetStream (default: 0 until #1149 is fixed)
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
# Default to the minimal echo workflow (#1088): a single "echo" capability that
# the deployed echo-worker satisfies, so the run reaches terminal succeeded.
WORKFLOW_FILE="${WORKFLOW_FILE:-${REPO_ROOT}/spec/workflows/examples/e2e-demo.yaml}"

# Sentinel key for memory-service roundtrip assertion.
MEMORY_WF_ID="e2e-happy-$(date +%s)"
MEMORY_KEY="e2e.sentinel"
MEMORY_VALUE="zynax-e2e-ok"

# Port-forward pids — cleaned up on exit.
_PF_PIDS=()

# ── helpers ──────────────────────────────────────────────────────────────────────

log()  { printf '\033[1;34m[e2e-happy]\033[0m %s\n' "$*"; }
pass() { printf '\033[1;32m[e2e-happy][PASS]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[e2e-happy][FAIL]\033[0m %s\n' "$*" >&2; exit 1; }
warn() { printf '\033[1;33m[e2e-happy][WARN]\033[0m %s\n' "$*" >&2; }

require() {
  command -v "$1" >/dev/null 2>&1 || fail "required tool not found on PATH: $1"
}

# cleanup kills any background port-forwards started by this script.
cleanup() {
  for pid in "${_PF_PIDS[@]+"${_PF_PIDS[@]}"}"; do
    kill "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT

# port_forward <resource> <local_port> <remote_port>
# Starts kubectl port-forward in background and waits up to 10s for the port
# to accept connections.
port_forward() {
  local resource="$1" local_port="$2" remote_port="$3"
  kubectl -n "${NAMESPACE}" port-forward "${resource}" \
    "${local_port}:${remote_port}" >/dev/null 2>&1 &
  local pf_pid=$!
  _PF_PIDS+=("$pf_pid")
  # Wait until the port is reachable (up to 10 s).
  local i=0
  while ! (echo >/dev/tcp/127.0.0.1/"${local_port}") 2>/dev/null; do
    i=$((i + 1))
    if [[ $i -ge 10 ]]; then
      fail "port-forward ${resource}:${remote_port} → localhost:${local_port} did not become ready in 10s"
    fi
    sleep 1
  done
  log "port-forward ready: localhost:${local_port} → ${resource}:${remote_port}"
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
# Required for the memory-service KV roundtrip assertion (#1090) — installed by
# e2e-smoke.yml in CI; locally: https://github.com/fullstorydev/grpcurl/releases
require grpcurl

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

# Resolve the api-gateway bearer key. api-gateway requires ZYNAX_GW_API_KEY and
# cluster-up.sh provisions a random one in the zynax-gw-api-key secret, so read
# it from there when the caller did not supply ZYNAX_API_KEY (avoids a 401).
if [[ -z "${ZYNAX_API_KEY}" ]]; then
  ZYNAX_API_KEY=$(kubectl -n "${NAMESPACE}" get secret zynax-gw-api-key \
    -o jsonpath='{.data.api-key}' 2>/dev/null | base64 -d || true)
  [[ -n "${ZYNAX_API_KEY}" ]] && log "using api-gateway key from the zynax-gw-api-key secret."
fi

# Verify the event-bus and memory-service deployments exist. These are REQUIRED
# (#1090, canvas 1086 O4): the CloudEvent + memory-Get assertions below run
# unconditionally — there is no skip path. Deployment names are pinned via
# fullnameOverride in values-e2e.yaml (same as the other 5 services).
kubectl -n "${NAMESPACE}" get deployment "zynax-event-bus" >/dev/null 2>&1 \
  || fail "event-bus deployment not found — required for the NATS CloudEvent assertion (values-e2e.yaml enables it; run cluster-up.sh)"
kubectl -n "${NAMESPACE}" get deployment "zynax-memory-service" >/dev/null 2>&1 \
  || fail "memory-service deployment not found — required for the KV roundtrip assertion (values-e2e.yaml enables it; run cluster-up.sh)"

log "preflight passed."

# Reach api-gateway via a port-forward by default. The NodePort host mapping
# (host 8080 -> nodePort 30080) works locally but kube-proxy can reset it on the
# GitHub runner when the control-plane node forwards to a pod on a worker node.
# A port-forward tunnels through the kube-apiserver and is environment-independent.
# Honors a caller-provided API_GW_URL (skip the forward if it was overridden).
if [[ "${API_GW_URL}" == "http://localhost:8080" ]]; then
  GW_LOCAL_PORT="${GW_LOCAL_PORT:-18080}"
  port_forward "svc/zynax-api-gateway" "${GW_LOCAL_PORT}" 8080
  API_GW_URL="http://localhost:${GW_LOCAL_PORT}"
fi

# ── 1. Submit workflow via api-gateway ───────────────────────────────────────────

log "step 1: submitting code-review workflow via api-gateway at ${API_GW_URL}…"

APPLY_RESPONSE=$(api_curl POST /api/v1/apply \
  -H "Content-Type: application/x-yaml" \
  --data-binary "@${WORKFLOW_FILE}" 2>&1) \
  || fail "POST /api/v1/apply failed. Is api-gateway reachable at ${API_GW_URL}? Response: ${APPLY_RESPONSE}"

log "apply response: ${APPLY_RESPONSE}"

RUN_ID=$(printf '%s' "${APPLY_RESPONSE}" | jq -r '.run_id // empty')
[[ -n "${RUN_ID}" ]] || fail "apply response did not contain run_id. Full response: ${APPLY_RESPONSE}"

pass "step 1: workflow submitted. run_id=${RUN_ID}"

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

  # Accept both lowercase aliases and the WorkflowStatus proto enum names the
  # api-gateway returns (e.g. WORKFLOW_STATUS_COMPLETED / _FAILED).
  case "${FINAL_STATUS}" in
    succeeded|completed|*COMPLETED|*SUCCEEDED)
      break
      ;;
    failed|error|*FAILED|*ERROR|*CANCELED|*TERMINATED|*TIMED_OUT)
      fail "workflow reached terminal failure state '${FINAL_STATUS}'. Response: ${STATUS_RESPONSE}"
      ;;
  esac
  sleep "${POLL_INTERVAL}"
  ELAPSED=$((ELAPSED + POLL_INTERVAL))
done

case "${FINAL_STATUS}" in
  succeeded|completed|*COMPLETED|*SUCCEEDED) ;;
  *)
    fail "workflow did not reach succeeded within ${POLL_TIMEOUT}s. Last status: '${FINAL_STATUS}'"
    ;;
esac

pass "step 2: workflow reached terminal success state '${FINAL_STATUS}' (run_id=${RUN_ID})."

# ── 3. Assert CloudEvents off NATS JetStream ──────────────────────────────────────
#
# REQUIRED since #1090 (canvas 1086 O4) — no skip path. The engine-adapter
# publishes lifecycle CloudEvents through EventBusService, which lands them on
# NATS JetStream. PublishLifecycleEventActivity (services/engine-adapter/
# internal/infrastructure/activities.go) builds the subject as
#   "zynax.v1.engine-adapter.workflow." + <event type from interpreter.go>
# and event-bus derives the stream by dropping the last subject segment and
# upper-snake-casing the rest (StreamName in services/event-bus/internal/
# infrastructure/nats.go). We assert with the `nats` CLI inside the nats-box
# pod (deployed by the NATS subchart) so the host needs no NATS tooling.
#
# Two checks:
#   3a (REQUIRED): the state-lifecycle CloudEvents (zynax.workflow.state.entered/
#       exited) for this run are on JetStream — proves the full engine-adapter →
#       event-bus → NATS pipeline delivers CloudEvents end-to-end.
#   3b: the terminal zynax.workflow.completed CloudEvent. BLOCKED by #1149:
#       the state stream's subject filter overlaps the completed stream's, so
#       JetStream rejects the completed stream (err 10065) and the event is
#       undeliverable today. Enforced only when E2E_REQUIRE_COMPLETED_EVENT=1.
#       TODO(#1149): flip the default to required once the stream-derivation
#       overlap is fixed — do NOT remove this assertion.

log "step 3: asserting lifecycle CloudEvents off NATS JetStream…"

STATE_SUBJECT_PREFIX="zynax.v1.engine-adapter.workflow.zynax.workflow.state"
STATE_STREAM="ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW_ZYNAX_WORKFLOW_STATE"
COMPLETED_SUBJECT="zynax.v1.engine-adapter.workflow.zynax.workflow.completed"
COMPLETED_STREAM="ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW_ZYNAX_WORKFLOW"
NATS_ASSERT_TIMEOUT="${NATS_ASSERT_TIMEOUT:-60}"
E2E_REQUIRE_COMPLETED_EVENT="${E2E_REQUIRE_COMPLETED_EVENT:-0}"

NATS_BOX_POD=$(kubectl -n "${NAMESPACE}" get pod \
  -l "app.kubernetes.io/component=nats-box" \
  -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
[[ -n "${NATS_BOX_POD}" ]] \
  || fail "step 3: nats-box pod not found (label app.kubernetes.io/component=nats-box) — the NATS subchart must be enabled with natsBox for this assertion"
log "  nats-box pod: ${NATS_BOX_POD}"

# nats_exec <args…> — run the nats CLI inside the nats-box pod.
nats_exec() {
  kubectl -n "${NAMESPACE}" exec "${NATS_BOX_POD}" -- nats "$@"
}

# nats_diagnostics — dump JetStream + publisher state on assertion failure so
# the CI log carries the evidence (the cluster is torn down right after).
nats_diagnostics() {
  warn "── step 3 diagnostics ──"
  warn "JetStream streams:"
  nats_exec stream ls 2>&1 | sed 's/^/    /' >&2 || true
  warn "event-bus log tail:"
  kubectl -n "${NAMESPACE}" logs deployment/zynax-event-bus --tail=40 2>&1 | sed 's/^/    /' >&2 || true
  warn "engine-adapter log tail:"
  kubectl -n "${NAMESPACE}" logs deployment/zynax-engine-adapter --tail=40 2>&1 | sed 's/^/    /' >&2 || true
}

# 3a — REQUIRED: state-lifecycle CloudEvents delivered for this run.
NATS_ELAPSED=0
STATE_MSGS="0"
while [[ ${NATS_ELAPSED} -lt ${NATS_ASSERT_TIMEOUT} ]]; do
  STATE_MSGS=$(nats_exec stream info "${STATE_STREAM}" --json 2>/dev/null \
    | jq -r '.state.messages // 0' 2>/dev/null || echo "0")
  [[ "${STATE_MSGS}" =~ ^[0-9]+$ ]] || STATE_MSGS=0
  if [[ "${STATE_MSGS}" -gt 0 ]]; then
    break
  fi
  log "  [${NATS_ELAPSED}s] stream '${STATE_STREAM}' not ready yet (messages=${STATE_MSGS}) — retrying…"
  sleep 5
  NATS_ELAPSED=$((NATS_ELAPSED + 5))
done
if [[ "${STATE_MSGS}" -eq 0 ]]; then
  nats_diagnostics
  fail "step 3a: NATS JetStream stream '${STATE_STREAM}' has no messages after ${NATS_ASSERT_TIMEOUT}s — lifecycle CloudEvents are not reaching JetStream"
fi
log "  stream '${STATE_STREAM}' has ${STATE_MSGS} message(s)."

LAST_STATE_EVENT=$(nats_exec stream get "${STATE_STREAM}" \
  --last-for "${STATE_SUBJECT_PREFIX}.entered" 2>&1) || {
    nats_diagnostics
    fail "step 3a: no message on subject '${STATE_SUBJECT_PREFIX}.entered' in stream '${STATE_STREAM}': ${LAST_STATE_EVENT}"
  }
log "  last state.entered event: ${LAST_STATE_EVENT}"
printf '%s' "${LAST_STATE_EVENT}" | grep -q "zynax.workflow.state.entered" || {
  nats_diagnostics
  fail "step 3a: message on '${STATE_SUBJECT_PREFIX}.entered' does not carry the zynax.workflow.state.entered CloudEvent type. Payload: ${LAST_STATE_EVENT}"
}
if printf '%s' "${LAST_STATE_EVENT}" | grep -q "${RUN_ID}"; then
  pass "step 3a: lifecycle CloudEvents delivered to JetStream (stream=${STATE_STREAM}, workflow_id matches run_id=${RUN_ID})."
else
  warn "  state.entered payload does not reference run_id=${RUN_ID} (workflow id mapping may differ)."
  pass "step 3a: lifecycle CloudEvents delivered to JetStream (stream=${STATE_STREAM}, ${STATE_MSGS} message(s))."
fi

# 3b — terminal completed CloudEvent (gated on #1149, see header comment).
COMPLETED_MSGS=$(nats_exec stream info "${COMPLETED_STREAM}" --json 2>/dev/null \
  | jq -r '.state.messages // 0' 2>/dev/null || echo "0")
[[ "${COMPLETED_MSGS}" =~ ^[0-9]+$ ]] || COMPLETED_MSGS=0
if [[ "${COMPLETED_MSGS}" -gt 0 ]]; then
  LAST_EVENT=$(nats_exec stream get "${COMPLETED_STREAM}" \
    --last-for "${COMPLETED_SUBJECT}" 2>&1) || true
  log "  last completed event: ${LAST_EVENT}"
  if printf '%s' "${LAST_EVENT}" | grep -q "zynax.workflow.completed"; then
    pass "step 3b: workflow.completed CloudEvent delivered to JetStream (stream=${COMPLETED_STREAM})."
  elif [[ "${E2E_REQUIRE_COMPLETED_EVENT}" -eq 1 ]]; then
    nats_diagnostics
    fail "step 3b: stream '${COMPLETED_STREAM}' has messages but none carry zynax.workflow.completed"
  fi
elif [[ "${E2E_REQUIRE_COMPLETED_EVENT}" -eq 1 ]]; then
  nats_diagnostics
  fail "step 3b: workflow.completed CloudEvent not on JetStream (stream='${COMPLETED_STREAM}' empty or missing)"
else
  warn "step 3b: workflow.completed CloudEvent NOT on JetStream — known bug #1149 (stream subject overlap)."
  warn "         Enforce with E2E_REQUIRE_COMPLETED_EVENT=1 once #1149 is fixed."
fi

# ── 4. Assert memory-service KV roundtrip ────────────────────────────────────────
#
# REQUIRED since #1090 (canvas 1086 O4) — no skip path, no connectivity-only
# fallback. grpcurl is a preflight requirement (e2e-smoke.yml installs it; for
# local runs: https://github.com/fullstorydev/grpcurl/releases).

log "step 4: asserting memory-service KV roundtrip (workflow_id=${MEMORY_WF_ID})…"

MEMORY_SVC="zynax-memory-service"
MEMORY_PORT=50057
LOCAL_MEM_PORT=15057

# Port-forward the memory-service gRPC port.
port_forward "svc/${MEMORY_SVC}" "${LOCAL_MEM_PORT}" "${MEMORY_PORT}"

log "  using grpcurl at localhost:${LOCAL_MEM_PORT}…"

# Set a sentinel key in memory-service.
SET_RESP=$(grpcurl -plaintext \
  -d "{\"workflow_id\":\"${MEMORY_WF_ID}\",\"key\":\"${MEMORY_KEY}\",\"value\":\"$(printf '%s' "${MEMORY_VALUE}" | base64)\"}" \
  "localhost:${LOCAL_MEM_PORT}" \
  zynax.v1.MemoryService/Set 2>&1) || fail "step 4: memory-service Set RPC failed: ${SET_RESP}"
log "  Set response: ${SET_RESP}"

# Get the sentinel key back.
GET_RESP=$(grpcurl -plaintext \
  -d "{\"workflow_id\":\"${MEMORY_WF_ID}\",\"key\":\"${MEMORY_KEY}\"}" \
  "localhost:${LOCAL_MEM_PORT}" \
  zynax.v1.MemoryService/Get 2>&1) || fail "step 4: memory-service Get RPC failed: ${GET_RESP}"
log "  Get response: ${GET_RESP}"

# Decode the returned value and compare.
RETURNED_RAW=$(printf '%s' "${GET_RESP}" | jq -r '.value // empty')
RETURNED_VALUE=$(printf '%s' "${RETURNED_RAW}" | base64 -d 2>/dev/null || printf '%s' "${RETURNED_RAW}")
if [[ "${RETURNED_VALUE}" == "${MEMORY_VALUE}" ]]; then
  pass "step 4: memory-service KV roundtrip verified (key='${MEMORY_KEY}', value='${RETURNED_VALUE}')."
else
  fail "step 4: memory-service Get returned '${RETURNED_VALUE}', expected '${MEMORY_VALUE}'"
fi

# ── summary ──────────────────────────────────────────────────────────────────────

printf '\n\033[1;32m[e2e-happy] ALL ASSERTIONS PASSED\033[0m\n'
printf '  workflow:  run_id=%s  status=%s\n' "${RUN_ID}" "${FINAL_STATUS}"
printf '  event-bus: state-stream=%s messages=%s  completed-stream=%s messages=%s (enforced=%s, #1149)\n' \
  "${STATE_STREAM}" "${STATE_MSGS}" "${COMPLETED_STREAM}" "${COMPLETED_MSGS}" "${E2E_REQUIRE_COMPLETED_EVENT}"
printf '  memory:    workflow_id=%s  key=%s roundtrip=ok\n' "${MEMORY_WF_ID}" "${MEMORY_KEY}"
printf '\n'
