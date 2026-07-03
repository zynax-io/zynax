#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# kind-demo.sh — one-command kind lifecycle for the first-run golden path.
#
# EPIC #1370 / O22 (#1492), governed by ADR-041 (kind-native unified runtime —
# Docker Compose retired as the primary path). Wraps the existing kind + Helm
# bring-up (scripts/e2e/cluster-up.sh — the single source of bring-up truth) so a
# brand-new user types ONE command and gets, in order:
#
#   1. a kind cluster (create → `kind load` local images → cert-manager →
#      `helm upgrade --install` the zynax-umbrella → wait every Deployment's
#      rollout) — all delegated to cluster-up.sh;
#   2. a default-model pre-flight that reports, with actionable remediation,
#      whether the reference model is available — BEFORE any workflow can fail;
#   3. the hero workflow run against the gateway on http://localhost:8080;
#   4. a "Platform ready" banner — printed ONLY AFTER the rollout actually
#      succeeded (it is gated on cluster-up.sh's rollout wait returning 0), with
#      the gateway URL and the exact next command, wedge-first.
#
# The banner is NEVER a premature "go" signal: cluster-up.sh blocks on
# `kubectl rollout status` for all 7 services (and Temporal / echo-worker), so if
# any Deployment is still coming up this script exits non-zero before the banner.
#
# AC5 (no host-LAN exposure of the model runtime): the kind demo dispatches the
# `echo` capability (echo-worker, in-cluster) — no model runtime is exposed on
# the host. The model pre-flight is host-side and informational; it never opens a
# port. Only the api-gateway NodePort (host 8080 → nodePort 30080) is mapped out.
#
# Usage:
#   scripts/demo/kind-demo.sh                 # full lifecycle + hero run
#   KIND_LOAD_IMAGES=1 scripts/demo/kind-demo.sh   # (set by `make demo`)
#
# Environment overrides (plus everything cluster-up.sh accepts):
#   CLUSTER_NAME    kind cluster name                 (default: zynax-e2e)
#   NAMESPACE       release namespace                 (default: zynax)
#   API_GW_URL      gateway base URL                  (default: http://localhost:8080)
#   DEMO_MODEL      default reference model to check  (default: from the llm-adapter config)
#   WORKFLOW_FILE   hero workflow to run              (default: spec/workflows/examples/e2e-demo.yaml)
#   SKIP_RUN        set to skip the hero run (banner only after bring-up)
#
# Minimum host resources: 4 CPU, 8 GB RAM (see scripts/e2e/README.md).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

CLUSTER_NAME="${CLUSTER_NAME:-zynax-e2e}"
NAMESPACE="${NAMESPACE:-zynax}"
API_GW_URL="${API_GW_URL:-http://localhost:8080}"
WORKFLOW_FILE="${WORKFLOW_FILE:-${REPO_ROOT}/spec/workflows/examples/e2e-demo.yaml}"
# Stack profile passed through to cluster-up.sh (ADR-041): "full" (default,
# prod-mirroring) or "lite" (lean laptop — one in-memory dev Temporal, no
# event-bus/NATS/memory-service). The hero echo workflow runs on both.
PROFILE="${PROFILE:-full}"
# Workflow engine passed through to cluster-up.sh (#1500, the engine-portability
# wedge — #1370 / ADR-041): "temporal" (default) or "argo". argo additionally
# installs the Argo Workflows control plane and deploys the umbrella with
# values-e2e-argo.yaml (cluster-up.sh, ADR-015). The SAME hero workflow runs
# unchanged on either — selection flows through umbrella values, never the
# manifest. Forwarded EXPLICITLY to cluster-up.sh below (no silent env
# inheritance) so `E2E_ENGINE=argo make demo` is a robust, documented contract.
E2E_ENGINE="${E2E_ENGINE:-temporal}"
# Default reference model — read from the single source (the llm-adapter config),
# so the pre-flight and the runtime config never drift. Override with DEMO_MODEL.
MODEL_CONFIG="${REPO_ROOT}/infra/ollama/llm-adapter.config.yaml"
DEMO_MODEL="${DEMO_MODEL:-$(awk '/^[[:space:]]*model:/{print $2; exit}' "${MODEL_CONFIG}" 2>/dev/null)}"
DEMO_MODEL="${DEMO_MODEL:-qwen2.5-coder:3b}"
# Local port the script tunnels the gateway on (a NodePort can be reset by
# kube-proxy on multi-node clusters; a port-forward is environment-independent).
GW_LOCAL_PORT="${GW_LOCAL_PORT:-18080}"
POLL_TIMEOUT="${POLL_TIMEOUT:-120}"
POLL_INTERVAL="${POLL_INTERVAL:-5}"

_PF_PIDS=()

# ── helpers ──────────────────────────────────────────────────────────────────────

