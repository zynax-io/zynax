# services/event-bus — AGENTS.md

> Go 1.22+. Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M3+ (not yet implemented).** BDD contract tests exist in `protos/tests/`.

---

## Purpose

The Event Bus provides **async, decoupled messaging** between platform services
and agents. Backend: NATS JetStream.

- Accepts event publications from any service or agent via gRPC.
- Delivers events to durable consumer groups (at-least-once semantics).
- Routes events by topic to subscriber groups.
- Manages a dead-letter queue (DLQ) for events that exhaust delivery retries.
- Exposes a subscription stream via gRPC server-streaming.

Does NOT: replace synchronous gRPC calls · store business data · orchestrate workflows.

**Topic naming:** `zynax.v1.<service>.<entity>.<event_type>`
Example: `zynax.v1.agent-registry.agent.registered`

---

## Internal Layout

```
services/event-bus/
├── cmd/event-bus/main.go
├── internal/
│   ├── api/
│   │   └── handler.go          ← Publish, Subscribe, Unsubscribe
│   ├── domain/
│   │   ├── event.go            ← CloudEvent, Topic, ConsumerGroup
│   │   ├── bus.go              ← EventBus interface
│   │   └── errors.go           ← ErrTopicNotFound, ErrDeadLetter
│   └── infrastructure/
│       └── nats.go             ← NATSEventBus (JetStream)
├── go.mod
└── Dockerfile
```

Config env prefix: `ZYNAX_EVENTBUS_` · gRPC port: 50054 · NATS URL: env var

---

## Running Tests

```bash
cd services/event-bus
GOWORK=off go test ./... -race -timeout 60s

# BDD contract tests
cd protos/tests
GOWORK=off go test ./event_bus_service/... -v -timeout 60s
```
