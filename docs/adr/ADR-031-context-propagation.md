# ADR-031: Context propagation model (trace · data · correlation)

**Status:** Proposed  **Date:** 2026-06-15
**Related:** ADR-030 (OTEL/Uptrace), ADR-029 (data-flow), ADR-008 (no shared databases/state)

---

## Context

For long-running autonomous workflows and expert handoffs, three distinct kinds of context must move
across service, engine (Temporal), and event (NATS) boundaries: **trace context** (for observability),
**workflow data context** (for data-flow, ADR-029), and a stable **correlation id** (for log/trace
joining). Today none of these propagate deterministically. Conflating them — e.g. stuffing data into
trace baggage — would be a hard-to-reverse mistake.

## Decision

1. **Trace context:** W3C `traceparent` propagated via gRPC metadata, Temporal memo/headers, and NATS
   message headers (per ADR-030).
2. **Correlation:** `x-request-id` (and `x-namespace`) propagated on every hop; emitted into every span
   and log line.
3. **Data context:** the workflow-run-scoped `WorkflowDataContext` (ADR-029), with **explicit read/write
   scoping** — never shared across runs or namespaces.
4. **Agent handoff:** a documented contract specifying exactly what context an agent receives on dispatch
   and returns on completion. No implicit globals.
5. The three contexts are **kept separate** — data is never carried in trace baggage.

## Rationale

| Option | Assessment |
|--------|------------|
| Three separate, explicit contexts (chosen) | ✅ Clear ownership; observability and data concerns don't entangle |
| Single merged "context blob" | ✗ Rejected — couples observability to data; leakage and size risks |
| Implicit ambient context | ✗ Rejected — non-deterministic; violates explicit-scoping |

## Consequences

- **Positive:** a request-id set at the gateway is observable in every downstream span+log; handoffs are
  deterministic and documented; data stays run-scoped.
- **Negative / trade-off:** every boundary (gRPC, Temporal, NATS) needs an explicit carrier
  inject/extract — more wiring, covered by `libs/zynaxobs` propagators.
- **Neutral / follow-up:** long-term memory, RAG, and context compression are deferred to M-dx.
