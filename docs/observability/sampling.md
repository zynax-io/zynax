<!-- SPDX-License-Identifier: Apache-2.0 -->
# Sampling and Retention

> EPIC O (#467) · D.4 (#1220). How to tune how much telemetry Zynax keeps —
> trace sampling, collector batching, and storage retention (TTL).

There are three levers between a service and the bytes stored in ClickHouse:
**trace sampling** (what a service exports), **collector batching/limiting**
(how it is transported), and **retention TTL** (how long Uptrace keeps it).

## Trace sampling (at the service)

Zynax ships with **parent-based, always-sample** behaviour: the tracer provider
in `libs/zynaxobs` is configured with a batch span processor and no head sampler
override, so a service records every span when telemetry is enabled and honours
the upstream sampling decision carried in the W3C `traceparent` header.

- Because the sampling decision rides on `traceparent`, a single workflow run is
  sampled **consistently** across every gRPC, Temporal, and NATS hop — you never
  get a half-sampled trace.
- For local development this default is what you want: keep everything, debug
  with full fidelity.
- For higher-volume environments, reduce head sampling at the entry service
  (api-gateway) so the decision propagates downstream. Reducing volume at the
  root keeps traces *whole* — sampling independently per service would shred them.

> Telemetry is off entirely until `ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT` is set
> ([opentelemetry.md](opentelemetry.md)). "No sampling" is the cheapest setting:
> unset the variable.

## Collector batching and memory limiting

The OpenTelemetry Collector (`infra/docker-compose/observability/otel-collector.yaml`
locally; the Helm `otel-collector-config` in-cluster) protects both the service
request path and the collector itself:

| Processor | Setting | Why |
|-----------|---------|-----|
| `memory_limiter` | `limit_percentage: 80`, `spike_limit_percentage: 25` | Drops data before OOM under trace/log bursts. |
| `batch` | `timeout: 5s`, `send_batch_size: 10000` | Batches async so a service never blocks on export. |

This is the canvas Safeguard "never block the request path on the exporter".
Raise `send_batch_size` for higher throughput; lower `timeout` for fresher data
in the UI at the cost of more, smaller export requests. Logs are forwarded
**verbatim** — no processor rewrites or strips the OTLP `trace_id`/`span_id`, so
log↔trace correlation survives transport.

## Retention (TTL) in Uptrace

Uptrace stores spans, metrics, and logs in ClickHouse, each with its own TTL.

### Local (compose)

`infra/docker-compose/observability/uptrace.yml` pins a **one-week** TTL on every
signal:

```yaml
ch_schema:
  ttl: 168h     # 7 days
spans:   { ttl: 168h }
metrics: { ttl: 168h }
logs:    { ttl: 168h }
```

This is deliberately short — local telemetry is for debugging, not long-term
analysis. Long-term retention is deferred to M8. To keep data longer locally,
raise these TTLs and re-run `make obs-up` (a TTL change applies to newly
ingested data).

### In-cluster (Helm)

The Helm `uptrace-config` ConfigMap does **not** pin TTLs, so Uptrace applies its
own defaults. To enforce a retention policy in a cluster, add `spans`/`metrics`/
`logs` TTL blocks to the chart's Uptrace config and size ClickHouse persistence
(`clickhouse.persistence.size`) to match the retained volume.

## Choosing values

| Goal | Lever |
|------|-------|
| Cheapest / off | Unset `ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT`. |
| Full fidelity locally | Default (keep all spans), 168h TTL. |
| Lower volume, whole traces | Reduce head sampling at api-gateway; let `traceparent` propagate. |
| Longer history | Raise TTLs; grow ClickHouse persistence. |
| Burst resilience | Tune collector `memory_limiter` / `batch`. |

## See also

- [opentelemetry.md](opentelemetry.md) · [uptrace.md](uptrace.md) ·
  [troubleshooting.md](troubleshooting.md) · [naming-conventions.md](naming-conventions.md)
- ADR-030 — OTEL + Uptrace backend decision.
