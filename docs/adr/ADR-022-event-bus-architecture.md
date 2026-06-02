<!-- SPDX-License-Identifier: Apache-2.0 -->

# ADR-022 — EventBusService gRPC wrapper over NATS JetStream

| Field | Value |
|-------|-------|
| **Status** | Accepted |
| **Date** | 2026-06-02 |
| **Deciders** | Oscar Gómez Manresa |
| **Governs** | `services/event-bus/`, `protos/zynax/v1/event_bus.proto`, all producer/consumer wiring |

---

## Context

The repo needed a formal decision on whether to implement the event bus as:

- **Option 1** — a full gRPC `EventBusService` wrapping NATS JetStream (stateless Go service).
- **Option 2** — a shared CloudEvents library + direct NATS access (no service).
- **Option 3** — a hybrid thin control-plane service + direct NATS for internal paths.

This is a one-way door: the proto contract (`event_bus.proto`) and BDD feature file were committed before this ADR (ADR-016: contracts before code). Reversing to Option 2 or 3 would require reverting accepted contracts.

### Pre-commitment evidence at decision time

| Artifact | Status |
|----------|--------|
| `protos/zynax/v1/event_bus.proto` — `EventBusService` with `Publish`, `Subscribe`, `Unsubscribe` | Committed |
| `services/event-bus/tests/features/event_bus.feature` — 6 BDD scenarios | Committed |
| `services/event-bus/AGENTS.md` — describes gRPC service with `NATSEventBus` adapter | Committed |
| ADR-001 — all inter-service calls are gRPC; no direct cross-service imports | Accepted |
| ADR-013 — Python adapters use gRPC stubs only; no NATS client library in agents | Accepted |

---

## Decision

**Option 1 — Full gRPC EventBusService wrapping NATS JetStream.**

The `event-bus` Go service is a **stateless Deployment** that wraps JetStream. All durability (stream retention, consumer groups, DLQ) lives entirely in JetStream — the Go service itself holds no persistent state.

All producers (engine-adapter, task-broker, agent-registry) call `EventBusService.Publish` via gRPC.  
All consumers (future observability, memory-service event triggers) call `EventBusService.Subscribe` via gRPC server-streaming.  
The `PublishLifecycleEventActivity` stub in `services/engine-adapter/internal/infrastructure/activities.go` will be wired to the `EventBusService` gRPC stub when this service ships (EPIC I).

---

## Rationale

1. **ADR-001 compliance** — All inter-service communication must be gRPC. Directing services or Python adapters to a NATS client library would violate ADR-001 and ADR-013.
2. **Proto/BDD irreversibility** — Reverting the committed proto + BDD contracts to adopt a library approach would violate the ADR-016 "contracts before code" invariant in reverse (removing accepted contracts is a one-way door in the other direction).
3. **Natural policy chokepoint** — The gRPC service boundary is the right place to enforce topic authorization, namespace isolation, and rate limits (planned for M7). A shared library has no natural enforcement point.
4. **Stateless service** — The extra gRPC hop (~0.1–1 ms per publish on local network) is acceptable for async event delivery. The service itself is stateless; scaling is a Deployment replica-count change.
5. **CNCF alignment** — A clean versioned gRPC contract (`zynax.v1.EventBusService`) is more portable than a shared library dependency.

---

## Consequences

**Positive**
- ADR-001 and ADR-013 compliance: zero exceptions needed.
- Policy, quota, and multi-namespace isolation fit naturally as gRPC middleware (M7).
- Horizontal scaling is trivial (stateless Deployment).
- BDD contract tests (`event_bus.feature`) can be run without NATS in test doubles.

**Negative**
- One extra network hop per `Publish` call vs direct JetStream.
- One additional service to deploy, monitor, and maintain.

**Neutral**
- NATS JetStream is a cluster-level StatefulSet. The event-bus Go service is a stateless Deployment.
- The Helm chart for event-bus (EPIC A, step A.6) is gated on this ADR (now unblocked).

---

## Implementation scope (EPIC I — #772)

Story decomposition (O-steps, to be created via `/spdd-story 772`):

| Step | Story | Notes |
|------|-------|-------|
| I.1 | Service scaffold — `go.mod`, `cmd/`, domain types, NATS JetStream client bootstrap | No external deps beyond NATS Go SDK |
| I.2 | Publish path — JetStream stream create + event publish | Domain: `EventBus.Publish` |
| I.3 | Subscribe path — durable consumer group + gRPC server-streaming | Domain: `EventBus.Subscribe` |
| I.4 | Unsubscribe + DLQ + retry-backoff wiring | Completes the EventBus interface |
| I.5 | Wire engine-adapter `PublishLifecycleEventActivity` to EventBusService gRPC | Removes the log-only stub |
| I.6 | BDD step implementations for `event_bus.feature` (6 scenarios) | `test:` type — SPDD-exempt |

Canvas must be created via `/spdd-reasons-canvas 772` before any implementation PR is opened.

---

## References

- ADR-001: gRPC as the inter-service protocol
- ADR-008: No shared databases — event durability lives in JetStream, not a service DB
- ADR-013: Adapter-first — Python adapters use gRPC stubs only
- ADR-014: Event-driven state machine workflow model
- ADR-016: Layered testing — BDD contracts before implementation
- Decision issue: [#764](https://github.com/zynax-io/zynax/issues/764)
- EPIC I: [#772](https://github.com/zynax-io/zynax/issues/772)
