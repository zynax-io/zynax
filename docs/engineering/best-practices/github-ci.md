<!-- SPDX-License-Identifier: Apache-2.0 -->

# GitHub Actions CI Best Practices — Zynax

> Scope: `.github/workflows/`  
> Enforcement: CI gates listed below; review against these standards on each PR.

---

## Pin Actions by Commit SHA

```yaml
# ❌ Floating tag — supply-chain risk
- uses: actions/checkout@v4

# ✅ Pinned by SHA — immutable
- uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683  # v4.2.2
```

Every third-party action must be pinned to a full commit SHA. Use Renovate to keep
SHA pins updated automatically (already configured in `renovate.json`).

---

## Least-Privilege permissions

```yaml
# ✅ At workflow level: default to read-only
permissions:
  contents: read

jobs:
  release:
    permissions:
      contents: write       # only jobs that need it
      packages: write       # only jobs that push to GHCR
      id-token: write       # only jobs using OIDC
```

Never use `permissions: write-all` at the workflow level.

---

## Concurrency Groups (cancel stale runs)

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```

Always set `concurrency:` on any workflow that runs on `push` or `pull_request`.
Stale-run cancellation was added in #545 — don't regress it.

---

## Required CI Gates

| Gate | Workflow | What it checks |
|---|---|---|
| `proto-breaking` | `pr-checks.yml` | `buf breaking` — no backward-incompatible proto changes |
| `stubs-freshness` | `pr-checks.yml` | Regenerate stubs and verify no diff |
| `layer-boundaries` | `pr-checks.yml` | `zynax-ci validate` — no domain→proto imports |
| `conventional-commit` | `pr-checks.yml` | PR title format (type: subject) |
| `pr-size` | `pr-size.yml` | ≤ 900 LOC (with exclusions: `*.pb.go`, `*_pb2.py`, `.github/workflows/`, lock files) |
| `validate-canvas` | `pr-checks.yml` | `zynax-ci validate canvas` for `docs/spdd/` |
| `coverage-gate` | `ci.yml` | ≥ 90% on all `internal/domain/` packages |
| `golangci-lint` | `ci.yml` | Go lint |
| `ruff + mypy + bandit` | `ci.yml` | Python lint + type check + SAST |
| `trivy-scan` | `release.yml` | CVE scan before GHCR push (#565) |

---

## Build Matrix and Caching

```yaml
strategy:
  matrix:
    service: [workflow-compiler, engine-adapter, api-gateway, task-broker]
jobs:
  build:
    runs-on: ubuntu-latest
    container:
      image: ghcr.io/zynax-io/zynax/ci-runner:latest  # ← pre-baked, no downloads
    steps:
      - uses: actions/cache@...
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
```

Use the `ci-runner` container image (built from `infra/docker/Dockerfile.ci-runner`) for
all Go + Python CI jobs. It bakes in every tool (Go 1.26.3, golangci-lint, buf, uv, ruff,
mypy, bandit, zynax-ci) so no step downloads tooling at runtime. Built in #551/#552.

---

## Change Detection (skip unchanged services)

```yaml
- uses: dorny/paths-filter@...
  id: changes
  with:
    filters: |
      workflow-compiler: services/workflow-compiler/**
      engine-adapter:    services/engine-adapter/**
```

Only run service-specific tests when that service's files change. The `changes` job
provides a JSON array of changed services; downstream jobs use `needs.changes.outputs`.
See #549 for the full per-module granularity enhancement.

---

## Release Artifacts (supply chain)

The unified release workflow (`release.yml`) produces:
- CLI binary `zynax` for darwin/linux × amd64/arm64 (4 platforms)
- `zynax-ci` binary for the same platforms
- Service Docker images pushed to GHCR (`ghcr.io/zynax-io/zynax/<service>:v<version>`)

**Planned (M6, #489):**
- SBOM per artifact (syft SPDX format via `anchore/sbom-action`)
- cosign keyless signing (`cosign sign --keyless`)
- SLSA L2 provenance (via `slsa-framework/slsa-github-generator`)

Until these are implemented, do NOT claim supply-chain security in documentation.

---

## GHCR Image Naming

```
ghcr.io/zynax-io/zynax/<service>:main        ← every merge to main
ghcr.io/zynax-io/zynax/<service>:v<semver>  ← every versioned release
ghcr.io/zynax-io/zynax/<service>:latest     ← same as latest release
```

All images are publicly readable (no authentication to pull). Set by `make-packages-public`
step in release workflow.

---

## Workflow File Structure

| File | Trigger | Purpose |
|---|---|---|
| `ci.yml` | push + PR | Main CI: lint, test, BDD, coverage, security |
| `pr-checks.yml` | PR only | Proto, canvas, conventional-commit, PR-size |
| `release.yml` | tag push | Unified release: CLI + zynax-ci + service images |
| `cli-release.yml` | called by release.yml | CLI binary build and upload |
| `service-release.yml` | called by release.yml | Service image build and push |
| `zynax-ci-release.yml` | called by release.yml | zynax-ci binary build |
| `tools-image.yml` | Dockerfile.tools change | Rebuild and push tools image |
| `proto-generate.yml` | proto file change (post-merge) | Auto-regenerate stubs |
| `ai-context-budget.yml` | relevant files change | Advisory line-count check |
| `pr-size.yml` | PR | PR size gate |

`ci.yml` is 1,325 lines — tracked for DRY refactor to composite/reusable workflows (#555).
Do NOT add more inline YAML blocks; extract to `tools/` scripts instead.

---

## Anti-patterns to Avoid

| Anti-pattern | Correct approach |
|---|---|
| Multi-line Python in `run: \|` YAML | Extract to `tools/<name>.py`; YAML scalar terminates on un-indented lines |
| `@latest` for tool install in CI | Pin version via env var (e.g. `GOVULNCHECK_VERSION`) |
| `go install tool@latest` in CI jobs | Bake into `Dockerfile.ci-runner` |
| Hardcoded `golang:1.22-alpine` in Dockerfiles | Use `golang:1.26.3-alpine` (match `go.work` toolchain) |
| GOPATH `/root/go/bin/` on Alpine | Use `/go/bin/` — GOPATH on Alpine is `/go` |
| Long-lived secrets in workflow env | Use OIDC + short-lived tokens (planned ADR-020) |
| `--no-verify` in CI commits | Investigate hook failure; never bypass |