log()  { printf '\033[1;34m[demo]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[demo]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[demo]\033[0m %s\n' "$*" >&2; exit 1; }

require() {
  command -v "$1" >/dev/null 2>&1 || die "required tool not found on PATH: $1"
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
      die "port-forward ${resource}:${remote_port} exited: $(cat "${pf_log}")"
    fi
    i=$((i + 1))
    [[ $i -ge ${PF_TIMEOUT} ]] && die "port-forward ${resource}:${remote_port} not ready in ${PF_TIMEOUT}s"
    sleep 1
  done
  rm -f "${pf_log}"
}

api_curl() {
  local method="$1" path="$2"; shift 2
  local auth_args=()
  [[ -n "${ZYNAX_API_KEY:-}" ]] && auth_args=(-H "Authorization: Bearer ${ZYNAX_API_KEY}")
  curl --silent --show-error --fail \
    -X "${method}" "${auth_args[@]+"${auth_args[@]}"}" "$@" "${API_GW_URL}${path}"
}

# ── 0. preflight: tools ──────────────────────────────────────────────────────────

require kind
require kubectl
require helm
require docker

# ── 1. default-model pre-flight (BEFORE bring-up / any workflow can fail, AC3) ───

# Surface the model situation up front with actionable remediation. The kind hero
# workflow uses the in-cluster `echo` capability (no model needed), so a missing
# model never breaks THIS run — but model-backed workflows would 404 mid-run, so
# we tell the user exactly how to make it available now, never as a cryptic later
# failure. The runtime stays in-cluster (AC5): this check never opens a host port.
log "default-model pre-flight: ${DEMO_MODEL}"
if command -v ollama >/dev/null 2>&1; then
  if ollama list 2>/dev/null | awk '{print $1}' | grep -qx "${DEMO_MODEL}"; then
    log "  ✓ default model '${DEMO_MODEL}' is available on the host."
  else
    warn "  ⚠️  default model '${DEMO_MODEL}' is NOT present yet."
    warn "      The kind hero workflow below uses the in-cluster 'echo' capability and runs"
    warn "      WITHOUT a model, so this run is unaffected. To run a model-backed workflow"
    warn "      (e.g. the code-review example) pull it once first — this avoids a mid-run 404:"
    warn "          ollama pull ${DEMO_MODEL}"
  fi
else
  warn "  ⚠️  'ollama' is not on PATH — the kind hero workflow ('echo' capability) needs NO model"
  warn "      and will run fine. For a model-backed workflow, install Ollama and pull the model:"
  warn "          https://ollama.com  →  ollama pull ${DEMO_MODEL}"
fi

# ── 2. bring up the cluster (delegated to the single bring-up source of truth) ───

# cluster-up.sh creates the kind cluster, `kind load`s the local images (we pass
# KIND_LOAD_IMAGES), installs cert-manager + the zynax-umbrella chart, and BLOCKS
# on `kubectl rollout status` for every Zynax Deployment. Its success is the gate
# for the "Platform ready" banner below — we never print it if this returns ≠ 0.
log "bringing up the kind cluster + Zynax umbrella (engine: ${E2E_ENGINE}; wraps scripts/e2e/cluster-up.sh)…"
KIND_LOAD_IMAGES="${KIND_LOAD_IMAGES:-1}" \
CLUSTER_NAME="${CLUSTER_NAME}" \
NAMESPACE="${NAMESPACE}" \
PROFILE="${PROFILE}" \
E2E_ENGINE="${E2E_ENGINE}" \
  "${REPO_ROOT}/scripts/e2e/cluster-up.sh"

# Defence-in-depth: re-assert every Zynax Deployment is actually Available before
# the banner, so the banner can NEVER be a premature "go" signal (AC4). This is a
# fast no-op when cluster-up.sh already waited, but it pins the contract here too.
log "verifying every Zynax Deployment reports a healthy rollout before signalling ready…"
SERVICE_DEPLOYMENTS=(
  zynax-api-gateway zynax-workflow-compiler zynax-engine-adapter
  zynax-task-broker zynax-agent-registry
)
# event-bus + memory-service exist in the full profile only (the lean profile
# disables them via values-lite.yaml) — only assert them when PROFILE != lite.
if [[ "${PROFILE}" != "lite" ]]; then
  SERVICE_DEPLOYMENTS+=(zynax-event-bus zynax-memory-service)
fi
for dep in "${SERVICE_DEPLOYMENTS[@]}"; do
  kubectl -n "${NAMESPACE}" rollout status "deployment/${dep}" --timeout=120s \
    || die "deployment ${dep} is not ready — NOT printing the ready banner (platform still coming up)"
done
log "all Zynax Deployments are Available."

# ── 3. run the hero workflow against the gateway (localhost:8080) ────────────────

