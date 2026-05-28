#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# bdd-select-packages.sh
#
# Prints the godog BDD packages to run based on changed proto files.
# Prints "ALL" if full suite should run. Prints "" if nothing should run.
#
# Env vars consumed:
#   BASE  — base commit SHA
#   HEAD  — head commit SHA
#   EVENT — github.event_name ("pull_request" | "push" | other)
#
set -euo pipefail

declare -A PKG_MAP
PKG_MAP["agent.proto"]="agent_service"
PKG_MAP["agent_registry.proto"]="agent_registry_service"
PKG_MAP["cloudevents.proto"]="cloudevents_envelope"
PKG_MAP["engine_adapter.proto"]="engine_adapter_service"
PKG_MAP["event_bus.proto"]="event_bus_service"
PKG_MAP["memory.proto"]="memory_service"
PKG_MAP["task_broker.proto"]="task_broker_service"
PKG_MAP["workflow_compiler.proto"]="workflow_compiler_service"

if ! changed=$(git diff --name-only "${BASE}..${HEAD}" -- 'protos/' 2>/tmp/bdd-diff-err.txt); then
  cat /tmp/bdd-diff-err.txt >&2 || true
  echo "ALL"
  exit 0
fi

# Shared test infrastructure → full suite
if echo "$changed" | grep -qE "^protos/tests/(go\.(mod|sum)|testserver/)"; then
  echo "ALL"
  exit 0
fi

pkgs=""
while IFS= read -r f; do
  [ -z "$f" ] && continue
  case "$f" in
    protos/zynax/v1/*.proto)
      b=$(basename "$f")
      pkg="${PKG_MAP[$b]:-}"
      [ -n "$pkg" ] && pkgs="$pkgs $pkg"
      ;;
    protos/tests/features/*)
      echo "ALL"
      exit 0
      ;;
    protos/tests/*/*)
      pkg=$(echo "$f" | cut -d/ -f3)
      [ "$pkg" != "features" ] && pkgs="$pkgs $pkg"
      ;;
  esac
done <<< "$changed"

# Deduplicate — guard avoids grep exiting 1 on empty input (pipefail)
if [ -n "$pkgs" ]; then
  pkgs=$(echo "$pkgs" | tr ' ' '\n' | sort -u | grep -v '^$' | tr '\n' ' ' | xargs)
fi
echo "${pkgs:-}"
