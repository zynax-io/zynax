# ADR-007: pydantic-settings for All Configuration

**Status:** Accepted  **Date:** 2025-04-01

## Decision
Use `pydantic-settings` for all service configuration. Config is sourced
exclusively from environment variables. No YAML/TOML config files read at runtime.

## Rationale
- Validates configuration at startup — misconfigured service crashes immediately.
- Self-documenting (Field descriptions become documentation).
- Type-safe: `int`, `bool`, `SecretStr`, etc. — no string parsing bugs.
- 12-Factor compliant: config from environment.
- `model_config = ConfigDict(frozen=True)` prevents accidental mutation.

## Consequences
- All config must be expressible as environment variables.
- Complex nested config requires flattened env var names.
- Docker and K8s deploy configs are env var lists (documented in Helm values).
- `Settings()` call in `config.py` validates at import time — catches issues early.
