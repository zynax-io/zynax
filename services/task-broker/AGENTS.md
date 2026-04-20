# services/task-broker — AGENTS.md

> **Language: Go 1.22+**
> Inherits all rules from root `AGENTS.md` and `services/AGENTS.md`.

---

## Purpose

The Task Broker is the **work scheduler** of the mesh.

**Why Go:** Sub-millisecond assignment latency. Thousands of concurrent
`WatchTask` server-streaming goroutines per instance. Priority queue
operations in hot path. Go's concurrency model is ideal here.

**Responsibilities:**
- Accept task submissions (async — returns `task_id` immediately).
- Discover eligible agents from `agent-registry` via gRPC.
- Assign tasks using a pluggable `AssignmentStrategy`.
- Manage state machine: `PENDING → ASSIGNED → RUNNING → SUCCEEDED | FAILED | TIMED_OUT | CANCELLED`.
- Retry failed tasks with exponential backoff + jitter (up to `MaxRetries`).
- Enforce per-task timeouts via a background watchdog goroutine.
- Fan-out real-time `WatchTask` state updates to concurrent server-streaming subscribers.
- Publish task lifecycle events to `event-bus`.

**Non-responsibilities:** Does not execute tasks. Does not store results.
Does not authenticate callers.

---

## Internal Layout

```
services/task-broker/
├── cmd/task-broker/main.go
├── internal/
│   ├── api/
│   │   ├── handler.go           ← SubmitTask, GetTask, CancelTask, WatchTask, ListTasks
│   │   └── middleware.go
│   ├── domain/
│   │   ├── model.go             ← TaskID, Task, TaskSpec, TaskState, TaskPriority
│   │   ├── service.go           ← TaskScheduler (Submit, Assign, Transition)
│   │   ├── repository.go        ← TaskRepository interface
│   │   ├── strategy.go          ← AssignmentStrategy interface + RoundRobin, LeastLoaded
│   │   ├── watcher.go           ← WatcherRegistry: fan-out state changes to subscribers
│   │   └── errors.go            ← ErrTaskNotFound, ErrTaskNotPending, ErrNoEligibleAgent
│   ├── infrastructure/
│   │   ├── postgres.go          ← PostgresTaskRepository
│   │   ├── redis_lock.go        ← distributed lock (prevent double-assignment in multi-replica)
│   │   ├── registry_client.go   ← gRPC client for agent-registry (discover agents)
│   │   ├── nats_events.go       ← publish task lifecycle events to event-bus
│   │   └── watchdog.go          ← background goroutine: enforce timeouts
│   └── config/
│       └── config.go            ← prefix: ZYNAX_BROKER_
├── tests/
│   ├── features/task_broker.feature
│   └── unit/
├── go.mod
└── Dockerfile
```

---

## Domain Model

```go
// internal/domain/model.go

type TaskID   = uuid.UUID
type AgentID  string
type RequestID string

type TaskState int
const (
    TaskStatePending   TaskState = iota
    TaskStateAssigned
    TaskStateRunning
    TaskStateSucceeded
    TaskStateFailed
    TaskStateTimedOut
    TaskStateCancelled
)

type TaskPriority int
const (
    PriorityLow      TaskPriority = 1
    PriorityNormal   TaskPriority = 5
    PriorityHigh     TaskPriority = 9
    PriorityCritical TaskPriority = 10
)

const DefaultMaxRetries = 3

type TaskSpec struct {
    RequiredCapability string
    Payload            map[string]any
    Priority           TaskPriority
    MaxRetries         int
    TimeoutSeconds     int
    Metadata           map[string]string
}

type Task struct {
    ID            TaskID
    Spec          TaskSpec
    State         TaskState
    AssignedAgent AgentID
    AttemptCount  int
    FailureReason string
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

func (t *Task) IsTerminal() bool {
    switch t.State {
    case TaskStateSucceeded, TaskStateFailed, TaskStateTimedOut, TaskStateCancelled:
        return true
    }
    return false
}

func (t *Task) CanBeAssigned() bool  { return t.State == TaskStatePending }
func (t *Task) ShouldRetry() bool    { return t.State == TaskStateFailed && t.AttemptCount < t.Spec.MaxRetries }
func (t *Task) IsTimedOut(now time.Time) bool {
    if t.Spec.TimeoutSeconds == 0 || t.State != TaskStateRunning { return false }
    deadline := t.UpdatedAt.Add(time.Duration(t.Spec.TimeoutSeconds) * time.Second)
    return now.After(deadline)
}
```

