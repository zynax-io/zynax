# services/event-bus — AGENTS.md

> **Language: Go 1.22+**
> Inherits all rules from root `AGENTS.md` and `services/AGENTS.md`.

---

## Purpose

The Event Bus provides **async, decoupled messaging** between platform
services and agents. It is the communication backbone for fire-and-forget
domain events where the sender must not block waiting for consumers.

**Backend:** NATS JetStream. Configured entirely via env vars — the service
is backend-agnostic at the domain level (swappable without changing consumers).

**Responsibilities:**
- Accept event publications from any service or agent via gRPC.
- Deliver events to durable consumer groups with at-least-once semantics.
- Route events by topic to correct subscriber groups.
- Manage a dead-letter queue (DLQ) for events that exhaust delivery retries.
- Validate event payloads against registered schemas (optional, config-driven).
- Expose a subscription stream via gRPC server-streaming.

**Non-responsibilities:** Does not replace synchronous gRPC calls.
Does not store business data — events are ephemeral (configurable retention).
Does not orchestrate workflows.

---

## Topic Naming Convention

```
zynax.v1.<service>.<entity>.<event_type>

Examples:
  zynax.v1.agent-registry.agent.registered
  zynax.v1.agent-registry.agent.status-changed
  zynax.v1.task-broker.task.assigned
  zynax.v1.task-broker.task.completed
  zynax.v1.task-broker.task.failed
  zynax.v1.memory-service.namespace.deleted
```

Rules:
- All lowercase, dot-separated.
- Always prefixed `zynax.v1.` — version baked into topic.
- Never use wildcards in topic names — be explicit.

---

## Internal Layout

```
services/event-bus/
├── cmd/event-bus/main.go
├── internal/
│   ├── api/
│   │   └── handler.go          ← Publish, Subscribe (server-streaming), Acknowledge
│   ├── domain/
│   │   ├── model.go            ← TopicID, ConsumerGroup, EventEnvelope, DeliveryAttempt
│   │   ├── service.go          ← EventRouter
│   │   ├── broker.go           ← EventBroker interface (port — implemented by NATS adapter)
│   │   └── errors.go           ← ErrTopicNotFound, ErrSchemaViolation, ErrDLQFull
│   ├── infrastructure/
│   │   ├── nats_broker.go      ← NATSBroker: implements EventBroker via JetStream
│   │   └── schema_registry.go  ← optional JSON Schema validation
│   └── config/
│       └── config.go           ← prefix: KEEL_EVENTS_
├── tests/
│   ├── features/event_bus.feature
│   └── unit/
├── go.mod
└── Dockerfile
```

---

## Domain Model

```go
// internal/domain/model.go

type TopicID        string  // e.g. "zynax.v1.task-broker.task.completed"
type ConsumerGroup  string  // e.g. "memory-service" or "my-agent-01"
type EventID        = uuid.UUID

type EventEnvelope struct {
    ID            EventID
    Topic         TopicID
    Payload       []byte          // Opaque bytes — schema validated if registry configured
    SchemaVersion string
    ProducedAt    time.Time
    CorrelationID string          // Trace ID for event correlation
    Metadata      map[string]string
}

type DeliveryAttempt struct {
    Envelope     EventEnvelope
    AttemptCount int
    LastError    string
}

const (
    MaxDeliveryAttempts = 5
    DLQTopicSuffix      = ".dlq"
)

func DLQTopic(original TopicID) TopicID {
    return TopicID(string(original) + DLQTopicSuffix)
}
```

---

## EventBroker Interface (port)

```go
// internal/domain/broker.go

// EventBroker is the port — implemented by NATSBroker in infrastructure.
// If NATS is swapped for Kafka, only NATSBroker changes.
type EventBroker interface {
    Publish(ctx context.Context, envelope EventEnvelope) error
    Subscribe(
        ctx context.Context,
        topic TopicID,
        group ConsumerGroup,
        handler func(EventEnvelope) error,
    ) (func(), error)  // returns unsubscribe function
    CreateStream(ctx context.Context, topic TopicID, retentionDays int) error
}
```

---

## NATS JetStream Adapter

