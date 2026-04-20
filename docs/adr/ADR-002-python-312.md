# ADR-002: Python 3.12 as Primary Language

**Status:** Accepted  **Date:** 2025-04-01

## Decision
Python ≥ 3.12 exclusively. No support for earlier versions.

## Rationale
- Best AI/ML ecosystem (LangChain, HuggingFace, numpy, etc.).
- 3.12 `asyncio` performance improvements (critical for gRPC servers).
- Improved `typing` module (TypeVar defaults, `@override`).
- `tomllib` in stdlib (no extra dep for config parsing).
- Active security support until 2028.

## Consequences
- All type hints use 3.12+ syntax (`list[str]` not `List[str]`, `X | None` not `Optional[X]`).
- `uv` manages exact Python version via `.python-version` file.
- CI matrix: Python 3.12 only (not a compatibility library — no multi-version matrix needed).
