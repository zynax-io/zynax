# ADR-030: Observability — OpenTelemetry instrumentation with Uptrace backend

**Status:** Proposed  **Date:** 2026-06-15
**Related:** ADR-022 (event-bus/NATS), ADR-020 (mTLS), ADR-026 (Postgres distribution)

---

## Context

M6 shipped a production platform with gRPC health and a Prometheus `/metrics` surface, but there is no
distributed tracing, no log correlation, and no UI to view a workflow run. A run is a black box: the
api-gateway → compiler → engine → broker → registry → agent path is unobservable. M7 must give a
developer a single place to see **traces, metrics, logs, and an APM/service map** — locally and
in-cluster — without standing up a sprawling telemetry stack (Jaeger + Loki + Elasticsearch + Grafana).
The instrumentation standard and backend are a one-way door (they shape every service's dependencies).

## Decision

1. **OpenTelemetry** is the instrumentation standard across all services and adapters (no vendor SDKs).
2. **Uptrace** is the default OTEL backend — a single binary/stack serving **traces, metrics, logs, and
   APM** with a **login web UI** and service map.
3. Default transport is **OTLP/gRPC**; OTLP/HTTP is optional. Telemetry is **off unless**
   `ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT` is set (zero overhead when unset).
4. **Logs are shipped to Uptrace via OTLP logs** and correlated to traces by `trace_id`/`span_id`, so
   logs are viewable in the same UI as traces.
5. **Sampling:** head-based parent sampling (configurable ratio; 100% in local dev).
6. Uptrace ships in **both** a local `docker compose` overlay **and** a **Helm chart**
   (`observability.enabled`), so logs/traces/APM are viewable in any environment.
7. We will **not** deploy Jaeger, Loki, or Elasticsearch.

## Rationale

| Option | Assessment |
|--------|------------|
| OTEL + Uptrace (chosen) | ✅ One lightweight, self-hostable, CNCF-aligned backend for traces+metrics+logs+APM with a UI |
| OTEL + Jaeger + Loki + Grafana | ✗ Rejected — heavy multi-component stack; more infra than the project needs at M7 |
| Vendor APM SDKs | ✗ Rejected — vendor lock-in; violates the no-custom-lock-in goal |
| Tracing only (no logs/metrics in backend) | ✗ Deferred — the brief requires logs+APM in one UI |

## Consequences

- **Positive:** one `docker compose up` (or `helm install` with `observability.enabled`) gives a full
  telemetry UI with login; the default DX is `Zynax → OpenTelemetry → Uptrace`.
- **Negative / trade-off:** Uptrace's single-binary mode is not tuned for high-scale retention — scale
  retention/sampling is deferred to M8.
- **Neutral / follow-up:** OTLP endpoint must respect mTLS in-cluster (ADR-020); tail-based sampling and
  production retention are M8 work.
