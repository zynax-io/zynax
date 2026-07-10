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
#   3. idempotency — a reconcile of the unchanged spec starts NO new run;
#   4. admission (ADR-045, M8.G): a CR whose spec.engine is OUTSIDE the
#      namespace allow-list is DENIED at admission by the engine-allowlist
#      ValidatingAdmissionPolicy; an allowed engine admits. Skipped with a
#      warning when the VAP is not installed (admissionPolicy.enabled=false).
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
# Fully-qualified resource name — do NOT shorten to the bare "workflow" alias.
# On the argo leg the Argo Workflows CRD (workflows.argoproj.io) is installed
# alongside the thin Zynax CRD (workflows.zynax.io), so the short name resolves
# ambiguously and kubectl reads the Argo CRD — which never holds a resource
# named "${CR_NAME}". The zynax CR reconciles to Dispatched=True either way, but
# every unqualified poll silently reads the wrong kind and the assertion times
# out (#1620). Pinning the group makes the script correct on both engine legs.
WF_RESOURCE="workflow.zynax.io"

log()  { printf '\033[1;34m[e2e-crd]\033[0m %s\n' "$*"; }
pass() { printf '\033[1;32m[e2e-crd][PASS]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[e2e-crd][FAIL]\033[0m %s\n' "$*" >&2; exit 1; }
require() { command -v "$1" >/dev/null 2>&1 || fail "required tool not found on PATH: $1"; }

require kubectl
require python3

wf() { kubectl -n "${NAMESPACE}" get "${WF_RESOURCE}" "${CR_NAME}" -o "$1" 2>/dev/null || true; }

cleanup() { kubectl -n "${NAMESPACE}" delete "${WF_RESOURCE}" "${CR_NAME}" --ignore-not-found >/dev/null 2>&1 || true; }
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
if [[ -z "${run1}" ]]; then
  # Surface why: the CR's own conditions and the controller's reconcile logs.
  log "CR status on failure: $(wf 'jsonpath={.status}')"
  log "api-gateway controller logs (reconcile lines):"
  kubectl -n "${NAMESPACE}" logs -l app.kubernetes.io/name=zynax-api-gateway --tail=50 2>/dev/null \
    | grep -iE 'workflow|reconcile|controller|apply' || true
  fail "CR never reached Dispatched=True with a runID (compile/submit failed?)"
fi
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
kubectl -n "${NAMESPACE}" annotate "${WF_RESOURCE}" "${CR_NAME}" e2e/poke="$(date +%s)" --overwrite >/dev/null
sleep "${IDEMPOTENCY_WAIT}"
run2=$(wf 'jsonpath={.status.runID}')
gen2=$(wf 'jsonpath={.status.observedGeneration}')
[[ "${run1}" == "${run2}" && "${gen1}" == "${gen2}" ]] \
  || fail "duplicate dispatch on reconcile of an unchanged spec (runID ${run1} -> ${run2})"
pass "idempotent — reconcile of the unchanged CR started no new run"

# ── Admission: engine allow-list VAP (ADR-045, M8.G #1637) ──────────────────
# values-e2e.yaml enables admissionPolicy with allowedEngines [temporal, argo].
# A CR pinning an engine outside that list must be rejected AT ADMISSION —
# kubectl apply itself fails with the policy message; the controller never
# sees the object. An allowed engine admits. Skip (warn) when the VAP is not
# installed so the script still works against a policy-less cluster.
VAP_NAME="zynax-api-gateway-engine-allowlist"
if kubectl get validatingadmissionpolicy "${VAP_NAME}" >/dev/null 2>&1; then
  log "asserting the engine allow-list VAP denies a disallowed spec.engine"
  DENIED_CR_NAME="e2e-crd-denied"
  set +e
  deny_out=$(kubectl -n "${NAMESPACE}" apply -f - 2>&1 <<EOF
apiVersion: zynax.io/v1alpha1
kind: Workflow
metadata:
  name: ${DENIED_CR_NAME}
spec:
  engine: forbidden-engine
  initial_state: greet
  states:
    greet:
      actions:
        - capability: echo
          input:
            message: "must never be admitted"
      "on":
        - event: echo.completed
          goto: done
    done:
      type: terminal
EOF
)
  deny_rc=$?
  set -e
  if [[ ${deny_rc} -eq 0 ]]; then
    kubectl -n "${NAMESPACE}" delete "${WF_RESOURCE}" "${DENIED_CR_NAME}" --ignore-not-found >/dev/null 2>&1 || true
    fail "a CR with spec.engine=forbidden-engine was ADMITTED — the engine-allowlist VAP is not enforcing"
  fi
  echo "${deny_out}" | grep -qi "allow-list" \
    || fail "the denial did not carry the allow-list policy message (got: ${deny_out})"
  pass "disallowed engine denied at admission with the policy message"

  # An ALLOWED engine admits (admission only — delete before it dispatches a
  # run; this leg's active engine may differ from the pinned hint).
  ALLOWED_CR_NAME="e2e-crd-allowed"
  kubectl -n "${NAMESPACE}" apply -f - >/dev/null <<EOF
apiVersion: zynax.io/v1alpha1
kind: Workflow
metadata:
  name: ${ALLOWED_CR_NAME}
spec:
  engine: temporal
  initial_state: greet
  states:
    greet:
      actions:
        - capability: echo
          input:
            message: "admitted by the allow-list"
      "on":
        - event: echo.completed
          goto: done
    done:
      type: terminal
EOF
  kubectl -n "${NAMESPACE}" delete "${WF_RESOURCE}" "${ALLOWED_CR_NAME}" --ignore-not-found >/dev/null 2>&1 || true
  pass "allowed engine admitted by the allow-list VAP"
else
  log "engine-allowlist VAP not installed (admissionPolicy.enabled=false?) — skipping admission assertions"
fi

pass "Workflow CRD reconcile assertion complete"
