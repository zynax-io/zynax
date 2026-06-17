<!-- SPDX-License-Identifier: Apache-2.0 -->
# Observability naming conventions

> EPIC O (#467), step O.9 (#1192). Backing decision: [ADR-030](../adr/ADR-030-observability-uptrace.md).
> Shared implementation: `libs/zynaxobs` (Go) and `agents/sdk/src/zynax_sdk/telemetry.py` (Python).

This document is the single source of truth for **span**, **metric**, **log**, and
**resource** naming across every Zynax service and adapter. Telemetry is **off by
default** — providers are no-ops until `ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT` is set,
so these conventions apply only when an OTLP collector is configured.

All three signals (traces, metrics, logs) flow OTLP/gRPC to the OpenTelemetry
Collector and on to Uptrace. The collector pipelines that carry them are defined in
[`infra/docker-compose/observability/otel-collector.yaml`](../../infra/docker-compose/observability/otel-collector.yaml)
(local) and
[`helm/charts/uptrace/templates/otel-collector-config.yaml`](../../helm/charts/uptrace/templates/otel-collector-config.yaml)
(in-cluster).

---

## Resource attributes

Every span, metric, and log record carries the same resource attributes, built once
by `zynaxobs.NewResource` (Go) / the SDK `Resource` (Python) using OpenTelemetry
semantic conventions — never custom vendor keys:

| Attribute | Source | Example |
|-----------|--------|---------|
| `service.name` | per-service constant | `workflow-compiler` |
| `service.version` | build version | `0.6.0` |

These attributes are what Uptrace uses to group traces, metrics, and logs into a
single service in the APM / service map.

---

## Span names

gRPC spans use the canvas O.3 convention `<service>.<rpc>` — the proto package
prefix is dropped so the leaf name stays short (`zynaxobs.spanName`):

| Source | Span name |
|--------|-----------|
| gRPC server / client | `<Service>.<Rpc>` — e.g. `WorkflowCompilerService.Compile` |
| api-gateway HTTP entry | `<METHOD> <route>` — e.g. `POST /v1/workflows` |
| Python capability handler | `capability.<name>` — e.g. `capability.git.clone` |

- Span **kind** is set explicitly: `SERVER` on inbound, `CLIENT` on outbound.
- The W3C `traceparent` is the **only** context propagated across gRPC, Temporal,
  and NATS hops (`zynaxobs.InjectMapHeader` / `ExtractMapHeader`). Never inject
  auth tokens, session data, or PII into trace headers, memos, or span attributes.

---

## Metric names and labels

All metrics use the `zynax_` prefix. **Labels stay low-cardinality** —
`service`/`method`/`status` only. Never embed workflow IDs, request IDs, route
templates, or any unbounded value as a label (canvas Safeguards, O.4).

| Metric | Type | Labels |
|--------|------|--------|
| `zynax_grpc_requests_total` | counter | `service`, `method`, `status` |
| `zynax_grpc_request_duration_seconds` | histogram | `service`, `method` |
| `zynax_http_requests_total` | counter | `service`, `method`, `status` |
| `zynax_http_request_duration_seconds` | histogram | `service`, `method` |
| `zynax_eventbus_publish_failed_total` | counter | `event_type` |

- `method` for HTTP is the **verb** (`GET`/`POST`), never the URL path — paths are
  unbounded and would explode series cardinality.
- `trace_id` is attached as a Prometheus **exemplar**, never as a label, so a
  metric sample links back to a representative trace in Uptrace while series
  cardinality stays bounded.
- Metrics are scraped at `/metrics` (OpenMetrics exposition for exemplars) **and**
  pushed via OTLP; both surfaces use the same names.

---

## Log records and trace correlation

Structured logs are shipped to Uptrace via the **OTLP logs** pipeline — the
`LoggerProvider` (Go: `zynaxobs.InitProviders`, `otlploggrpc` batch exporter;
Python: `LoggerProvider` + `LoggingHandler` in `telemetry.py`) — not scraped from
stdout. The same batch/async export keeps the request path off the exporter.

Every log record emitted inside an active span automatically carries:

| Field | Meaning |
|-------|---------|
| `trace_id` | the enclosing trace — links the log to its trace in the UI |
| `span_id` | the enclosing span — pinpoints the exact operation |

Correlation is automatic: the OTLP log bridge reads the active span context, so any
`slog` call (Go) or stdlib `logging` call (Python) made while a span is in flight is
correlated by `trace_id`/`span_id` in Uptrace. Do not log secrets, credentials,
tokens, or raw request payloads — redact by default (canvas Safeguards).

---

## Checklist for new instrumentation

- [ ] Span name follows `<service>.<rpc>` (gRPC) or `capability.<name>` (handler).
- [ ] Metric name uses the `zynax_` prefix and only `service`/`method`/`status` labels.
- [ ] `trace_id` rides as an exemplar, never a label.
- [ ] Logs go through the OTLP `LoggerProvider`, never a second sink.
- [ ] No secrets, tokens, PII, or unbounded IDs in any span/metric/log attribute.
- [ ] Works with telemetry **disabled** (endpoint unset → no-op, zero overhead).
