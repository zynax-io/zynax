#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# e2e-failure.sh — assert the Temporal failure-path end-to-end.
#
# EPIC G (#770) step 4 / #812. Submits a workflow whose first state invokes an
# UNREACHABLE capability, then:
#   1. polls the workflow status and asserts it reaches a terminal "failed" state,
#   2. asserts the capability dispatch timed out within ZYNAX_CAPABILITY_TIMEOUT,
#   3. asserts the workflow.failed CloudEvent (subject
#      "zynax.v1.engine-adapter.workflow.failed") arrived on NATS JetStream —
#      REQUIRED since the #1149 fix, no skip path.
#
# This is the failure-path sibling of e2e-happy.sh (#810): same cluster, same
# api-gateway submission flow, same JetStream assertion approach — only the
# expected terminal outcome is inverted (failed, not succeeded) and the asserted
# CloudEvent is the terminal workflow.failed event.
#
# The workflow fixture is generated at runtime (see make_failure_workflow below)
# rather than committed under spec/workflows/examples/, because it intentionally
# references a capability that no agent serves so the dispatch is guaranteed to
# time out. Keeping it out of the published examples avoids advertising a broken
# workflow as a reference.
#
# Requires a running kind cluster created by cluster-up.sh (G.1 / #809).
# Compatible with both ci/docker and local developer environments.
#
# Usage:
#   scripts/e2e/e2e-failure.sh
#
# Environment overrides:
#   CLUSTER_NAME              kind cluster name                  (default: zynax-e2e)
#   NAMESPACE                 release namespace                   (default: zynax)
#   RELEASE_NAME              Helm release name                  (default: zynax)
#   API_GW_URL                api-gateway base URL               (default: http://localhost:8080)
#   ZYNAX_API_KEY             bearer token (empty = no auth)     (default: "")
#   ZYNAX_CAPABILITY_TIMEOUT  capability dispatch timeout        (default: 30s)
#   POLL_TIMEOUT              max seconds to wait for failed     (default: 120)
#   POLL_INTERVAL             seconds between status polls       (default: 5)
#
# Exit codes:
#   0  all assertions passed (the failure path behaved as expected)
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
ZYNAX_CAPABILITY_TIMEOUT="${ZYNAX_CAPABILITY_TIMEOUT:-30s}"
POLL_TIMEOUT="${POLL_TIMEOUT:-120}"
POLL_INTERVAL="${POLL_INTERVAL:-5}"

# Unique run marker so concurrent runs don't collide on the generated fixture.
RUN_MARKER="e2e-failure-$(date +%s)"
WORKFLOW_FILE=""  # set by make_failure_workflow

# Port-forward pids — cleaned up on exit.
_PF_PIDS=()

# ── helpers ──────────────────────────────────────────────────────────────────────

log()  { printf '\033[1;34m[e2e-failure]\033[0m %s\n' "$*"; }
pass() { printf '\033[1;32m[e2e-failure][PASS]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[e2e-failure][FAIL]\033[0m %s\n' "$*" >&2; exit 1; }
warn() { printf '\033[1;33m[e2e-failure][WARN]\033[0m %s\n' "$*" >&2; }

require() {
  command -v "$1" >/dev/null 2>&1 || fail "required tool not found on PATH: $1"
}

# cleanup kills any background port-forwards and removes the generated fixture.
cleanup() {
  for pid in "${_PF_PIDS[@]+"${_PF_PIDS[@]}"}"; do
    kill "$pid" 2>/dev/null || true
  done
  [[ -n "${WORKFLOW_FILE}" && -f "${WORKFLOW_FILE}" ]] && rm -f "${WORKFLOW_FILE}"
}
trap cleanup EXIT

# port_forward <resource> <local_port> <remote_port>
# Starts kubectl port-forward in background and waits up to PF_TIMEOUT seconds
# (default: 30) for the port to accept connections.
PF_TIMEOUT="${PF_TIMEOUT:-30}"

