<!-- SPDX-License-Identifier: Apache-2.0 -->

# Architecture Patterns — Zynax

> Patterns Zynax uses, patterns it should adopt, and when each applies.
> Every pattern here is either enforced by CI or tracked by an open issue.
> Reference: `docs/architecture/2026-05-20-principal-architect-review.md`

---

## Patterns In Use

### Hexagonal Architecture (Ports & Adapters)

**Use when:** Building a platform service.

Every platform service in `services/*/` has:
- `internal/domain/` — pure business logic; zero infrastructure imports
- `internal/api/` — gRPC handler; maps proto types to domain types
- `internal/infrastructure/` — concrete adapters (DB, gRPC clients, Temporal)

```
gRPC call
    → api/ (handler: validate, map proto → domain)
    → domain/ (business logic: domain types only)
    → infrastructure/ (concrete: talk to Temporal, DB, other services)
```

The `layer-boundaries` CI gate enforces that `domain/` has zero proto imports.
This is verified by the 2026-05-20 review as a "crown jewel" — do not erode it.

---

### WorkflowEngine Interface (Strategy Pattern)

**Use when:** Plugging in a new workflow engine.

```go
// services/engine-adapter/internal/domain/engine.go
type WorkflowEngine interface {
    Submit(ctx, ir, input) (ExecutionID, error)
    Signal(ctx, id, event) error
    GetWorkflowStatus(ctx, id) (*ExecutionState, error)
    Cancel(ctx, id, reason) error
    Watch(ctx, id) (<-chan ExecutionEvent, error)
    Name() string
}
```

To add a new engine (e.g. `ArgoEngine`): implement this interface in
`internal/infrastructure/`. Do not touch the gRPC handler or domain code.
Adding a second engine is ~500 LoC. See ADR-015.

---

### Capability Routing (Indirect Dispatch)

**Use when:** A workflow action invokes an external system.

Workflows route to **capabilities** (`summarize`, `open_pr`), never to **named agents**.
The task-broker resolves capabilities to agents at dispatch time. This decouples the
workflow definition from any specific executor.

```
WorkflowIR.ActionIR.Capability = "summarize"
task-broker: FindByCapability("summarize") → agent-registry
agent-registry: returns [agent-a, agent-b]  (round-robin for M5)
task-broker: calls agent-a.ExecuteCapability(...)
```

**Pattern classification:** The dispatch call is a **Command** (gRPC call), not an event.
Task result callbacks are **Event-Carried State Transfer**.

---

### BDD-First Contract Testing

**Use when:** Adding any new gRPC method.

Workflow (ADR-016):
1. Write `.feature` file in `protos/tests/features/`
2. Commit and open PR → CI runs BDD scenarios against stub server
3. Implement domain logic and wire gRPC handler
4. Coverage gate ≥ 90% on `internal/domain/`

Never write implementation code before the feature file is committed and CI-green.

---

### Idempotent Apply (SHA-256 Manifest Hash)

**Use when:** Accepting workflow manifest submissions.

`ManifestWorkflowID(yaml)` computes a SHA-256 of the canonical YAML and takes the
first 16 hex chars as the stable workflow ID. Same manifest → same `run_id`.
Re-submitting a *completed* workflow appends a Unix timestamp suffix.

This makes `zynax apply` safe to retry and GitOps-friendly. See `cmd/zynax/apply.go`
and `services/api-gateway/internal/domain/apply.go`.

---

### REASONS Canvas (SPDD)

**Use when:** Starting any `feat:` PR.

Every `feat:` PR requires a REASONS Canvas at `docs/spdd/<issue>-<slug>/canvas.md`
committed before any implementation code. The canvas records the reasoning (why this
change), the options considered, and the operations steps.

CI enforces this via `validate-canvas`. See ADR-019 and `docs/patterns/spdd-guide.md`.

---

## Patterns to Adopt (Tracked)

### Outbox Pattern (for reliable event publish)

**Use when:** event-bus is implemented (M6+).

When event-bus goes live, the naive implementation will have a dual-write hazard:
1. Temporal records the state transition
2. Code also publishes a CloudEvent to NATS

If step 2 fails, Temporal retries the Activity — but the event was never published.
The outbox pattern avoids this: persist the event to a transactional store alongside
the state change, then publish from the outbox asynchronously.

**Zynax's mitigation:** Temporal's Activity retry already provides at-least-once delivery
for the `PublishLifecycleEventActivity` call. When implementing, ensure the publish is
idempotent (CloudEvents `id` field as deduplication key) and treat NATS publish errors
as retriable (Activity retry handles this).

**Action:** File an ADR when event-bus is designed.

---

### Circuit Breaker / Timeout Budget

**Use when:** Making gRPC calls from one service to another.

Today, gRPC client calls have no explicit deadlines. Under load, a slow task-broker
will cause engine-adapter Activities to pile up and exhaust Temporal's thread pool.

**Recommended:**
- Add `context.WithTimeout(ctx, 30s)` on every outgoing gRPC call
- Set explicit `RetryPolicy` on Temporal Activities (#569)
- Consider `grpc.WithBlock() + DialTimeout` on startup for fast-fail

---

### OpenTelemetry Distributed Tracing

**Use when:** Adding any new service or significant new code path.

Today only workflow-compiler has OTel tracing. A workflow run is untraceable end-to-end.
Target: all three live services (api-gateway, engine-adapter, workflow-compiler) export
traces to an OTLP collector with the same `workflow_id` as the root span attribute.

See #491. Pattern: use `go.opentelemetry.io/otel/trace` and inject the tracer via the
service constructor (not a global).

---

## Event-Pattern Classification Reference (Fowler)

Applied to Zynax flows — use these labels when discussing events:

| Flow | Pattern | Why |
|---|---|---|
| CloudEvents `state.entered/exited` | **Event Notification** | Announces a change; consumers call back to query state |
| `task.completed` with result | **Event-Carried State Transfer** | Result payload carried so consumer doesn't call back |
| Temporal activity history | **Event Sourcing** (Temporal-internal) | Temporal maintains replayable event log |
| `DispatchCapabilityActivity` call | **Command** (gRPC) | Expects a specific response; not fire-and-forget |

The 2026-05-20 review calls out the event-notification trap: "cross-service logical flow
only visible at runtime in the IR-interpreter." Mitigation: the WorkflowIR state machine
IS the explicit contract — the events are callbacks to it, not the control flow itself.
