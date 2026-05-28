#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# run-go-svc-loop.sh <base-dir> <space-separated-list> -- cmd [args...]
#
# Runs <cmd> for each module in <base-dir>/<name> that is marked as changed.
# Change detection: env var <NAME_UPPER>_CHANGED must be "true".
# GOWORK=off is set before each command (ADR-017).
# Exits 1 if any command fails, 0 otherwise.
#
# Example:
#   run-go-svc-loop.sh services/ "$SERVICE_LIST" -- golangci-lint run ./... --config ../../tools/golangci-lint.yml
#
set -euo pipefail

BASE_DIR="$1"; shift
LIST="$1"; shift
if [ "$1" = "--" ]; then shift; fi
CMD=("$@")

ROOT_DIR="${GITHUB_WORKSPACE:-$(pwd)}"
failed=false

for name in $LIST; do
  mod_file="${ROOT_DIR}/${BASE_DIR}/${name}/go.mod"
  [ -f "$mod_file" ] || continue

  var="$(echo "$name" | tr '-' '_' | tr '[:lower:]' '[:upper:]')_CHANGED"
  if [ "${!var:-false}" != "true" ]; then
    echo "── ${CMD[*]}: ${BASE_DIR}/${name} [SKIPPED — not changed]"
    continue
  fi

  echo "── ${BASE_DIR}/${name}"
  pushd "${ROOT_DIR}/${BASE_DIR}/${name}" > /dev/null
  GOWORK=off "${CMD[@]}" || failed=true
  popd > /dev/null
done

if $failed; then
  exit 1
fi