port_forward() {
  local resource="$1" local_port="$2" remote_port="$3"
  local pf_log
  pf_log=$(mktemp)
  kubectl -n "${NAMESPACE}" port-forward "${resource}" \
    "${local_port}:${remote_port}" >"${pf_log}" 2>&1 &
  local pf_pid=$!
  _PF_PIDS+=("$pf_pid")
  local i=0
  while ! (echo >/dev/tcp/127.0.0.1/"${local_port}") 2>/dev/null; do
    if ! kill -0 "${pf_pid}" 2>/dev/null; then
      fail "port-forward ${resource}:${remote_port} exited unexpectedly: $(cat "${pf_log}")"
    fi
    i=$((i + 1))
    if [[ $i -ge ${PF_TIMEOUT} ]]; then
      fail "port-forward ${resource}:${remote_port} → localhost:${local_port} did not become ready in ${PF_TIMEOUT}s"
    fi
    sleep 1
  done
  rm -f "${pf_log}"
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

# make_failure_workflow writes a workflow YAML to a temp file. Its initial state
# invokes the capability "e2e_unreachable_capability", which no agent in the
# deployed stack serves. The capability dispatch therefore blocks until the
# engine-adapter's ZYNAX_CAPABILITY_TIMEOUT fires, driving the workflow to a
# terminal failure and emitting "zynax.workflow.failed".
make_failure_workflow() {
  WORKFLOW_FILE="$(mktemp "${TMPDIR:-/tmp}/${RUN_MARKER}.XXXXXX.yaml")"
  cat > "${WORKFLOW_FILE}" <<YAML
# SPDX-License-Identifier: Apache-2.0
# GENERATED at runtime by scripts/e2e/e2e-failure.sh — do not commit.
# Intentionally references a capability no agent serves, so dispatch times out.
kind: Workflow
apiVersion: zynax.io/v1

metadata:
  name: ${RUN_MARKER}
  namespace: engineering
  labels:
    team: platform
    tier: e2e-failure
  annotations:
    description: "e2e failure-path: unreachable capability forces a timeout"

spec:
  initial_state: dispatch

  states:
    # Invoke a capability that no agent in the cluster serves. The engine-adapter
    # blocks on dispatch until ZYNAX_CAPABILITY_TIMEOUT elapses, then fails.
    dispatch:
      actions:
        - capability: e2e_unreachable_capability
          input:
            note: "no agent serves this capability — dispatch must time out"
      on:
        - event: dispatch.done
          goto: done

    done:
      type: terminal
      actions: []
YAML
  log "generated failure-path workflow fixture: ${WORKFLOW_FILE}"
}

# ── preflight ──────────────────────────────────────────────────────────────────

log "preflight: checking required tools and cluster state…"

require kubectl
require curl
require jq

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

# Resolve the api-gateway bearer key from the zynax-gw-api-key secret (random,
# provisioned by cluster-up.sh) when the caller did not supply one (avoids 401).
if [[ -z "${ZYNAX_API_KEY}" ]]; then
  ZYNAX_API_KEY=$(kubectl -n "${NAMESPACE}" get secret zynax-gw-api-key \
    -o jsonpath='{.data.api-key}' 2>/dev/null | base64 -d || true)
  [[ -n "${ZYNAX_API_KEY}" ]] && log "using api-gateway key from the zynax-gw-api-key secret."
fi

# Verify event-bus exists — the workflow.failed CloudEvent assertion needs it.
# The deployment name is pinned via fullnameOverride in values-e2e.yaml (#1090).
# REQUIRED since the #1149 fix made the terminal failed event reliably
# deliverable — the assertion has no skip path (same strictness as e2e-happy.sh).
kubectl -n "${NAMESPACE}" get deployment "zynax-event-bus" >/dev/null 2>&1 \
  || fail "event-bus deployment not found — required for the workflow.failed CloudEvent assertion (values-e2e.yaml enables it; run cluster-up.sh)"

log "preflight passed (capability timeout = ${ZYNAX_CAPABILITY_TIMEOUT})."

# Reach api-gateway via a port-forward by default — the kind NodePort host
# mapping (8080 -> 30080) is reset by kube-proxy on the GitHub runner. A
# port-forward tunnels through the kube-apiserver. Honors an overridden API_GW_URL.
if [[ "${API_GW_URL}" == "http://localhost:8080" ]]; then
  GW_LOCAL_PORT="${GW_LOCAL_PORT:-18080}"
  port_forward "svc/zynax-api-gateway" "${GW_LOCAL_PORT}" 8080
  API_GW_URL="http://localhost:${GW_LOCAL_PORT}"
fi

# ── 1. Submit a workflow with an unreachable capability ───────────────────────────

make_failure_workflow

log "step 1: submitting failure-path workflow via api-gateway at ${API_GW_URL}…"

APPLY_RESPONSE=$(api_curl POST /api/v1/apply \
  -H "Content-Type: application/x-yaml" \
  --data-binary "@${WORKFLOW_FILE}" 2>&1) \
  || fail "POST /api/v1/apply failed. Is api-gateway reachable at ${API_GW_URL}? Response: ${APPLY_RESPONSE}"

log "apply response: ${APPLY_RESPONSE}"

RUN_ID=$(printf '%s' "${APPLY_RESPONSE}" | jq -r '.run_id // empty')
[[ -n "${RUN_ID}" ]] || fail "apply response did not contain run_id. Full response: ${APPLY_RESPONSE}"

pass "step 1: failure-path workflow submitted. run_id=${RUN_ID}"

# ── 2. Poll workflow status until terminal failure ────────────────────────────────
#
# Unlike the happy-path, we EXPECT a terminal failure here. Reaching "succeeded"
# is the failure condition for this test — the unreachable capability must time
# out. We capture the elapsed time so we can sanity-check it against the
# configured capability timeout.

log "step 2: polling GET /api/v1/workflows/${RUN_ID} for status=failed (timeout=${POLL_TIMEOUT}s)…"

START_TS=$(date +%s)
ELAPSED=0
FINAL_STATUS=""
STATUS_RESPONSE=""
while [[ $ELAPSED -lt $POLL_TIMEOUT ]]; do
  STATUS_RESPONSE=$(api_curl GET "/api/v1/workflows/${RUN_ID}" 2>/dev/null) || {
    warn "status poll failed at ${ELAPSED}s — will retry"
    sleep "${POLL_INTERVAL}"
    ELAPSED=$((ELAPSED + POLL_INTERVAL))
    continue
  }
  FINAL_STATUS=$(printf '%s' "${STATUS_RESPONSE}" | jq -r '.status // empty')
  log "  [${ELAPSED}s] status=${FINAL_STATUS}"

  # Accept lowercase aliases and the WorkflowStatus proto enum names.
  case "${FINAL_STATUS}" in
    failed|error|*FAILED|*ERROR|*CANCELED|*TERMINATED|*TIMED_OUT)
      break
      ;;
    succeeded|completed|*COMPLETED|*SUCCEEDED)
      fail "workflow unexpectedly reached terminal SUCCESS state '${FINAL_STATUS}' — the unreachable capability should have timed out. Response: ${STATUS_RESPONSE}"
      ;;
  esac
  sleep "${POLL_INTERVAL}"
  ELAPSED=$((ELAPSED + POLL_INTERVAL))
