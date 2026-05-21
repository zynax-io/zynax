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
- Non-root containers — all service images run as `USER zynax` (uid 1001)
- Bearer-token authentication on all mutating api-gateway endpoints
- X-Request-ID correlation propagated across all services
- Structured JSON logging with no sensitive fields
- Renovate dependency updates (weekly, patch auto-merge)
- Trivy CVE scanning in release pipeline (#565)
- `ZYNAX_API_KEY` required at startup — gateway refuses to start without it

### In Progress
- SBOM per release artifact — tracked by [#235](https://github.com/zynax-io/zynax/issues/235) / [#556](https://github.com/zynax-io/zynax/issues/556)
- cosign-signed images (SLSA L2) — tracked by [#239](https://github.com/zynax-io/zynax/issues/239) / [#489](https://github.com/zynax-io/zynax/issues/489)
- Constant-time bearer comparison — tracked by [#567](https://github.com/zynax-io/zynax/issues/567) (to be filed)

### Planned (M6+)
- mTLS between all platform services — tracked by [#464](https://github.com/zynax-io/zynax/issues/464) / [#488](https://github.com/zynax-io/zynax/issues/488)
- OIDC/JWT authentication replacing static bearer token
- Rate limiting on `POST /api/v1/apply`
- `ReadHeaderTimeout` on all HTTP servers