# The opposite engine, for the wedge-switch hint in the banner below: the SAME
# workflow runs unchanged on the other engine — just re-run with E2E_ENGINE flipped
# (the engine-portability wedge, #1500 / #1370). Always defined (even with SKIP_RUN).
if [[ "${E2E_ENGINE}" == "argo" ]]; then
  OTHER_ENGINE="temporal"
else
  OTHER_ENGINE="argo"
fi

RAN_RESULT=""
if [[ -z "${SKIP_RUN:-}" ]]; then
  require curl
  require jq
  # Resolve the gateway bearer key cluster-up.sh provisioned (avoids a 401).
  if [[ -z "${ZYNAX_API_KEY:-}" ]]; then
    ZYNAX_API_KEY="$(kubectl -n "${NAMESPACE}" get secret zynax-gw-api-key \
      -o jsonpath='{.data.api-key}' 2>/dev/null | base64 -d || true)"
  fi
  # Tunnel the gateway. The kind NodePort maps host 8080 → 30080, but a
  # port-forward is environment-independent (kube-proxy can reset the NodePort on
  # a multi-node cluster). Honour a caller-supplied non-default API_GW_URL.
  if [[ "${API_GW_URL}" == "http://localhost:8080" ]]; then
    port_forward "svc/zynax-api-gateway" "${GW_LOCAL_PORT}" 8080
    API_GW_URL="http://localhost:${GW_LOCAL_PORT}"
  fi
  log "running the hero workflow (${WORKFLOW_FILE##*/}) on the '${E2E_ENGINE}' engine via api-gateway at ${API_GW_URL}…"
  APPLY_RESPONSE="$(api_curl POST /api/v1/apply \
    -H "Content-Type: application/x-yaml" \
    --data-binary "@${WORKFLOW_FILE}" 2>&1)" \
    || die "POST /api/v1/apply failed — is api-gateway reachable? Response: ${APPLY_RESPONSE}"
  RUN_ID="$(printf '%s' "${APPLY_RESPONSE}" | jq -r '.run_id // empty')"
  [[ -n "${RUN_ID}" ]] || die "apply response had no run_id: ${APPLY_RESPONSE}"
  log "  submitted run_id=${RUN_ID} — polling for a terminal state (timeout ${POLL_TIMEOUT}s)…"
  deadline=$(( $(date +%s) + POLL_TIMEOUT ))
  status=""
  while [[ $(date +%s) -lt ${deadline} ]]; do
    # The api-gateway returns the WorkflowStatus proto enum under `.status`
    # (e.g. WORKFLOW_STATUS_COMPLETED) and accepts lowercase aliases — match the
    # same set as scripts/e2e/e2e-happy.sh so the two stay in lockstep.
    status="$(api_curl GET "/api/v1/workflows/${RUN_ID}" 2>/dev/null | jq -r '.status // empty' || true)"
    case "${status}" in
      succeeded|completed|*COMPLETED|*SUCCEEDED) RAN_RESULT="succeeded"; break ;;
      failed|error|*FAILED|*ERROR|*CANCELED|*TERMINATED|*TIMED_OUT)
        die "hero workflow reached terminal failure state '${status}' (run_id=${RUN_ID})" ;;
    esac
    sleep "${POLL_INTERVAL}"
  done
  [[ "${RAN_RESULT}" == "succeeded" ]] \
    || die "hero workflow did not reach success within ${POLL_TIMEOUT}s (last status: '${status:-unknown}')"
  log "  ✓ hero workflow reached terminal success on the '${E2E_ENGINE}' engine ('${status}', run_id=${RUN_ID})."
fi

# ── 4. "Platform ready" banner — ONLY reached after rollout + (optional) run ─────

# wedge-first copy (ADR-041 / #1492 AC6): lead with engine portability.
cat <<'BANNER'

╔══════════════════════════════════════════════════════════════════════════════╗
║  ✅  Platform ready                                                           ║
║                                                                              ║
║  Write your workflow ONCE — run it on Temporal OR Argo, on the same kind     ║
║  cluster that mirrors production. (Switch engines: E2E_ENGINE=argo make demo)║
║                                                                              ║
║  Gateway:  http://localhost:8080                                             ║
╚══════════════════════════════════════════════════════════════════════════════╝
BANNER

echo "  Next, run a workflow against the live platform:"
echo ""
echo "      zynax --api-url http://localhost:8080 apply spec/workflows/examples/e2e-demo.yaml"
echo ""
echo "  More:"
echo "    • Engine-portability:  E2E_ENGINE=${OTHER_ENGINE} make demo   (same workflow, ${OTHER_ENGINE} engine — the wedge)"
echo "    • Inspect a run:        zynax --api-url http://localhost:8080 logs <run-id> --follow"
echo "    • Tear it all down:     make kind-down"
echo ""
[[ "${RAN_RESULT}" == "succeeded" ]] \
  && echo "  (the hero workflow already ran to 'succeeded' above — the platform is live.)"
echo ""
