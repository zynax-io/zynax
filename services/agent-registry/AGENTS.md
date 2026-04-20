# services/agent-registry — AGENTS.md

> **Language: Go 1.22+**
> Inherits all rules from root `AGENTS.md` and `services/AGENTS.md`.

---

## Purpose

The Agent Registry is the **source of truth for agent identity** in the mesh.
It is the first service any agent calls when joining the platform.

**Responsibilities:**
- Register agents with their capabilities, endpoint, and metadata.
- Discover agents by capability and status.
- Track liveness via a client-streaming heartbeat RPC.
- Transition agents to `OFFLINE` after missed heartbeat windows.
- Publish `AgentRegistered`, `AgentDeregistered`, `AgentStatusChanged` events.

**Non-responsibilities:** Does not assign tasks. Does not store agent memory.
Does not authenticate external callers (that is `api-gateway`).

---

## Internal Layout

```
services/agent-registry/
├── cmd/agent-registry/main.go
├── internal/
│   ├── api/
│   │   ├── handler.go            ← RegisterAgent, GetAgent, ListByCapability, Heartbeat, WatchEvents
│   │   └── middleware.go         ← OTel, logging, recovery interceptors
│   ├── domain/
│   │   ├── model.go              ← AgentID, Capability, AgentSpec, Agent, AgentStatus
│   │   ├── service.go            ← AgentRegistrar, AgentDiscovery, HeartbeatMonitor
│   │   ├── repository.go         ← AgentRepository interface (port)
│   │   ├── events.go             ← AgentEvent, EventType — domain events
│   │   └── errors.go             ← ErrAgentNotFound, ErrAgentExists, ErrInvalidCapability
│   ├── infrastructure/
│   │   ├── postgres.go           ← PostgresAgentRepository
│   │   ├── redis_idempotency.go  ← idempotency key store (request_id dedup)
│   │   ├── nats_events.go        ← publishes domain events to event-bus
│   │   └── watchdog.go           ← background goroutine: marks stale agents OFFLINE
│   └── config/
│       └── config.go             ← envconfig, prefix: ZYNAX_REGISTRY_
├── tests/
│   ├── features/
│   │   └── agent_registry.feature
│   └── unit/
│       ├── registration_test.go  ← godog + table-driven
│       └── discovery_test.go
├── go.mod
└── Dockerfile
```

---

## Domain Model

```go
// internal/domain/model.go

// Newtypes prevent mixing IDs
type AgentID    string
type Capability string
type RequestID  string

// Constants — no magic numbers
const (
    MinCapabilities = 1
    MaxCapabilities = 50
)

// capabilityRegexp is compiled once and used in Capability.Validate()
var capabilityRegexp = regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)

func (c Capability) Validate() error {
    if !capabilityRegexp.MatchString(string(c)) {
        return fmt.Errorf("%w: %q must match ^[a-z][a-z0-9-]{1,63}$",
            ErrInvalidCapability, c)
    }
    return nil
}

type AgentStatus int
const (
    AgentStatusActive   AgentStatus = iota
    AgentStatusDraining
    AgentStatusOffline
)

// AgentSpec is an immutable value object. Created at registration. Never mutated.
type AgentSpec struct {
    ID           AgentID
    DisplayName  string
    Capabilities []Capability
    Endpoint     string
    Metadata     map[string]string
}

// Agent is a mutable entity with identity (AgentID).
type Agent struct {
    Spec          AgentSpec
    Status        AgentStatus
    RegisteredAt  time.Time
    LastHeartbeat *time.Time
}

func (a *Agent) IsDiscoverable() bool       { return a.Status == AgentStatusActive }
func (a *Agent) HasCapability(c Capability) bool {
    for _, cap := range a.Spec.Capabilities { if cap == c { return true } }
    return false
}
func (a *Agent) RecordHeartbeat(at time.Time) {
    a.LastHeartbeat = &at
    if a.Status == AgentStatusOffline { a.Status = AgentStatusActive }
}
```

---

## Domain Services

```go
// internal/domain/service.go

// AgentRepository — interface defined HERE in domain, implemented in infrastructure
type AgentRepository interface {
    Save(ctx context.Context, agent *Agent) error
    FindByID(ctx context.Context, id AgentID) (*Agent, error)
    FindByCapability(ctx context.Context, cap Capability, opts ListOptions) ([]*Agent, string, error)
    UpdateStatus(ctx context.Context, id AgentID, status AgentStatus) error
    UpdateHeartbeat(ctx context.Context, id AgentID, at time.Time) error
    FindStale(ctx context.Context, before time.Time) ([]*Agent, error)
    SoftDelete(ctx context.Context, id AgentID) error
}

// IdempotencyStore prevents duplicate registrations from repeated requests
type IdempotencyStore interface {
    Get(ctx context.Context, requestID RequestID) (AgentID, bool, error)
    Set(ctx context.Context, requestID RequestID, agentID AgentID, ttl time.Duration) error
}

// EventPublisher publishes domain events to the event-bus
type EventPublisher interface {
    Publish(ctx context.Context, event AgentEvent) error
}

type AgentRegistrar struct {
    repo       AgentRepository
    idempotency IdempotencyStore
    publisher  EventPublisher
}

func (r *AgentRegistrar) Register(
    ctx context.Context, reqID RequestID, spec AgentSpec,
) (AgentID, error) {
    // Idempotency: same request_id → same response
    if id, ok, err := r.idempotency.Get(ctx, reqID); err != nil {
        return "", fmt.Errorf("idempotency check: %w", err)
    } else if ok {
        return id, nil
    }
    if _, err := r.repo.FindByID(ctx, spec.ID); err == nil {
        return "", fmt.Errorf("%w: %s", ErrAgentExists, spec.ID)
    }
    agent := &Agent{
        Spec: spec, Status: AgentStatusActive,
        RegisteredAt: time.Now().UTC(),
    }
    if err := r.repo.Save(ctx, agent); err != nil {
        return "", fmt.Errorf("save agent: %w", err)
    }
    if err := r.idempotency.Set(ctx, reqID, spec.ID, 24*time.Hour); err != nil {
        // Non-fatal: worst case we allow a harmless duplicate. Log and continue.
        slog.WarnContext(ctx, "failed to set idempotency key", "err", err)
    }
    _ = r.publisher.Publish(ctx, AgentEvent{Type: EventTypeRegistered, Agent: agent})
    return spec.ID, nil
}
```

