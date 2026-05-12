#!/usr/bin/env bash
# Pre-commit wrapper: runs golangci-lint in each Go module that has staged changes.
#
# Workspace modules are auto-discovered from go.work so new adapters/services are
# linted without needing manual updates here. Standalone modules (not in go.work)
# are listed explicitly. Generated stubs (protos/generated/) are always skipped.
set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
STAGED=$(git diff --cached --name-only --diff-filter=ACM)

# Auto-discover from go.work; exclude generated stubs (never hand-edited).
WORKSPACE_MODULES=$(grep -E '^\s+\.' "$ROOT/go.work" | sed 's|^\s*\./||' | grep -v '^protos/generated')

# Standalone modules not in go.work but still subject to linting.
STANDALONE_MODULES=(
  cmd/zynax
  cmd/zynax-ci
  protos/tests
)

for mod in $WORKSPACE_MODULES "${STANDALONE_MODULES[@]}"; do
  if echo "$STAGED" | grep -q "^${mod}/"; then
    echo "── golangci-lint: $mod"
    (cd "$ROOT/$mod" && GOWORK=off golangci-lint run --config "$ROOT/tools/golangci-lint.yml" ./...)
  fi
done
