<!-- SPDX-License-Identifier: Apache-2.0 -->
# OpenTelemetry in Zynax

> EPIC O (#467) · D.4 (#1220). How Zynax services and adapters emit traces,
> metrics, and logs over OTLP, and how to turn telemetry on.

Zynax instruments every Go service and Python adapter with OpenTelemetry. All
three signals — traces, metrics, logs — leave a process as **OTLP** and are
forwarded by an OpenTelemetry Collector to [Uptrace](uptrace.md). Naming rules
that make those signals join up live in
[naming-conventions.md](naming-conventions.md).

## Telemetry is off by default

A service runs with **zero exporter overhead and no collector required** until
you point it at one. The single switch is one environment variable:

```bash
export ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:7017
```

- **Unset** — every provider is a no-op. No OTLP connections are opened, the
  request path is untouched. This is the default for `make run-local`.
- **Set** — the shared OTel package wires real OTLP/gRPC tracer, meter, and
  logger providers, installs them as the OTel globals, and registers the W3C
  trace-context propagator.

This is the *only* variable required to enable telemetry. It is consumed
identically by the Go services (via `libs/zynaxobs`) and the Python SDK, so the
same value works for the whole stack.

| Variable | Purpose | Example |
|----------|---------|---------|
| `ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP/gRPC collector endpoint. Unset ⇒ telemetry off. | `http://localhost:7017` (compose) · `http://zynax-uptrace-otel-collector.<namespace>.svc.cluster.local:4317` (Helm) <!-- gitleaks:allow --> |

> The endpoint must point at an **OTLP/gRPC collector**, not at Uptrace
> directly. The collector batches and memory-limits the export so a service
> request path never blocks on the backend.

## What is emitted

The shared OTel wiring (`libs/zynaxobs` for Go, the Python SDK for adapters)
produces all three signals from a single resource, so they share `service.name`,
`service.version`, and the other resource attributes from
[naming-conventions.md](naming-conventions.md#resource-attributes).

### Traces

- gRPC server/client hops between services (e.g. `engine-adapter.DispatchCapability`).
- api-gateway HTTP routes (e.g. `api-gateway.POST /v1/workflows`).
- Temporal workflow + activity spans (e.g. `IRInterpreter.DispatchCapability`).
- Python capability handler spans (e.g. `capability.git.clone`).

The W3C `traceparent` header is propagated across gRPC, Temporal, and NATS, so
one workflow run is a single connected trace end-to-end. Only `traceparent` is
propagated — never auth tokens or session data.

### Metrics

RED-style metrics with the `zynax_` prefix and low-cardinality labels
(`service`, `method`, `status` only):

- `zynax_grpc_requests_total` — counter.
- `zynax_grpc_request_duration_seconds` — histogram, carries **exemplars**
  linking a latency bucket to a `trace_id`.
- `zynax_eventbus_publish_failed_total` — counter.

Never label a metric with workflow IDs, request IDs, or any unbounded value.

### Logs

Logs are emitted as **OTLP log records** (not scraped from stdout). Every log
line written inside an active span carries `trace_id` and `span_id` as
first-class OTLP fields, so Uptrace joins a log to its trace and back. Secrets,
credentials, tokens, and raw request payloads are never logged — redact by
default.

## How the wiring works (Go)

`libs/zynaxobs` owns the wiring. `InitProviders` reads
`ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT`; if it is empty it installs no-op providers
and a no-op shutdown, otherwise it builds OTLP/gRPC exporters for all three
signals, batches them, installs the globals, and returns a `shutdown` func the
caller defers to flush on exit.

```go
providers, shutdown, err := zynaxobs.InitProviders(ctx, "api-gateway", version)
if err != nil { /* ... */ }
defer shutdown(ctx)
```

Resource attributes (`service.name`, `service.version`) come from `semconv`
constants — never hand-rolled string keys — so they stay identical across
services. The Python SDK mirrors this contract for adapters.

## Verifying telemetry is flowing

1. Start the [Uptrace stack](uptrace.md) (`make obs-up`).
2. Export the endpoint and (re)start the platform services:

   ```bash
   export ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:7017
   make run-local
   ```
3. Drive a request (e.g. `zynax apply spec/workflows/examples/code-review.yaml`).
4. Open the Uptrace UI at <http://localhost:7020> and confirm a trace with
   correlated logs appears.

If nothing shows up, see [troubleshooting.md](troubleshooting.md).

## See also

- [uptrace.md](uptrace.md) — run the Uptrace backend (compose + Helm).
- [sampling.md](sampling.md) — sampling and retention tuning.
- [naming-conventions.md](naming-conventions.md) — span/metric/log naming.
- ADR-030 — OTEL + Uptrace backend decision.
- `libs/zynaxobs/providers.go` — the Go provider wiring.
