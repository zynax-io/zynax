# REASONS Canvas — EPIC L: Execution Log/Event Streaming

> Tier 1 (public-safe). Tier 2 → `canvas.private.md`. Run `/spdd-security-review` before committing.

**Issue:** #468 (engine side; EPIC L) · **Milestone:** M7 (v0.6.0)
**Author:** M7 program plan · **Date:** 2026-06-15 · **Status:** Aligned

---

## R — Requirements
- **Problem:** `GET /api/v1/workflows/{id}/logs` returns `{"error":"streaming not supported"}` — a
  developer cannot watch a run. The engine-adapter polls Temporal instead of streaming history.
- A developer must **stream live execution events** (state transitions + capability events) for a run.
- **Done when:** `zynax logs <run-id> --follow` shows each transition/capability event for the
  e2e-demo run with no `streaming not supported` error.

## E — Entities
```
EngineAdapter.HistoryStream      ← GetWorkflowExecutionHistory(isLongPoll=true) → ordered events
  Temporal history event types   ← WorkflowExecutionStarted / ActivityTaskCompleted /
                                    WorkflowExecutionCompleted … → StateTransitionEvent
                                    (from_state/to_state/event_type, ordered, non-empty)
  WatchTask RPC (TaskBrokerSvc)  ← broker pushes activity events (replaces GetTask 500ms poll)
EventBus subscription (scoped)   ← per-workflow CloudEvents (TypePattern + WorkflowID)
api-gateway log stream           ← SSE/chunked HTTP merging engine history + events
CLI follower (`zynax logs`)      ← consumes the stream
```
> L.1 (history long-poll + WatchTask streaming) was delivered via #1180 / PR #1237; the original
> M7.B refactor canvas (`468-history-streaming`) is consolidated here.

## A — Approach
**We will:** replace polling with Temporal **history long-poll** (closes #468); reuse EventBus
`Subscribe` scoped by `WorkflowID`; expose a real streaming `/logs` (SSE/chunked); add `zynax logs --follow`.
**We will NOT:** persist logs to a store (Uptrace handles log UI — EPIC O); add log search/filter (M-dx).
**Governing ADRs:** ADR-015 (engine interface), ADR-022 (event-bus), ADR-016 (.feature first).

## S — Structure (first S)
```
services/engine-adapter/internal/domain/   ← history streaming (replaces polling)
services/event-bus/                          ← scoped subscription reuse
services/api-gateway/internal/api/handler.go ← streaming /logs handler
cmd/zynax/                                    ← `zynax logs --follow`
```
Config env prefix: `ZYNAX_ENGINE_ADAPTER_` / `ZYNAX_GW_`.

## O — Operations (stories — `spdd-story` form)

**GitHub issues:** L.1 #1180 · L.2 #1181 · L.3 #1182 · L.4 #1183 (epic #468)
**L.1 — Engine-adapter history streaming (closes #468)** · M · `refactor` · **delivered: #1180 (PR #1237)**
- As an `operator`, I want history long-poll instead of polling so events stream with low latency.
- AC: [x] `GetWorkflowExecutionHistory(isLongPoll=true)` replaces `DescribeWorkflowExecution` polling (no periodic `DescribeWorkflowExecution` calls during a running workflow); [x] history events mapped to ordered `StateTransitionEvent`s with correct non-empty `from_state`/`to_state` (fixes the empty-state fidelity bug from review §5.6/H6/R10); [x] `GetTask` polling replaced with `WatchTask` streaming RPC (reconnect-with-backoff on broker restart); [x] activity completion latency p50 ≤ 100 ms (broker ACK → Temporal completion); [x] domain cov ≥90%. Deps: none.

**L.2 — Per-workflow event subscription** · S · `feat`
- As the `gateway`, I want a workflow-scoped event stream so capability events reach clients.
- AC: [ ] reuse EventBus `Subscribe` with `WorkflowID` scope; [ ] stream closes on terminal state. Deps: L.1.

**L.3 — Streaming `/logs` endpoint** · M · `feat`
- As a `developer`, I want a real streaming `/logs` so I can watch a run live.
- AC: [ ] SSE/chunked response merging history + events; [ ] no `streaming not supported`; [ ] `.feature` committed first. Deps: L.1, L.2.

**L.4 — `zynax logs --follow`** · S · `feat`
- As a `developer`, I want a CLI follower so I can tail a run from the terminal.
- AC: [ ] `zynax logs <run-id> --follow` prints transitions/events until terminal. Deps: L.3.

**Order:** L.1 → L.2 → L.3 → L.4.

## N — Norms
- `.feature` before the streaming endpoint impl (ADR-016); `GOWORK=off` (ADR-017).
- `Signed-off-by:` + `Assisted-by:`; one logical change per commit.

## S — Safeguards (second S)
### Context Security
- [ ] No Tier 2 content; [ ] no PII; [ ] no prompt-injection; [ ] `/spdd-security-review` — PENDING

### Feature Safeguards
- Never leak payload secrets in streamed events — redact (consistent with EPIC O).
- Never block the engine worker on a slow client — bounded buffering / drop-with-marker.
- Never bypass auth on the streaming endpoint — same bearer rules as other gateway routes.
