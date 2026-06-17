<!-- SPDX-License-Identifier: Apache-2.0 -->

# Context System — correlation, trace, and data-flow propagation

Every Zynax run threads a small, **deterministic context** from the api-gateway,
through the engine, across each agent hop, and back. This guide explains the two
kinds of context, the exact carrier keys, and the SDK helpers an agent author
uses to honour the contract so a run stays traceable and data-scoped end to end.

There are two distinct kinds of context, propagated by different mechanisms:

1. **Correlation + trace context** — flows *on the wire* as gRPC metadata /
   HTTP headers, so one run is one stitched trace and one searchable log group.
2. **Data-flow context** — the run-scoped key/value store that carries a state's
   output to a downstream state's input. It lives **server-side** and is never
   handed to the agent as a store handle; the agent only ever sees the scope
   identifiers.

---

## 1. Correlation + trace context

### Carrier keys

These keys are defined once and mirrored byte-for-byte across Go and Python so a
`request-id` set at the gateway is the *same* value every downstream hop observes
— there are no bespoke header formats.

| Context field | HTTP header (ingress) | gRPC metadata key (hops) | Set by |
|---------------|-----------------------|--------------------------|--------|
| Correlation id | `X-Request-ID` | `request-id` | api-gateway correlation interceptor (generated when absent) |
| Namespace | `X-Namespace` | `x-namespace` | api-gateway correlation interceptor |
| Trace context | `traceparent` (W3C) | `traceparent` (W3C) | tracing interceptor |
| Trace vendor data | `tracestate` (W3C) | `tracestate` (W3C) | tracing interceptor |

The HTTP header constants live in
`services/api-gateway/internal/api/requestid.go`; the gRPC metadata constants
(`requestIDMetaKey = "request-id"`, `namespaceMetaKey = "x-namespace"`) live in
`services/api-gateway/internal/infrastructure/clients.go`. `traceparent` /
`tracestate` are the standard W3C Trace Context keys propagated by the
OpenTelemetry propagator in `libs/zynaxobs/propagation.go` — they are *not*
custom Zynax keys.

`X-Request-ID` is **generated at the gateway when the inbound request omits it**,
so every run has a correlation id even when the caller supplies none. The
gateway-set value is authoritative: downstream code reads the wire value first
and only falls back to a request field when the wire carried none.

### What crosses the handoff — and what never does

Only **correlation ids and W3C trace headers** cross a handoff. Auth tokens,
cookies, API keys, session data, and secrets never do. The SDK enforces this
with an explicit forbidden-key set — `authorization`, `cookie`, `x-api-key`,
`set-cookie`, `proxy-authorization` are dropped before a context is ever built,
so a credential cannot leak into a downstream hop even by accident.

### The handoff contract (`HandoffContext`)

An agent receives, and forwards, a single immutable value. `HandoffContext`
(`agents/sdk/src/zynax_sdk/handoff.py`) is a frozen dataclass with these fields:

| Field | Meaning |
|-------|---------|
| `request_id` | Stable correlation id; appears in every downstream span and log line. Empty when the call did not originate behind the gateway. |
| `namespace` | Tenant / routing namespace; also half of the data-context scope key. |
| `workflow_id` | The workflow run id; with `namespace` it scopes the data context. |
| `task_id` | The opaque per-task identifier from the originating request. |
| `traceparent` | W3C `traceparent` carrying the remote span, or empty. |
| `tracestate` | W3C `tracestate` vendor data, or empty. |

`HandoffContext.is_traced()` returns `True` when a `traceparent` was carried.

### SDK handoff helpers (from #1196)

Two helpers in `zynax_sdk.handoff` are all an agent author needs. The helpers
keep correlation and trace alive across the *next* hop the agent itself makes.

**`inbound_context(request, context=None)`** — read the context the agent
**receives**. Correlation (`request_id`, `namespace`) and the W3C trace headers
are read from the inbound gRPC metadata when `context` is supplied;
`request_id`, `workflow_id`, and `task_id` fall back to the proto request fields
so the context is populated even outside a transport (e.g. in unit tests). It
never raises on missing fields — absent identifiers come back as empty strings.

```python
from zynax_sdk.handoff import inbound_context, outbound_metadata

def ExecuteCapability(self, request, context):
    ctx = inbound_context(request, context)   # frozen HandoffContext
    # ... do the work ...
```

**`outbound_metadata(ctx)`** — emit the gRPC metadata that **forwards** the
context to the next hop. Unset fields are omitted (no empty headers), and the
ordering is deterministic (`request-id` → `x-namespace` → `traceparent` →
`tracestate`) so two equal contexts always produce identical metadata.

```python
    # when this agent calls another Zynax hop, forward the context:
    downstream_stub.SomeMethod(req, metadata=outbound_metadata(ctx))
```

That single round-trip — `inbound_context` in, `outbound_metadata` out — is what
keeps a multi-hop run as **one trace** and **one correlation group**.

---

## 2. Data-flow context (scoped, #1195)

The data-flow context is the run-scoped key/value store that carries a state's
output to a downstream state's input. It backs workflow data-flow bindings and
is defined in `services/engine-adapter/internal/domain/datacontext.go`.

### Run scoping — cross-run access is denied

A data context is bound to exactly one **`DataContextScope`** at construction:

```go
type DataContextScope struct {
    RunID     string  // the workflow run / envelope workflow_id
    Namespace string  // the run's namespace (may be empty single-namespace)
}
```

Every read and write must present a scope **equal** to the owning scope. One run
can never read or write another run's data — even if it somehow obtains a
reference to the instance. A mismatched access fails *loudly* with a
`ScopeError` rather than silently returning or dropping data. Prefer
`NewScopedWorkflowDataContext(scope)` over the back-compat
`NewWorkflowDataContext()` so cross-run access is denied. This is why the agent
receives only the `namespace` + `workflow_id` identifiers and never the store
handle itself — it has no way to reach across runs.

### Keys and references

`WorkflowDataContext` keys are canonical dotted paths:

```
states.<stateID>.output.<key>
```

An input binding references a stored value with the `$.` prefix, e.g.
`$.states.search.output.results`. There is no implicit fallback to an empty
value: a reference that cannot resolve, or a value that is not a coercible
scalar, fails the run with a structured `DataReferenceError` (the offending
`InputKey`, `Reference`, and `Reason`).

The store is written only by an action's `output_bindings` and read only by a
downstream action's `input_bindings` — there is no global mutable state. It
lives for a single run and is never persisted beyond it.

---

## See also

- Workflow data-flow bindings: [`docs/authoring/workflows.md`](../authoring/workflows.md)
- Git MCP guide (a concrete handoff surface): [`docs/git-mcp/git-mcp.md`](../git-mcp/git-mcp.md)
- OpenTelemetry setup (where `traceparent` becomes a stitched trace): [`docs/observability/opentelemetry.md`](../observability/opentelemetry.md)
- Handoff contract source: `agents/sdk/src/zynax_sdk/handoff.py`
- Data-context source: `services/engine-adapter/internal/domain/datacontext.go`
