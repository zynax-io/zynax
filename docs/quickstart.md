# Quick Start

Get from a fresh clone to a **traced workflow run** in minutes. Everything runs in
Docker — the only prerequisite is Docker Desktop (or Docker Engine + the Compose plugin).

By the end you will have:

1. The local platform stack running (`docker compose up` via `make run-local`).
2. A real workflow applied with `zynax apply`.
3. The run watched live with `zynax status` and `zynax logs --follow`.
4. (Optional) The trace and logs for that run visible in the Uptrace UI.

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

The CLI talks to the api-gateway over HTTP REST. Build it from source (requires Go
1.26.3) or grab a release binary:

```bash
make install-cli                       # builds → ~/bin/zynax (ensure ~/bin is on PATH)
# or download a pre-built binary — see docs/local-dev.md
```

Confirm it runs:

```bash
zynax --version
```

---

## 3. Start the local stack

A single command builds the images and starts the api-gateway, engine-adapter,
workflow-compiler, Temporal, and NATS:

```bash
make run-local    # docker compose up -d --build
```

When it finishes, the api-gateway is reachable on port `7080`. The CLI defaults to
`http://localhost:8080`, so point it at the local gateway:

```bash
export ZYNAX_API_URL=http://localhost:7080
```

Verify the stack is healthy:

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

## 5. Apply a real workflow

Three real, runnable reference workflows ship under
[spec/workflows/examples/](../spec/workflows/examples/). Validate one before applying:

```bash
zynax validate spec/workflows/examples/code-review.yaml      # static + data-flow checks
zynax apply --dry-run spec/workflows/examples/code-review.yaml   # compile without submitting
```

Then submit it:

```bash
zynax apply spec/workflows/examples/code-review.yaml
```

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
capability events) and exits once the workflow reaches a terminal state:

```bash
zynax logs <run-id> --follow
```

For a full snapshot of a run (id, workflow, status, current state, version):

```bash
zynax get workflow <run-id>
```

---

## 7. See the trace and logs (if observability is running)

Open the Uptrace UI at `http://localhost:7020`, log in, and find the trace for your run.
Traces, metrics, logs, and APM are correlated by `trace_id`/`span_id`, so you can jump
from a span to the matching log records for the same run.

---

## 8. Tear down

```bash
make stop-local    # stop and remove the platform stack
make obs-down      # stop the observability stack (if started)
```

---

## What next

- **[Developer Guide](developer-guide.md)** — the Make targets and daily workflow.
- **[Local Development Guide](local-dev.md)** — CLI install options and persona paths.
- **[spec/workflows/examples/](../spec/workflows/examples/)** — the three reference
  workflows and their data-flow patterns.
</content>
</invoke>
