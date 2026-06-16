#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# bench-regression.sh — run domain benchmarks and gate on >20% regression vs baseline.
#
# Usage:
#   tools/ci/bench-regression.sh <baseline.txt> <svc> [<svc> ...]
#
# For each service it runs the domain benchmarks (-bench=. -benchmem -count=COUNT,
# GOWORK=off per ADR-017), concatenates the results, and compares them against the
# committed baseline with benchstat. A benchmark whose time/op regresses by more
# than THRESHOLD_PCT (default 20%) is flagged.
#
# Fail-open policy (EPIC R canvas O1 safeguard): the gate WARNS but does not fail
# the build until a stable baseline is established. Set BENCH_GATE_ENFORCE=true to
# make a flagged regression exit non-zero. Until then regressions are reported and
# the script exits 0.
#
# Env knobs:
#   THRESHOLD_PCT       regression threshold percent (default 20)
#   BENCH_COUNT         -count for go test -bench (default 3)
#   BENCH_GATE_ENFORCE  "true" to fail on regression; anything else = warn-only
set -euo pipefail

BASELINE="${1:?usage: bench-regression.sh <baseline.txt> <svc>...}"; shift
SERVICES=("$@")
if [ "${#SERVICES[@]}" -eq 0 ]; then
  echo "error: no services given" >&2
  exit 2
fi

THRESHOLD_PCT="${THRESHOLD_PCT:-20}"
BENCH_COUNT="${BENCH_COUNT:-3}"
ENFORCE="${BENCH_GATE_ENFORCE:-false}"
ROOT_DIR="${GITHUB_WORKSPACE:-$(pwd)}"

if [ ! -f "$ROOT_DIR/$BASELINE" ]; then
  echo "error: baseline $BASELINE not found — run 'make bench' and commit it" >&2
  exit 2
fi

if ! command -v benchstat >/dev/null 2>&1; then
  echo "error: benchstat not on PATH — install golang.org/x/perf/cmd/benchstat" >&2
  exit 2
fi

NEW="$(mktemp)"
trap 'rm -f "$NEW"' EXIT

# benchstat ignores comment/non-benchmark lines, so the baseline header is harmless.
echo "📊 Running benchmarks for: ${SERVICES[*]}"
for svc in "${SERVICES[@]}"; do
  dir="$ROOT_DIR/services/$svc"
  [ -f "$dir/go.mod" ] || { echo "── skip services/$svc (no go.mod)"; continue; }
  echo "── bench: services/$svc"
  ( cd "$dir" && GOWORK=off go test ./internal/domain/... -run='^$' \
      -bench=. -benchmem -count="$BENCH_COUNT" ) | tee -a "$NEW"
done

echo
echo "📈 benchstat baseline → new:"
# benchstat exits 0 even when deltas exist; capture its table for parsing + display.
REPORT="$(benchstat "$ROOT_DIR/$BASELINE" "$NEW" || true)"
echo "$REPORT"

# Parse the delta column. benchstat prints a "vs base" percentage like "+24.10%"
# on rows that changed; "~" means no statistically significant change. A positive
# percentage on a time/op (sec/op) metric is a regression.
regressed=0
while IFS= read -r line; do
  # Match a signed percentage token, e.g. +24.10% or -5.00%.
  pct="$(printf '%s\n' "$line" | grep -oE '[+-][0-9]+(\.[0-9]+)?%' | head -n1 || true)"
  [ -n "$pct" ] || continue
  num="${pct%\%}"          # strip trailing %
  sign="${num:0:1}"
  [ "$sign" = "+" ] || continue   # only positive deltas (slower) are regressions
  mag="${num#+}"
  # Integer-compare the magnitude against the threshold (awk for float safety).
  if awk "BEGIN{exit !($mag > $THRESHOLD_PCT)}"; then
    echo "⚠️  REGRESSION: $line  (>${THRESHOLD_PCT}%)"
    regressed=1
  fi
done <<< "$REPORT"

if [ "$regressed" -eq 0 ]; then
  echo "✅ No benchmark regressed beyond ${THRESHOLD_PCT}%."
  exit 0
fi

if [ "$ENFORCE" = "true" ]; then
  echo "❌ Benchmark regression gate ENFORCED — failing build."
  exit 1
fi

echo "⚠️  Benchmark regression detected, but gate is in fail-open mode"
echo "    (baseline not yet stabilised over 3 runs). Set BENCH_GATE_ENFORCE=true to block."
exit 0
