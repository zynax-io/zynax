# services/task-broker — AGENTS.md

> Go 1.26.3+. Inherits rules from root `AGENTS.md` and `services/AGENTS.md`.
> **Status: M5 Implemented — quality cleanup in progress (#531, #532).**
> Implementation merged in PRs #520, #522, #523. BDD contract tests in `protos/tests/task_broker_service/`.

---

## Purpose

The Task Broker is the **work scheduler** of the mesh.

- Accepts `DispatchTask` calls from `engine-adapter`; returns a broker-assigned `task_id` immediately (async execution).
- Discovers eligible agents from `agent-registry` via gRPC (`AgentFinder` port).
- Assigns tasks to agents using atomic round-robin selection.
- Drives the task lifecycle state machine: `PENDING → DISPATCHED → COMPLETED | FAILED | CANCELLED`. Failed tasks with remaining retries transition to `RETRYING` before re-dispatching.
- Executes capability invocations via `CapabilityExecutor` (HTTP-based agent call).
- Exposes `AcknowledgeTask` for agents to report outcomes; applies retry logic in the domain layer.
- Exposes `ListTasks` for filtered, paginated queries (by workflow, status, or agent).

Does NOT: execute tasks directly · store results permanently · authenticate callers.

---

## gRPC API

**Service:** `zynax.v1.TaskBrokerService`
**Port:** `50053` (env: `ZYNAX_BROKER_GRPC_PORT`)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `DispatchTask` | `DispatchTaskRequest` | `DispatchTaskResponse` | Submit a task; returns `task_id` + `created_at` |
| `AcknowledgeTask` | `AcknowledgeTaskRequest` | `AcknowledgeTaskResponse` | Agent reports outcome (COMPLETED/FAILED/CANCELLED); triggers retry logic |
| `GetTask` | `GetTaskRequest` | `WorkflowTask` | Fetch current task state by ID |
| `ListTasks` | `ListTasksRequest` | `ListTasksResponse` | Paginated list filtered by workflow, status, or agent |
| `CancelTask` | `CancelTaskRequest` | `CancelTaskResponse` | Transition a non-terminal task to CANCELLED |

---

## TaskStatus Enum

| Value | Ordinal | Description |
|-------|---------|-------------|
| `PENDING` | 1 | Task accepted; waiting for agent dispatch |
| `DISPATCHED` | 2 | Routed to an agent; execution in progress |
| `RETRYING` | 3 | Agent reported failure; retry budget remaining |
| `COMPLETED` | 4 | Terminal — successful execution |
| `FAILED` | 5 | Terminal — retries exhausted |
| `CANCELLED` | 6 | Terminal — explicitly cancelled via `CancelTask` |

Ordinal values are stable and must never be reordered or reassigned (ADR-001).

---

## Internal Layout

```
services/task-broker/
├── cmd/task-broker/main.go          ← wires repo + finder + executor + gRPC server
├── internal/
│   ├── api/
│   │   └── handler.go               ← gRPC handler: DispatchTask, AcknowledgeTask, GetTask, ListTasks, CancelTask
│   ├── domain/
│   │   ├── model.go                 ← TaskStatus, Task, TaskError, AgentInfo, ListFilter, ListResult
│   │   ├── ports.go                 ← TaskRepository, AgentFinder, CapabilityExecutor interfaces
│   │   ├── service.go               ← TaskService: core dispatch, acknowledge, cancel, list logic
│   │   ├── errors.go                ← ErrTaskNotFound, ErrNoEligibleAgent, ErrTaskTerminal, ErrInvalidArgument
│   │   └── service_test.go
│   └── infrastructure/
│       ├── memory_repo.go           ← in-memory TaskRepository (M5; Postgres in M6)
│       ├── agent_executor.go        ← CapabilityExecutor: HTTP invocation of agent endpoints
│       └── registry_client.go       ← AgentFinder: gRPC client for agent-registry
├── tests/
│   └── features/task_broker.feature ← BDD contract scenarios
├── go.mod
├── go.sum
└── Dockerfile
```

**Hexagonal layout invariants:**
- `internal/domain/` — pure business logic; zero imports from `api` or `infrastructure`.
- `internal/api/` — gRPC handler; imports `domain` types only; translates proto ↔ domain.
- `internal/infrastructure/` — port implementations; imports `domain` interfaces; never imported by `api`.

---

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `ZYNAX_BROKER_GRPC_PORT` | `50053` | gRPC listen port |
| `ZYNAX_BROKER_REGISTRY_ADDR` | `localhost:50052` | agent-registry gRPC address |
| `ZYNAX_BROKER_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |

---

## Error Mapping

| Domain error | gRPC code |
|---|---|
| `ErrTaskNotFound` | `NOT_FOUND` |
| `ErrNoEligibleAgent` | `NOT_FOUND` |
| `ErrTaskTerminal` | `FAILED_PRECONDITION` |
| `ErrInvalidArgument` | `INVALID_ARGUMENT` |
| `context.Canceled` | `CANCELED` |
| `context.DeadlineExceeded` | `DEADLINE_EXCEEDED` |
| all other errors | `INTERNAL` |

---

## Running Tests

```bash
cd services/task-broker
GOWORK=off go test ./... -race -timeout 60s

# BDD contract tests
cd protos/tests
GOWORK=off go test ./task_broker_service/... -v -timeout 60s
```

Coverage gate: `internal/domain/` ≥ 90% (ADR-016). Current: 92.7%.
