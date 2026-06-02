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

**Verify an image signature:**
```bash
cosign verify \
  --certificate-identity-regexp="https://github.com/zynax-io/zynax/.github/workflows/" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  ghcr.io/zynax-io/zynax/api-gateway:v0.5.0
```

Replace `api-gateway` with any service name (`engine-adapter`, `workflow-compiler`, `task-broker`, `agent-registry`, `http-adapter`, `tools`, `ci-runner`) and the tag with the release version.

### Planned (M6+)
- mTLS between all platform services — ✅ shipped #488; cert-manager Helm chart in M6.Helm
- OIDC/JWT authentication replacing static bearer token
- Rate limiting on `POST /api/v1/apply`