---

## Heartbeat (client-streaming gRPC)

```go
// internal/api/handler.go (Heartbeat RPC)
func (h *Handler) Heartbeat(stream pb.AgentRegistryService_HeartbeatServer) error {
    for {
        ping, err := stream.Recv()
        if errors.Is(err, io.EOF) {
            return stream.SendAndClose(&pb.HeartbeatAck{ServerTime: timestamppb.Now()})
        }
        if err != nil {
            return status.Errorf(codes.Internal, "recv: %v", err)
        }
        if err := h.monitor.RecordHeartbeat(stream.Context(), domain.AgentID(ping.AgentId)); err != nil {
            slog.WarnContext(stream.Context(), "heartbeat failed", "agent_id", ping.AgentId, "err", err)
        }
    }
}
```

---

## Configuration

```go
// internal/config/config.go — prefix: ZYNAX_REGISTRY_
type Config struct {
    GRPCPort               int           `envconfig:"GRPC_PORT"    default:"50051"`
    HealthPort             int           `envconfig:"HEALTH_PORT"  default:"8080"`
    MetricsPort            int           `envconfig:"METRICS_PORT" default:"9090"`
    DatabaseURL            string        `envconfig:"DATABASE_URL" required:"true"`
    RedisURL               string        `envconfig:"REDIS_URL"    required:"true"`
    NATSUrl                string        `envconfig:"NATS_URL"     required:"true"`
    HeartbeatIntervalSecs  int           `envconfig:"HEARTBEAT_INTERVAL_SECS"  default:"30"`
    HeartbeatMissThreshold int           `envconfig:"HEARTBEAT_MISS_THRESHOLD" default:"3"`
    SoftDeleteRetentionDays int          `envconfig:"SOFT_DELETE_RETENTION_DAYS" default:"30"`
    ShutdownGraceSecs      int           `envconfig:"SHUTDOWN_GRACE_SECS" default:"30"`
    LogLevel               string        `envconfig:"LOG_LEVEL"    default:"INFO"`
    OtelEndpoint           string        `envconfig:"OTEL_ENDPOINT" default:"http://otel-collector:4317"`
    ServiceName            string        `envconfig:"SERVICE_NAME"  default:"agent-registry"`
}
```

---

## BDD Scenarios (write `.feature` file FIRST)

```gherkin
# tests/features/agent_registry.feature

Feature: Agent Registration and Discovery

  Background:
    Given the agent registry is running

  Scenario: Successfully register a new agent
    Given an agent spec with id "analyst-01" and capabilities ["summarize","search"]
    When the agent is registered with request_id "req-001"
    Then the response agent_id is "analyst-01"
    And the agent is discoverable by capability "summarize"

  Scenario: Registration is idempotent for same request_id
    Given agent "analyst-02" was registered with request_id "req-002"
    When the same registration is attempted again with request_id "req-002"
    Then the response is identical to the first registration
    And no duplicate record exists

  Scenario: Reject duplicate agent_id with different request_id
    Given agent "duplicate-01" is already registered
    When a new registration for "duplicate-01" is attempted with a different request_id
    Then the gRPC status is ALREADY_EXISTS

  Scenario: Reject agent with invalid capability format
    Given an agent spec with capabilities ["InvalidCap","UPPER"]
    When the agent is registered
    Then the gRPC status is INVALID_ARGUMENT

  Scenario: Discovery excludes OFFLINE agents by default
    Given agents "a1" (ACTIVE) and "a2" (OFFLINE) both have capability "search"
    When agents are listed by capability "search"
    Then the response contains "a1" and not "a2"

  Scenario: Agent transitions to OFFLINE after missing heartbeats
    Given agent "heartbeat-agent" is ACTIVE
    When 3 consecutive heartbeat windows are missed
    Then the agent status becomes OFFLINE
```

---

## Database Schema

```sql
CREATE TABLE agents (
    id              TEXT PRIMARY KEY,
    display_name    TEXT NOT NULL,
    endpoint        TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'ACTIVE',
    metadata        JSONB NOT NULL DEFAULT '{}',
    registered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat  TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ
);
CREATE TABLE agent_capabilities (
    agent_id    TEXT REFERENCES agents(id) ON DELETE CASCADE,
    capability  TEXT NOT NULL,
    PRIMARY KEY (agent_id, capability)
);
CREATE INDEX idx_capabilities ON agent_capabilities(capability);
CREATE INDEX idx_agents_status ON agents(status) WHERE deleted_at IS NULL;
```
