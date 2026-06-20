<!-- SPDX-License-Identifier: Apache-2.0 -->

# Running Zynax with raw `docker compose`

Everything `make demo` does, as plain `docker compose` + `zynax` commands — no `make` required.
Start with the simplest run; the rest is optional depth.

> For the gentle, narrated walkthrough see [quickstart.md](quickstart.md). This page is the
> no-`make` operator reference.

---

## Simplest run

Three steps to a real code review. (Prereqs: Docker, the `zynax` CLI, and
`ollama pull qwen2.5-coder:3b` — see [Prerequisites](#prerequisites).)

```bash
# 1) boot the two demo services (their dependencies start automatically)
docker compose \
  -f infra/docker-compose/docker-compose.yml \
  -f infra/docker-compose/docker-compose.ollama.yml \
  up -d --build --wait api-gateway llm-adapter

# 2) apply a workflow — copy the run_id it prints
export ZYNAX_API_URL=http://localhost:7080
zynax apply spec/workflows/examples/code-review-ollama.yaml
#   → run_id: wf-abcd1234...

# 3) print the model's review
zynax result wf-abcd1234...
```

What each step does: (1) builds current images and starts only the api-gateway + llm-adapter and the
services they depend on; (2) compiles and submits the workflow over the gateway's REST API;
(3) tails the run until it finishes and prints the capability output.

Tear it down when finished:

```bash
docker compose \
  -f infra/docker-compose/docker-compose.yml \
  -f infra/docker-compose/docker-compose.ollama.yml \
  down -v --remove-orphans
```

> The whole `docker compose -f … -f …` prefix repeats a lot. Save it once:
> `DC="docker compose -f infra/docker-compose/docker-compose.yml -f infra/docker-compose/docker-compose.ollama.yml"`
> then use `$DC up …` / `$DC down …`.

---

## Prerequisites

- **Docker** Engine + the Compose plugin (`docker compose version`).
- **The `zynax` CLI** — build it from the repo (Go 1.26.3):
  ```bash
  cd cmd/zynax && GOWORK=off go build -trimpath -o ~/bin/zynax .   # ensure ~/bin is on your PATH
  ```
  or download a release binary (see the root `README.md` § *zynax CLI*).
- **A local model** on the host (the ollama container reuses host models read-only):
  ```bash
  ollama pull qwen2.5-coder:3b
  ```
  To use a model you already have, set `model:` in
  `infra/docker-compose/ollama/llm-adapter.config.yaml` and pull that one instead.

---

## More workflows

Same boot as above — just change the `zynax apply` target (and reuse `$DC` from the tip above):

```bash
# Two-step data-flow: review, then categorize (type) + rank (severity) the issues
zynax apply spec/workflows/examples/code-review-rank-ollama.yaml

# Watch every step live (run right after apply, not after it finishes)
zynax logs <run_id> --follow

# Review a real GitHub PR's diff — read-only, never writes to the PR (needs `gh`)
tools/pr-review-workflow.sh 1446 > /tmp/review.yaml
zynax apply /tmp/review.yaml

# Declarative scenario (injects spec/scenarios/code-review/diff.patch as context)
zynax apply spec/scenarios/code-review
```

`zynax result` prints the final capability output; `zynax logs --follow` streams **every** state and
each step's output as it completes (per step, not per token). `zynax status workflow <run_id>` reports
progress.

---

## The full platform (optional)

To run all seven services plus the five adapters (not just the demo two), use the base file alone:

```bash
export GITHUB_TOKEN=<token>   # the git- and ci-adapters exit(1) without it
docker compose -f infra/docker-compose/docker-compose.yml up -d --build
docker compose -f infra/docker-compose/docker-compose.yml logs -f      # tail
docker compose -f infra/docker-compose/docker-compose.yml down -v --remove-orphans
```

| Host port | Service |
|-----------|---------|
| 7080 | api-gateway (HTTP REST — `ZYNAX_API_URL`) |
| 7088 | Temporal Web UI |
| 7233 | Temporal gRPC |
| 7422 | NATS |

**Lighter Temporal** — swap the durable Temporal trio (auto-setup + Postgres + UI) for a single
in-memory `temporal server start-dev` by layering the eval overlay onto any of the above:

```bash
docker compose \
  -f infra/docker-compose/docker-compose.yml \
  -f infra/docker-compose/docker-compose.eval-temporal.yml \
  up -d --build --wait api-gateway llm-adapter
```

---

## Observability (optional)

Bring up the Uptrace stack (traces, metrics, logs, APM, login UI):

```bash
cp infra/docker-compose/observability/.env.observability.example \
   infra/docker-compose/observability/.env.observability
# edit the five change-me values: UPTRACE_ADMIN_EMAIL / UPTRACE_ADMIN_PASSWORD /
# UPTRACE_PROJECT_TOKEN / PG_PASSWORD / UPTRACE_DSN

docker compose \
  --env-file infra/docker-compose/observability/.env.observability \
  -f infra/docker-compose/docker-compose.observability.yml up -d
# Uptrace UI → http://localhost:7020 ; OTLP/gRPC → localhost:7017
```

Wiring the platform services to emit to it (OTEL endpoint, sampling, etc.) is covered in
[observability/](observability/) — see `opentelemetry.md` and `uptrace.md`.

---

## Reference

**Compose files** (all under `infra/docker-compose/`):

| File | Purpose |
|------|---------|
| `docker-compose.yml` | Base stack — all services, adapters, Temporal, NATS |
| `docker-compose.ollama.yml` | Zero-cost local-LLM overlay (no API key) |
| `docker-compose.eval-temporal.yml` | Single in-memory `temporal server start-dev` (no Postgres/UI) |
| `docker-compose.observability.yml` | Local Uptrace stack |

**Key environment variables:**

| Variable | Used for |
|----------|----------|
| `ZYNAX_API_URL` | the `zynax` CLI → gateway (`http://localhost:7080`) |
| `GITHUB_TOKEN` | git-adapter + ci-adapter (full platform only) |
| `OPENAI_API_KEY` | only if you drop the ollama overlay and use the baked OpenAI default |

**Teardown** always uses `down -v --remove-orphans` (the `-v` clears the persistent Postgres-backed
registry so the next run starts clean).

> **Intermittent boot race:** `up --wait` can occasionally report a container `exited (0)` /
> `dependency failed to start` during startup. It's transient — just run the command again.

See also: [quickstart.md](quickstart.md) · [infra/docker-compose/README.md](../infra/docker-compose/README.md).
