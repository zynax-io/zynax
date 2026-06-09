<!-- SPDX-License-Identifier: Apache-2.0 -->

# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| main    | ✅ Always |
| latest release | ✅ |
| previous release | ✅ Security patches only |
| older  | ❌ |

## Reporting a Vulnerability

**Do NOT open a public GitHub issue for security vulnerabilities.**

Please report security vulnerabilities via GitHub's private
[Security Advisories](https://github.com/zynax-io/zynax/security/advisories/new).

Include: description, reproduction steps, impact assessment, and suggested fix if any.

**Response SLAs:**
- Acknowledgement: within 48 hours
- Initial assessment: within 5 business days
- Fix timeline: based on severity (Critical: 7 days, High: 30 days, Medium: 90 days)

## Security Controls

See `AGENTS.md`, `docs/adr/ADR-001`, `ADR-008` and `docs/reviews/04-architecture-gaps.md` for the full security posture.

### Current (shipped)
- Non-root containers — all service images use `gcr.io/distroless/static:nonroot` (uid 65532)
- Bearer-token authentication on all mutating api-gateway endpoints (constant-time compare ✅ #567)
- `ReadHeaderTimeout` + `MaxBytesReader` on HTTP server (slow-read DoS protection ✅ #568)
- X-Request-ID correlation propagated across all services
- Structured JSON logging with no sensitive fields
- Renovate dependency updates (weekly, patch auto-merge)
- Trivy CVE scanning in release pipeline (#565)
- `ZYNAX_API_KEY` required at startup — gateway refuses to start without it
- gRPC call deadlines on all outbound calls (#622); configurable via `ZYNAX_GRPC_TIMEOUT_MS`
- SPDX SBOM attached to every GitHub release for all service + tools images (✅ #489)
- cosign keyless signing of all container image releases — SLSA L2 (✅ #489)
- Multi-arch container images (linux/amd64 + linux/arm64) for all service + tools images (✅ #489)
- cosign sign-blob + SPDX SBOM for zynax-sdk wheel/sdist PyPI artifacts (✅ #807)
- Minimal image footprint — smaller images mean a smaller attack surface (✅ #841)

### Container image sizes (attack-surface budget)

Compressed sizes for `linux/amd64`. Smaller is better: a distroless Go image has
no shell, package manager, or libc, so most container CVE classes do not apply.

| Image | Base | Compressed (amd64) | Budget | Notes |
|-------|------|--------------------|--------|-------|
| api-gateway, workflow-compiler, engine-adapter, task-broker, agent-registry | `distroless/static:nonroot` | ~8 MB | ≤ 15 MB | static, stripped (`-s -w -trimpath`), no shell |
| http-adapter, git-adapter, ci-adapter | `distroless/static:nonroot` | ~8 MB | ≤ 15 MB | static, stripped, no shell |
| llm-adapter | `python:3.12-slim` | ~75 MB | ≤ 80 MB | frozen uv venv (no-dev); slim over alpine (manylinux wheels) |
| langgraph-adapter | `python:3.12-slim` | ~78 MB | ≤ 80 MB | frozen uv venv (no-dev); slim over alpine (manylinux wheels) |
| tools | `python:3.14-alpine` + Go | ~480 MB | dev-only | not deployed; full Go toolchain + Python dev tools (justified in Dockerfile header) |
| ci-runner | `alpine:3.21` + Go | ~430 MB | CI-only | not deployed; Go toolchain + CI tools (justified in Dockerfile header) |

Per-build sizes are reported in the GitHub Actions step summary of `release.yml`
and `tools-image.yml`. Test files and fixtures are excluded from every production
build context via `.dockerignore`.

**Verify an image signature:**
```bash
cosign verify \
  --certificate-identity-regexp="https://github.com/zynax-io/zynax/.github/workflows/" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  ghcr.io/zynax-io/zynax/api-gateway:v0.5.0
```

Replace `api-gateway` with any service name (`engine-adapter`, `workflow-compiler`, `task-broker`, `agent-registry`, `http-adapter`, `tools`, `ci-runner`) and the tag with the release version.

**Verify a SDK wheel signature:**
```bash
# Download the wheel, sdist, and their cosign bundles from the GitHub Release:
# https://github.com/zynax-io/zynax/releases/tag/v0.1.0
cosign verify-blob \
  --bundle zynax_sdk-0.1.0-py3-none-any.whl.bundle \
  --certificate-identity-regexp="https://github.com/zynax-io/zynax/.github/workflows/" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  zynax_sdk-0.1.0-py3-none-any.whl
```

Replace the version with the release you downloaded. To verify the sdist, use the `.tar.gz` and its corresponding `.tar.gz.bundle` instead.

### Planned (M6+)
- mTLS between all platform services — ✅ shipped #488; cert-manager Helm chart in M6.Helm
- OIDC/JWT authentication replacing static bearer token
- Rate limiting on `POST /api/v1/apply`
