<!-- SPDX-License-Identifier: Apache-2.0 -->
# Telemetry Naming Conventions

> EPIC O (#467), step O.9. Governs span, metric, and log naming across every
> Zynax service and adapter so traces, metrics, and logs join up in the Uptrace
> UI. Telemetry is **off by default** — these conventions apply only when
> `ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT` is set.

All telemetry follows the [OpenTelemetry semantic conventions][semconv]; the
rules below are the Zynax-specific bindings on top of them. They exist so a
single workflow run produces a connected trace, RED metrics, and correlated logs
that a developer can pivot between from one place.

## Resource attributes

Every signal (trace, metric, log) carries the same resource attributes, set once
in `libs/zynaxobs` and the Python SDK:

| Attribute | Source | Example |
|-----------|--------|---------|
| `service.name` | per service | `api-gateway`, `engine-adapter` |
| `service.version` | build version | `0.6.0` |
| `service.namespace` | deployment | `zynax` |
| `deployment.environment` | env-gated | `local`, `staging`, `prod` |

These come from `semconv` constants — never hand-rolled string keys.

## Span names

Spans use a lowercase, dot-delimited `<producer>.<operation>` form. Never embed
high-cardinality values (IDs, payloads) in the **span name** — put those in span
attributes instead.

| Producer | Pattern | Example |
|----------|---------|---------|
| gRPC server/client hop | `<service>.<rpc>` | `engine-adapter.DispatchCapability` |
| api-gateway HTTP route | `<service>.<method> <route>` | `api-gateway.POST /v1/workflows` |
| Python capability handler | `capability.<name>` | `capability.git.clone` |
| Temporal workflow/activity | `<workflow>.<activity>` | `IRInterpreter.DispatchCapability` |

## Metric names

Metrics use the `zynax_` prefix and OpenTelemetry unit suffixes. **Labels stay
low-cardinality — only `service`, `method`, `status`.** Never label with workflow
IDs, request IDs, or any unbounded value (canvas O.4 Safeguard).

| Metric | Type | Labels |
|--------|------|--------|
| `zynax_grpc_requests_total` | counter | `service`, `method`, `status` |
| `zynax_grpc_request_duration_seconds` | histogram | `service`, `method` |
| `zynax_eventbus_publish_failed_total` | counter | `event_type` |

Histograms carry **exemplars** that link to a `trace_id`, so a dashboard spike
jumps straight to the offending trace.

## Log correlation (trace_id / span_id)

Logs are shipped to Uptrace as **OTLP log records** through the OpenTelemetry
Collector (`logs` pipeline), not scraped from stdout. The single most important
rule:

> **Every log line emitted inside an active span MUST carry `trace_id` and
> `span_id`.**

How this is guaranteed end-to-end:

1. **Emit** — services log through the `libs/zynaxobs` log bridge (and the Python
   SDK through the OTEL logging handler), which stamps the current span context
   onto each `LogRecord`'s `trace_id` / `span_id` fields from the OTLP log data
   model. These are first-class fields, not free-text attributes.
2. **Transport** — the OpenTelemetry Collector `logs` pipeline (`otlp` receiver →
   `memory_limiter` → `batch` → `otlp/uptrace` exporter) forwards records
   verbatim. No processor rewrites or strips the trace/span identifiers.
3. **Display** — Uptrace joins logs to traces on `trace_id`, so a log line in the
   UI links to its span and vice versa.

Conventions for log records:

- Use structured key/value fields, never string interpolation of IDs into the
  message body.
- Reuse the resource attributes above; do not duplicate `service.name` into the
  message.
- **Never** log secrets, credentials, tokens, or raw request payloads — redact by
  default (canvas O.9 Safeguard).
- Severity uses the OTLP severity model (`DEBUG`/`INFO`/`WARN`/`ERROR`); the
  collector preserves it.

## Context propagation

The only context propagated across gRPC, Temporal, and NATS is the W3C
`traceparent` header — never auth tokens or session data (canvas O.5 Safeguard).
This is what keeps `trace_id` stable across async hops so logs from different
services correlate to one run.

## See also

- [opentelemetry.md](opentelemetry.md) — what is emitted and how to enable telemetry
- [uptrace.md](uptrace.md) — run the Uptrace backend (compose + Helm, login UI)
- [sampling.md](sampling.md) — sampling and retention tuning
- [troubleshooting.md](troubleshooting.md) — when signals do not show up
- `infra/docker-compose/docker-compose.observability.yml` — local Uptrace + collector stack (O.7)
- `helm/charts/uptrace/` — in-cluster Uptrace + collector chart (O.8)
- ADR-030 — OTEL + Uptrace backend decision

[semconv]: https://opentelemetry.io/docs/specs/semconv/
