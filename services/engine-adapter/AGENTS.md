# services/engine-adapter — AGENTS.md

> **Language: Go 1.22+**
> **★ NEW SERVICE** — See `ARCHITECTURE.md §6` for adapter design.

---

## Purpose

The Engine Adapter is the **execution bridge** between the Zynax IR
and concrete workflow engines. It implements one Go interface (`WorkflowEngine`)
with different backends (Temporal, LangGraph, Argo) selectable at deploy time.

Think of it as the containerd of Zynax: a stable interface for
pluggable execution backends.

**Responsibilities:**
- Implement `WorkflowEngine` for Temporal (primary target, M3).
- Implement `WorkflowEngine` for LangGraph (M5).
- Implement `WorkflowEngine` for Argo Workflows (M6).
- Translate Canonical IR → engine-native format.
- Translate engine-native events → Zynax `WorkflowEvent`.
- Route capability invocations to task-broker.
- Relay task results back to the engine as workflow signals.
- Stream execution state changes via gRPC server-streaming.

**Non-responsibilities:**
- Does NOT compile YAML (that is workflow-compiler).
- Does NOT route capabilities (that is task-broker).
- Does NOT decide which engine to use (that is workflow-compiler).

---

## Internal Layout

```
services/engine-adapter/
├── cmd/engine-adapter/main.go
├── internal/
│   ├── api/
│   │   ├── handler.go           ← Submit, Signal, Query, Cancel, Watch
│   │   └── middleware.go
│   ├── domain/
│   │   ├── engine.go            ← WorkflowEngine interface (the core port)
│   │   ├── model.go             ← ExecutionID, ExecutionState, WorkflowEvent
│   │   └── errors.go            ← ErrEngineUnavailable, ErrExecutionNotFound
│   ├── infrastructure/
│   │   ├── adapters/
│   │   │   ├── temporal.go      ← TemporalEngine — primary target
│   │   │   ├── langgraph.go     ← LangGraphEngine (M5)
│   │   │   └── argo.go          ← ArgoEngine (M6)
│   │   ├── broker_client.go     ← gRPC client to task-broker (capability dispatch)
│   │   └── registry.go          ← EngineRegistry: selects engine by name
│   └── config/
│       └── config.go            ← prefix: ZYNAX_ENGINE_
├── tests/
│   ├── features/engine_adapter.feature
│   └── unit/
└── Dockerfile
```

---

## The WorkflowEngine Interface

```go
// internal/domain/engine.go

// WorkflowEngine is the ONLY interface engine adapters implement.
// A new engine = a new struct that satisfies this interface.
// Selecting a different engine = config change, zero code change.
type WorkflowEngine interface {
    // Name identifies this engine for routing decisions
    Name() string

    // Submit starts a workflow from a compiled IR
    Submit(ctx context.Context, ir compiler.WorkflowIR, input map[string]any) (ExecutionID, error)

    // Signal injects an external event into a running workflow
    // (e.g. "review.approved", "push", "human.approved")
    Signal(ctx context.Context, id ExecutionID, event WorkflowEvent) error

    // Query returns the current execution state (non-blocking)
    Query(ctx context.Context, id ExecutionID) (*ExecutionState, error)

    // Cancel terminates a running workflow gracefully
    Cancel(ctx context.Context, id ExecutionID, reason string) error

    // Watch returns a channel that receives execution state changes
    Watch(ctx context.Context, id ExecutionID) (<-chan ExecutionEvent, error)
}
```

---

## Temporal Adapter (Primary)

```go
// internal/infrastructure/adapters/temporal.go

type TemporalEngine struct {
    client     client.Client
    broker     task_broker.TaskBrokerServiceClient
    taskQueue  string
}

func (e *TemporalEngine) Name() string { return "temporal" }

func (e *TemporalEngine) Submit(
    ctx context.Context,
    ir compiler.WorkflowIR,
    input map[string]any,
) (ExecutionID, error) {
    // Translate IR → Temporal workflow options
    // The Temporal workflow definition is a generic "state machine runner"
    // that interprets the IR at runtime — no code generation required
    workflowOptions := client.StartWorkflowOptions{
        ID:        fmt.Sprintf("%s/%s", ir.Namespace, ir.Name),
        TaskQueue: e.taskQueue,
        // Temporal handles retries, timeouts at the workflow level
    }
    run, err := e.client.ExecuteWorkflow(ctx, workflowOptions,
        "zynax-state-machine",  // generic Temporal worker
        ir, input,
    )
    if err != nil { return "", fmt.Errorf("submit to temporal: %w", err) }
    return ExecutionID(run.GetID()), nil
}

func (e *TemporalEngine) Signal(
    ctx context.Context, id ExecutionID, event WorkflowEvent,
) error {
    return e.client.SignalWorkflow(ctx, string(id), "",
        event.Type, event.Payload,
    )
}
```

