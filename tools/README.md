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

## CI gates

Deterministic CI-gate logic is implemented as tested `zynax-ci` subcommands, one
source of truth per gate (ADR-036). The shell scripts that previously lived here
and under `.github/` (`bench-regression.sh`, `bdd-select-packages.sh`,
`build-coverage-comment.sh`, `bump-ci-runner.sh`) and the `report-image-meta`
composite action were retired in M7.S.7 (#1292). Their replacements:

| Gate | Verb |
|------|------|
| PR coverage comment | `zynax-ci coverage-comment` |
| Benchmark regression | `zynax-ci bench-gate` |
| BDD package matrix | `zynax-ci bdd-select` |
| ci-runner digest bump | `zynax-ci bump-runner <digest>` |
| Image metadata/budget | `zynax-ci images meta` |
| GHCR version cleanup | `zynax-ci images cleanup` |
| Release retag list | `zynax-ci images retag` |

`tools/ci/run-go-svc-loop.sh` stays bash (a thin per-service `go` loop), as does
the e2e harness (`scripts/e2e/*`) — both are thin orchestration over external
CLIs (ADR-036).

See [cmd/zynax-ci/AGENTS.md](../cmd/zynax-ci/AGENTS.md) for the full command reference.
