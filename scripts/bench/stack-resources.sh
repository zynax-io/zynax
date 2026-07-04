#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# stack-resources.sh — measure a RUNNING Zynax stack's footprint (pods/containers,
# CPU, memory, disk) and the hero workflow's wall-clock, then append one row to a
# markdown comparison table. The data source for the ADR-041 before/after numbers
# (compose vs full-kind vs lean-kind). Read-only against the stack; it never
# brings anything up or tears anything down — point it at a stack that is already
# up (e.g. after `make kind-up PROFILE=lite`).
#
# Usage:
#   scripts/bench/stack-resources.sh --runtime kind    --profile full-kind
#   scripts/bench/stack-resources.sh --runtime kind    --profile lean-kind
#   scripts/bench/stack-resources.sh --runtime compose --profile compose
#
# Options:
#   --runtime kind|compose   how to introspect the stack (default: kind)
#   --profile <label>        row label written to the table (required)
#   --namespace <ns>         kind release namespace (default: zynax)
#   --cluster <name>         kind cluster name (default: zynax-e2e)
#   --out <file>             markdown table to append to
#                            (default: docs/benchmarks/kind-lean-resources.md)
#   --no-workload            skip the demo wall-clock timing (resources only)
#   --workflow <path>        workflow to time (default: spec/workflows/examples/e2e-demo.yaml)
#
# kind CPU/memory "used" needs metrics-server; this script installs it into the
# kind cluster on first run (idempotent) and falls back to summed Pod *requests*
# when `kubectl top` is unavailable (offline) — the column header says which.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

RUNTIME="kind"
PROFILE=""
NAMESPACE="zynax"
CLUSTER_NAME="zynax-e2e"
OUT="${REPO_ROOT}/docs/benchmarks/kind-lean-resources.md"
RUN_WORKLOAD="1"
WORKFLOW_FILE="${REPO_ROOT}/spec/workflows/examples/e2e-demo.yaml"
GW_LOCAL_PORT="${GW_LOCAL_PORT:-18099}"
POLL_TIMEOUT="${POLL_TIMEOUT:-180}"
POLL_INTERVAL="${POLL_INTERVAL:-3}"

# log/warn/die all go to STDERR — measure_*() stdout is captured into the row, so
# any progress text on stdout would corrupt the parsed result.
log()  { printf '\033[1;34m[bench]\033[0m %s\n' "$*" >&2; }
warn() { printf '\033[1;33m[bench]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[bench]\033[0m %s\n' "$*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --runtime)   RUNTIME="$2"; shift 2 ;;
    --profile)   PROFILE="$2"; shift 2 ;;
    --namespace) NAMESPACE="$2"; shift 2 ;;
    --cluster)   CLUSTER_NAME="$2"; shift 2 ;;
    --out)       OUT="$2"; shift 2 ;;
    --no-workload) RUN_WORKLOAD=""; shift ;;
    --workflow)  WORKFLOW_FILE="$2"; shift 2 ;;
    *) die "unknown arg: $1" ;;
  esac
done
[[ -n "${PROFILE}" ]] || die "--profile <label> is required"
case "${RUNTIME}" in kind|compose) ;; *) die "--runtime must be kind|compose" ;; esac

_PF_PIDS=()
cleanup() { for pid in "${_PF_PIDS[@]+"${_PF_PIDS[@]}"}"; do kill "$pid" 2>/dev/null || true; done; }
trap cleanup EXIT

# ── kind: ensure metrics-server (for `kubectl top`) ──────────────────────────────
ensure_metrics_server() {
  kubectl top pods -n kube-system >/dev/null 2>&1 && return 0
  log "installing metrics-server into the kind cluster (for kubectl top)…"
  kubectl apply -f \
    https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml \
    >/dev/null 2>&1 || { warn "metrics-server apply failed (offline?) — will use Pod requests"; return 1; }
  # kind kubelets serve a self-signed cert; metrics-server needs --kubelet-insecure-tls.
  kubectl -n kube-system patch deployment metrics-server --type=json \
    -p '[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]' \
    >/dev/null 2>&1 || true
  kubectl -n kube-system rollout status deployment/metrics-server --timeout=120s >/dev/null 2>&1 || true
  # metrics take ~15s to first populate.
  for _ in $(seq 1 12); do kubectl top pods -n "${NAMESPACE}" >/dev/null 2>&1 && return 0; sleep 5; done
  return 1
}

