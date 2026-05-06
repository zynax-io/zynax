# tools/

Developer tooling configuration files used across CI and local development.

## Files

| File | Purpose |
|------|---------|
| `gitleaks-ai-context.toml` | Gitleaks rules for scanning AI context files (CLAUDE.md, AGENTS.md, docs/ai-assistant-setup.md) for Tier 2 content |
| `golangci-lint.yml` | golangci-lint configuration for all Go services |
| `mypy.ini` | mypy type-checking configuration for Python agents |
| `ruff.toml` | ruff linter/formatter configuration for Python agents |

## Validation

All YAML manifest, Canvas, JSON Schema, and AI context validation is now handled by
the `zynax-ci` binary. The Python scripts and shell scripts that previously lived here
(`validate_canvas.py`, `validate_workflows.py`, `validate_agent_defs.py`,
`validate_capabilities.py`, `validate_policies.py`, `validate_json_schemas.py`,
`count-ai-context.sh`) were removed in M4 step 11 (#336).

To run validation:

```bash
# All spec validation (via Makefile — uses zynax-ci internally):
make validate-spec
make validate-canvas

# Direct zynax-ci invocations:
zynax-ci validate canvas docs/spdd/
zynax-ci validate schema spec/schemas/workflow.json
zynax-ci validate workflows spec/workflows/examples/
zynax-ci validate agent-defs spec/
zynax-ci validate capabilities spec/
zynax-ci validate policies spec/
zynax-ci check ai-context

# Install zynax-ci locally:
make install-ci-tools
```

See [cmd/zynax-ci/AGENTS.md](../cmd/zynax-ci/AGENTS.md) for the full command reference.
