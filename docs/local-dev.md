# Local Development Guide

Docker-first quickstart. Only prerequisite: Docker Desktop (or Docker Engine + Compose plugin).

---

## One-time setup

```bash
git clone https://github.com/zynax-io/zynax.git
cd zynax
make bootstrap    # builds the zynax-tools:local Docker image — run once after clone
```

Everything else runs inside containers. No Go, Python, or buf installation needed locally.

---

## Daily commands

| Command | What it does | When to use |
|---------|-------------|-------------|
| `make lint` | Proto + Go + Python lint (ruff, mypy, golangci-lint, buf) | Before every commit |
| `make test` | Full suite: spec validation + Go unit tests + BDD contracts + Python tests | Before pushing |
| `make test-unit-go` | Go unit tests with coverage report for all services | During Go development |
| `make test-bdd` | Godog BDD contract tests in `protos/tests/` | After changing a proto or BDD step |
| `make test-unit-agents` | pytest-bdd for SDK and all Python agents | During Python development |
| `make test-integration` | Integration tests against NATS and Redis (spins up containers) | Before opening a PR |
| `make generate-protos` | Regenerate Go + Python stubs from `.proto` files — commit the output | After editing a `.proto` file |
| `make validate-spec` | Validate YAML manifests against JSON schemas | After editing `spec/` files |
| `make security` | govulncheck + bandit + pip-audit — full security scan | Before releasing |
| `make audit` | govulncheck + pip-audit only — faster dependency CVE check | Weekly or on dependency change |

---

## Persona paths

**Go service contributor**

```
make lint          # catch style issues early
make test-unit-go  # fast feedback loop while coding
# ... write code ...
make test          # full suite before pushing
```

Work happens in `services/<service-name>/`. Use `GOWORK=off` for any `go` command run
directly (not via make) inside a service directory — see `CLAUDE.md` for why.

**Python agent contributor**

```
make lint              # ruff + mypy on agents/
make test-unit-agents  # pytest-bdd for SDK and examples
make security-agents   # bandit + pip-audit
```

Work happens in `agents/sdk/` or `agents/examples/<agent>/`. Each agent is an isolated
`uv` project with its own `pyproject.toml`.

**Proto author**

```
make lint-protos        # buf lint + format check
# ... edit .proto files ...
make generate-protos    # regenerate stubs — always commit the output
make test-bdd           # verify BDD contract tests still pass
make lint               # full lint pass before PR
```

Proto changes require a `.feature` file committed first (ADR-016). Generated stubs in
`protos/generated/` are committed — never hand-edit them.

---

## Quick reference

- Per-layer AGENTS.md files live alongside the code they govern (`services/AGENTS.md`, `agents/AGENTS.md`, etc.)
- Architecture decisions: [docs/adr/INDEX.md](adr/INDEX.md)
- Full engineering contract: [CLAUDE.md](../CLAUDE.md) and [CONTRIBUTING.md](../CONTRIBUTING.md)
