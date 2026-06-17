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
- CLI binary `zynax` for darwin/linux/windows × amd64/arm64 (5 platforms)
- `zynax-ci` binary for darwin/linux × amd64/arm64 (4 platforms)
- Service Docker images pushed to GHCR (`ghcr.io/zynax-io/zynax/<service>:v<version>`)
- SPDX SBOM per service image (syft — attached to GitHub Release)
- cosign keyless signature per image (tag builds and `workflow_dispatch` only)
- SLSA Build L1 provenance attestation — generated automatically by `docker/build-push-action`
  (see [ADR-025](../../adr/ADR-025-slsa-provenance-attestation.md) and the GHCR Package Hygiene
  section below)

---

## GHCR Package Hygiene

### OCI labels vs manifest annotations

Docker image metadata requires **both** Docker labels (for `docker inspect`) and OCI
manifest annotations (for the GHCR web UI).  Labels are set via
`org.opencontainers.image.*` in the `docker/metadata-action` labels block; annotations
are forwarded via the `annotations:` key in `docker/build-push-action`.  Omitting
annotations leaves the package description blank in the GHCR UI.

### unknown/unknown rows are expected — do not delete them

Every multi-arch push produces two extra entries in the GHCR Packages UI labelled
`unknown/unknown`.  These are **SLSA Build L1 provenance attestation manifests**
generated automatically by `docker buildx` (`--provenance=mode=min`).

- One entry per platform variant (linux/amd64, linux/arm64).
- Manifest size ~565 bytes; encodes build invocation, GitHub Actions run ID, source
  commit, and SLSA Build L1 statement.
- Media type: `application/vnd.oci.image.manifest.v1+json` with
  `vnd.docker.reference.type: attestation-manifest`.

**Decision:** keep attestations enabled. See
[ADR-025](../../adr/ADR-025-slsa-provenance-attestation.md).  Do **not** set
`provenance: false` — it drops SLSA provenance, weakens the OpenSSF Scorecard posture,
and breaks `cosign verify-attestation`.

### Verifying a published image

```bash
# Inspect the full manifest index (shows platform entries + attestation refs):
docker buildx imagetools inspect ghcr.io/zynax-io/zynax/<image>:<tag> --raw

# Verify SLSA attestation:
cosign verify-attestation --type slsaprovenance \
  ghcr.io/zynax-io/zynax/<image>@sha256:<digest>

# Verify cosign signature (tag builds only):
cosign verify ghcr.io/zynax-io/zynax/<image>:<version>
```

### Retention policy

GHCR is capped at the last **5** `main-sha` builds per image.  `:latest` and `v*.*.*`
tags are never pruned.  The cleanup job runs automatically in `tools-image.yml` and
`release.yml` after every successful multi-arch push.

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
Do NOT add more inline `run:` logic; move deterministic decisions into a tested
`zynax-ci` subcommand and call it from the step (ADR-036). The external primitives
(`cosign` / `crane` / `kubectl` / `helm` / `docker` / `curl` / `gh api`) stay in
thin shell; `zynax-ci` only computes the decisions around them.

### Deterministic CI gates as `zynax-ci` verbs (ADR-036)

Each gate is one tested subcommand — one source of truth per gate. The bash
scripts these replaced (`build-coverage-comment.sh`, `bench-regression.sh`,
`bdd-select-packages.sh`, `bump-ci-runner.sh`) and the `report-image-meta`
composite action were retired in M7.S.7 (#1292).

| Gate / step | Verb | Used by |
|-------------|------|---------|
| PR coverage comment | `zynax-ci coverage-comment` | `_test-go.yml` |
| Benchmark regression | `zynax-ci bench-gate` | benchmark gate |
| BDD package matrix | `zynax-ci bdd-select` | `_test-go.yml` |
| ci-runner digest bump | `zynax-ci bump-runner <digest>` | `make bump-ci-runner` |
| Image metadata/budget | `zynax-ci images meta` | `tools-image.yml` |
| GHCR version cleanup | `zynax-ci images cleanup` | pr-image / prune |
| Release retag list | `zynax-ci images retag` | `tools-image.yml` |
| Release notes/matrix | `zynax-ci release notes` / `matrix` | `release.yml` |

The e2e harness (`scripts/e2e/*`) is the one sanctioned exception — it stays bash
(thin orchestration over kind/kubectl/helm/docker, ADR-036).

---

## Generated Container References

All container image digests and tags used in workflow files and Dockerfiles are **generated**
from [`images/images.yaml`](../../../images/images.yaml) via `zynax-ci images sync`.

Regions managed by the generator are delimited by banner comments:
```yaml
# BEGIN generated by zynax-ci images sync
image: ghcr.io/zynax-io/zynax/tools@sha256:<digest>
# END generated by zynax-ci images sync
```

**Rules:**
- Do not hand-edit any banner-marked region — it will be overwritten on the next sync.
- To update a digest: edit `images/images.yaml`, then run `make sync-images`.
- CI runs `make check-images` on every PR; any stale banner region fails the build.
- See [`cmd/zynax-ci/AGENTS.md`](../../../cmd/zynax-ci/AGENTS.md) for the `images sync`
  and `images check` subcommand reference.

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
