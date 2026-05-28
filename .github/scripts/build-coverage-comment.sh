#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# build-coverage-comment.sh
#
# Reads /tmp/coverage-results.txt and writes /tmp/coverage-comment.md.
# Format of results file: type|name|pkg_label|pct (e.g. service|api-gateway||91.2)
#
# Env vars consumed (set by ci.yml via GITHUB_ENV or step env:):
#   RUN_URL, SHA
#   COVERAGE_DOMAIN_GATE, COVERAGE_ADAPTER_GATE, COVERAGE_CLI_ZYNAX_GATE,
#   COVERAGE_CLI_ZYNAX_CI_GATE, COVERAGE_PYTHON_GATE
#
set -euo pipefail

RESULTS="${RESULTS:-/tmp/coverage-results.txt}"
OUT="${OUT:-/tmp/coverage-comment.md}"

gate_icon() {
  local pct="$1" gate="$2"
  awk -v p="${pct:-0}" -v g="$gate" 'BEGIN{print (p+0 >= g+0) ? "✅" : "❌"}'
}

{
  echo "<!-- zynax-coverage-report -->"
  echo "## Coverage Report"
  echo ""

  if grep -q "^service|" "$RESULTS" 2>/dev/null; then
    echo "### Go services — \`internal/domain\` (gate ≥ ${COVERAGE_DOMAIN_GATE:-90}%)"
    echo ""
    echo "| Service | Coverage | Gate |"
    echo "|---------|----------|------|"
    grep "^service|" "$RESULTS" | while IFS='|' read -r _ name _pkg pct; do
      echo "| \`services/${name}\` | **${pct}%** | $(gate_icon "$pct" "${COVERAGE_DOMAIN_GATE:-90}") |"
    done
    echo ""
  fi

  if grep -q "^adapter|" "$RESULTS" 2>/dev/null; then
    echo "### Go adapters (gate ≥ ${COVERAGE_ADAPTER_GATE:-85}%)"
    echo ""
    echo "| Adapter | Coverage | Gate |"
    echo "|---------|----------|------|"
    grep "^adapter|" "$RESULTS" | while IFS='|' read -r _ name _ pct; do
      echo "| \`agents/adapters/${name}\` | **${pct}%** | $(gate_icon "$pct" "${COVERAGE_ADAPTER_GATE:-85}") |"
    done
    echo ""
  fi

  if grep -q "^cli|" "$RESULTS" 2>/dev/null; then
    echo "### CLI tools (gate ≥ ${COVERAGE_CLI_ZYNAX_GATE:-79}% zynax / ${COVERAGE_CLI_ZYNAX_CI_GATE:-80}% zynax-ci)"
    echo ""
    echo "| Tool | Coverage | Gate |"
    echo "|------|----------|------|"
    grep "^cli|" "$RESULTS" | while IFS='|' read -r _ name _ pct; do
      if [[ "$name" == "zynax" ]]; then g="${COVERAGE_CLI_ZYNAX_GATE:-79}"; else g="${COVERAGE_CLI_ZYNAX_CI_GATE:-80}"; fi
      echo "| \`cmd/${name}\` | **${pct}%** | $(gate_icon "$pct" "$g") |"
    done
    echo ""
  fi

  if grep -q "^python|" "$RESULTS" 2>/dev/null; then
    echo "### Python (gate ≥ ${COVERAGE_PYTHON_GATE:-90}%)"
    echo ""
    echo "| Module | Coverage | Gate |"
    echo "|--------|----------|------|"
    grep "^python|" "$RESULTS" | while IFS='|' read -r _ name _ pct; do
      echo "| \`agents/${name}\` | **${pct}%** | $(gate_icon "$pct" "${COVERAGE_PYTHON_GATE:-90}") |"
    done
    echo ""
  fi

  if ! grep -q "^" "$RESULTS" 2>/dev/null || [ ! -s "$RESULTS" ]; then
    echo "_No coverage data collected — tests may have been skipped._"
    echo ""
  fi

  echo "<sub>Run [#${RUN_NUMBER:-?}](${RUN_URL:-}) · \`${SHA:0:7}\`</sub>"
} > "$OUT"
