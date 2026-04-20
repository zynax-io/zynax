# ADR-005: Apache 2.0 License

**Status:** Accepted  **Date:** 2025-04-01

## Decision
License Keel under Apache License 2.0.

## Rationale
- Permissive: enterprises can adopt without legal friction.
- Patent grant: contributors grant patent rights to users (important for AI infra).
- CNCF preferred license for Sandbox projects.
- Compatible with most OSS dependencies.
- Does not require derivative works to be open-sourced (unlike AGPL).

## Consequences
- SPDX header required in every source file (enforced by `license-eye` in CI).
- Contributors must sign CLA (automated via GitHub CLA bot).
- All dependencies must have Apache 2.0-compatible licenses (checked by `pip-licenses` in CI).
