#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# cluster-down.sh — tear down the kind cluster created by cluster-up.sh.
#
# EPIC G (#770) step 1 / #809. Deletes the ephemeral kind cluster and all of
# its resources. Idempotent: succeeds even if the cluster does not exist.
#
# Usage:
#   scripts/e2e/cluster-down.sh
#
# Environment overrides:
#   CLUSTER_NAME   kind cluster name to delete   (default: zynax-e2e)

set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-zynax-e2e}"

log() { printf '\033[1;34m[cluster-down]\033[0m %s\n' "$*"; }
die() { printf '\033[1;31m[cluster-down]\033[0m %s\n' "$*" >&2; exit 1; }

command -v kind >/dev/null 2>&1 || die "required tool not found on PATH: kind"

if kind get clusters 2>/dev/null | grep -qx "${CLUSTER_NAME}"; then
  log "deleting kind cluster '${CLUSTER_NAME}'…"
  kind delete cluster --name "${CLUSTER_NAME}"
  log "cluster '${CLUSTER_NAME}' deleted."
else
  log "kind cluster '${CLUSTER_NAME}' not found — nothing to tear down (idempotent)."
fi
