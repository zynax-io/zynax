# ADR-006: Monorepo with Per-Service pyproject.toml

**Status:** Accepted  **Date:** 2025-04-01

## Decision
Single Git repository for all Keel services, with each service having
its own `pyproject.toml`, virtualenv, and independent versioning.

## Rationale
- Single CI pipeline with shared tooling configuration.
- Atomic cross-service changes (proto change + all consumers in one PR).
- Easier for contributors to run the full platform locally.
- `uv workspaces` supports this pattern natively.

## Consequences
- Root `pyproject.toml` defines the uv workspace.
- Each service is independently installable: `uv pip install ./services/agent-registry`.
- Proto changes affect multiple services — all must be updated in the same PR.
- CI runs affected service tests only (via path-based job filtering).
