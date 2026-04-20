# ADR-003: uv as Package Manager

**Status:** Accepted  **Date:** 2025-04-01

## Decision
Use `uv` as the sole package manager, virtualenv manager, and lockfile tool.
Replace pip, pip-tools, poetry, and virtualenv.

## Rationale
- 10-100x faster than pip for installs and dependency resolution.
- `uv.lock` is deterministic and cross-platform.
- Native workspace support (monorepo with per-service `pyproject.toml`).
- Single binary — no separate virtualenv tooling.
- Compatible with PyPI and `pyproject.toml` standard.

## Consequences
- `pyproject.toml` is the single source of truth for every service.
- `uv.lock` is committed. `uv sync --frozen` in CI and Docker builds.
- No `requirements.txt` files anywhere in the repo.
- Contributors must install `uv` (documented in CONTRIBUTING.md).