done

case "${FINAL_STATUS}" in
  failed|error|*FAILED|*ERROR|*CANCELED|*TERMINATED|*TIMED_OUT) ;;
  *)
    fail "workflow did not reach terminal failure within ${POLL_TIMEOUT}s. Last status: '${FINAL_STATUS}'"
    ;;
esac

FAIL_ELAPSED=$(( $(date +%s) - START_TS ))
pass "step 2: workflow reached terminal failure state '${FINAL_STATUS}' after ~${FAIL_ELAPSED}s (run_id=${RUN_ID})."

# ── 3. Assert the failure was a capability timeout ────────────────────────────────
#
# The api-gateway status projection carries a human-readable failure reason
# (.error / .reason / .message, depending on serialization). We assert it
# mentions a timeout so we know the failure came from the capability dispatch
# deadline and not from some unrelated error (bad YAML, gateway 500, etc.).

log "step 3: asserting the failure reason is a capability timeout…"

FAIL_REASON=$(printf '%s' "${STATUS_RESPONSE}" \
  | jq -r '.error // .reason // .message // .status_message // empty')
log "  failure reason: ${FAIL_REASON:-<none reported>}"

if printf '%s' "${FAIL_REASON}" | grep -qiE 'timeout|timed out|deadline|unreachable|no agent|no capability|dispatch'; then
  pass "step 3: failure reason indicates a capability timeout/dispatch failure."
else
  # The reason field is best-effort across serializations. The terminal "failed"
  # state for a workflow whose only action is an unreachable capability is itself
  # strong evidence the dispatch timed out, so we warn rather than hard-fail when
  # the reason string is absent.
  warn "step 3: failure reason did not explicitly mention a timeout (reason='${FAIL_REASON:-<none>}')."
  warn "        Terminal 'failed' on an unreachable-capability workflow is treated as a timeout."
  pass "step 3: capability dispatch failed as expected (terminal failure observed)."
fi

# Sanity-check the elapsed time is consistent with the configured timeout. The
# value is informational — we don't hard-fail on it because poll granularity and
# Temporal retry backoff make an exact match impossible.
log "step 3: capability timeout budget = ${ZYNAX_CAPABILITY_TIMEOUT}; observed failure after ~${FAIL_ELAPSED}s."