---

## Domain Service

```go
// internal/domain/service.go

type TaskRepository interface {
    Save(ctx context.Context, task *Task) error
    FindByID(ctx context.Context, id TaskID) (*Task, error)
    Transition(ctx context.Context, id TaskID, to TaskState, opts ...TransitionOption) error
    FindPending(ctx context.Context, capability string, limit int) ([]*Task, error)
    FindTimedOut(ctx context.Context, now time.Time) ([]*Task, error)
    List(ctx context.Context, filter ListFilter, page Page) ([]*Task, string, error)
}

// AssignmentStrategy is pluggable — configured via ZYNAX_BROKER_ASSIGNMENT_STRATEGY
type AssignmentStrategy interface {
    Assign(ctx context.Context, task *Task, eligible []AgentID) (AgentID, error)
}

type TaskScheduler struct {
    repo     TaskRepository
    strategy AssignmentStrategy
    watchers *WatcherRegistry
}

func (s *TaskScheduler) Submit(ctx context.Context, reqID RequestID, spec TaskSpec) (TaskID, error) {
    task := &Task{ID: uuid.New(), Spec: spec, State: TaskStatePending, CreatedAt: time.Now().UTC()}
    if err := s.repo.Save(ctx, task); err != nil {
        return uuid.Nil, fmt.Errorf("save task: %w", err)
    }
    return task.ID, nil
}

func (s *TaskScheduler) AssignPending(ctx context.Context, cap string, eligible []AgentID) (*Task, error) {
    pending, err := s.repo.FindPending(ctx, cap, 1)
    if err != nil { return nil, fmt.Errorf("find pending: %w", err) }
    if len(pending) == 0 { return nil, nil } // no work — not an error

    task := pending[0]
    if !task.CanBeAssigned() { return nil, ErrTaskNotPending }

    agentID, err := s.strategy.Assign(ctx, task, eligible)
    if err != nil { return nil, fmt.Errorf("assign: %w", err) }

    if err := s.repo.Transition(ctx, task.ID, TaskStateAssigned,
        WithAgent(agentID), WithIncrementAttempt()); err != nil {
        return nil, fmt.Errorf("transition: %w", err)
    }
    task.State = TaskStateAssigned
    task.AssignedAgent = agentID
    s.watchers.Notify(task.ID, TaskStateAssigned)
    return task, nil
}
```

---

## WatchTask Fan-out

```go
// internal/domain/watcher.go

// WatcherRegistry fans out task state changes to all active WatchTask subscribers.
// Each subscriber gets its own buffered channel.
// Uses sync.RWMutex — high read (many watchers), low write (state transitions).
type WatcherRegistry struct {
    mu       sync.RWMutex
    watchers map[TaskID][]chan TaskState
}

func (r *WatcherRegistry) Subscribe(id TaskID) (<-chan TaskState, func()) {
    ch := make(chan TaskState, 16) // buffered — never block the publisher
    r.mu.Lock()
    r.watchers[id] = append(r.watchers[id], ch)
    r.mu.Unlock()
    unsubscribe := func() {
        r.mu.Lock()
        defer r.mu.Unlock()
        subs := r.watchers[id]
        for i, s := range subs {
            if s == ch { r.watchers[id] = append(subs[:i], subs[i+1:]...); break }
        }
        close(ch)
    }
    return ch, unsubscribe
}

func (r *WatcherRegistry) Notify(id TaskID, state TaskState) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    for _, ch := range r.watchers[id] {
        select {
        case ch <- state:
        default: // subscriber is slow — drop rather than block
            slog.Warn("watcher channel full, dropping event", "task_id", id)
        }
    }
}
```

