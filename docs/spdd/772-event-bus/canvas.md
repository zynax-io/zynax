# REASONS Canvas — Event Bus NATS JetStream Implementation

> **All content in this Canvas is Tier 1 (public-safe).**
> Tier 2 content (internal hostnames, IPs, credentials, deployment specifics) belongs in
> `canvas.private.md` (gitignored). Run `/spdd-security-review <path>` before committing.

**Issue:** #772
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-08
**Status:** Aligned

---

## R — Requirements

- **Missing service implementation.** `services/event-bus/` has only a stub AGENTS.md and a committed BDD feature file; the Go service itself does not exist. `PublishLifecycleEventActivity` in engine-adapter is log-only and produces no observable events.
- **Workflow state transitions are invisible.** Without a working event bus, there is no way for external consumers or future services (memory-service, observability) to observe workflow lifecycle events (submitted → running → completed/failed).
- **ADR-022 unblocked, delivery pending.** The architecture decision (gRPC `EventBusService` wrapping NATS JetStream) was accepted in ADR-022. The proto contract and 6 BDD scenarios are committed. The service just needs to be built.
- **DevAuto Wave 4 gated.** EPIC M6.DevAuto #881 is blocked on M6.I (#772) being complete.

**Definition of done:**
- `services/event-bus/` compiles, all unit tests pass (`GOWORK=off go test ./... -race`), `internal/domain/` coverage ≥ 90%.
- BDD contract tests in `protos/tests/event_bus_service/` pass green against a real NATS JetStream (testcontainers-go).
- `engine-adapter`'s `PublishLifecycleEventActivity` publishes real gRPC events to `EventBusService`; log-only stub removed.
- Helm chart placeholder for event-bus (already merged, #785) updated to point at the real image tag.
- The 6 BDD scenarios in `event_bus.feature` are all green on CI.

---

## E — Entities

```
CloudEvent (zynax.v1.cloudevents.proto)
  id          string   — service-assigned, unique per event
  source      string   — originating service (e.g. "zynax.engine-adapter")
  type        string   — topic name (e.g. "zynax.v1.workflow.completed")
  data        bytes    — opaque payload

EventBus (domain port — interface)
  Publish(ctx, CloudEvent) → (event_id, error)
  Subscribe(ctx, SubscribeRequest) → <chan CloudEvent>
  Unsubscribe(ctx, subscriber_id) → error

NATSEventBus (infrastructure adapter — implements EventBus)
  wraps nats.JetStreamContext
  creates/reuses JetStream streams per topic prefix
  manages durable consumer groups (at-least-once delivery)
  routes failed deliveries to DLQ subject

EventBusHandler (api layer)
  implements zynax.v1.EventBusService gRPC server
  translates between proto messages and domain types
  validates required fields (id, source, type non-empty)

Subscriber (domain value object)
  subscriber_id  string
  type_pattern   string   — glob syntax (* = single segment, ** = zero or more)
  workflow_id    string   — optional scope filter; empty = all workflows
```

Relationships:
```
engine-adapter ──gRPC Publish──► EventBusHandler ──► NATSEventBus ──► JetStream
                                                                          │
future consumers ◄──gRPC Subscribe stream──── EventBusHandler ◄──────────┘
                                                    │
                                               DLQ subject (exhausted retries)
```

Topic naming convention: `zynax.v1.<service>.<entity>.<event_type>`
Example: `zynax.v1.engine-adapter.workflow.completed`

---

## A — Approach

**We will:**
- Implement `services/event-bus/` as a stateless Go service (ADR-022): `cmd/event-bus/main.go`, `internal/domain/`, `internal/api/`, `internal/infrastructure/nats.go`.
- Use NATS JetStream as the sole durability layer; the Go service itself holds zero state.
- Implement `EventBus` as a domain interface with `NATSEventBus` as the only production adapter; in-memory fake for unit tests.
- Wire `PublishLifecycleEventActivity` in engine-adapter to call `EventBusService.Publish` via the generated gRPC stub, replacing the log-only stub.
- Implement BDD step functions for all 6 scenarios in `event_bus.feature` using testcontainers-go (real NATS JetStream in CI).
- Deliver in 6 sequential O-steps (one PR each), matching the existing story issues #823–#828.

**We will NOT:**
- Store any event state inside the Go service process (ADR-022: all durability in JetStream).
- Implement topic authorization, namespace isolation, or quota enforcement — those belong to M7 (natural gRPC middleware hooks are established here, enforcement deferred).
- Add direct NATS client calls in engine-adapter, task-broker, or agents — all producers call `EventBusService.Publish` via gRPC (ADR-001, ADR-013).
- Replace or modify the `event_bus.proto` contract — it is committed and stable (ADR-016).
- Add a CloudEvents HTTP sink or HTTP endpoint — the service speaks gRPC only (ADR-001).

**Governing ADRs:** ADR-001 (gRPC inter-service), ADR-013 (Python adapters gRPC-only), ADR-014 (event-driven workflow execution), ADR-016 (contracts before code), ADR-022 (EventBusService architecture decision).

---

## S — Structure

**New files (services/event-bus/):**
```
services/event-bus/
├── cmd/event-bus/main.go            ← gRPC server bootstrap, NATS connect, graceful shutdown
├── internal/
│   ├── domain/
│   │   ├── event.go                 ← CloudEvent domain type, Topic, ConsumerGroup value objects
│   │   ├── bus.go                   ← EventBus interface (Publish/Subscribe/Unsubscribe)
│   │   └── errors.go                ← ErrTopicNotFound, ErrDeadLetter, ErrSubscriberNotFound
│   ├── api/
│   │   └── handler.go               ← EventBusService gRPC handler; proto ↔ domain translation
│   └── infrastructure/
│       └── nats.go                  ← NATSEventBus: JetStream stream mgmt, consumer groups, DLQ
├── go.mod                           ← module github.com/zynax-io/zynax/services/event-bus
└── Dockerfile                       ← already present via Helm placeholder; wire real binary
```

**Modified files:**
- `services/engine-adapter/internal/infrastructure/activities.go` — replace `PublishLifecycleEventActivity` log stub with real `EventBusService` gRPC call (O5 / #827).
- `protos/tests/event_bus_service/` — add BDD step implementations for 6 scenarios (O6 / #828).
- `infra/charts/event-bus/` — Helm placeholder already merged (#785); values.yaml image tag update post-O1.

**gRPC contracts used (no modifications):**
- `protos/zynax/v1/event_bus.proto` — `EventBusService` (Publish, Subscribe, Unsubscribe).
- `protos/zynax/v1/cloudevents.proto` — `CloudEvent` message.

**Config env vars (prefix `ZYNAX_EVENTBUS_`):**
- `ZYNAX_EVENTBUS_GRPC_PORT` (default `50054`)
- `ZYNAX_EVENTBUS_NATS_URL` — JetStream endpoint
- `ZYNAX_EVENTBUS_STREAM_RETENTION_HOURS` — stream retention window
- `ZYNAX_EVENTBUS_DLQ_MAX_RETRIES` — retry ceiling before DLQ

---

## O — Operations

Each step is one PR. Canvas must be `Status: Aligned` before O1 starts.

1. ✅ **O1 — Service scaffold** (#823)
   `feat(event-bus): service scaffold — go.mod, domain interfaces, NATS client bootstrap`
   - `go.mod` for `github.com/zynax-io/zynax/services/event-bus`
   - Domain types: `event.go` (CloudEvent, Topic, ConsumerGroup), `bus.go` (EventBus interface), `errors.go`
   - `NATSEventBus` stub (connect + ping only, methods return `errors.New("not implemented")`)
   - `cmd/event-bus/main.go`: NATS connect, gRPC listen on `50054`, graceful shutdown
   - Domain unit tests ≥ 90% on `internal/domain/`

2. **O2 — Publish path** (#824)
   `feat(event-bus): Publish path — JetStream stream create + event publish`
   - `NATSEventBus.Publish`: create/reuse JetStream stream, publish CloudEvent as JSON
   - `EventBusHandler.Publish`: validate id/source/type non-empty → INVALID_ARGUMENT; assign `event_id`; populate `accepted_at`
   - Unit tests for validation cases and happy path

3. **O3 — Subscribe path** (#825)
   `feat(event-bus): Subscribe path — durable consumer group + gRPC server-streaming`
   - `NATSEventBus.Subscribe`: create durable consumer group, glob-pattern matching on event type
   - `EventBusHandler.Subscribe`: open gRPC server-streaming, forward CloudEvents from consumer
   - Optional `workflow_id` scope filter applied at the Go layer
   - Unit tests for pattern matching logic

4. **O4 — Unsubscribe + DLQ + retry-backoff** (#826)
   `feat(event-bus): Unsubscribe + DLQ + retry-backoff wiring`
   - `NATSEventBus.Unsubscribe`: delete durable consumer, close subscription channel
   - `EventBusHandler.Unsubscribe`: return NOT_FOUND for unknown subscriber_id
   - JetStream retry-backoff configuration; after `DLQ_MAX_RETRIES` exhausted → nack to DLQ subject
   - Unit tests for retry exhaustion and NOT_FOUND path

5. **O5 — Engine-adapter wiring** (#827)
   `feat(engine-adapter): wire PublishLifecycleEventActivity to EventBusService gRPC`
   - Replace log-only stub in `services/engine-adapter/internal/infrastructure/activities.go`
   - Add `EventBusService` gRPC client to engine-adapter's dependency graph
   - Publish workflow lifecycle events: `submitted`, `running`, `completed`, `failed`
   - Unit test: mock gRPC client, assert correct event type + workflow_id propagation

6. **O6 — BDD step implementations** (#828) *(SPDD-exempt: `test:` type)*
   `test: BDD step implementations for event_bus.feature — 6 scenarios`
   - Implement step functions in `protos/tests/event_bus_service/` using testcontainers-go
   - Cover all 6 scenarios: fan-out, topic isolation, retry, DLQ, catch-up, consumer groups
   - Integration build tag (`//go:build integration`) — runs in `test-integration` CI job only

---

## N — Norms

- **Commit hygiene:** every commit carries `Signed-off-by` + `Assisted-by: Claude/<model>` trailers per `AGENTS.md §Hard Constraints`. Never `Co-Authored-By:` for AI.
- **GOWORK=off:** mandatory for all `go` commands inside `services/event-bus/` and `services/engine-adapter/` (ADR-017).
- **ctx-first:** every domain and infrastructure function that performs I/O accepts `ctx context.Context` as first parameter; handlers check `ctx.Err()` at entry (services/AGENTS.md).
- **Layer rule:** `domain` has zero imports from `api` or `infrastructure`. `api` imports `domain` only. `infrastructure` imports `domain` only (CI import-analysis gate).
- **In-memory test double:** `NATSEventBus` has a corresponding in-memory fake used in unit tests. Never connect to a real NATS instance from unit tests (services/AGENTS.md §Integration test convention).
- **Integration tests tagged:** files connecting to real NATS carry `//go:build integration` on line 1 (services/AGENTS.md).
- **Domain coverage ≥ 90%:** `internal/domain/` must reach ≥ 90% coverage (pure logic, no I/O to mock).
- **Proto immutability:** `event_bus.proto` field numbers are permanent (ADR-001 §backward-compat). Do not modify field numbers or remove fields.
- **No direct NATS in other services:** engine-adapter, task-broker, and agents must call `EventBusService.Publish` via gRPC stub; never import `nats.go` library outside `services/event-bus/` (ADR-001, ADR-013).
- **Stateless service:** the Go process holds zero persistent state. No in-memory subscriber map that survives pod restart — JetStream durable consumers survive pod restarts natively.
- **PR size:** ≤ 200 lines ideal; > 900 blocked. Generated stubs excluded (CLAUDE.md).

---

## S — Safeguards

### Context Security (mandatory before committing this Canvas)
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII: no personal names in sensitive context, no email addresses
- [x] No prompt injection: no instruction-like phrasing that would override AGENTS.md rules
- [x] All entities in E are public-safe abstractions
- [x] `/spdd-security-review` passed on this file

### Feature Safeguards
- **Never** let the Go service process hold subscriber state that is not backed by JetStream durable consumers — if the pod restarts, all active subscriptions must be recoverable from JetStream alone (ADR-022).
- **Never** add a direct NATS client to `services/engine-adapter/`, `services/task-broker/`, `services/agent-registry/`, or any Python adapter — all producers use `EventBusService` gRPC (ADR-001, ADR-013).
- **Never** modify `event_bus.proto` field numbers or remove existing fields — the contract is committed and binary-compatible changes only (ADR-016).
- **Never** add quota, topic-authorization, or namespace-isolation logic in this EPIC — those are M7 gRPC middleware concerns. The bus is open to all callers within the cluster for M6.
- **Never** use `context.Background()` or `context.TODO()` in production domain/infrastructure code outside of Temporal workflow functions (services/AGENTS.md §Context propagation).
- **Never** skip the `//go:build integration` tag on any test that connects to a real NATS instance (services/AGENTS.md §Integration test convention).
