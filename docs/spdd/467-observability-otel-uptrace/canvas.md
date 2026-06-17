# REASONS Canvas — EPIC O: Observability (OTEL + Uptrace + Prometheus)

> **All content in this Canvas is Tier 1 (public-safe).** Tier 2 → `canvas.private.md`. Run `/spdd-security-review` before committing.

**Issue:** #467 (absorbed into EPIC O)
**Author:** M7 program plan · **Date:** 2026-06-15 · **Status:** Implemented (EPIC O complete 2026-06-17 — all O-steps merged incl. O.5 #1188, O.6 #1189, O.9 #1192; EPIC #467 closed)
**Supersedes:** the original "M7.A Wire Prometheus + OTel" canvas (child issue #491; folds in the M5.D #483 event-publish counter and #484 request-id propagation).

---

## R — Requirements

- **Problem:** there is no telemetry — no traces, no scraped metrics, no log correlation. A run is
  a black box; you cannot see the api-gateway → compiler → engine → broker → registry → agent path.
- A developer must see **traces, metrics, logs, and an APM/service-map** in **one UI with login**,
  both **locally** (`docker compose up`) and **in-cluster** (Helm).
- Logs must be **viewable in the UI** alongside traces (shipped via OTLP) and correlated by `trace_id`.
- **Done when:** one workflow run yields a connected trace across every hop **and its logs**, visible
  in the Uptrace login UI (compose + Helm); RED metrics scraped; service map populated.

---

## E — Entities

```
zynaxobs (shared lib)
├── TracerProvider / MeterProvider / LoggerProvider  ← OTLP/gRPC exporters
└── ResourceAttributes                                ← semconv (service.name, service.version, …)
Uptrace (backend)
├── UI (login)            ← traces · metrics · logs · APM · service map
├── OTLP ingest (gRPC)    ← default endpoint for all services + adapters
└── storage deps          ← (single-binary or ClickHouse/Postgres per deployment)
Prometheus /metrics       ← existing scrape surface (M6) + RED metrics + exemplars
├── zynax_grpc_requests_total{service,method,status}        ← counter, per incoming gRPC call
├── zynax_grpc_request_duration_seconds{service,method}     ← histogram, gRPC handler latency
└── zynax_eventbus_publish_failed_total{event_type}         ← counter, wired to the M5.D (#483) slog.Warn site
pprof (engine-adapter)    ← net/http/pprof on a separate admin port (perf investigation only)
```

> Metric names follow the `zynax_` prefix convention; labels stay low-cardinality
> (service/method/status only — never workflow or request IDs).

---

## A — Approach

**We will:**
- Standardise on **OpenTelemetry**; default backend **Uptrace** (traces+metrics+logs+APM, login UI).
- Default transport **OTLP/gRPC**; OTLP/HTTP optional. Off unless `ZYNAX_OTEL_EXPORTER_OTLP_ENDPOINT` set.
- Provide a shared `libs/zynaxobs` Go package and OTEL in the Python SDK.
- Propagate W3C `traceparent` across gRPC, Temporal activities, and NATS headers.
- Ship Uptrace in **both** a compose overlay **and** a Helm chart (`observability.enabled`), with the **login UI** exposed.
- Ship structured logs to Uptrace via **OTLP logs**; keep `/metrics` for Prometheus scrape.

**We will NOT:**
- Deploy Jaeger, Loki, or Elasticsearch (single backend; avoid sprawl).
- Implement tail-based sampling or long-term retention tuning — **deferred to M8**.

**Governing ADRs:** ADR-030 (OTEL + Uptrace — this EPIC), ADR-022 (event-bus/NATS), ADR-020 (mTLS — secure OTLP in-cluster).

---

## S — Structure (first S)

```
libs/zynaxobs/                                  ← providers, exporters, interceptors
services/*/cmd/*/main.go                          ← init providers; gRPC/HTTP interceptors
agents/sdk/src/zynax_sdk/                          ← OTEL traces+logs for capability handlers
infra/docker-compose/docker-compose.observability.yml  ← Uptrace + deps + collector + login UI (70xx)
infra/helm/charts/uptrace/                         ← Deployment/Service/Ingress + UI; values toggle
docs/observability/                                ← naming conventions, sampling, troubleshooting
```

Config env prefix: `ZYNAX_OTEL_` · Uptrace UI host port: 70xx (local).

---

## O — Operations (stories — `spdd-story` form)

**GitHub issues:** O.1 #1184 · O.2 #1185 · O.3 #1186 · O.4 #1187 · O.5 #1188 · O.6 #1189 · O.7 #1190 · O.8 #1191 · O.9 #1192 (epic #467)

**O.1 — ADR: OTEL + Uptrace (traces+metrics+logs+APM)** · S · `adr-proposal`
- As a `maintainer`, I want the backend + transport + sampling decision recorded so the stack is stable.
- AC: [ ] ADR-030 committed (Uptrace default, OTLP/gRPC, head sampling, logs-via-OTLP); [ ] non-goals listed. Deps: none.

**O.2 — Shared `libs/zynaxobs` package** · M · `feat`
- As a `service author`, I want one package for tracer/meter/logger providers so instrumentation is consistent.
- AC: [ ] providers + OTLP exporter + semconv resource attrs; [ ] no-op when endpoint unset; [ ] unit tests. Deps: O.1.

**O.3 — Instrument all 7 services** · M · `feat`
- As an `operator`, I want every gRPC/HTTP hop traced so requests are followable end-to-end.
- AC: [ ] server+client gRPC interceptors wired; [ ] api-gateway HTTP middleware; [ ] spans named `<service>.<rpc>`. Deps: O.2.

**O.4 — RED metrics + exemplars** · S · `feat`
- As an `operator`, I want rate/error/duration metrics with exemplars so dashboards link to traces.
- AC: [ ] RED metrics on gRPC + HTTP; [ ] exemplars carry `trace_id`; [ ] scraped at `/metrics`. Deps: O.2.

**O.5 — Trace propagation across Temporal + NATS** · M · `feat`
- As an `operator`, I want trace context preserved across the engine and event bus so the trace is unbroken.
- AC: [ ] `traceparent` in Temporal memo/headers + NATS message headers; [ ] connected trace across async hops. Deps: O.3.

**O.6 — Python adapter instrumentation** · M · `feat`
- As an `agent author`, I want capability handlers auto-traced + logs exported so agent work appears in the UI.
- AC: [ ] OTEL traces+logs in `agents/sdk`; [ ] `capability.<name>` spans; [ ] context extracted from inbound task. Deps: O.2, O.5.

**O.7 — Local Uptrace compose stack (login UI for logs/traces/APM)** · M · `feat`/`infra`
- As a `developer`, I want `docker compose up` to give me a UI for logs/traces/APM so I can see runs locally.
- AC: [ ] `docker-compose.observability.yml` runs Uptrace + deps + collector; [ ] login UI on a 70xx port; [ ] a run's traces+logs visible. Deps: O.2.

**O.8 — Uptrace Helm chart** · M · `feat`/`infra`
- As an `operator`, I want Uptrace deployable in-cluster so logs/traces are visible in any environment.
- AC: [ ] `infra/helm/charts/uptrace/` (Deployment/Service/Ingress + UI); [ ] `observability.enabled` toggle; [ ] services point at in-cluster OTLP endpoint; [ ] `helm lint` green. Deps: O.7.

**O.9 — Logs to Uptrace + naming conventions** · S · `feat`/`docs`
- As an `operator`, I want structured logs in the UI correlated to traces so I can debug from one place.
- AC: [ ] logs shipped via OTLP logs; [ ] every log line carries `trace_id`/`span_id`; [ ] span/metric naming doc committed. Deps: O.3, O.7.

**Order:** O.1 → O.2 → {O.3, O.4, O.7} → {O.5, O.8, O.9} → O.6.

---

## N — Norms
- `Signed-off-by:` + `Assisted-by:` per commit; one logical change per commit.
- OTEL semantic conventions for resource + span attributes; no custom vendor lock-in.
- Telemetry **off by default** (env-gated) — zero overhead when disabled.
- `helm lint` gate for the new chart; `GOWORK=off` for service `go` commands (ADR-017).

## S — Safeguards (second S)

### Context Security
- [x] No Tier 2 content (the compose/Helm values use placeholders, not real hostnames/credentials)
- [x] No PII in span/log attributes; redact request payloads by default
- [x] No prompt-injection phrasing
- [x] `/spdd-security-review` — result: PASS-with-flags (see `SECURITY-REVIEW.md`); telemetry-pipeline flags bound below

### Feature Safeguards
- Never emit secrets/credentials/tokens into spans, metrics, or logs — redact by default.
- Never make telemetry mandatory for the core stack — must run with observability disabled.
- Never add a second tracing backend — Uptrace is the single sink (avoid Jaeger/Loki/ES sprawl).
- Never block the request path on the exporter — use batch/async export.
- **Uptrace login credentials** (compose + Helm) must come from `.env` / Helm secret values — never committed defaults; no hard-coded admin password in `docker-compose.observability.yml` or chart `values.yaml`. (O.7, O.8)
- **OTLP ingest must not be publicly exposed:** compose binds the OTLP/UI ports to `127.0.0.1`; in-cluster OTLP is mTLS-secured per ADR-020 and the Ingress for the login UI is auth-gated. (O.7, O.8)
- **`traceparent` is the only context propagated** across gRPC/Temporal/NATS — never inject auth tokens or session data into trace headers/memos. (O.5)
- **Metric labels must stay low-cardinality** — only `service`/`method`/`status`; never embed workflow IDs, request IDs, or other unbounded values. (O.4)
- **pprof (engine-adapter)** must bind to a separate admin port, never the production HTTP/API port — no accidental public exposure. (O.3)