## Capability Dispatch (Cross-Service)

When the Temporal worker needs to execute a capability (e.g. `summarize`),
it calls back into Zynax via task-broker. The engine adapter provides
a Temporal Activity that bridges this:

```go
// internal/infrastructure/adapters/temporal_activity.go

// DispatchCapabilityActivity is a Temporal Activity that calls task-broker.
// Temporal handles retries, timeouts, and heartbeats.
func (a *Activities) DispatchCapabilityActivity(
    ctx context.Context,
    req CapabilityRequest,
) (CapabilityResult, error) {
    resp, err := a.broker.SubmitTask(ctx, &pb.SubmitTaskRequest{
        RequestId: req.RequestID,
        Spec: &pb.TaskSpec{
            RequiredCapability: req.Capability,
            Payload:            structpb.NewStringValue(string(req.InputJSON)),
        },
    })
    if err != nil { return CapabilityResult{}, fmt.Errorf("dispatch capability: %w", err) }

    // Poll for result (task-broker is async)
    return a.waitForResult(ctx, resp.TaskId)
}
```

---

## Configuration

```go
// prefix: ZYNAX_ENGINE_
type Config struct {
    GRPCPort           int    `envconfig:"GRPC_PORT"            default:"50056"`
    HealthPort         int    `envconfig:"HEALTH_PORT"          default:"8080"`
    MetricsPort        int    `envconfig:"METRICS_PORT"         default:"9090"`
    ActiveEngine       string `envconfig:"ACTIVE_ENGINE"        default:"temporal"`
    // Temporal
    TemporalAddress    string `envconfig:"TEMPORAL_ADDRESS"     default:"temporal:7233"`
    TemporalNamespace  string `envconfig:"TEMPORAL_NAMESPACE"   default:"zynax"`
    TemporalTaskQueue  string `envconfig:"TEMPORAL_TASK_QUEUE"  default:"zynax-workflows"`
    // Task Broker
    BrokerURL          string `envconfig:"BROKER_URL"           required:"true"`
    ShutdownGraceSecs  int    `envconfig:"SHUTDOWN_GRACE_SECS"  default:"30"`
    LogLevel           string `envconfig:"LOG_LEVEL"            default:"INFO"`
    OtelEndpoint       string `envconfig:"OTEL_ENDPOINT"        default:"http://otel-collector:4317"`
    ServiceName        string `envconfig:"SERVICE_NAME"         default:"engine-adapter"`
}
```

---

## BDD Scenarios

```gherkin
Feature: Workflow Engine Adapter

  Scenario: Submit IR creates a running workflow execution
    Given the Temporal engine is running
    And a compiled WorkflowIR with 3 states
    When Submit is called with the IR
    Then a valid ExecutionID is returned
    And Query returns state RUNNING

  Scenario: Signal transitions workflow state
    Given a running workflow in state "review"
    When Signal is sent with event "review.approved"
    Then Query returns state "merge" (or the next state after review.approved)

  Scenario: Capability dispatch calls task-broker
    Given a workflow execution reaches an action: capability "summarize"
    When the Temporal worker executes the capability activity
    Then a task is submitted to task-broker with capability "summarize"

  Scenario: Cancel terminates the workflow
    Given a running workflow execution
    When Cancel is called
    Then Query returns state CANCELLED

  Scenario: Engine is swappable via config
    Given ZYNAX_ENGINE_ACTIVE_ENGINE is set to "langgraph"
    When a workflow is submitted
    Then the LangGraph engine handles execution (not Temporal)
    And the gRPC API contract remains identical
```
