# Quick Start

> **In a hurry?** Run `make demo` for the one-command path: it boots a zero-secret local-LLM
> stack and runs the hero code-review workflow end-to-end. Prereq: `ollama pull qwen2.5-coder:3b`.
> The steps below walk through the same flow manually.

Get from a fresh clone to a **traced workflow run** in minutes. Everything runs in
Docker — the only prerequisite is Docker Desktop (or Docker Engine + the Compose plugin).

By the end you will have:

1. A zero-secret local-LLM stack running (the base stack + the Ollama overlay).
2. The hero `code-review-ollama` workflow applied with `zynax apply`.
3. The run watched live with `zynax logs --follow` and its review printed with `zynax result`.
4. (Optional) The trace and logs for that run visible in the Uptrace UI.

Every command shown below maps to a real `zynax` subcommand — see
[Command reference](#command-reference) at the end for the full list and how each was
verified against the CLI source. Prefer watching it run first? See the recorded
walkthroughs in [docs/casts/](casts/).

---

## 1. Clone and bootstrap

```bash
git clone https://github.com/zynax-io/zynax.git
cd zynax
make bootstrap    # one-time: pulls ghcr.io/zynax-io/zynax/tools:latest from GHCR
```

`make bootstrap` is only needed once after cloning. Nothing else needs Go, Python, or
`buf` installed locally — they all run inside containers.

---

## 2. Install the `zynax` CLI

The CLI talks to the api-gateway over HTTP REST. Build it from source (requires the
Go toolchain pinned in [go.work](../go.work)) or grab a release binary:

```bash
make install-cli                       # builds → ~/bin/zynax (ensure ~/bin is on PATH)
# or download a pre-built binary — see docs/local-dev.md
```

Confirm it runs:

```bash
zynax --version
```

---

## 3. Start the stack with the zero-secret Ollama overlay

The fastest runnable path needs **no API keys and no cloud LLM**. A small Compose
overlay ([docker-compose.ollama.yml](../infra/docker-compose/docker-compose.ollama.yml))
adds an `ollama` service inside the network and repoints the llm-adapter at it via
[ollama/llm-adapter.config.yaml](../infra/docker-compose/ollama/llm-adapter.config.yaml) —
which registers a `codereview` capability against a local model. Pull the demo model
once on the host (the overlay reuses host-pulled models read-only — nothing is
re-downloaded), then layer the overlay on the base stack:

```bash
ollama pull qwen2.5-coder:3b               # default model (see override note below)

docker compose \
  -f infra/docker-compose/docker-compose.yml \
  -f infra/docker-compose/docker-compose.ollama.yml \
  up -d --wait
```

This starts the api-gateway, engine-adapter, workflow-compiler, Temporal, NATS, the
`ollama` service, and the llm-adapter — all locally, with no secrets.

**Model / host-path overrides** (both optional):

- The model lives in one line of `ollama/llm-adapter.config.yaml`
  (`model: qwen2.5-coder:3b`). To switch, edit that line (e.g. `model: llama3.2:3b`) and
  pull the new model on the host.
- The overlay bind-mounts the host models dir read-only, defaulting to a systemd
  install path. Override it for other setups:

  ```bash
  OLLAMA_HOST_MODELS=$HOME/.ollama/models docker compose \
    -f infra/docker-compose/docker-compose.yml \
    -f infra/docker-compose/docker-compose.ollama.yml up -d --wait
  ```

> **Cloud LLM instead?** Drop the `-f docker-compose.ollama.yml` overlay and run plain
> `make run-local`; the baked llm-adapter config then expects an OpenAI key.

When it finishes, the api-gateway is reachable on port `7080`. The CLI defaults to
`http://localhost:8080`, so point it at the local gateway:

```bash
export ZYNAX_API_URL=http://localhost:7080
```

Verify the stack is healthy — `/healthz` returns a small JSON status body:

```bash
curl http://localhost:7080/healthz
```

Useful endpoints:

| URL | What |
|-----|------|
| `http://localhost:7080` | api-gateway HTTP REST (`ZYNAX_API_URL`) |
| `http://localhost:7088` | Temporal Web UI — inspect workflow executions |

See [infra/docker-compose/README.md](../infra/docker-compose/README.md) for the full
port map and startup order.

---

## 4. (Optional) Start the observability stack

To see traces and logs in a UI, bring up the Uptrace overlay. Telemetry is **off by
default** and env-gated, so this step is optional — the workflow run works without it.

```bash
cp infra/docker-compose/observability/.env.observability.example \
   infra/docker-compose/observability/.env.observability
# edit the file — set a login + token (there are no committed defaults)

make obs-up    # Uptrace UI → http://localhost:7020
```

Then point the platform services at the collector and restart the stack so they export
telemetry:

```bash
export ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:7017
make run-local
```

The Uptrace login UI is at `http://localhost:7020` (use the login/token you set in the
`.env.observability` file). For the full telemetry guide see
[docs/observability/](observability/).

---

## 5. Apply the runnable workflow

The hero example is
[code-review-ollama.yaml](../spec/workflows/examples/code-review-ollama.yaml) — it runs
to completion from the CLI alone: its initial state dispatches the `codereview`
capability (served by the llm-adapter pointed at your local Ollama model) over a real git
diff, then transitions to a terminal state. Validate it, then submit:

```bash
zynax validate spec/workflows/examples/code-review-ollama.yaml      # static + data-flow checks
zynax apply --dry-run spec/workflows/examples/code-review-ollama.yaml   # compile without submitting
zynax apply spec/workflows/examples/code-review-ollama.yaml         # submit
```

> Other examples under [spec/workflows/examples/](../spec/workflows/examples/) (e.g.
> `code-review.yaml`) are **reference specs** that wait on external GitHub/review events —
> use them to learn the data-flow patterns, but they do not run to completion from the CLI
> alone.

`apply` prints the run identifier:

```
run_id: <run-id>
```

Copy that `run_id` — the next step uses it.

> **Starting from scratch?** `zynax init workflow my-pipeline -o my-pipeline.yaml`
> scaffolds a valid, versioned manifest from a template. Run `zynax validate
> my-pipeline.yaml` before applying.

---

## 6. Watch the run

Check the current status (exits `0` if terminal, `2` if still running):

```bash
zynax status workflow <run-id>
```

Tail the run live — the command follows lifecycle events (state transitions and
capability events) and exits once the workflow reaches a terminal state (`-f` is the
short flag; `--format json` emits one JSON object per event):

```bash
zynax logs <run-id> --follow
```

Print just the capability output — the model's review text — straight from the CLI:

```bash
zynax result <run-id>
```

For a full snapshot of a run (id, workflow, status, current state, version):

```bash
zynax get workflow <run-id>
```

For event-driven workflows (the reference `code-review.yaml` waits on review events),
inject an event from the CLI to drive the run forward:

```bash
zynax events publish <run-id> review.approved --data reviewer=alice
```

---

## 7. See the trace and logs (if observability is running)

Open the Uptrace UI at `http://localhost:7020`, log in, and find the trace for your run.
Traces, metrics, logs, and APM are correlated by `trace_id`/`span_id`, so you can jump
from a span to the matching log records for the same run.

---

## 8. Tear down

```bash
docker compose \
  -f infra/docker-compose/docker-compose.yml \
  -f infra/docker-compose/docker-compose.ollama.yml \
  down --remove-orphans      # or: make demo-clean

make obs-down                # stop the observability stack (if started)
```

---

## Command reference

Every command this guide shows is a real `zynax` subcommand. Verify the surface yourself
with `zynax --help` (and `zynax <cmd> --help`); the source of record is
[cmd/zynax/cmd/](../cmd/zynax/cmd/).

| Command | Purpose | Key flags |
|---------|---------|-----------|
| `zynax validate <file>` | Local schema + data-flow checks (no gateway) | `--schema-dir`, `--format text\|json` |
| `zynax init workflow\|expert [name]` | Scaffold a manifest from a template | `-o/--output`, `--template-dir` |
| `zynax apply <file>` | Submit a manifest to the gateway | `--dry-run`, `--engine` |
| `zynax status workflow <run-id>` | Status (exit 0 terminal, 2 running) | — |
| `zynax logs <run-id>` | Stream lifecycle events | `--follow/-f`, `--format text\|json` |
| `zynax result <run-id>` | Print the capability output (review text) | — |
| `zynax get workflow <run-id>` | Full run snapshot | — |
| `zynax delete workflow <run-id>` | Cancel/remove a run | — |
| `zynax events publish <run-id> <event-type>` | Inject an event into a running workflow | `--data key=value` (repeatable) |

---

## What next

- **[Developer Guide](developer-guide.md)** — the Make targets and daily workflow.
- **[Local Development Guide](local-dev.md)** — CLI install options and persona paths.
- **[Human-validation guide](contributing/human-validation-guide.md)** — how to validate a
  change actually runs (the standard the demo path is checked against).
- **[docs/casts/](casts/)** — recorded terminal walkthroughs of these flows.
- **[spec/workflows/examples/](../spec/workflows/examples/)** — the reference workflows and
  their data-flow patterns.
