<!-- SPDX-License-Identifier: Apache-2.0 -->
# Observability Troubleshooting

> EPIC O (#467) · D.4 (#1220). When traces, metrics, or logs do not show up in
> Uptrace, work down this list. Most issues are the endpoint variable, the
> project token, or the collector connection.

Start from the signal flow — a problem is always at one of these hops:

```
service/adapter ──OTLP/gRPC──▶ OTel Collector ──OTLP──▶ Uptrace (ClickHouse + Postgres) ──▶ login UI
```

## Nothing appears in the UI at all

1. **Is telemetry enabled?** Telemetry is off until the endpoint is set:

   ```bash
   echo "$ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT"
   ```

   Empty ⇒ every provider is a no-op and nothing is exported. Set it
   (`http://localhost:7017` for compose) and **restart the service** — the
   variable is read once at startup.

2. **Is the stack up and healthy?**

   ```bash
   make obs-logs        # look for collector/uptrace/clickhouse/postgres errors
   ```

   Uptrace depends on ClickHouse and Postgres being healthy first; the collector
   depends on Uptrace being healthy. A collector that started before Uptrace will
   reconnect, but a crash-looping ClickHouse blocks everything.

3. **Did you point at the collector, not Uptrace?** The endpoint must be the
   **collector** OTLP/gRPC port (`7017` locally / `:4317` in-cluster), not the
   Uptrace UI port. Exporting straight to Uptrace bypasses batching and will not
   work.

## Stack will not start

- **`make obs-up` errors about a missing env file** — copy the example:

  ```bash
  cp infra/docker-compose/observability/.env.observability.example \
     infra/docker-compose/observability/.env.observability
  ```

- **A container exits citing `set <VAR> in .env.observability`** — a required
  credential (`PG_PASSWORD`, `UPTRACE_ADMIN_*`, `UPTRACE_PROJECT_TOKEN`,
  `UPTRACE_DSN`) is unset. There are no committed defaults; fill them all in.

- **Helm: pods stuck / Uptrace `CreateContainerConfigError`** — the credentials
  Secret is missing. Create `zynax-uptrace-credentials` (see
  [uptrace.md](uptrace.md#1-create-the-credentials-secret-prerequisite)).

## Traces arrive but logs/metrics do not (or token mismatch)

- **Token mismatch** — the collector's `UPTRACE_DSN` token must equal
  `UPTRACE_PROJECT_TOKEN`. If they differ, Uptrace silently rejects forwarded
  data. Make them identical and restart the stack.

- **Metrics missing** — metrics are exported on a periodic reader, so allow one
  export interval before they appear. Confirm the service actually records
  instruments (`zynax_grpc_requests_total`, etc.).

## Logs are not correlated to traces

Every log line inside a span should carry `trace_id`/`span_id`. If logs appear
unlinked:

- Confirm the log was emitted **inside an active span** — a log written before a
  span starts (or after it ends) has no context to stamp.
- Confirm the service logs through the `libs/zynaxobs` log bridge (Go) or the
  OTEL logging handler (Python) — stdout `print`/`fmt.Println` is not OTLP and is
  never correlated.
- The collector forwards log records verbatim; it never strips identifiers, so a
  missing `trace_id` is an emit-side issue, not transport.

See [naming-conventions.md](naming-conventions.md#log-correlation-trace_id--span_id).

## Traces are broken across services

A trace that splits into disconnected pieces means context propagation broke:

- Only the W3C `traceparent` header is propagated across gRPC, Temporal, and
  NATS. If a custom client drops headers, the downstream span starts a new trace.
- Independent per-service sampling shreds traces — sample at the root
  (api-gateway) and let the decision propagate. See [sampling.md](sampling.md).

## Cannot reach the login UI

- **Compose** — the UI binds `127.0.0.1:7020` only (never public). Browse from
  the host, or set up an SSH tunnel; it is not reachable from another machine by
  design.
- **Helm** — Ingress is off by default; port-forward `svc/zynax-uptrace` (see
  [uptrace.md](uptrace.md#4-reach-the-login-ui)).
- **Login rejected** — the user is created on first start from
  `UPTRACE_ADMIN_EMAIL`/`UPTRACE_ADMIN_PASSWORD`. If you changed them after first
  boot, the original Postgres metadata still holds the old user; recreate the
  metadata volume or update the user in the UI.

## See also

- [opentelemetry.md](opentelemetry.md) · [uptrace.md](uptrace.md) ·
  [sampling.md](sampling.md) · [naming-conventions.md](naming-conventions.md)
- ADR-030 — OTEL + Uptrace backend decision.
