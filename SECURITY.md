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
[Security Advisories](https://github.com/keel-io/keel/security/advisories/new).

Include: description, reproduction steps, impact assessment, and suggested fix if any.

**Response SLAs:**
- Acknowledgement: within 48 hours
- Initial assessment: within 5 business days
- Fix timeline: based on severity (Critical: 7 days, High: 30 days, Medium: 90 days)

## Security Controls

See `AGENTS.md §7` and `docs/adr/` for the full security architecture.
Key controls: mTLS between all services, non-root containers, SBOM per release,
cosign-signed images, Dependabot weekly updates, trivy CVE scanning in CI.
