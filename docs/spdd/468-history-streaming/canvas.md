<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M7.B Replace Polling Watch with Temporal History Streaming

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #468
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-18
**Status:** Aligned

**Child issues:** #492 (GetWorkflowHistory long-poll implementation)

---

## R — Requirements

**Problem:** Two polling anti-patterns in engine-adapter identified by the review (§5.6, H6, R10):
1. `Watch` polls `DescribeWorkflowExecution` every 2 s — emits a heartbeat event whether or not anything changed; `from_state` and `to_state` are always empty; fidelity is wrong.
2. `DispatchCapabilityActivity` polls `GetTask` every 500 ms — at scale this produces hundreds of thousands of RPCs per second.

**Definition of done:**
- `zynax logs wf-<hex>` emits `StateTransitionEvent` with correct non-empty `from_state` and `to_state` fields.
- No periodic `DescribeWorkflowExecution` calls during a running workflow.
- Activity completion latency p50 ≤ 100 ms from broker ACK to Temporal completion.

---

## E — Entities

- **`GetWorkflowExecutionHistory`** — Temporal API with `isLongPoll=true`; returns history events as they are written, blocking until new events arrive.
- **`WatchTask` RPC** — server-streaming RPC in `TaskBrokerService` proto; already defined; broker pushes events to engine-adapter instead of engine-adapter polling.
- **`StateTransitionEvent`** — proto message with `from_state`, `to_state`, `event_type` fields; currently emitted with empty `from_state`/`to_state`.
- **Temporal history event types** — `WorkflowExecutionStarted`, `ActivityTaskCompleted`, `WorkflowExecutionCompleted`, etc.; mapped to `StateTransitionEvent`.
- **`DescribeWorkflowExecution`** — current polling API; to be removed from the Watch hot path.

---

## A — Approach

**What we WILL do:**
- Replace `DescribeWorkflowExecution` polling with `GetWorkflowExecutionHistory(isLongPoll=true)` iterator.
- Map each Temporal history event type to the corresponding IR state transitions.
- Replace `GetTask` polling in `DispatchCapabilityActivity` with a `WatchTask` streaming RPC call to the broker (requires broker to implement `WatchTask`, which is done in M5.C).
- Benchmark: assert p50 event latency ≤ 100 ms.

**What we WON'T do:**
- Change the `WatchWorkflow` proto contract (the streaming API is already correct; we fix the implementation).
- Change how Temporal dispatches activities (Temporal's scheduling is the engine's responsibility).

**ADR references:**
- ADR-015: Pluggable workflow engines — the Watch improvement is engine-specific (Temporal). Other engines implement their own Watch adapters.

---

## S — Structure

**Files touched:**
- `services/engine-adapter/internal/infrastructure/temporal.go` — replace polling with `GetWorkflowHistory` long-poll
- `services/engine-adapter/internal/domain/activity.go` — replace `GetTask` polling with `WatchTask` streaming
- `services/engine-adapter/internal/infrastructure/temporal_test.go` — benchmark tests

---

## O — Operations

1. **[#492]** Replace `DescribeWorkflowExecution` polling with `GetWorkflowHistory` long-poll; map history events to `StateTransitionEvent` with correct `from_state`/`to_state`; replace `GetTask` polling with `WatchTask` streaming; benchmark.

---

## N — Norms

- `refactor:` PR type (behaviour-preserving from the API caller's perspective; observable improvement in event fidelity).
- `GOWORK=off go test ./... -race` in `services/engine-adapter/`.
- Benchmark must be committed to `tools/bench-baseline.txt` (prerequisite: M7.C bench infrastructure).

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed

### Feature Safeguards
- Never call `DescribeWorkflowExecution` in a tight loop after this change — that re-introduces the polling anti-pattern.
- The `WatchTask` streaming connection must handle broker restarts gracefully (reconnect with backoff).
