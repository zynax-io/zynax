#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# report-image-meta.sh — thin GHCR primitive for the image metadata gate.
#
# Usage:
#   report-image-meta.sh <label> <image-type>
#
# Env (set by the workflow step):
#   TOKEN          GitHub token with packages:read (secrets.GITHUB_TOKEN)
#   IMAGE_NAME     image path under ghcr.io (e.g. zynax-io/zynax/ci-runner)
#   DIGEST         index digest (sha256:...) from the imagetools create step
#   EXTRA_SUMMARY  optional markdown appended after the digest line
#
# This script keeps ONLY the external primitives in shell (curl against GHCR,
# the read-after-write retry, and the per-platform compressed-size budget loop —
# explicitly left in YAML per cmd/zynax-ci/internal/imagesreport, ADR-036). The
# deterministic annotation/attestation decisions and the summary header are
# computed by `zynax-ci images meta`, which fails on a missing description
# annotation and warns on a missing title.
set -euo pipefail

LABEL="${1:?usage: report-image-meta.sh <label> <image-type>}"
IMAGE_TYPE="${2:-service}"

BASE="https://ghcr.io/v2/${IMAGE_NAME}"
AUTH_HEADER="Authorization: Bearer ${TOKEN}"

# ── Fetch the index manifest (retry — GHCR can lag right after push, #1022) ──
INDEX=""
for attempt in 1 2 3 4 5; do
  INDEX=$(curl -fsSL \
    -H "${AUTH_HEADER}" \
    -H "Accept: application/vnd.oci.image.index.v1+json" \
    "${BASE}/manifests/${DIGEST}" 2>/dev/null || true)
  [ -n "$INDEX" ] && break
  echo "::notice::index manifest for ${IMAGE_NAME}@${DIGEST} not ready (attempt ${attempt}/5); retrying in $((attempt * 3))s"
  sleep $((attempt * 3))
done

# ── Annotations + attestation count + summary header (tested Go logic) ───────
# Pipes the fetched index to zynax-ci; an empty body is handled as the
# non-fatal "GHCR read lag" skip path inside the verb. Built from source until
# the ci-runner image carries the verb.
printf '%s' "$INDEX" | GOWORK=off go -C cmd/zynax-ci run . images meta \
  --label "$LABEL" \
  --digest "$DIGEST" \
  --extra "${EXTRA_SUMMARY:-}"

# A non-resolved index already emitted the skip block above — nothing more to do.
if [ -z "$INDEX" ]; then
  exit 0
fi

# ── Size budget thresholds (external primitive: per-platform curl loop) ───────
if [ "$IMAGE_TYPE" = "tooling" ]; then
  BUDGET_MB=500
else
  BUDGET_MB=20
fi

{
  echo ""
  echo "| Platform | Compressed size | Budget |"
  echo "|----------|-----------------|--------|"
} >> "$GITHUB_STEP_SUMMARY"

SIZE_WARN=false
while IFS='|' read -r osname arch pdigs; do
  [ -z "$pdigs" ] && continue
  MF=$(curl -fsSL \
    -H "${AUTH_HEADER}" \
    -H "Accept: application/vnd.oci.image.manifest.v1+json" \
    "${BASE}/manifests/${pdigs}" 2>/dev/null) || true
  if [ -n "$MF" ]; then
    BYTES=$(echo "$MF" | jq '[.layers[].size] | add // 0')
    SIZE_MB=$(echo "scale=1; ${BYTES} / 1048576" | bc)
    SIZE_INT=$(echo "$SIZE_MB" | cut -d'.' -f1)
    if [ "${SIZE_INT}" -gt "${BUDGET_MB}" ]; then
      STATUS=":warning: > ${BUDGET_MB} MB"
      SIZE_WARN=true
    else
      STATUS="OK"
    fi
    echo "| ${osname}/${arch} | ${SIZE_MB} MB | ${STATUS} |" >> "$GITHUB_STEP_SUMMARY"
  else
    echo "| ${osname}/${arch} | — | — |" >> "$GITHUB_STEP_SUMMARY"
  fi
done < <(echo "$INDEX" | jq -r \
  '.manifests[] | select(.platform.architecture == "amd64" or .platform.architecture == "arm64") | select(.platform.os != "unknown") | [.platform.os, .platform.architecture, .digest] | join("|")' \
  2>/dev/null)

echo "" >> "$GITHUB_STEP_SUMMARY"

if [ "$SIZE_WARN" = "true" ]; then
  echo "::warning::One or more platforms of ${IMAGE_NAME} exceed the ${BUDGET_MB} MB compressed size budget."
fi
