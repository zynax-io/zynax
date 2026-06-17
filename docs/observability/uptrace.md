<!-- SPDX-License-Identifier: Apache-2.0 -->
# Running Uptrace

> EPIC O (#467) · D.4 (#1220). Run the Uptrace observability backend locally
> (Docker Compose, step O.7) and in-cluster (Helm, step O.8). Uptrace gives you
> traces, metrics, logs, APM, and a service map behind a single login UI.

Uptrace is the single backend for all three signals. Services and adapters never
talk to it directly — they export OTLP to an OpenTelemetry Collector, which
forwards traces, metrics, and logs to Uptrace. See
[opentelemetry.md](opentelemetry.md) for how signals are produced.

The stack is the same shape in both environments: **single-binary Uptrace +
ClickHouse (span/metric/log storage) + Postgres (metadata) + OTel Collector**.

## Credentials are never committed

The login user, project token, and Postgres password come **only** from
environment / a Secret — there are no committed default credentials. Each runner
fails fast (`${VAR:?...}` guards in compose, a required Secret in Helm) if a
value is missing.

## Local — Docker Compose (step O.7)

### 1. Create the env file

```bash
cp infra/docker-compose/observability/.env.observability.example \
   infra/docker-compose/observability/.env.observability
# edit infra/docker-compose/observability/.env.observability and fill in:
#   UPTRACE_ADMIN_EMAIL     — login UI user (any valid email)
#   UPTRACE_ADMIN_PASSWORD  — login UI password
#   UPTRACE_PROJECT_TOKEN   — opaque random string
#   PG_PASSWORD             — Uptrace metadata Postgres password
#   UPTRACE_DSN             — must embed the SAME token as UPTRACE_PROJECT_TOKEN
```

The real `.env.observability` is gitignored. The collector's `UPTRACE_DSN` token
**must equal** `UPTRACE_PROJECT_TOKEN`, or Uptrace rejects the forwarded data.

### 2. Bring it up

```bash
make obs-up        # Uptrace UI → http://localhost:7020
make obs-logs      # tail the observability stack
make obs-down      # stop and remove the stack
```

`make obs-up` refuses to start until `.env.observability` exists.

### 3. Point services at the collector

```bash
export ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:7017
```

Then start the platform (`make run-local`) and open the login UI at
<http://localhost:7020>.

### Host port map

| Host port | Bound to | Service | Purpose |
|-----------|----------|---------|---------|
| 7020 | `127.0.0.1` | Uptrace UI | Login UI — traces/metrics/logs/APM |
| 7017 | `127.0.0.1` | OTel Collector | OTLP/gRPC ingest (`ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT`) |
| 7018 | `127.0.0.1` | OTel Collector | OTLP/HTTP ingest |

**All observability host ports bind to `127.0.0.1` only** — OTLP ingest is never
publicly exposed. ClickHouse and Postgres have no host ports (container-internal).

## In-cluster — Helm (step O.8)

The `helm/charts/uptrace/` chart mirrors the compose stack: single-binary
Uptrace + ClickHouse + Postgres + OTel Collector, with persistence and pod
security context. The whole chart is gated by the umbrella
`observability.enabled` condition, so telemetry stays off by default and there is
no second tracing backend.

### 1. Create the credentials Secret (prerequisite)

No credentials are committed; the chart requires a pre-created Secret:

```bash
kubectl create secret generic zynax-uptrace-credentials \
  --namespace zynax \
  --from-literal=admin-email=<you AT example DOT com> \
  --from-literal=admin-password=<password> \
  --from-literal=project-token=<token> \
  --from-literal=pg-password=<pg-password>
```

The secret name and keys are configurable via `uptrace.existingSecret` and
`uptrace.secretKeys` in `values.yaml`.

### 2. Install

```bash
helm install zynax-uptrace helm/charts/uptrace/ --namespace zynax
```

### 3. Point services at the in-cluster collector

```
ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT=http://zynax-uptrace-otel-collector.<namespace>.svc.cluster.local:4317  # gitleaks:allow
```

(replace `<namespace>` with the release namespace, e.g. `zynax`.)

The collector Service's OTLP/gRPC port (`4317`) is the canonical in-cluster
ingest endpoint.

### 4. Reach the login UI

The OTLP collector Service is **never** exposed via Ingress. The login UI is
disabled at Ingress by default; reach it with a port-forward:

```bash
kubectl port-forward svc/zynax-uptrace 14318:14318 --namespace zynax
# then open http://localhost:14318
```

To expose the UI permanently, set `ingress.enabled=true` and configure
`ingress.host` plus auth/cert-manager annotations — the UI must stay auth-gated.

## Retention and tuning

Local compose pins a one-week TTL on spans, metrics, and logs; the Helm chart
uses Uptrace defaults. Tune retention and sampling in
[sampling.md](sampling.md). When telemetry does not appear in the UI, see
[troubleshooting.md](troubleshooting.md).

## See also

- `infra/docker-compose/docker-compose.observability.yml` — local stack.
- `helm/charts/uptrace/` — in-cluster chart.
- [opentelemetry.md](opentelemetry.md) · [sampling.md](sampling.md) ·
  [troubleshooting.md](troubleshooting.md) · [naming-conventions.md](naming-conventions.md)
- ADR-030 — OTEL + Uptrace backend decision.