# Sum Pod CPU(m)/memory(Mi) requests in a namespace — the offline fallback.
sum_pod_requests() {
  local ns="$1"
  kubectl -n "${ns}" get pods -o json 2>/dev/null | python3 -c '
import sys, json, re
def cpu(v):
    if not v: return 0.0
    return float(v[:-1]) if v.endswith("m") else float(v)*1000
def mem(v):
    if not v: return 0.0
    u={"Ki":1/1024,"Mi":1,"Gi":1024}; m=re.match(r"([0-9.]+)([A-Za-z]*)",v)
    return float(m.group(1))*u.get(m.group(2),1/1024/1024)
c=mm=0.0
for p in json.load(sys.stdin).get("items",[]):
    for ct in p["spec"]["containers"]:
        r=ct.get("resources",{}).get("requests",{})
        c+=cpu(r.get("cpu","0")); mm+=mem(r.get("memory","0"))
print(f"{int(c)} {int(mm)}")' 2>/dev/null || echo "0 0"
}

# ── kind measurement ─────────────────────────────────────────────────────────────
measure_kind() {
  local pods cpu_m mem_mi pvc_gi disk_mb src="requests"
  pods="$(kubectl -n "${NAMESPACE}" get pods --no-headers 2>/dev/null | wc -l | tr -d ' ')"

  if kubectl -n "${NAMESPACE}" top pods --no-headers >/dev/null 2>&1; then
    src="used (metrics-server)"
    read -r cpu_m mem_mi < <(kubectl -n "${NAMESPACE}" top pods --no-headers 2>/dev/null | \
      awk '{c+=$2+0; m+=$3+0} END{printf "%d %d\n", c, m}')
  else
    read -r cpu_m mem_mi < <(sum_pod_requests "${NAMESPACE}")
  fi

  # PVC reserved across the whole cluster (Gi).
  pvc_gi="$(kubectl get pvc -A -o json 2>/dev/null | python3 -c '
import sys,json,re
def gi(v):
    u={"Mi":1/1024,"Gi":1,"Ti":1024}; m=re.match(r"([0-9.]+)([A-Za-z]*)",v or "0")
    return float(m.group(1))*u.get(m.group(2),1) if m else 0
print(round(sum(gi(i["spec"]["resources"]["requests"]["storage"]) for i in json.load(sys.stdin).get("items",[])),1))' 2>/dev/null || echo 0)"

  # containerd on-disk across every kind node (images + layers).
  disk_mb=0
  for node in $(kind get nodes --name "${CLUSTER_NAME}" 2>/dev/null); do
    local kb
    kb="$(docker exec "${node}" du -sk /var/lib/containerd 2>/dev/null | awk '{print $1}')"
    [[ -n "${kb}" ]] && disk_mb=$((disk_mb + kb / 1024))
  done

  echo "${pods}|${cpu_m}m (${src})|${mem_mi}Mi (${src})|${pvc_gi}Gi|${disk_mb}MB"
}

# ── compose measurement ──────────────────────────────────────────────────────────
measure_compose() {
  local compose="docker compose -f ${REPO_ROOT}/infra/docker-compose/docker-compose.yml"
  local ids n cpu mem_mb img_mb vol_mb
  ids="$(${compose} ps -q 2>/dev/null)"
  n="$(printf '%s\n' "${ids}" | grep -c . || true)"
  # docker stats: sum CPU% and MEM (convert MiB/GiB → MiB).
  read -r cpu mem_mb < <(docker stats --no-stream --format '{{.CPUPerc}} {{.MemUsage}}' ${ids} 2>/dev/null | \
    awk '{gsub(/%/,"",$1); c+=$1; v=$2; u=$3;
          if(u ~ /GiB/) v=v*1024; m+=v} END{printf "%.0f %d\n", c, m}')
  # image disk for the stack + named-volume disk.
  img_mb="$(${compose} images --format '{{.Size}}' 2>/dev/null | \
    awk '{v=$1; if(v ~ /GB/){gsub(/GB/,"",v); v=v*1024} else gsub(/MB/,"",v); s+=v} END{printf "%d", s}')"
  vol_mb="$(docker system df -v --format '{{json .Volumes}}' 2>/dev/null | python3 -c '
import sys,json,re
try: vols=json.load(sys.stdin)
except Exception: print(0); sys.exit()
def mb(s):
    m=re.match(r"([0-9.]+)([A-Za-z]+)",s or "0B");
    if not m: return 0
    u={"B":1/1024/1024,"kB":1/1024,"KB":1/1024,"MB":1,"GB":1024}; return float(m.group(1))*u.get(m.group(2),0)
print(int(sum(mb(v.get("Size","0B")) for v in vols if "postgres" in v.get("Name","") or "nats" in v.get("Name","") or "certs" in v.get("Name",""))))' 2>/dev/null || echo 0)"
  echo "${n}|${cpu}% (stats)|${mem_mb}Mi (stats)|n/a (named vols)|$((img_mb + vol_mb))MB"
}

