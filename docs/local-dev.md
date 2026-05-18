# Local Development Guide

Docker-first quickstart. Only prerequisite: Docker Desktop (or Docker Engine + Compose plugin).

---

## Install the zynax CLI

Download a pre-built binary from [GitHub Releases](https://github.com/zynax-io/zynax/releases/latest):

```bash
# macOS Apple Silicon
curl -L https://github.com/zynax-io/zynax/releases/latest/download/zynax_darwin_arm64.tar.gz | tar xz
sudo mv zynax /usr/local/bin/

# Linux amd64
curl -L https://github.com/zynax-io/zynax/releases/latest/download/zynax_linux_amd64.tar.gz | tar xz
sudo mv zynax /usr/local/bin/

# Or build from source (requires Go 1.25 installed locally):
cd cmd/zynax && GOWORK=off go build -o ~/bin/zynax .
# or: make install-cli
```

---

## One-time setup

```bash
git clone https://github.com/zynax-io/zynax.git
cd zynax
make bootstrap    # pulls ghcr.io/zynax-io/zynax-tools:main from GHCR — run once after clone
```

Everything else runs inside containers. No Go, Python, or buf installation needed locally.

---

## Daily commands

| Command | What it does | When to use |
|---------|-------------|-------------|
| `make ci` | **Full local CI gate** — lint → test → security → secret scan in sequence | Before opening a PR |
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
make ci            # full CI gate before opening a PR
```

Work happens in `services/<service-name>/`. Use `GOWORK=off` for any `go` command run
directly (not via make) inside a service directory — see `CLAUDE.md` for why.

**Python agent contributor**

```
make lint              # ruff + mypy on agents/
make test-unit-agents  # pytest-bdd for SDK and examples
make security-agents   # bandit + pip-audit
make ci                # full CI gate before opening a PR
```

Work happens in `agents/sdk/` or `agents/examples/<agent>/`. Each agent is an isolated
`uv` project with its own `pyproject.toml`.

**Proto author**

```
make lint-protos        # buf lint + format check
# ... edit .proto files ...
make generate-protos    # regenerate stubs — always commit the output
make test-bdd           # verify BDD contract tests still pass
make ci                 # full CI gate before opening a PR
```

Proto changes require a `.feature` file committed first (ADR-016). Generated stubs in
`protos/generated/` are committed — never hand-edit them.

---

## Running the local stack

Start all three platform services plus Temporal and NATS with a single command:

```bash
make run-local    # build images + start (api-gateway, engine-adapter, workflow-compiler, Temporal, NATS)
make logs-local   # tail all logs
make stop-local   # stop and remove containers
```

Port map: api-gateway `http://localhost:7080` · Temporal UI `http://localhost:7088` · Temporal gRPC `localhost:7233` · NATS `localhost:7422`.

See [infra/docker-compose/README.md](../infra/docker-compose/README.md) for the full port map and startup order.

---

## Quick reference

- Per-layer AGENTS.md files live alongside the code they govern (`services/AGENTS.md`, `agents/AGENTS.md`, etc.)
- Architecture decisions: [docs/adr/INDEX.md](adr/INDEX.md)
- Full engineering contract: [CLAUDE.md](../CLAUDE.md) and [CONTRIBUTING.md](../CONTRIBUTING.md)
