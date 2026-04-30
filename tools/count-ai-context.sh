#!/bin/sh
# Counts lines in all AI context files (CLAUDE.md, AGENTS.md files, ai-assistant-setup.md).
# Always exits 0 — non-blocking advisory only.
set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

# Per-file thresholds
THRESHOLD_CLAUDE=200
THRESHOLD_ROOT_AGENTS=300
THRESHOLD_AI_SETUP=150
THRESHOLD_SERVICE_AGENTS=150
THRESHOLD_TOTAL=2000

total=0
warnings=0

print_row() {
  file="$1"
  lines="$2"
  threshold="$3"
  if [ "$lines" -gt "$threshold" ]; then
    status="WARN"
    warnings=$((warnings + 1))
  else
    status="OK"
  fi
  printf "| %-55s | %5d | %5d | %-4s |\n" "$file" "$lines" "$threshold" "$status"
}

echo "| File                                                    | Lines | Limit | Status |"
echo "|----------------------------------------------------------|-------|-------|--------|"

# CLAUDE.md (root only)
if [ -f "CLAUDE.md" ]; then
  n=$(wc -l < "CLAUDE.md")
  total=$((total + n))
  print_row "CLAUDE.md" "$n" "$THRESHOLD_CLAUDE"
fi

# Root AGENTS.md
if [ -f "AGENTS.md" ]; then
  n=$(wc -l < "AGENTS.md")
  total=$((total + n))
  print_row "AGENTS.md" "$n" "$THRESHOLD_ROOT_AGENTS"
fi

# docs/ai-assistant-setup.md
if [ -f "docs/ai-assistant-setup.md" ]; then
  n=$(wc -l < "docs/ai-assistant-setup.md")
  total=$((total + n))
  print_row "docs/ai-assistant-setup.md" "$n" "$THRESHOLD_AI_SETUP"
fi

# Per-directory AGENTS.md files (all except root)
find . -name "AGENTS.md" ! -path "./AGENTS.md" | sort | while IFS= read -r f; do
  rel="${f#./}"
  n=$(wc -l < "$f")
  # Use a temp file to accumulate total and warnings across subshell boundary
  printf "%s %d\n" "$rel" "$n" >> /tmp/ai_context_sub.$$
  print_row "$rel" "$n" "$THRESHOLD_SERVICE_AGENTS"
done

# Re-read sub-files to get total (find+while runs in subshell)
if [ -f "/tmp/ai_context_sub.$$" ]; then
  while read -r _rel n; do
    total=$((total + n))
    if [ "$n" -gt "$THRESHOLD_SERVICE_AGENTS" ]; then
      warnings=$((warnings + 1))
    fi
  done < "/tmp/ai_context_sub.$$"
  rm -f "/tmp/ai_context_sub.$$"
fi

echo "|----------------------------------------------------------|-------|-------|--------|"

if [ "$total" -gt "$THRESHOLD_TOTAL" ]; then
  total_status="WARN"
  warnings=$((warnings + 1))
else
  total_status="OK"
fi
printf "| %-55s | %5d | %5d | %-4s |\n" "TOTAL" "$total" "$THRESHOLD_TOTAL" "$total_status"
echo ""

if [ "$warnings" -gt 0 ]; then
  echo "WARNING: $warnings file(s) exceed their line threshold."
  echo "Consider trimming AI context files — smaller budgets improve signal density."
else
  echo "All AI context files are within budget."
fi

# Always exit 0 — this check is advisory only
exit 0
