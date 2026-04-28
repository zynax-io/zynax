# services/task-broker — AGENTS.md

> Go 1.22+. Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M3+ (not yet implemented).** BDD contract tests exist in `protos/tests/`.

---

## Purpose

The Task Broker is the **work scheduler** of the mesh.

- Accepts task submissions (async — returns `task_id` immediately).
- Discovers eligible agents from `agent-registry` via gRPC.
- Assigns tasks using a pluggable `AssignmentStrategy`.
- Manages state machine: `PENDING → ASSIGNED → RUNNING → SUCCEEDED | FAILED | TIMED_OUT | CANCELLED`.
- Retries failed tasks with exponential backoff + jitter.
- Enforces per-task timeouts via a background watchdog goroutine.
- Fan-out real-time `WatchTask` state updates to concurrent server-streaming subscribers.
- Publishes task lifecycle events to `event-bus`.

Does NOT: execute tasks · store results · authenticate callers.

---

## Internal Layout

```
services/task-broker/
├── cmd/task-broker/main.go
├── internal/
│   ├── api/
│   │   └── handler.go          ← SubmitTask, GetTask, CancelTask, WatchTask, ListTasks
│   ├── domain/
│   │   ├── model.go            ← TaskID, Task, TaskState, TaskPriority
│   │   ├── service.go          ← TaskScheduler (Submit, Assign, Transition)
│   │   ├── repository.go       ← TaskRepository interface
│   │   ├── strategy.go         ← AssignmentStrategy: RoundRobin, LeastLoaded
│   │   ├── watcher.go          ← WatcherRegistry: fan-out state changes
│   │   └── errors.go           ← ErrTaskNotFound, ErrNoEligibleAgent
│   └── infrastructure/
│       ├── postgres.go         ← PostgresTaskRepository
│       ├── redis_lock.go       ← distributed lock (prevent double-assignment)
│       ├── registry_client.go  ← gRPC client for agent-registry
│       ├── nats_events.go      ← publish task lifecycle events
│       └── watchdog.go         ← background goroutine: enforce timeouts
├── go.mod
└── Dockerfile
```

Config env prefix: `ZYNAX_BROKER_` · gRPC port: 50052

---

## Running Tests

```bash
cd services/task-broker
GOWORK=off go test ./... -race -timeout 60s

# BDD contract tests
cd protos/tests
GOWORK=off go test ./task_broker_service/... -v -timeout 60s
```
