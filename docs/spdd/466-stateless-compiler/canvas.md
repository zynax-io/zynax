<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M6.D Stateless Compiler

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #466
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-18
**Status:** Implemented

**Child issues:** #490 (drop in-memory IR store)

---

## R — Requirements

**Problem:** The workflow-compiler's in-memory `sync.RWMutex` IR store (review §9.2, H7, R9) is the single largest blocker to horizontal scaling. Multiple replicas cannot share IRs compiled by other replicas. The store is also lost on restart. ADR-008 forbids shared databases, which directly points to the correct solution: make the compiler stateless.

**Definition of done:**
- `GetCompiledWorkflow` returns `codes.NotFound` unconditionally.
- Multiple workflow-compiler replicas behind a load balancer handle concurrent `zynax apply` calls correctly.
- `CompileWorkflow` round-trip latency is not regressed by the change.

---

## E — Entities

- **In-memory IR store** — `sync.RWMutex` over `map[string]*zynaxv1.WorkflowIR` in workflow-compiler; to be deleted.
- **`GetCompiledWorkflow` RPC** — returns `codes.NotFound` after this change; callers must not rely on it (api-gateway is the only caller and will be updated).
- **`ir_payload` field** — `bytes ir_payload` in `EngineAdapterService.SubmitWorkflow` request; api-gateway passes the compiled IR bytes directly here after compile, bypassing any storage.
- **`CompileWorkflowResponse`** — proto response; the compiled IR bytes are already present in the response; api-gateway reads them and passes them forward.

---

## A — Approach

**What we WILL do:**
- Delete the in-memory store from `workflow-compiler`.
- `GetCompiledWorkflow` returns `codes.NotFound` unconditionally with a message directing callers to use `CompileWorkflow` and retain the returned IR.
- Update api-gateway: after `CompileWorkflow` returns, store the IR bytes in the request context and pass them directly to `engine-adapter.SubmitWorkflow` via `ir_payload`.

**What we WON'T do:**
- Add a shared persistent store (violates ADR-008).
- Change the proto contract — `ir_payload` field already exists in `SubmitWorkflowRequest`.

**ADR references:**
- ADR-008: No shared databases — confirms stateless is the correct direction.
- ADR-012: WorkflowIR as engine-agnostic IR — the IR is passed by value, not by reference.

---

## S — Structure

**Files modified:**
- `services/workflow-compiler/internal/infrastructure/` — delete IR store implementation
- `services/workflow-compiler/internal/api/server.go` — `GetCompiledWorkflow` returns NOT_FOUND
- `services/api-gateway/internal/api/handler.go` — after `CompileWorkflow`, pass IR bytes to `SubmitWorkflow`

---

## O — Operations

1. **[#490]** Delete IR store; update `GetCompiledWorkflow` to return NOT_FOUND; update api-gateway to pass IR bytes forward; latency regression test; verify multi-replica correctness.

---

## N — Norms

- `refactor:` PR type (behaviour-preserving from the caller's perspective; no new features).
- Regression test: `CompileWorkflow` latency p50 and p99 before/after (assert no regression).
- `GOWORK=off go test ./... -race` in both `services/workflow-compiler/` and `services/api-gateway/`.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed

### Feature Safeguards
- `GetCompiledWorkflow` returning NOT_FOUND is already permitted by the proto contract comment — verify this before deleting the store.
- Never cache IR bytes in the api-gateway beyond the request lifetime — statelessness is the goal.
