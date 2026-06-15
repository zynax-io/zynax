# REASONS Canvas — EPIC C: Context Propagation

> Tier 1 (public-safe). Tier 2 → `canvas.private.md`. Run `/spdd-security-review` before committing.

**Issue:** #1168 · **Milestone:** M7 (v0.6.0)
**Author:** M7 program plan · **Date:** 2026-06-15 · **Status:** Draft

---

## R — Requirements
- **Problem:** there is no deterministic context carried across service/engine/agent boundaries —
  neither **trace context** nor **workflow data context** nor a stable **correlation id**.
- A request id set at the gateway must appear in **every** downstream span and log line for that run.
- Agents must receive a **documented, deterministic** context on handoff and return a defined context.
- **Done when:** a request-id set at api-gateway is observable in every downstream span+log; the
  agent handoff contract is documented and exercised by a test.

## E — Entities
```
RequestContext = { trace_id, span_id, request_id, namespace }
ContextCarrier  ← gRPC metadata · Temporal memo/header · NATS header
WorkflowDataContext (from EPIC W)  ← workflow-scoped data, read/write-scoped
AgentHandoff contract              ← inbound context an agent receives / outbound it returns
```

## A — Approach
**We will:** propagate W3C `traceparent` + `x-request-id` + `x-namespace` through every gRPC hop,
Temporal memo, and NATS header; define explicit read/write scoping for the data context (EPIC W);
document the agent handoff contract.
**We will NOT:** build long-term memory/RAG or context compression — **deferred to M-dx**.
**Governing ADRs:** ADR-031 (context model — this EPIC), ADR-030 (trace context), ADR-008 (no shared state).

## S — Structure (first S)
```
libs/zynaxotel/ (propagators)   ← inject/extract carriers
services/*/ (interceptors)       ← attach RequestContext to ctx
services/engine-adapter/         ← Temporal memo/header carrier; data-context scoping
agents/sdk/                       ← inbound context extraction; handoff helpers
docs/context/                     ← context model + handoff contract guide
```

## O — Operations (stories — `spdd-story` form)
**C.1 — ADR: context model** · S · `adr-proposal`
- As a `maintainer`, I want trace vs data vs correlation contexts defined so propagation is deterministic.
- AC: [ ] ADR-031 committed (carriers, inheritance, handoff rules, non-goals). Deps: none.

**C.2 — Propagate correlation context across all hops** · M · `feat`
- As an `operator`, I want `request_id`/`namespace`/`traceparent` on every hop so a run is traceable.
- AC: [ ] gRPC metadata + Temporal memo + NATS headers carry the context; [ ] visible in every span+log. Deps: C.1, O.5.

**C.3 — Workflow data-context scoping** · M · `feat`
- As a `workflow author`, I want explicit read/write scoping so states can't leak data across runs.
- AC: [ ] read/write scoping enforced on the EPIC-W data context; [ ] cross-run access denied; [ ] tested. Deps: W.4, C.1.

**C.4 — Agent handoff contract** · S · `feat`/`docs`
- As an `agent author`, I want a documented handoff contract so agents receive/return deterministic context.
- AC: [ ] contract documented; [ ] SDK helper to read inbound + emit outbound context; [ ] example test. Deps: C.2.

**Order:** C.1 → C.2 → {C.3, C.4}.

## N — Norms
- W3C tracecontext standard; no bespoke header formats. `Signed-off-by:` + `Assisted-by:` per commit.
- Cross-service only via gRPC (ADR-008); `GOWORK=off` (ADR-017).

## S — Safeguards (second S)
### Context Security
- [ ] No Tier 2 content; [ ] no PII in context fields; [ ] no prompt-injection; [ ] `/spdd-security-review` — PENDING

### Feature Safeguards
- Never put secrets/credentials into propagated context or memo — correlation ids only.
- Never share a data context across workflows or namespaces — strict run+namespace scoping.
- Never rely on implicit globals for context — always explicit carriers on `ctx`.
