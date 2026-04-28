# services/workflow-compiler — AGENTS.md

> **Language: Go 1.22+**
> **★ NEW SERVICE** — See `ARCHITECTURE.md §3` for full IR design.

---

## Purpose

The Workflow Compiler is the **brain of the control plane**. It translates
YAML workflow manifests into the Canonical IR (Intermediate Representation)
and routes executions to the correct workflow engine.

Think of it as the "LLVM for workflows": YAML is the high-level language,
IR is the portable bytecode, engine adapters are the backends.

**Responsibilities:**
- Parse and validate YAML manifests against JSON Schema.
- Compile YAML → Canonical IR (engine-agnostic state machine representation).
- Select the target workflow engine (Temporal / LangGraph / Argo) based on Policy.
- Submit IR to the Engine Adapter for execution.
- Store compiled workflows and execution state in PostgreSQL.
- Apply updates (`zynax apply`) idempotently.
- Diff existing vs new workflow and emit change events.

**Non-responsibilities:**
- Does NOT execute workflows (that is the engine adapter).
- Does NOT route capability calls (that is task-broker).
- Does NOT store agent memory (that is memory-service).

---

## Internal Layout

```
services/workflow-compiler/
├── cmd/workflow-compiler/main.go
├── internal/
│   ├── api/
│   │   ├── handler.go          ← ApplyWorkflow, GetWorkflow, DeleteWorkflow, DryRun
│   │   └── middleware.go
│   ├── domain/
│   │   ├── ir.go               ← WorkflowIR, StateIR, ActionIR, TransitionIR types
│   │   ├── compiler.go         ← YAMLCompiler: YAML → IR (core logic)
│   │   ├── validator.go        ← Schema validation, semantic validation
│   │   ├── differ.go           ← Diff two IRs to compute change set
│   │   ├── repository.go       ← WorkflowRepository interface (port)
│   │   └── errors.go           ← ErrInvalidYAML, ErrSchemaViolation, ErrUnknownCapability
│   ├── infrastructure/
│   │   ├── postgres.go         ← WorkflowRepository implementation
│   │   ├── engine_client.go    ← gRPC client for engine-adapter service
│   │   └── registry_client.go  ← Validates capabilities exist in agent-registry
│   └── config/
│       └── config.go           ← prefix: ZYNAX_COMPILER_
├── tests/
│   ├── features/workflow_compiler.feature
│   └── unit/
│       ├── compiler_test.go    ← Table-driven + godog BDD
│       └── validator_test.go
└── Dockerfile
```

---

## Domain: Workflow IR

```go
// internal/domain/ir.go

// WorkflowIR is the canonical, engine-agnostic representation of a workflow.
// It is the output of compilation and the input to engine adapters.
// This type must NEVER contain engine-specific concepts.
type WorkflowIR struct {
    ID           string
    Name         string
    Namespace    string
    Version      string            // content-hash of source YAML
    InitialState string
    States       map[string]*StateIR
    Triggers     []TriggerIR
    Metadata     map[string]string
}

type StateType string
const (
    StateTypeActive         StateType = "active"
    StateTypeTerminal       StateType = "terminal"
    StateTypeHumanInLoop    StateType = "human_in_the_loop"
    StateTypeWaiting        StateType = "waiting"
)

type StateIR struct {
    Name        string
    Type        StateType
    Actions     []ActionIR
    Transitions []TransitionIR
    Timeout     *time.Duration
}

// ActionIR represents a capability invocation.
// NEVER contains an agent name — only a capability name.
type ActionIR struct {
    Capability string
    InputMap   map[string]string   // template expressions → workflow context fields
    OutputMap  map[string]string
    Async      bool
    Timeout    *time.Duration
}

// TransitionIR defines when and how to move to the next state.
type TransitionIR struct {
    OnEvent string
    Guard   *ExpressionIR         // optional condition expression
    Goto    string
}

type TriggerIR struct {
    EventPattern string            // e.g. "github.pull_request.opened"
    Filter       map[string]string // additional filter criteria
}
```

---

## Domain: Compiler

```go
// internal/domain/compiler.go

type YAMLCompiler struct {
    validator  *Validator
    schemaPath string
}

// Compile transforms a raw YAML manifest into a WorkflowIR.
// Returns ErrInvalidYAML if YAML is malformed.
// Returns ErrSchemaViolation if manifest fails JSON Schema validation.
// Returns ErrUnknownCapability if a declared capability is not in the registry.
func (c *YAMLCompiler) Compile(ctx context.Context, raw []byte) (*WorkflowIR, error) {
    // Step 1: Parse YAML
    manifest, err := parseManifest(raw)
    if err != nil { return nil, fmt.Errorf("parse yaml: %w", ErrInvalidYAML) }

    // Step 2: JSON Schema validation
    if err := c.validator.ValidateSchema(manifest); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrSchemaViolation, err)
    }

    // Step 3: Semantic validation (state reachability, no orphan states, etc.)
    if err := c.validator.ValidateSemantic(manifest); err != nil {
        return nil, fmt.Errorf("semantic validation: %w", err)
    }

    // Step 4: Transform to IR
    ir, err := c.transform(manifest)
    if err != nil { return nil, fmt.Errorf("transform to IR: %w", err) }

    return ir, nil
}
```

