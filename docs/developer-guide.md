# Developer Guide

The day-to-day workflow for contributing to Zynax. If you have not run a workflow
locally yet, start with the **[Quick Start](quickstart.md)** first.

Everything runs inside Docker — the only prerequisite is Docker Desktop (or Docker
Engine + the Compose plugin). `make help` lists every target.

---

## One-time setup

```bash
git clone https://github.com/zynax-io/zynax.git
cd zynax
make bootstrap      # pulls ghcr.io/zynax-io/zynax/tools:latest from GHCR — run once
make install-cli    # builds the zynax CLI → ~/bin/zynax (requires the Go toolchain pinned in go.work)
```

---

## The local stack

| Command | What it does |
|---------|-------------|
| `zynax up` | kind cluster + Helm umbrella — the one local runtime (ADR-041) |
| `make demo` | Same bring-up + the hero workflow ("Platform ready" banner) |
| `zynax down` / `make kind-down` | Tear the cluster back down |

After `zynax up`, point the CLI at the gateway (port-forward; it defaults to port `8080`):

```bash
export ZYNAX_API_URL=http://localhost:7080
```

### Observability overlay (optional)

| Command | What it does |
|---------|-------------|
| `make obs-up` | Start the local Uptrace stack — UI at `http://localhost:7020` |
| `make obs-logs` | Tail the observability stack logs |
| `make obs-down` | Stop the observability stack |

`make obs-up` requires `infra/docker-compose/observability/.env.observability` (copy the
`.env.observability.example` next to it and set a login + token — there are no committed
defaults). Telemetry is off until services are pointed at the collector:

```bash
export ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:7017
```

See [infra/docker-compose/README.md](../infra/docker-compose/README.md) and
[docs/observability/](observability/) for the full telemetry setup.

---

## The `zynax` CLI

The CLI is a standalone tool that talks to the api-gateway over HTTP REST only.

| Command | What it does |
|---------|-------------|
| `zynax init workflow [name]` | Scaffold a Workflow manifest from a template (stdout or `-o`) |
| `zynax init expert [name]` | Scaffold an expert (AgentDef) manifest |
| `zynax validate <file>` | Static + data-flow validation of a manifest (no gateway needed) |
| `zynax apply --dry-run <file>` | Compile a manifest without submitting it |
| `zynax apply <file>` | Apply a Workflow or AgentDef — prints `run_id` / `agent_id` |
| `zynax get workflow <run-id>` | Full snapshot of a run (id, workflow, status, state, version) |
| `zynax status workflow <run-id>` | Print the run status (exit `0` if terminal, `2` if running) |
| `zynax logs <run-id> --follow` | Stream lifecycle events; exit on terminal state |

`init` and `validate` run entirely locally — no gateway connection required. Global
flags: `--api-url` (defaults to `$ZYNAX_API_URL`) and `--insecure`.

A typical author loop:

```bash
zynax init workflow my-pipeline -o my-pipeline.yaml   # scaffold
zynax validate my-pipeline.yaml                       # check before submitting
zynax apply --dry-run my-pipeline.yaml                # compile-only
zynax apply my-pipeline.yaml                          # submit → run_id
zynax logs <run-id> --follow                          # watch
```

---

## Make targets — daily workflow

| Command | What it does | When to use |
|---------|-------------|-------------|
| `make ci` | Full local CI gate — lint → test → security → secret scan | Before opening a PR |
| `make lint` | Proto + Go + Python lint (ruff, mypy, golangci-lint, buf) | Before every commit |
| `make test` | Full suite: spec validation + Go unit + BDD + Python | Before pushing |
| `make test-unit-go` | Go unit tests with coverage for all services | During Go development |
| `make test-bdd` | Godog BDD contract tests in `protos/tests/` | After a proto/BDD step change |
| `make test-unit-agents` | pytest-bdd for the SDK and all Python agents | During Python development |
| `make test-integration` | Integration tests against NATS and Redis (spins up containers) | Before opening a PR |
| `make generate-protos` | Regenerate Go + Python stubs from `.proto` — commit the output | After editing a `.proto` |
| `make validate-spec` | Validate YAML manifests against JSON schemas | After editing `spec/` files |
| `make security` | govulncheck + bandit + pip-audit — full scan | Before releasing |
| `make audit` | govulncheck + pip-audit only — faster CVE check | Weekly / on dependency change |
| `make sync-images` | Stamp consumer files with digests from `images/images.yaml` | After an image digest changes |
| `make check-images` | Verify banner-marked regions match `images/images.yaml` | CI gate / before PR |

> **Image digests** are managed in `images/images.yaml` — the single source of truth. Do
> not hand-edit banner-marked regions in workflow files or Dockerfiles; use
> `make sync-images` to update them.

---

## Contributor paths

**Go service contributor** — work in `services/<service-name>/`:

```bash
make lint            # catch style issues early
make test-unit-go    # fast feedback while coding
make test            # full suite before pushing
make ci              # full CI gate before opening a PR
```

Use `GOWORK=off` for any `go` command run directly (not via make) inside a service
directory — see [CLAUDE.md](../CLAUDE.md) for why.

**Python agent contributor** — work in `agents/sdk/` or `agents/examples/<agent>/`:

```bash
make lint                # ruff + mypy on agents/
make test-unit-agents    # pytest-bdd for SDK and examples
make security-agents     # bandit + pip-audit
make ci                  # full CI gate before opening a PR
```

**Proto author** — proto changes require a `.feature` file committed first (ADR-016):

```bash
make lint-protos         # buf lint + format check
make generate-protos     # regenerate stubs — always commit the output
make test-bdd            # verify BDD contract tests still pass
make ci                  # full CI gate before opening a PR
```

Generated stubs in `protos/generated/` are committed — never hand-edit them.

---

## Before you open a PR

1. `make ci` — the full local gate (lint → test → security → secret scan).
2. Commit with `Signed-off-by:` (DCO) and, for AI-assisted work, `Assisted-by:`.
3. Build the PR body from [docs/contributing/pr-templates.md](contributing/pr-templates.md).

Keep PRs small: ≤ 200 lines ideal, 201–400 acceptable, 401–900 justify, > 900 blocked.
See [CLAUDE.md](../CLAUDE.md) for the full PR-size policy and conventional-commit rules.

---

## Reference

- **[Quick Start](quickstart.md)** — clone → run → traced workflow run.
- **[Local Development Guide](local-dev.md)** — CLI install options and persona paths.
- **[Architecture](../ARCHITECTURE.md)** — the three-layer separation.
- **[docs/adr/INDEX.md](adr/INDEX.md)** — Architecture Decision Records.
- **[CLAUDE.md](../CLAUDE.md)** / **[CONTRIBUTING.md](../CONTRIBUTING.md)** — engineering contract.
</content>
