#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#
# DEPRECATED: This script is superseded by the images.yaml-based bump flow.
# Use the new flow instead:
#   1. Edit images/images.yaml — update the ci-runner digest field
#   2. make sync-images        — stamp all consumer files from images.yaml
#   3. make check-images       — verify all consumers match images.yaml
#   4. Open PR, CI green, squash-merge, delete branch
#
# This script is kept for backward compatibility and will be removed in M7.
#
# bump-ci-runner.sh — update every ci-runner digest reference in the repo.
#
# Usage:
#   scripts/bump-ci-runner.sh <new-digest>
#   scripts/bump-ci-runner.sh --check <expected-digest>
#
# Arguments:
#   <new-digest>       New digest in the form sha256:<64-hex-chars>
#   --check <digest>   Dry-run: exit 0 if all refs already match <digest>,
#                      exit 1 if any ref is stale (prints differing files)
#
# Files updated (18 occurrences total):
#   config/ci-runner-digest.txt
#   .github/workflows/ci.yml              (8 refs)
#   .github/workflows/pr-checks.yml       (7 refs)
#   .github/workflows/_test-go.yml        (1 ref)
#   .github/workflows/_test-python.yml    (1 ref)
#   .github/workflows/ai-context-budget.yml (1 ref)
#
# The script is idempotent: running it twice with the same digest is a no-op.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

IMAGE_PREFIX="ghcr.io/zynax-io/zynax/ci-runner@"

WORKFLOW_FILES=(
  ".github/workflows/ci.yml"
  ".github/workflows/pr-checks.yml"
  ".github/workflows/_test-go.yml"
  ".github/workflows/_test-python.yml"
  ".github/workflows/ai-context-budget.yml"
)
DIGEST_FILE="config/ci-runner-digest.txt"

usage() {
  echo "Usage: $0 <sha256:hex>" >&2
  echo "       $0 --check <sha256:hex>" >&2
  exit 1
}

# ── argument parsing ───────────────────────────────────────────────────────────

CHECK_MODE=false
if [[ "${1:-}" == "--check" ]]; then
  CHECK_MODE=true
  NEW_DIGEST="${2:-}"
else
  NEW_DIGEST="${1:-}"
fi

[[ -z "$NEW_DIGEST" ]] && usage

if [[ ! "$NEW_DIGEST" =~ ^sha256:[0-9a-f]{64}$ ]]; then
  echo "❌ digest must match sha256:<64 lowercase hex chars>, got: $NEW_DIGEST" >&2
  exit 1
fi

# ── helpers ────────────────────────────────────────────────────────────────────

# Pattern that matches any existing ci-runner digest in workflow files.
# Captures: ghcr.io/zynax-io/zynax/ci-runner@sha256:<64 hex>
CI_RUNNER_PATTERN="${IMAGE_PREFIX}sha256:[0-9a-f]{64}"

stale_files=()

check_file() {
  local rel="$1"
  local file="${REPO_ROOT}/${rel}"
  if grep -qE "${CI_RUNNER_PATTERN}" "$file" 2>/dev/null; then
    if ! grep -qF "${IMAGE_PREFIX}${NEW_DIGEST}" "$file"; then
      stale_files+=("$rel")
    fi
  fi
}

update_file() {
  local rel="$1"
  local file="${REPO_ROOT}/${rel}"
  # Replace any existing ci-runner digest with the new one (portable sed)
  sed -i "s|${IMAGE_PREFIX}sha256:[0-9a-f]*|${IMAGE_PREFIX}${NEW_DIGEST}|g" "$file"
}

# ── check digest file separately (different pattern) ──────────────────────────

DIGEST_PATH="${REPO_ROOT}/${DIGEST_FILE}"
if [[ -f "$DIGEST_PATH" ]]; then
  current=$(grep -E '^sha256:[0-9a-f]{64}$' "$DIGEST_PATH" | head -1 || true)
  if [[ "$current" != "$NEW_DIGEST" ]]; then
    stale_files+=("$DIGEST_FILE")
  fi
fi

# ── check workflow files ───────────────────────────────────────────────────────

for rel in "${WORKFLOW_FILES[@]}"; do
  check_file "$rel"
done

# ── report / exit in check mode ───────────────────────────────────────────────

if $CHECK_MODE; then
  if [[ ${#stale_files[@]} -eq 0 ]]; then
    echo "✅ All ci-runner refs already match ${NEW_DIGEST}"
    exit 0
  else
    echo "❌ Stale ci-runner refs found (run: make bump-ci-runner NEW_DIGEST=${NEW_DIGEST}):" >&2
    for f in "${stale_files[@]}"; do
      echo "   $f" >&2
    done
    exit 1
  fi
fi

# ── apply updates ─────────────────────────────────────────────────────────────

if [[ ${#stale_files[@]} -eq 0 ]]; then
  echo "✅ All refs already up to date — nothing to do."
  exit 0
fi

echo "🔄 Updating ci-runner digest to ${NEW_DIGEST} in ${#stale_files[@]} file(s)…"

# Update digest file
if [[ -f "$DIGEST_PATH" ]]; then
  sed -i "s|^sha256:[0-9a-f]*$|${NEW_DIGEST}|" "$DIGEST_PATH"
  echo "   ✓ ${DIGEST_FILE}"
fi

# Update workflow files
for rel in "${WORKFLOW_FILES[@]}"; do
  file="${REPO_ROOT}/${rel}"
  if [[ -f "$file" ]] && grep -qE "${CI_RUNNER_PATTERN}" "$file" 2>/dev/null; then
    update_file "$rel"
    count=$(grep -cF "${IMAGE_PREFIX}${NEW_DIGEST}" "$file" || true)
    echo "   ✓ ${rel} (${count} ref(s))"
  fi
done

echo "✅ Done. Verify with: scripts/bump-ci-runner.sh --check ${NEW_DIGEST}"
