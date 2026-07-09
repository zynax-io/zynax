#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# automation/milestone-env.sh — print the ACTIVE milestone from state/milestone.yaml
# as shell-quoted KEY=value lines (one per line).
#
# Single source of truth consumer for the delivery commands and agents
# (/deliver, /lib:deliver-batch, /lib:deliver-one, .claude/agents/*) — replaces
# the awk block previously duplicated in each command file. state/milestone.yaml
# is written ONLY by /milestone open|close; nothing here mutates it.
#
# Usage:
#   bash automation/milestone-env.sh [path/to/milestone.yaml]
#   # interactive shells may:  eval "$(bash automation/milestone-env.sh)"
#   # sandboxed agents: run it as a single call and parse the printed lines.
#
# Output keys:
#   MILESTONE_NAME MILESTONE_TITLE MILESTONE_NUMBER MILESTONE_VERSION
#   PLANNING_DOC MILESTONE_LABEL GH_MILESTONE
set -euo pipefail

CFG="${1:-state/milestone.yaml}"
if [ "$#" -eq 0 ] && [ ! -f "$CFG" ]; then
  # No explicit path given: resolve relative to the repo root when invoked from
  # elsewhere. An explicitly-passed path that does not exist is an error below —
  # never silently substituted.
  ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
  [ -n "$ROOT" ] && [ -f "$ROOT/state/milestone.yaml" ] && CFG="$ROOT/state/milestone.yaml"
fi
[ -f "$CFG" ] || { echo "milestone-env: $CFG not found" >&2; exit 1; }

# Same extraction logic the delivery commands used inline — first match wins.
# NAME..PLANNING_DOC are scoped to the `active:` block; MILESTONE_LABEL matches the
# first `labels: milestone:` entry in the file (bug-compatible with the inline
# block this replaces — safe because `active:` precedes `history:` in the SSoT).
MILESTONE_NAME=$(awk '/^active:/{f=1} f && /^  name:/{print $2; exit}' "$CFG")
MILESTONE_TITLE=$(awk -F'"' '/^active:/{f=1} f && /^  title:/{print $2; exit}' "$CFG")
MILESTONE_NUMBER=$(awk '/^active:/{f=1} f && /^  github_milestone_number:/{print $2; exit}' "$CFG")
MILESTONE_VERSION=$(awk '/^active:/{f=1} f && /^  version:/{print $2; exit}' "$CFG")
PLANNING_DOC=$(awk '/^active:/{f=1} f && /^  planning_doc:/{print $2; exit}' "$CFG")
MILESTONE_LABEL=$(awk -F'"' '/^    milestone:/{print $2; exit}' "$CFG")
GH_MILESTONE="${MILESTONE_TITLE} (${MILESTONE_NAME})"

[ -n "$MILESTONE_NAME" ] || { echo "milestone-env: no active milestone in $CFG" >&2; exit 1; }

printf 'MILESTONE_NAME=%q\n'    "$MILESTONE_NAME"
printf 'MILESTONE_TITLE=%q\n'   "$MILESTONE_TITLE"
printf 'MILESTONE_NUMBER=%q\n'  "$MILESTONE_NUMBER"
printf 'MILESTONE_VERSION=%q\n' "$MILESTONE_VERSION"
printf 'PLANNING_DOC=%q\n'      "$PLANNING_DOC"
printf 'MILESTONE_LABEL=%q\n'   "$MILESTONE_LABEL"
printf 'GH_MILESTONE=%q\n'      "$GH_MILESTONE"
