<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M7.A Wire Prometheus + OTel

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #467
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-18
**Status:** Aligned

**Child issues:** #491 (Prometheus metrics + OTel tracing)

---

## R — Requirements

**Problem:** `prometheus/client_golang` and `otelgrpc` are declared in `go.mod` but no `/metrics` endpoint is registered and no OTel tracer is initialized in any service (review §14.1, R7). The declared observability intent is not wired. Additionally, the event-publish failure counter proposed in M5.D (#483) has no Prometheus backing until this EPIC.

**Definition of done:**
- `curl http://localhost:<port>/metrics` returns Prometheus text exposition on all services.
- A `zynax apply` produces a distributed trace with at least one span per service visible in a local Jaeger/Tempo instance.
- `zynax_grpc_requests_total` and `zynax_grpc_request_duration_seconds` counters populated after 10 requests.
- pprof accessible on engine-adapter admin port.

---

## E — Entities

- **`promhttp.Handler()`** — Prometheus HTTP exposition handler; registered at `/metrics` in each service's HTTP mux.
- **`zynax_grpc_requests_total{service,method,status}`** — Prometheus counter; incremented per incoming gRPC call.
- **`zynax_grpc_request_duration_seconds{service,method}`** — Prometheus histogram; measures gRPC handler latency.
- **`zynax_eventbus_publish_failed_total{event_type}`** — Prometheus counter; wired to the `slog.Warn` site from M5.D (#483).
- **OTel tracer** — initialized with OTLP exporter at `OTEL_EXPORTER_OTLP_ENDPOINT`; one root span per incoming gRPC request.
- **`otelgrpc.UnaryServerInterceptor()`** — gRPC server interceptor that creates and manages OTel spans.
- **`net/http/pprof`** — registered on a separate admin port in engine-adapter only.

---

## A — Approach

**What we WILL do:**
- Register `/metrics` handler in every service.
- Add a `grpc.UnaryServerInterceptor` that increments counters and records latency histograms.
- Initialize OTel tracer from `OTEL_EXPORTER_OTLP_ENDPOINT` env var; emit per-request spans.
- Propagate trace context through gRPC metadata (building on `request-id` propagation from M5.D #484).
- Wire the event-publish failure counter from M5.D.
- Register pprof on engine-adapter only (hot path; pprof is for performance investigation).

**What we WON'T do:**
- Add custom business-logic metrics beyond the standard gRPC counters and histograms (those are per-feature additions).
- Add alerting rules or dashboards (that is an operational concern outside the repo).

**ADR references:**
- ADR-016: Layered testing — verify metrics increment in unit tests.

---

## S — Structure

**Files touched (per service):**
- `services/*/cmd/*/main.go` — register `/metrics` endpoint; initialize OTel tracer; wire interceptors
- `services/*/internal/api/metrics.go` (new) — Prometheus counter and histogram definitions
- `services/engine-adapter/cmd/engine-adapter/main.go` — add pprof admin port registration
- `infra/docker-compose/` — optional: add local Prometheus + Jaeger for `make metrics-up`

---

## O — Operations

1. **[#491]** Register `/metrics` on all services; add gRPC interceptors for counters/histograms; initialize OTel tracer; add pprof to engine-adapter; wire event-publish counter from M5.D.

---

## N — Norms

- `feat:` PR type.
- Prometheus metric names follow the `zynax_` prefix convention.
- OTel exporter endpoint via env var — never hardcoded.
- pprof must be on a separate port from the production HTTP port (no accidental public exposure).

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed

### Feature Safeguards
- pprof endpoint must never be exposed on the same port as the production API.
- Metric labels must not include high-cardinality values (workflow IDs, request IDs) — only service/method/status.
