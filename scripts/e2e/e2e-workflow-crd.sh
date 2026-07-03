#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# e2e-workflow-crd.sh — assert the thin Workflow CRD front-end (ADR-043, M8.E).
#
# kubectl apply a Workflow custom resource; the api-gateway's embedded controller
# (crdController.enabled in values-e2e.yaml) reconciles it through the SAME
# compile->submit path the REST /api/v1/apply uses. The assertions:
#   1. status reaches Dispatched=True with a runID (reconcile -> dispatched run);
#   2. status is a thin mirror only — no run state leaks into the CR (ADR-040 §3);
#   3. idempotency — a reconcile of the unchanged spec starts NO new run.
#
# Runs on both engine legs (temporal, argo): no engine is pinned in the CR, so
# the run uses whichever engine the leg deployed.
#
# Requires a running kind cluster from cluster-up.sh with crdController enabled.
#
# Environment overrides:
#   CLUSTER_NAME   kind cluster name              (default: zynax-e2e)
#   NAMESPACE      release namespace              (default: zynax)
#   POLL_TIMEOUT   max seconds to wait            (default: 90)
#   POLL_INTERVAL  seconds between status polls   (default: 3)
#   IDEMPOTENCY_WAIT  seconds to observe a poked reconcile (default: 8)
#
# Exit codes: 0 all assertions passed; 1 an assertion failed / tool missing.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

NAMESPACE="${NAMESPACE:-zynax}"
POLL_TIMEOUT="${POLL_TIMEOUT:-90}"
POLL_INTERVAL="${POLL_INTERVAL:-3}"
IDEMPOTENCY_WAIT="${IDEMPOTENCY_WAIT:-8}"
CR_FILE="${REPO_ROOT}/scripts/e2e/manifests/workflow-cr.yaml"
CR_NAME="e2e-crd-echo"

log()  { printf '\033[1;34m[e2e-crd]\033[0m %s\n' "$*"; }
pass() { printf '\033[1;32m[e2e-crd][PASS]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[e2e-crd][FAIL]\033[0m %s\n' "$*" >&2; exit 1; }
require() { command -v "$1" >/dev/null 2>&1 || fail "required tool not found on PATH: $1"; }

require kubectl
require python3

wf() { kubectl -n "${NAMESPACE}" get workflow "${CR_NAME}" -o "$1" 2>/dev/null || true; }

cleanup() { kubectl -n "${NAMESPACE}" delete workflow "${CR_NAME}" --ignore-not-found >/dev/null 2>&1 || true; }
trap cleanup EXIT

# The CRD must be installed — it ships in the api-gateway chart crds/ and is only
# useful with crdController.enabled. Its absence means the front-end is off.
kubectl get crd workflows.zynax.io >/dev/null 2>&1 \
  || fail "Workflow CRD not installed — is crdController.enabled set in values-e2e.yaml?"

log "applying Workflow CR (${CR_NAME}) into namespace ${NAMESPACE}"
kubectl -n "${NAMESPACE}" apply -f "${CR_FILE}"

log "polling for Dispatched=True + runID (<=${POLL_TIMEOUT}s)"
run1=""
deadline=$(( $(date +%s) + POLL_TIMEOUT ))
while [[ $(date +%s) -lt ${deadline} ]]; do
  disp=$(wf 'jsonpath={.status.conditions[?(@.type=="Dispatched")].status}')
  run1=$(wf 'jsonpath={.status.runID}')
  if [[ "${disp}" == "True" && -n "${run1}" ]]; then break; fi
  sleep "${POLL_INTERVAL}"
done
[[ -n "${run1}" ]] || fail "CR never reached Dispatched=True with a runID (compile/submit failed?)"
pass "reconciled to a dispatched run: runID=${run1} engine=$(wf 'jsonpath={.status.engine}')"

# Thin-status contract: only the mirror keys may appear — run state stays in the engine.
keys=$(wf 'json' | python3 -c 'import json,sys; d=json.load(sys.stdin); print(" ".join((d.get("status") or {}).keys()))')
read -ra key_arr <<< "${keys}"
for k in "${key_arr[@]}"; do
  case "${k}" in
    observedGeneration|workflowID|runID|engine|conditions) ;;
    *) fail "status carries a non-thin key '${k}' — run state must stay in the engine" ;;
  esac
done
pass "status is a thin mirror (keys: ${keys})"

# Idempotency: annotate to trigger a reconcile (metadata change, generation
# unchanged) and assert no new run is dispatched.
gen1=$(wf 'jsonpath={.status.observedGeneration}')
kubectl -n "${NAMESPACE}" annotate workflow "${CR_NAME}" e2e/poke="$(date +%s)" --overwrite >/dev/null
sleep "${IDEMPOTENCY_WAIT}"
run2=$(wf 'jsonpath={.status.runID}')
gen2=$(wf 'jsonpath={.status.observedGeneration}')
[[ "${run1}" == "${run2}" && "${gen1}" == "${gen2}" ]] \
  || fail "duplicate dispatch on reconcile of an unchanged spec (runID ${run1} -> ${run2})"
pass "idempotent — reconcile of the unchanged CR started no new run"

pass "Workflow CRD reconcile assertion complete"
