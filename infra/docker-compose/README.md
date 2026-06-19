# SPDX-License-Identifier: Apache-2.0
# Zynax — Docker Compose Files

All Compose files live in this directory:

| File | Purpose |
|------|---------|
| `docker-compose.yml` | Canonical local dev stack — used by `make run-local` / `make dev-up` |
| `docker-compose.services.yml` | Profiles-based overlay (testing/dev) — used by `make test-integration` |
| `docker-compose.tools.yml` | Test-runner container for Python integration tests |
| `docker-compose.test.yml` | CI overlay — disables persistent volumes for ephemeral test runs |
| `docker-compose.observability.yml` | Local Uptrace stack — traces/metrics/logs/APM with a login UI |
| `docker-compose.ollama.yml` | Zero-cost local LLM overlay — bundles `ollama` and repoints `llm-adapter` at it (no API key) |
| `docker-compose.eval-temporal.yml` | Day-0 overlay — swaps the durable Temporal trio for a single in-memory `temporal server start-dev` (no Postgres/UI container) |

## Local dev stack

Canonical stack for end-to-end testing of the three implemented platform services.

## Quick start

```bash
make run-local    # build images + start all services
make logs-local   # tail all logs
make stop-local   # stop and remove containers
```

## Port map

| Host port | Service | Purpose |
|-----------|---------|---------|
| 7080 | api-gateway | HTTP REST — `export ZYNAX_API_URL=http://localhost:7080` |
| 7233 | Temporal | gRPC (worker/SDK connections) |
| 7088 | Temporal Web UI | Workflow inspection — http://localhost:7088 |
| 7422 | NATS | Client port (optional direct access) |

Internal-only (no host port):

| Container port | Service |
|---------------|---------|
| 50054 | workflow-compiler gRPC |
| 50055 | engine-adapter gRPC |

## Service startup order

```
postgres (healthy) → temporal (healthy) → engine-adapter (healthy)
                                        → workflow-compiler (healthy) → api-gateway
```

## Observability stack (Uptrace)

A separate overlay brings up Uptrace (traces, metrics, logs, APM, service map) with
a login UI, backed by ClickHouse + Postgres and fronted by an OpenTelemetry Collector.

```bash
cp observability/.env.observability.example observability/.env.observability
# edit observability/.env.observability — set login + token (no committed defaults)
make obs-up        # Uptrace UI → http://localhost:7020
make obs-logs
make obs-down
```

Point services at the collector (telemetry is off by default — env-gated):

```bash
export ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:7017
```

| Host port | Service | Purpose |
|-----------|---------|---------|
| 7020 | Uptrace UI | Login UI — traces/metrics/logs/APM |
| 7017 | OTel Collector | OTLP/gRPC ingest (`ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT`) |
| 7018 | OTel Collector | OTLP/HTTP ingest |

All observability host ports bind to `127.0.0.1` only — OTLP ingest is never publicly
exposed (canvas O.7 Safeguards). ClickHouse and Postgres have no host ports.

Logs ship to Uptrace as OTLP log records through the collector `logs` pipeline,
correlated to traces by `trace_id`/`span_id`. Span, metric, and log naming rules
live in [docs/observability/naming-conventions.md](../../docs/observability/naming-conventions.md).

## Local Ollama overlay (zero-cost, offline LLM)

The shipped `llm-adapter` points at OpenAI/gpt-4o, which needs a paid key. This overlay
bundles an `ollama` service inside the compose network (nothing exposed to the host LAN),
reuses host-pulled models via a read-only bind mount, and repoints `llm-adapter` at it
with an Ollama provider config — no API key, no cost.

```bash
docker compose \
  -f infra/docker-compose/docker-compose.yml \
  -f infra/docker-compose/docker-compose.ollama.yml \
  up -d ollama llm-adapter
```

The overlay defaults the host models directory to a systemd `ollama` install
(`/usr/share/ollama/.ollama/models`). Override it for a different install path:

```bash
OLLAMA_HOST_MODELS=/path/to/.ollama/models docker compose \
  -f infra/docker-compose/docker-compose.yml \
  -f infra/docker-compose/docker-compose.ollama.yml \
  up -d ollama llm-adapter
```

The adapter config (`ollama/llm-adapter.config.yaml`) registers a `codereview`
capability against the default reference model `qwen2.5-coder:3b` — a small, fast,
code-focused local model chosen so the first-run demo is zero-cost and
deterministic. Pull it on the host before bringing the overlay up:

```bash
ollama pull qwen2.5-coder:3b
```

**Switching model/provider (one-line override):** edit the `model:` line under
`provider:` in `ollama/llm-adapter.config.yaml` (e.g. `model: llama3.2:3b`), or
point `provider.name` / `provider.ollama_base_url` at a different provider. The
model/provider is config-only and never travels in the workflow input payload
(ADR-013 / ADR-035), so any workflow stays portable across models. A full
human-validation guide standard is tracked in #1388.

## Eval-Temporal overlay (single in-memory binary)

By default the stack runs Temporal as **three containers** — `temporalio/auto-setup`,
a dedicated Postgres, and `temporalio/ui`. For a Day-0 evaluation that is overkill.
The `docker-compose.eval-temporal.yml` overlay replaces all three with **one**
self-contained binary, `temporal server start-dev` (embedded in-memory SQLite, no
external DB), and makes activities **fail fast** (`ZYNAX_ENGINE_MAX_ACTIVITY_ATTEMPTS=1`,
no durable retry loops). It reuses the proven `TemporalEngine` — no new engine to own
(this supersedes the rejected in-process `EvalEngine` of ADR-037 / #1359 for the compose
path).

```bash
# Bring the whole stack up on the lightweight Temporal:
docker compose \
  -f infra/docker-compose/docker-compose.yml \
  -f infra/docker-compose/docker-compose.eval-temporal.yml \
  up -d

# Or via the demo (one opt-in flag):
EVAL_TEMPORAL=1 make demo
```

The single binary's built-in Web UI is on the same host port as before — http://localhost:7088.

**Graduate to durable production Temporal — one flag, no edits:** simply drop the overlay
(use the base `docker-compose.yml` alone). That restores `auto-setup` + Postgres + the
standalone UI and the engine-adapter's default 3 retry attempts. If you need to run the
durable trio while the overlay is layered on, opt in by profile:

```bash
docker compose -f infra/docker-compose/docker-compose.yml \
  -f infra/docker-compose/docker-compose.eval-temporal.yml \
  --profile durable-temporal up -d
```

## Not included

The following services are unimplemented stubs awaiting M5 and are intentionally
omitted from this stack: `agent-registry`, `task-broker`, `memory-service`, `event-bus`.

## Verifying the stack

```bash
# All healthz probes
curl http://localhost:7080/healthz

# Apply an example workflow manifest
export ZYNAX_API_URL=http://localhost:7080
zynax apply spec/workflows/examples/code-review.yaml
```
