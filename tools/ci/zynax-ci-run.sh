#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# zynax-ci-run.sh — run a zynax-ci verb from the binary baked into the
# ci-runner image, falling back to compiling from source when the change under
# test touches cmd/zynax-ci itself (#1715; pipeline speed-up analysis #1687).
#
# Why: `go run ./cmd/zynax-ci` cold-downloads the module graph and compiles on
# every invocation (~30–60 s each, ~6× per PR) although the runner image bakes
# the binary at /usr/local/bin/zynax-ci and tools-image.yml rebuilds it on
# every merge touching cmd/zynax-ci/** — it is at most one merge stale.
#
# The one case where staleness matters is a PR that CHANGES zynax-ci and relies
# on the new behavior in the same run: callers export CLI_CHANGED=true there
# (ci.yml wires it from the changes job's `cli` output) and this script
# compiles from source. A missing baked binary (running outside the ci-runner)
# also falls back, so the script works on any host with Go.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [[ "${CLI_CHANGED:-false}" == "true" ]] || ! command -v zynax-ci >/dev/null 2>&1; then
  echo "zynax-ci-run: compiling from source (CLI_CHANGED=${CLI_CHANGED:-unset})" >&2
  cd "${REPO_ROOT}/cmd/zynax-ci"
  GOWORK=off exec go run . "$@"
fi
exec zynax-ci "$@"