# ── workload timing (apply → terminal), run twice; report the 2nd (warm) ─────────
time_workflow() {
  local base="$1" api_key="${2:-}" cold warm
  _one_run() {
    local t0 t1 run_id status auth=()
    [[ -n "${api_key}" ]] && auth=(-H "Authorization: Bearer ${api_key}")
    t0="$(date +%s)"
    run_id="$(curl -sS --fail -X POST "${auth[@]+"${auth[@]}"}" \
      -H 'Content-Type: application/x-yaml' --data-binary "@${WORKFLOW_FILE}" \
      "${base}/api/v1/apply" 2>/dev/null | jq -r '.run_id // empty')"
    [[ -n "${run_id}" ]] || { echo "-1"; return; }
    local deadline=$(( $(date +%s) + POLL_TIMEOUT ))
    while [[ $(date +%s) -lt ${deadline} ]]; do
      status="$(curl -sS "${auth[@]+"${auth[@]}"}" "${base}/api/v1/workflows/${run_id}" 2>/dev/null | jq -r '.status // empty')"
      case "${status}" in
        succeeded|completed|*COMPLETED|*SUCCEEDED) t1="$(date +%s)"; echo "$((t1 - t0))"; return ;;
        failed|error|*FAILED|*ERROR|*CANCELED|*TERMINATED|*TIMED_OUT) echo "-1"; return ;;
      esac
      sleep "${POLL_INTERVAL}"
    done
    echo "-1"
  }
  cold="$(_one_run)"; warm="$(_one_run)"
  echo "${cold}s cold / ${warm}s warm"
}

# ── run ──────────────────────────────────────────────────────────────────────────
command -v python3 >/dev/null 2>&1 || die "python3 required for unit parsing"
log "measuring runtime='${RUNTIME}' profile='${PROFILE}'…"

if [[ "${RUNTIME}" == "kind" ]]; then
  command -v kubectl >/dev/null 2>&1 || die "kubectl required for --runtime kind"
  kubectl config use-context "kind-${CLUSTER_NAME}" >/dev/null 2>&1 || true
  ensure_metrics_server || warn "kubectl top unavailable — CPU/memory shown as summed Pod requests"
  ROW="$(measure_kind)"
  WALL="skipped"
  if [[ -n "${RUN_WORKLOAD}" ]]; then
    command -v jq >/dev/null 2>&1 || die "jq required for workload timing (or pass --no-workload)"
    api_key="$(kubectl -n "${NAMESPACE}" get secret zynax-edge-apikey -o jsonpath='{.data.zynax-cli}' 2>/dev/null | base64 -d || true)"
    kubectl -n "${NAMESPACE}" port-forward svc/zynax-api-gateway "${GW_LOCAL_PORT}:8080" >/dev/null 2>&1 &
    _PF_PIDS+=("$!"); sleep 4
    WALL="$(time_workflow "http://localhost:${GW_LOCAL_PORT}" "${api_key}")"
  fi
else
  command -v docker >/dev/null 2>&1 || die "docker required for --runtime compose"
  ROW="$(measure_compose)"
  WALL="skipped"
  if [[ -n "${RUN_WORKLOAD}" ]]; then
    command -v jq >/dev/null 2>&1 || die "jq required for workload timing (or pass --no-workload)"
    WALL="$(time_workflow "http://localhost:7080" "")"
  fi
fi

IFS='|' read -r N CPU MEM PVC DISK <<<"${ROW}"
log "result: pods/containers=${N} cpu=${CPU} mem=${MEM} pvc=${PVC} disk=${DISK} demo=${WALL}"

# ── append a markdown table row (create the file + header on first write) ────────
mkdir -p "$(dirname "${OUT}")"
if [[ ! -f "${OUT}" ]] || ! grep -q '| Profile | Pods/containers |' "${OUT}" 2>/dev/null; then
  {
    echo "# Lean kind resource comparison (ADR-041) — measured"
    echo ""
    echo "Generated by \`scripts/bench/stack-resources.sh\`. Each row is a stack measured"
    echo "at rest after bring-up; demo time is the echo hero workflow (apply→terminal),"
    echo "run twice (cold/warm). CPU/memory source is noted per cell."
    echo ""
    echo "| Profile | Pods/containers | CPU | Memory | PVC reserved | Image+layer disk | Demo (apply→done) |"
    echo "|---------|-----------------|-----|--------|--------------|------------------|-------------------|"
  } >>"${OUT}"
fi
printf '| %s | %s | %s | %s | %s | %s | %s |\n' \
  "${PROFILE}" "${N}" "${CPU}" "${MEM}" "${PVC}" "${DISK}" "${WALL}" >>"${OUT}"
log "appended row '${PROFILE}' → ${OUT}"