---

## gRPC API

```protobuf
service WorkflowCompilerService {
    // Apply a YAML manifest (idempotent — creates or updates)
    rpc ApplyWorkflow(ApplyWorkflowRequest) returns (ApplyWorkflowResponse);

    // Dry-run: compile and validate without executing
    rpc DryRun(DryRunRequest) returns (DryRunResponse);

    // Get the compiled IR for a workflow
    rpc GetWorkflow(GetWorkflowRequest) returns (GetWorkflowResponse);

    // Delete a workflow (cancels running executions)
    rpc DeleteWorkflow(DeleteWorkflowRequest) returns (DeleteWorkflowResponse);

    // List workflows in a namespace
    rpc ListWorkflows(ListWorkflowsRequest) returns (ListWorkflowsResponse);
}
```

---

## Configuration

```go
// prefix: ZYNAX_COMPILER_
type Config struct {
    GRPCPort          int    `envconfig:"GRPC_PORT"          default:"50055"`
    HealthPort        int    `envconfig:"HEALTH_PORT"        default:"8080"`
    MetricsPort       int    `envconfig:"METRICS_PORT"       default:"9090"`
    DatabaseURL       string `envconfig:"DATABASE_URL"       required:"true"`
    EngineAdapterURL  string `envconfig:"ENGINE_ADAPTER_URL" required:"true"`
    RegistryURL       string `envconfig:"REGISTRY_URL"       required:"true"`
    SchemaDir         string `envconfig:"SCHEMA_DIR"         default:"/app/schemas"`
    DefaultEngine     string `envconfig:"DEFAULT_ENGINE"     default:"temporal"`
    ShutdownGraceSecs int    `envconfig:"SHUTDOWN_GRACE_SECS" default:"30"`
    LogLevel          string `envconfig:"LOG_LEVEL"          default:"INFO"`
    OtelEndpoint      string `envconfig:"OTEL_ENDPOINT"      default:"http://otel-collector:4317"`
    ServiceName       string `envconfig:"SERVICE_NAME"       default:"workflow-compiler"`
}
```

---

## BDD Scenarios

```gherkin
Feature: Workflow Compilation

  Scenario: Valid YAML workflow compiles to correct IR
    Given a valid Workflow YAML with states [review, fix, merge, done]
    When ApplyWorkflow is called
    Then the compiled IR has 4 states
    And the initial_state is "review"
    And each state has the correct transitions

  Scenario: YAML with unknown capability is rejected
    Given a Workflow YAML with action capability "nonexistent_cap"
    And "nonexistent_cap" is not registered in agent-registry
    When ApplyWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error mentions "unknown capability: nonexistent_cap"

  Scenario: YAML with no terminal state is rejected
    Given a Workflow YAML where no state has type: terminal
    When ApplyWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT
    And the error mentions "no terminal state"

  Scenario: Unreachable state is rejected
    Given a Workflow YAML with an orphan state "orphan" with no transitions pointing to it
    When ApplyWorkflow is called
    Then the gRPC status is INVALID_ARGUMENT

  Scenario: Apply is idempotent
    Given a Workflow "my-workflow" has been applied
    When the same YAML is applied again
    Then the gRPC status is OK
    And no duplicate workflow is created

  Scenario: DryRun returns compiled IR without executing
    Given a valid Workflow YAML
    When DryRun is called
    Then the response contains the compiled IR
    And no workflow execution is started
    And agent-registry is NOT queried for capability validation
```

---

## Running Tests

```bash
# All unit tests for this service (always use GOWORK=off — ADR-017)
cd services/workflow-compiler
GOWORK=off go test ./... -race -timeout 60s

# With coverage
GOWORK=off go test ./... -coverprofile=coverage.out -covermode=atomic
GOWORK=off go tool cover -func=coverage.out | grep total:

# BDD contract tests (in protos/tests/ — separate module)
cd ../../protos/tests
GOWORK=off go test ./workflow_compiler_service/... -v -timeout 60s

# Via Makefile (runs inside Docker — no local Go needed)
make test-unit-svc SVC=workflow-compiler
make test-bdd
```

---

## Common AI Mistakes

| Mistake | Why it fails | Correct approach |
|---------|-------------|-----------------|
| `go test` without `GOWORK=off` | Workspace resolves missing modules → unrelated error | `GOWORK=off go test ./...` — every time (ADR-017) |
| Calling `ParseManifest` with YAML that uses `transitions:` / `event_type:` / `target_state:` | The parser expects `on:` / `event:` / `goto:` — wrong keys silently produce zero transitions | Check the `yamlTransition` struct field tags in `domain/parser.go` |
| Returning a raw `error` from `CompileWorkflow` instead of a gRPC status | Client receives `Unknown` status — no actionable error code | Map domain errors to `status.Errorf(codes.InvalidArgument, …)` |
| Adding business logic to `internal/api/server.go` | `api/` is a translation layer only; logic in the wrong layer bypasses validators | Put new logic in `internal/domain/`; the server just calls domain functions |
| Writing `ToIR` output that is non-deterministic (e.g. unsorted states) | Flaky tests; same input produces different proto byte sequences | Sort map keys before iterating — see `sort.Strings(ids)` in `ir/ir.go` |
