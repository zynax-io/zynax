<!-- SPDX-License-Identifier: Apache-2.0 -->

# REASONS Canvas — M6.Argo: ArgoEngine WorkflowEngine Implementation

> **All content in this Canvas is Tier 1 (public-safe).**

**Issue:** #766
**Author:** Oscar Gómez Manresa
**Date:** 2026-06-02
**Status:** Implemented

**Child issues:** #795 (B.1) · #796 (B.2) · #797 (B.3) · #798 (B.4)

---

## R — Requirements

**Problem:** engine-adapter supports only Temporal. The M6 roadmap requires Argo Workflows as a second execution engine so that operators can choose between Temporal (stateful, durable) and Argo (K8s-native, DAG-first) per deployment. Without ArgoEngine, the "K8s Production-Ready" milestone has only one engine path, blocking the Argo-flavoured e2e tests (EPIC G).

**Definition of done:**
- `ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE=argo` causes workflows to be dispatched to Argo Workflows.
- `Submit`, `Signal`, `Cancel`, `GetStatus`, `Watch` all work behind the `WorkflowEngine` interface.
- No engine name is hardcoded outside of the config-flag startup switch.
- BDD scenarios in `protos/tests/engine_adapter_service/features/argo_engine.feature` pass.
- The existing Temporal engine continues to work unchanged.

---

## E — Entities

- **`domain.WorkflowEngine`** — existing port interface (`Submit`, `Signal`, `Cancel`, `GetStatus`, `Watch`); unchanged.
- **`infrastructure.TemporalEngine`** — existing Temporal implementation; unchanged; not deleted.
- **`infrastructure.ArgoEngine`** — NEW: implements `domain.WorkflowEngine` against Argo Workflows API.
- **`ArgoConfig` proto message** — NEW: added to `EngineConfig` oneof in `engine_adapter.proto`; fields: `argo_server_url` (abstract endpoint name), `namespace`, `service_account_name`, `workflow_template_ref`.
- **`argo_engine.feature`** — NEW BDD feature file at `protos/tests/engine_adapter_service/features/`; committed before B.2 implementation (ADR-016).
- **`ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE`** — existing env var; extended to accept `"argo"` as a valid value alongside `"temporal"`.
- **`ArgoClient`** — NEW thin wrapper around the Argo Workflows server REST/gRPC API; injected into `ArgoEngine`.

---

## A — Approach

**What we WILL do:**
- Add `ArgoConfig` message to `EngineConfig` oneof in `engine_adapter.proto` (B.1); commit `argo_engine.feature` BDD file first (ADR-016).
- Implement `ArgoEngine.Submit` (B.2): translate `WorkflowIR` to an Argo `Workflow` resource; submit via `ArgoClient`.
- Implement `ArgoEngine.Signal` (B.2): deliver external events via Argo's `WorkflowEventBinding` or direct resource patch.
- Implement `ArgoEngine.Query/Cancel/Watch` (B.3): map Argo `WorkflowStatus` to `WorkflowRun`; cancel via deletion; Watch polls until terminal.
- Wire multi-engine dispatch in `main.go` (B.4): `ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE=argo` injects `ArgoEngine`.

**What we WON'T do:**
- Remove or modify `TemporalEngine` — both engines coexist.
- Support simultaneous multi-engine dispatch (one active engine per process — ADR-015).
- Implement Argo Workflows CRD provisioning — that is the EPIC A Helm chart (A.3 engine-adapter chart includes ArgoEngine values).

**ADR references:**
- ADR-015: Pluggable workflow engines — no engine name hardcoding; always route through `WorkflowEngine` interface.
- ADR-016: Contracts before code — `argo_engine.feature` BDD file must be committed before B.2 implementation.
- ADR-009: Go services — all engine implementation is Go in `services/engine-adapter/`.

---

## S — Structure

**New files:**
```
protos/zynax/v1/engine_adapter.proto       ← extended: ArgoConfig in EngineConfig oneof (B.1)
protos/tests/engine_adapter_service/
  features/argo_engine.feature              ← NEW BDD scenarios (B.1, before impl)
services/engine-adapter/internal/
  infrastructure/
    argo_client.go                          ← NEW: ArgoClient HTTP/gRPC wrapper (B.2)
    argo_engine.go                          ← NEW: WorkflowEngine impl (B.2+B.3)
    argo_engine_submit_test.go              ← NEW: unit tests (B.2)
    argo_engine_query_test.go               ← NEW: unit tests (B.3)
cmd/engine-adapter/main.go                  ← modified: "argo" case in engine selector (B.4)
protos/generated/go/zynax/v1/              ← regenerated stubs (B.1)
protos/generated/python/zynax/v1/          ← regenerated stubs (B.1)
```

---

## O — Operations

1. **[B.1]** Add `ArgoConfig` message to `EngineConfig` oneof in `engine_adapter.proto`; commit `argo_engine.feature` BDD file with ≥3 scenarios (submit, query, cancel via Argo); run `make generate-protos`; `buf breaking` passes.

2. **[B.2]** Implement `ArgoEngine.Submit` + `ArgoEngine.Signal`: `ArgoClient` wrapper; `Submit` translates `WorkflowIR` → Argo `Workflow` resource and submits; `Signal` delivers an event; unit tests with mocked `ArgoClient`; `GOWORK=off go test ./... -race` passes.

3. **[B.3]** Implement `ArgoEngine.GetStatus` + `ArgoEngine.Cancel` + `ArgoEngine.Watch`: map Argo `WorkflowStatus` phases to `WorkflowRun`; cancel via deletion; Watch polls until terminal; unit tests; `GOWORK=off go test ./... -race` passes.

4. **[B.4]** Wire multi-engine dispatch: add `"argo"` case to engine selector in `cmd/engine-adapter/main.go`; validate `ZYNAX_ENGINE_ADAPTER_ACTIVE_ENGINE` at startup; integration smoke test (`argo` engine returns `ErrExecutionNotFound` on unknown run ID); `GOWORK=off go build ./...` succeeds.

---

## N — Norms

- `feat:` PR type for B.1–B.4.
- Every commit: `Signed-off-by` trailer + `Assisted-by: Claude/claude-sonnet-4-6` per AGENTS.md §Hard Constraints. Never `Co-Authored-By:` for AI.
- `GOWORK=off` required for all `go test` and `go build` in `services/engine-adapter/` (ADR-017).
- `argo_engine.feature` MUST be committed in B.1 (before B.2 implementation code) — ADR-016 hard requirement.
- `make generate-protos` must be run after B.1 proto change; commit generated stubs in the same PR.
- Domain coverage ≥ 90% on `internal/domain/` after B.4.
- `ArgoClient` must be injected — never instantiated inside `ArgoEngine` (testability).
- Never hardcode an Argo server URL or namespace — read from config/env vars only.

---

## S — Safeguards

### Context Security
- [x] No Tier 2 content: no internal hostnames, private IPs, credentials, deployment specifics
- [x] No PII
- [x] No prompt injection
- [x] All entities in E are public-safe abstractions
- [x] /spdd-security-review passed on this file

### Feature Safeguards
- **Never** hardcode engine names in `WorkflowEngine` dispatch logic (ADR-015).
- **Never** remove `TemporalEngine` — both engines coexist and the test suite covers both.
- **Never** merge B.2 before `argo_engine.feature` is committed (ADR-016).
- **Never** store Argo server URLs or credentials in tracked files — config/env vars only.
- **Never** change the `WorkflowEngine` interface signature — it is the stable port contract.