---

## Distributed Lock (multi-replica safety)

```go
// internal/infrastructure/redis_lock.go
// Prevent two broker replicas from assigning the same task simultaneously.

func (s *TaskScheduler) AssignWithLock(ctx context.Context, cap string, eligible []AgentID) (*Task, error) {
    lockKey := fmt.Sprintf("task-broker:assign:%s", cap)
    lock, err := s.locker.Obtain(ctx, lockKey, 5*time.Second, nil)
    if err != nil { return nil, nil } // another replica holds the lock — skip
    defer lock.Release(ctx)
    return s.AssignPending(ctx, cap, eligible)
}
```

---

## Configuration

```go
// prefix: ZYNAX_BROKER_
type Config struct {
    GRPCPort             int    `envconfig:"GRPC_PORT"              default:"50052"`
    HealthPort           int    `envconfig:"HEALTH_PORT"            default:"8080"`
    MetricsPort          int    `envconfig:"METRICS_PORT"           default:"9090"`
    DatabaseURL          string `envconfig:"DATABASE_URL"           required:"true"`
    RedisURL             string `envconfig:"REDIS_URL"              required:"true"`
    NATSUrl              string `envconfig:"NATS_URL"               required:"true"`
    AgentRegistryURL     string `envconfig:"AGENT_REGISTRY_URL"     required:"true"`
    AssignmentStrategy   string `envconfig:"ASSIGNMENT_STRATEGY"    default:"round_robin"`
    WatchdogIntervalSecs int    `envconfig:"WATCHDOG_INTERVAL_SECS" default:"15"`
    ShutdownGraceSecs    int    `envconfig:"SHUTDOWN_GRACE_SECS"    default:"30"`
    LogLevel             string `envconfig:"LOG_LEVEL"              default:"INFO"`
    OtelEndpoint         string `envconfig:"OTEL_ENDPOINT"          default:"http://otel-collector:4317"`
    ServiceName          string `envconfig:"SERVICE_NAME"           default:"task-broker"`
}
```

---

## BDD Scenarios

```gherkin
Feature: Task Scheduling

  Scenario: Submit task returns task_id immediately
    When a task with capability "summarize" and priority NORMAL is submitted
    Then the response contains a valid task_id
    And the task state is PENDING

  Scenario: Task assigned to eligible agent
    Given agent "a1" is ACTIVE with capability "summarize"
    And a PENDING task with capability "summarize" exists
    When the broker runs an assignment cycle
    Then the task state becomes ASSIGNED
    And the assigned_agent_id is "a1"

  Scenario: Task stays PENDING when no eligible agent exists
    Given no agents with capability "rare-skill" are ACTIVE
    When a task with capability "rare-skill" is submitted
    Then the task state remains PENDING after the assignment cycle
    And no error is raised

  Scenario: Failed task is retried with backoff
    Given a task with max_retries=3 is in state FAILED with attempt_count=1
    When the retry cycle runs
    Then the task state becomes PENDING again
    And attempt_count is still 1 (incremented on next assignment)

  Scenario: Task becomes FAILED after exhausting retries
    Given a task with max_retries=3 and attempt_count=3 fails
    When the task is marked FAILED
    Then the task state is permanently FAILED
    And no further retry is attempted

  Scenario: WatchTask receives state transitions in order
    Given a client is watching task "task-123"
    When the task transitions PENDING → ASSIGNED → RUNNING → SUCCEEDED
    Then the WatchTask stream delivers those 4 events in order

  Scenario: High priority task is assigned before low priority
    Given a LOW priority task and a HIGH priority task both PENDING with same capability
    When one assignment cycle runs
    Then the HIGH priority task is assigned first
```