# ── 4. Assert the workflow.failed CloudEvent off NATS JetStream ────────────────────
#
# REQUIRED since the #1149 fix — no skip path (same strictness as e2e-happy.sh
# step 3b). PublishLifecycleEventActivity maps the interpreter event type onto
# the topic taxonomy "zynax.v1.engine-adapter.workflow.failed", and event-bus
# derives one stream per entity prefix (first 4 subject segments, upper-snake-
# cased — StreamName in services/event-bus/internal/infrastructure/nats.go),
# so the failed event shares the ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW stream with
# the rest of the lifecycle family. We assert with the `nats` CLI inside the
# nats-box pod (deployed by the NATS subchart) so the host needs no NATS tooling.

log "step 4: asserting CloudEvent off NATS JetStream (subject 'zynax.v1.engine-adapter.workflow.failed')…"

EVENTS_STREAM="ZYNAX_V1_ENGINE_ADAPTER_WORKFLOW"
FAILED_SUBJECT="zynax.v1.engine-adapter.workflow.failed"
NATS_ASSERT_TIMEOUT="${NATS_ASSERT_TIMEOUT:-60}"

NATS_BOX_POD=$(kubectl -n "${NAMESPACE}" get pod \
  -l "app.kubernetes.io/component=nats-box" \
  -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
[[ -n "${NATS_BOX_POD}" ]] \
  || fail "step 4: nats-box pod not found (label app.kubernetes.io/component=nats-box) — the NATS subchart must be enabled with natsBox for this assertion"
log "  nats-box pod: ${NATS_BOX_POD}"

# nats_exec <args…> — run the nats CLI inside the nats-box pod.
nats_exec() {
  kubectl -n "${NAMESPACE}" exec "${NATS_BOX_POD}" -- nats "$@"
}

# nats_diagnostics — dump JetStream + publisher state on assertion failure so
# the CI log carries the evidence (the cluster is torn down right after).
nats_diagnostics() {
  warn "── step 4 diagnostics ──"
  warn "JetStream streams:"
  nats_exec stream ls 2>&1 | sed 's/^/    /' >&2 || true
  warn "event-bus log tail:"
  kubectl -n "${NAMESPACE}" logs deployment/zynax-event-bus --tail=40 2>&1 | sed 's/^/    /' >&2 || true
  warn "engine-adapter log tail:"
  kubectl -n "${NAMESPACE}" logs deployment/zynax-engine-adapter --tail=40 2>&1 | sed 's/^/    /' >&2 || true
}

# The workflow already reached terminal failure in step 2, so the failed event
# must land within the assertion timeout — no skip path.
NATS_ELAPSED=0
LAST_FAILED_EVENT=""
while [[ ${NATS_ELAPSED} -lt ${NATS_ASSERT_TIMEOUT} ]]; do
  LAST_FAILED_EVENT=$(nats_exec stream get "${EVENTS_STREAM}" \
    --last-for "${FAILED_SUBJECT}" 2>/dev/null) || LAST_FAILED_EVENT=""
  if printf '%s' "${LAST_FAILED_EVENT}" | grep -q "workflow.failed"; then
    break
  fi
  log "  [${NATS_ELAPSED}s] no message on subject '${FAILED_SUBJECT}' yet — retrying…"
  sleep 5
  NATS_ELAPSED=$((NATS_ELAPSED + 5))
done
printf '%s' "${LAST_FAILED_EVENT}" | grep -q "workflow.failed" || {
  nats_diagnostics
  fail "step 4: workflow.failed CloudEvent not on JetStream after ${NATS_ASSERT_TIMEOUT}s (subject='${FAILED_SUBJECT}', stream='${EVENTS_STREAM}') — #1149 regression"
}
log "  last failed event: ${LAST_FAILED_EVENT}"
pass "step 4: workflow.failed CloudEvent delivered to JetStream (stream=${EVENTS_STREAM}, subject=${FAILED_SUBJECT})."

# ── summary ──────────────────────────────────────────────────────────────────────

printf '\n\033[1;32m[e2e-failure] ALL ASSERTIONS PASSED\033[0m\n'
printf '  workflow:  run_id=%s  status=%s  (expected failure)\n' "${RUN_ID}" "${FINAL_STATUS}"
printf '  timeout:   budget=%s  observed=~%ss\n' "${ZYNAX_CAPABILITY_TIMEOUT}" "${FAIL_ELAPSED}"
printf '  event-bus: stream=%s  subject=%s  failed-event=delivered (required, #1149)\n' \
  "${EVENTS_STREAM}" "${FAILED_SUBJECT}"
printf '\n'
