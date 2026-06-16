<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-030 — Observability: OpenTelemetry Instrumentation with Uptrace Backend

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-16 |
| **Deciders** | Oscar Gómez Manresa |
| **Scope** | all services + adapters, `libs/zynaxotel`, observability compose + Helm — M7 EPIC O (#467) |
| **Related** | ADR-022 (event-bus/NATS — trace propagation over message headers), ADR-020 (mTLS — secure OTLP in-cluster), ADR-026 (Postgres distribution — backend storage option) |
| **Canvas** | `docs/spdd/467-observability-otel-uptrace/canvas.md` step O.1 |

---

## Context

M6 shipped a production platform with gRPC health and a Prometheus `/metrics`
surface, but there is no distributed tracing, no log correlation, and no UI to
view a workflow run. A run is a black box: the
`api-gateway → workflow-compiler → engine-adapter → task-broker → agent-registry → agent`
path is unobservable. There is no way to follow a single request end-to-end,
no scraped RED metrics on the request path, and no place where a developer can
read a log line in the context of the trace that produced it.

M7 must give a developer a single place to see **traces, metrics, logs, and an
APM/service map** — both **locally** (`docker compose up`) and **in-cluster**
(Helm) — without standing up a sprawling telemetry stack
(Jaeger + Loki + Elasticsearch + Grafana). Logs must be viewable in that UI
alongside traces and correlated by `trace_id`.

Both the **instrumentation standard** and the **backend** are one-way doors:
they shape every service's dependencies and the wire format of every span,
metric, and log record. Reversing either after services are instrumented means
re-instrumenting the whole platform. The decision is therefore recorded here,
before any instrumentation code (`libs/zynaxotel`, O.2+) is written.

---

## Decision

1. **OpenTelemetry** is the instrumentation standard across all services and
   adapters. No vendor SDKs — instrumentation emits OTLP and nothing more, so
   the backend stays swappable.
2. **Uptrace** is the default OTEL backend — a single, self-hostable stack
   serving **traces, metrics, logs, and APM** with a **login web UI** and a
   service map.
3. Default transport is **OTLP/gRPC**; OTLP/HTTP is optional. Telemetry is
   **off unless** `ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT` is set — zero overhead
   when unset, so the core stack never depends on the collector being up.
4. **Logs are shipped to Uptrace via OTLP logs** and correlated to traces by
   `trace_id`/`span_id`, so logs are viewable in the same UI as traces.
   `/metrics` is retained for the existing Prometheus scrape (RED metrics +
   exemplars).
5. **Sampling:** head-based parent sampling with a configurable ratio (100% in
   local dev). Tail-based sampling is explicitly out of scope (deferred to M8).
6. Uptrace ships in **both** a local `docker compose` overlay **and** a **Helm
   chart**, gated by an `observability.enabled` toggle, so logs/traces/APM are
   viewable in any environment with the login UI exposed.

### Non-goals

- **No Jaeger** — Uptrace is the single tracing sink.
- **No Loki** — logs go to Uptrace via OTLP logs, not a separate log store.
- **No Elasticsearch** — no separate search/index backend.
- **No tail-based sampling and no long-term retention tuning** — head sampling
  only at M7; both deferred to M8.
- **No implementation** — this ADR records the decision only. The shared
  library, instrumentation, compose overlay, and Helm chart are stories O.2–O.9
  under EPIC O (#467).

---

## Rationale

| Option | Assessment |
|--------|------------|
| **OTEL + Uptrace** (chosen) | ✅ One lightweight, self-hostable, CNCF-aligned backend for traces + metrics + logs + APM with a login UI and service map; one `docker compose up` or `helm install` gives the full picture |
| OTEL + Jaeger + Loki + Grafana | ✗ Rejected — a heavy multi-component stack (4 deployments + their storage) is more infra than the project needs at M7 and multiplies the local-dev footprint |
| Vendor APM SDKs (Datadog/New Relic agents) | ✗ Rejected — vendor lock-in; violates the no-custom-vendor-lock-in goal and is not self-hostable |
| Tracing only (no logs/metrics in the backend) | ✗ Rejected — the brief requires logs + APM correlated in one UI; tracing-only leaves logs uncorrelated |
| OTLP/HTTP as the default transport | ✗ Rejected as default — gRPC is the platform's native inter-service transport (ADR-001); HTTP is kept as an optional fallback |

---

## Consequences

### Positive

- The default DX is `Zynax → OpenTelemetry → Uptrace`: one `docker compose up`
  (or `helm install` with `observability.enabled`) yields a full telemetry UI
  with login, traces, metrics, logs, and a service map.
- OTEL-only emission keeps the backend swappable — replacing Uptrace later is a
  config change, not a re-instrumentation.
- Telemetry off-by-default means the core stack carries zero observability
  overhead and no hard dependency on the collector.

### Negative / trade-offs

- Uptrace's single-binary mode is not tuned for high-scale retention; scale
  retention and tail-based sampling are deferred to M8.
- A single backend is a single point of failure for observability — acceptable
  because telemetry is non-critical to the request path (async batch export,
  never blocking).

### Neutral / follow-up

- The OTLP endpoint must respect mTLS in-cluster (ADR-020).
- Trace context must propagate across Temporal activities and NATS headers
  (ADR-022) — covered by story O.5.
- Tail-based sampling and production retention tuning are M8 work.

---

### Follow-up stories (EPIC O, #467)

| Story | Work |
|-------|------|
| O.2 #1185 | Shared `libs/zynaxotel` package (providers + OTLP exporter + semconv) |
| O.3 #1186 | Instrument all 7 services (gRPC/HTTP interceptors) |
| O.4 #1187 | RED metrics + exemplars |
| O.5 #1188 | Trace propagation across Temporal + NATS |
| O.6 #1189 | Python adapter instrumentation |
| O.7 #1190 | Local Uptrace compose stack (login UI) |
| O.8 #1191 | Uptrace Helm chart |
| O.9 #1192 | Logs to Uptrace + naming conventions |
