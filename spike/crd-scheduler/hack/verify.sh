#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# ADR-039 M7 spike runtime proof on KIND. Proves, against a real API server:
#   1. happy path        — SelectAgent returns a ready agent
#   2. scoring           — the lower-latency ready agent wins
#   3. stale-liveness fix — the not-ready "dead" agent is never selected
#   4. degradation       — Prometheus-down still returns a ready agent (no failure)
#   5. resync-on-restart — kill the scheduler; the index rebuilds from the API server
#
# Throwaway harness. Requires: kind, kubectl, go (run with GOWORK=off), curl, jq.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLUSTER="adr039-spike"
ADDR="127.0.0.1:8088"
POC_PID=""
PASS=0; FAIL=0

log()  { printf '\n\033[1;36m== %s\033[0m\n' "$*"; }
ok()   { printf '  \033[1;32mPASS\033[0m %s\n' "$*"; PASS=$((PASS+1)); }
bad()  { printf '  \033[1;31mFAIL\033[0m %s\n' "$*"; FAIL=$((FAIL+1)); }

cleanup() {
  [[ -n "$POC_PID" ]] && kill "$POC_PID" 2>/dev/null || true
  kind delete cluster --name "$CLUSTER" >/dev/null 2>&1 || true
}
trap cleanup EXIT

start_poc() {
  GOWORK=off go run ./cmd/poc --addr "$ADDR" >/tmp/adr039-poc.log 2>&1 &
  POC_PID=$!
  for _ in $(seq 1 60); do
    curl -sf "http://$ADDR/healthz" >/dev/null 2>&1 && return 0
    sleep 1
  done
  echo "poc did not become healthy; log:" >&2; cat /tmp/adr039-poc.log >&2; return 1
}
stop_poc() { kill "$POC_PID" 2>/dev/null || true; wait "$POC_PID" 2>/dev/null || true; POC_PID=""; }

cd "$HERE"

log "create KIND cluster"
kind create cluster --name "$CLUSTER" >/dev/null

log "apply CRD + samples"
kubectl apply -f config/crd/agents.zynax.io.yaml >/dev/null
kubectl wait --for=condition=Established crd/agents.zynax.io --timeout=60s >/dev/null
kubectl apply -f config/samples/agents.yaml >/dev/null
# Mark the two healthy agents ready via the status subresource; leave reviewer-dead not-ready.
kubectl patch agent reviewer-fast --subresource=status --type=merge -p '{"status":{"ready":true,"replicas":2}}' >/dev/null
kubectl patch agent reviewer-slow --subresource=status --type=merge -p '{"status":{"ready":true,"replicas":1}}' >/dev/null

log "start scheduler PoC"
start_poc

log "1/2. happy path + scoring (lower latency wins)"
sleep 2
SEL=$(curl -s "http://$ADDR/select?cap=review")
echo "  -> $SEL"
[[ "$(echo "$SEL" | jq -r .chosen)" == "reviewer-fast" ]] && ok "scoring picked reviewer-fast (80ms over 400ms)" || bad "expected reviewer-fast"
[[ "$(echo "$SEL" | jq -r .prometheus_consulted)" == "true" ]] && ok "prometheus consulted" || bad "expected prometheus_consulted=true"

log "3. stale-liveness fix (reviewer-dead has lowest latency but ready=false)"
[[ "$(echo "$SEL" | jq -r .chosen)" != "reviewer-dead" ]] && ok "not-ready reviewer-dead was skipped" || bad "selected a not-ready agent"

log "4. degradation (Prometheus down)"
DEG=$(curl -s "http://$ADDR/select?cap=review&fail=1")
echo "  -> $DEG"
[[ "$(echo "$DEG" | jq -r .prometheus_consulted)" == "false" ]] && ok "degraded mode flagged" || bad "expected prometheus_consulted=false"
CH=$(echo "$DEG" | jq -r .chosen)
[[ "$CH" == "reviewer-fast" || "$CH" == "reviewer-slow" ]] && ok "degraded still returned a ready agent ($CH)" || bad "degraded returned no/dead agent"

log "5. resync-on-restart (kill scheduler, restart, index rebuilds from API server)"
BEFORE=$(curl -s "http://$ADDR/index" | jq -r .agents)
echo "  index before restart: $BEFORE agents"
stop_poc
start_poc
sleep 2
AFTER=$(curl -s "http://$ADDR/index" | jq -r .agents)
echo "  index after restart:  $AFTER agents (zero persisted state)"
[[ "$AFTER" == "$BEFORE" && "$AFTER" == "3" ]] && ok "index rebuilt to $AFTER from the API server" || bad "resync mismatch (before=$BEFORE after=$AFTER)"
SEL2=$(curl -s "http://$ADDR/select?cap=review" | jq -r .chosen)
[[ "$SEL2" == "reviewer-fast" ]] && ok "SelectAgent works immediately after restart" || bad "post-restart select failed ($SEL2)"

log "RESULT: $PASS passed, $FAIL failed"
[[ "$FAIL" == "0" ]]
