#!/usr/bin/env bash
# Pre-commit wrapper: runs golangci-lint in each Go workspace module that has
# staged changes. Avoids the "directory prefix . does not contain modules" error
# that occurs when golangci-lint is invoked from the workspace root.
set -euo pipefail

MODULES=(
  services/api-gateway
  services/engine-adapter
  services/workflow-compiler
  cmd/zynax
  cmd/zynax-ci
  protos/tests
)

ROOT="$(git rev-parse --show-toplevel)"
STAGED=$(git diff --cached --name-only --diff-filter=ACM)

for mod in "${MODULES[@]}"; do
  if echo "$STAGED" | grep -q "^${mod}/"; then
    echo "── golangci-lint: $mod"
    (cd "$ROOT/$mod" && GOWORK=off golangci-lint run --config "$ROOT/tools/golangci-lint.yml" ./...)
  fi
done
