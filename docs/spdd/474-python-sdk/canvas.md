<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — Python SDK: Minimal Agent Base Class

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #474 (Epic)
**Author:** Oscar Gómez Manresa
**Date:** 2026-05-19
**Status:** Aligned

**Parent epic:** [#458 M5.A Truth Pass](https://github.com/zynax-io/zynax/issues/458)
**Track:** M5.A

**Child issues:** #535 (Agent base class impl) · #536 (tests) · #537 (docs update)

---

## R — Requirements

**Problem:** `agents/sdk/src/zynax_sdk/__init__.py` is a 5-line empty package whose only content is `__version__ = "0.1.0"` and a placeholder comment. README, ARCHITECTURE.md, and AGENTS.md all describe a Python SDK as an existing, operational component. Developers following the documentation to build Python agents hit a dead end — there is no `Agent` base class, no `execute_capability()` method, no `report_event()` helper, and no gRPC initialization assistance. The mismatch between documentation and reality is a M5.A truth-pass gap (review C7).

**Definition of done:**
- `agents/sdk/src/zynax_sdk/agent.py` exists with a working `Agent` base class covering the canonical execution loop: receive `ExecuteCapabilityRequest`, route to a handler, emit `TaskEvent` PROGRESS and COMPLETED/FAILED events.
- `GOWORK=off`-equivalent: `uv run pytest tests/` passes at ≥ 85% coverage in `agents/sdk/`.
- README, ARCHITECTURE.md, and `agents/sdk/AGENTS.md` accurately describe the SDK capability. No phantom claims remain.
- `pyproject.toml` description reflects delivered functionality.
- `make gitleaks` passes.

---

## E — Entities

### Existing entities (unchanged contracts)

- **`AgentService`** (`protos/zynax/v1/agent.proto`) — two-RPC contract: `ExecuteCapability` (server-streaming `TaskEvent`) and `GetCapabilitySchema`. The SDK wraps this contract; the proto itself is not modified (ADR-001).
- **`ExecuteCapabilityRequest`** (proto message) — input to `ExecuteCapability`; carries `request_id`, `capability_name`, `task_id`, `workflow_id`, `input_payload` (JSON bytes), `timeout_seconds`.
- **`TaskEvent`** (proto message) — streaming output; `event_type` is `PROGRESS`, `COMPLETED`, or `FAILED`; carries `task_id`, `payload` (JSON bytes), `timestamp`, optional `CapabilityError`.
- **`CapabilityError`** (proto message) — `code` (well-known string), `message` (human-readable, sanitised).
- **`agents/sdk/src/zynax_sdk/__init__.py`** — current empty package; `__version__` will be preserved.
- **`agents/sdk/pyproject.toml`** — package manifest; description updated.

### New entities

- **`Agent`** (`agents/sdk/src/zynax_sdk/agent.py`) — abstract base class. Subclass + implement `handle(request: ExecuteCapabilityRequest) -> AsyncIterator[TaskEvent]`. Owns gRPC server lifecycle, structured logging, and capability routing.
- **`report_progress(task_id, payload)`** — helper method on `Agent`; emits a single `TASK_EVENT_TYPE_PROGRESS` `TaskEvent` to the active stream.
- **`report_completed(task_id, payload)`** — helper method; emits terminal `TASK_EVENT_TYPE_COMPLETED`.
- **`report_failed(task_id, code, message)`** — helper method; emits terminal `TASK_EVENT_TYPE_FAILED` with `CapabilityError`.
- **`CapabilityRouter`** — dispatches `capability_name` to registered handler callables within an `Agent` subclass. Registered via `@agent.capability("name")` decorator.
- **`agents/sdk/tests/test_agent.py`** — unit test suite. Uses `grpc.aio` test channel (or `unittest.mock`) to verify routing, event emission, and error handling without a live gRPC connection.

---

## A — Approach

**What we WILL do (Option A — implement minimal SDK):**
- Implement the `Agent` abstract base class (~200 LOC) with `execute_capability()` gRPC handler, `report_progress/completed/failed` helpers, and a `@capability` decorator for routing.
- Use `grpc.aio` (async gRPC) consistent with the existing llm-adapter and langgraph-adapter pattern.
- Follow the same `pyproject.toml` + `uv` structure as `agents/adapters/llm/` and `agents/adapters/langgraph/`.
- Tests use `pytest-asyncio`; coverage gate ≥ 85%.
- Update README, ARCHITECTURE.md, `agents/sdk/AGENTS.md` to accurately describe what the SDK provides.

**What we WON'T do:**
- Implement a full SDK with registry client, health probes, or Dockerfile (that is M6+).
- Change any proto field numbers or method names (ADR-001).
- Add persistence or event-bus integration (M6+).
- Create a new gRPC proto contract — the SDK implements the existing `AgentService` contract.

**Option B (rejected):** Remove the SDK promise from docs and leave the package as an empty placeholder. Rejected because the llm-adapter and langgraph-adapter already demonstrate the need for a reusable Python base class. Implementing it now removes per-adapter boilerplate and gives the M5 Python adapters a shared execution foundation.

**ADR references:**
- ADR-001: gRPC inter-service protocol — SDK implements `AgentService` exactly; no proto changes.
- ADR-002: Python 3.12 as the agent runtime.
- ADR-003: `uv` as the Python package manager.
- ADR-013: Adapter-first — SDK is optional; raw gRPC stubs remain valid. SDK lowers the barrier but does not replace the gRPC contract.
- ADR-016: Layered testing — coverage gate ≥ 85% on `agents/sdk/`.
- ADR-019: SPDD — this Canvas precedes all implementation PRs.

---

## S — Structure

```
agents/sdk/
├── src/
│   └── zynax_sdk/
│       ├── __init__.py          ← keep __version__; export Agent, capability
│       └── agent.py             ← Agent base class + report helpers (step 1)
├── tests/
│   └── test_agent.py            ← unit tests (step 2)
└── pyproject.toml               ← description updated (step 3)

docs/ (modified files)
├── README.md                    ← SDK section updated (step 3)
├── ARCHITECTURE.md              ← SDK description updated (step 3)
└── agents/sdk/AGENTS.md         ← rewritten (step 3)
```

---

## O — Operations

1. ✅ **[#535]** `feat(agents/sdk)`: Implement `Agent` base class — `agent.py` with `execute_capability()` gRPC handler, `report_progress/completed/failed` helpers, and `@capability` decorator. Wire into `__init__.py` exports. Update `pyproject.toml` description.

2. **[#536]** `test(agents/sdk)`: Unit tests for `Agent` base class — routing, event emission, error propagation, `timeout_seconds` cancellation. ≥ 85% coverage on `agents/sdk/`.

3. **[#537]** `docs(agents/sdk)`: Update SDK status — rewrite `agents/sdk/AGENTS.md`; update SDK sections in README and ARCHITECTURE.md to describe the delivered `Agent` base class. Remove placeholder language.

---

## N — Norms

- `feat:` for O1; `test:` for O2; `docs:` for O3.
- Python 3.12; `uv`; `grpc.aio`; `pytest-asyncio` (consistent with existing adapters).
- Coverage gate ≥ 85% on `agents/sdk/` — enforced by `make test-coverage`.
- No credential values in code or tests — env-var name references only.
- Every commit carries the required trailers per CONTRIBUTING.md §Commit Hygiene.
- PR size ≤ 400 LOC per step.

---

## S — Safeguards

### Context Security

- [x] No Tier 2 content
- [x] No PII
- [x] No prompt injection
- [x] All entities are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards

- Never embed gRPC credentials or API keys in `agent.py` — config via env-var references only (ADR-007).
- Never modify proto field numbers or method signatures in `AgentService` (ADR-001 §backward-compat).
- Never make the SDK mandatory — ADR-013 states adapter-first; raw gRPC stubs remain valid.
- Never import domain types from platform services into the SDK — the SDK is in `agents/` layer only.