```go
// internal/infrastructure/nats_broker.go

type NATSBroker struct {
    js nats.JetStreamContext
}

func (b *NATSBroker) Publish(ctx context.Context, env EventEnvelope) error {
    data, err := proto.Marshal(envelopeToProto(env))
    if err != nil { return fmt.Errorf("marshal event: %w", err) }
    _, err = b.js.PublishAsync(string(env.Topic), data,
        nats.MsgId(env.ID.String()), // JetStream dedup via MsgId
    )
    return err
}

func (b *NATSBroker) Subscribe(
    ctx context.Context, topic TopicID, group ConsumerGroup,
    handler func(EventEnvelope) error,
) (func(), error) {
    sub, err := b.js.QueueSubscribeSync(string(topic), string(group),
        nats.Durable(string(group)),
        nats.AckExplicit(),
        nats.MaxDeliver(MaxDeliveryAttempts),
        nats.DeliverAll(),
    )
    if err != nil { return nil, fmt.Errorf("subscribe: %w", err) }

    go func() {
        for {
            select {
            case <-ctx.Done(): return
            default:
                msg, err := sub.NextMsgWithContext(ctx)
                if err != nil { continue }
                var env EventEnvelope
                if parseErr := parseEnvelope(msg.Data, &env); parseErr != nil {
                    _ = msg.Nak() // send to DLQ after MaxDeliver
                    continue
                }
                if handlerErr := handler(env); handlerErr != nil {
                    _ = msg.NakWithDelay(backoff(msg.Metadata.NumDelivered))
                } else {
                    _ = msg.Ack()
                }
            }
        }
    }()
    return sub.Drain, nil
}

func backoff(attempt uint64) time.Duration {
    base := 500 * time.Millisecond
    return time.Duration(math.Pow(2, float64(attempt))) * base
}
```

---

## Configuration

```go
// prefix: KEEL_EVENTS_
type Config struct {
    GRPCPort              int    `envconfig:"GRPC_PORT"              default:"50054"`
    HealthPort            int    `envconfig:"HEALTH_PORT"            default:"8080"`
    MetricsPort           int    `envconfig:"METRICS_PORT"           default:"9090"`
    NATSUrl               string `envconfig:"NATS_URL"               required:"true"`
    DefaultRetentionDays  int    `envconfig:"DEFAULT_RETENTION_DAYS" default:"7"`
    MaxDeliveryAttempts   int    `envconfig:"MAX_DELIVERY_ATTEMPTS"  default:"5"`
    SchemaValidation      bool   `envconfig:"SCHEMA_VALIDATION"      default:"false"`
    ShutdownGraceSecs     int    `envconfig:"SHUTDOWN_GRACE_SECS"    default:"30"`
    LogLevel              string `envconfig:"LOG_LEVEL"              default:"INFO"`
    OtelEndpoint          string `envconfig:"OTEL_ENDPOINT"          default:"http://otel-collector:4317"`
    ServiceName           string `envconfig:"SERVICE_NAME"           default:"event-bus"`
}
```

---

## BDD Scenarios

```gherkin
Feature: Event Bus

  Scenario: Published event is received by all subscribers
    Given consumer "a" and consumer "b" subscribe to "zynax.v1.task-broker.task.completed"
    When an event is published to that topic
    Then both "a" and "b" receive the event

  Scenario: Event is NOT received by subscriber on a different topic
    Given consumer "c" subscribes to "zynax.v1.task-broker.task.assigned"
    When an event is published to "zynax.v1.task-broker.task.completed"
    Then consumer "c" does NOT receive the event

  Scenario: Delivery is retried on handler failure
    Given a subscriber that fails on the first delivery
    When an event is published
    Then the event is redelivered at least once

  Scenario: Event is sent to DLQ after exhausting retries
    Given a subscriber that always fails
    When an event is published
    And 5 delivery attempts are exhausted
    Then the event appears on the DLQ topic

  Scenario: Durable consumer receives events published while offline
    Given consumer "d" was offline when an event was published
    When consumer "d" reconnects
    Then consumer "d" receives the missed event

  Scenario: Multiple consumer groups receive same event independently
    Given groups "indexer" and "notifier" both subscribe to the same topic
    When one event is published
    Then both "indexer" and "notifier" receive their own copy
```
